package mqmetric

/*
  Copyright (c) IBM Corporation 2016, 2024

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
	"github.com/ibm-messaging/mq-golang/v5/ibmmq"
)

type sessionInfo struct {
	qMgr            ibmmq.MQQueueManager
	cmdQObj         ibmmq.MQObject
	replyQObj       ibmmq.MQObject
	qMgrObject      ibmmq.MQObject
	replyQBaseName  string
	replyQ2BaseName string
	statusReplyQObj ibmmq.MQObject
	statusReplyBuf  []byte

	platform         int32
	commandLevel     int32
	version          string // includes fixpack levels eg "09040001"
	maxHandles       int32
	resolvedQMgrName string

	qmgrConnected bool
	queuesOpened  bool
	subsOpened    bool
}

type connectionInfo struct {
	si sessionInfo

	tzOffsetSecs         float64
	usePublications      bool
	useStatus            bool
	useResetQStats       bool
	useDepthFromStatus   bool
	showInactiveChannels bool
	hideSvrConnJobname   bool
	hideAMQPClientId     bool

	durableSubPrefix string

	// Only issue the warning about a '/' in an object name once.
	globalSlashWarning bool
	localSlashWarning  bool

	discoveryDone    bool
	publicationCount int

	waitInterval int

	objectStatus     [OT_LAST_USED + 1]objectStatus
	publishedMetrics AllMetrics
}

type objectStatus struct {
	init       bool
	objectSeen map[string]bool
	s          *StatusSet
}

// These object types are the same where possible as the MQI MQOT definitions
// but there are some unique types here so that correspondence is not
// completely identical.
const (
	OT_Q             = 1
	OT_NAMELIST      = 2
	OT_PROCESS       = 3
	OT_STORAGE_CLASS = 4
	OT_Q_MGR         = 5
	OT_CHANNEL       = 6
	OT_AUTH_INFO     = 7
	OT_TOPIC         = 8
	OT_COMM_INFO     = 9
	OT_CF_STRUC      = 10
	OT_LISTENER      = 11
	OT_SERVICE       = 12
	OT_APP           = 13
	OT_PUB           = 14
	OT_SUB           = 15
	OT_NHA           = 16
	OT_BP            = 17
	OT_PS            = 18
	OT_CLUSTER       = 19
	OT_CHANNEL_AMQP  = 20
	OT_LAST_USED     = OT_CHANNEL_AMQP
)

var connectionMap = make(map[string]*connectionInfo)
var connectionKey string

const DUMMY_STRING = "-" // To provide a non-empty value for certain fields
const DEFAULT_CONNECTION_KEY = "@defaultConnection"

// This are used externally so we need to maintain them as public exports until
// there's a major version change. At which point we will move them to fields of
// the objectStatus structure, retrievable by a getXXX() call instead of as public
// variables. The mq-metric-samples exporters will then need to change to match.
var (
	Metrics            AllMetrics
	QueueManagerStatus StatusSet
	ChannelStatus      StatusSet
	ChannelAMQPStatus  StatusSet
	QueueStatus        StatusSet
	TopicStatus        StatusSet
	SubStatus          StatusSet
	UsagePsStatus      StatusSet
	UsageBpStatus      StatusSet
	ClusterStatus      StatusSet
)

func newConnectionInfo(key string) *connectionInfo {

	traceEntry("newConnectionInfo")

	ci := new(connectionInfo)
	ci.si.qmgrConnected = false
	ci.si.queuesOpened = false
	ci.si.subsOpened = false

	ci.usePublications = true
	ci.useStatus = false
	ci.useResetQStats = false
	ci.showInactiveChannels = false
	ci.hideSvrConnJobname = false
	ci.hideAMQPClientId = false

	ci.globalSlashWarning = false
	ci.localSlashWarning = false
	ci.discoveryDone = false
	ci.publicationCount = 0

	for i := 1; i <= OT_LAST_USED; i++ {
		ci.objectStatus[i].init = false
		ci.objectStatus[i].s = new(StatusSet)
	}

	if key == "" {
		key = DEFAULT_CONNECTION_KEY
	}
	connectionMap[key] = ci

	traceExitF("newConnectionInfo", 0, "Key: %s", key)

	return ci
}

// Initialise this package
func initConnection(key string) {
	traceEntryF("initConnection", "key: %s", key)

	newConnectionInfo(key)

	traceExit("initConnection", 0)

}

// This will be the preferred interface in future
// to get at the values, at which point it will
// change to not use the global public variables
func GetObjectStatus(key string, objectType int) *StatusSet {
	if key == "" || key == DEFAULT_CONNECTION_KEY {

		switch objectType {
		case OT_CHANNEL:
			return &ChannelStatus
		case OT_CHANNEL_AMQP:
			return &ChannelAMQPStatus
		case OT_Q_MGR:
			return &QueueManagerStatus
		case OT_Q:
			return &QueueStatus
		case OT_TOPIC:
			return &TopicStatus
		case OT_SUB:
			return &SubStatus
		case OT_BP:
			return &UsagePsStatus
		case OT_PS:
			return &UsageBpStatus
		case OT_CLUSTER:
			return &ClusterStatus
		default:
			return nil
		}
	} else {
		ci := getConnection(key)
		return ci.objectStatus[objectType].s
	}
}

func GetPublishedMetrics(key string) *AllMetrics {
	if key == "" || key == DEFAULT_CONNECTION_KEY {
		return &Metrics
	} else {
		ci := getConnection(key)
		return &ci.publishedMetrics
	}
}

func SetConnectionKey(key string) {
	connectionKey = key
}

func GetConnectionKey() string {
	return connectionKey
}

func getConnection(key string) *connectionInfo {
	if key == "" {
		return connectionMap[DEFAULT_CONNECTION_KEY]
	} else {
		return connectionMap[key]
	}
}
