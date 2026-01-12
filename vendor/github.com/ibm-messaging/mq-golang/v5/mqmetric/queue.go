package mqmetric

/*
  Copyright (c) IBM Corporation 2018,2025

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
Functions in this file use the DISPLAY QSTATUS, DISPLAY QUEUE and RESET QSTATS commands
to extract metrics about MQ queues
*/

import (
	_ "fmt"

	"strings"
	"time"

	"github.com/ibm-messaging/mq-golang/v5/ibmmq"
)

const (
	ATTR_Q_NAME        = "name"
	ATTR_Q_MSGAGE      = "oldest_message_age"
	ATTR_Q_IPPROCS     = "input_handles"
	ATTR_Q_OPPROCS     = "output_handles"
	ATTR_Q_QTIME_SHORT = "qtime_short"
	ATTR_Q_QTIME_LONG  = "qtime_long"
	ATTR_Q_CURFSIZE    = "qfile_current_size"
	ATTR_Q_SINCE_PUT   = "time_since_put"
	ATTR_Q_SINCE_GET   = "time_since_get"
	ATTR_Q_MAX_DEPTH   = "attribute_max_depth"
	ATTR_Q_USAGE       = "attribute_usage"
	ATTR_Q_CURMAXFSIZE = "qfile_max_size"
	// Uncommitted messages - on Distributed platforms, this is any integer;
	// but on z/OS it only indicates 0/1 (MQQSUM_NO/YES)
	ATTR_Q_UNCOM = "uncommitted_messages"

	// The next attributes are given the same name
	// as the published statistics from the amqsrua-style
	// values. That allows a dashboard for Distributed and z/OS
	// to merge the same query.
	ATTR_Q_DEPTH        = "depth"
	ATTR_Q_INTERVAL_PUT = "mqput_mqput1_count"
	ATTR_Q_INTERVAL_GET = "mqget_count"

	// This is the Highest Depth returned over an interval via the
	// RESET QSTATS command. Contrast with the attribute_max_depth
	// value which is the DISPLAY QL(x) MAXDEPTH attribute.
	ATTR_Q_INTERVAL_HI_DEPTH = "hi_depth"
)

