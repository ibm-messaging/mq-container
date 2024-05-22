/*
© Copyright IBM Corporation 2018, 2024

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
	"context"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/ibm-messaging/mq-container/internal/command"
)

const readyFile string = "/run/runmqserver/ready"
const readyToSyncFile string = "/run/runmqserver/ready-to-sync"

func fileExists(fileName string) (bool, error) {
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
	err := clearFile(readyFile)
	if err != nil {
		return err
	}
	err = clearFile(readyToSyncFile)
	if err != nil {
		return err
	}

	return nil
}

// clearFile removes the specified file if it exists
func clearFile(fileName string) error {
	exist, err := fileExists(fileName)
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
	return os.WriteFile(readyFile, []byte("1"), 0770)
}

// Check checks whether or not the queue manager has finished its
// configuration steps
func Check() (bool, error) {
	exists, err := fileExists(readyFile)
	if err != nil {
		return false, err
	}
	return exists, nil
}

// SetReadyToSync is used to indicate that a Native-HA queue manager instance is ready-to-sync
func SetReadyToSync() error {

	exists, err := fileExists(readyToSyncFile)
	if err != nil {
		return err
	} else if exists {
		return nil
	}

	readyToSyncStartTime := strconv.FormatInt(time.Now().Unix(), 10)
	// #nosec G306 - required permissions
	return os.WriteFile(readyToSyncFile, []byte(readyToSyncStartTime), 0660)
}

// GetReadyToSyncStartTime returns the start-time a Native-HA queue manager instance was ready-to-sync
func GetReadyToSyncStartTime() (bool, time.Time, error) {

	exists, err := fileExists(readyToSyncFile)
	if err != nil {
		return exists, time.Time{}, err
	}

	if exists {
		buf, err := os.ReadFile(readyToSyncFile)
		if err != nil {
			return true, time.Time{}, err
		}
		readyToSyncStartTime, err := strconv.ParseInt(string(buf), 10, 64)
		if err != nil {
			return true, time.Time{}, err
		}
		return true, time.Unix(readyToSyncStartTime, 0), nil
	}

	return false, time.Time{}, nil
}

// Status returns an enum representing the current running status of the queue manager
func Status(ctx context.Context, name string) (QMStatus, error) {
	out, _, err := command.RunContext(ctx, "dspmq", "-n", "-m", name)
	if err != nil {
		return StatusUnknown, err
	}
	if strings.Contains(string(out), "(RUNNING)") {
		return StatusActiveQM, nil
	}
	if strings.Contains(string(out), "(RUNNING AS STANDBY)") {
		return StatusStandbyQM, nil
	}
	if strings.Contains(string(out), "(REPLICA)") {
		return StatusStandbyQM, nil
	}
	if strings.Contains(string(out), "(RECOVERY GROUP LEADER)") {
		return StatusRecoveryQM, nil
	}
	return StatusUnknown, nil
}

type QMStatus int

const (
	StatusUnknown QMStatus = iota
	StatusActiveQM
	StatusStandbyQM
	StatusReplicaQM
	StatusRecoveryQM
)

// ActiveQM returns true if the queue manager is running in active mode
func (s QMStatus) ActiveQM() bool { return s == StatusActiveQM }

// StandbyQM returns true if the queue manager is running in standby mode
func (s QMStatus) StandbyQM() bool { return s == StatusStandbyQM }

// ReplicaQM returns true if the queue manager is running in replica mode
func (s QMStatus) ReplicaQM() bool { return s == StatusReplicaQM }

// ReplicaQM returns true if the queue manager is running in recovery mode
func (s QMStatus) RecoveryQM() bool { return s == StatusRecoveryQM }
