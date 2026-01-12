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

/*
Functions in this file use the DISPLAY TPSTATUS command to extract metrics
about MQ topics
*/

import (
	"strings"
	"time"

	"github.com/ibm-messaging/mq-golang/v5/ibmmq"
)

const (
	ATTR_TOPIC_STRING           = "name"
	ATTR_TOPIC_STATUS_TYPE      = "type"
	ATTR_TOPIC_PUB_MESSAGES     = "messages_published"
	ATTR_TOPIC_SUB_MESSAGES     = "messages_received"
	ATTR_TOPIC_SINCE_PUB_MSG    = "time_since_msg_published"
	ATTR_TOPIC_SINCE_SUB_MSG    = "time_since_msg_received"
	ATTR_TOPIC_PUBLISHER_COUNT  = "publisher_count"
	ATTR_TOPIC_SUBSCRIBER_COUNT = "subscriber_count"
)

/*
Unlike the statistics produced via a topic, there is no discovery
of the attributes available in object STATUS queries. There is also
no discovery of descriptions for them. So this function hardcodes the
attributes we are going to look for and gives the associated descriptive
text. The elements can be expanded later; just trying to give a starting point
for now.
*/
func TopicInitAttributes() {
	traceEntry("TopicInitAttributes")

	ci := getConnection(GetConnectionKey())
	os := &ci.objectStatus[OT_TOPIC]
	st := GetObjectStatus(GetConnectionKey(), OT_TOPIC)

	if os.init {
		traceExit("TopicInitAttributes", 1)
		return
	}
	st.Attributes = make(map[string]*StatusAttribute)

	// These fields are used to construct the key to the per-topic map values and
	// as tags to uniquely identify a topic instance
	attr := ATTR_TOPIC_STRING
	st.Attributes[attr] = newPseudoStatusAttribute(attr, "Topic String")
	attr = ATTR_TOPIC_STATUS_TYPE
	st.Attributes[attr] = newPseudoStatusAttribute(attr, "Topic Status Type")

	// These are the integer status fields that are of interest
	attr = ATTR_TOPIC_PUB_MESSAGES
	st.Attributes[attr] = newStatusAttribute(attr, "Published Messages", ibmmq.MQIACF_PUBLISH_COUNT)
	st.Attributes[attr].Delta = true // We have to manage the differences as MQ reports cumulative values
	attr = ATTR_TOPIC_SUB_MESSAGES
	st.Attributes[attr] = newStatusAttribute(attr, "Received Messages", ibmmq.MQIACF_MESSAGE_COUNT)
	st.Attributes[attr].Delta = true // We have to manage the differences as MQ reports cumulative values

	attr = ATTR_TOPIC_PUBLISHER_COUNT
	st.Attributes[attr] = newStatusAttribute(attr, "Number of publishers", ibmmq.MQIA_PUB_COUNT)
	attr = ATTR_TOPIC_SUBSCRIBER_COUNT
	st.Attributes[attr] = newStatusAttribute(attr, "Number of subscribers", ibmmq.MQIA_SUB_COUNT)

	attr = ATTR_TOPIC_SINCE_PUB_MSG
	st.Attributes[attr] = newStatusAttribute(attr, "Time Since Msg", DUMMY_PCFATTR)
	attr = ATTR_TOPIC_SINCE_SUB_MSG
	st.Attributes[attr] = newStatusAttribute(attr, "Time Since Msg", DUMMY_PCFATTR)

	os.init = true
	traceExit("TopicInitAttributes", 0)

}

// If we need to list the topics that match a pattern. Not needed for
// the status queries as they (unlike the pub/sub resource stats) accept
// patterns in the PCF command
func InquireTopics(patterns string) ([]string, error) {
	traceEntry("InquireTopics")
	TopicInitAttributes()
	rc, err := inquireObjects(patterns, ibmmq.MQOT_TOPIC)
	traceExitErr("InquireTopics", 0, err)
	return rc, err
}

func CollectTopicStatus(patterns string) error {
	var err error
	traceEntry("CollectTopicStatus")

	ci := getConnection(GetConnectionKey())
	os := &ci.objectStatus[OT_TOPIC]
	st := GetObjectStatus(GetConnectionKey(), OT_TOPIC)
	os.objectSeen = make(map[string]bool) // Record which topics have been seen in this period
	TopicInitAttributes()

	// Empty any collected values
	for k := range st.Attributes {
		st.Attributes[k].Values = make(map[string]*StatusValue)
	}

	topicPatterns := strings.Split(patterns, ",")
	if len(topicPatterns) == 0 {
		traceExit("CollectTopicStatus", 1)
		return nil
	}

	for _, pattern := range topicPatterns {
		pattern = strings.TrimSpace(pattern)
		if len(pattern) == 0 {
			continue
		}

		// Collect 3 types of status for the topics
		err1 := collectTopicStatus(pattern, ibmmq.MQIACF_TOPIC_SUB)
		err2 := collectTopicStatus(pattern, ibmmq.MQIACF_TOPIC_PUB)
		err3 := collectTopicStatus(pattern, ibmmq.MQIACF_TOPIC_STATUS)

		// If any error occurred, then report one of them
		if err1 != nil {
			err = err1
		} else if err2 != nil {
			err = err2
		} else {
			err = err3
		}

	}

	// Need to clean out the prevValues elements to stop short-lived topics
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

	traceExitErr("CollectTopicStatus", 0, err)

	return err
}

