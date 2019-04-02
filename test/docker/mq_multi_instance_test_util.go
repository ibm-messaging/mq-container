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

	"github.com/docker/docker/client"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
)

// configureMultiInstance creates the volumes and containers required for testing multi
// instance queue managers. Returns error, qm1a ID, qm1b ID, slice of volume names
func configureMultiInstance(t *testing.T, cli *client.Client) (error, string, string, []string) {

	qmsharedlogs := createVolume(t, cli, "qmsharedlogs")
	qmshareddata := createVolume(t, cli, "qmshareddata")

	qm1adata := createVolume(t, cli, "qm1adata")
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
		return err, "", "", []string{}
	}
	startContainer(t, cli, qm1a.ID)
	waitForReady(t, cli, qm1a.ID)
	
	qm1bdata := createVolume(t, cli, "qm1bdata")
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
		return err, "", "", []string{}
	}
	startContainer(t, cli, qm1b.ID)
	waitForReady(t, cli, qm1b.ID)
	
	volumes := []string{qmsharedlogs.Name, qmshareddata.Name, qm1adata.Name, qm1bdata.Name}

	return nil, qm1a.ID, qm1b.ID, volumes
}
