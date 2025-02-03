/*
Package mqmetric contains a set of routines common to several
commands used to export MQ metrics to different backend
storage mechanisms including Prometheus and InfluxDB.
*/
package mqmetric

/*
  Copyright (c) IBM Corporation 2016, 2022

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
Functions in this file use the DISPLAY CHSTATUS command to extract metrics
about running MQ channels
*/

import (
	_ "fmt"
	"regexp"
	"strings"
	"time"

	"github.com/ibm-messaging/mq-golang/v5/ibmmq"
)

const (
	ATTR_CHL_NAME     = "name"
	ATTR_CHL_CONNNAME = "connname"
	ATTR_CHL_JOBNAME  = "jobname"
	ATTR_CHL_RQMNAME  = "rqmname"

	ATTR_CHL_MESSAGES      = "messages"
	ATTR_CHL_BYTES_SENT    = "bytes_sent"
	ATTR_CHL_BYTES_RCVD    = "bytes_rcvd"
	ATTR_CHL_BUFFERS_SENT  = "buffers_sent"
	ATTR_CHL_BUFFERS_RCVD  = "buffers_rcvd"
	ATTR_CHL_BATCHES       = "batches"
	ATTR_CHL_STATUS        = "status"
	ATTR_CHL_STATUS_SQUASH = ATTR_CHL_STATUS + "_squash"
	ATTR_CHL_SUBSTATE      = "substate"
	ATTR_CHL_TYPE          = "type"
	ATTR_CHL_INSTANCE_TYPE = "instance_type"
	ATTR_CHL_SINCE_MSG     = "time_since_msg"

	ATTR_CHL_NETTIME_SHORT = "nettime_short"
	ATTR_CHL_NETTIME_LONG  = "nettime_long"
	ATTR_CHL_BATCHSZ_SHORT = "batchsz_short"
	ATTR_CHL_BATCHSZ_LONG  = "batchsz_long"
	ATTR_CHL_XQTIME_SHORT  = "xmitq_time_short"
	ATTR_CHL_XQTIME_LONG   = "xmitq_time_long"

	ATTR_CHL_MAX_INSTC = "attribute_max_instc"
	ATTR_CHL_MAX_INST  = "attribute_max_inst"
	ATTR_CHL_CUR_INST  = "cur_inst"

	SQUASH_CHL_STATUS_STOPPED    = 0
	SQUASH_CHL_STATUS_TRANSITION = 1
	SQUASH_CHL_STATUS_RUNNING    = 2
)

// If connected to a z/OS queue manager, the short/long index is
// as setup here. Otherwise we need to swap 0/1 indices.
func idxDefault(zos bool, val int) int {
	if zos {
		return val
	} else {
		return (1 - val)
	}
}