/*
Unlike the statistics produced via a topic, there is no discovery
of the attributes available in object STATUS queries. There is also
no discovery of descriptions for them. So this function hardcodes the
attributes we are going to look for and gives the associated descriptive
text.
*/
func QueueInitAttributes() {
	traceEntry("QueueInitAttributes")
	ci := getConnection(GetConnectionKey())
	os := &ci.objectStatus[OT_Q]
	st := GetObjectStatus(GetConnectionKey(), OT_Q)

	if os.init {
		traceExit("QueueInitAttributes", 1)
		return
	}

	st.Attributes = make(map[string]*StatusAttribute)

	attr := ATTR_Q_NAME
	st.Attributes[attr] = newPseudoStatusAttribute(attr, "Queue Name")

	attr = ATTR_Q_SINCE_PUT
	st.Attributes[attr] = newStatusAttribute(attr, "Time Since Put", DUMMY_PCFATTR)
	attr = ATTR_Q_SINCE_GET
	st.Attributes[attr] = newStatusAttribute(attr, "Time Since Get", DUMMY_PCFATTR)

	// These are the integer status fields that are of interest
	attr = ATTR_Q_MSGAGE
	st.Attributes[attr] = newStatusAttribute(attr, "Oldest Message", ibmmq.MQIACF_OLDEST_MSG_AGE)

	// Don't want to add these if only the UsePublication option is active, as the descriptions
	// of the metrics can conflict in Prometheus. These are the same attributes from the
	// excludeMetric map in discover.go
	if ci.useStatus {
		attr = ATTR_Q_IPPROCS
		st.Attributes[attr] = newStatusAttribute(attr, "Input Handles", ibmmq.MQIA_OPEN_INPUT_COUNT)
		attr = ATTR_Q_OPPROCS
		st.Attributes[attr] = newStatusAttribute(attr, "Output Handles", ibmmq.MQIA_OPEN_OUTPUT_COUNT)
	}
	attr = ATTR_Q_UNCOM
	if ci.si.platform == ibmmq.MQPL_ZOS {
		st.Attributes[attr] = newStatusAttribute(attr, "Uncommitted Messages (Yes/No)", ibmmq.MQIACF_UNCOMMITTED_MSGS)
	} else {
		st.Attributes[attr] = newStatusAttribute(attr, "Uncommitted Messages (Count)", ibmmq.MQIACF_UNCOMMITTED_MSGS)
	}

	// QFile sizes - current, and the "current maximum" which may not be
	// the same as the qdefinition but is the one in effect for now until
	// the qfile empties
	attr = ATTR_Q_CURFSIZE
	st.Attributes[attr] = newStatusAttribute(attr, "Queue File Current Size", ibmmq.MQIACF_CUR_Q_FILE_SIZE)
	attr = ATTR_Q_CURMAXFSIZE
	st.Attributes[attr] = newStatusAttribute(attr, "Queue File Maximum Size", ibmmq.MQIACF_CUR_MAX_FILE_SIZE)

	// Usually we get the QDepth from published resources, But on z/OS we can get it from the QSTATUS response. We
	// also have an option where we are ignoring most of the queue publications even if we use subscriptions for other
	// object (qmgr/NHA) resources
	if !ci.usePublications || ci.useDepthFromStatus || ci.useStatistics {
		attr = ATTR_Q_DEPTH
		// The description should match the published metric, including case
		st.Attributes[attr] = newStatusAttribute(attr, "Queue depth", ibmmq.MQIA_CURRENT_Q_DEPTH)
	}

	if ci.si.platform == ibmmq.MQPL_ZOS && ci.useResetQStats {
		attr = ATTR_Q_INTERVAL_PUT
		st.Attributes[attr] = newStatusAttribute(attr, "Put/Put1 Count", ibmmq.MQIA_MSG_ENQ_COUNT)
		attr = ATTR_Q_INTERVAL_GET
		st.Attributes[attr] = newStatusAttribute(attr, "Get Count", ibmmq.MQIA_MSG_DEQ_COUNT)
		attr = ATTR_Q_INTERVAL_HI_DEPTH
		st.Attributes[attr] = newStatusAttribute(attr, "Highest Depth", ibmmq.MQIA_HIGH_Q_DEPTH)
	}

	// This is not really a monitoring metric but it enables calculations to be made such as %full for
	// the queue. It's extracted at startup of the program via INQUIRE_Q and not updated later even if the
	// queue definition is changed until rediscovery of the queues on a schedule.
	// It's not easy to generate the % value in this program as the CurDepth will
	// usually - but not always - come from the published resource stats. So we don't have direct access to it.
	// Recording the MaxDepth allows Prometheus etc to do the calculation regardless of how the CurDepth was obtained.
	attr = ATTR_Q_MAX_DEPTH
	st.Attributes[attr] = newStatusAttribute(attr, "Queue Max Depth", DUMMY_PCFATTR)
	attr = ATTR_Q_USAGE
	st.Attributes[attr] = newStatusAttribute(attr, "Queue Usage", DUMMY_PCFATTR)

	attr = ATTR_Q_QTIME_SHORT
	st.Attributes[attr] = newStatusAttribute(attr, "Queue Time Short", ibmmq.MQIACF_Q_TIME_INDICATOR)
	st.Attributes[attr].index = 0
	attr = ATTR_Q_QTIME_LONG
	st.Attributes[attr] = newStatusAttribute(attr, "Queue Time Long", ibmmq.MQIACF_Q_TIME_INDICATOR)
	st.Attributes[attr].index = 1

	os.init = true

	traceExit("QueueInitAttributes", 0)

}

// If we need to list the queues that match a pattern. Not needed for
// the status queries as they (unlike the pub/sub resource stats) accept
// patterns in the PCF command
func InquireQueues(patterns string) ([]string, error) {
	traceEntry("InquireQueues")
	QueueInitAttributes()
	rc, err := inquireObjects(patterns, ibmmq.MQOT_Q)
	traceExitErr("InquireQueues", 0, err)
	return rc, err
}

