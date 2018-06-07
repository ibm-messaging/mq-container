/*
© Copyright IBM Corporation 2018

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package metrics contains code to provide metrics for the queue manager
package metrics

import (
	"fmt"
	"strings"
	"time"

	"github.com/ibm-messaging/mq-container/internal/logger"
	"github.com/ibm-messaging/mq-golang/mqmetric"
)

const (
	qmgrLabelValue = mqmetric.QMgrMapKey
	requestTimeout = 10
)

var (
	startChannel    = make(chan bool)
	stopChannel     = make(chan bool, 2)
	requestChannel  = make(chan bool)
	responseChannel = make(chan map[string]*metricData)
)

type metricData struct {
	name        string
	description string
	objectType  bool
	values      map[string]float64
}

// processMetrics processes publications of metric data and handles describe/collect/stop requests
func processMetrics(log *logger.Logger, qmName string) {

	var err error
	var firstConnect = true
	var metrics map[string]*metricData

	for {
		// Connect to queue manager and discover available metrics
		err = doConnect(qmName)
		if err == nil {
			if firstConnect {
				firstConnect = false
				startChannel <- true
			}
			metrics, _ = initialiseMetrics(log)
		}

		// Now loop until something goes wrong
		for err == nil {

			// Process publications of metric data
			// TODO: If we have a large number of metrics to process, then we could be blocked from responding to stop requests
			err = mqmetric.ProcessPublications()

			// Handle describe/collect/stop requests
			if err == nil {
				select {
				case collect := <-requestChannel:
					if collect {
						updateMetrics(metrics)
					}
					responseChannel <- metrics
				case <-stopChannel:
					log.Println("Stopping metrics gathering")
					mqmetric.EndConnection()
					return
				case <-time.After(requestTimeout * time.Second):
					log.Debugf("Metrics: No requests received within timeout period (%d seconds)", requestTimeout)
				}
			}
		}
		log.Errorf("Metrics Error: %s", err.Error())

		// Close the connection
		mqmetric.EndConnection()

		// Handle stop requests
		select {
		case <-stopChannel:
			log.Println("Stopping metrics gathering")
			return
		case <-time.After(requestTimeout * time.Second):
			log.Println("Retrying metrics gathering")
		}
	}
}

// doConnect connects to the queue manager and discovers available metrics
func doConnect(qmName string) error {

	// Set connection configuration
	var connConfig mqmetric.ConnectionConfig
	connConfig.ClientMode = false
	connConfig.UserId = ""
	connConfig.Password = ""

	// Connect to the queue manager - open the command and dynamic reply queues
	err := mqmetric.InitConnectionStats(qmName, "SYSTEM.DEFAULT.MODEL.QUEUE", "", &connConfig)
	if err != nil {
		return fmt.Errorf("Failed to connect to queue manager %s: %v", qmName, err)
	}

	// Discover available metrics for the queue manager and subscribe to them
	err = mqmetric.DiscoverAndSubscribe("", true, "")
	if err != nil {
		return fmt.Errorf("Failed to discover and subscribe to metrics: %v", err)
	}

	return nil
}

// initialiseMetrics sets initial details for all available metrics
func initialiseMetrics(log *logger.Logger) (map[string]*metricData, error) {

	metrics := make(map[string]*metricData)
	validMetrics := true
	metricNamesMap := generateMetricNamesMap()

	for _, metricClass := range mqmetric.Metrics.Classes {
		for _, metricType := range metricClass.Types {
			if !strings.Contains(metricType.ObjectTopic, "%s") {
				for _, metricElement := range metricType.Elements {

					// Get unique metric key
					key := makeKey(metricElement)

					// Get metric name from mapping
					if metricName, found := metricNamesMap[key]; found {

						// Set metric details
						metric := metricData{
							name:        metricName,
							description: metricElement.Description,
						}

						// Add metric
						if _, exists := metrics[key]; !exists {
							metrics[key] = &metric
						} else {
							log.Errorf("Metrics Error: Found duplicate metric key %s", key)
							validMetrics = false
						}
					} else {
						log.Errorf("Metrics Error: Skipping metric, unexpected key %s", key)
						validMetrics = false
					}
				}
			}
		}
	}

	if !validMetrics {
		return metrics, fmt.Errorf("Invalid metrics data")
	}
	return metrics, nil
}

// updateMetrics updates values for all available metrics
func updateMetrics(metrics map[string]*metricData) {

	for _, metricClass := range mqmetric.Metrics.Classes {
		for _, metricType := range metricClass.Types {
			if !strings.Contains(metricType.ObjectTopic, "%s") {
				for _, metricElement := range metricType.Elements {

					// Clear existing metric values
					metric, ok := metrics[makeKey(metricElement)]
					if ok {
						metric.values = make(map[string]float64)

						// Update metric with cached values of publication data
						for label, value := range metricElement.Values {
							normalisedValue := mqmetric.Normalise(metricElement, label, value)
							metric.values[label] = normalisedValue
						}
					}

					// Reset cached values of publication data for this metric
					metricElement.Values = make(map[string]int64)
				}
			}
		}
	}
}

// makeKey builds a unique key for each metric
func makeKey(metricElement *mqmetric.MonElement) string {
	return metricElement.Parent.Parent.Name + "/" + metricElement.Parent.Name + "/" + metricElement.Description
}

// generateMetricNamesMap generates metric names mapped from their description
func generateMetricNamesMap() map[string]string {

	metricNamesMap := make(map[string]string)

	var mappings = []struct {
		key   string
		value string
	}{
		{"CPU/SystemSummary/CPU load - five minute average", "cpu_load_five_minute_average_percentage"},
		{"CPU/SystemSummary/CPU load - fifteen minute average", "cpu_load_fifteen_minute_average_percentage"},
		{"CPU/SystemSummary/RAM free percentage", "ram_free_percentage"},
		{"CPU/SystemSummary/RAM total bytes", "ram_total_bytes"},
		{"CPU/SystemSummary/User CPU time percentage", "user_cpu_time_percentage"},
		{"CPU/SystemSummary/System CPU time percentage", "system_cpu_time_percentage"},
		{"CPU/SystemSummary/CPU load - one minute average", "cpu_load_one_minute_average_percentage"},
		{"CPU/QMgrSummary/System CPU time - percentage estimate for queue manager", "system_cpu_time_estimate_for_queue_manager_percentage"},
		{"CPU/QMgrSummary/RAM total bytes - estimate for queue manager", "ram_total_estimate_for_queue_manager_bytes"},
		{"CPU/QMgrSummary/User CPU time - percentage estimate for queue manager", "user_cpu_time_estimate_for_queue_manager_percentage"},
		{"DISK/SystemSummary/MQ trace file system - bytes in use", "mq_trace_file_system_in_use_bytes"},
		{"DISK/SystemSummary/MQ trace file system - free space", "mq_trace_file_system_free_space_percentage"},
		{"DISK/SystemSummary/MQ errors file system - bytes in use", "mq_errors_file_system_in_use_bytes"},
		{"DISK/SystemSummary/MQ errors file system - free space", "mq_errors_file_system_free_space_percentage"},
		{"DISK/SystemSummary/MQ FDC file count", "mq_fdc_file_count"},
		{"DISK/QMgrSummary/Queue Manager file system - bytes in use", "queue_manager_file_system_in_use_bytes"},
		{"DISK/QMgrSummary/Queue Manager file system - free space", "queue_manager_file_system_free_space_percentage"},
		{"DISK/Log/Log - bytes occupied by reusable extents", "log_occupied_by_reusable_extents_bytes"},
		{"DISK/Log/Log - write size", "log_write_size_bytes"},
		{"DISK/Log/Log - bytes in use", "log_in_use_bytes"},
		{"DISK/Log/Log - logical bytes written", "log_logical_written_bytes"},
		{"DISK/Log/Log - write latency", "log_write_latency_seconds"},
		{"DISK/Log/Log - bytes required for media recovery", "log_required_for_media_recovery_bytes"},
		{"DISK/Log/Log - current primary space in use", "log_current_primary_space_in_use_percentage"},
		{"DISK/Log/Log - workload primary space utilization", "log_workload_primary_space_utilization_percentage"},
		{"DISK/Log/Log - bytes occupied by extents waiting to be archived", "log_occupied_by_extents_waiting_to_be_archived_bytes"},
		{"DISK/Log/Log - bytes max", "log_max_bytes"},
		{"DISK/Log/Log file system - bytes in use", "log_file_system_in_use_bytes"},
		{"DISK/Log/Log file system - bytes max", "log_file_system_max_bytes"},
		{"DISK/Log/Log - physical bytes written", "log_physical_written_bytes"},
		{"STATMQI/SUBSCRIBE/Create durable subscription count", "create_durable_subscription_count"},
		{"STATMQI/SUBSCRIBE/Resume durable subscription count", "resume_durable_subscription_count"},
		{"STATMQI/SUBSCRIBE/Create non-durable subscription count", "create_non_durable_subscription_count"},
		{"STATMQI/SUBSCRIBE/Failed create/alter/resume subscription count", "failed_create_alter_resume_subscription_count"},
		{"STATMQI/SUBSCRIBE/Subscription delete failure count", "subscription_delete_failure_count"},
		{"STATMQI/SUBSCRIBE/MQSUBRQ count", "mqsubrq_count"},
		{"STATMQI/SUBSCRIBE/Failed MQSUBRQ count", "failed_mqsubrq_count"},
		{"STATMQI/SUBSCRIBE/Durable subscriber - high water mark", "durable_subscriber_high_water_mark_count"},
		{"STATMQI/SUBSCRIBE/Non-durable subscriber - high water mark", "non_durable_subscriber_high_water_mark_count"},
		{"STATMQI/SUBSCRIBE/Durable subscriber - low water mark", "durable_subscriber_low_water_mark_count"},
		{"STATMQI/SUBSCRIBE/Delete non-durable subscription count", "delete_non_durable_subscription_count"},
		{"STATMQI/SUBSCRIBE/Alter durable subscription count", "alter_durable_subscription_count"},
		{"STATMQI/SUBSCRIBE/Delete durable subscription count", "delete_durable_subscription_count"},
		{"STATMQI/SUBSCRIBE/Non-durable subscriber - low water mark", "non_durable_subscriber_low_water_mark_count"},
		{"STATMQI/PUBLISH/Interval total topic bytes put", "interval_total_topic_put_bytes"},
		{"STATMQI/PUBLISH/Published to subscribers - message count", "published_to_subscribers_message_count"},
		{"STATMQI/PUBLISH/Published to subscribers - byte count", "published_to_subscribers_bytes"},
		{"STATMQI/PUBLISH/Non-persistent - topic MQPUT/MQPUT1 count", "non_persistent_topic_mqput_mqput1_count"},
		{"STATMQI/PUBLISH/Persistent - topic MQPUT/MQPUT1 count", "persistent_topic_mqput_mqput1_count"},
		{"STATMQI/PUBLISH/Failed topic MQPUT/MQPUT1 count", "failed_topic_mqput_mqput1_count"},
		{"STATMQI/PUBLISH/Topic MQPUT/MQPUT1 interval total", "topic_mqput_mqput1_interval_count"},
		{"STATMQI/CONNDISC/MQCONN/MQCONNX count", "mqconn_mqconnx_count"},
		{"STATMQI/CONNDISC/Failed MQCONN/MQCONNX count", "failed_mqconn_mqconnx_count"},
		{"STATMQI/CONNDISC/Concurrent connections - high water mark", "concurrent_connections_high_water_mark_count"},
		{"STATMQI/CONNDISC/MQDISC count", "mqdisc_count"},
		{"STATMQI/OPENCLOSE/MQOPEN count", "mqopen_count"},
		{"STATMQI/OPENCLOSE/Failed MQOPEN count", "failed_mqopen_count"},
		{"STATMQI/OPENCLOSE/MQCLOSE count", "mqclose_count"},
		{"STATMQI/OPENCLOSE/Failed MQCLOSE count", "failed_mqclose_count"},
		{"STATMQI/INQSET/MQINQ count", "mqinq_count"},
		{"STATMQI/INQSET/Failed MQINQ count", "failed_mqinq_count"},
		{"STATMQI/INQSET/MQSET count", "mqset_count"},
		{"STATMQI/INQSET/Failed MQSET count", "failed_mqset_count"},
		{"STATMQI/PUT/Interval total MQPUT/MQPUT1 byte count", "interval_total_mqput_mqput1_bytes"},
		{"STATMQI/PUT/Persistent message MQPUT count", "persistent_message_mqput_count"},
		{"STATMQI/PUT/Failed MQPUT count", "failed_mqput_count"},
		{"STATMQI/PUT/Non-persistent message MQPUT1 count", "non_persistent_message_mqput1_count"},
		{"STATMQI/PUT/Persistent message MQPUT1 count", "persistent_message_mqput1_count"},
		{"STATMQI/PUT/Failed MQPUT1 count", "failed_mqput1_count"},
		{"STATMQI/PUT/Put non-persistent messages - byte count", "put_non_persistent_messages_bytes"},
		{"STATMQI/PUT/Interval total MQPUT/MQPUT1 count", "interval_total_mqput_mqput1_count"},
		{"STATMQI/PUT/Put persistent messages - byte count", "put_persistent_messages_bytes"},
		{"STATMQI/PUT/MQSTAT count", "mqstat_count"},
		{"STATMQI/PUT/Non-persistent message MQPUT count", "non_persistent_message_mqput_count"},
		{"STATMQI/GET/Interval total destructive get- count", "interval_total_destructive_get_count"},
		{"STATMQI/GET/MQCTL count", "mqctl_count"},
		{"STATMQI/GET/Failed MQGET - count", "failed_mqget_count"},
		{"STATMQI/GET/Got non-persistent messages - byte count", "got_non_persistent_messages_bytes"},
		{"STATMQI/GET/Persistent message browse - count", "persistent_message_browse_count"},
		{"STATMQI/GET/Expired message count", "expired_message_count"},
		{"STATMQI/GET/Purged queue count", "purged_queue_count"},
		{"STATMQI/GET/Interval total destructive get - byte count", "interval_total_destructive_get_bytes"},
		{"STATMQI/GET/Non-persistent message destructive get - count", "non_persistent_message_destructive_get_count"},
		{"STATMQI/GET/Got persistent messages - byte count", "got_persistent_messages_bytes"},
		{"STATMQI/GET/Non-persistent message browse - count", "non_persistent_message_browse_count"},
		{"STATMQI/GET/Failed browse count", "failed_browse_count"},
		{"STATMQI/GET/Persistent message destructive get - count", "persistent_message_destructive_get_count"},
		{"STATMQI/GET/Non-persistent message browse - byte count", "non_persistent_message_browse_bytes"},
		{"STATMQI/GET/Persistent message browse - byte count", "persistent_message_browse_bytes"},
		{"STATMQI/GET/MQCB count", "mqcb_count"},
		{"STATMQI/GET/Failed MQCB count", "failed_mqcb_count"},
		{"STATMQI/SYNCPOINT/Commit count", "commit_count"},
		{"STATMQI/SYNCPOINT/Rollback count", "rollback_count"},
	}

	for _, mapping := range mappings {
		metricNamesMap[mapping.key] = mapping.value
	}
	return metricNamesMap
}