/*
Unlike the statistics produced via a topic, there is no discovery
of the attributes available in object STATUS queries. There is also
no discovery of descriptions for them. So this function hardcodes the
attributes we are going to look for and gives the associated descriptive
text.
*/
func ChannelInitAttributes() {

	traceEntry("ChannelInitAttributes")

	ci := getConnection(GetConnectionKey())
	os := &ci.objectStatus[OT_CHANNEL]
	st := GetObjectStatus(GetConnectionKey(), OT_CHANNEL)

	zos := false
	if ci.si.platform == ibmmq.MQPL_ZOS {
		zos = true
	}

	if os.init {
		traceExit("ChannelInitAttributes", 1)
		return
	}
	st.Attributes = make(map[string]*StatusAttribute)

	// These fields are used to construct the key to the per-channel map values and
	// as tags to uniquely identify a channel instance
	attr := ATTR_CHL_NAME
	st.Attributes[attr] = newPseudoStatusAttribute(attr, "Channel Name")
	attr = ATTR_CHL_RQMNAME
	st.Attributes[attr] = newPseudoStatusAttribute(attr, "Remote Queue Manager Name")
	attr = ATTR_CHL_JOBNAME
	st.Attributes[attr] = newPseudoStatusAttribute(attr, "MCA Job Name")
	attr = ATTR_CHL_CONNNAME
	st.Attributes[attr] = newPseudoStatusAttribute(attr, "Connection Name")

	// These are the integer status fields that are of interest
	attr = ATTR_CHL_MESSAGES
	st.Attributes[attr] = newStatusAttribute(attr, "Messages (API Calls for SVRCONN)", ibmmq.MQIACH_MSGS)
	st.Attributes[attr].Delta = true // We have to manage the differences as MQ reports cumulative values
	attr = ATTR_CHL_BYTES_SENT
	st.Attributes[attr] = newStatusAttribute(attr, "Bytes sent", ibmmq.MQIACH_BYTES_SENT)
	st.Attributes[attr].Delta = true // We have to manage the differences as MQ reports cumulative values
	attr = ATTR_CHL_BYTES_RCVD
	st.Attributes[attr] = newStatusAttribute(attr, "Bytes rcvd", ibmmq.MQIACH_BYTES_RCVD)
	st.Attributes[attr].Delta = true // We have to manage the differences as MQ reports cumulative values
	attr = ATTR_CHL_BUFFERS_SENT
	st.Attributes[attr] = newStatusAttribute(attr, "Buffers sent", ibmmq.MQIACH_BUFFERS_SENT)
	st.Attributes[attr].Delta = true // We have to manage the differences as MQ reports cumulative values
	attr = ATTR_CHL_BUFFERS_RCVD
	st.Attributes[attr] = newStatusAttribute(attr, "Buffers rcvd", ibmmq.MQIACH_BUFFERS_RCVD)
	st.Attributes[attr].Delta = true // We have to manage the differences as MQ reports cumulative values
	attr = ATTR_CHL_BATCHES
	st.Attributes[attr] = newStatusAttribute(attr, "Completed Batches", ibmmq.MQIACH_BATCHES)
	st.Attributes[attr].Delta = true // We have to manage the differences as MQ reports cumulative values

	// This is decoded by MQCHS_* values
	attr = ATTR_CHL_STATUS
	st.Attributes[attr] = newStatusAttribute(attr, "Channel Status", ibmmq.MQIACH_CHANNEL_STATUS)
	// The next value can be decoded from the MQCHSSTATE_* values
	attr = ATTR_CHL_SUBSTATE
	st.Attributes[attr] = newStatusAttribute(attr, "Channel Substate", ibmmq.MQIACH_CHANNEL_SUBSTATE)
	attr = ATTR_CHL_TYPE
	st.Attributes[attr] = newStatusAttribute(attr, "Channel Type", ibmmq.MQIACH_CHANNEL_TYPE)
	attr = ATTR_CHL_INSTANCE_TYPE
	st.Attributes[attr] = newStatusAttribute(attr, "Channel Instance Type", ibmmq.MQIACH_CHANNEL_INSTANCE_TYPE)

	// This is the same attribute as earlier, except that we indicate the values are to be modified in
	// a special way.
	attr = ATTR_CHL_STATUS_SQUASH
	st.Attributes[attr] = newStatusAttribute(attr, "Channel Status - Simplified", ibmmq.MQIACH_CHANNEL_STATUS)
	st.Attributes[attr].squash = true
	os.init = true

	// Some of the short/long status values are the opposite way round on the different platforms!
	// Really a bug in the PCF code (internal reference 304982), but it's not likely to be fixed because of compatibility.
	// In most cases, and always on ZOS, the SHORT is first followed by LONG. But the following
	// attributes are reversed:
	//   COMPRESSION_RATE, COMPRESSION_TIME, EXIT_TIME: Not reported here anyway
	//   NETWORK_TIME, XMITQ_TIME, BATCH_SIZE: Reported here
	attr = ATTR_CHL_NETTIME_SHORT
	st.Attributes[attr] = newStatusAttribute(attr, "Network Time Short", ibmmq.MQIACH_NETWORK_TIME_INDICATOR)
	st.Attributes[attr].index = idxDefault(zos, 0)
	attr = ATTR_CHL_NETTIME_LONG
	st.Attributes[attr] = newStatusAttribute(attr, "Network Time Long", ibmmq.MQIACH_NETWORK_TIME_INDICATOR)
	st.Attributes[attr].index = idxDefault(zos, 1)

	attr = ATTR_CHL_BATCHSZ_SHORT
	st.Attributes[attr] = newStatusAttribute(attr, "Batch Size Average Short", ibmmq.MQIACH_BATCH_SIZE_INDICATOR)
	st.Attributes[attr].index = idxDefault(zos, 0)
	attr = ATTR_CHL_BATCHSZ_LONG
	st.Attributes[attr] = newStatusAttribute(attr, "Batch Size Average Long", ibmmq.MQIACH_BATCH_SIZE_INDICATOR)
	st.Attributes[attr].index = idxDefault(zos, 1)

	attr = ATTR_CHL_XQTIME_SHORT
	st.Attributes[attr] = newStatusAttribute(attr, "XmitQ Time Average Short", ibmmq.MQIACH_XMITQ_TIME_INDICATOR)
	st.Attributes[attr].index = idxDefault(zos, 0)
	attr = ATTR_CHL_XQTIME_LONG
	st.Attributes[attr] = newStatusAttribute(attr, "XmitQ Time Average Long", ibmmq.MQIACH_XMITQ_TIME_INDICATOR)
	st.Attributes[attr].index = idxDefault(zos, 1)

	attr = ATTR_CHL_SINCE_MSG
	st.Attributes[attr] = newStatusAttribute(attr, "Time Since Msg", -1)

	// These are not really monitoring metrics but it may enable calculations to be made such as %used for
	// the channel instance availability. It's extracted at startup of the program via INQUIRE_CHL and not updated later
	// until rediscovery is done based on a separate schedule.
	attr = ATTR_CHL_MAX_INST
	st.Attributes[attr] = newStatusAttribute(attr, "MaxInst", -1)
	attr = ATTR_CHL_MAX_INSTC
	st.Attributes[attr] = newStatusAttribute(attr, "MaxInstC", -1)
	// Current Instances is treated a bit oddly. Although reported on each channel status,
	// it actually refers to the total number of instances of the same name.
	attr = ATTR_CHL_CUR_INST
	st.Attributes[attr] = newStatusAttribute(attr, "Current Instances", -1)

	traceExit("ChannelInitAttributes", 0)
}

