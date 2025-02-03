/*
Package mqmetric contains a set of routines common to several
commands used to export MQ metrics to different backend
storage mechanisms including Prometheus and InfluxDB.
*/
package mqmetric

/*
  Copyright (c) IBM Corporation 2016, 2024

  Licensed under the Apache License, Version 2.0 (the "License");
  you may not use this file except in compliance with the License.
  You may obtain a copy of the License at

  http://www.apache.org/licenses/LICENSE-2.0

  Unless required by applicable law or agreed to in writing, software
  distributed under the License is distributed on an "AS IS" BASIS,
  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
  See the License for the specific language governing permissions and
  limitations under the License.

   Contributors:
     Mark Taylor - Initial Contribution
*/

/*
Need to turn the "friendly" name of each element into something
that is suitable for metric names.

Should also have consistency of units (always use seconds,
bytes etc), and organisation of the elements of the name (units last)

While we can't change the MQ-generated descriptions for its statistics,
we can reformat most of them heuristically here.
*/
import (
	"os"
	"regexp"
	"strings"

	"github.com/ibm-messaging/mq-golang/v5/ibmmq"
)

var (
	UseManualMetricMaps = false // May move to a config option at some point
)

// These are the original heuristically-derived metric names. This was built from running
// the code once and capturing info from traces. Any new metrics should show up from tools run
// during the release process.
var mHeur = map[string]string{
	"user_cpu_time_percentage":                                "user_cpu_time_percentage",
	"system_cpu_time_percentage":                              "system_cpu_time_percentage",
	"cpu_load_-_one_minute_average":                           "cpu_load_one_minute_average_percentage",
	"cpu_load_-_five_minute_average":                          "cpu_load_five_minute_average_percentage",
	"cpu_load_-_fifteen_minute_average":                       "cpu_load_fifteen_minute_average_percentage",
	"ram_free_percentage":                                     "ram_free_percentage",
	"ram_total_bytes":                                         "ram_total_bytes",
	"user_cpu_time_-_percentage_estimate_for_queue_manager":   "user_cpu_time_estimate_for_queue_manager_percentage",
	"system_cpu_time_-_percentage_estimate_for_queue_manager": "system_cpu_time_estimate_for_queue_manager_percentage",
	"ram_total_bytes_-_estimate_for_queue_manager":            "ram_total_estimate_for_queue_manager_bytes",

	// Class: Disk
	"mq_trace_file_system_-_bytes_in_use":                    "mq_trace_file_system_in_use_bytes",
	"mq_trace_file_system_-_free_space":                      "mq_trace_file_system_free_space_percentage",
	"mq_errors_file_system_-_bytes_in_use":                   "mq_errors_file_system_in_use_bytes",
	"mq_errors_file_system_-_free_space":                     "mq_errors_file_system_free_space_percentage",
	"mq_fdc_file_count":                                      "mq_fdc_file_count",
	"queue_manager_file_system_-_bytes_in_use":               "queue_manager_file_system_in_use_bytes",
	"queue_manager_file_system_-_free_space":                 "queue_manager_file_system_free_space_percentage",
	"log_-_bytes_in_use":                                     "log_in_use_bytes",
	"log_-_bytes_max":                                        "log_max_bytes",
	"log_file_system_-_bytes_in_use":                         "log_file_system_in_use_bytes",
	"log_file_system_-_bytes_max":                            "log_file_system_max_bytes",
	"log_-_physical_bytes_written":                           "log_physical_written_bytes",
	"log_-_logical_bytes_written":                            "log_logical_written_bytes",
	"log_-_write_latency":                                    "log_write_latency_seconds",
	"log_-_current_primary_space_in_use":                     "log_current_primary_space_in_use_percentage",
	"log_-_workload_primary_space_utilization":               "log_workload_primary_space_utilization_percentage",
	"log_-_bytes_required_for_media_recovery":                "log_required_for_media_recovery_bytes",
	"log_-_bytes_occupied_by_reusable_extents":               "log_occupied_by_reusable_extents_bytes",
	"log_-_bytes_occupied_by_extents_waiting_to_be_archived": "log_occupied_by_extents_waiting_to_be_archived_bytes",
	"log_-_write_size":                                       "log_write_size_bytes",

	// Still class disk, but specifically for the appliance
	"appliance_data_-_bytes_in_use": "appliance_data_in_use_bytes",
	"appliance_data_-_free_space":   "appliance_data_free_space_percentage",
	"system_volume_-_bytes_in_use":  "system_volume_in_use_bytes",
	"system_volume_-_free_space":    "system_volume_free_space_percentage",

	// Class: STATQ and STATMQI
	"mqinq_count":                                            "mqinq_count",
	"failed_mqinq_count":                                     "failed_mqinq_count",
	"mqset_count":                                            "mqset_count",
	"failed_mqset_count":                                     "failed_mqset_count",
	"interval_total_mqput/mqput1_count":                      "interval_mqput_mqput1_total_count",
	"interval_total_mqput/mqput1_byte_count":                 "interval_mqput_mqput1_total_bytes",
	"non-persistent_message_mqput_count":                     "non_persistent_message_mqput_count",
	"persistent_message_mqput_count":                         "persistent_message_mqput_count",
	"failed_mqput_count":                                     "failed_mqput_count",
	"non-persistent_message_mqput1_count":                    "non_persistent_message_mqput1_count",
	"persistent_message_mqput1_count":                        "persistent_message_mqput1_count",
	"failed_mqput1_count":                                    "failed_mqput1_count",
	"put_non-persistent_messages_-_byte_count":               "put_non_persistent_messages_bytes",
	"put_persistent_messages_-_byte_count":                   "put_persistent_messages_bytes",
	"mqstat_count":                                           "mqstat_count",
	"interval_total_destructive_get-_count":                  "interval_destructive_get_total_count",
	"interval_total_destructive_get_-_byte_count":            "interval_destructive_get_total_bytes",
	"non-persistent_message_destructive_get_-_count":         "non_persistent_message_destructive_get_count",
	"persistent_message_destructive_get_-_count":             "persistent_message_destructive_get_count",
	"failed_mqget_-_count":                                   "failed_mqget_count",
	"got_non-persistent_messages_-_byte_count":               "got_non_persistent_messages_bytes",
	"got_persistent_messages_-_byte_count":                   "got_persistent_messages_bytes",
	"non-persistent_message_browse_-_count":                  "non_persistent_message_browse_count",
	"persistent_message_browse_-_count":                      "persistent_message_browse_count",
	"failed_browse_count":                                    "failed_browse_count",
	"non-persistent_message_browse_-_byte_count":             "non_persistent_message_browse_bytes",
	"persistent_message_browse_-_byte_count":                 "persistent_message_browse_bytes",
	"expired_message_count":                                  "expired_message_count",
	"purged_queue_count":                                     "purged_queue_count",
	"mqcb_count":                                             "mqcb_count",
	"failed_mqcb_count":                                      "failed_mqcb_count",
	"mqctl_count":                                            "mqctl_count",
	"commit_count":                                           "commit_count",
	"rollback_count":                                         "rollback_count",
	"create_durable_subscription_count":                      "create_durable_subscription_count",
	"alter_durable_subscription_count":                       "alter_durable_subscription_count",
	"resume_durable_subscription_count":                      "resume_durable_subscription_count",
	"create_non-durable_subscription_count":                  "create_non_durable_subscription_count",
	"failed_create/alter/resume_subscription_count":          "failed_create_alter_resume_subscription_count",
	"delete_durable_subscription_count":                      "delete_durable_subscription_count",
	"delete_non-durable_subscription_count":                  "delete_non_durable_subscription_count",
	"subscription_delete_failure_count":                      "subscription_delete_failure_count",
	"mqsubrq_count":                                          "mqsubrq_count",
	"failed_mqsubrq_count":                                   "failed_mqsubrq_count",
	"durable_subscriber_-_high_water_mark":                   "durable_subscriber_high_water_mark",
	"durable_subscriber_-_low_water_mark":                    "durable_subscriber_low_water_mark",
	"non-durable_subscriber_-_high_water_mark":               "non_durable_subscriber_high_water_mark",
	"non-durable_subscriber_-_low_water_mark":                "non_durable_subscriber_low_water_mark",
	"topic_mqput/mqput1_interval_total":                      "topic_mqput_mqput1_interval_total",
	"interval_total_topic_bytes_put":                         "interval_topic_put_total",
	"published_to_subscribers_-_message_count":               "published_to_subscribers_message_count",
	"published_to_subscribers_-_byte_count":                  "published_to_subscribers_bytes",
	"non-persistent_-_topic_mqput/mqput1_count":              "non_persistent_topic_mqput_mqput1_count",
	"persistent_-_topic_mqput/mqput1_count":                  "persistent_topic_mqput_mqput1_count",
	"failed_topic_mqput/mqput1_count":                        "failed_topic_mqput_mqput1_count",
	"mqconn/mqconnx_count":                                   "mqconn_mqconnx_count",
	"failed_mqconn/mqconnx_count":                            "failed_mqconn_mqconnx_count",
	"concurrent_connections_-_high_water_mark":               "concurrent_connections_high_water_mark",
	"mqdisc_count":                                           "mqdisc_count",
	"mqopen_count":                                           "mqopen_count",
	"failed_mqopen_count":                                    "failed_mqopen_count",
	"mqclose_count":                                          "mqclose_count",
	"failed_mqclose_count":                                   "failed_mqclose_count",
	"mqput/mqput1_count":                                     "mqput_mqput1_count",
	"mqput_byte_count":                                       "mqput_bytes",
	"mqput_non-persistent_message_count":                     "mqput_non_persistent_message_count",
	"mqput_persistent_message_count":                         "mqput_persistent_message_count",
	"mqput1_non-persistent_message_count":                    "mqput1_non_persistent_message_count",
	"mqput1_persistent_message_count":                        "mqput1_persistent_message_count",
	"non-persistent_byte_count":                              "non_persistent_bytes",
	"persistent_byte_count":                                  "persistent_bytes",
	"queue_avoided_puts":                                     "queue_avoided_puts_percentage",
	"queue_avoided_bytes":                                    "queue_avoided_percentage",
	"lock_contention":                                        "lock_contention_percentage",
	"rolled_back_mqput_count":                                "rolled_back_mqput_count",
	"mqget_count":                                            "mqget_count",
	"mqget_byte_count":                                       "mqget_bytes",
	"destructive_mqget_non-persistent_message_count":         "destructive_mqget_non_persistent_message_count",
	"destructive_mqget_persistent_message_count":             "destructive_mqget_persistent_message_count",
	"destructive_mqget_non-persistent_byte_count":            "destructive_mqget_non_persistent_bytes",
	"destructive_mqget_persistent_byte_count":                "destructive_mqget_persistent_bytes",
	"mqget_browse_non-persistent_message_count":              "mqget_browse_non_persistent_message_count",
	"mqget_browse_persistent_message_count":                  "mqget_browse_persistent_message_count",
	"mqget_browse_non-persistent_byte_count":                 "mqget_browse_non_persistent_bytes",
	"mqget_browse_persistent_byte_count":                     "mqget_browse_persistent_bytes",
	"destructive_mqget_fails":                                "destructive_mqget_fails",
	"destructive_mqget_fails_with_mqrc_no_msg_available":     "destructive_mqget_fails_with_mqrc_no_msg_available",
	"destructive_mqget_fails_with_mqrc_truncated_msg_failed": "destructive_mqget_fails_with_mqrc_truncated_msg_failed",
	"mqget_browse_fails":                                     "mqget_browse_fails",
	"mqget_browse_fails_with_mqrc_no_msg_available":          "mqget_browse_fails_with_mqrc_no_msg_available",
	"mqget_browse_fails_with_mqrc_truncated_msg_failed":      "mqget_browse_fails_with_mqrc_truncated_msg_failed",
	"rolled_back_mqget_count":                                "rolled_back_mqget_count",
	"messages_expired":                                       "expired_messages",
	"queue_purged_count":                                     "queue_purged_count",
	"average_queue_time":                                     "average_queue_time_seconds",
	"queue_depth":                                            "queue_depth",

	// Class: Native HA
	"synchronous_log_bytes_sent":                "synchronous_log_sent_bytes",
	"catch-up_log_bytes_sent":                   "catch_up_log_sent_bytes",
	"log_write_average_acknowledgement_latency": "log_write_average_acknowledgement_latency",
	"log_write_average_acknowledgement_size":    "log_write_average_acknowledgement_size",
	"backlog_bytes":                             "backlog_bytes",
	"backlog_average_bytes":                     "backlog_average_bytes",
	"synchronous_compressed_log_bytes_sent":     "synchronous_compressed_log_sent_bytes",
	"catch-up_compressed_log_bytes_sent":        "catch_up_compressed_log_sent_bytes",
	"synchronous_uncompressed_log_bytes_sent":   "synchronous_uncompressed_log_sent_bytes",
	"catch-up_uncompressed_log_bytes_sent":      "catch_up_uncompressed_log_sent_bytes",
}