func CollectQueueStatus(patterns string) error {
	var err error
	traceEntry("CollectQueueStatus")

	ci := getConnection(GetConnectionKey())
	st := GetObjectStatus(GetConnectionKey(), OT_Q)
	QueueInitAttributes()

	// Empty any collected values
	for k := range st.Attributes {
		st.Attributes[k].Values = make(map[string]*StatusValue)
	}

	queuePatterns := strings.Split(patterns, ",")
	if len(queuePatterns) == 0 {
		traceExit("CollectQueueStatus", 1)
		return nil
	}

	// If there was a negative pattern, then we have to look through the
	// list of queues and query status individually. Otherwise we can
	// use regular MQ patterns to query queues in a batch.
	if strings.Contains(patterns, "!") {
		for qName, qi := range qInfoMap {
			if len(qName) == 0 || !qi.exists {
				continue
			}
			err = collectQueueStatus(qName, ibmmq.MQOT_Q)
			if err == nil && ci.useResetQStats {
				err = collectResetQStats(qName)
			}
		}
	} else {
		for _, pattern := range queuePatterns {
			pattern = strings.TrimSpace(pattern)
			if len(pattern) == 0 {
				continue
			}

			err = collectQueueStatus(pattern, ibmmq.MQOT_Q)
			if err == nil && ci.useResetQStats {
				err = collectResetQStats(pattern)
			}
		}
	}
	traceExitErr("CollectQueueStatus", 0, err)
	return err
}

// Issue the INQUIRE_QUEUE_STATUS command for a queue or wildcarded queue name
// Collect the responses and build up the statistics
func collectQueueStatus(pattern string, instanceType int32) error {
	var err error
	traceEntryF("collectQueueStatus", "Pattern: %s", pattern)

	ci := getConnection(GetConnectionKey())

	statusClearReplyQ()

	putmqmd, pmo, cfh, buf := statusSetCommandHeaders()

	// Can allow all the other fields to default
	cfh.Command = ibmmq.MQCMD_INQUIRE_Q_STATUS

	// Add the parameters one at a time into a buffer
	pcfparm := new(ibmmq.PCFParameter)
	pcfparm.Type = ibmmq.MQCFT_STRING
	pcfparm.Parameter = ibmmq.MQCA_Q_NAME
	pcfparm.String = []string{pattern}
	cfh.ParameterCount++
	buf = append(buf, pcfparm.Bytes()...)

	pcfparm = new(ibmmq.PCFParameter)
	pcfparm.Type = ibmmq.MQCFT_INTEGER
	pcfparm.Parameter = ibmmq.MQIACF_Q_STATUS_TYPE
	pcfparm.Int64Value = []int64{int64(ibmmq.MQIACF_Q_STATUS)}
	cfh.ParameterCount++
	buf = append(buf, pcfparm.Bytes()...)

	// Once we know the total number of parameters, put the
	// CFH header on the front of the buffer.
	buf = append(cfh.Bytes(), buf...)

	// And now put the command to the queue
	err = ci.si.cmdQObj.Put(putmqmd, pmo, buf)
	if err != nil {
		traceExit("collectQueueStatus", 1)
		return err
	}

	// Now get the responses - loop until all have been received (one
	// per queue) or we run out of time
	statusMsgCount := 0
	for allReceived := false; !allReceived; {
		cfh, buf, allReceived, err = statusGetReply(putmqmd.MsgId)
		if buf != nil {
			statusMsgCount++
			parseQData(instanceType, cfh, buf)
		}
	}
	//logDebug("collectQueueStatus response count: %d", statusMsgCount)
	traceExitErr("collectQueueStatus", 0, err)
	return err
}

func collectResetQStats(pattern string) error {
	var err error

	traceEntry("collectResetQStats")

	ci := getConnection(GetConnectionKey())

	statusClearReplyQ()
	putmqmd, pmo, cfh, buf := statusSetCommandHeaders()

	// Can allow all the other fields to default
	cfh.Command = ibmmq.MQCMD_RESET_Q_STATS

	// Add the parameters one at a time into a buffer
	pcfparm := new(ibmmq.PCFParameter)
	pcfparm.Type = ibmmq.MQCFT_STRING
	pcfparm.Parameter = ibmmq.MQCA_Q_NAME
	pcfparm.String = []string{pattern}
	cfh.ParameterCount++
	buf = append(buf, pcfparm.Bytes()...)

	buf = append(cfh.Bytes(), buf...)

	// And now put the command to the queue
	err = ci.si.cmdQObj.Put(putmqmd, pmo, buf)
	if err != nil {
		traceExitErr("collectResetQueueStats", 1, err)
		return err
	}

	// Now get the responses - loop until all have been received (one
	// per queue) or we run out of time
	for allReceived := false; !allReceived; {
		cfh, buf, allReceived, err = statusGetReply(putmqmd.MsgId)
		if buf != nil {
			parseResetQStatsData(cfh, buf)
		}
	}
	traceExitErr("collectResetQueueStats", 0, err)
	return err
}

