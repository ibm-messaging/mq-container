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
		cleanupVolume(t, cli, volume)
	}
	cleanupAfterTest(t, cli, qm1aId, false)
	cleanupAfterTest(t, cli, qm1bId, false)

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
		cleanupVolume(t, cli, volume)
	}
	cleanupAfterTest(t, cli, qm1aId, false)
	cleanupAfterTest(t, cli, qm1bId, false)

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
	cleanupVolume(t, cli, qmsharedlogs)
	qmshareddata := createVolume(t, cli, "qmshareddata")
	cleanupVolume(t, cli, qmshareddata)

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

	cleanupVolume(t, cli, qm1aData)
	cleanupVolume(t, cli, qm1bData)
	cleanupAfterTest(t, cli, qm1aId, false)
	cleanupAfterTest(t, cli, qm1bId, false)

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

	cleanupVolume(t, cli, qm1aData)
	cleanupAfterTest(t, cli, qm1aId, false)

	waitForTerminationMessage(t, cli, qm1aId, "Missing required mount '/mnt/mqm-log'", 30*time.Second)
}

// TestMultiInstanceNoSharedLogs starts 2 multi instance queue managers without providing a shared log
// mount, then checks to ensure that the container terminates with the expected message
func TestMultiInstanceNoSharedLogs(t *testing.T) {
	cli := ce.NewContainerClient()

	qmshareddata := createVolume(t, cli, "qmshareddata")
	cleanupVolume(t, cli, qmshareddata)

	err, qm1aId, qm1aData := startMultiVolumeQueueManager(t, cli, true, "", qmshareddata, miEnv, "", "", false)
	if err != nil {
		t.Fatal(err)
	}

	cleanupVolume(t, cli, qm1aData)
	cleanupAfterTest(t, cli, qm1aId, false)

	waitForTerminationMessage(t, cli, qm1aId, "Missing required mount '/mnt/mqm-log'", 30*time.Second)
}

// TestMultiInstanceNoSharedData starts 2 multi instance queue managers without providing a shared data
// mount, then checks to ensure that the container terminates with the expected message
func TestMultiInstanceNoSharedData(t *testing.T) {
	cli := ce.NewContainerClient()

	qmsharedlogs := createVolume(t, cli, "qmsharedlogs")
	cleanupVolume(t, cli, qmsharedlogs)

	err, qm1aId, qm1aData := startMultiVolumeQueueManager(t, cli, true, qmsharedlogs, "", miEnv, "", "", false)
	if err != nil {
		t.Fatal(err)
	}

	cleanupVolume(t, cli, qm1aData)
	cleanupAfterTest(t, cli, qm1aId, false)

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

	cleanupVolume(t, cli, qm1aData)
	cleanupAfterTest(t, cli, qm1aId, false)

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
		cleanupVolume(t, cli, volume)
	}
	cleanupAfterTest(t, cli, qm1aId, false)
	cleanupAfterTest(t, cli, qm1bId, false)

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

// TestMultiInstanceProbeLoggingGoldenPath tests probe logging in Multi-Instance configuration.
// Verifies first successful probe is logged on both instances (active and standby), and SIGTERM summary is generated on the active instance.
func TestMultiInstanceProbeLoggingGoldenPath(t *testing.T) {
	cli := ce.NewContainerClient()
	err, qm1aId, qm1bId, volumes := configureMultiInstance(t, cli, false)
	if err != nil {
		t.Fatal(err)
	}
	for _, volume := range volumes {
		cleanupVolume(t, cli, volume)
	}

	cleanupAfterTest(t, cli, qm1aId, false)
	cleanupAfterTest(t, cli, qm1bId, false)

	waitForReady(t, cli, qm1aId)
	waitForReady(t, cli, qm1bId)

	// Execute chkmqready and chkmqhealthy on both instances
	qmIds := []string{qm1aId, qm1bId}

	for _, id := range qmIds {
		// Execute chkmqready and chkmqhealthy multiple times
		for i := 1; i <= 3; i++ {

			rc, _ := execContainer(t, cli, id, "", []string{"chkmqstarted"})
			if rc != 0 {
				t.Errorf("Expected startup probe to pass with rc=0, got rc=%d", rc)
			}

			rc, _ = execContainer(t, cli, id, "", []string{"chkmqhealthy"})
			if rc != 0 {
				t.Errorf("Expected liveness probe to pass with rc=0, got rc=%d", rc)
			}
		}
	}

	// Verify first successful startup probe and liveness probe were logged once only.
	// Additional identical successful probes should be suppressed by deduplication.
	for _, id := range qmIds {
		containerLogs := inspectLogs(t, cli, id)

		startupPassedCount := strings.Count(containerLogs, "Startup Probe Passed")
		if startupPassedCount != 1 {
			t.Errorf("Expected exactly one startup probe pass log due to deduplication, got %d", startupPassedCount)
		}

		livenessPassedCount := strings.Count(containerLogs, "Liveness Probe Passed")
		if livenessPassedCount != 1 {
			t.Errorf("Expected exactly one liveness probe pass log due to deduplication, got %d", livenessPassedCount)
		}
	}

	// Get the active instance
	err, active, _ := getActiveStandbyQueueManager(t, cli, qm1aId, qm1bId)
	if err != nil {
		t.Fatal(err)
	}

	// Kill active with SIGTERM
	killContainer(t, cli, active, "SIGTERM")

	// Verify probe summary in active logs
	containerLogs := inspectLogs(t, cli, active)

	// Since the liveness probe has been executed, startup probe summary should not be logged
	if strings.Contains(containerLogs, "----- Start Startup Probe Summary -----") {
		t.Errorf("Startup probe summary logged at SIGTERM, even when liveness probe has been executed")
	}

	if !strings.Contains(containerLogs, "----- Start Liveness Probe Summary -----") {
		t.Errorf("Expected liveness probe summary at SIGTERM")
	}
}