// If we need to list the channels that match a pattern. Not needed for
// the status queries as they (unlike the pub/sub resource stats) accept
// patterns in the PCF command
func InquireChannels(patterns string) ([]string, error) {
	traceEntry("InquireChannels")
	ChannelInitAttributes()
	rc, err := inquireObjects(patterns, ibmmq.MQOT_CHANNEL)

	traceExitErr("InquireChannels", 0, err)
	return rc, err
}

func CollectChannelStatus(patterns string) error {
	var err error

	traceEntry("CollectChannelStatus")

	ci := getConnection(GetConnectionKey())
	os := &ci.objectStatus[OT_CHANNEL]
	st := GetObjectStatus(GetConnectionKey(), OT_CHANNEL)

	os.objectSeen = make(map[string]bool) // Record which channels have been seen in this period

	ChannelInitAttributes()

	// Empty any collected values
	for k := range st.Attributes {
		st.Attributes[k].Values = make(map[string]*StatusValue)
	}

	for k := range chlInfoMap {
		chlInfoMap[k].AttrCurInst = 0
	}

	channelPatterns := strings.Split(patterns, ",")
	if len(channelPatterns) == 0 {
		traceExit("CollectChannelStatus", 1)
		return nil
	}

	for _, pattern := range channelPatterns {
		pattern = strings.TrimSpace(pattern)
		if len(pattern) == 0 {
			continue
		}

		// This would allow us to extract SAVED information too
		errCurrent := collectChannelStatus(pattern, ibmmq.MQOT_CURRENT_CHANNEL)
		err = errCurrent
		//errSaved := collectChannelStatus(pattern, ibmmq.MQOT_SAVED_CHANNEL)
		//if errCurrent != nil {
		//	err = errCurrent
		//} else {
		//	err = errSaved
		//}

	}

	// Need to clean out the prevValues elements to stop short-lived channels
	// building up in the map
	for a, _ := range st.Attributes {
		if st.Attributes[a].Delta {
			m := st.Attributes[a].prevValues
			for key, _ := range m {
				if _, ok := os.objectSeen[key]; ok {
					// Leave it in the map
				} else {
					// need to delete it from the map
					delete(m, key)
				}
			}
		}
	}

	// If someone has asked to see all the channel definitions, not just those that have a valid
	// CHSTATUS response, then we can look through the list of all known channels that match
	// our patterns (chlInfoMap) and add some dummy values to the status maps if the channel
	// is not already there. Some of the fields do need to be faked up as we don't know anything about
	// the "partner"
	if err == nil && ci.showInactiveChannels {
		for chlName, v := range chlInfoMap {
			found := false
			chlPrefix := chlName + "/"
			for k, _ := range st.Attributes[ATTR_CHL_STATUS].Values {
				if strings.HasPrefix(k, chlPrefix) {
					found = true
					break
				}
			}
			if !found {
				// A channel key is normally made up of 4 elements but we only know 1
				key := chlName + "/" + DUMMY_STRING + "/" + DUMMY_STRING + "/" + DUMMY_STRING

				st.Attributes[ATTR_CHL_NAME].Values[key] = newStatusValueString(chlName)
				st.Attributes[ATTR_CHL_CONNNAME].Values[key] = newStatusValueString(DUMMY_STRING)
				st.Attributes[ATTR_CHL_JOBNAME].Values[key] = newStatusValueString(DUMMY_STRING)
				st.Attributes[ATTR_CHL_RQMNAME].Values[key] = newStatusValueString(DUMMY_STRING)

				st.Attributes[ATTR_CHL_STATUS].Values[key] = newStatusValueInt64(int64(ibmmq.MQCHS_INACTIVE))
				st.Attributes[ATTR_CHL_STATUS_SQUASH].Values[key] = newStatusValueInt64(SQUASH_CHL_STATUS_STOPPED)
				st.Attributes[ATTR_CHL_TYPE].Values[key] = newStatusValueInt64(v.AttrChlType)
			}
		}
	}

	// Set the metrics corresponding to attributes for all the monitored channels
	// The current instance count is not, strictly speaking, an attribute but it's a way
	// of providing a metric alongside each channel which shows how many there are of that name.
	// All instances of the same channel name, regardless of other aspects (eg remote connName)
	// are given the same instance count so it could be extracted.
	for key, _ := range st.Attributes[ATTR_CHL_NAME].Values {
		chlName := st.Attributes[ATTR_CHL_NAME].Values[key].ValueString
		if s, ok := chlInfoMap[chlName]; ok {
			maxInstC := s.AttrMaxInstC
			st.Attributes[ATTR_CHL_MAX_INSTC].Values[key] = newStatusValueInt64(maxInstC)
			maxInst := s.AttrMaxInst
			st.Attributes[ATTR_CHL_MAX_INST].Values[key] = newStatusValueInt64(maxInst)
			curInst := s.AttrCurInst
			st.Attributes[ATTR_CHL_CUR_INST].Values[key] = newStatusValueInt64(curInst)
		}
	}

	traceExitErr("CollectChannelStatus", 0, err)
	return err

}

