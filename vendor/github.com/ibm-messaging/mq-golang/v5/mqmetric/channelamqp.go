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
Functions in this file use the DISPLAY CHSTATUS CLIENTID(*) command to extract metrics
about running MQ AMQP channels
*/

import (
	_ "fmt"
	"strings"
	"time"

	"github.com/ibm-messaging/mq-golang/v5/ibmmq"
)

const (
	// Most of the ATTR_ fields can be inherited from the channel.go module
	ATTR_CHL_AMQP_CLIENT_ID         = "clientid"
	ATTR_CHL_AMQP_MESSAGES_RECEIVED = "messages_rcvd"
	ATTR_CHL_AMQP_MESSAGES_SENT     = "messages_sent"
	ATTR_CHL_AMQP_CONNECTIONS       = "connection_count"
)

/*
Unlike the statistics produced via a topic, there is no discovery
of the attributes available in object STATUS queries. There is also
no discovery of descriptions for them. So this function hardcodes the
attributes we are going to look for and gives the associated descriptive
text.
*/
func ChannelAMQPInitAttributes() {

	traceEntry("ChannelAMQPInitAttributes")

	ci := getConnection(GetConnectionKey())
	os := &ci.objectStatus[OT_CHANNEL_AMQP]
	st := GetObjectStatus(GetConnectionKey(), OT_CHANNEL_AMQP)

	if os.init {
		traceExit("ChannelAMQPInitAttributes", 1)
		return
	}
	st.Attributes = make(map[string]*StatusAttribute)

	// These fields are used to construct the key to the per-channel map values and
	// as tags to uniquely identify a channel instance
	attr := ATTR_CHL_NAME
	st.Attributes[attr] = newPseudoStatusAttribute(attr, "Channel Name")
	attr = ATTR_CHL_AMQP_CLIENT_ID
	st.Attributes[attr] = newPseudoStatusAttribute(attr, "Client ID")

	// Some other fields
	attr = ATTR_CHL_CONNNAME
	st.Attributes[attr] = newPseudoStatusAttribute(attr, "Connection Name")

	// These are the integer status fields that are of interest
	attr = ATTR_CHL_AMQP_MESSAGES_RECEIVED
	st.Attributes[attr] = newStatusAttribute(attr, "Messages Received", ibmmq.MQIACH_MSGS_RCVD)
	st.Attributes[attr].Delta = true // We have to manage the differences as MQ reports cumulative values
	attr = ATTR_CHL_AMQP_MESSAGES_SENT
	st.Attributes[attr] = newStatusAttribute(attr, "Messages Sent", ibmmq.MQIACH_MSGS_SENT)
	st.Attributes[attr].Delta = true // We have to manage the differences as MQ reports cumulative values

	attr = ATTR_CHL_AMQP_CONNECTIONS
	st.Attributes[attr] = newStatusAttribute(attr, "Connections", ibmmq.MQIACF_CONNECTION_COUNT)

	// This is decoded by MQCHS_* values
	attr = ATTR_CHL_STATUS
	st.Attributes[attr] = newStatusAttribute(attr, "Channel Status", ibmmq.MQIACH_CHANNEL_STATUS)

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

	os.init = true

	traceExit("ChannelAMQPInitAttributes", 0)
}

// If we need to list the channels that match a pattern. Not needed for
// the status queries as they (unlike the pub/sub resource stats) accept
// patterns in the PCF command
func InquireAMQPChannels(patterns string) ([]string, error) {
	traceEntry("InquireAMQPChannels")
	ChannelAMQPInitAttributes()
	rc, err := inquireObjectsWithFilter(patterns, ibmmq.MQOT_CHANNEL, OT_CHANNEL_AMQP)

	traceExitErr("InquireAMQPChannels", 0, err)
	return rc, err
}

