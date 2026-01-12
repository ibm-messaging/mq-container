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
This file holds most of the calls to the MQI, so we
don't need to repeat common setups eg of MQMD or MQSD structures.
*/

import (
	"errors"
	"fmt"
	"os"

	"github.com/ibm-messaging/mq-golang/v5/ibmmq"
)

var (
	getBuffer = make([]byte, 32768)
	// if true, then use qmgr ccsid to convert resource metric publications. if false, always assume 1208
	convertToAlternateCP = false
	transNeeded          = true
)

const ALTERNATE_CCSID = 1208

type ConnectionConfig struct {
	ClientMode    bool
	UserId        string
	Password      string
	TZOffsetSecs  float64
	SingleConnect bool

	UsePublications bool
	UseStatus       bool
	UseResetQStats  bool
	UseStatistics   bool
	StatisticsQ     string

	ShowInactiveChannels bool
	ShowCustomAttribute  bool
	HideSvrConnJobname   bool
	HideAMQPClientId     bool
	HideMQTTClientId     bool

	WaitInterval int

	CcdtUrl  string
	ConnName string
	Channel  string

	DurableSubPrefix string
}

// Which objects are available for subscription. How
// do we define which ones to subscribe to and filter the
// specific subscriptions.

type DiscoverObject struct {
	ObjectNames          string
	UseWildcard          bool
	SubscriptionSelector string
}

// For now, only queues are subscribable through this interface
// but there are now Application resources that might be relevant
// at some time.
type DiscoverConfig struct {
	MetaPrefix      string // Root of all meta-data discovery
	MonitoredQueues DiscoverObject
}

type MQMetricError struct {
	Err      string
	MQReturn *ibmmq.MQReturn
}

type MQTopicDescriptor struct {
	hObj    ibmmq.MQObject
	topic   string
	durable bool
	managed bool
}

func init() {
	if os.Getenv("IBMMQ_CONVERT_SUBS") != "" {
		convertToAlternateCP = true
	}
}

func (e MQMetricError) Error() string { return e.Err + " : " + e.MQReturn.Error() }
func (e MQMetricError) Unwrap() error { return e.MQReturn }

