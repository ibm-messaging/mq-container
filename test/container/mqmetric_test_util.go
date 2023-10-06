/*
© Copyright IBM Corporation 2018, 2023

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
package main

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	ce "github.com/ibm-messaging/mq-container/test/container/containerengine"
)

type mqmetric struct {
	Key    string
	Value  string
	Labels map[string]string
}

const defaultMetricURL = "/metrics"
const defaultMetricPort = 9157
const defaultMQNamespace = "ibmmq"
const defaultMetricQMName = "qm1"

func getMetrics(t *testing.T, port string) []mqmetric {
	returned := []mqmetric{}
	urlToUse := fmt.Sprintf("http://localhost:%s%s", port, defaultMetricURL)
	resp, err := http.Get(urlToUse)
	if err != nil {
		t.Fatalf("Error from HTTP GET for metrics: %v", err)
		return returned
	}
	defer resp.Body.Close()
	metricsRaw, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Error reading metrics data: %v", err)
		return returned
	}
	return convertRawMetricToMap(t, string(metricsRaw))
}

// Also filters out all non "ibmmq" metrics
func convertRawMetricToMap(t *testing.T, input string) []mqmetric {
	returnList := []mqmetric{}
	scanner := bufio.NewScanner(strings.NewReader(input))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "#") {
			// Comment line of HELP or TYPE. Ignore
			continue
		}
		if !strings.HasPrefix(line, defaultMQNamespace) {
			// Not an ibmmq_ metric. Ignore
			continue
		}
		//It's an IBM MQ metric!
		key, value, labelMap, err := convertMetricLineToMetric(line)
		if err != nil {
			t.Fatalf("ibmmq_ metric could not be deciphered - %v", err)
		}

		toAdd := mqmetric{
			Key:    key,
			Value:  value,
			Labels: labelMap,
		}

		returnList = append(returnList, toAdd)
	}

	return returnList
}

func convertMetricLineToMetric(input string) (string, string, map[string]string, error) {
	// Lines are in the form "<key>{<labels>}<value>" or "<key> <value>"
	// Get the key and value while skipping the label
	var key, value string
	labelMap := make(map[string]string)
	if strings.Contains(input, "{") {
		// Get key
		splitted := strings.Split(input, "{")
		if len(splitted) != 2 {
			return "", "", labelMap, fmt.Errorf("Could not split by { Expected 2 but got %d - %s", len(splitted), input)
		}
		key = strings.TrimSpace(splitted[0])

		// Get value
		splitted = strings.Split(splitted[1], "}")
		if len(splitted) != 2 {
			return "", "", labelMap, fmt.Errorf("Could not split by } Expected 2 but got %d - %s", len(splitted), input)
		}
		value = strings.TrimSpace(splitted[1])

		// Get labels
		allLabels := strings.Split(splitted[0], ",")
		for _, e := range allLabels {
			labelPair := strings.Split(e, "=")
			if len(labelPair) != 2 {
				return "", "", labelMap, fmt.Errorf("Could not split label by '=' Expected 2 but got %d - %s", len(labelPair), e)
			}
			lkey := strings.TrimSpace(labelPair[0])
			lvalue := strings.TrimSpace(labelPair[1])
			lvalue = strings.Trim(lvalue, "\"")
			labelMap[lkey] = lvalue
		}

	} else {
		splitted := strings.Split(input, " ")
		if len(splitted) != 2 {
			return "", "", labelMap, fmt.Errorf("Could not split by ' ' Expected 2 but got %d - %s", len(splitted), input)
		}
		key = strings.TrimSpace(splitted[0])
		value = strings.TrimSpace(splitted[1])
	}
	return key, value, labelMap, nil
}

func waitForMetricReady(t *testing.T, port string) {
	timeout := 12 // 12 * 5 = 1 minute
	for i := 0; i < timeout; i++ {
		urlToUse := fmt.Sprintf("http://localhost:%s", port)
		resp, err := http.Get(urlToUse)
		if err == nil {
			resp.Body.Close()
			return
		}

		time.Sleep(time.Second * 10)
	}
	t.Fatalf("Metric endpoint failed to startup in timely manner")
}

func metricsContainerConfig() *ce.ContainerConfig {
	return &ce.ContainerConfig{
		Env: []string{
			"LICENSE=accept",
			"MQ_QMGR_NAME=" + defaultMetricQMName,
			"MQ_ENABLE_METRICS=true",
		},
	}
}

func metricNames() []string {

	// NB: There are currently a total of 93 metrics, but the following 3 do not generate values (based on the queue manager configuration)
	// - log_occupied_by_reusable_extents_bytes
	// - log_occupied_by_extents_waiting_to_be_archived_bytes
	// - log_required_for_media_recovery_bytes

	names := []string{
		"cpu_load_one_minute_average_percentage",
		"cpu_load_five_minute_average_percentage",
		"cpu_load_fifteen_minute_average_percentage",
		"system_cpu_time_percentage",
		"user_cpu_time_percentage",
		"ram_free_percentage",
		// disabled : "system_ram_size_bytes",
		"system_cpu_time_estimate_for_queue_manager_percentage",
		"user_cpu_time_estimate_for_queue_manager_percentage",
		"ram_usage_estimate_for_queue_manager_bytes",
		"trace_file_system_free_space_percentage",
		"trace_file_system_in_use_bytes",
		"errors_file_system_free_space_percentage",
		"errors_file_system_in_use_bytes",
		"fdc_files",
		"queue_manager_file_system_free_space_percentage",
		"queue_manager_file_system_in_use_bytes",
		"log_logical_written_bytes_total",
		"log_physical_written_bytes_total",
		"log_primary_space_in_use_percentage",
		"log_workload_primary_space_utilization_percentage",
		"log_write_latency_seconds",
		"log_max_bytes",
		"log_write_size_bytes",
		"log_in_use_bytes",
		"log_file_system_max_bytes",
		"log_file_system_in_use_bytes",
		"durable_subscription_create_total",
		"durable_subscription_alter_total",
		"durable_subscription_resume_total",
		"durable_subscription_delete_total",
		"non_durable_subscription_create_total",
		"non_durable_subscription_delete_total",
		"failed_subscription_create_alter_resume_total",
		"failed_subscription_delete_total",
		"mqsubrq_total",
		"failed_mqsubrq_total",
		// disabled : "durable_subscriber_high_water_mark",
		// disabled : "durable_subscriber_low_water_mark",
		// disabled : "non_durable_subscriber_high_water_mark",
		// disabled : "non_durable_subscriber_low_water_mark",
		"topic_mqput_mqput1_total",
		"topic_put_bytes_total",
		"failed_topic_mqput_mqput1_total",
		"persistent_topic_mqput_mqput1_total",
		"non_persistent_topic_mqput_mqput1_total",
		"published_to_subscribers_message_total",
		"published_to_subscribers_bytes_total",
		"mqconn_mqconnx_total",
		"failed_mqconn_mqconnx_total",
		"mqdisc_total",
		// disabled : "concurrent_connections_high_water_mark",
		"mqopen_total",
		"failed_mqopen_total",
		"mqclose_total",
		"failed_mqclose_total",
		"mqinq_total",
		"failed_mqinq_total",
		"mqset_total",
		"failed_mqset_total",
		"persistent_message_mqput_total",
		"persistent_message_mqput1_total",
		"persistent_message_put_bytes_total",
		"non_persistent_message_mqput_total",
		"non_persistent_message_mqput1_total",
		"non_persistent_message_put_bytes_total",
		"mqput_mqput1_total",
		"mqput_mqput1_bytes_total",
		"failed_mqput_total",
		"failed_mqput1_total",
		"mqstat_total",
		"persistent_message_destructive_get_total",
		"persistent_message_browse_total",
		"persistent_message_get_bytes_total",
		"persistent_message_browse_bytes_total",
		"non_persistent_message_destructive_get_total",
		"non_persistent_message_browse_total",
		"non_persistent_message_get_bytes_total",
		"non_persistent_message_browse_bytes_total",
		"destructive_get_total",
		"destructive_get_bytes_total",
		"failed_mqget_total",
		"failed_browse_total",
		"mqctl_total",
		"expired_message_total",
		"purged_queue_total",
		"mqcb_total",
		"failed_mqcb_total",
		"commit_total",
		"rollback_total",
	}
	return names
}
