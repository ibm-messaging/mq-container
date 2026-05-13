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

// chkmqhealthy checks that MQ is healthy, by checking the output of the "dspmq" command
package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"time"

	"github.com/ibm-messaging/mq-container/internal/probes"
	"github.com/ibm-messaging/mq-container/pkg/name"
)

func queueManagerHealthy(ctx context.Context) (bool, string, error) {

	name, err := name.GetQueueManagerName()
	if err != nil {
		return false, "", err
	}

	// Specify the queue manager name, just in case someone's created a second queue manager
	// #nosec G204
	cmd := exec.CommandContext(ctx, "dspmq", "-n", "-m", name)
	// Run the command and wait for completion
	out, err := cmd.CombinedOutput()
	fmt.Printf("%s", out)
	if err != nil {
		fmt.Println(err)
		return false, "", err
	}

	readyStrings := []string{
		"(RUNNING)",
		"(RUNNING AS STANDBY)",
		"(RECOVERY GROUP LEADER)",
		"(STARTING)",
		"(REPLICA)",
	}
	for _, checkString := range readyStrings {
		if strings.Contains(string(out), checkString) {
			return true, string(out), nil
		}
	}

	return false, string(out), nil
}

func doMain() int {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
	defer cancel()

	isProbeLoggingEnabled := probes.IsProbeLoggingEnabled(probes.LivenessProbe)
	var probeLoggingExecution *probes.ProbeLoggingExecution

	if isProbeLoggingEnabled {
		probeStartTime := time.Now()
		probeLoggingExecution = probes.InitializeProbeLoggingExecution(probes.LivenessProbe, probeStartTime, probes.ProbeIncomplete)
	}

	healthy, out, err := queueManagerHealthy(ctx)

	if isProbeLoggingEnabled && probeLoggingExecution != nil {
		probeEndTime := time.Now()
		probeLoggingExecution.LogLivenessProbeMessage(probeEndTime, healthy, out, err)
	}

	if err != nil {
		return 2
	}

	if !healthy {
		return 1
	}

	return 0
}

func main() {
	os.Exit(doMain())
}
