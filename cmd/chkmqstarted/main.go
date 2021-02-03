/*
© Copyright IBM Corporation 2021

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
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/ibm-messaging/mq-container/pkg/name"
)

func queueManagerStarted() (bool, error) {
	name, err := name.GetQueueManagerName()
	if err != nil {
		return false, err
	}
	// Specify the queue manager name, just in case someone's created a second queue manager
	// #nosec G204
	cmd := exec.Command("dspmq", "-n", "-m", name)
	// Run the command and wait for completion
	out, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println(err)
		return false, err
	}
	if !strings.Contains(string(out), "(RUNNING)") && !strings.Contains(string(out), "(RUNNING AS STANDBY)") && !strings.Contains(string(out), "(STARTING)") && !strings.Contains(string(out), "(REPLICA)") {
		return false, nil
	}
	if os.Getenv("MQ_NATIVE_HA") == "true" {
		// Specify the queue manager name, just in case someone's created a second queue manager
		// #nosec G204
		cmd = exec.Command("dspmq", "-n", "-o", "nativeha", "-m", name)
		// Run the command and wait for completion
		out, err = cmd.CombinedOutput()
		if err != nil {
			fmt.Println(err)
			return false, err
		}
		if !strings.Contains(string(out), "INSYNC(YES)") {
			return false, nil
		}
	}
	return true, nil
}

func main() {
	started, err := queueManagerStarted()
	if err != nil {
		os.Exit(2)
	}
	if !started {
		os.Exit(1)
	}
	os.Exit(0)
}
