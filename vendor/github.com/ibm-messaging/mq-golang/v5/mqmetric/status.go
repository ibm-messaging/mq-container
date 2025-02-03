/*
Package mqmetric contains a set of routines common to several
commands used to export MQ metrics to different backend
storage mechanisms including Prometheus and InfluxDB.
*/
package mqmetric

/*
  Copyright (c) IBM Corporation 2016, 2023

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
	// "fmt"
	"strings"
	"time"

	ibmmq "github.com/ibm-messaging/mq-golang/v5/ibmmq"
)

// var statusDummy = fmt.Sprintf("dummy")
var timeTravelWarningIssued = false

/*
This file defines types and constructors for elements related to status
of MQ objects that are retrieved via polling commands such as DISPLAY CHSTATUS
*/

type StatusAttribute struct {
	Description string
	MetricName  string
	Pseudo      bool
	Delta       bool
	pcfAttr     int32
	squash      bool
	index       int
	Values      map[string]*StatusValue
	prevValues  map[string]int64
}

type StatusSet struct {
	Attributes map[string]*StatusAttribute
}

// All we care about for attributes are ints and strings. Other complex
// PCF datatypes are not currently going to be returned through this mechanism
type StatusValue struct {
	IsInt64     bool
	ValueInt64  int64
	ValueString string
}

// Initialise with default values.
func newStatusAttribute(n string, d string, p int32) *StatusAttribute {
	s := new(StatusAttribute)
	s.MetricName = formatAttrName(n) // Convert to a canonical metric name
	s.Description = d
	s.Delta = false
	s.pcfAttr = p
	s.squash = false
	s.index = -1
	s.Values = make(map[string]*StatusValue)
	s.prevValues = make(map[string]int64)
	s.Pseudo = false
	return s
}

func newPseudoStatusAttribute(n string, d string) *StatusAttribute {
	s := newStatusAttribute(n, d, -1)
	s.Pseudo = true
	return s
}

func newStatusValueInt64(v int64) *StatusValue {
	s := new(StatusValue)
	s.ValueInt64 = v
	s.IsInt64 = true
	return s
}

func newStatusValueString(v string) *StatusValue {
	s := new(StatusValue)
	s.ValueString = v
	s.IsInt64 = false
	return s
}

// Go uses an example-based method for formatting and parsing timestamps
// This layout matches the MQ PutDate and PutTime strings. An additional TZ
// may eventually have to be turned into a config parm. Note the "15" to indicate
// a 24-hour timestamp. There also seems to be two formats for the time layout coming
// from MQ - TPSTATUS uses a colon format time, QSTATUS uses the dots.
const timeStampLayoutDot = "2006-01-02 15.04.05"
const timeStampLayoutColon = "2006-01-02 15:04:05"

// Convert the MQ Time and Date formats
func statusTimeDiff(now time.Time, d string, t string) int64 {
	var rc int64
	var err error
	var parsedT time.Time

	traceEntry("statusTimeDiff")
	ci := getConnection(GetConnectionKey())

	// If there's any error in parsing the timestamp - perhaps
	// the value has not been set yet - then just return 0
	rc = 0

	timeStampLayout := timeStampLayoutDot
	if len(d) == 10 && len(t) == 8 {
		if strings.Contains(t, ":") {
			timeStampLayout = timeStampLayoutColon
		}
		parsedT, err = time.ParseInLocation(timeStampLayout, d+" "+t, now.Location())
		if err == nil {
			diff := now.Sub(parsedT).Seconds() + ci.tzOffsetSecs

			if diff < 0 {
				if diff < -(60 * 5) { // Cannot have status from the future but allow a tiny amount of flex
					if !timeTravelWarningIssued {
						logError("Status reports appear to be from the future. Difference is approximately %d seconds. Check the TZ Offset value in the program configuration.", int64(-diff))
						logDebug("statusTimeDiff d:%s t:%s diff:%f tzoffset: %f err:%v\n", d, t, diff, ci.tzOffsetSecs, err)
						timeTravelWarningIssued = true
					}
				}
				diff = 0
			}
			rc = int64(diff)
		}
	}
	//logError("statusTimeDiff d:%s t:%s diff:%d tzoffset: %f err:%v\n", d, t, rc, ci.tzOffsetSecs, err)
	traceExitF("statusTimeDiff", 0, "Diff: %d", rc)
	return rc
}

