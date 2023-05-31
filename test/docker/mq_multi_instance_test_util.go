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
	"fmt"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/docker/docker/client"
)

type QMChan struct {
	QMId   string
	QMData string
	Error  error
}

// configureMultiInstance creates the volumes and containers required for basic testing
// of multi instance queue managers. Returns error, qm1a ID, qm1b ID, slice of volume names
func configureMultiInstance(t *testing.T, cli *client.Client) (error, string, string, []string) {

	qmsharedlogs := createVolume(t, cli, "qmsharedlogs")
	qmshareddata := createVolume(t, cli, "qmshareddata")

	err, qm1aId, qm1aData := startMultiVolumeQueueManager(t, cli, true, qmsharedlogs.Name, qmshareddata.Name, miEnv)
	if err != nil {
		return err, "", "", []string{}
	}
	time.Sleep(10 * time.Second)
	err, qm1bId, qm1bData := startMultiVolumeQueueManager(t, cli, true, qmsharedlogs.Name, qmshareddata.Name, miEnv)
	if err != nil {
		return err, "", "", []string{}
	}

	volumes := []string{qmsharedlogs.Name, qmshareddata.Name, qm1aData, qm1bData}

	return nil, qm1aId, qm1bId, volumes
}

func singleMultiInstanceQueueManager(t *testing.T, cli *client.Client, qmsharedlogs string, qmshareddata string, qmsChannel chan QMChan) {
	err, qmId, qmData := startMultiVolumeQueueManager(t, cli, true, qmsharedlogs, qmshareddata, miEnv)
	if err != nil {
		qmsChannel <- QMChan{Error: err}
	}
	qmsChannel <- QMChan{QMId: qmId, QMData: qmData}
}

func getActiveStandbyQueueManager(t *testing.T, cli *client.Client, qm1aId string, qm1bId string) (error, string, string) {
	qm1aStatus := getQueueManagerStatus(t, cli, qm1aId, "QM1")
	qm1bStatus := getQueueManagerStatus(t, cli, qm1bId, "QM1")

	if qm1aStatus == "Running" && qm1bStatus == "Running as standby" {
		return nil, qm1aId, qm1bId
	} else if qm1bStatus == "Running" && qm1aStatus == "Running as standby" {
		return nil, qm1bId, qm1aId
	}
	err := fmt.Errorf("Expected to be running in multi instance configuration, got status 1) %v status 2) %v", qm1aStatus, qm1bStatus)
	return err, "", ""
}

func getQueueManagerStatus(t *testing.T, cli *client.Client, containerID string, queueManagerName string) string {
	_, dspmqOut := execContainer(t, cli, containerID, "", []string{"bash", "-c", "dspmq", "-m", queueManagerName})
	t.Logf("dspmq for %v (%v) returned: %v", containerID, queueManagerName, dspmqOut)
	regex := regexp.MustCompile(`STATUS\(.*\)`)
	status := regex.FindString(dspmqOut)
	status = strings.TrimSuffix(strings.TrimPrefix(status, "STATUS("), ")")
	return status
}

func waitForTerminationMessage(t *testing.T, cli *client.Client, qmId string, terminationString string, timeout time.Duration) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	for {
		select {
		case <-time.After(1 * time.Second):
			m := terminationMessage(t, cli, qmId)
			if m != "" {
				if !strings.Contains(m, terminationString) {
					t.Fatalf("Expected container to fail on missing required mount. Got termination message: %v", m)
				}
				return
			}
		case <-ctx.Done():
			t.Fatal("Timed out waiting for container to terminate")
		}
	}
}