// Issue the INQUIRE_Q call for wildcarded queue names and
// extract the required attributes
func inquireQueueAttributes(objectPatternsList string) error {
	var err error

	traceEntry("inquireQueueAttributes")

	ci := getConnection(GetConnectionKey())
	statusClearReplyQ()

	if objectPatternsList == "" {
		traceExitErr("inquireQueueAttributes", 1, err)
		return err
	}

	objectPatterns := strings.Split(strings.TrimSpace(objectPatternsList), ",")
	for i := 0; i < len(objectPatterns) && err == nil; i++ {
		var buf []byte
		pattern := strings.TrimSpace(objectPatterns[i])
		if len(pattern) == 0 {
			continue
		}

		putmqmd, pmo, cfh, buf := statusSetCommandHeaders()

		// Can allow all the other fields to default
		cfh.Command = ibmmq.MQCMD_INQUIRE_Q
		cfh.ParameterCount = 0

		// Add the parameters one at a time into a buffer
		pcfparm := new(ibmmq.PCFParameter)
		pcfparm.Type = ibmmq.MQCFT_STRING
		pcfparm.Parameter = ibmmq.MQCA_Q_NAME
		pcfparm.String = []string{pattern}
		cfh.ParameterCount++
		buf = append(buf, pcfparm.Bytes()...)

		pcfparm = new(ibmmq.PCFParameter)
		pcfparm.Type = ibmmq.MQCFT_INTEGER_LIST
		pcfparm.Parameter = ibmmq.MQIACF_Q_ATTRS
		pcfparm.Int64Value = []int64{int64(ibmmq.MQIA_MAX_Q_DEPTH), int64(ibmmq.MQIA_USAGE), int64(ibmmq.MQIA_DEFINITION_TYPE), int64(ibmmq.MQCA_Q_DESC), int64(ibmmq.MQCA_CLUSTER_NAME)}
		if ci.showCustomAttribute {
			pcfparm.Int64Value = append(pcfparm.Int64Value, int64(ibmmq.MQCA_CUSTOM))
		}
		cfh.ParameterCount++
		buf = append(buf, pcfparm.Bytes()...)

		// Once we know the total number of parameters, put the
		// CFH header on the front of the buffer.
		buf = append(cfh.Bytes(), buf...)

		// And now put the command to the queue
		err = ci.si.cmdQObj.Put(putmqmd, pmo, buf)
		if err != nil {
			traceExitErr("inquireQueueAttributes", 2, err)
			return err
		}

		for allReceived := false; !allReceived; {
			cfh, buf, allReceived, err = statusGetReply(putmqmd.MsgId)
			if buf != nil {
				parseQAttrData(cfh, buf)
			}
		}
	}
	traceExit("inquireQueueAttributes", 0)
	return nil
}