/*
InitConnection connects to the queue manager, and then
opens both the command queue and a dynamic reply queue
to be used for all responses including the publications
*/
func InitConnection(qMgrName string, replyQ string, replyQ2 string, cc *ConnectionConfig) error {
	return initConnectionKey("", qMgrName, replyQ, replyQ2, cc)
}
func InitConnectionKey(key string, qMgrName string, replyQ string, replyQ2 string, cc *ConnectionConfig) error {
	return initConnectionKey(key, qMgrName, replyQ, replyQ2, cc)
}
func initConnectionKey(key string, qMgrName string, replyQ string, replyQ2 string, cc *ConnectionConfig) error {
	var err error
	var gocd *ibmmq.MQCD
	var mqreturn *ibmmq.MQReturn
	var errorString = ""

	traceEntryF("initConnectionKey", "QMgrName %s", qMgrName)

	initConnection(key)

	gocno := ibmmq.NewMQCNO()
	gocsp := ibmmq.NewMQCSP()

	// Copy initialisation configuraton information to local structure
	ci := getConnection(GetConnectionKey())

	ci.tzOffsetSecs = cc.TZOffsetSecs
	ci.showInactiveChannels = cc.ShowInactiveChannels
	ci.showCustomAttribute = cc.ShowCustomAttribute
	ci.hideSvrConnJobname = cc.HideSvrConnJobname
	ci.hideAMQPClientId = cc.HideAMQPClientId
	ci.hideMQTTClientId = cc.HideMQTTClientId

	ci.durableSubPrefix = cc.DurableSubPrefix

	// Explicitly force client mode if requested. Otherwise use the "default"
	// Client mode can be come from a simple boolean, or from having
	// common configurations with the CCDT or ConnName/Channel being set.
	if cc.CcdtUrl != "" {
		cc.ClientMode = true
	} else if cc.ConnName != "" || cc.Channel != "" {
		cc.ClientMode = true
		gocd = ibmmq.NewMQCD()
		gocd.ChannelName = cc.Channel
		gocd.ConnectionName = cc.ConnName
	}

	// connection mechanism depending on what is installed or configured.
	if cc.ClientMode {
		gocno.Options = ibmmq.MQCNO_CLIENT_BINDING
		// Force reconnection to only be to the same qmgr. Cannot do this with externally
		// configured (eg MQ_CONNECT_TYPE or client-only installation) connections. But
		// it is a bad idea to try to reconnect to a different queue manager.
		// If the collector is managing its own reconnect, then don't use the MQ automatic mode
		if cc.SingleConnect {
			gocno.Options |= ibmmq.MQCNO_RECONNECT_DISABLED
		} else {
			gocno.Options |= ibmmq.MQCNO_RECONNECT_Q_MGR
		}
		if cc.CcdtUrl != "" {
			gocno.CCDTUrl = cc.CcdtUrl
			logInfo("Trying to connect as client using CCDT: %s", gocno.CCDTUrl)
		} else if gocd != nil {
			gocno.ClientConn = gocd
			logInfo("Trying to connect as client using ConnName: %s, Channel: %s", gocd.ConnectionName, gocd.ChannelName)
		} else {
			logInfo("Trying to connect as client with external configuration")
		}
	}
	gocno.Options |= ibmmq.MQCNO_HANDLE_SHARE_BLOCK

	if cc.Password != "" {
		gocsp.Password = cc.Password
	}
	if cc.UserId != "" {
		gocsp.UserId = cc.UserId
		gocno.SecurityParms = gocsp
	}

	logDebug("Connecting to queue manager %s", qMgrName)
	ci.si.qMgr, err = ibmmq.Connx(qMgrName, gocno)
	if err == nil {
		ci.si.qmgrConnected = true
	} else {
		errorString = "Cannot connect to queue manager " + qMgrName
		mqreturn = err.(*ibmmq.MQReturn)
	}

	// Discover important information about the qmgr - its real name
	// and the platform type. Also check if it is at least V9 (on Distributed platforms)
	// so that monitoring will work.
	if err == nil {
		var v map[int32]interface{}

		ci.useStatus = cc.UseStatus
		ci.waitInterval = cc.WaitInterval

		mqod := ibmmq.NewMQOD()
		openOptions := ibmmq.MQOO_INQUIRE + ibmmq.MQOO_FAIL_IF_QUIESCING

		mqod.ObjectType = ibmmq.MQOT_Q_MGR
		mqod.ObjectName = ""

		ci.si.qMgrObject, err = ci.si.qMgr.Open(mqod, openOptions)

		if err == nil {
			selectors := []int32{ibmmq.MQCA_Q_MGR_NAME,
				ibmmq.MQIA_COMMAND_LEVEL,
				ibmmq.MQIA_PERFORMANCE_EVENT,
				ibmmq.MQIA_MAX_HANDLES,
				ibmmq.MQIA_PLATFORM}
			if cc.UseStatistics {
				// We don't know yet what the platform is.
				// If it turns out to be z/OS, the MQINQ will fail, but that's OK:
				// You would have to correct the configuration to not use these events.
				selectors = append(selectors, ibmmq.MQIA_STATISTICS_MQI)
			}

			v, err = ci.si.qMgrObject.Inq(selectors)
			if err == nil {
				ci.si.resolvedQMgrName = v[ibmmq.MQCA_Q_MGR_NAME].(string)
				ci.si.platform = v[ibmmq.MQIA_PLATFORM].(int32)
				ci.si.commandLevel = v[ibmmq.MQIA_COMMAND_LEVEL].(int32)
				ci.si.maxHandles = v[ibmmq.MQIA_MAX_HANDLES].(int32)
				ci.useStatistics = cc.UseStatistics

				if ci.si.platform == ibmmq.MQPL_ZOS {
					ci.usePublications = false
					ci.useStatistics = false

					ci.useResetQStats = cc.UseResetQStats
					evEnabled := v[ibmmq.MQIA_PERFORMANCE_EVENT].(int32)
					if ci.useResetQStats && evEnabled == 0 {
						errorString = "Requested use of RESET QSTATS but queue manager has PERFMEV(DISABLED)"
						err = errors.New(errorString)
						mqreturn = &ibmmq.MQReturn{MQCC: ibmmq.MQCC_FAILED, MQRC: ibmmq.MQRC_ENVIRONMENT_ERROR}
					}
				} else {
					if cc.UsePublications {
						if ci.si.commandLevel < 900 && ci.si.platform != ibmmq.MQPL_APPLIANCE {
							errorString = "Unsupported system: Queue manager must be at least V9.0 for full monitoring. Disable the usePublications attribute for limited capability."
							err = errors.New(errorString)
							mqreturn = &ibmmq.MQReturn{MQCC: ibmmq.MQCC_FAILED, MQRC: ibmmq.MQRC_ENVIRONMENT_ERROR}
						} else {
							ci.usePublications = cc.UsePublications
						}
					} else {
						ci.usePublications = false
					}

					if cc.UseStatistics {
						evEnabled := v[ibmmq.MQIA_STATISTICS_MQI].(int32)
						if evEnabled == 0 {
							errorString = "Configuration has asked to use statistics events but queue manager has STATMQI set to OFF."
							err = errors.New(errorString)
							mqreturn = &ibmmq.MQReturn{MQCC: ibmmq.MQCC_FAILED, MQRC: ibmmq.MQRC_ENVIRONMENT_ERROR}
						}
					}
				}
			} else {
				errorString = "Cannot inquire on queue manager object"
				if cc.UseStatistics {
					errorString += ". Cannot set the useStatistics option for z/OS queue managers"
				}
				mqreturn = err.(*ibmmq.MQReturn)
			}
		} else {
			errorString = "Cannot open queue manager object"
			mqreturn = err.(*ibmmq.MQReturn)
		}
	}

	// err = errors.New("TESTING") // for testing - force an error

	// MQOPEN of the COMMAND QUEUE
	if err == nil {
		mqod := ibmmq.NewMQOD()

		openOptions := ibmmq.MQOO_OUTPUT | ibmmq.MQOO_FAIL_IF_QUIESCING

		mqod.ObjectType = ibmmq.MQOT_Q
		mqod.ObjectName = "SYSTEM.ADMIN.COMMAND.QUEUE"
		if ci.si.platform == ibmmq.MQPL_ZOS {
			mqod.ObjectName = "SYSTEM.COMMAND.INPUT"
		}

		ci.si.cmdQObj, err = ci.si.qMgr.Open(mqod, openOptions)
		if err != nil {
			errorString = "Cannot open queue " + mqod.ObjectName
			mqreturn = err.(*ibmmq.MQReturn)
		}

	}

	// MQOPEN of a reply queue used for status polling, and responses to any other PCF commands.
	// Although this is replyQ2, we now open it first to give a chance of executing CLEAR QL(replyQ) before
	// that subscription delivery queue has been opened
	inputOpt := ibmmq.MQOO_INPUT_EXCLUSIVE
	if err == nil {
		mqod := ibmmq.NewMQOD()
		openOptions := inputOpt | ibmmq.MQOO_FAIL_IF_QUIESCING
		openOptions |= ibmmq.MQOO_INQUIRE

		mqod.ObjectType = ibmmq.MQOT_Q
		ci.si.replyQ2BaseName = replyQ2
		if replyQ2 != "" {
			mqod.ObjectName = replyQ2
		} else {
			mqod.ObjectName = replyQ
		}
		ci.si.statusReplyQObj, err = ci.si.qMgr.Open(mqod, openOptions)
		if err != nil {
			errorString = "Cannot open queue " + mqod.ObjectName
			mqreturn = err.(*ibmmq.MQReturn)
		} else {
			ci.si.queuesOpened = true

			// There may be performance benefits to using READAHEAD on the reply queues, but we
			// need to know if it's going to be used. We don't use the MQOO option to ask for
			// it explicitly, but rely on the queue's default option. So find that with an MQINQ
			// and stash it.
			ci.si.statusReplyQReadAhead = false

			selectors := []int32{ibmmq.MQIA_DEF_READ_AHEAD}
			vals, inqErr := ci.si.statusReplyQObj.Inq(selectors)
			if inqErr == nil {
				ra := vals[ibmmq.MQIA_DEF_READ_AHEAD]
				if ra == ibmmq.MQREADA_YES {
					ci.si.statusReplyQReadAhead = true
				}
				logTrace("DEF_READ_AHEAD for %s: %d", mqod.ObjectName, ra)

			} else {
				// log the error but ignore it
				logWarn("Cannot find DEF_READ_AHEAD for %s: %v", mqod.ObjectName, inqErr)

			}

			// This queue ought always to be empty, but we'll clear it via repeated MQGETs anyway.
			clearQ(ci.si.statusReplyQObj, ci.si.statusReplyQReadAhead)

			// Before we open the subscription delivery queue, we will try to clear it using the CLEAR QLOCAL.
			// command. We can't do that once it's been opened. But if this fails, ignore the error. The reply queue
			// names have to be different, and replyQ must refer to a local queue. But we can't check the queue type
			// in advance, so we'll try it anyway.
			if replyQ2 != "" && replyQ2 != replyQ {
				clearQPCF(replyQ, ci.si.cmdQObj, ci.si.statusReplyQObj, ci.si.statusReplyQReadAhead)
			}

			// Similarly, clear the statistics queue (default SYSTEM.ADMIN.STATISTICS.QUEUE) if that is configured
			if cc.UseStatistics && cc.StatisticsQ != "" {
				// clearQPCF(cc.StatisticsQ, ci.si.cmdQObj, ci.si.statusReplyQObj, ci.si.statusReplyQReadAhead)
			}
		}
	}

	// MQOPEN of a reply queue used for subscription delivery
	if err == nil {
		mqod := ibmmq.NewMQOD()
		openOptions := inputOpt | ibmmq.MQOO_FAIL_IF_QUIESCING
		openOptions |= ibmmq.MQOO_INQUIRE
		mqod.ObjectType = ibmmq.MQOT_Q
		mqod.ObjectName = replyQ
		ci.si.replyQObj, err = ci.si.qMgr.Open(mqod, openOptions)
		ci.si.replyQBaseName = replyQ
		if err == nil {
			ci.si.queuesOpened = true
			ci.si.replyQReadAhead = false

			// Do the inquire again, this time for the main replyQ, if it's not the same as the 2ary reply queue
			if replyQ2 != "" && replyQ2 != replyQ {
				selectors := []int32{ibmmq.MQIA_DEF_READ_AHEAD}
				vals, inqErr := ci.si.replyQObj.Inq(selectors)
				if inqErr == nil {
					ra := vals[ibmmq.MQIA_DEF_READ_AHEAD]
					if ra == ibmmq.MQREADA_YES {
						ci.si.replyQReadAhead = true
					}
					logTrace("DEF_READ_AHEAD for %s: %d", mqod.ObjectName, ra)
				} else {
					// log the error but ignore it
					logWarn("Cannot find DEF_READ_AHEAD for %s: %v", mqod.ObjectName, inqErr)
				}
			} else {
				ci.si.replyQReadAhead = ci.si.statusReplyQReadAhead
			}

			clearQ(ci.si.replyQObj, ci.si.replyQReadAhead)
		} else {
			errorString = "Cannot open queue " + mqod.ObjectName
			mqreturn = err.(*ibmmq.MQReturn)
		}
	}

	// MQOPEN of the queue used for statistics messages if configured
	if err == nil && cc.UseStatistics {
		mqod := ibmmq.NewMQOD()
		openOptions := inputOpt | ibmmq.MQOO_FAIL_IF_QUIESCING
		openOptions |= ibmmq.MQOO_INQUIRE

		mqod.ObjectType = ibmmq.MQOT_Q
		mqod.ObjectName = cc.StatisticsQ
		ci.si.statisticsQObj, err = ci.si.qMgr.Open(mqod, openOptions)
		ci.si.statisticsQName = cc.StatisticsQ
		ci.si.statisticsQBuf = make([]byte, 32768)
		if err == nil {
			ci.si.queuesOpened = true
			ci.si.statisticsQReadAhead = false

			// Do the inquire again, this time for the statistics queue
			selectors := []int32{ibmmq.MQIA_DEF_READ_AHEAD}
			vals, inqErr := ci.si.statisticsQObj.Inq(selectors)
			if inqErr == nil {
				ra := vals[ibmmq.MQIA_DEF_READ_AHEAD]
				if ra == ibmmq.MQREADA_YES {
					ci.si.statisticsQReadAhead = true
				}
				logTrace("DEF_READ_AHEAD for %s: %d", mqod.ObjectName, ra)
			} else {
				// log the error but ignore it
				logWarn("Cannot find DEF_READ_AHEAD for %s: %v", mqod.ObjectName, inqErr)
			}

			clearQ(ci.si.statisticsQObj, ci.si.statisticsQReadAhead)

			initStatisticsAttrs(ci)

		} else {
			errorString = "Cannot open queue " + mqod.ObjectName
			mqreturn = err.(*ibmmq.MQReturn)
		}
	}

	// Start from a clean set of subscriptions. Errors from this can be ignored.
	if err == nil && ci.durableSubPrefix != "" && ci.usePublications {
		clearDurableSubscriptions(ci.durableSubPrefix, ci.si.cmdQObj, ci.si.statusReplyQObj, ci.si.statusReplyQReadAhead)
	}

	// If anything has gone wrong in the initial connection and object access then return an error.
	if err != nil {
		// All the `err` objects should have been converted to `mqreturn` objects. But if not,
		// then force it with a FAILED code.
		if mqreturn == nil {
			mqreturn = &ibmmq.MQReturn{MQCC: ibmmq.MQCC_FAILED, MQRC: ibmmq.MQRC_ENVIRONMENT_ERROR}
		}
		if errorString == "" {
			errorString = err.Error()
		}
		traceExitErr("initConnectionKey", 1, mqreturn)
		return MQMetricError{Err: errorString, MQReturn: mqreturn}
	}

	logTrace("initConnectionKey: Queue manager resolved info - %+v", ci /*.si*/)
	traceExitErr("initConnectionKey", 0, mqreturn)

	return err
}

