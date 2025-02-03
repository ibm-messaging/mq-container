/*
Package mqmetric contains a set of routines common to several
commands used to export MQ metrics to different backend
storage mechanisms including Prometheus and InfluxDB.
*/
package mqmetric

/*
  Copyright (c) IBM Corporation 2018,2021

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
Functions in this file use the DISPLAY SubStatus command to extract metrics
about MQ subscriptions
*/

import (
	"strings"
	"time"

	"github.com/ibm-messaging/mq-golang/v5/ibmmq"
)

const (
	ATTR_SUB_NAME = "name"
	ATTR_SUB_ID   = "subid"

	ATTR_SUB_TOPIC_STRING  = "topic"
	ATTR_SUB_TYPE          = "type"
	ATTR_SUB_SINCE_PUB_MSG = "time_since_message_published"
	ATTR_SUB_MESSAGES      = "messsages_received"
)

/*
Unlike the statistics produced via a topic, there is no discovery
of the attributes available in object STATUS queries. There is also
no discovery of descriptions for them. So this function hardcodes the
attributes we are going to look for and gives the associated descriptive
text. The elements can be expanded later; just trying to give a starting point
for now.
*/
func SubInitAttributes() {
	traceEntry("SubInitAttributes")
	ci := getConnection(GetConnectionKey())
	os := &ci.objectStatus[OT_SUB]
	st := GetObjectStatus(GetConnectionKey(), OT_SUB)

	if os.init {
		traceExit("SubInitAttributes", 1)
		return
	}
	st.Attributes = make(map[string]*StatusAttribute)

	attr := ATTR_SUB_ID
	st.Attributes[attr] = newPseudoStatusAttribute(attr, "Subscription Id")
	attr = ATTR_SUB_NAME
	st.Attributes[attr] = newPseudoStatusAttribute(attr, "Subscription Name")
	attr = ATTR_SUB_TOPIC_STRING
	st.Attributes[attr] = newPseudoStatusAttribute(attr, "Topic String")

	attr = ATTR_SUB_TYPE
	st.Attributes[attr] = newStatusAttribute(attr, "Subscription Type", ibmmq.MQIACF_SUB_TYPE)

	attr = ATTR_SUB_SINCE_PUB_MSG
	st.Attributes[attr] = newStatusAttribute(attr, "Time Since Message Received", -1)

	// These are the integer status fields that are of interest
	attr = ATTR_SUB_MESSAGES
	st.Attributes[attr] = newStatusAttribute(attr, "Messages Received", ibmmq.MQIACF_MESSAGE_COUNT)
	st.Attributes[attr].Delta = true

	os.init = true
	traceExit("SubInitAttributes", 0)
}

func CollectSubStatus(patterns string) error {
	var err error
	traceEntry("CollectSubStatus")

	st := GetObjectStatus(GetConnectionKey(), OT_SUB)
	SubInitAttributes()

	// Empty any collected values
	for k := range st.Attributes {
		st.Attributes[k].Values = make(map[string]*StatusValue)
	}

	subPatterns := strings.Split(patterns, ",")
	if len(subPatterns) == 0 {
		traceExit("CollectSubStatus", 1)
		return nil
	}

	for _, pattern := range subPatterns {
		pattern = strings.TrimSpace(pattern)
		if len(pattern) == 0 {
			continue
		}

		err = collectSubStatus(pattern)

	}

	traceExitErr("CollectSubStatus", 0, err)

	return err
}

// Issue the INQUIRE_SUB_STATUS command for a subscription name pattern
// Collect the responses and build up the statistics
func collectSubStatus(pattern string) error {
	var err error

	traceEntryF("collectSubStatus", "Pattern: %s", pattern)
	ci := getConnection(GetConnectionKey())

	statusClearReplyQ()

	putmqmd, pmo, cfh, buf := statusSetCommandHeaders()

	// Can allow all the other fields to default
	cfh.Command = ibmmq.MQCMD_INQUIRE_SUB_STATUS

	// Add the parameters one at a time into a buffer
	pcfparm := new(ibmmq.PCFParameter)
	pcfparm.Type = ibmmq.MQCFT_STRING
	pcfparm.Parameter = ibmmq.MQCACF_SUB_NAME
	pcfparm.String = []string{pattern}
	cfh.ParameterCount++
	buf = append(buf, pcfparm.Bytes()...)

	// Once we know the total number of parameters, put the
	// CFH header on the front of the buffer.
	buf = append(cfh.Bytes(), buf...)

	// And now put the command to the queue
	err = ci.si.cmdQObj.Put(putmqmd, pmo, buf)
	if err != nil {
		traceExitErr("collectSubStatus", 1, err)

		return err
	}

	// Now get the responses - loop until all have been received (one
	// per queue) or we run out of time
	for allReceived := false; !allReceived; {
		cfh, buf, allReceived, err = statusGetReply(putmqmd.MsgId)
		if buf != nil {
			parseSubData(cfh, buf)
		}
	}

	traceExitErr("collectSubStatus", 0, err)

	return err
}

// Given a PCF response message, parse it to extract the desired statistics
func parseSubData(cfh *ibmmq.MQCFH, buf []byte) string {
	var elem *ibmmq.PCFParameter

	traceEntry("parseSubData")

	st := GetObjectStatus(GetConnectionKey(), OT_SUB)
	subName := ""
	subId := ""
	key := ""
	topicString := ""

	lastTime := ""
	lastDate := ""

	parmAvail := true
	bytesRead := 0
	offset := 0
	datalen := len(buf)
	if cfh == nil || cfh.ParameterCount == 0 {
		traceExit("parseSubData", 1)
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
		case ibmmq.MQBACF_SUB_ID:
			subId = trimToNull(elem.String[0])
		case ibmmq.MQCA_TOPIC_STRING:
			topicString = trimToNull(elem.String[0])
		}
	}

	// Create a unique key for this instance
	key = subId

	st.Attributes[ATTR_SUB_ID].Values[key] = newStatusValueString(subId)

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

		if !statusGetIntAttributes(GetObjectStatus(GetConnectionKey(), OT_SUB), elem, key) {
			switch elem.Parameter {
			case ibmmq.MQCACF_LAST_MSG_TIME:
				lastTime = strings.TrimSpace(elem.String[0])
			case ibmmq.MQCACF_LAST_MSG_DATE:
				lastDate = strings.TrimSpace(elem.String[0])
			case ibmmq.MQCA_TOPIC_STRING:
				topicString = trimToNull(elem.String[0])
			case ibmmq.MQCACF_SUB_NAME:
				subName = trimToNull(elem.String[0])
			}
		}
	}

	now := time.Now()
	st.Attributes[ATTR_SUB_SINCE_PUB_MSG].Values[key] = newStatusValueInt64(statusTimeDiff(now, lastDate, lastTime))
	st.Attributes[ATTR_SUB_TOPIC_STRING].Values[key] = newStatusValueString(topicString)
	st.Attributes[ATTR_SUB_NAME].Values[key] = newStatusValueString(subName)

	traceExitF("parseSubData", 0, "Key : %s", key)

	return key
}

// Return a standardised value. If the attribute indicates that something
// special has to be done, then do that. Otherwise just make sure it's a non-negative
// value of the correct datatype
func SubNormalise(attr *StatusAttribute, v int64) float64 {
	return statusNormalise(attr, v)
}