// Issue the INQUIRE_CHANNEL_STATUS command for a channel or wildcarded channel name
// Collect the responses and build up the statistics
func collectChannelStatus(pattern string, instanceType int32) error {
	var err error

	traceEntryF("collectChannelStatus", "Pattern: %s", pattern)
	ci := getConnection(GetConnectionKey())
	os := &ci.objectStatus[OT_CHANNEL]

	statusClearReplyQ()

	putmqmd, pmo, cfh, buf := statusSetCommandHeaders()

	// Can allow all the other fields to default
	cfh.Command = ibmmq.MQCMD_INQUIRE_CHANNEL_STATUS

	// Add the parameters one at a time into a buffer
	pcfparm := new(ibmmq.PCFParameter)
	pcfparm.Type = ibmmq.MQCFT_STRING
	pcfparm.Parameter = ibmmq.MQCACH_CHANNEL_NAME
	pcfparm.String = []string{pattern}
	cfh.ParameterCount++
	buf = append(buf, pcfparm.Bytes()...)

	// Add the parameters one at a time into a buffer
	pcfparm = new(ibmmq.PCFParameter)
	pcfparm.Type = ibmmq.MQCFT_INTEGER
	pcfparm.Parameter = ibmmq.MQIACH_CHANNEL_INSTANCE_TYPE
	pcfparm.Int64Value = []int64{int64(instanceType)}
	cfh.ParameterCount++
	buf = append(buf, pcfparm.Bytes()...)

	// Once we know the total number of parameters, put the
	// CFH header on the front of the buffer.
	buf = append(cfh.Bytes(), buf...)

	// And now put the command to the queue
	err = ci.si.cmdQObj.Put(putmqmd, pmo, buf)
	if err != nil {
		traceExitErr("collectChannelStatus", 1, err)
		return err
	}

	// Now get the responses - loop until all have been received (one
	// per channel) or we run out of time
	for allReceived := false; !allReceived; {
		cfh, buf, allReceived, err = statusGetReply(putmqmd.MsgId)
		if buf != nil {
			key := parseChlData(instanceType, cfh, buf)
			if key != "" {
				os.objectSeen[key] = true
			}
		}
	}

	traceExitErr("collectChannelStatus", 0, err)
	return err
}