// Issue the INQUIRE_TOPIC_STATUS command for a topic or wildcarded topic name
// Collect the responses and build up the statistics
func collectTopicStatus(pattern string, instanceType int32) error {
	var err error
	traceEntryF("collectTopicStatus", "Pattern: %s", pattern)

	ci := getConnection(GetConnectionKey())
	os := &ci.objectStatus[OT_TOPIC]
	statusClearReplyQ()

	putmqmd, pmo, cfh, buf := statusSetCommandHeaders()
	// Can allow all the other fields to default
	cfh.Command = ibmmq.MQCMD_INQUIRE_TOPIC_STATUS

	// Add the parameters one at a time into a buffer
	pcfparm := new(ibmmq.PCFParameter)
	pcfparm.Type = ibmmq.MQCFT_STRING
	pcfparm.Parameter = ibmmq.MQCA_TOPIC_STRING
	pcfparm.String = []string{pattern}
	cfh.ParameterCount++
	buf = append(buf, pcfparm.Bytes()...)

	// Add the parameters one at a time into a buffer
	pcfparm = new(ibmmq.PCFParameter)
	pcfparm.Type = ibmmq.MQCFT_INTEGER
	pcfparm.Parameter = ibmmq.MQIACF_TOPIC_STATUS_TYPE
	pcfparm.Int64Value = []int64{int64(instanceType)}
	cfh.ParameterCount++
	buf = append(buf, pcfparm.Bytes()...)

	// Once we know the total number of parameters, put the
	// CFH header on the front of the buffer.
	buf = append(cfh.Bytes(), buf...)

	// And now put the command to the queue
	err = ci.si.cmdQObj.Put(putmqmd, pmo, buf)
	if err != nil {
		traceExitErr("collectTopicStatus", 1, err)
		return err

	}

	for allReceived := false; !allReceived; {
		cfh, buf, allReceived, err = statusGetReply(putmqmd.MsgId)
		if buf != nil {
			key := parseTopicData(instanceType, cfh, buf)
			if key != "" {
				os.objectSeen[key] = true
			}
		}

	}

	traceExitErr("collectTopicStatus", 0, err)

	return err
}

// Given a PCF response message, parse it to extract the desired statistics
func parseTopicData(instanceType int32, cfh *ibmmq.MQCFH, buf []byte) string {
	var elem *ibmmq.PCFParameter
	traceEntry("parseTopicData")

	st := GetObjectStatus(GetConnectionKey(), OT_TOPIC)
	tpName := ""
	key := ""

	lastMsgDate := ""
	lastMsgTime := ""

	parmAvail := true
	bytesRead := 0
	offset := 0
	datalen := len(buf)
	if cfh == nil || cfh.ParameterCount == 0 {
		traceExit("parseTopicData", 1)
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
		case ibmmq.MQCA_TOPIC_STRING:
			tpName = trimToNull(elem.String[0])
		}
	}

	instanceTypeString := "pub"
	if instanceType == ibmmq.MQIACF_TOPIC_SUB {
		instanceTypeString = "sub"
	} else if instanceType == ibmmq.MQIACF_TOPIC_STATUS {
		instanceTypeString = "status"
	}

	// It's valid for TPSTATUS to return empty topic object names. In such situations, change it to a dummy _ so we
	// have something
	if tpName == "" {
		tpName = "_"
	}
	// Create a unique key for this topic instance
	key = TopicKey(tpName, instanceTypeString)

	st.Attributes[ATTR_TOPIC_STRING].Values[key] = newStatusValueString(tpName)
	st.Attributes[ATTR_TOPIC_STATUS_TYPE].Values[key] = newStatusValueString(instanceTypeString)

	parmAvail = true
	// And then re-parse the message so we can store the metrics now knowing the map key
	offset = 0
	for parmAvail && cfh.CompCode != ibmmq.MQCC_FAILED {
		elem, bytesRead = ibmmq.ReadPCFParameter(buf[offset:])
		offset += bytesRead
		// Have we now reached the end of the message
		if offset >= datalen {
			parmAvail = false
		}

		if !statusGetIntAttributes(GetObjectStatus(GetConnectionKey(), OT_TOPIC), elem, key) {
			switch elem.Parameter {
			case ibmmq.MQCACF_LAST_MSG_TIME, ibmmq.MQCACF_LAST_PUB_TIME:
				lastMsgTime = strings.TrimSpace(elem.String[0])
			case ibmmq.MQCACF_LAST_MSG_DATE, ibmmq.MQCACF_LAST_PUB_DATE:
				lastMsgDate = strings.TrimSpace(elem.String[0])
			}
		}
	}

	// Only two of the 3 types of query return a last-used timestamp
	if lastMsgDate != "" {
		now := time.Now()
		diff := statusTimeDiff(now, lastMsgDate, lastMsgTime)
		switch instanceType {
		case ibmmq.MQIACF_TOPIC_SUB:
			st.Attributes[ATTR_TOPIC_SINCE_SUB_MSG].Values[key] = newStatusValueInt64(diff)
		case ibmmq.MQIACF_TOPIC_PUB:
			st.Attributes[ATTR_TOPIC_SINCE_PUB_MSG].Values[key] = newStatusValueInt64(diff)
		}
	}

	traceExitF("parseTopicData", 0, "Key: %s", key)

	return key
}

// Return a standardised value. If the attribute indicates that something
// special has to be done, then do that. Otherwise just make sure it's a non-negative
// value of the correct datatype
func TopicNormalise(attr *StatusAttribute, v int64) float64 {
	return statusNormalise(attr, v)
}

// Return a combination of the topic name and the status query type so we
// get unique keys in the map. There might be valid data for the same
// topic name in TYPE(SUB) and TYPE(TOPIC).
func TopicKey(n string, t string) string {
	return n + "[!" + t + "!]"
}
