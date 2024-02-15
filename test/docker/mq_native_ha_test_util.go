/*
© Copyright IBM Corporation 2021,2024

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
	"path/filepath"
	"testing"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/ibm-messaging/mq-container/test/docker/pathutils"
)

const defaultHAPort = 9414

// HAReplicaStatus represents the Active/Replica/Replica container status of the queue manager
type HAReplicaStatus struct {
	Active  string
	Replica [2]string
}

func createBridgeNetwork(cli *client.Client, t *testing.T) (types.NetworkCreateResponse, error) {
	return cli.NetworkCreate(context.Background(), t.Name(), types.NetworkCreate{})
}

func removeBridgeNetwork(cli *client.Client, networkID string) error {
	return cli.NetworkRemove(context.Background(), networkID)
}

func getNativeHAContainerConfig(containerName string, replicaNames [3]string, haPort int) container.Config {
	return container.Config{
		Env: []string{
			"LICENSE=accept",
			"MQ_QMGR_NAME=QM1",
			"AMQ_CLOUD_PAK=true",
			"MQ_NATIVE_HA=true",
			fmt.Sprintf("HOSTNAME=%s", containerName),
			fmt.Sprintf("MQ_NATIVE_HA_INSTANCE_0_NAME=%s", replicaNames[0]),
			fmt.Sprintf("MQ_NATIVE_HA_INSTANCE_1_NAME=%s", replicaNames[1]),
			fmt.Sprintf("MQ_NATIVE_HA_INSTANCE_2_NAME=%s", replicaNames[2]),
			fmt.Sprintf("MQ_NATIVE_HA_INSTANCE_0_REPLICATION_ADDRESS=%s(%d)", replicaNames[0], haPort),
			fmt.Sprintf("MQ_NATIVE_HA_INSTANCE_1_REPLICATION_ADDRESS=%s(%d)", replicaNames[1], haPort),
			fmt.Sprintf("MQ_NATIVE_HA_INSTANCE_2_REPLICATION_ADDRESS=%s(%d)", replicaNames[2], haPort),
		},
	}
}

func getNativeHASecureHostConfig(t *testing.T) container.HostConfig {
	return container.HostConfig{
		Binds: []string{
			coverageBind(t),
			pathutils.CleanPath(filepath.Dir(getCwd(t, true)), "tls") + ":/etc/mqm/ha/pki/keys/ha",
		},
	}
}

func getNativeHANetworkConfig(networkID string) network.NetworkingConfig {
	return network.NetworkingConfig{
		EndpointsConfig: map[string]*network.EndpointSettings{
			networkID: &network.EndpointSettings{},
		},
	}
}

func getActiveReplicaInstances(t *testing.T, cli *client.Client, qmReplicaIDs [3]string) (HAReplicaStatus, error) {

	var actives []string
	var replicas []string

	for _, id := range qmReplicaIDs {
		qmReplicaStatus := getQueueManagerStatus(t, cli, id, "QM1")
		if qmReplicaStatus == "Running" {
			actives = append(actives, id)
		} else if qmReplicaStatus == "Replica" {
			replicas = append(replicas, id)
		} else {
			err := fmt.Errorf("Expected status to be Running or Replica, got status: %s", qmReplicaStatus)
			return HAReplicaStatus{}, err
		}
	}

	if len(actives) != 1 || len(replicas) != 2 {
		err := fmt.Errorf("Expected 1 Active and 2 Replicas, got: %d Active and %d Replica", len(actives), len(replicas))
		return HAReplicaStatus{}, err
	}

	return HAReplicaStatus{actives[0], [2]string{replicas[0], replicas[1]}}, nil
}

func waitForReadyHA(t *testing.T, cli *client.Client, qmReplicaIDs [3]string) {

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	for {
		select {
		case <-time.After(1 * time.Second):
			for _, id := range qmReplicaIDs {
				rc, _ := execContainer(t, cli, id, "", []string{"chkmqready"})
				if rc == 0 {
					t.Log("MQ is ready")
					rc, _ := execContainer(t, cli, id, "", []string{"chkmqstarted"})
					if rc == 0 {
						t.Log("MQ has started")
						return
					}
				}
			}
		case <-ctx.Done():
			t.Fatal("Timed out waiting for HA Queue Manager to become ready")
		}
	}
}

func waitForFailoverHA(t *testing.T, cli *client.Client, replicas [2]string) {

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	for {
		select {
		case <-time.After(1 * time.Second):
			for _, id := range replicas {
				if status := getQueueManagerStatus(t, cli, id, "QM1"); status == "Running" {
					return
				}
			}
		case <-ctx.Done():
			t.Fatal("Timed out waiting for Native HA Queue Manager to failover to an available replica")
		}
	}
}