// Given a PCF response message, parse it to extract the desired statistics
func parseChlData(instanceType int32, cfh *ibmmq.MQCFH, buf []byte) string {
	var elem *ibmmq.PCFParameter

	traceEntry("parseChlData")

	ci := getConnection(GetConnectionKey())
	os := &ci.objectStatus[OT_CHANNEL]
	st := GetObjectStatus(GetConnectionKey(), OT_CHANNEL)
	chlType := ibmmq.MQCHT_ALL
	chlName := ""
	connName := ""
	jobName := ""
	rqmName := ""
	startDate := ""
	startTime := ""
	key := ""

	lastMsgDate := ""
	lastMsgTime := ""

	parmAvail := true
	bytesRead := 0
	offset := 0
	datalen := len(buf)
	if cfh == nil || cfh.ParameterCount == 0 {
		traceExit("parseChlData", 1)
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

		switch elem.Parameter {
		case ibmmq.MQCACH_CHANNEL_NAME:
			chlName = strings.TrimSpace(elem.String[0])
		case ibmmq.MQCACH_CONNECTION_NAME:
			connName = strings.TrimSpace(elem.String[0])
		case ibmmq.MQCACH_MCA_JOB_NAME:
			jobName = strings.TrimSpace(elem.String[0])
		case ibmmq.MQCA_REMOTE_Q_MGR_NAME:
			rqmName = strings.TrimSpace(elem.String[0])
		case ibmmq.MQCACH_CHANNEL_START_TIME:
			startTime = strings.TrimSpace(elem.String[0])
		case ibmmq.MQCACH_CHANNEL_START_DATE:
			startDate = strings.TrimSpace(elem.String[0])
		case ibmmq.MQIACH_CHANNEL_TYPE:
			chlType = int32(elem.Int64Value[0])
		}
	}

	// Prometheus was ignoring a blank string which then got translated into "0.00" in Grafana
	// So if there is no remote qmgr, force a non-empty string value in there. Similarly, the jobname for
	// inactive channels often arrives looking like "00000" but not filling the entire length
	// allowed. So reset that one too.
	if rqmName == "" {
		rqmName = DUMMY_STRING
	}
	if jobName == "" || allZero(jobName) {
		jobName = DUMMY_STRING
	}

	if chlType == ibmmq.MQCHT_SVRCONN {
		if ci.hideSvrConnJobname {
			jobName = DUMMY_STRING
		} else if ci.si.platform == ibmmq.MQPL_ZOS {
			// The jobName does not exist on z/OS so it cannot be used to distinguish
			// unique instances of the same channel name. Instead, we try to fake it with
			// the channel start timestamp. That may still be wrong if lots of channel
			// instances start at the same time, but it's a lot better than combining the
			// instances badly.
			jobName = startDate + ":" + startTime
		}
	}

	// Create a unique key for this channel instance
	key = chlName + "/" + connName + "/" + rqmName + "/" + jobName

	// Look to see if we've already seen a Current channel status that matches
	// the Saved version. If so, then don't bother with the Saved status
	if instanceType == ibmmq.MQOT_SAVED_CHANNEL {
		subKey := chlName + "/" + connName + "/" + rqmName + "/.*"
		for k, _ := range os.objectSeen {
			re := regexp.MustCompile(subKey)
			if re.MatchString(k) {
				traceExit("parseChlData", 2)
				return ""
			}
		}
	}

	st.Attributes[ATTR_CHL_NAME].Values[key] = newStatusValueString(chlName)
	st.Attributes[ATTR_CHL_CONNNAME].Values[key] = newStatusValueString(connName)
	st.Attributes[ATTR_CHL_RQMNAME].Values[key] = newStatusValueString(rqmName)
	st.Attributes[ATTR_CHL_JOBNAME].Values[key] = newStatusValueString(jobName)

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

		if !statusGetIntAttributes(GetObjectStatus(GetConnectionKey(), OT_CHANNEL), elem, key) {
			switch elem.Parameter {
			case ibmmq.MQCACH_LAST_MSG_TIME:
				lastMsgTime = strings.TrimSpace(elem.String[0])
			case ibmmq.MQCACH_LAST_MSG_DATE:
				lastMsgDate = strings.TrimSpace(elem.String[0])
			}
		}
	}

	now := time.Now()
	diff := statusTimeDiff(now, lastMsgDate, lastMsgTime)
	st.Attributes[ATTR_CHL_SINCE_MSG].Values[key] = newStatusValueInt64(diff)

	// Bump the number of active instances of the channel, treating it a bit like a
	// regular config attribute.
	if s, ok := chlInfoMap[chlName]; ok {
		if instanceType != ibmmq.MQOT_SAVED_CHANNEL {
			s.AttrCurInst++
		}
	}

	traceExitF("parseChlData", 0, "Key: %s", key)
	return key
}