// This map contains only the additional elements where the heuristic version might not be suitable or
// match well-enough to some other implementations like the MQ Cloud package. For now, you have to
// opt in to using this map with an environment variable, as it would break compatibility with existing dashboards.
var mManual = map[string]string{
	"ram_total_bytes": "ram_size_bytes",

	// Don't need the "mq_" on the front
	"mq_trace_file_system_-_bytes_in_use":  "trace_file_system_in_use_bytes",
	"mq_trace_file_system_-_free_space":    "trace_file_system_free_space_percentage",
	"mq_errors_file_system_-_bytes_in_use": "errors_file_system_in_use_bytes",
	"mq_errors_file_system_-_free_space":   "errors_file_system_free_space_percentage",
	"mq_fdc_file_count":                    "fdc_files",

	// Flip around some of the elements
	"create_durable_subscription_count": "durable_subscription_create_count",
	"delete_durable_subscription_count": "durable_subscription_delete_count",
	"alter_durable_subscription_count":  "durable_subscription_alter_count",
	"resume_durable_subscription_count": "durable_subscription_resume_count",

	"create_non-durable_subscription_count": "non_durable_subscription_create_count",
	"delete_non-durable_subscription_count": "non_durable_subscription_delete_count",

	"failed_create/alter/resume_subscription_count": "failed_subscription_create_alter_resume_count",
	"subscription_delete_failure_count":             "failed_subscription_delete_count",
}

