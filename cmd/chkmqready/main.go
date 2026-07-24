/*
© Copyright IBM Corporation 2017, 2026

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

	"github.com/ibm-messaging/mq-container/internal/metadata"
	"github.com/ibm-messaging/mq-container/internal/ready"
	"github.com/ibm-messaging/mq-container/pkg/name"
)

// return codes for failure modes of the probe
const (
	rcQMReady             = 0
	rcQMNotReady          = 1
	rcSendQMStatusFailure = 2
	rcQMNotReadyIsStandby = 10
	rcQMNotReadyIsReplica = 20
)

func doMain() int {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
	defer cancel()

	// Check if runmqserver has indicated that it's finished configuration
	r, err := ready.Check()
	if !r || err != nil {
		return rcQMNotReady
	}
	name, err := name.GetQueueManagerName()
	if err != nil {
		fmt.Println(err)
		return rcQMNotReady
	}

	// Check if the queue manager has a running listener
	status, err := ready.Status(ctx, name)
	if err != nil {
		return rcQMNotReady
	}

	switch status {
	case ready.StatusActiveQM:
		portOpen, err := checkPort("127.0.0.1:1414")
		if err != nil {
			fmt.Println(err)
		}
		if !portOpen {
			return rcQMNotReady
		}

		// If the metadata sidecar cannot be reached do not fail the probe.
		// Active QueueManagers should always report ready as they may be correctly labelled for routing.
		_ = sendQMStatusIfRequired(ctx, status)

		return rcQMReady
	case ready.StatusRecoveryQM:
		fmt.Printf("Detected queue manager running as recovery leader")

		// If the metadata sidecar cannot be reached do not fail the probe.
		// Recovery Leader QueueManagers should always report ready as they may be correctly labelled for routing.
		_ = sendQMStatusIfRequired(ctx, status)

		return rcQMReady
	case ready.StatusStandbyQM:
		fmt.Printf("Detected queue manager running in standby mode")

		err = sendQMStatusIfRequired(ctx, status)
		if err != nil {
			return rcSendQMStatusFailure
		}

		if readyReplicasEnabled() {
			return rcQMReady
		}
		return rcQMNotReadyIsStandby
	case ready.StatusReplicaQM:
		fmt.Printf("Detected queue manager running in replica mode")

		err = sendQMStatusIfRequired(ctx, status)
		if err != nil {
			return rcSendQMStatusFailure
		}

		if readyReplicasEnabled() {
			return rcQMReady
		}
		return rcQMNotReadyIsReplica
	default:
		return rcQMNotReady
	}
}

func sendQMStatusIfRequired(ctx context.Context, status ready.QMStatus) error {
	if !readyReplicasEnabled() {
		return nil
	}
	return metadata.SendQMStatus(ctx, status)
}

func readyReplicasEnabled() bool {
	return os.Getenv("MQ_ENABLE_READY_REPLICAS") == "true"
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