// Given a PCF response message, parse it to extract the desired statistics
func parseQData(instanceType int32, cfh *ibmmq.MQCFH, buf []byte) string {
	var elem *ibmmq.PCFParameter

	traceEntry("parseQData")

	st := GetObjectStatus(GetConnectionKey(), OT_Q)

	qName := ""
	key := ""

	lastPutTime := ""
	lastGetTime := ""
	lastPutDate := ""
	lastGetDate := ""

	parmAvail := true
	bytesRead := 0
	offset := 0
	datalen := len(buf)
	if cfh == nil || cfh.ParameterCount == 0 {
		traceExit("parseQData", 1)
		return ""
	}

	// Parse it once to extract the fields that are needed for the map key
	for parmAvail && cfh.CompCode != ibmmq.MQCC_FAILED {
		elem, bytesRead = ibmmq.ReadPCFParameter(buf[offset:])
		offset += bytesRead
		// Have we now reached the end of the message
		if offset >= datalen {
			parmAvail = false
		}

		// Only one field needed for queues
		switch elem.Parameter {
		case ibmmq.MQCA_Q_NAME:
			qName = strings.TrimSpace(elem.String[0])
		}
	}

	// Create a unique key for this instance
	key = qName
	st.Attributes[ATTR_Q_NAME].Values[key] = newStatusValueString(qName)

	// And then re-parse the message so we can store the metrics now knowing the map key
	parmAvail = true
	offset = 0
	for parmAvail && cfh.CompCode != ibmmq.MQCC_FAILED {
		elem, bytesRead = ibmmq.ReadPCFParameter(buf[offset:])
		offset += bytesRead
		// Have we now reached the end of the message
		if offset >= datalen {
			parmAvail = false
		}

		if !statusGetIntAttributes(GetObjectStatus(GetConnectionKey(), OT_Q), elem, key) {
			switch elem.Parameter {
			case ibmmq.MQCACF_LAST_PUT_TIME:
				lastPutTime = strings.TrimSpace(elem.String[0])
			case ibmmq.MQCACF_LAST_PUT_DATE:
				lastPutDate = strings.TrimSpace(elem.String[0])
			case ibmmq.MQCACF_LAST_GET_TIME:
				lastGetTime = strings.TrimSpace(elem.String[0])
			case ibmmq.MQCACF_LAST_GET_DATE:
				lastGetDate = strings.TrimSpace(elem.String[0])
			}
		}
	}

	now := time.Now()
	if lastPutTime != "" {
		st.Attributes[ATTR_Q_SINCE_PUT].Values[key] = newStatusValueInt64(statusTimeDiff(now, lastPutDate, lastPutTime))
	}
	if lastGetTime != "" {
		st.Attributes[ATTR_Q_SINCE_GET].Values[key] = newStatusValueInt64(statusTimeDiff(now, lastGetDate, lastGetTime))
	}
	if s, ok := qInfoMap[key]; ok {
		maxDepth := s.AttrMaxDepth
		st.Attributes[ATTR_Q_MAX_DEPTH].Values[key] = newStatusValueInt64(maxDepth)
		usage := s.AttrUsage
		st.Attributes[ATTR_Q_USAGE].Values[key] = newStatusValueInt64(usage)
	}
	traceExitF("parseQData", 0, "Key: %s", key)
	return key
}

// Given a PCF response message, parse it to extract the desired statistics
func parseResetQStatsData(cfh *ibmmq.MQCFH, buf []byte) string {
	var elem *ibmmq.PCFParameter

	traceEntry("parseResetQStatsData")

	st := GetObjectStatus(GetConnectionKey(), OT_Q)

	qName := ""
	key := ""

	parmAvail := true
	bytesRead := 0
	offset := 0
	datalen := len(buf)
	if cfh == nil || cfh.ParameterCount == 0 {
		traceExit("parseResetQStatsData", 1)
		return ""
	}

	// Parse it once to extract the fields that are needed for the map key
	for parmAvail && cfh.CompCode != ibmmq.MQCC_FAILED {
		elem, bytesRead = ibmmq.ReadPCFParameter(buf[offset:])
		offset += bytesRead
		// Have we now reached the end of the message
		if offset >= datalen {
			parmAvail = false
		}

		// Only one field needed for queues
		switch elem.Parameter {
		case ibmmq.MQCA_Q_NAME:
			qName = strings.TrimSpace(elem.String[0])
		}
	}

	// Create a unique key for this instance
	key = qName

	st.Attributes[ATTR_Q_NAME].Values[key] = newStatusValueString(qName)

	// And then re-parse the message so we can store the metrics now knowing the map key
	parmAvail = true
	offset = 0
	for parmAvail && cfh.CompCode != ibmmq.MQCC_FAILED {
		elem, bytesRead = ibmmq.ReadPCFParameter(buf[offset:])
		offset += bytesRead
		// Have we now reached the end of the message
		if offset >= datalen {
			parmAvail = false
		}

		statusGetIntAttributes(GetObjectStatus(GetConnectionKey(), OT_Q), elem, key)
	}

	traceExitF("parseResetQStatsData", 0, "Key: %s", key)
	return key
}

