/*
© Copyright IBM Corporation 2021, 2024

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

// chkmqstarted checks that MQ has successfully started, by checking the output of the "dspmq" command
package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"strings"
	"time"

	"github.com/ibm-messaging/mq-container/internal/ready"
	"github.com/ibm-messaging/mq-container/pkg/name"
)

func queueManagerStarted(ctx context.Context) (bool, error) {

	name, err := name.GetQueueManagerName()
	if err != nil {
		return false, err
	}

	readyStrings := []string{
		"(RUNNING)",
		"(RUNNING AS STANDBY)",
		"(RECOVERY GROUP LEADER)",
		"(STARTING)",
		"(REPLICA)",
	}

	// For Native-HA only, check if the queue manager instance is in-sync with one or more replicas
	// - If not in-sync within the expected time period, revert to checking on queue manager 'ready' status
	// - This ensures we do not block indefinitely for breaking changes (i.e. protocol changes)
	if os.Getenv("MQ_NATIVE_HA") == "true" {

		// Check if the Native-HA queue manager instance is currently in-sync
		isReadyToSync, isInSync, err := isInSyncWithReplicas(ctx, name, readyStrings)
		if err != nil {
			return false, err
		} else if isInSync {
			return true, nil
		}

		// Check if the Native-HA queue manager instance is ready-to-sync
		// - A successful queue manager 'ready' status indicates that we are ready-to-sync
		if !isReadyToSync {
			return false, nil
		}
		err = ready.SetReadyToSync()
		if err != nil {
			return false, err
		}

		// Check if the time period for checking in-sync has now expired
		// - We have already confirmed a successful queue manager 'ready' status
		// - Therefore the expiration of the in-sync time period will result in success
		expired, err := hasInSyncTimePeriodExpired()
		if err != nil {
			return false, err
		} else if expired {
			return true, nil
		}

		return false, nil
	}

	// Specify the queue manager name, just in case someone's created a second queue manager
	// #nosec G204
	cmd := exec.CommandContext(ctx, "dspmq", "-n", "-m", name)
	// Run the command and wait for completion
	out, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println(err)
		return false, err
	}

	for _, checkString := range readyStrings {
		if strings.Contains(string(out), checkString) {
			return true, nil
		}
	}

	return false, nil
}

// isInSyncWithReplicas returns the in-sync status for a Native-HA queue manager instance
func isInSyncWithReplicas(ctx context.Context, name string, readyStrings []string) (bool, bool, error) {

	cmd := exec.CommandContext(ctx, "dspmq", "-n", "-o", "nativeha", "-m", name)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return false, false, err
	} else if strings.Contains(string(out), "INSYNC(YES)") {
		return true, true, nil
	}

	for _, checkString := range readyStrings {
		if strings.Contains(string(out), checkString) {
			return true, false, nil
		}
	}

	return false, false, nil
}

// hasInSyncTimePeriodExpired returns true if a Native-HA queue manager instance is not in-sync within the expected time period, otherwise false
func hasInSyncTimePeriodExpired() (bool, error) {

	// Default timeout 5 seconds
	var timeout int64 = 5
	var err error

	// Check if a timeout override has been set
	customTimeout := os.Getenv("MQ_NATIVE_HA_IN_SYNC_TIMEOUT")
	if customTimeout != "" {
		timeout, err = strconv.ParseInt(customTimeout, 10, 64)
		if err != nil {
			return false, err
		}
	}

	isReadyToSync, readyToSyncStartTime, err := ready.GetReadyToSyncStartTime()
	if err != nil {
		return false, err
	}
	if isReadyToSync && time.Now().Unix()-readyToSyncStartTime.Unix() >= timeout {
		return true, nil
	}

	return false, nil
}

func doMain() int {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
	defer cancel()

	started, err := queueManagerStarted(ctx)
	if err != nil {
		return 2
	}
	if !started {
		return 1
	}
	return 0
}

func main() {
	os.Exit(doMain())
}
