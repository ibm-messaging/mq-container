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
	"strings"
	"context"

	"github.com/docker/docker/client"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
)

// TestMultiInstanceStartup creates 2 containers
func TestMultiInstanceStartup(t *testing.T) {
	t.Parallel()

	cli, err := client.NewEnvClient()
	if err != nil {
		t.Fatal(err)
	}
	qm1adata := createVolume(t, cli, "qm1adata")
	defer removeVolume(t, cli, qm1adata.Name)
	qmsharedlogs := createVolume(t, cli, "qmsharedlogs")
	defer removeVolume(t, cli, qmsharedlogs.Name)
	qmshareddata := createVolume(t, cli, "qmshareddata")
	defer removeVolume(t, cli, qmshareddata.Name)
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
			qm1adata.Name + ":/mnt/mqm",
			qmsharedlogs.Name + ":/mnt/mqm-log",
			qmshareddata.Name + ":/mnt/mqm-data",
		},
	}
	networkingConfig := network.NetworkingConfig{}
	qm1a, err := cli.ContainerCreate(context.Background(), &containerConfig, &hostConfig, &networkingConfig, t.Name()+"qm1a")
	if err != nil {
		t.Fatal(err)
	}
	defer cleanContainer(t, cli, qm1a.ID)
	startContainer(t, cli, qm1a.ID)
	waitForReady(t, cli, qm1a.ID)

	qm1bdata := createVolume(t, cli, "qm1bdata")
	defer removeVolume(t, cli, qm1bdata.Name)
	containerConfig = container.Config{
		Image: imageName(),
		Env: []string{
			"LICENSE=accept",
			"MQ_QMGR_NAME=QM1",
			"MQ_MULTI_INSTANCE=true",
		},
	}
	hostConfig = container.HostConfig{
		Binds: []string{
			coverageBind(t),
			qm1bdata.Name + ":/mnt/mqm",
			qmsharedlogs.Name + ":/mnt/mqm-log",
			qmshareddata.Name + ":/mnt/mqm-data",
		},
	}
	networkingConfig = network.NetworkingConfig{}
	qm1b, err := cli.ContainerCreate(context.Background(), &containerConfig, &hostConfig, &networkingConfig, t.Name()+"qm1b")
	if err != nil {
		t.Fatal(err)
	}
	defer cleanContainer(t, cli, qm1b.ID)
	startContainer(t, cli, qm1b.ID)
	waitForReady(t, cli, qm1b.ID)

	_, dspmqOut := execContainer(t, cli, qm1a.ID, "mqm", []string{"bash", "-c", "dspmq", "-m", "QM1"})
	if strings.Contains(dspmqOut, "STATUS(Running)") == false {
		t.Fatalf("Expected QM1 to be running on active queue manager, dspmq returned %v", dspmqOut)
	}

	_, dspmqOut = execContainer(t, cli, qm1b.ID, "mqm", []string{"bash", "-c", "dspmq", "-m", "QM1"})
	if strings.Contains(dspmqOut, "STATUS(Running as standby)") == false {
		t.Fatalf("Expected QM1 to be running as standby on standby queue manager, dspmq returned %v", dspmqOut)
	}

}