/*
EndConnection tidies up by closing the queues and disconnecting. All errors here
are ignored as there's very little that can be done if anything goes wrong.
*/
func EndConnection() {
	traceEntry("EndConnection")

	ci := getConnection(GetConnectionKey())
	if ci == nil {
		traceExit("EndConnection", 1)
		return
	}
	m := GetPublishedMetrics(GetConnectionKey())
	// MQCLOSE all subscriptions
	if ci.si.subsOpened {
		for _, cl := range m.Classes {
			for _, ty := range cl.Types {
				for _, hObj := range ty.subHobj {
					hObj.unsubscribe()
				}
			}
		}
	}

	// MQCLOSE the queues
	if ci.si.queuesOpened {
		ci.si.cmdQObj.Close(0)
		ci.si.replyQObj.Close(0)
		ci.si.statusReplyQObj.Close(0)
		ci.si.qMgrObject.Close(0)
		if ci.useStatistics {
			ci.si.statisticsQObj.Close(0)
		}
	}

	// MQDISC regardless of other errors
	if ci.si.qmgrConnected {
		ci.si.qMgr.Disc()
	}

	traceExit("EndConnection", 0)
}

/*
getMessage returns a message from the replyQ. The "wait"
parameter to the function says whether this should block
for 30 seconds or return immediately if there is no message
available. When working with the command queue, blocking is
required; when getting publications, non-blocking is better.

A 32K buffer was created at the top of this file, and should always
be big enough for what we are expecting.
*/
func getMessage(ci *connectionInfo, wait bool) ([]byte, error) {
	traceEntry("getMessage")

	rc, err := getMessageWithHObj(wait, ci.si.replyQObj)
	traceExitErr("getMessage", 0, err)
	return rc, err
}

