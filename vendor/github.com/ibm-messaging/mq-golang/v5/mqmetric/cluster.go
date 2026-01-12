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
Functions in this file use the DISPLAY CLUSQMGR command to extract metrics
about MQ clusters
*/

import (
	"github.com/ibm-messaging/mq-golang/v5/ibmmq"
)

const (
	ATTR_CLUSTER_NAME    = "name"
	ATTR_CLUSTER_QMTYPE  = "qmtype"  // "repos" or "normal" = "full" or "partial"
	ATTR_CLUSTER_STATUS  = "status"  // clussdr status
	ATTR_CLUSTER_SUSPEND = "suspend" // yes/no
	// do we want a channel status squash?
)

/*
Unlike the statistics produced via a topic, there is no discovery
of the attributes available in object STATUS queries. There is also
no discovery of descriptions for them. So this function hardcodes the
attributes we are going to look for and gives the associated descriptive
text. The elements can be expanded later; just trying to give a starting point
for now.
*/
func ClusterInitAttributes() {
	traceEntry("ClusInitAttributes")
	ci := getConnection(GetConnectionKey())
	os := &ci.objectStatus[OT_CLUSTER]
	st := GetObjectStatus(GetConnectionKey(), OT_CLUSTER)

	if os.init {
		traceExit("ClusterInitAttributes", 1)
		return
	}
	st.Attributes = make(map[string]*StatusAttribute)

	attr := ATTR_CLUSTER_NAME //MQCA_CLUSTER_NAME
	st.Attributes[attr] = newPseudoStatusAttribute(attr, "Cluster Name")
	attr = ATTR_CLUSTER_STATUS
	st.Attributes[attr] = newStatusAttribute(attr, "Cluster Status", ibmmq.MQIACH_CHANNEL_STATUS)
	attr = ATTR_CLUSTER_SUSPEND
	st.Attributes[attr] = newStatusAttribute(attr, "Cluster Suspend", ibmmq.MQIACF_SUSPEND)
	attr = ATTR_CLUSTER_QMTYPE
	st.Attributes[attr] = newStatusAttribute(attr, "Queue Manager Type", ibmmq.MQIACF_Q_MGR_TYPE)
	os.init = true
	traceExit("ClusterInitAttributes", 0)
}

func CollectClusterStatus() error {
	var err error
	traceEntry("CollectClusterStatus")

	st := GetObjectStatus(GetConnectionKey(), OT_CLUSTER)
	ClusterInitAttributes()

	// Empty any collected values
	for k := range st.Attributes {
		st.Attributes[k].Values = make(map[string]*StatusValue)
	}

	err = collectClusterStatus()

	traceExitErr("CollectClusterStatus", 0, err)

	return err
}

// Issue the INQUIRE_CLUSQMGR command for this qmgr
// Collect the responses and build up the metrics
func collectClusterStatus() error {
	var err error

	traceEntryF("collectClusterStatus", "")
	ci := getConnection(GetConnectionKey())

	statusClearReplyQ()

	putmqmd, pmo, cfh, buf := statusSetCommandHeaders()

	// Can allow all the other fields to default
	cfh.Command = ibmmq.MQCMD_INQUIRE_CLUSTER_Q_MGR

	// Add the parameters one at a time into a buffer
	pcfparm := new(ibmmq.PCFParameter)
	pcfparm.Type = ibmmq.MQCFT_STRING
	pcfparm.Parameter = ibmmq.MQCA_CLUSTER_Q_MGR_NAME
	pcfparm.String = []string{ci.si.resolvedQMgrName}
	cfh.ParameterCount++
	buf = append(buf, pcfparm.Bytes()...)

	// Once we know the total number of parameters, put the
	// CFH header on the front of the buffer.
	buf = append(cfh.Bytes(), buf...)

	// And now put the command to the queue
	err = ci.si.cmdQObj.Put(putmqmd, pmo, buf)
	if err != nil {
		traceExitErr("collectClusterStatus", 1, err)

		return err
	}

	// Now get the responses - loop until all have been received (one
	// per queue) or we run out of time
	for allReceived := false; !allReceived; {
		cfh, buf, allReceived, err = statusGetReply(putmqmd.MsgId)
		if buf != nil {
			parseClusterData(cfh, buf)
		}
	}

	traceExitErr("collectClusterStatus", 0, err)

	return err
}

// Given a PCF response message, parse it to extract the desired statistics
func parseClusterData(cfh *ibmmq.MQCFH, buf []byte) string {
	var elem *ibmmq.PCFParameter

	traceEntry("parseClusterData")

	st := GetObjectStatus(GetConnectionKey(), OT_CLUSTER)
	ClusterName := ""

	parmAvail := true
	bytesRead := 0
	offset := 0
	datalen := len(buf)
	if cfh == nil || cfh.ParameterCount == 0 {
		traceExit("parseClusterData", 1)
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
		case ibmmq.MQCA_CLUSTER_NAME:
			ClusterName = trimToNull(elem.String[0])
		}
	}

	// Create a unique key for this instance
	key := ClusterName

	st.Attributes[ATTR_CLUSTER_NAME].Values[key] = newStatusValueString(ClusterName)

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

		logTrace("parseClusterData - looking at elem %+v", elem)
		if !statusGetIntAttributes(GetObjectStatus(GetConnectionKey(), OT_CLUSTER), elem, key) {
			// There's not actually any additional attributes we care about for now
			switch elem.Parameter {
			//case ibmmq.MQCA_CLUSTER_NAME:
			// ClusterName = trimToNull(elem.String[0])
			}
		}
	}
	traceExitF("parseClusterData", 0, "Key : %s", key)

	return key
}

// Return a standardised value. If the attribute indicates that something
// special has to be done, then do that. Otherwise just make sure it's a non-negative
// value of the correct datatype
func ClusterNormalise(attr *StatusAttribute, v int64) float64 {
	return statusNormalise(attr, v)
}