func parseQAttrData(cfh *ibmmq.MQCFH, buf []byte) {
	var elem *ibmmq.PCFParameter
	traceEntry("parseQAttrData")
	qName := ""

	parmAvail := true
	bytesRead := 0
	offset := 0
	datalen := len(buf)
	if cfh.ParameterCount == 0 {
		traceExit("parseQAttrData", 1)
		return
	}
	// Parse it once to extract the fields that are needed for the map key
	for parmAvail && cfh.CompCode != ibmmq.MQCC_FAILED {
		elem, bytesRead = ibmmq.ReadPCFParameter(buf[offset:])
		offset += bytesRead
		// Have we now reached the end of the message
		if offset >= datalen {
			parmAvail = false
		}

		// Only one field needed for queues
		switch elem.Parameter {
		case ibmmq.MQCA_Q_NAME:
			qName = strings.TrimSpace(elem.String[0])
		}
	}

	// And then re-parse the message so we can store the metrics now knowing the map key
	parmAvail = true
	offset = 0
	for parmAvail && cfh.CompCode != ibmmq.MQCC_FAILED {
		elem, bytesRead = ibmmq.ReadPCFParameter(buf[offset:])
		offset += bytesRead
		// Have we now reached the end of the message
		if offset >= datalen {
			parmAvail = false
		}

		switch elem.Parameter {
		case ibmmq.MQIA_MAX_Q_DEPTH:
			v := elem.Int64Value[0]
			if v > 0 {
				if qInfo, ok := qInfoMap[qName]; ok {
					qInfo.AttrMaxDepth = v
				}
			}
			//fmt.Printf("MaxQDepth for %s = %d \n",qName,v)
		case ibmmq.MQIA_USAGE:
			v := elem.Int64Value[0]
			if v > 0 {
				if qInfo, ok := qInfoMap[qName]; ok {
					qInfo.AttrUsage = v
				}
			}
		case ibmmq.MQIA_DEFINITION_TYPE:
			v := elem.Int64Value[0]
			if v > 0 {
				if qInfo, ok := qInfoMap[qName]; ok {
					qInfo.DefType = v
				}
			}
		case ibmmq.MQCA_Q_DESC:
			v := elem.String[0]
			if v != "" {
				if qInfo, ok := qInfoMap[qName]; ok {
					qInfo.Description = printableStringUTF8(v)
				}
			}

		case ibmmq.MQCA_CUSTOM:
			v := elem.String[0]
			if v != "" {
				if qInfo, ok := qInfoMap[qName]; ok {
					qInfo.Custom = printableStringUTF8(v)
				}
			}

		case ibmmq.MQCA_CLUSTER_NAME:
			v := elem.String[0]
			if v != "" {
				if qInfo, ok := qInfoMap[qName]; ok {
					qInfo.Cluster = printableStringUTF8(v)
				}
			}
		}

	}

	traceExit("parseQAttrData", 0)
	return
}

// Return a standardised value.
func QueueNormalise(attr *StatusAttribute, v int64) float64 {
	return statusNormalise(attr, v)
}

// Return the nominated MQCA*/MQIA* attribute from the object's attributes
// stored in the map
func GetQueueAttribute(key string, attribute int32) string {
	var o *ObjInfo
	v := DUMMY_STRING
	ok := false

	o, ok = qInfoMap[key]

	if !ok {
		// return something so Prometheus doesn't turn it into "0.0"
		return DUMMY_STRING
	}

	switch attribute {
	case ibmmq.MQCA_CLUSTER_NAME:
		v = o.Cluster
	case ibmmq.MQIA_DEFINITION_TYPE:
		defType := int32(o.DefType)
		switch defType {
		case ibmmq.MQQDT_PREDEFINED:
			v = "Predefined"
		case ibmmq.MQQDT_PERMANENT_DYNAMIC:
			v = "PermDyn"
		case ibmmq.MQQDT_TEMPORARY_DYNAMIC:
			v = "TempDyn"
		case ibmmq.MQQDT_SHARED_DYNAMIC:
			v = "SharedDyn"
		}
	default:
		v = DUMMY_STRING
	}
	v = strings.TrimSpace(v)

	if v == "" {
		v = DUMMY_STRING
	}
	return v
}