func getMessageWithHObj(wait bool, hObj ibmmq.MQObject) ([]byte, error) {
	var err error
	var datalen int

	traceEntry("getMessageWithHObj")
	getmqmd := ibmmq.NewMQMD()

	if convertToAlternateCP {
		getmqmd.CodedCharSetId = ALTERNATE_CCSID
	}

	gmo := ibmmq.NewMQGMO()
	gmo.Options = ibmmq.MQGMO_FAIL_IF_QUIESCING
	gmo.Options |= ibmmq.MQGMO_CONVERT

	if transNeeded {
		gmo.Options |= ibmmq.MQGMO_SYNCPOINT
	} else {
		gmo.Options |= ibmmq.MQGMO_NO_SYNCPOINT
	}

	gmo.MatchOptions = ibmmq.MQMO_NONE

	if wait {
		gmo.Options |= ibmmq.MQGMO_WAIT
		gmo.WaitInterval = 30 * 1000
	}

	datalen, err = hObj.Get(getmqmd, gmo, getBuffer)

	// If the GET failed with a conversion error, try again with the alternate CCSID
	// Backout the first attempt so we retry the same message
	if err != nil && err.(*ibmmq.MQReturn).MQRC == ibmmq.MQRC_NOT_CONVERTED {
		if transNeeded {
			if ibmmq.IsUsableHObj(hObj) {
				hObj.GetHConn().Back()
			}
		}
		convertToAlternateCP = true
		getmqmd.CodedCharSetId = ALTERNATE_CCSID
		datalen, err = hObj.Get(getmqmd, gmo, getBuffer)
	}

	if transNeeded {
		if ibmmq.IsUsableHObj(hObj) {
			hObj.GetHConn().Cmit()
		}
	}
	// By the time we've been through here once successfully, we should know
	// whether we need to continue with the transactional version
	if err == nil {
		transNeeded = false
	}

	traceExitErr("getMessageWithHObj", 0, err)

	return getBuffer[0:datalen], err
}

