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
package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/docker/docker/api/types/container"
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
	metricsRaw, err := ioutil.ReadAll(resp.Body)
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

func metricsContainerConfig() *container.Config {
	return &container.Config{
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
		"cpu_load_five_minute_average_percentage",
		"cpu_load_fifteen_minute_average_percentage",
		"ram_free_percentage",
		"ram_total_bytes",
		"user_cpu_time_percentage",
		"system_cpu_time_percentage",
		"cpu_load_one_minute_average_percentage",
		"system_cpu_time_estimate_for_queue_manager_percentage",
		"ram_total_estimate_for_queue_manager_bytes",
		"user_cpu_time_estimate_for_queue_manager_percentage",
		"mq_trace_file_system_in_use_bytes",
		"mq_trace_file_system_free_space_percentage",
		"mq_errors_file_system_in_use_bytes",
		"mq_errors_file_system_free_space_percentage",
		"mq_fdc_file_count",
		"queue_manager_file_system_in_use_bytes",
		"queue_manager_file_system_free_space_percentage",
		"log_write_size_bytes",
		"log_in_use_bytes",
		"log_logical_written_bytes",
		"log_write_latency_seconds",
		"log_current_primary_space_in_use_percentage",
		"log_workload_primary_space_utilization_percentage",
		"log_max_bytes",
		"log_file_system_in_use_bytes",
		"log_file_system_max_bytes",
		"log_physical_written_bytes",
		"create_durable_subscription_count",
		"resume_durable_subscription_count",
		"create_non_durable_subscription_count",
		"failed_create_alter_resume_subscription_count",
		"subscription_delete_failure_count",
		"mqsubrq_count",
		"failed_mqsubrq_count",
		"durable_subscriber_high_water_mark_count",
		"non_durable_subscriber_high_water_mark_count",
		"durable_subscriber_low_water_mark_count",
		"delete_non_durable_subscription_count",
		"alter_durable_subscription_count",
		"delete_durable_subscription_count",
		"non_durable_subscriber_low_water_mark_count",
		"interval_total_topic_put_bytes",
		"published_to_subscribers_message_count",
		"published_to_subscribers_bytes",
		"non_persistent_topic_mqput_mqput1_count",
		"persistent_topic_mqput_mqput1_count",
		"failed_topic_mqput_mqput1_count",
		"topic_mqput_mqput1_interval_count",
		"mqconn_mqconnx_count",
		"failed_mqconn_mqconnx_count",
		"concurrent_connections_high_water_mark_count",
		"mqdisc_count",
		"mqopen_count",
		"failed_mqopen_count",
		"mqclose_count",
		"failed_mqclose_count",
		"mqinq_count",
		"failed_mqinq_count",
		"mqset_count",
		"failed_mqset_count",
		"interval_total_mqput_mqput1_bytes",
		"persistent_message_mqput_count",
		"failed_mqput_count",
		"non_persistent_message_mqput1_count",
		"persistent_message_mqput1_count",
		"failed_mqput1_count",
		"put_non_persistent_messages_bytes",
		"interval_total_mqput_mqput1_count",
		"put_persistent_messages_bytes",
		"mqstat_count",
		"non_persistent_message_mqput_count",
		"interval_total_destructive_get_count",
		"mqctl_count",
		"failed_mqget_count",
		"got_non_persistent_messages_bytes",
		"persistent_message_browse_count",
		"expired_message_count",
		"purged_queue_count",
		"interval_total_destructive_get_bytes",
		"non_persistent_message_destructive_get_count",
		"got_persistent_messages_bytes",
		"non_persistent_message_browse_count",
		"failed_browse_count",
		"persistent_message_destructive_get_count",
		"non_persistent_message_browse_bytes",
		"persistent_message_browse_bytes",
		"mqcb_count",
		"failed_mqcb_count",
		"commit_count",
		"rollback_count",
	}
	return names
}
