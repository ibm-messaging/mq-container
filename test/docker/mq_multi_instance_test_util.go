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
	"testing"
	"context"
	"regexp"
	"strings"
	"strconv"
	"time"
	"fmt"

	"github.com/docker/docker/client"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
)

type QMChan struct {
	QMId string
	QMData string
    Error error
}

// configureMultiInstance creates the volumes and containers required for basic testing 
// of multi instance queue managers. Returns error, qm1a ID, qm1b ID, slice of volume names
func configureMultiInstance(t *testing.T, cli *client.Client) (error, string, string, []string) {

	qmsharedlogs := createVolume(t, cli, "qmsharedlogs")
	qmshareddata := createVolume(t, cli, "qmshareddata")

	err, qm1aId, qm1aData := startMultiInstanceQueueManager(t, cli, qmsharedlogs.Name, qmshareddata.Name)
	if err != nil {
		return err, "", "", []string{}
	}
	err, qm1bId, qm1bData := startMultiInstanceQueueManager(t, cli, qmsharedlogs.Name, qmshareddata.Name)
	if err != nil {
		return err, "", "", []string{}
	}

	volumes := []string{qmsharedlogs.Name, qmshareddata.Name, qm1aData, qm1bData}

	return nil, qm1aId, qm1bId, volumes
}

func singleInstance(t *testing.T, cli *client.Client, qmsharedlogs string, qmshareddata string, qmsChannel chan QMChan) {
	err, qmId, qmData := startMultiInstanceQueueManager(t, cli, qmsharedlogs, qmshareddata)
	if err != nil {
		qmsChannel <- QMChan{Error: err}
	}
	qmsChannel <- QMChan{QMId: qmId, QMData: qmData}
}

func startMultiInstanceQueueManager(t *testing.T, cli *client.Client, qmsharedlogs string, qmshareddata string) (error, string, string) {
	id := strconv.FormatInt(time.Now().UnixNano(), 10)
	qmData := createVolume(t, cli, id)
	containerConfig := container.Config{
		Image: imageName(),
		Env: []string{
			"LICENSE=accept",
			"MQ_QMGR_NAME=QM1",
			"MQ_MULTI_INSTANCE=true",
		},
	}
	hostConfig := container.HostConfig{
		Binds: []string{
			coverageBind(t),
			qmData.Name + ":/mnt/mqm",
			qmsharedlogs + ":/mnt/mqm-log",
			qmshareddata + ":/mnt/mqm-data",
		},
	}
	networkingConfig := network.NetworkingConfig{}
	qm, err := cli.ContainerCreate(context.Background(), &containerConfig, &hostConfig, &networkingConfig, t.Name()+id)
	if err != nil {
		return err, "", ""
	}
	err = startContainer(t, cli, qm.ID)
	if err != nil {
		return err, "", ""
	}
	err = waitForReady(t, cli, qm.ID)
	if err != nil {
		return err, "", ""
	}

	return nil, qm.ID, qmData.Name
}

func getActiveStandbyQueueManager(t *testing.T, cli *client.Client, qm1aId string, qm1bId string) (error, string, string) {
	qm1aStatus := getQueueManagerStatus(t, cli, qm1aId, "QM1")
	qm1bStatus := getQueueManagerStatus(t, cli, qm1bId, "QM1")

	if (qm1aStatus == "Running" && qm1bStatus == "Running as standby") {
		return nil, qm1aId, qm1bId
	} else if (qm1bStatus == "Running" && qm1aStatus == "Running as standby") {
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