/*
subscribe to the nominated topic. The previously-opened
replyQ is used for publications; we do not use a managed queue here,
so that everything can be read from one queue. The object handle for the
subscription is returned so we can close it when it's no longer needed.
*/
func subscribe(topic string, pubQObj *ibmmq.MQObject) (*MQTopicDescriptor, error) {
	return subscribeWithOptions(topic, pubQObj, false, false)
}

func subscribeDurable(topic string, pubQObj *ibmmq.MQObject) (*MQTopicDescriptor, error) {
	return subscribeWithOptions(topic, pubQObj, false, true)
}

/*
subscribe to the nominated topic, but ask the queue manager to
allocate the replyQ for us
*/
func subscribeManaged(topic string, pubQObj *ibmmq.MQObject) (*MQTopicDescriptor, error) {
	return subscribeWithOptions(topic, pubQObj, true, false)
}

func subscribeWithOptions(topic string, pubQObj *ibmmq.MQObject, managed bool, durable bool) (*MQTopicDescriptor, error) {
	var err error

	traceEntry("subscribeWithOptions")

	mqtd := new(MQTopicDescriptor)
	ci := getConnection(GetConnectionKey())

	mqsd := ibmmq.NewMQSD()
	mqsd.Options = ibmmq.MQSO_CREATE

	if durable {
		mqsd.Options |= ibmmq.MQSO_DURABLE | ibmmq.MQSO_RESUME
		mqsd.SubName = ci.durableSubPrefix + "_" + topic
	} else {
		mqsd.Options |= ibmmq.MQSO_NON_DURABLE
	}
	mqsd.Options |= ibmmq.MQSO_FAIL_IF_QUIESCING
	if managed {
		mqsd.Options |= ibmmq.MQSO_MANAGED
	}

	mqsd.ObjectString = topic

	hObj, err := ci.si.qMgr.Sub(mqsd, pubQObj)
	if err != nil {
		extraInfo := ""
		mqrc := err.(*ibmmq.MQReturn).MQRC
		switch mqrc {
		case ibmmq.MQRC_HANDLE_NOT_AVAILABLE:
			extraInfo = "You may need to increase the MAXHANDS attribute on the queue manager."
		case ibmmq.MQRC_INVALID_DESTINATION:
			extraInfo = "You cannot use durable subcriptions with temporary dynamic (model) reply queues. Configure system with predefined reply queues"
		}

		e2 := fmt.Errorf("Error subscribing to topic '%s': %v %s", topic, err, extraInfo)
		traceExitErr("subscribeWithOptions", 1, e2)
		return nil, e2
	}

	mqtd.hObj = hObj
	mqtd.durable = durable
	mqtd.topic = topic
	mqtd.managed = managed

	if durable {
		// The subscription can be closed immediately, but still left to
		// collect messages (ie don't use the MQCO_REMOVE_DURABLE option).
		// This can help reduce MAXHANDS impact.
		mqtd.hObj.Close(0)
	}

	traceExitErr("subscribeWithOptions", 0, err)
	return mqtd, err
}

