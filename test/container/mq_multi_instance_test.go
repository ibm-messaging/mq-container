/*
© Copyright IBM Corporation 2019, 2023

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
package main

import (
	"context"
	"strings"
	"testing"
	"time"

	ce "github.com/ibm-messaging/mq-container/test/container/containerengine"
)

var miEnv = []string{
	"LICENSE=accept",
	"MQ_QMGR_NAME=QM1",
	"MQ_MULTI_INSTANCE=true",
}

// TestMultiInstanceStartStop creates 2 containers in a multi instance queue manager configuration
// and starts/stop them checking we always have an active and standby
func TestMultiInstanceStartStop(t *testing.T) {
	t.Skipf("Skipping %v until test defect fixed", t.Name())
	cli := ce.NewContainerClient()
	err, qm1aId, qm1bId, volumes := configureMultiInstance(t, cli, false)
	if err != nil {
		t.Fatal(err)
	}
	for _, volume := range volumes {
		defer removeVolume(t, cli, volume)
	}
	defer cleanContainer(t, cli, qm1aId, false)
	defer cleanContainer(t, cli, qm1bId, false)

	waitForReady(t, cli, qm1aId)
	waitForReady(t, cli, qm1bId)

	err, active, standby := getActiveStandbyQueueManager(t, cli, qm1aId, qm1bId)
	if err != nil {
		t.Fatal(err)
	}

	killContainer(t, cli, active, "SIGTERM")
	time.Sleep(2 * time.Second)

	if status := getQueueManagerStatus(t, cli, standby, "QM1"); strings.Compare(status, "Running") != 0 {
		t.Fatalf("Expected QM1 to be running as active queue manager, dspmq returned status of %v", status)
	}

	startContainer(t, cli, qm1aId)
	waitForReady(t, cli, qm1aId)

	err, _, _ = getActiveStandbyQueueManager(t, cli, qm1aId, qm1bId)
	if err != nil {
		t.Fatal(err)
	}

}

// TestMultiInstanceContainerStop starts 2 containers in a multi instance queue manager configuration,
// stops the active queue manager, then checks to ensure the backup queue manager becomes active
func TestMultiInstanceContainerStop(t *testing.T) {
	cli := ce.NewContainerClient()
	err, qm1aId, qm1bId, volumes := configureMultiInstance(t, cli, false)
	if err != nil {
		t.Fatal(err)
	}
	for _, volume := range volumes {
		defer removeVolume(t, cli, volume)
	}
	defer cleanContainer(t, cli, qm1aId, false)
	defer cleanContainer(t, cli, qm1bId, false)

	waitForReady(t, cli, qm1aId)
	waitForReady(t, cli, qm1bId)

	err, originalActive, originalStandby := getActiveStandbyQueueManager(t, cli, qm1aId, qm1bId)
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	stopContainer(t, cli, originalActive)

	for {
		status := getQueueManagerStatus(t, cli, originalStandby, "QM1")
		select {
		case <-time.After(1 * time.Second):
			if status == "Running" {
				t.Logf("Original standby is now the active")
				return
			} else if status == "Starting" {
				t.Logf("Original standby is starting")
			}
		case <-ctx.Done():
			t.Fatalf("%s Timed out waiting for standby to become the active.  Status=%v", time.Now().Format(time.RFC3339), status)
		}
	}
}

// TestMultiInstanceRace starts 2 containers in separate goroutines in a multi instance queue manager
// configuration, then checks to ensure that both an active and standby queue manager have been started
func TestMultiInstanceRace(t *testing.T) {
	t.Skipf("Skipping %v until file lock is implemented", t.Name())
	cli := ce.NewContainerClient()
	qmsharedlogs := createVolume(t, cli, "qmsharedlogs")
	defer removeVolume(t, cli, qmsharedlogs)
	qmshareddata := createVolume(t, cli, "qmshareddata")
	defer removeVolume(t, cli, qmshareddata)

	qmsChannel := make(chan QMChan)

	go singleMultiInstanceQueueManager(t, cli, qmsharedlogs, qmshareddata, qmsChannel)
	go singleMultiInstanceQueueManager(t, cli, qmsharedlogs, qmshareddata, qmsChannel)

	qm1a := <-qmsChannel
	if qm1a.Error != nil {
		t.Fatal(qm1a.Error)
	}

	qm1b := <-qmsChannel
	if qm1b.Error != nil {
		t.Fatal(qm1b.Error)
	}

	qm1aId, qm1aData := qm1a.QMId, qm1a.QMData
	qm1bId, qm1bData := qm1b.QMId, qm1b.QMData

	defer removeVolume(t, cli, qm1aData)
	defer removeVolume(t, cli, qm1bData)
	defer cleanContainer(t, cli, qm1aId, false)
	defer cleanContainer(t, cli, qm1bId, false)

	waitForReady(t, cli, qm1aId)
	waitForReady(t, cli, qm1bId)

	err, _, _ := getActiveStandbyQueueManager(t, cli, qm1aId, qm1bId)
	if err != nil {
		t.Fatal(err)
	}
}

// TestMultiInstanceNoSharedMounts starts 2 multi instance queue managers without providing shared log/data
// mounts, then checks to ensure that the container terminates with the expected message
func TestMultiInstanceNoSharedMounts(t *testing.T) {
	t.Parallel()
	cli := ce.NewContainerClient()

	err, qm1aId, qm1aData := startMultiVolumeQueueManager(t, cli, true, "", "", miEnv, "", "", false)
	if err != nil {
		t.Fatal(err)
	}

	defer removeVolume(t, cli, qm1aData)
	defer cleanContainer(t, cli, qm1aId, false)

	waitForTerminationMessage(t, cli, qm1aId, "Missing required mount '/mnt/mqm-log'", 30*time.Second)
}

// TestMultiInstanceNoSharedLogs starts 2 multi instance queue managers without providing a shared log
// mount, then checks to ensure that the container terminates with the expected message
func TestMultiInstanceNoSharedLogs(t *testing.T) {
	cli := ce.NewContainerClient()

	qmshareddata := createVolume(t, cli, "qmshareddata")
	defer removeVolume(t, cli, qmshareddata)

	err, qm1aId, qm1aData := startMultiVolumeQueueManager(t, cli, true, "", qmshareddata, miEnv, "", "", false)
	if err != nil {
		t.Fatal(err)
	}

	defer removeVolume(t, cli, qm1aData)
	defer cleanContainer(t, cli, qm1aId, false)

	waitForTerminationMessage(t, cli, qm1aId, "Missing required mount '/mnt/mqm-log'", 30*time.Second)
}

// TestMultiInstanceNoSharedData starts 2 multi instance queue managers without providing a shared data
// mount, then checks to ensure that the container terminates with the expected message
func TestMultiInstanceNoSharedData(t *testing.T) {
	cli := ce.NewContainerClient()

	qmsharedlogs := createVolume(t, cli, "qmsharedlogs")
	defer removeVolume(t, cli, qmsharedlogs)

	err, qm1aId, qm1aData := startMultiVolumeQueueManager(t, cli, true, qmsharedlogs, "", miEnv, "", "", false)
	if err != nil {
		t.Fatal(err)
	}

	defer removeVolume(t, cli, qm1aData)
	defer cleanContainer(t, cli, qm1aId, false)

	waitForTerminationMessage(t, cli, qm1aId, "Missing required mount '/mnt/mqm-data'", 30*time.Second)
}

// TestMultiInstanceNoMounts starts 2 multi instance queue managers without providing a shared data
// mount, then checks to ensure that the container terminates with the expected message
func TestMultiInstanceNoMounts(t *testing.T) {
	cli := ce.NewContainerClient()

	err, qm1aId, qm1aData := startMultiVolumeQueueManager(t, cli, false, "", "", miEnv, "", "", false)
	if err != nil {
		t.Fatal(err)
	}

	defer removeVolume(t, cli, qm1aData)
	defer cleanContainer(t, cli, qm1aId, false)

	waitForTerminationMessage(t, cli, qm1aId, "Missing required mount '/mnt/mqm'", 30*time.Second)
}

// TestRoRFsMultiInstanceContainerStop starts 2 containers in a multi instance queue manager configuration,
// with read-only root filesystem stops the active queue manager, then checks to ensure the backup queue
// manager becomes active
func TestRoRFsMultiInstanceContainerStop(t *testing.T) {
	cli := ce.NewContainerClient()
	err, qm1aId, qm1bId, volumes := configureMultiInstance(t, cli, true)
	if err != nil {
		t.Fatal(err)
	}
	for _, volume := range volumes {
		defer removeVolume(t, cli, volume)
	}
	defer cleanContainer(t, cli, qm1aId, false)
	defer cleanContainer(t, cli, qm1bId, false)

	waitForReady(t, cli, qm1aId)
	waitForReady(t, cli, qm1bId)

	err, originalActive, originalStandby := getActiveStandbyQueueManager(t, cli, qm1aId, qm1bId)
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	stopContainer(t, cli, originalActive)

	for {
		status := getQueueManagerStatus(t, cli, originalStandby, "QM1")
		select {
		case <-time.After(1 * time.Second):
			if status == "Running" {
				t.Logf("Original standby is now the active")
				return
			} else if status == "Starting" {
				t.Logf("Original standby is starting")
			}
		case <-ctx.Done():
			t.Fatalf("%s Timed out waiting for standby to become the active.  Status=%v", time.Now().Format(time.RFC3339), status)
		}
	}
}
