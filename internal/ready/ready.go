/*
© Copyright IBM Corporation 2018, 2023

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

// Package ready contains code to provide a ready signal mechanism between processes
package ready

import (
	"os"
	"strings"

	"github.com/ibm-messaging/mq-container/internal/command"
)

const fileName string = "/run/runmqserver/ready"

func fileExists() (bool, error) {
	_, err := os.Stat(fileName)
	if err != nil {
		if !os.IsNotExist(err) {
			return false, err
		}
		return false, nil
	}
	return true, nil
}

// Clear ensures that any readiness state is cleared
func Clear() error {
	exist, err := fileExists()
	if err != nil {
		return err
	}
	if exist {
		return os.Remove(fileName)
	}
	return nil
}

// Set lets any subsequent calls to `CheckReady` know that the queue
// manager has finished its configuration step
func Set() error {
	// #nosec G306 - this gives permissions to owner/s group only.
	return os.WriteFile(fileName, []byte("1"), 0770)
}

// Check checks whether or not the queue manager has finished its
// configuration steps
func Check() (bool, error) {
	exists, err := fileExists()
	if err != nil {
		return false, err
	}
	return exists, nil
}

// IsRunningAsActiveQM returns true if the queue manager is running in active mode
func IsRunningAsActiveQM(name string) (bool, error) {
	return isRunningQM(name, "(RUNNING)")
}

// IsRunningAsStandbyQM returns true if the queue manager is running in standby mode
func IsRunningAsStandbyQM(name string) (bool, error) {
	return isRunningQM(name, "(RUNNING AS STANDBY)")
}

// IsRunningAsReplicaQM returns true if the queue manager is running in replica mode
func IsRunningAsReplicaQM(name string) (bool, error) {
	return isRunningQM(name, "(REPLICA)")
}

func isRunningQM(name string, status string) (bool, error) {
	out, _, err := command.Run("dspmq", "-n", "-m", name)
	if err != nil {
		return false, err
	}
	if strings.Contains(string(out), status) {
		return true, nil
	}
	return false, nil
}