/*
We work with durable subscriptions normally by closing them immediately after setting them up. Leaving the
subscription in place, publications are still going to the designated output queue. But if we want to
delete them, then we need an hObj. So we redo the subscribe in order to get the hObj that allows it to be
deleted. We have to use the same destination queue as is already being used.
*/
func (mqtd *MQTopicDescriptor) unsubscribe() {
	var err error

	traceEntry("unsubscribe")
	topic := mqtd.topic
	logTrace("Removing subscription for %+v ", mqtd)
	if mqtd.durable {
		if ibmmq.IsUsableHObj(mqtd.hObj) {
			mqtd.hObj.Close(ibmmq.MQCO_REMOVE_SUB)
		} else {

			ci := getConnection(GetConnectionKey())

			mqsd := ibmmq.NewMQSD()
			mqsd.Options = ibmmq.MQSO_CREATE | ibmmq.MQSO_RESUME | ibmmq.MQSO_DURABLE | ibmmq.MQSO_FAIL_IF_QUIESCING
			mqsd.SubName = ci.durableSubPrefix + "_" + topic
			mqsd.ObjectString = topic

			subObj, err := ci.si.qMgr.Sub(mqsd, &ci.si.replyQObj)
			if err == nil {
				subObj.Close(ibmmq.MQCO_REMOVE_SUB)
			} else {
				logDebug("Resub failed for %s with %v %+v", topic, err, subObj)
			}
		}
	} else {
		err = mqtd.hObj.Close(ibmmq.MQCO_REMOVE_SUB)
	}

	// The error is not returned, but we do log it in the trace
	traceExitErr("unsubscribe", 0, err)
}

