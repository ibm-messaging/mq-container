/*
© Copyright IBM Corporation 2021, 2022

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
	"strings"

	"github.com/ibm-messaging/mq-container/pkg/name"
)

func queueManagerStarted(ctx context.Context) (bool, error) {
	name, err := name.GetQueueManagerName()
	if err != nil {
		return false, err
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
	if !strings.Contains(string(out), "(RUNNING)") && !strings.Contains(string(out), "(RUNNING AS STANDBY)") && !strings.Contains(string(out), "(STARTING)") && !strings.Contains(string(out), "(REPLICA)") {
		return false, nil
	}
	return true, nil
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
