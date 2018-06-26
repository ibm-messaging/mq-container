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

type metricLookup struct {
	name    string
	enabled bool
}

// generateMetricNamesMap generates metric names mapped from their description
func generateMetricNamesMap() map[string]metricLookup {

	metricNamesMap := map[string]metricLookup{
		"CPU/SystemSummary/CPU load - one minute average":                         metricLookup{"cpu_load_one_minute_average_percentage", true},
		"CPU/SystemSummary/CPU load - five minute average":                        metricLookup{"cpu_load_five_minute_average_percentage", true},
		"CPU/SystemSummary/CPU load - fifteen minute average":                     metricLookup{"cpu_load_fifteen_minute_average_percentage", true},
		"CPU/SystemSummary/System CPU time percentage":                            metricLookup{"system_cpu_time_percentage", true},
		"CPU/SystemSummary/User CPU time percentage":                              metricLookup{"user_cpu_time_percentage", true},
		"CPU/SystemSummary/RAM free percentage":                                   metricLookup{"ram_free_percentage", true},
		"CPU/SystemSummary/RAM total bytes":                                       metricLookup{"system_ram_size_bytes", true},
		"CPU/QMgrSummary/System CPU time - percentage estimate for queue manager": metricLookup{"system_cpu_time_estimate_for_queue_manager_percentage", true},
		"CPU/QMgrSummary/User CPU time - percentage estimate for queue manager":   metricLookup{"user_cpu_time_estimate_for_queue_manager_percentage", true},
		"CPU/QMgrSummary/RAM total bytes - estimate for queue manager":            metricLookup{"ram_usage_estimate_for_queue_manager_bytes", true},
		"DISK/SystemSummary/MQ trace file system - free space":                    metricLookup{"trace_file_system_free_space_percentage", true},
		"DISK/SystemSummary/MQ trace file system - bytes in use":                  metricLookup{"trace_file_system_in_use_bytes", true},
		"DISK/SystemSummary/MQ errors file system - free space":                   metricLookup{"errors_file_system_free_space_percentage", true},
		"DISK/SystemSummary/MQ errors file system - bytes in use":                 metricLookup{"errors_file_system_in_use_bytes", true},
		"DISK/SystemSummary/MQ FDC file count":                                    metricLookup{"fdc_files", true},
		"DISK/QMgrSummary/Queue Manager file system - free space":                 metricLookup{"queue_manager_file_system_free_space_percentage", true},
		"DISK/QMgrSummary/Queue Manager file system - bytes in use":               metricLookup{"queue_manager_file_system_in_use_bytes", true},
		"DISK/Log/Log - logical bytes written":                                    metricLookup{"log_logical_written_bytes_total", true},
		"DISK/Log/Log - physical bytes written":                                   metricLookup{"log_physical_written_bytes_total", true},
		"DISK/Log/Log - current primary space in use":                             metricLookup{"log_primary_space_in_use_percentage", true},
		"DISK/Log/Log - workload primary space utilization":                       metricLookup{"log_workload_primary_space_utilization_percentage", true},
		"DISK/Log/Log - write latency":                                            metricLookup{"log_write_latency_seconds", true},
		"DISK/Log/Log - bytes max":                                                metricLookup{"log_max_bytes", true},
		"DISK/Log/Log - write size":                                               metricLookup{"log_write_size_bytes", true},
		"DISK/Log/Log - bytes in use":                                             metricLookup{"log_in_use_bytes", true},
		"DISK/Log/Log file system - bytes max":                                    metricLookup{"log_file_system_max_bytes", true},
		"DISK/Log/Log file system - bytes in use":                                 metricLookup{"log_file_system_in_use_bytes", true},
		"DISK/Log/Log - bytes occupied by reusable extents":                       metricLookup{"log_occupied_by_reusable_extents_bytes", true},
		"DISK/Log/Log - bytes occupied by extents waiting to be archived":         metricLookup{"log_occupied_by_extents_waiting_to_be_archived_bytes", true},
		"DISK/Log/Log - bytes required for media recovery":                        metricLookup{"log_required_for_media_recovery_bytes", true},
		"STATMQI/SUBSCRIBE/Create durable subscription count":                     metricLookup{"durable_subscription_create_total", true},
		"STATMQI/SUBSCRIBE/Alter durable subscription count":                      metricLookup{"durable_subscription_alter_total", true},
		"STATMQI/SUBSCRIBE/Resume durable subscription count":                     metricLookup{"durable_subscription_resume_total", true},
		"STATMQI/SUBSCRIBE/Delete durable subscription count":                     metricLookup{"durable_subscription_delete_total", true},
		"STATMQI/SUBSCRIBE/Create non-durable subscription count":                 metricLookup{"non_durable_subscription_create_total", true},
		"STATMQI/SUBSCRIBE/Delete non-durable subscription count":                 metricLookup{"non_durable_subscription_delete_total", true},
		"STATMQI/SUBSCRIBE/Failed create/alter/resume subscription count":         metricLookup{"failed_subscription_create_alter_resume_total", true},
		"STATMQI/SUBSCRIBE/Subscription delete failure count":                     metricLookup{"failed_subscription_delete_total", true},
		"STATMQI/SUBSCRIBE/MQSUBRQ count":                                         metricLookup{"mqsubrq_total", true},
		"STATMQI/SUBSCRIBE/Failed MQSUBRQ count":                                  metricLookup{"failed_mqsubrq_total", true},
		"STATMQI/SUBSCRIBE/Durable subscriber - high water mark":                  metricLookup{"durable_subscriber_high_water_mark", false},
		"STATMQI/SUBSCRIBE/Durable subscriber - low water mark":                   metricLookup{"durable_subscriber_low_water_mark", false},
		"STATMQI/SUBSCRIBE/Non-durable subscriber - high water mark":              metricLookup{"non_durable_subscriber_high_water_mark", false},
		"STATMQI/SUBSCRIBE/Non-durable subscriber - low water mark":               metricLookup{"non_durable_subscriber_low_water_mark", false},
		"STATMQI/PUBLISH/Topic MQPUT/MQPUT1 interval total":                       metricLookup{"topic_mqput_mqput1_total", true},
		"STATMQI/PUBLISH/Interval total topic bytes put":                          metricLookup{"topic_put_bytes_total", true},
		"STATMQI/PUBLISH/Failed topic MQPUT/MQPUT1 count":                         metricLookup{"failed_topic_mqput_mqput1_total", true},
		"STATMQI/PUBLISH/Persistent - topic MQPUT/MQPUT1 count":                   metricLookup{"persistent_topic_mqput_mqput1_total", true},
		"STATMQI/PUBLISH/Non-persistent - topic MQPUT/MQPUT1 count":               metricLookup{"non_persistent_topic_mqput_mqput1_total", true},
		"STATMQI/PUBLISH/Published to subscribers - message count":                metricLookup{"published_to_subscribers_message_total", true},
		"STATMQI/PUBLISH/Published to subscribers - byte count":                   metricLookup{"published_to_subscribers_bytes_total", true},
		"STATMQI/CONNDISC/MQCONN/MQCONNX count":                                   metricLookup{"mqconn_mqconnx_total", true},
		"STATMQI/CONNDISC/Failed MQCONN/MQCONNX count":                            metricLookup{"failed_mqconn_mqconnx_total", true},
		"STATMQI/CONNDISC/MQDISC count":                                           metricLookup{"mqdisc_total", true},
		"STATMQI/CONNDISC/Concurrent connections - high water mark":               metricLookup{"concurrent_connections_high_water_mark", false},
		"STATMQI/OPENCLOSE/MQOPEN count":                                          metricLookup{"mqopen_total", true},
		"STATMQI/OPENCLOSE/Failed MQOPEN count":                                   metricLookup{"failed_mqopen_total", true},
		"STATMQI/OPENCLOSE/MQCLOSE count":                                         metricLookup{"mqclose_total", true},
		"STATMQI/OPENCLOSE/Failed MQCLOSE count":                                  metricLookup{"failed_mqclose_total", true},
		"STATMQI/INQSET/MQINQ count":                                              metricLookup{"mqinq_total", true},
		"STATMQI/INQSET/Failed MQINQ count":                                       metricLookup{"failed_mqinq_total", true},
		"STATMQI/INQSET/MQSET count":                                              metricLookup{"mqset_total", true},
		"STATMQI/INQSET/Failed MQSET count":                                       metricLookup{"failed_mqset_total", true},
		"STATMQI/PUT/Persistent message MQPUT count":                              metricLookup{"persistent_message_mqput_total", true},
		"STATMQI/PUT/Persistent message MQPUT1 count":                             metricLookup{"persistent_message_mqput1_total", true},
		"STATMQI/PUT/Put persistent messages - byte count":                        metricLookup{"persistent_message_put_bytes_total", true},
		"STATMQI/PUT/Non-persistent message MQPUT count":                          metricLookup{"non_persistent_message_mqput_total", true},
		"STATMQI/PUT/Non-persistent message MQPUT1 count":                         metricLookup{"non_persistent_message_mqput1_total", true},
		"STATMQI/PUT/Put non-persistent messages - byte count":                    metricLookup{"non_persistent_message_put_bytes_total", true},
		"STATMQI/PUT/Interval total MQPUT/MQPUT1 count":                           metricLookup{"mqput_mqput1_total", true},
		"STATMQI/PUT/Interval total MQPUT/MQPUT1 byte count":                      metricLookup{"mqput_mqput1_bytes_total", true},
		"STATMQI/PUT/Failed MQPUT count":                                          metricLookup{"failed_mqput_total", true},
		"STATMQI/PUT/Failed MQPUT1 count":                                         metricLookup{"failed_mqput1_total", true},
		"STATMQI/PUT/MQSTAT count":                                                metricLookup{"mqstat_total", true},
		"STATMQI/GET/Persistent message destructive get - count":                  metricLookup{"persistent_message_destructive_get_total", true},
		"STATMQI/GET/Persistent message browse - count":                           metricLookup{"persistent_message_browse_total", true},
		"STATMQI/GET/Got persistent messages - byte count":                        metricLookup{"persistent_message_get_bytes_total", true},
		"STATMQI/GET/Persistent message browse - byte count":                      metricLookup{"persistent_message_browse_bytes_total", true},
		"STATMQI/GET/Non-persistent message destructive get - count":              metricLookup{"non_persistent_message_destructive_get_total", true},
		"STATMQI/GET/Non-persistent message browse - count":                       metricLookup{"non_persistent_message_browse_total", true},
		"STATMQI/GET/Got non-persistent messages - byte count":                    metricLookup{"non_persistent_message_get_bytes_total", true},
		"STATMQI/GET/Non-persistent message browse - byte count":                  metricLookup{"non_persistent_message_browse_bytes_total", true},
		"STATMQI/GET/Interval total destructive get- count":                       metricLookup{"destructive_get_total", true},
		"STATMQI/GET/Interval total destructive get - byte count":                 metricLookup{"destructive_get_bytes_total", true},
		"STATMQI/GET/Failed MQGET - count":                                        metricLookup{"failed_mqget_total", true},
		"STATMQI/GET/Failed browse count":                                         metricLookup{"failed_browse_total", true},
		"STATMQI/GET/MQCTL count":                                                 metricLookup{"mqctl_total", true},
		"STATMQI/GET/Expired message count":                                       metricLookup{"expired_message_total", true},
		"STATMQI/GET/Purged queue count":                                          metricLookup{"purged_queue_total", true},
		"STATMQI/GET/MQCB count":                                                  metricLookup{"mqcb_total", true},
		"STATMQI/GET/Failed MQCB count":                                           metricLookup{"failed_mqcb_total", true},
		"STATMQI/SYNCPOINT/Commit count":                                          metricLookup{"commit_total", true},
		"STATMQI/SYNCPOINT/Rollback count":                                        metricLookup{"rollback_total", true},
	}
	return metricNamesMap
}