// These are the explicitly-named attributes returned from DISPLAY xxSTATUS or DISPLAY xx
// commands. This map will be used to convert the metric names from the hardcoded defaults in
// each object type's module to a canonical format if the current version is wrong. For now,
// this is still an opt-in function as using it would break compatibility with existing
// dashboards. We start with a map that returns an unchanged name; it may need to evolve.
// Note that some of the keys would be repeated for the different object types (eg "status") but
// those duplicates are commented out.
var mAttr = map[string]string{
	// AMQP CHANNELS
	"clientid": "clientid",
	// "connection_count": "connection_count", // QMgr already has a map for this name
	"messages_rcvd": "messages_rcvd",
	"messages_sent": "messages_sent",

	// CHANNELS
	"batches":             "batches",
	"batchsz_long":        "batchsz_long",
	"batchsz_short":       "batchsz_short",
	"buffers_rcvd":        "buffers_rcvd",
	"buffers_sent":        "buffers_sent",
	"bytes_rcvd":          "bytes_rcvd",
	"bytes_sent":          "bytes_sent",
	"connname":            "connname",
	"cur_inst":            "cur_inst",
	"instance_type":       "instance_type",
	"jobname":             "jobname",
	"attribute_max_inst":  "attribute_max_inst",
	"attribute_max_instc": "attribute_max_instc",
	"messages":            "messages",
	"nettime_long":        "nettime_long",
	"nettime_short":       "nettime_short",
	"rqmname":             "rqmname",
	"time_since_msg":      "time_since_msg",
	"status":              "status",
	"status_squash":       "status_squash",
	"substate":            "substate",
	"type":                "type",
	"xmitq_time_long":     "xmitq_time_long",
	"xmitq_time_short":    "xmitq_time_short",

	// CLUSTER
	"qmtype": "qmtype",
	// "status":  "status",  // already have a map entry for this
	"suspend": "suspend",

	// QMGR
	"active_listeners":         "active_listeners",
	"channel_initiator_status": "channel_initiator_status",
	"command_server_status":    "command_server_status",
	"connection_count":         "connection_count",
	"log_extent_archive":       "log_extent_archive",
	"log_size_archive":         "log_size_archive",
	"log_extent_current":       "log_extent_current",
	"log_extent_media":         "log_extent_media",
	"log_size_media":           "log_size_media",
	"log_extent_restart":       "log_extent_restart",
	"log_size_restart":         "log_size_restart",
	"log_size_reusable":        "log_size_reusable",
	"max_active_channels":      "max_active_channels",
	"max_channels":             "max_channels",
	"max_tcp_channels":         "max_tcp_channels",
	// "status":                   "status",
	"uptime": "uptime",

	// QUEUE
	"qfile_current_size":   "qfile_current_size",
	"qfile_max_size":       "qfile_max_size",
	"depth":                "depth",
	"mqget_count":          "mqget_count",
	"hi_depth":             "hi_depth",
	"mqput_mqput1_count":   "mqput_mqput1_count",
	"input_handles":        "input_handles",
	"attribute_max_depth":  "attribute_max_depth",
	"oldest_message_age":   "oldest_message_age",
	"output_handles":       "output_handles",
	"qtime_long":           "qtime_long",
	"qtime_short":          "qtime_short",
	"time_since_get":       "time_since_get",
	"time_since_put":       "time_since_put",
	"uncommitted_messages": "uncommitted_messages",
	"attribute_usage":      "attribute_usage",

	// SUBSCRIPTIONS
	"subid":                        "subid",
	"messsages_received":           "messsages_received",
	"time_since_message_published": "time_since_message_published",
	"topic":                        "topic",
	// "type":                         "type",

	// TOPICS
	"publisher_count":          "publisher_count",
	"messages_published":       "messages_published",
	"time_since_msg_published": "time_since_msg_published",
	"time_since_msg_received":  "time_since_msg_received",
	// "type":                     "type", // Already have a map entry for this
	"messages_received": "messages_received",
	"subscriber_count":  "subscriber_count",

	// USAGE - pageset/bufferpool attributes
	"pageclass":            "pageclass",
	"buffers_free":         "buffers_free",
	"buffers_free_percent": "buffers_free_percent",
	"location":             "location",
	"buffers_total":        "buffers_total",
	"bufferpool":           "bufferpool",
	"expansion_count":      "expansion_count",
	"pages_nonpersistent":  "pages_nonpersistent",
	"pages_persistent":     "pages_persistent",
	// "status":               "status",
	"pages_total":  "pages_total",
	"pages_unused": "pages_unused",

	"id":   "id",
	"name": "name",
}