// Return a standardised value. If the attribute indicates that something
// special has to be done, then do that. Otherwise just make sure it's a non-negative
// value of the correct datatype
func ChannelNormalise(attr *StatusAttribute, v int64) float64 {
	var f float64

	traceEntry("ChannelNormalise")
	if attr.squash {
		switch attr.pcfAttr {

		case ibmmq.MQIACH_CHANNEL_STATUS:
			v32 := int32(v)
			switch v32 {
			case ibmmq.MQCHS_INACTIVE,
				ibmmq.MQCHS_DISCONNECTED,
				ibmmq.MQCHS_STOPPED,
				ibmmq.MQCHS_PAUSED:
				f = float64(SQUASH_CHL_STATUS_STOPPED)

			case ibmmq.MQCHS_BINDING,
				ibmmq.MQCHS_STARTING,
				ibmmq.MQCHS_STOPPING,
				ibmmq.MQCHS_RETRYING,
				ibmmq.MQCHS_REQUESTING,
				ibmmq.MQCHS_INITIALIZING,
				ibmmq.MQCHS_SWITCHING:
				f = float64(SQUASH_CHL_STATUS_TRANSITION)
			case ibmmq.MQCHS_RUNNING:
				f = float64(SQUASH_CHL_STATUS_RUNNING)
			default:
				f = float64(SQUASH_CHL_STATUS_STOPPED)
			}
		default:
			f = float64(v)
			if f < 0 {
				f = 0
			}
		}
	} else {
		f = float64(v)
		if f < 0 {
			f = 0
		}
	}

	traceExit("ChannelNormalise", 0)

	return f
}

