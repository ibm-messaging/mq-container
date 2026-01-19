/*
© Copyright IBM Corporation 2017, 2024

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// chkmqready checks that MQ is ready for work, by checking if the MQ listener port is available
package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"

	"github.com/ibm-messaging/mq-container/internal/ready"
	"github.com/ibm-messaging/mq-container/pkg/name"
	"github.com/ibm-messaging/mq-container/pkg/probesocket"
)

func doMain() int {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
	defer cancel()

	// Check if runmqserver has indicated that it's finished configuration
	r, err := ready.Check()
	if !r || err != nil {
		if err != nil {
			probesocket.SendProbeLogs(probesocket.ERROR, probesocket.ReadinessProbeSockPath, fmt.Sprintf("Readiness Probe Failed: %v", err))
		} else {
			probesocket.SendProbeLogs(probesocket.ERROR, probesocket.ReadinessProbeSockPath, "Readiness Probe Failed: queue manager configuration not completed")
		}
		return 1
	}
	name, err := name.GetQueueManagerName()
	if err != nil {
		probesocket.SendProbeLogs(probesocket.ERROR, probesocket.ReadinessProbeSockPath, fmt.Sprintf("Readiness Probe Failed: %v", err))
		fmt.Println(err)
		return 1
	}

	// Check if the queue manager has a running listener
	status, err := ready.Status(ctx, name)
	if err != nil {
		probesocket.SendProbeLogs(probesocket.ERROR, probesocket.ReadinessProbeSockPath, fmt.Sprintf("Readiness Probe Failed: %v", err))
		return 1
	}
	switch status {
	case ready.StatusActiveQM:
		portOpen, err := checkPort("127.0.0.1:1414")

		if err != nil {
			fmt.Println(err)
			if portOpen {
				probesocket.SendProbeLogs(probesocket.INFO, probesocket.ReadinessProbeSockPath, fmt.Sprintf("Readiness Probe: error closing readiness probe connection: %v", err))
				return 0
			}
		}
		if !portOpen {
			probesocket.SendProbeLogs(probesocket.ERROR, probesocket.ReadinessProbeSockPath, fmt.Sprintf("Readiness Probe Failed: error connecting to port 1414: %v", err))
			return 1
		}

		probesocket.SendProbeLogs(probesocket.INFO, probesocket.ReadinessProbeSockPath, "Readiness Probe Passed: detected queue manager running as active instance")

		return 0
	case ready.StatusRecoveryQM:
		probesocket.SendProbeLogs(probesocket.INFO, probesocket.ReadinessProbeSockPath, "Readiness Probe: detected queue manager running as recovery leader")
		fmt.Printf("Detected queue manager running as recovery leader")
		return 0
	case ready.StatusStandbyQM:
		probesocket.SendProbeLogs(probesocket.INFO, probesocket.ReadinessProbeSockPath, "Readiness Probe: detected queue manager running in standby mode")
		fmt.Printf("Detected queue manager running in standby mode")
		return 10
	case ready.StatusReplicaQM:
		probesocket.SendProbeLogs(probesocket.INFO, probesocket.ReadinessProbeSockPath, "Readiness Probe: detected queue manager running in replica mode")
		fmt.Printf("Detected queue manager running in replica mode")
		return 20
	default:
		probesocket.SendProbeLogs(probesocket.ERROR, probesocket.ReadinessProbeSockPath, "Readiness Probe Failed: detected queue manager running with unknown status")
		return 1
	}
}

func main() {
	os.Exit(doMain())
}

func checkPort(address string) (portOpen bool, err error) {
	var conn net.Conn
	conn, err = net.Dial("tcp", address)
	if err != nil {
		return
	}
	portOpen = true
	err = conn.Close()
	return
}
