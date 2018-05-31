/*
This is a short sample to show how to connect to a remote
queue manager in a Go program without requiring external
client configuration such as a CCDT. Only the basic
parameters are needed here - channel name and connection information -
along with the queue manager name.

For example, run as
   clientconn QMGR1 "SYSTEM.DEF.SVRCONN" "myhost.example.com(1414)"

If the MQSAMP_USER_ID environment variable is set, then a userid/password
flow is also made to authenticate to the queue manager.

There is no attempt in this sample to configure advanced security features
such TLS.

If an error occurs, the error is reported.
*/
package main

/*
  Copyright (c) IBM Corporation 2017

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
	"bufio"
	"fmt"
	"os"
	"time"

	"github.com/ibm-messaging/mq-golang/ibmmq"
)

func main() {
	var qMgrName string
	var err error
	var qMgr ibmmq.MQQueueManager
	var rc int

	if len(os.Args) != 4 {
		fmt.Println("clientconn <qmgrname> <channelname> <conname>")
		fmt.Println("")
		fmt.Println("For example")
		fmt.Println("  clientconn QMGR1 \"SYSTEM.DEF.SVRCONN\" \"myhost.example.com(1414)\"")
		fmt.Println("All parameters are required.")
		os.Exit(1)
	}

	// Which queue manager do we want to connect to
	qMgrName = os.Args[1]

	// Allocate the MQCNO and MQCD structures needed for the
	// MQCONNX call.
	cno := ibmmq.NewMQCNO()
	cd := ibmmq.NewMQCD()

	// Fill in the required fields in the
	// MQCD channel definition structure
	cd.ChannelName = os.Args[2]
	cd.ConnectionName = os.Args[3]

	// Reference the CD structure from the CNO
	// and indicate that we want to use the client
	// connection method.
	cno.ClientConn = cd
	cno.Options = ibmmq.MQCNO_CLIENT_BINDING

	// Also fill in the userid and password if the MQSAMP_USER_ID
	// environment variable is set. This is the same as the C
	// sample programs such as amqsput.
	userId := os.Getenv("MQSAMP_USER_ID")
	if userId != "" {
		scanner := bufio.NewScanner(os.Stdin)
		csp := ibmmq.NewMQCSP()
		csp.AuthenticationType = ibmmq.MQCSP_AUTH_USER_ID_AND_PWD
		csp.UserId = userId

		fmt.Printf("Enter password for qmgr %s: \n", qMgrName)
		scanner.Scan()
		csp.Password = scanner.Text()

		// And make the CNO refer to the CSP structure
		cno.SecurityParms = csp
	}

	// And connect. Wait a short time before
	// disconnecting.
	qMgr, err = ibmmq.Connx(qMgrName, cno)
	if err == nil {
		fmt.Printf("Connection to %s succeeded.\n", qMgrName)
		d, _ := time.ParseDuration("5s")
		time.Sleep(d)
		qMgr.Disc()
		rc = 0
	} else {
		fmt.Printf("Connection to %s failed.\n", qMgrName)
		fmt.Println(err)
		rc = int(err.(*ibmmq.MQReturn).MQCC)
	}

	fmt.Println("Done.")
	os.Exit(rc)

}