// Issue the INQUIRE_CHANNEL call for wildcarded channel names and
// extract the required attributes
func inquireChannelAttributes(objectPatternsList string, infoMap map[string]*ObjInfo) error {
	var err error

	traceEntry("inquireChannelAttributes")

	ci := getConnection(GetConnectionKey())
	statusClearReplyQ()

	if objectPatternsList == "" {
		traceExitErr("inquireChannelAttributes", 1, err)
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
		cfh.Command = ibmmq.MQCMD_INQUIRE_CHANNEL
		cfh.ParameterCount = 0

		// Add the parameters one at a time into a buffer
		pcfparm := new(ibmmq.PCFParameter)
		pcfparm.Type = ibmmq.MQCFT_STRING
		pcfparm.Parameter = ibmmq.MQCACH_CHANNEL_NAME
		pcfparm.String = []string{pattern}
		cfh.ParameterCount++
		buf = append(buf, pcfparm.Bytes()...)

		pcfparm = new(ibmmq.PCFParameter)
		pcfparm.Type = ibmmq.MQCFT_INTEGER_LIST
		pcfparm.Parameter = ibmmq.MQIACF_CHANNEL_ATTRS
		pcfparm.Int64Value = []int64{int64(ibmmq.MQIACH_MAX_INSTANCES), int64(ibmmq.MQIACH_MAX_INSTS_PER_CLIENT), int64(ibmmq.MQCACH_DESC), int64(ibmmq.MQIACH_CHANNEL_TYPE)}
		cfh.ParameterCount++
		buf = append(buf, pcfparm.Bytes()...)

		// Once we know the total number of parameters, put the
		// CFH header on the front of the buffer.
		buf = append(cfh.Bytes(), buf...)

		// And now put the command to the queue
		err = ci.si.cmdQObj.Put(putmqmd, pmo, buf)
		if err != nil {
			traceExitErr("inquireChannelAttributes", 2, err)
			return err
		}

		for allReceived := false; !allReceived; {
			cfh, buf, allReceived, err = statusGetReply(putmqmd.MsgId)
			if buf != nil {
				parseChannelAttrData(cfh, buf, infoMap)
			}
		}
	}

	traceExit("inquireChannelAttributes", 0)
	return nil
}

func parseChannelAttrData(cfh *ibmmq.MQCFH, buf []byte, infoMap map[string]*ObjInfo) {
	var elem *ibmmq.PCFParameter
	var ci *ObjInfo
	var ok bool

	traceEntry("parseChannelAttrData")

	chlName := ""

	parmAvail := true
	bytesRead := 0
	offset := 0
	datalen := len(buf)
	if cfh.ParameterCount == 0 {
		traceExit("parseChannelAttrData", 1)
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

		// Only one field needed for channels
		switch elem.Parameter {
		case ibmmq.MQCACH_CHANNEL_NAME:
			chlName = strings.TrimSpace(elem.String[0])
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
		case ibmmq.MQIACH_MAX_INSTANCES:
			v := elem.Int64Value[0]
			if v > 0 {
				if ci, ok = infoMap[chlName]; !ok {
					ci = new(ObjInfo)
					infoMap[chlName] = ci
				}
				ci.AttrMaxInst = v
				ci.exists = true

			}
		case ibmmq.MQIACH_MAX_INSTS_PER_CLIENT:
			v := elem.Int64Value[0]
			if v > 0 {
				if ci, ok = infoMap[chlName]; !ok {
					ci = new(ObjInfo)
					infoMap[chlName] = ci
				}
				ci.AttrMaxInstC = v
				ci.exists = true

			}

		case ibmmq.MQIACH_CHANNEL_TYPE:
			v := elem.Int64Value[0]
			if v > 0 {
				if ci, ok = infoMap[chlName]; !ok {
					ci = new(ObjInfo)
					infoMap[chlName] = ci
				}
				ci.AttrChlType = v
				ci.exists = true

			}

		case ibmmq.MQCACH_DESC:
			v := elem.String[0]
			if v != "" {
				if ci, ok = infoMap[chlName]; !ok {
					ci = new(ObjInfo)
					infoMap[chlName] = ci
				}
				ci.Description = printableStringUTF8(v)
				ci.exists = true
			}
		}
	}

	traceExit("parseChannelAttrData", 0)
	return
}

func allZero(s string) bool {
	rc := true
	for i := 0; i < len(s); i++ {
		if s[i] != '0' {
			return false
		}
	}
	return rc
}
