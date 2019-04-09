/*
© Copyright IBM Corporation 2018, 2019

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
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
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

func getHostConfig(t *testing.T, mounts int, qmsharedlogs string, qmshareddata string, qmdata string) container.HostConfig {

	var hostConfig container.HostConfig

	switch mounts {
	case 1:
		hostConfig = container.HostConfig{
			Binds: []string{
				coverageBind(t),
				qmdata + ":/mnt/mqm",
			},
		}
	case 2:
		hostConfig = container.HostConfig{
			Binds: []string{
				coverageBind(t),
				qmdata + ":/mnt/mqm",
				qmshareddata + ":/mnt/mqm-data",
			},
		}
	case 3:
		hostConfig = container.HostConfig{
			Binds: []string{
				coverageBind(t),
				qmdata + ":/mnt/mqm",
				qmsharedlogs + ":/mnt/mqm-log",
			},
		}
	case 4:
		hostConfig = container.HostConfig{
			Binds: []string{
				coverageBind(t),
				qmdata + ":/mnt/mqm",
				qmsharedlogs + ":/mnt/mqm-log",
				qmshareddata + ":/mnt/mqm-data",
			},
		}
	}

	return hostConfig
}

func startMultiVolumeQueueManager(t *testing.T, cli *client.Client, dataVol bool, qmsharedlogs string, qmshareddata string, env []string) (error, string, string) {
	id := strconv.FormatInt(time.Now().UnixNano(), 10)
	qmdata := createVolume(t, cli, id)
	containerConfig := container.Config{
		Image: imageName(),
		Env:   env,
	}
	var hostConfig container.HostConfig

	if !dataVol {
		hostConfig = container.HostConfig{}
	} else if qmsharedlogs == "" && qmshareddata == "" {
		hostConfig = getHostConfig(t, 1, "", "", qmdata.Name)
	} else if qmsharedlogs == "" {
		hostConfig = getHostConfig(t, 2, "", qmshareddata, qmdata.Name)
	} else if qmshareddata == "" {
		hostConfig = getHostConfig(t, 3, qmsharedlogs, "", qmdata.Name)
	} else {
		hostConfig = getHostConfig(t, 4, qmsharedlogs, qmshareddata, qmdata.Name)
	}
	networkingConfig := network.NetworkingConfig{}
	qm, err := cli.ContainerCreate(context.Background(), &containerConfig, &hostConfig, &networkingConfig, t.Name()+id)
	if err != nil {
		return err, "", ""
	}
	startContainer(t, cli, qm.ID)

	return nil, qm.ID, qmdata.Name
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
	_, dspmqOut := execContainer(t, cli, containerID, "mqm", []string{"bash", "-c", "dspmq", "-m", queueManagerName})
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
