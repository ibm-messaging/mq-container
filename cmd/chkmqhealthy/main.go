/*
© Copyright IBM Corporation 2017, 2019

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
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/ibm-messaging/mq-container/pkg/name"
)

func queueManagerHealthy() (bool, error) {
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
	fmt.Printf("%s", out)
	if !strings.Contains(string(out), "(RUNNING)") && !strings.Contains(string(out), "(RUNNING AS STANDBY)") && !strings.Contains(string(out), "(STARTING)") {
		return false, nil
	}
	return true, nil
}

func main() {
	healthy, err := queueManagerHealthy()
	if err != nil {
		os.Exit(2)
	}
	if !healthy {
		os.Exit(1)
	}
	os.Exit(0)
}
