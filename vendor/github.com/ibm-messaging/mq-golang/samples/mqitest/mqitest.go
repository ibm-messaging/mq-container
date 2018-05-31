/*
This is a short sample to show how to call IBM MQ from
a Go program.

The flow is to connect to a queue manager,
open the queue named on the command line,
put a message and then get it back.
The queue is closed.

The program then subscribes to the topic corresponding
to collecting activity trace for itself - this requires MQ V9.

Finally, it closes the subscription and target queue, and
disconnects.

If an error occurs at any stage, the error is reported and
subsequent steps skipped.
*/
package main

/*
  Copyright (c) IBM Corporation 2016

  Licensed under the Apache License, Version 2.0 (the "License");
  you may not use this file except in compliance with the License.
  You may obtain a copy of the License at

  http://www.apache.org/licenses/LICENSE-2.0

  Unless required by applicable law or agreed to in writing, software
  distributed under the License is distributed on an "AS IS" BASIS,
  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
  See the License for the specific

   Contributors:
     Mark Taylor - Initial Contribution
*/

import (
	"fmt"
	"os"
	"strings"

	"github.com/ibm-messaging/mq-golang/ibmmq"
)

func main() {

	var openOptions int32

	var qMgrObject ibmmq.MQObject
	var qObject ibmmq.MQObject
	var managedQObject ibmmq.MQObject
	var subObject ibmmq.MQObject

	var qMgrName string

	if len(os.Args) != 3 {
		fmt.Println("mqitest <qname> <qmgrname>")
		fmt.Println("  Both parms required")
		os.Exit(1)
	}

	qMgrName = os.Args[2]
	connected := false
	qMgr, err := ibmmq.Conn(qMgrName)
	if err != nil {
		fmt.Println(err)
	} else {
		connected = true
		fmt.Println("Connected to queue manager ", qMgrName)
	}

	// MQOPEN of the queue named on command line
	if err == nil {
		mqod := ibmmq.NewMQOD()

		openOptions = ibmmq.MQOO_OUTPUT + ibmmq.MQOO_FAIL_IF_QUIESCING
		openOptions |= ibmmq.MQOO_INPUT_AS_Q_DEF

		mqod.ObjectType = ibmmq.MQOT_Q
		mqod.ObjectName = os.Args[1]

		qObject, err = qMgr.Open(mqod, openOptions)
		if err != nil {
			fmt.Println(err)
		} else {
			fmt.Println("Opened queue", qObject.Name)
		}
	}

	// MQPUT a message
	//   Create the standard MQI structures MQMD, MQPMO and
	//   set the values.
	// The message is always sent as bytes, so has to be converted
	// before the MQPUT.
	if err == nil {
		putmqmd := ibmmq.NewMQMD()
		pmo := ibmmq.NewMQPMO()

		pmo.Options = ibmmq.MQPMO_SYNCPOINT | ibmmq.MQPMO_NEW_MSG_ID | ibmmq.MQPMO_NEW_CORREL_ID

		putmqmd.Format = "MQSTR"
		msgData := "Hello from Go"
		buffer := []byte(msgData)

		err = qObject.Put(putmqmd, pmo, buffer)

		if err != nil {
			fmt.Println(err)
		} else {
			fmt.Println("Put message to", qObject.Name)
		}
	}

	// The message was put in syncpoint so it needs
	// to be committed.
	if err == nil {
		err = qMgr.Cmit()
		if err != nil {
			fmt.Println(err)
		}
	}

	// MQGET all messages on the queue. Wait 3 seconds for any more
	// to arrive.
	if err == nil {
		msgAvail := true

		for msgAvail == true {
			var datalen int

			getmqmd := ibmmq.NewMQMD()
			gmo := ibmmq.NewMQGMO()
			gmo.Options = ibmmq.MQGMO_NO_SYNCPOINT | ibmmq.MQGMO_FAIL_IF_QUIESCING
			gmo.Options |= ibmmq.MQGMO_WAIT
			gmo.WaitInterval = 3000
			buffer := make([]byte, 32768)

			datalen, err = qObject.Get(getmqmd, gmo, buffer)

			if err != nil {
				msgAvail = false
				fmt.Println(err)
				mqret := err.(*ibmmq.MQReturn)
				if mqret.MQRC == ibmmq.MQRC_NO_MSG_AVAILABLE {
					// not a real error so reset err
					err = nil
				}
			} else {
				fmt.Printf("Got message of length %d: ", datalen)
				fmt.Println(strings.TrimSpace(string(buffer[:datalen])))
			}
		}
	}

	// MQCLOSE the queue
	if err == nil {
		err = qObject.Close(0)
		if err != nil {
			fmt.Println(err)
		} else {
			fmt.Println("Closed queue")
		}
	}

	// This section demonstrates subscribing to a topic
	// where the topic string is set to collect activity trace
	// from this program - it needs MQ V9 for publications to
	// automatically be generated on this topic.
	if err == nil {
		mqsd := ibmmq.NewMQSD()
		mqsd.Options = ibmmq.MQSO_CREATE
		mqsd.Options |= ibmmq.MQSO_NON_DURABLE
		mqsd.Options |= ibmmq.MQSO_FAIL_IF_QUIESCING
		mqsd.Options |= ibmmq.MQSO_MANAGED
		mqsd.ObjectString = "$SYS/MQ/INFO/QMGR/" + qMgrName + "/ActivityTrace/ApplName/mqitest"

		subObject, err = qMgr.Sub(mqsd, &managedQObject)

		if err != nil {
			fmt.Println(err)
		} else {
			fmt.Println("Subscribed to topic ", mqsd.ObjectString)
		}
	}

	// Loop on the managed queue created by the MQSUB call until there
	// are no more messages. Because these are going to be PCF-format
	// events, they cannot be simply printed so here I'm just
	// printing the format of the message to show that something has
	// been retrieved.
	if err == nil {
		msgAvail := true

		for msgAvail == true {
			var datalen int

			getmqmd := ibmmq.NewMQMD()
			gmo := ibmmq.NewMQGMO()
			gmo.Options = ibmmq.MQGMO_NO_SYNCPOINT | ibmmq.MQGMO_FAIL_IF_QUIESCING
			gmo.Options |= ibmmq.MQGMO_WAIT
			gmo.WaitInterval = 3000
			buffer := make([]byte, 32768)

			datalen, err = managedQObject.Get(getmqmd, gmo, buffer)

			if err != nil {
				msgAvail = false
				fmt.Println(err)
				mqret := err.(*ibmmq.MQReturn)
				if mqret.MQRC == ibmmq.MQRC_NO_MSG_AVAILABLE {
					// not a real error so reset err, but
					// end retrieval loop
					err = nil
				}
			} else {
				fmt.Printf("Got message of length %d. Format = %s\n", datalen, getmqmd.Format)
			}
		}
	}

	// MQCLOSE the subscription, ignoring errors.
	if err == nil {
		subObject.Close(0)
	}

	if err == nil {
		mqod := ibmmq.NewMQOD()
		openOptions = ibmmq.MQOO_INQUIRE + ibmmq.MQOO_FAIL_IF_QUIESCING

		mqod.ObjectType = ibmmq.MQOT_Q_MGR
		mqod.ObjectName = ""

		qMgrObject, err = qMgr.Open(mqod, openOptions)

		if err != nil {
			fmt.Println(err)
		} else {
			fmt.Printf("Opened QMgr for MQINQ\n")
		}
	}

	if err == nil {
		selectors := []int32{ibmmq.MQCA_Q_MGR_NAME,
			ibmmq.MQCA_DEAD_LETTER_Q_NAME,
			ibmmq.MQIA_MSG_MARK_BROWSE_INTERVAL}

		intAttrs, charAttrs, err := qMgrObject.Inq(selectors, 2, 160)

		if err != nil {
			fmt.Println(err)
		} else {
			returnedName := string(charAttrs[0:48])
			fmt.Printf("MQINQ returned +%v %s \n",
				intAttrs, string(charAttrs))
			fmt.Printf("               '%s'\n", returnedName)
		}

	}

	// MQDISC regardless of other errors
	if connected {
		err = qMgr.Disc()
		fmt.Println("Disconnected from queue manager ", qMgrName)
	}

	if err == nil {
		os.Exit(0)
	} else {
		mqret := err.(*ibmmq.MQReturn)
		os.Exit((int)(mqret.MQCC))
	}

}