func CollectAMQPChannelStatus(patterns string) error {
	var err error

	traceEntry("CollectAMQPChannelStatus")

	ci := getConnection(GetConnectionKey())
	os := &ci.objectStatus[OT_CHANNEL_AMQP]
	st := GetObjectStatus(GetConnectionKey(), OT_CHANNEL_AMQP)

	os.objectSeen = make(map[string]bool) // Record which channels have been seen in this period

	ChannelAMQPInitAttributes()

	// Empty any collected values
	for k := range st.Attributes {
		st.Attributes[k].Values = make(map[string]*StatusValue)
	}

	for k := range amqpInfoMap {
		amqpInfoMap[k].AttrCurInst = 0
	}

	channelPatterns := strings.Split(patterns, ",")
	if len(channelPatterns) == 0 {
		traceExit("CollectAMQPChannelStatus", 1)
		return nil
	}

	for _, pattern := range channelPatterns {
		pattern = strings.TrimSpace(pattern)
		if len(pattern) == 0 {
			continue
		}

		// This would allow us to extract SAVED information too
		errCurrent := collectAMQPChannelStatus(pattern, ibmmq.MQOT_CURRENT_CHANNEL)
		err = errCurrent
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

	// Set the metrics corresponding to attributes for all the monitored channels
	// The current instance count is not, strictly speaking, an attribute but it's a way
	// of providing a metric alongside each channel which shows how many there are of that name.
	// All instances of the same channel name, regardless of other aspects (eg remote connName)
	// are given the same instance count so it could be extracted.
	for key, _ := range st.Attributes[ATTR_CHL_NAME].Values {
		chlName := st.Attributes[ATTR_CHL_NAME].Values[key].ValueString
		if s, ok := amqpInfoMap[chlName]; ok {
			maxInstC := s.AttrMaxInstC
			st.Attributes[ATTR_CHL_MAX_INSTC].Values[key] = newStatusValueInt64(maxInstC)
			maxInst := s.AttrMaxInst
			st.Attributes[ATTR_CHL_MAX_INST].Values[key] = newStatusValueInt64(maxInst)
			curInst := s.AttrCurInst
			st.Attributes[ATTR_CHL_CUR_INST].Values[key] = newStatusValueInt64(curInst)
		}
	}

	traceExitErr("CollectAMQPChannelStatus", 0, err)
	return err

}

// Issue the INQUIRE_CHANNEL_STATUS command for a channel or wildcarded channel name
// Collect the responses and build up the statistics. Add CLIENTID(*) to get the actual
// instances instead of an aggregated response
func collectAMQPChannelStatus(pattern string, instanceType int32) error {
	var err error

	traceEntryF("collectAMQPChannelStatus", "Pattern: %s", pattern)
	ci := getConnection(GetConnectionKey())
	os := &ci.objectStatus[OT_CHANNEL_AMQP]

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
	pcfparm.Parameter = ibmmq.MQIACH_CHANNEL_TYPE
	pcfparm.Int64Value = []int64{int64(ibmmq.MQCHT_AMQP)}
	cfh.ParameterCount++
	buf = append(buf, pcfparm.Bytes()...)

	pcfparm = new(ibmmq.PCFParameter)
	pcfparm.Type = ibmmq.MQCFT_STRING
	pcfparm.Parameter = ibmmq.MQCACH_CLIENT_ID
	pcfparm.String = []string{"*"}
	cfh.ParameterCount++
	buf = append(buf, pcfparm.Bytes()...)

	// Once we know the total number of parameters, put the
	// CFH header on the front of the buffer.
	buf = append(cfh.Bytes(), buf...)

	// And now put the command to the queue
	err = ci.si.cmdQObj.Put(putmqmd, pmo, buf)
	if err != nil {
		traceExitErr("collectAMQPChannelStatus", 1, err)
		return err
	}

	// Now get the responses - loop until all have been received (one
	// per channel) or we run out of time
	for allReceived := false; !allReceived; {
		cfh, buf, allReceived, err = statusGetReply(putmqmd.MsgId)
		if buf != nil {
			key := parseAMQPChlData(instanceType, cfh, buf)
			if key != "" {
				os.objectSeen[key] = true
			}
		}
	}

	traceExitErr("collectAMQPChannelStatus", 0, err)
	return err
}

// Given a PCF response message, parse it to extract the desired statistics
func parseAMQPChlData(instanceType int32, cfh *ibmmq.MQCFH, buf []byte) string {
	var elem *ibmmq.PCFParameter

	traceEntry("parseAMQPChlData")

	ci := getConnection(GetConnectionKey())
	//os := &ci.objectStatus[OT_CHANNEL_AMQP]
	st := GetObjectStatus(GetConnectionKey(), OT_CHANNEL_AMQP)

	chlName := ""
	connName := ""
	clientId := ""
	key := ""

	lastMsgDate := ""
	lastMsgTime := ""

	parmAvail := true
	bytesRead := 0
	offset := 0
	datalen := len(buf)
	if cfh == nil || cfh.ParameterCount == 0 {
		traceExit("parseAMQPChlData", 1)
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
		case ibmmq.MQCACH_CLIENT_ID:
			clientId = strings.TrimSpace(elem.String[0])
		}
	}

	// Create a unique key for this channel instance
	if connName == "" {
		connName = DUMMY_STRING
	}

	if ci.hideAMQPClientId {
		clientId = DUMMY_STRING
	}

	key = chlName + "/" + connName + "/" + clientId

	logDebug("AMQP status    - key: %s", key)
	st.Attributes[ATTR_CHL_NAME].Values[key] = newStatusValueString(chlName)
	st.Attributes[ATTR_CHL_CONNNAME].Values[key] = newStatusValueString(connName)
	st.Attributes[ATTR_CHL_AMQP_CLIENT_ID].Values[key] = newStatusValueString(clientId)

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

		if !statusGetIntAttributes(GetObjectStatus(GetConnectionKey(), OT_CHANNEL_AMQP), elem, key) {
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
	if s, ok := amqpInfoMap[chlName]; ok {
		s.AttrCurInst++
	}

	traceExitF("parseAMQPChlData", 0, "Key: %s", key)
	return key
}

// Issue the INQUIRE_CHANNEL call for wildcarded channel names and
// extract the required attributes
func inquireAMQPChannelAttributes(objectPatternsList string, infoMap map[string]*ObjInfo) error {
	var err error

	traceEntry("inquireAMQPChannelAttributes")

	ci := getConnection(GetConnectionKey())
	statusClearReplyQ()

	if objectPatternsList == "" {
		traceExitErr("inquireAMQPChannelAttributes", 1, err)
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
		pcfparm.Type = ibmmq.MQCFT_INTEGER
		pcfparm.Parameter = ibmmq.MQIACH_CHANNEL_TYPE
		pcfparm.Int64Value = []int64{int64(ibmmq.MQCHT_AMQP)}
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
			traceExitErr("inquireAMQPChannelAttributes", 2, err)
			return err
		}

		for allReceived := false; !allReceived; {
			cfh, buf, allReceived, err = statusGetReply(putmqmd.MsgId)
			if buf != nil {
				parseAMQPChannelAttrData(cfh, buf, infoMap)
			}
		}
	}

	traceExit("inquireAMQPChannelAttributes", 0)
	return nil
}

func parseAMQPChannelAttrData(cfh *ibmmq.MQCFH, buf []byte, infoMap map[string]*ObjInfo) {
	var elem *ibmmq.PCFParameter
	var ci *ObjInfo
	var ok bool

	traceEntry("parseAMQPChannelAttrData")

	chlName := ""

	parmAvail := true
	bytesRead := 0
	offset := 0
	datalen := len(buf)
	if cfh.ParameterCount == 0 {
		traceExit("parseAMQPChannelAttrData", 1)
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

	traceExit("parseAMQPChannelAttrData", 0)
	return
}
