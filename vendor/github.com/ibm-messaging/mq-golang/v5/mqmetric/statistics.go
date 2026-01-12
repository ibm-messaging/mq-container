package mqmetric

/*
  Copyright (c) IBM Corporation 2016, 2025

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

import (
	"strings"

	mq "github.com/ibm-messaging/mq-golang/v5/ibmmq"
)

// This maps the MQIAMO values in the statistics messages to a metric name. Some of the
// elements are arrays, so we also need to know which index corresponds to a particular metric. There are
// separate maps here for qmgr and queue-level metrics, even if some of the elements are duplicated between them.
// There's enough variation between the metrics produced via this mechanism and the resource publications that I'm not
// going to try to keep the metric names the same where they refer to the same number.
//
// Where there is more than one metric name in a list, then the final entry is not directly given by the qmgr, but is instead
// calculated as a total. So we can easily see the total number of messages put, not just split into P and NP (though those individuals are
// still available too).
//
// The names in this list get "_count" added during initialisation if appropriate
var qStatisticsAttrsMap = map[int32][]string{
	mq.MQIAMO_Q_MIN_DEPTH: []string{"min_depth"},
	mq.MQIAMO_Q_MAX_DEPTH: []string{"max_depth"},
	mq.MQIAMO_Q_TIME_AVG:  []string{"average_queue_time_seconds"},

	mq.MQIAMO_PUTS:         []string{"mqput_non_persistent_message", "mqput_persistent_message", "mqput_message"},
	mq.MQIAMO_PUTS_FAILED:  []string{"failed_mqput"},
	mq.MQIAMO_PUT1S:        []string{"mqput1_non_persistent_message", "mqput1_persistent_message", "mqput1_message"},
	mq.MQIAMO_PUT1S_FAILED: []string{"failed_mqput1"},
	mq.MQIAMO64_PUT_BYTES:  []string{"put_non_persistent_message_bytes", "put_persistent_message_bytes", "put_message_bytes"},

	mq.MQIAMO_GETS:        []string{"destructive_mqget_non_persistent_message", "destructive_mqget_persistent_message", "destructive_mqget_message"},
	mq.MQIAMO64_GET_BYTES: []string{"destructive_mqget_non_persistent_message_bytes", "destructive_mqget_persistent_message_bytes", "destructive_mqget_message_bytes"},
	mq.MQIAMO_GETS_FAILED: []string{"failed_destructive_mqget"},

	mq.MQIAMO_BROWSES:        []string{"mqget_browse_non_persistent_message", "mqget_browse_persistent_message", "mqget_browse_message"},
	mq.MQIAMO64_BROWSE_BYTES: []string{"mqget_browse_non_persistent_bytes", "mqget_browse_persistent_bytes", "mqget_browse_bytes"},
	mq.MQIAMO_BROWSES_FAILED: []string{"failed_mqget_browse"},

	// The not_queued metric here is a number, while the published equivalent is a percentage
	mq.MQIAMO_MSGS_NOT_QUEUED: []string{"queue_avoided_put"},
	mq.MQIAMO_MSGS_EXPIRED:    []string{"expired_message"},
	mq.MQIAMO_MSGS_PURGED:     []string{"purged_message"},
}

var qmgrStatisticsAttrsMap = map[int32][]string{
	mq.MQIAMO_CONNS:        []string{"mqconn_mqconnx"},
	mq.MQIAMO_CONNS_FAILED: []string{"failed_mqconn_mqconnx"},
	mq.MQIAMO_CONNS_MAX:    []string{"concurrent_connections_high_water_mark"},
	mq.MQIAMO_DISCS:        []string{"mqdisc_normal", "mqdisc_implicit", "mqdisc_qmgr", "mqdisc"},
	// The returned array is indexed by object type, even if you can't actually MQOPEN some of them
	mq.MQIAMO_OPENS: []string{
		"mqopen_unknown",
		"mqopen_queue",
		"mqopen_namelist",
		"mqopen_process",
		"mqopen_stgclass",
		"mqopen_qmgr",
		"mqopen_channel",
		"mqopen_authinfo",
		"mqopen_topic",
		"mqopen_comminfo",
		"mqopen_cfstruc",
		"mqopen_listener",
		"mqopen_service",
		"mqopen",
	},
	mq.MQIAMO_OPENS_FAILED: []string{
		"failed_mqopen_unknown",
		"failed_mqopen_queue",
		"failed_mqopen_namelist",
		"failed_mqopen_process",
		"failed_mqopen_stgclass",
		"failed_mqopen_qmgr",
		"failed_mqopen_channel",
		"failed_mqopen_authinfo",
		"failed_mqopen_topic",
		"failed_mqopen_comminfo",
		"failed_mqopen_cfstruc",
		"failed_mqopen_listener",
		"failed_mqopen_service",
		"failed_mqopen",
	},

	mq.MQIAMO_CLOSES: []string{
		"mqclose_unknown",
		"mqclose_queue",
		"mqclose_namelist",
		"mqclose_process",
		"mqclose_stgclass",
		"mqclose_qmgr",
		"mqclose_channel",
		"mqclose_authinfo",
		"mqclose_topic",
		"mqclose_comminfo",
		"mqclose_cfstruc",
		"mqclose_listener",
		"mqclose_service",
		"mqclose",
	},
	mq.MQIAMO_CLOSES_FAILED: []string{
		"failed_mqclose_unknown",
		"failed_mqclose_queue",
		"failed_mqclose_namelist",
		"failed_mqclose_process",
		"failed_mqclose_stgclass",
		"failed_mqclose_qmgr",
		"failed_mqclose_channel",
		"failed_mqclose_authinfo",
		"failed_mqclose_topic",
		"failed_mqclose_comminfo",
		"failed_mqclose_cfstruc",
		"failed_mqclose_listener",
		"failed_mqclose_service",
		"failed_mqclose",
	},

	mq.MQIAMO_INQS: []string{
		"mqinq_unknown",
		"mqinq_queue",
		"mqinq_namelist",
		"mqinq_process",
		"mqinq_stgclass",
		"mqinq_qmgr",
		"mqinq_channel",
		"mqinq_authinfo",
		"mqinq_topic",
		"mqinq_comminfo",
		"mqinq_cfstruc",
		"mqinq_listener",
		"mqinq_service",
		"mqinq",
	},
	mq.MQIAMO_INQS_FAILED: []string{
		"failed_mqinq_unknown",
		"failed_mqinq_queue",
		"failed_mqinq_namelist",
		"failed_mqinq_process",
		"failed_mqinq_stgclass",
		"failed_mqinq_qmgr",
		"failed_mqinq_channel",
		"failed_mqinq_authinfo",
		"failed_mqinq_topic",
		"failed_mqinq_comminfo",
		"failed_mqinq_cfstruc",
		"failed_mqinq_listener",
		"failed_mqinq_service",
		"failed_mqinq",
	},

	mq.MQIAMO_SETS: []string{
		"mqset_unknown",
		"mqset_queue",
		"mqset_namelist",
		"mqset_process",
		"mqset_stgclass",
		"mqset_qmgr",
		"mqset_channel",
		"mqset_authinfo",
		"mqset_topic",
		"mqset_comminfo",
		"mqset_cfstruc",
		"mqset_listener",
		"mqset_service",
		"mqset",
	},
	mq.MQIAMO_SETS_FAILED: []string{
		"failed_mqset_unknown",
		"failed_mqset_queue",
		"failed_mqset_namelist",
		"failed_mqset_process",
		"failed_mqset_stgclass",
		"failed_mqset_qmgr",
		"failed_mqset_channel",
		"failed_mqset_authinfo",
		"failed_mqset_topic",
		"failed_mqset_comminfo",
		"failed_mqset_cfstruc",
		"failed_mqset_listener",
		"failed_mqset_service",
		"failed_mqset",
	},

	mq.MQIAMO_PUTS:         []string{"mqput_non_persistent_message", "mqput_persistent_message", "mqput_message"},
	mq.MQIAMO_PUTS_FAILED:  []string{"failed_mqput"},
	mq.MQIAMO_PUT1S:        []string{"mqput1_non_persistent_message", "mqput1_persistent_message", "mqput1_message"},
	mq.MQIAMO_PUT1S_FAILED: []string{"failed_mqput1"},
	mq.MQIAMO64_PUT_BYTES:  []string{"put_non_persistent_message_bytes", "put_persistent_message_bytes", "put_message_bytes"},

	mq.MQIAMO_GETS:        []string{"destructive_mqget_non_persistent_message", "destructive_mqget_persistent_message", "destructive_mqget_message"},
	mq.MQIAMO64_GET_BYTES: []string{"destructive_mqget_non_persistent_message_bytes", "destructive_mqget_persistent_message_bytes", "destructive_mqget_message_bytes"},
	mq.MQIAMO_GETS_FAILED: []string{"failed_destructive_mqget"},

	mq.MQIAMO_BROWSES:        []string{"mqget_browse_non_persistent_message", "mqget_browse_persistent_message", "mqget_browse_message"},
	mq.MQIAMO64_BROWSE_BYTES: []string{"mqget_browse_non_persistent_bytes", "mqget_browse_persistent_bytes", "mqget_browse_bytes"},
	mq.MQIAMO_BROWSES_FAILED: []string{"failed_mqget_browse"},

	mq.MQIAMO_COMMITS:        []string{"mqcmit"},
	mq.MQIAMO_COMMITS_FAILED: []string{"failed_mqcmit"},
	mq.MQIAMO_BACKOUTS:       []string{"mqback"},

	mq.MQIAMO_MSGS_EXPIRED: []string{"expired_message"},
	mq.MQIAMO_MSGS_PURGED:  []string{"purged_messages"},

	mq.MQIAMO_SUBS_DUR:      []string{"mqsub_durable_create", "mqsub_durable_alter", "mqsub_durable_resume", "mqsub_durable"},
	mq.MQIAMO_SUBS_NDUR:     []string{"mqsub_non_durable_create", "mqsub_non_durable_alter", "mqsub_non_durable_resume", "mqsub_non_durable"},
	mq.MQIAMO_SUBS_FAILED:   []string{"failed_mqsub"},
	mq.MQIAMO_UNSUBS_DUR:    []string{"unsubscribe_durable_keep", "unsubscribe_durable_remove", "unsubscribe_durable"},
	mq.MQIAMO_UNSUBS_NDUR:   []string{"unsubscribe_non_durable_keep", "unsubscribe_non_durable_remove", "unsubscribe_non_durable"},
	mq.MQIAMO_UNSUBS_FAILED: []string{"failed_unsubscribe"},

	mq.MQIAMO_SUBRQS:        []string{"mqsubrq"},
	mq.MQIAMO_SUBRQS_FAILED: []string{"failed_mqsubrq"},
	mq.MQIAMO_CBS:           []string{"mqcb_register", "mqcb_deregister", "mqcb_resume", "mqcb_suspend", "mqcb"},
	mq.MQIAMO_CBS_FAILED:    []string{"failed_mqcb"},
	mq.MQIAMO_CTLS:          []string{"mqctl_start", "mqctl_stop", "mqctl_resume", "mqctl_suspect", "mqctl"},
	mq.MQIAMO_CTLS_FAILED:   []string{"failed_mqctl"},
	mq.MQIAMO_STATS:         []string{"mqstat"},
	mq.MQIAMO_STATS_FAILED:  []string{"failed_mqstat"},

	mq.MQIAMO_SUB_DUR_HIGHWATER:  []string{"mqsub_durable_highwater_all", "mqsub_durable_highwater_api", "mqsub_durable_highwater_admin", "mqsub_durable_highwater_proxy"},
	mq.MQIAMO_SUB_DUR_LOWWATER:   []string{"mqsub_durable_lowwater_all", "mqsub_durable_lowwater_api", "mqsub_durable_lowwater_admin", "mqsub_durable_lowwater_proxy"},
	mq.MQIAMO_SUB_NDUR_HIGHWATER: []string{"mqsub_non_durable_highwater_all", "mqsub_non_durable_highwater_api", "mqsub_non_durable_highwater_admin", "mqsub_non_durable_highwater_proxy"},
	mq.MQIAMO_SUB_NDUR_LOWWATER:  []string{"mqsub_non_durable_lowwater_all", "mqsub_non_durable_lowwater_api", "mqsub_non_durable_lowwater_admin", "mqsub_non_durable_lowwater_proxy"},

	mq.MQIAMO_TOPIC_PUTS:          []string{"mqput_topic_non_persistent_messages", "mqput_topic_persistent_messages", "mqput_topic_messages"},
	mq.MQIAMO_TOPIC_PUTS_FAILED:   []string{"failed_mqput_topic"},
	mq.MQIAMO_TOPIC_PUT1S:         []string{"mqput1_topic_non_persistent_messages", "mqput1_topic_persistent_messages", "mqput1_topic_messages"},
	mq.MQIAMO_TOPIC_PUT1S_FAILED:  []string{"failed_mqput1_topic"},
	mq.MQIAMO64_TOPIC_PUT_BYTES:   []string{"put_topic_non_persistent_message_bytes", "put_topic_persistent_message_bytes", "put_topic_message_bytes"},
	mq.MQIAMO_PUBLISH_MSG_COUNT:   []string{"publish_non_persistent_messages", "publish_persistent_messages", "publish_messages"},
	mq.MQIAMO64_PUBLISH_MSG_BYTES: []string{"publish_non_persistent_message_bytes", "publish_persistent_message_bytes", "publish_message_bytes"},
}

// Create the attributes to represent each metric. They are nearly all "delta" values. The few "instant" or "gauge" values are called
// out here explicitly. Also add the "_count" to the delta/Counter metric names.
func initStatisticsAttrs(ci *connectionInfo) {
	// os := &ci.objectStatus[OT_Q]
	traceEntry("initStatisticsAttrs")

	st := GetObjectStatistics(GetConnectionKey(), OT_Q)
	st.Attributes = make(map[string]*StatusAttribute)

	for k, v := range qStatisticsAttrsMap {
		for i := 0; i < len(v); i++ {
			switch k {
			case mq.MQIAMO_Q_MIN_DEPTH, mq.MQIAMO_Q_MAX_DEPTH, mq.MQIAMO64_AVG_Q_TIME:
				st.Attributes[v[i]] = newStatusAttribute(v[i], v[i], DUMMY_PCFATTR)
				st.Attributes[v[i]].Delta = false
			default:
				if !strings.HasSuffix(v[i], "_count") {
					v[i] += "_count"
				}
				st.Attributes[v[i]] = newStatusAttribute(v[i], v[i], DUMMY_PCFATTR)
				st.Attributes[v[i]].Delta = true
			}
		}
	}

	st.Attributes[ATTR_Q_NAME] = newPseudoStatusAttribute(ATTR_Q_NAME, "Queue Name")
	//logDebug("initStatisticsAttrs: q=%+v", st.Attributes)

	st = GetObjectStatistics(GetConnectionKey(), OT_Q_MGR)
	st.Attributes = make(map[string]*StatusAttribute)

	for k, v := range qmgrStatisticsAttrsMap {
		for i := 0; i < len(v); i++ {
			switch k {
			case mq.MQIAMO_CONNS_MAX,
				mq.MQIAMO_SUB_DUR_HIGHWATER,
				mq.MQIAMO_SUB_NDUR_HIGHWATER,
				mq.MQIAMO_SUB_DUR_LOWWATER,
				mq.MQIAMO_SUB_NDUR_LOWWATER:
				st.Attributes[v[i]] = newStatusAttribute(v[i], v[i], DUMMY_PCFATTR)
				st.Attributes[v[i]].Delta = false

			default:
				if !strings.HasSuffix(v[i], "_count") {
					v[i] += "_count"
				}
				st.Attributes[v[i]] = newStatusAttribute(v[i], v[i], DUMMY_PCFATTR)
				st.Attributes[v[i]].Delta = true
			}
		}
	}

	//logDebug("initStatisticsAttrs: qm=%+v", st.Attributes)
	traceExit("initStatisticsAttrs", 0)

}

// Read all the available statistics messages
func getStatisticsMessages(ci *connectionInfo) {
	var err error
	datalen := 0

	ci.statisticsCount = 0

	for err == nil {
		datalen, err = getStatisticsMessage(ci)
		if err == nil {
			parseStatisticsMsg(ci.si.statisticsQBuf, datalen)
		}
	}
}

// Get each message in turn, making sure we have a large enough buffer to avoid truncation
func getStatisticsMessage(ci *connectionInfo) (int, error) {
	var md *mq.MQMD
	var err error

	traceEntry("getStatisticsMessage")

	msgLen := 0
	hObj := ci.si.statisticsQObj
	buf := ci.si.statisticsQBuf

	for trunc := true; trunc; {
		// Now get the response. Reset the MD and GMO on each iteration to ensure we don't get mixed up
		// with anything that gets modified (like the CCSID) even on failed/truncated GETs.
		md = mq.NewMQMD()
		gmo := mq.NewMQGMO()
		gmo.Options = mq.MQGMO_NO_SYNCPOINT
		gmo.Options |= mq.MQGMO_FAIL_IF_QUIESCING
		gmo.Options |= mq.MQGMO_NO_WAIT
		gmo.Options |= mq.MQGMO_CONVERT

		//gmo.Options |= mq.MQGMO_BROWSE_NEXT

		// logTrace("clearQWithoutTruncation: Trying MQGET with clearQBuffer size %d ", len(clearQBuf))
		msgLen, err = hObj.Get(md, gmo, ci.si.statisticsQBuf)
		if err != nil {
			mqreturn := err.(*mq.MQReturn)
			if mqreturn.MQCC != mq.MQCC_OK && mqreturn.MQRC == mq.MQRC_TRUNCATED_MSG_FAILED && len(buf) < maxBufSize {
				// Double the size, apart from capping it at 100MB
				buf = append(buf, make([]byte, len(buf))...)
				if len(buf) > maxBufSize {
					buf = buf[0:maxBufSize]
				}
				logDebug("getStatisticsMessage: extending message buffer to %d", len(buf))

			} else {
				if mqreturn.MQRC != mq.MQRC_NO_MSG_AVAILABLE {
					traceExitF("getStatisticsMessage", 1, "BufSize %d Error %v", msgLen, err)
					return 0, err
				} else {
					// Quit cleanly
					trunc = false
				}
			}
		} else {
			trunc = false
		}
	}

	if err == nil {
		ci.statisticsCount++
	}
	traceExitErr("getStatisticsMessage", 0, err)
	return msgLen, err
}

// This is where we actually parse the groups inside the statistics message and put the invidual metrics into the various structures
func parseStatisticsMsg(buf []byte, datalen int) {
	// var err error
	var elem *mq.PCFParameter

	parmAvail := true
	bytesRead := 0
	offset := 0

	traceEntry("parseStatisticsMsg")

	stq := GetObjectStatistics(GetConnectionKey(), OT_Q)
	stqmgr := GetObjectStatistics(GetConnectionKey(), OT_Q_MGR)
	qMgrElemMap := make(map[int32]*mq.PCFParameter)

	cfh, offset := mq.ReadPCFHeader(buf)
	// logDebug("Statistics Msg: %d %+v", offset, cfh)

	if cfh == nil || cfh.ParameterCount == 0 {
		return
	}

	// Check the data
	if cfh.Type > mq.MQCFT_STATUS {
		logTrace("Ignoring event with type %d (%s)", cfh.Type, mq.MQItoString("CFT", int(cfh.Type)))
		return
	}

	// Verify that it's the right version
	if cfh.Version < mq.MQCFH_VERSION_1 || cfh.Version > mq.MQCFH_CURRENT_VERSION {
		logTrace("Ignoring event with version %d ", cfh.Version)
		return
	}

	switch cfh.Command {
	case mq.MQCMD_STATISTICS_MQI:
		for k := range stqmgr.Attributes {
			stqmgr.Attributes[k].Values = make(map[string]*StatusValue)
		}
	case mq.MQCMD_STATISTICS_Q:
		for k := range stq.Attributes {
			stq.Attributes[k].Values = make(map[string]*StatusValue)
		}
	default:
		logTrace("Ignoring event with command %d (%s)", cfh.Command, mq.MQItoString("CMD", int(cfh.Command)))
		return
	}

	for parmAvail && cfh.CompCode != mq.MQCC_FAILED {
		elem, bytesRead = mq.ReadPCFParameter(buf[offset:])
		offset += bytesRead
		// Have we now reached the end of the message
		if offset >= datalen {
			parmAvail = false
		}

		// The message should now consist of EITHER
		// - a list of groups, one group per queue OR
		// - a single set of metrics for a qmgr
		if elem.Type == mq.MQCFT_GROUP {

			cnt := elem.ParameterCount

			// logDebug("G: %+v", elem)

			// There are no cases of nested groups in MQ events, so we can just walk
			// through the elements without worrying about recursion

			elemMap := make(map[int32]*mq.PCFParameter)
			for i := 0; i < int(cnt); i++ {
				elem := elem.GroupList[i]
				key := elem.Parameter
				elemMap[key] = elem
				// logDebug("E: %+v", elem)
			}

			e, ok := elemMap[mq.MQCA_Q_NAME]
			qName := e.String[0]
			stq.Attributes[ATTR_Q_NAME].Values[qName] = newStatusValueString(qName)
			/*
				e, ok = elemMap[mq.MQCA_Q_MGR_NAME]
				qMgrName := e.String[0]
				stq.Attributes[ATTR_QMGR_NAME].Values[qMgrName] = newStatusValueString(qMgrName)
			*/
			if ok {
				for k, v := range qStatisticsAttrsMap {
					vals, found := elemMap[k]
					if found {
						// Check that we have enough metric names to cover all the returned values in the array
						if len(v) > len(vals.Int64Value)+1 {
							logWarn("Mismatch between field %d in statistics message and expected number of elements", k)
						}

						for i := 0; i < len(v); i++ {
							attr := v[i]

							val := int64(0)
							if i < len(vals.Int64Value) {
								val = vals.Int64Value[i]
							} else {
								val = sum(vals.Int64Value)
							}

							curVal, attrFound := stq.Attributes[attr].Values[qName]
							if attrFound && stq.Attributes[attr].Delta {
								val += curVal.ValueInt64
							}
							stq.Attributes[attr].Values[qName] = newStatusValueInt64(val)
							// logDebug("Attrs: %+v", stq.Attributes[attr].Values)
						}
					}
				}

			}

		} else {
			key := elem.Parameter
			qMgrElemMap[key] = elem
			// logDebug("E: %+v", elem)
		}
	}

	if cfh.Command == mq.MQCMD_STATISTICS_MQI {

		// logDebug("Collected qmgr metrics: %+v", qMgrElemMap)
		e, ok := qMgrElemMap[mq.MQCA_Q_MGR_NAME]
		qmName := e.String[0]

		// stqmgr.Attributes[ATTR_QMGR_NAME].Values[qmName] = newStatusValueString(qmName)

		// logDebug("Attrs: %+v", stqmgr.Attributes)
		if ok {
			for k, v := range qmgrStatisticsAttrsMap {

				vals, found := qMgrElemMap[k]
				if found {
					// Check that we have enough metric names to cover all the returned values in the array
					// If there is one more metric name than actual metrics then we will be creating a total
					// from the list
					if len(v) > len(vals.Int64Value)+1 {
						logWarn("Mismatch between field %d in statistics message and expected number of elements", k)
					}
					for i := 0; i < len(v); i++ {
						attr := v[i]

						val := int64(0)
						if i < len(vals.Int64Value) {
							val = vals.Int64Value[i]
						} else {
							val = sum(vals.Int64Value)
						}
						curVal, attrFound := stqmgr.Attributes[attr].Values[qmName]
						if attrFound && stqmgr.Attributes[attr].Delta {
							val += curVal.ValueInt64
						}
						stqmgr.Attributes[attr].Values[qmName] = newStatusValueInt64(val)
					}
				}
			}
		}

		// logDebug("E: %+v", elem)
	}

	traceExit("parseStatisticsMsg", 0)

}

// Return a total across the Int64Value array for a set of similar metrics
func sum(a []int64) int64 {
	s := int64(0)
	for i := 0; i < len(a); i++ {
		s += a[i]
	}
	return s
}