func clearQ(hObj ibmmq.MQObject) {
	buf := make([]byte, 0)
	// Empty replyQ in case any left over from previous errors
	for ok := true; ok; {
		getmqmd := ibmmq.NewMQMD()
		gmo := ibmmq.NewMQGMO()
		gmo.Options = ibmmq.MQGMO_NO_SYNCPOINT
		gmo.Options |= ibmmq.MQGMO_FAIL_IF_QUIESCING
		gmo.Options |= ibmmq.MQGMO_NO_WAIT
		gmo.Options |= ibmmq.MQGMO_CONVERT
		gmo.Options |= ibmmq.MQGMO_ACCEPT_TRUNCATED_MSG
		_, err := hObj.Get(getmqmd, gmo, buf)

		if err != nil && err.(*ibmmq.MQReturn).MQCC == ibmmq.MQCC_FAILED {
			ok = false
		}
	}
	return
}

func statusClearReplyQ() {
	traceEntry("statusClearReplyQ")
	ci := getConnection(GetConnectionKey())

	clearQ(ci.si.statusReplyQObj)

	traceExit("statusClearReplyQ", 0)
	return
}

// Create the control blocks needed to send an admin message to the command
// server. The caller of this function will complete the message contents
// with elements specific to the object type.
func statusSetCommandHeaders() (*ibmmq.MQMD, *ibmmq.MQPMO, *ibmmq.MQCFH, []byte) {
	traceEntry("statusSetCommandHeaders")
	ci := getConnection(GetConnectionKey())

	cfh := ibmmq.NewMQCFH()
	cfh.Version = ibmmq.MQCFH_VERSION_3
	cfh.Type = ibmmq.MQCFT_COMMAND_XR

	putmqmd := ibmmq.NewMQMD()
	pmo := ibmmq.NewMQPMO()

	pmo.Options = ibmmq.MQPMO_NO_SYNCPOINT
	pmo.Options |= ibmmq.MQPMO_NEW_MSG_ID
	pmo.Options |= ibmmq.MQPMO_NEW_CORREL_ID
	pmo.Options |= ibmmq.MQPMO_FAIL_IF_QUIESCING

	putmqmd.Format = "MQADMIN"
	putmqmd.ReplyToQ = ci.si.statusReplyQObj.Name
	putmqmd.MsgType = ibmmq.MQMT_REQUEST
	putmqmd.Report = ibmmq.MQRO_PASS_DISCARD_AND_EXPIRY

	buf := make([]byte, 0)

	traceExit("statusSetCommandHeaders", 0)

	return putmqmd, pmo, cfh, buf
}

// Get a reply from the command server, returning the buffer
// to be parsed. This function is called in a loop until
// it has returned allDone=true (with or without an error)
// The command request is assigned a MsgId when MQPUT; that is
// used as the CorrelId for all related responses.
func statusGetReply(correlId []byte) (*ibmmq.MQCFH, []byte, bool, error) {
	var offset int
	var cfh *ibmmq.MQCFH

	traceEntry("statusGetReply")
	ci := getConnection(GetConnectionKey())

	replyBuf := make([]byte, 10240)

	getmqmd := ibmmq.NewMQMD()
	gmo := ibmmq.NewMQGMO()
	gmo.Options = ibmmq.MQGMO_NO_SYNCPOINT
	gmo.Options |= ibmmq.MQGMO_FAIL_IF_QUIESCING
	gmo.Options |= ibmmq.MQGMO_WAIT
	gmo.Options |= ibmmq.MQGMO_CONVERT
	gmo.WaitInterval = int32(ci.waitInterval) * 1000 // 3 seconds by default

	getmqmd.CorrelId = correlId
	gmo.MatchOptions = ibmmq.MQMO_MATCH_CORREL_ID
	gmo.Version = ibmmq.MQGMO_VERSION_2

	allDone := false
	datalen, err := ci.si.statusReplyQObj.Get(getmqmd, gmo, replyBuf)
	if err == nil {
		cfh, offset = ibmmq.ReadPCFHeader(replyBuf)

		if cfh.Control == ibmmq.MQCFC_LAST {
			allDone = true
		}

		if cfh.Reason != ibmmq.MQRC_NONE {
			// A "normal" error might come back in 2 messages so we do not
			// force allDone here. For example, issuing an INQUIRE_CHL_STATUS
			// might get first response with Reason=STATUS_NOT_FOUND followed by
			// second response with Reason=MQRCCF_COMMAND_FAILED and Control=Last
			traceExitF("statusGetReply", 1, "CFH: %+v Error: %v", cfh, err)
			return cfh, nil, allDone, err
		}
		// Returned by z/OS qmgrs but are not interesting
		if cfh.Type == ibmmq.MQCFT_XR_SUMMARY || cfh.Type == ibmmq.MQCFT_XR_MSG {
			traceExitErr("statusGetReply", 2, err)
			return cfh, nil, allDone, err
		}
	} else {
		// After any MQ error (including 2033) we are not likely to receive more usable messages
		// so return that this is done - whether or not we've seen a CFC_LAST flag.
		// If further messages do show up later, they should be discarded before the next
		// command tries to use this replyQ.
		allDone = true
		if err.(*ibmmq.MQReturn).MQRC != ibmmq.MQRC_NO_MSG_AVAILABLE {
			logError("StatusGetReply error : %v\n", err)
		}
		traceExitErr("statusGetReply", 3, err)
		return nil, nil, allDone, err
	}
	traceExitErr("statusGetReply", 0, err)
	return cfh, replyBuf[offset:datalen], allDone, err
}