// TestMultiInstanceLivenessProbeLoggingFailureRecovery tests liveness probe logging deduplication on the active Multi-Instance queue manager.
// Verifies pass-fail-fail-pass pattern logging and SIGTERM summary generation.
func TestMultiInstanceLivenessProbeLoggingFailureRecovery(t *testing.T) {
	cli := ce.NewContainerClient()
	err, qm1aId, qm1bId, volumes := configureMultiInstance(t, cli, false)
	if err != nil {
		t.Fatal(err)
	}
	for _, volume := range volumes {
		cleanupVolume(t, cli, volume)
	}

	cleanupAfterTest(t, cli, qm1aId, false)
	cleanupAfterTest(t, cli, qm1bId, false)

	waitForReady(t, cli, qm1aId)
	waitForReady(t, cli, qm1bId)

	err, active, _ := getActiveStandbyQueueManager(t, cli, qm1aId, qm1bId)
	if err != nil {
		t.Fatal(err)
	}

	// First success
	rc, _ := execContainer(t, cli, active, "", []string{"chkmqhealthy"})
	if rc != 0 {
		t.Errorf("Expected liveness probe to pass with rc=0, got rc=%d", rc)
	}

	// Stop the QueueManager
	execContainer(t, cli, active, "", []string{"endmqm", "-i", "QM1"})
	time.Sleep(2 * time.Second)

	// Execute the chkmqhealthy command, it will now fail
	rc, _ = execContainer(t, cli, active, "", []string{"chkmqhealthy"})
	if rc == 0 {
		t.Errorf("Expected liveness probe to fail")
	}

	rc, _ = execContainer(t, cli, active, "", []string{"chkmqhealthy"})
	if rc == 0 {
		t.Errorf("Expected liveness probe to fail")
	}

	// Start the QueueManager
	execContainer(t, cli, active, "", []string{"strmqm", "QM1"})
	waitForReady(t, cli, active)

	// Execute the chkmqhealthy command
	rc, _ = execContainer(t, cli, active, "", []string{"chkmqhealthy"})
	if rc != 0 {
		t.Errorf("Expected liveness probe to pass with rc=0, got rc=%d", rc)
	}

	containerLogs := inspectLogs(t, cli, active)

	// Verify: 1st pass, 2 fails, recovery pass all logged
	passedRuntimeLogCount := strings.Count(containerLogs, "Liveness Probe Passed")
	failedRuntimeLogCount := strings.Count(containerLogs, "Liveness Probe Failed")

	if passedRuntimeLogCount != 2 {
		t.Errorf("Expected 2 liveness probe pass logs (first + recovery), got %d", passedRuntimeLogCount)
	}

	if failedRuntimeLogCount != 2 {
		t.Errorf("Expected 2 liveness probe failure logs, got %d", passedRuntimeLogCount)
	}
}

// TestMultiinstanceStartupProbeLoggingOnSigterm verifies that when only the
// startup probe has executed, the startup probe summary is logged on SIGTERM.
func TestMultiinstanceStartupProbeLoggingOnSigterm(t *testing.T) {
	cli := ce.NewContainerClient()
	err, qm1aId, qm1bId, volumes := configureMultiInstance(t, cli, false)
	if err != nil {
		t.Fatal(err)
	}
	for _, volume := range volumes {
		cleanupVolume(t, cli, volume)
	}

	cleanupAfterTest(t, cli, qm1aId, false)
	cleanupAfterTest(t, cli, qm1bId, false)

	waitForReady(t, cli, qm1aId)
	waitForReady(t, cli, qm1bId)

	err, active, _ := getActiveStandbyQueueManager(t, cli, qm1aId, qm1bId)
	if err != nil {
		t.Fatal(err)
	}

	// Stop thre QueueManager
	execContainer(t, cli, active, "", []string{"endmqm", "-i", "QM1"})
	time.Sleep(2 * time.Second)

	// Execute the chkmqstarted command multiple times
	for i := 1; i <= 5; i++ {
		rc, _ := execContainer(t, cli, active, "", []string{"chkmqstarted"})
		if rc == 0 {
			t.Errorf("Expected startup probe to fail on attempt %d", i)
		}
	}

	// Kill active with SIGTERM
	killContainer(t, cli, active, "SIGTERM")

	containerLogs := inspectLogs(t, cli, active)

	if !strings.Contains(containerLogs, "----- Start Startup Probe Summary -----") {
		t.Errorf("Expected startup probe summary at SIGTERM")
	}

	if strings.Contains(containerLogs, "----- Start Liveness Probe Summary -----") {
		t.Errorf("Did not expect liveness probe summary at SIGTERM when liveness probe has not been executed")
	}

	if !strings.Contains(containerLogs, "Last State: Failed") {
		t.Errorf("Expected startup probe summary to show failed last run, logs were: %s", containerLogs)
	}
}