/*
Try to delete any durable subscriptions that exist at startup that correspond to our prefix.
Any errors are ignored here, as this is best-effort. Deletion can only be done one object at a
time (no wildcards). So we have to first issue an inquire to get a list of the subscriptions
associated with our prefix. For each of them, then do the delete. This is a best-effort cleanup
so mostly ignore errors.

We can't use the resubscribe/close technique here because a) we don't know in advance what the
subscription names are and b) we don't know which queue is attached - the collector configuration
might have changed. So we do this cleanup using the PCF commands.
*/
func clearDurableSubscriptions(prefix string, cmdQObj ibmmq.MQObject, replyQObj ibmmq.MQObject, readAhead bool) {
	var err error

	subNameList := make(map[string]string)
	traceEntry("clearDurableSubscriptions")

	clearQ(replyQObj, readAhead)
	putmqmd, pmo, cfh, buf := statusSetCommandHeaders()

	// Can allow all the other fields to default
	cfh.Command = ibmmq.MQCMD_INQUIRE_SUBSCRIPTION

	// Add the parameters one at a time into a buffer
	pcfparm := new(ibmmq.PCFParameter)
	pcfparm.Type = ibmmq.MQCFT_STRING
	pcfparm.Parameter = ibmmq.MQCACF_SUB_NAME
	pcfparm.String = []string{prefix + "_*"} // This is the pattern for all our durable subs
	cfh.ParameterCount++
	buf = append(buf, pcfparm.Bytes()...)

	pcfparm = new(ibmmq.PCFParameter)
	pcfparm.Type = ibmmq.MQCFT_INTEGER
	pcfparm.Parameter = ibmmq.MQIACF_DURABLE_SUBSCRIPTION
	pcfparm.Int64Value = []int64{int64(ibmmq.MQSUB_DURABLE_YES)} // This is the pattern for all our durable subs
	cfh.ParameterCount++
	buf = append(buf, pcfparm.Bytes()...)

	// Once we know the total number of parameters, put the
	// CFH header on the front of the buffer.
	buf = append(cfh.Bytes(), buf...)

	// And now put the command to the queue
	err = cmdQObj.Put(putmqmd, pmo, buf)
	if err != nil {
		traceExitErr("clearDurableSubscriptions", 1, err)
		return
	}

	// Now get the responses - loop until all have been received (one
	// per queue) or we run out of time
	for allReceived := false; !allReceived; {
		cfh, buf, allReceived, err = statusGetReply(putmqmd.MsgId)
		if buf != nil {
			subName, subId := parseInqSubData(cfh, buf)
			if subName != "" {
				subNameList[subName] = subId
			}
		}
	}

	// For each of th returned subscription names, do the delete
	for subName, _ := range subNameList {
		logDebug("About to delete subscription %s", subName)
		clearQ(replyQObj, readAhead)

		putmqmd, pmo, cfh, buf := statusSetCommandHeaders()

		// Can allow all the other fields to default
		cfh.Command = ibmmq.MQCMD_DELETE_SUBSCRIPTION

		// Add the parameters one at a time into a buffer
		pcfparm := new(ibmmq.PCFParameter)
		pcfparm.Type = ibmmq.MQCFT_STRING
		pcfparm.Parameter = ibmmq.MQCACF_SUB_NAME
		pcfparm.String = []string{subName}
		cfh.ParameterCount++
		buf = append(buf, pcfparm.Bytes()...)

		// Once we know the total number of parameters, put the
		// CFH header on the front of the buffer.
		buf = append(cfh.Bytes(), buf...)

		// And now put the command to the queue
		err = cmdQObj.Put(putmqmd, pmo, buf)
		if err != nil {
			traceExitErr("clearDurableSubscriptions", 2, err)
			return
		}

		// Don't really care about the responses, just loop until
		// the operation is complete one way or the other
		for allReceived := false; !allReceived; {
			_, _, allReceived, _ = statusGetReply(putmqmd.MsgId)
		}
	}

	traceExitErr("clearDurableSubscriptions", 0, err)

}

