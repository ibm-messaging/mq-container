/*
© Copyright IBM Corporation 2017

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
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
)

func TestLicenseNotSet(t *testing.T) {
	cli, err := client.NewEnvClient()
	if err != nil {
		t.Fatal(err)
	}
	containerConfig := container.Config{}
	id := runContainer(t, cli, &containerConfig)
	defer cleanContainer(t, cli, id)
	rc := waitForContainer(t, cli, id, 5)
	if rc != 1 {
		t.Errorf("Expected rc=1, got rc=%v", rc)
	}
}

func TestLicenseView(t *testing.T) {
	cli, err := client.NewEnvClient()
	if err != nil {
		t.Fatal(err)
	}
	containerConfig := container.Config{
		Env: []string{"LICENSE=view"},
	}
	id := runContainer(t, cli, &containerConfig)
	defer cleanContainer(t, cli, id)
	rc := waitForContainer(t, cli, id, 5)
	if rc != 1 {
		t.Errorf("Expected rc=1, got rc=%v", rc)
	}
	l := inspectLogs(t, cli, id)
	const s string = "terms"
	if !strings.Contains(l, s) {
		t.Errorf("Expected license string to contain \"%v\", got %v", s, l)
	}
}

// TestGoldenPath starts a queue manager successfully
func TestGoldenPath(t *testing.T) {
	cli, err := client.NewEnvClient()
	if err != nil {
		t.Fatal(err)
	}
	containerConfig := container.Config{
		Env: []string{"LICENSE=accept", "MQ_QMGR_NAME=qm1"},
		//ExposedPorts: ports,
		ExposedPorts: nat.PortSet{
			"1414/tcp": struct{}{},
		},
	}
	id := runContainer(t, cli, &containerConfig)
	defer cleanContainer(t, cli, id)
	waitForReady(t, cli, id)
}

func utilTestNoQueueManagerName(t *testing.T, hostName string, expectedName string) {
	search := "QMNAME(" + expectedName + ")"
	cli, err := client.NewEnvClient()
	if err != nil {
		t.Fatal(err)
	}
	containerConfig := container.Config{
		Env:      []string{"LICENSE=accept"},
		Hostname: hostName,
		ExposedPorts: nat.PortSet{
			"1414/tcp": struct{}{},
		},
	}
	id := runContainer(t, cli, &containerConfig)
	defer cleanContainer(t, cli, id)
	waitForReady(t, cli, id)
	out := execContainerWithOutput(t, cli, id, "mqm", []string{"dspmq"})
	if !strings.Contains(out, search) {
		t.Errorf("Expected result of running dspmq to contain name=%v, got name=%v", search, out)
	}
}
func TestNoQueueManagerName(t *testing.T) {
	utilTestNoQueueManagerName(t, "test", "test")
}

func TestNoQueueManagerNameInvalidHostname(t *testing.T) {
	utilTestNoQueueManagerName(t, "test-1", "test1")
}

// TestWithVolume runs a container with a Docker volume, then removes that
// container and starts a new one with same volume.
func TestWithVolume(t *testing.T) {
	cli, err := client.NewEnvClient()
	if err != nil {
		t.Fatal(err)
	}
	vol := createVolume(t, cli)
	defer removeVolume(t, cli, vol.Name)
	containerConfig := container.Config{
		Image: imageName(),
		Env:   []string{"LICENSE=accept", "MQ_QMGR_NAME=qm1"},
	}
	hostConfig := container.HostConfig{
		Binds: []string{
			coverageBind(t),
			vol.Name + ":/mnt/mqm",
		},
	}
	networkingConfig := network.NetworkingConfig{}
	ctr, err := cli.ContainerCreate(context.Background(), &containerConfig, &hostConfig, &networkingConfig, t.Name())
	if err != nil {
		t.Fatal(err)
	}
	startContainer(t, cli, ctr.ID)
	// TODO: If this test gets an error waiting for readiness, the first container might not get cleaned up
	waitForReady(t, cli, ctr.ID)

	// Delete the first container
	cleanContainer(t, cli, ctr.ID)

	// Start a new container with the same volume
	ctr2, err := cli.ContainerCreate(context.Background(), &containerConfig, &hostConfig, &networkingConfig, t.Name())
	if err != nil {
		t.Fatal(err)
	}
	defer cleanContainer(t, cli, ctr2.ID)
	startContainer(t, cli, ctr2.ID)
	waitForReady(t, cli, ctr2.ID)
}

// TestNoVolumeWithRestart ensures a queue manager container can be stopped
// and restarted cleanly
func TestNoVolumeWithRestart(t *testing.T) {
	cli, err := client.NewEnvClient()
	if err != nil {
		t.Fatal(err)
	}
	containerConfig := container.Config{
		Env: []string{"LICENSE=accept", "MQ_QMGR_NAME=qm1"},
		//ExposedPorts: ports,
		ExposedPorts: nat.PortSet{
			"1414/tcp": struct{}{},
		},
	}
	id := runContainer(t, cli, &containerConfig)
	defer cleanContainer(t, cli, id)
	waitForReady(t, cli, id)
	stopContainer(t, cli, id)
	startContainer(t, cli, id)
	waitForReady(t, cli, id)
}

// TestCreateQueueManagerFail causes a failure of `crtmqm`
func TestCreateQueueManagerFail(t *testing.T) {
	cli, err := client.NewEnvClient()
	if err != nil {
		t.Fatal(err)
	}
	img, _, err := cli.ImageInspectWithRaw(context.Background(), imageName())
	oldEntrypoint := strings.Join(img.Config.Entrypoint, " ")
	containerConfig := container.Config{
		Env: []string{"LICENSE=accept", "MQ_QMGR_NAME=qm1"},
		//ExposedPorts: ports,
		ExposedPorts: nat.PortSet{
			"1414/tcp": struct{}{},
		},
		// Override the entrypoint to create the queue manager directory, but leave it empty.
		// This will cause `crtmqm` to return with an exit code of 2.
		Entrypoint: []string{"bash", "-c", "mkdir -p /mnt/mqm/data && mkdir -p /var/mqm/qmgrs/qm1 && exec " + oldEntrypoint},
	}
	id := runContainer(t, cli, &containerConfig)
	defer cleanContainer(t, cli, id)
	rc := waitForContainer(t, cli, id, 10)
	if rc != 1 {
		t.Errorf("Expected rc=1, got rc=%v", rc)
	}
}

// TestStartQueueManagerFail causes a failure of `strmqm`
func TestStartQueueManagerFail(t *testing.T) {
	cli, err := client.NewEnvClient()
	if err != nil {
		t.Fatal(err)
	}
	img, _, err := cli.ImageInspectWithRaw(context.Background(), imageName())
	oldEntrypoint := strings.Join(img.Config.Entrypoint, " ")
	containerConfig := container.Config{
		Env: []string{"LICENSE=accept", "MQ_QMGR_NAME=qm1"},
		//ExposedPorts: ports,
		ExposedPorts: nat.PortSet{
			"1414/tcp": struct{}{},
		},
		// Override the entrypoint to replace `crtmqm` with a no-op script.
		// This will cause `strmqm` to return with an exit code of 16.
		Entrypoint: []string{"bash", "-c", "echo '#!/bin/bash\n' > /opt/mqm/bin/crtmqm && exec " + oldEntrypoint},
	}
	id := runContainer(t, cli, &containerConfig)
	defer cleanContainer(t, cli, id)
	rc := waitForContainer(t, cli, id, 10)
	if rc != 1 {
		t.Errorf("Expected rc=1, got rc=%v", rc)
	}
}

// TestVolumeUnmount runs a queue manager with a volume, and then forces an
// unmount of the volume.  The health check should then fail.
// This simulates behaviour seen in some cloud environments, where network
// attached storage gets unmounted.
func TestVolumeUnmount(t *testing.T) {
	cli, err := client.NewEnvClient()
	if err != nil {
		t.Fatal(err)
	}
	vol := createVolume(t, cli)
	defer removeVolume(t, cli, vol.Name)
	containerConfig := container.Config{
		Image: imageName(),
		Env:   []string{"LICENSE=accept", "MQ_QMGR_NAME=qm1"},
	}
	hostConfig := container.HostConfig{
		// SYS_ADMIN capability is required to unmount file systems
		CapAdd: []string{
			"SYS_ADMIN",
		},
		Binds: []string{
			coverageBind(t),
			vol.Name + ":/mnt/mqm",
		},
	}
	networkingConfig := network.NetworkingConfig{}
	ctr, err := cli.ContainerCreate(context.Background(), &containerConfig, &hostConfig, &networkingConfig, t.Name())
	if err != nil {
		t.Fatal(err)
	}
	startContainer(t, cli, ctr.ID)
	defer cleanContainer(t, cli, ctr.ID)
	waitForReady(t, cli, ctr.ID)
	// Unmount the volume as root
	rc := execContainerWithExitCode(t, cli, ctr.ID, "root", []string{"umount", "-l", "-f", "/mnt/mqm"})
	if rc != 0 {
		t.Fatalf("Expected umount to work with rc=0, got %v", rc)
	}
	time.Sleep(3 * time.Second)
	rc = execContainerWithExitCode(t, cli, ctr.ID, "mqm", []string{"chkmqhealthy"})
	if rc == 0 {
		t.Errorf("Expected chkmqhealthy to fail")
		t.Logf(execContainerWithOutput(t, cli, ctr.ID, "mqm", []string{"df"}))
		t.Logf(execContainerWithOutput(t, cli, ctr.ID, "mqm", []string{"ps", "-ef"}))
	}
}

// TestZombies starts a queue manager, then causes a zombie process to be
// created, then checks that no zombies exist (runmqserver should reap them)
func TestZombies(t *testing.T) {
	cli, err := client.NewEnvClient()
	if err != nil {
		t.Fatal(err)
	}
	containerConfig := container.Config{
		Env: []string{"LICENSE=accept", "MQ_QMGR_NAME=qm1", "DEBUG=true"},
		//ExposedPorts: ports,
		ExposedPorts: nat.PortSet{
			"1414/tcp": struct{}{},
		},
	}
	id := runContainer(t, cli, &containerConfig)
	defer cleanContainer(t, cli, id)
	waitForReady(t, cli, id)
	// Kill an MQ process with children.  After it is killed, its children
	// will be adopted by PID 1, and should then be reaped when they die.
	out := execContainerWithOutput(t, cli, id, "mqm", []string{"pkill", "--signal", "kill", "-c", "amqzxma0"})
	if out == "0" {
		t.Fatalf("Expected pkill to kill a process, got %v", out)
	}
	time.Sleep(3 * time.Second)
	// Create a zombie process for up to ten seconds
	out = execContainerWithOutput(t, cli, id, "mqm", []string{"bash", "-c", "ps -lA | grep '^. Z' | wc -l"})
	count, err := strconv.Atoi(out)
	if err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("Expected zombies=0, got %v", count)
	}
}