// This is automatically called at package startup. It doesn't affect any other
// part of this package.
func init() {
	// Have to deliberately opt into using the non-heuristic maps for now
	if os.Getenv("IBMMQ_MANUAL_METRIC_MAPS") != "" {
		UseManualMetricMaps = true
	} else {
		UseManualMetricMaps = false
	}
}

// Convert the description of the resource publication element into a metric name.
// This has always been done using a heuristic algorithm, but we may want to change some of them
// to a format preferred by the backend (eg OpenTelemetry).
func formatDescription(elem *MonElement) string {
	s := ""

	if UseManualMetricMaps {
		s = formatDescriptionManual(elem.Description)
	}
	if s == "" {
		s = FormatDescriptionHeuristic(elem, true)
	}
	return s
}

func formatDescriptionManual(s string) string {
	desc := strings.ReplaceAll(s, " ", "_")
	desc = strings.ToLower(desc)

	// Is there an overriding metric name
	if s, ok := mManual[desc]; ok {
		return s
	} else {
		return ""
	}
}

// Attributes are already in a preferred format, but we will have a chance to override them.
func formatAttrName(in string) string {

	if UseManualMetricMaps {
		if out, ok := mAttr[in]; ok {
			return out
		}
		logWarn("Attribute \"%s\" does not have a defined metric name in mAttr map", in)
	}

	return in
}