// This tries to issue the "CLEAR QLOCAL" command for fast emptying of a queue during startup, before
// we've actually opened the queue. This will fail if the queue is not a predefined local queue, but
// we can ignore that error.
func clearQPCF(qName string, cmdQObj ibmmq.MQObject, replyQObj ibmmq.MQObject, readAhead bool) {
	var err error

	traceEntryF("clearQPCF", "for queue %s", qName)

	clearQ(replyQObj, readAhead)
	putmqmd, pmo, cfh, buf := statusSetCommandHeaders()

	// Can allow all the other fields to default
	cfh.Command = ibmmq.MQCMD_CLEAR_Q

	// Add the parameters one at a time into a buffer
	pcfparm := new(ibmmq.PCFParameter)
	pcfparm.Type = ibmmq.MQCFT_STRING
	pcfparm.Parameter = ibmmq.MQCA_Q_NAME
	pcfparm.String = []string{qName}
	cfh.ParameterCount++
	buf = append(buf, pcfparm.Bytes()...)

	// Once we know the total number of parameters, put the
	// CFH header on the front of the buffer.
	buf = append(cfh.Bytes(), buf...)

	// And now put the command to the queue
	err = cmdQObj.Put(putmqmd, pmo, buf)
	if err != nil {
		traceExitErr("clearQPCF", 1, err)
		return
	}

	// Don't really care about the responses, just loop until
	// the operation is complete one way or the other
	for allReceived := false; !allReceived; {
		_, _, allReceived, err = statusGetReply(putmqmd.MsgId)
	}

	traceExitErr("clearQPCF", 0, err)

	return
}

// Given a PCF response message, parse it to extract the desired fields
func parseInqSubData(cfh *ibmmq.MQCFH, buf []byte) (string, string) {
	var elem *ibmmq.PCFParameter

	traceEntry("parseInqSubData")

	subName := ""
	subId := ""

	parmAvail := true
	bytesRead := 0
	offset := 0
	datalen := len(buf)
	if cfh == nil || cfh.ParameterCount == 0 {
		traceExit("parseInqSubData", 1)
		return "", ""
	}

	for parmAvail && cfh.CompCode != ibmmq.MQCC_FAILED {
		elem, bytesRead = ibmmq.ReadPCFParameter(buf[offset:])
		offset += bytesRead
		// Have we now reached the end of the message
		if offset >= datalen {
			parmAvail = false
		}

		switch elem.Parameter {
		case ibmmq.MQCACF_SUB_NAME:
			subName = trimToNull(elem.String[0])
		case ibmmq.MQBACF_SUB_ID:
			subId = trimToNull(elem.String[0])
		}
	}

	traceExitF("parseInqSubData", 0, "SubName: %s SubId:%s ", subName, subId)
	return subName, subId
}

/*
Return the current platform - the MQPL_* definition value. It
can be turned into a string if necessary via ibmmq.MQItoString("PL"...)
*/
func GetPlatform() int32 {
	ci := getConnection(GetConnectionKey())
	return ci.si.platform
}

/*
Return the current command level
*/
func GetCommandLevel() int32 {
	ci := getConnection(GetConnectionKey())
	return ci.si.commandLevel
}

func GetResolvedQMgrName() string {
	ci := getConnection(GetConnectionKey())
	return ci.si.resolvedQMgrName
}
