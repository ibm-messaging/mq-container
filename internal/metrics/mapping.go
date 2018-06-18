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
		"CPU/SystemSummary/CPU load - one minute average":                         setMetricName("cpu_load_one_minute_average_percentage", true),
		"CPU/SystemSummary/CPU load - five minute average":                        setMetricName("cpu_load_five_minute_average_percentage", true),
		"CPU/SystemSummary/CPU load - fifteen minute average":                     setMetricName("cpu_load_fifteen_minute_average_percentage", true),
		"CPU/SystemSummary/System CPU time percentage":                            setMetricName("system_cpu_time_percentage", true),
		"CPU/SystemSummary/User CPU time percentage":                              setMetricName("user_cpu_time_percentage", true),
		"CPU/SystemSummary/RAM free percentage":                                   setMetricName("ram_free_percentage", true),
		"CPU/SystemSummary/RAM total bytes":                                       setMetricName("system_ram_size_bytes", true),
		"CPU/QMgrSummary/System CPU time - percentage estimate for queue manager": setMetricName("system_cpu_time_estimate_for_queue_manager_percentage", true),
		"CPU/QMgrSummary/User CPU time - percentage estimate for queue manager":   setMetricName("user_cpu_time_estimate_for_queue_manager_percentage", true),
		"CPU/QMgrSummary/RAM total bytes - estimate for queue manager":            setMetricName("ram_usage_estimate_for_queue_manager_bytes", true),
		"DISK/SystemSummary/MQ trace file system - free space":                    setMetricName("trace_file_system_free_space_percentage", true),
		"DISK/SystemSummary/MQ trace file system - bytes in use":                  setMetricName("trace_file_system_in_use_bytes", true),
		"DISK/SystemSummary/MQ errors file system - free space":                   setMetricName("errors_file_system_free_space_percentage", true),
		"DISK/SystemSummary/MQ errors file system - bytes in use":                 setMetricName("errors_file_system_in_use_bytes", true),
		"DISK/SystemSummary/MQ FDC file count":                                    setMetricName("fdc_files", true),
		"DISK/QMgrSummary/Queue Manager file system - free space":                 setMetricName("queue_manager_file_system_free_space_percentage", true),
		"DISK/QMgrSummary/Queue Manager file system - bytes in use":               setMetricName("queue_manager_file_system_in_use_bytes", true),
		"DISK/Log/Log - logical bytes written":                                    setMetricName("log_logical_written_bytes_interval_total", true),
		"DISK/Log/Log - physical bytes written":                                   setMetricName("log_physical_written_bytes_interval_total", true),
		"DISK/Log/Log - current primary space in use":                             setMetricName("log_primary_space_in_use_percentage", true),
		"DISK/Log/Log - workload primary space utilization":                       setMetricName("log_workload_primary_space_utilization_percentage", true),
		"DISK/Log/Log - write latency":                                            setMetricName("log_write_latency_seconds", true),
		"DISK/Log/Log - bytes max":                                                setMetricName("log_max_bytes", true),
		"DISK/Log/Log - write size":                                               setMetricName("log_write_size_bytes", true),
		"DISK/Log/Log - bytes in use":                                             setMetricName("log_in_use_bytes", true),
		"DISK/Log/Log file system - bytes max":                                    setMetricName("log_file_system_max_bytes", true),
		"DISK/Log/Log file system - bytes in use":                                 setMetricName("log_file_system_in_use_bytes", true),
		"DISK/Log/Log - bytes occupied by reusable extents":                       setMetricName("log_occupied_by_reusable_extents_bytes", true),
		"DISK/Log/Log - bytes occupied by extents waiting to be archived":         setMetricName("log_occupied_by_extents_waiting_to_be_archived_bytes", true),
		"DISK/Log/Log - bytes required for media recovery":                        setMetricName("log_required_for_media_recovery_bytes", true),
		"STATMQI/SUBSCRIBE/Create durable subscription count":                     setMetricName("durable_subscription_create_interval_total", true),
		"STATMQI/SUBSCRIBE/Alter durable subscription count":                      setMetricName("durable_subscription_alter_interval_total", true),
		"STATMQI/SUBSCRIBE/Resume durable subscription count":                     setMetricName("durable_subscription_resume_interval_total", true),
		"STATMQI/SUBSCRIBE/Delete durable subscription count":                     setMetricName("durable_subscription_delete_interval_total", true),
		"STATMQI/SUBSCRIBE/Create non-durable subscription count":                 setMetricName("non_durable_subscription_create_interval_total", true),
		"STATMQI/SUBSCRIBE/Delete non-durable subscription count":                 setMetricName("non_durable_subscription_delete_interval_total", true),
		"STATMQI/SUBSCRIBE/Failed create/alter/resume subscription count":         setMetricName("failed_subscription_create_alter_resume_interval_total", true),
		"STATMQI/SUBSCRIBE/Subscription delete failure count":                     setMetricName("failed_subscription_delete_interval_total", true),
		"STATMQI/SUBSCRIBE/MQSUBRQ count":                                         setMetricName("mqsubrq_interval_total", true),
		"STATMQI/SUBSCRIBE/Failed MQSUBRQ count":                                  setMetricName("failed_mqsubrq_interval_total", true),
		"STATMQI/SUBSCRIBE/Durable subscriber - high water mark":                  setMetricName("durable_subscriber_high_water_mark", true),
		"STATMQI/SUBSCRIBE/Durable subscriber - low water mark":                   setMetricName("durable_subscriber_low_water_mark", true),
		"STATMQI/SUBSCRIBE/Non-durable subscriber - high water mark":              setMetricName("non_durable_subscriber_high_water_mark", true),
		"STATMQI/SUBSCRIBE/Non-durable subscriber - low water mark":               setMetricName("non_durable_subscriber_low_water_mark", true),
		"STATMQI/PUBLISH/Topic MQPUT/MQPUT1 interval total":                       setMetricName("topic_mqput_mqput1_interval_total", true),
		"STATMQI/PUBLISH/Interval total topic bytes put":                          setMetricName("topic_put_bytes_interval_total", true),
		"STATMQI/PUBLISH/Failed topic MQPUT/MQPUT1 count":                         setMetricName("failed_topic_mqput_mqput1_interval_total", true),
		"STATMQI/PUBLISH/Persistent - topic MQPUT/MQPUT1 count":                   setMetricName("persistent_topic_mqput_mqput1_interval_total", true),
		"STATMQI/PUBLISH/Non-persistent - topic MQPUT/MQPUT1 count":               setMetricName("non_persistent_topic_mqput_mqput1_interval_total", true),
		"STATMQI/PUBLISH/Published to subscribers - message count":                setMetricName("published_to_subscribers_message_interval_total", true),
		"STATMQI/PUBLISH/Published to subscribers - byte count":                   setMetricName("published_to_subscribers_bytes_interval_total", true),
		"STATMQI/CONNDISC/MQCONN/MQCONNX count":                                   setMetricName("mqconn_mqconnx_interval_total", true),
		"STATMQI/CONNDISC/Failed MQCONN/MQCONNX count":                            setMetricName("failed_mqconn_mqconnx_interval_total", true),
		"STATMQI/CONNDISC/MQDISC count":                                           setMetricName("mqdisc_interval_total", true),
		"STATMQI/CONNDISC/Concurrent connections - high water mark":               setMetricName("concurrent_connections_high_water_mark", true),
		"STATMQI/OPENCLOSE/MQOPEN count":                                          setMetricName("mqopen_interval_total", true),
		"STATMQI/OPENCLOSE/Failed MQOPEN count":                                   setMetricName("failed_mqopen_interval_total", true),
		"STATMQI/OPENCLOSE/MQCLOSE count":                                         setMetricName("mqclose_interval_total", true),
		"STATMQI/OPENCLOSE/Failed MQCLOSE count":                                  setMetricName("failed_mqclose_interval_total", true),
		"STATMQI/INQSET/MQINQ count":                                              setMetricName("mqinq_interval_total", true),
		"STATMQI/INQSET/Failed MQINQ count":                                       setMetricName("failed_mqinq_interval_total", true),
		"STATMQI/INQSET/MQSET count":                                              setMetricName("mqset_interval_total", true),
		"STATMQI/INQSET/Failed MQSET count":                                       setMetricName("failed_mqset_interval_total", true),
		"STATMQI/PUT/Persistent message MQPUT count":                              setMetricName("persistent_message_mqput_interval_total", true),
		"STATMQI/PUT/Persistent message MQPUT1 count":                             setMetricName("persistent_message_mqput1_interval_total", true),
		"STATMQI/PUT/Put persistent messages - byte count":                        setMetricName("persistent_message_put_bytes_interval_total", true),
		"STATMQI/PUT/Non-persistent message MQPUT count":                          setMetricName("non_persistent_message_mqput_interval_total", true),
		"STATMQI/PUT/Non-persistent message MQPUT1 count":                         setMetricName("non_persistent_message_mqput1_interval_total", true),
		"STATMQI/PUT/Put non-persistent messages - byte count":                    setMetricName("non_persistent_message_put_bytes_interval_total", true),
		"STATMQI/PUT/Interval total MQPUT/MQPUT1 count":                           setMetricName("mqput_mqput1_interval_total", true),
		"STATMQI/PUT/Interval total MQPUT/MQPUT1 byte count":                      setMetricName("mqput_mqput1_bytes_interval_total", true),
		"STATMQI/PUT/Failed MQPUT count":                                          setMetricName("failed_mqput_interval_total", true),
		"STATMQI/PUT/Failed MQPUT1 count":                                         setMetricName("failed_mqput1_interval_total", true),
		"STATMQI/PUT/MQSTAT count":                                                setMetricName("mqstat_interval_total", true),
		"STATMQI/GET/Persistent message destructive get - count":                  setMetricName("persistent_message_destructive_get_interval_total", true),
		"STATMQI/GET/Persistent message browse - count":                           setMetricName("persistent_message_browse_interval_total", true),
		"STATMQI/GET/Got persistent messages - byte count":                        setMetricName("persistent_message_get_bytes_interval_total", true),
		"STATMQI/GET/Persistent message browse - byte count":                      setMetricName("persistent_message_browse_bytes_interval_total", true),
		"STATMQI/GET/Non-persistent message destructive get - count":              setMetricName("non_persistent_message_destructive_get_interval_total", true),
		"STATMQI/GET/Non-persistent message browse - count":                       setMetricName("non_persistent_message_browse_interval_total", true),
		"STATMQI/GET/Got non-persistent messages - byte count":                    setMetricName("non_persistent_message_get_bytes_interval_total", true),
		"STATMQI/GET/Non-persistent message browse - byte count":                  setMetricName("non_persistent_message_browse_bytes_interval_total", true),
		"STATMQI/GET/Interval total destructive get- count":                       setMetricName("destructive_get_interval_total", true),
		"STATMQI/GET/Interval total destructive get - byte count":                 setMetricName("destructive_get_bytes_interval_total", true),
		"STATMQI/GET/Failed MQGET - count":                                        setMetricName("failed_mqget_interval_total", true),
		"STATMQI/GET/Failed browse count":                                         setMetricName("failed_browse_interval_total", true),
		"STATMQI/GET/MQCTL count":                                                 setMetricName("mqctl_interval_total", true),
		"STATMQI/GET/Expired message count":                                       setMetricName("expired_message_interval_total", true),
		"STATMQI/GET/Purged queue count":                                          setMetricName("purged_queue_interval_total", true),
		"STATMQI/GET/MQCB count":                                                  setMetricName("mqcb_interval_total", true),
		"STATMQI/GET/Failed MQCB count":                                           setMetricName("failed_mqcb_interval_total", true),
		"STATMQI/SYNCPOINT/Commit count":                                          setMetricName("commit_interval_total", true),
		"STATMQI/SYNCPOINT/Rollback count":                                        setMetricName("rollback_interval_total", true),
	}
	return metricNamesMap
}

// setMetricName sets the metric name & specifies if the metric is enabled
func setMetricName(name string, enabled bool) metricLookup {
	return metricLookup{
		name:    name,
		enabled: enabled,
	}
}