// This function is exported so it can be called during the build/test process for automatic generation
// of some of the mHeur map elements. No longer fully used at runtime as the metric names have been pre-coded in the map.
func FormatDescriptionHeuristic(elem *MonElement, useMap bool) string {

	// The map has been generated once by hand and we will try to use it.
	desc := strings.ReplaceAll(elem.Description, " ", "_")
	desc = strings.ToLower(desc)

	if useMap {
		if s, ok := mHeur[desc]; ok {
			return s
		} else {
			logWarn("Element \"%s\" does not have a defined metric name in mHeur map", elem.Description)
		}
	}

	// If that fails, we go through generating the metric name using this set of rules.
	// From here, we should only usually be running during the build-checking process
	s := elem.Description
	s = strings.Replace(s, " ", "_", -1)
	s = strings.Replace(s, "/", "_", -1)
	s = strings.Replace(s, "-", "_", -1)

	/* Make sure we don't have multiple underscores */
	multiunder := regexp.MustCompile("__*")
	s = multiunder.ReplaceAllLiteralString(s, "_")

	/* make it all lowercase. Not essential, but looks better */
	s = strings.ToLower(s)

	/* Remove all cases of bytes, seconds, count or percentage (we add them back in later) */
	s = strings.Replace(s, "_count", "", -1)
	s = strings.Replace(s, "_bytes", "", -1)
	s = strings.Replace(s, "_byte", "", -1)
	s = strings.Replace(s, "_seconds", "", -1)
	s = strings.Replace(s, "_second", "", -1)
	s = strings.Replace(s, "_percentage", "", -1)

	// Switch round a couple of specific names
	s = strings.Replace(s, "messages_expired", "expired_messages", -1)

	// Add the unit at end
	switch elem.Datatype {
	case ibmmq.MQIAMO_MONITOR_PERCENT, ibmmq.MQIAMO_MONITOR_HUNDREDTHS:
		s = s + "_percentage"
	case ibmmq.MQIAMO_MONITOR_MB, ibmmq.MQIAMO_MONITOR_GB:
		s = s + "_bytes"
	case ibmmq.MQIAMO_MONITOR_MICROSEC:
		s = s + "_seconds"
	default:
		if strings.Contains(s, "_total") {
			/* If we specify it is a total in description put that at the end */
			s = strings.Replace(s, "_total", "", -1)
			s = s + "_total"
		} else if strings.Contains(s, "log_") {
			/* Weird case where the log datatype is not MB or GB but should be bytes */
			s = s + "_bytes"
		}

		// There are some metrics that have both "count" and "byte count" in
		// the descriptions. They were getting mapped to the same string, so
		// we have to ensure uniqueness.
		if strings.Contains(elem.Description, "byte count") {
			s = s + "_bytes"
		} else if strings.HasSuffix(elem.Description, " count") && !strings.Contains(s, "_count") {
			s = s + "_count"
		}
	}

	logTrace("  [%s] in:%s out:%s", "formatDescription", elem.Description, s)

	return s
}