// Called in a loop for each PCF Parameter element returned from the command
// server messages. We can deal here with the various integer responses; string
// responses need to be handled in the object-specific caller.
func statusGetIntAttributes(s *StatusSet, elem *ibmmq.PCFParameter, key string) bool {
	// traceEntry("statusGetIntAttributes") // Don't trace as too noisy
	usableValue := false
	if elem.Type == ibmmq.MQCFT_INTEGER || elem.Type == ibmmq.MQCFT_INTEGER64 ||
		elem.Type == ibmmq.MQCFT_INTEGER_LIST || elem.Type == ibmmq.MQCFT_INTEGER64_LIST {
		usableValue = true
	}

	if !usableValue {
		//traceExit("statusGetIntAttributes", 1)
		return false
	}

	// Look at the Parameter and loop through all the possible status
	// attributes to find it. We don't break from the loop after finding a match
	// because there might be more than one attribute associated with the
	// attribute (in particular status/status_squash)
	for attr, _ := range s.Attributes {
		if s.Attributes[attr].pcfAttr == elem.Parameter {
			index := s.Attributes[attr].index

			// Some MQ responses (eg QTIME) are arrays which we need to split into
			// individual metrics which we do via the index field describing the
			// metric attribute.
			if index == -1 {
				v := elem.Int64Value[0]
				if s.Attributes[attr].Delta {
					// If we have already got a value for this attribute and queue
					// then use it to create the delta. Otherwise make the initial
					// value 0.
					if prevVal, ok := s.Attributes[attr].prevValues[key]; ok {
						if v-prevVal < 0 {
							// Value might have wrapped. This number may be temporarily "wrong" but it's
							// sorted out on the next iteration
							s.Attributes[attr].Values[key] = newStatusValueInt64(v)
						} else {
							s.Attributes[attr].Values[key] = newStatusValueInt64(v - prevVal)
						}
					} else {
						s.Attributes[attr].Values[key] = newStatusValueInt64(0)
					}
					s.Attributes[attr].prevValues[key] = v
				} else {
					// Return the actual number
					s.Attributes[attr].Values[key] = newStatusValueInt64(v)
				}
			} else {
				v := elem.Int64Value
				if s.Attributes[attr].Delta {
					// If we have already got a value for this attribute and queue
					// then use it to create the delta. Otherwise make the initial
					// value 0.
					if prevVal, ok := s.Attributes[attr].prevValues[key]; ok {
						if v[index]-prevVal < 0 {
							s.Attributes[attr].Values[key] = newStatusValueInt64(v[index])
						} else {
							s.Attributes[attr].Values[key] = newStatusValueInt64(v[index] - prevVal)
						}
					} else {
						s.Attributes[attr].Values[key] = newStatusValueInt64(0)
					}
					s.Attributes[attr].prevValues[key] = v[index]
				} else {
					// Return the actual number
					s.Attributes[attr].Values[key] = newStatusValueInt64(v[index])
				}
			}
		}
	}

	//traceExit("statusGetIntAttributes", 0)
	return true
}

// Common function to turn MQ integer value into a non-negative float. May
// be overridden in specific object types where special processing may be needed.
func statusNormalise(attr *StatusAttribute, v int64) float64 {
	f := float64(v)
	if f < 0 {
		f = 0
	}
	return f
}
