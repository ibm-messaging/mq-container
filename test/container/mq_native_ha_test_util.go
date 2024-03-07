/*
© Copyright IBM Corporation 2021, 2023

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
	"strconv"
	"testing"
	"time"

	ce "github.com/ibm-messaging/mq-container/test/container/containerengine"
	"github.com/ibm-messaging/mq-container/test/container/pathutils"
)

const defaultHAPort = 9414

// HAReplicaStatus represents the Active/Replica/Replica container status of the queue manager
type HAReplicaStatus struct {
	Active  string
	Replica [2]string
}

func getNativeHAContainerConfig(containerName string, replicaNames [3]string, haPort int) ce.ContainerConfig {
	return ce.ContainerConfig{
		Env: []string{
			"LICENSE=accept",
			"MQ_QMGR_NAME=QM1",
			"AMQ_CLOUD_PAK=true",
			"MQ_NATIVE_HA=true",
			fmt.Sprintf("HOSTNAME=%s", containerName),
			fmt.Sprintf("MQ_NATIVE_HA_INSTANCE_0_NAME=%s", replicaNames[0]),
			fmt.Sprintf("MQ_NATIVE_HA_INSTANCE_1_NAME=%s", replicaNames[1]),
			fmt.Sprintf("MQ_NATIVE_HA_INSTANCE_2_NAME=%s", replicaNames[2]),
			fmt.Sprintf("MQ_NATIVE_HA_INSTANCE_0_REPLICATION_ADDRESS=%s(%d)", "127.0.0.1", haPort+0),
			fmt.Sprintf("MQ_NATIVE_HA_INSTANCE_1_REPLICATION_ADDRESS=%s(%d)", "127.0.0.1", haPort+1),
			fmt.Sprintf("MQ_NATIVE_HA_INSTANCE_2_REPLICATION_ADDRESS=%s(%d)", "127.0.0.1", haPort+2),
		},
		//When using the host for networking a consistent user was required. If a random user is used then the following example error was recorded.
		//AMQ3209E: Native HA connection rejected due to configuration mismatch of 'QmgrUserId=5024'
		User: "1111",
	}
}

func getNativeHASecureHostConfig(t *testing.T) ce.ContainerHostConfig {
	hostConfig := ce.ContainerHostConfig{
		Binds: []string{
			pathutils.CleanPath(filepath.Dir(getCwd(t, true)), "../tls") + ":/etc/mqm/ha/pki/keys/ha",
		},
	}
	addCoverageBindIfAvailable(t, &hostConfig)
	return hostConfig
}

func getNativeHANetworkConfig(networkID string) ce.ContainerNetworkSettings {
	return ce.ContainerNetworkSettings{
		Networks: []string{networkID},
	}
}

// populatePortBindings writes port bindings to the host config
func populateNativeHAPortBindings(ports []int, nativeHaPort int, hostConfig ce.ContainerHostConfig) ce.ContainerHostConfig {
	hostConfig.PortBindings = []ce.PortBinding{}
	var binding ce.PortBinding
	for i, p := range ports {
		port := fmt.Sprintf("%v/tcp", p)
		binding = ce.PortBinding{
			ContainerPort: port,
			HostIP:        "0.0.0.0",
			//Offset the ports by 50 if there are multiple
			HostPort: strconv.Itoa(nativeHaPort + 50*i),
		}
		hostConfig.PortBindings = append(hostConfig.PortBindings, binding)
	}
	return hostConfig
}

func getActiveReplicaInstances(t *testing.T, cli ce.ContainerInterface, qmReplicaIDs [3]string) (HAReplicaStatus, error) {

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

func waitForReadyHA(t *testing.T, cli ce.ContainerInterface, qmReplicaIDs [3]string) {

	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
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

func waitForFailoverHA(t *testing.T, cli ce.ContainerInterface, replicas [2]string) {

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
