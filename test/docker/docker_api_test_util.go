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
	"bytes"
	"context"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/volume"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
)

func imageName() string {
	image, ok := os.LookupEnv("TEST_IMAGE")
	if !ok {
		image = "mq-devserver:latest-x86-64"
	}
	return image
}

func coverageBind(t *testing.T) string {
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	return filepath.Join(dir, "coverage") + ":/var/coverage"
}

func cleanContainer(t *testing.T, cli *client.Client, ID string) {
	i, err := cli.ContainerInspect(context.Background(), ID)
	if err == nil {
		// Log the results and continue
		t.Logf("Inspected container %v: %#v", ID, i)
	}
	t.Logf("Killing container: %v", ID)
	// Kill the container.  This allows the coverage output to be generated.
	err = cli.ContainerKill(context.Background(), ID, "SIGTERM")
	if err != nil {
		// Just log the error and continue
		t.Log(err)
	}
	//waitForContainer(t, cli, ID, 20, container.WaitConditionNotRunning)

	// TODO: This is probably no longer necessary
	time.Sleep(20 * time.Second)
	// Log the container output for any container we're about to delete
	t.Logf("Console log from container %v:\n%v", ID, inspectLogs(t, cli, ID))

	t.Logf("Removing container: %s", ID)
	opts := types.ContainerRemoveOptions{
		RemoveVolumes: true,
		Force:         true,
	}
	err = cli.ContainerRemove(context.Background(), ID, opts)
	if err != nil {
		t.Error(err)
	}
}

// runContainer creates and starts a container.  If no image is specified in
// the container config, then the image name is retrieved from the TEST_IMAGE
// environment variable.
func runContainer(t *testing.T, cli *client.Client, containerConfig *container.Config) string {
	if containerConfig.Image == "" {
		containerConfig.Image = imageName()
	}
	hostConfig := container.HostConfig{
		PortBindings: nat.PortMap{
			"1414/tcp": []nat.PortBinding{
				{
					HostIP:   "0.0.0.0",
					HostPort: "1414",
				},
			},
		},
	}
	networkingConfig := network.NetworkingConfig{}
	t.Logf("Running container (%s)", containerConfig.Image)
	ctr, err := cli.ContainerCreate(context.Background(), containerConfig, &hostConfig, &networkingConfig, t.Name())
	if err != nil {
		t.Fatal(err)
	}
	startContainer(t, cli, ctr.ID)
	return ctr.ID
}

func startContainer(t *testing.T, cli *client.Client, ID string) {
	t.Logf("Starting container: %v", ID)
	startOptions := types.ContainerStartOptions{}
	err := cli.ContainerStart(context.Background(), ID, startOptions)
	if err != nil {
		t.Fatal(err)
	}
}

func stopContainer(t *testing.T, cli *client.Client, ID string) {
	t.Logf("Stopping container: %v", ID)
	timeout := 10 * time.Second
	err := cli.ContainerStop(context.Background(), ID, &timeout) //Duration(20)*time.Second)
	if err != nil {
		t.Fatal(err)
	}
}

// waitForContainer waits until a container has exited
func waitForContainer(t *testing.T, cli *client.Client, ID string, timeout int64) int64 {
	//ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
	//defer cancel()
	rc, err := cli.ContainerWait(context.Background(), ID)
	//	err := <-errC
	if err != nil {
		t.Fatal(err)
	}
	//	wait := <-waitC
	return rc
}

// execContainer runs the specified command inside the container, returning the
// exit code and the stdout/stderr string.
func execContainer(t *testing.T, cli *client.Client, ID string, cmd []string) (int, string) {
	config := types.ExecConfig{
		User:         "mqm",
		Privileged:   false,
		Tty:          false,
		AttachStdin:  false,
		AttachStdout: true,
		AttachStderr: true,
		Detach:       false,
		Cmd:          cmd,
	}
	resp, err := cli.ContainerExecCreate(context.Background(), ID, config)
	if err != nil {
		t.Fatal(err)
	}
	hijack, err := cli.ContainerExecAttach(context.Background(), resp.ID, config)
	if err != nil {
		t.Fatal(err)
	}
	cli.ContainerExecStart(context.Background(), resp.ID, types.ExecStartCheck{
		Detach: false,
		Tty:    false,
	})
	if err != nil {
		t.Fatal(err)
	}
	inspect, err := cli.ContainerExecInspect(context.Background(), resp.ID)
	if err != nil {
		t.Fatal(err)
	}
	// TODO: For some reason, each line seems to start with an extra, random character
	buf, err := ioutil.ReadAll(hijack.Reader)
	if err != nil {
		t.Fatal(err)
	}
	hijack.Close()
	return inspect.ExitCode, string(buf)
}

func waitForReady(t *testing.T, cli *client.Client, ID string) {
	for {
		resp, err := cli.ContainerExecCreate(context.Background(), ID, types.ExecConfig{
			User:         "mqm",
			Privileged:   false,
			Tty:          false,
			AttachStdin:  false,
			AttachStdout: true,
			AttachStderr: true,
			Detach:       false,
			Cmd:          []string{"chkmqready"},
		})
		if err != nil {
			t.Fatal(err)
		}
		cli.ContainerExecStart(context.Background(), resp.ID, types.ExecStartCheck{
			Detach: false,
			Tty:    false,
		})
		if err != nil {
			t.Fatal(err)
		}
		inspect, err := cli.ContainerExecInspect(context.Background(), resp.ID)
		if err != nil {
			t.Fatal(err)
		}
		if inspect.ExitCode == 0 {
			t.Log("MQ is ready")
			return
		}
	}
}

func getIPAddress(t *testing.T, cli *client.Client, ID string) string {
	ctr, err := cli.ContainerInspect(context.Background(), ID)
	if err != nil {
		t.Fatal(err)
	}
	return ctr.NetworkSettings.IPAddress
}

func createNetwork(t *testing.T, cli *client.Client) string {
	name := "test"
	t.Logf("Creating network: %v", name)
	opts := types.NetworkCreate{}
	net, err := cli.NetworkCreate(context.Background(), name, opts)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Created network %v with ID %v", name, net.ID)
	return net.ID
}

func removeNetwork(t *testing.T, cli *client.Client, ID string) {
	t.Logf("Removing network ID: %v", ID)
	err := cli.NetworkRemove(context.Background(), ID)
	if err != nil {
		t.Fatal(err)
	}
}

func createVolume(t *testing.T, cli *client.Client) types.Volume {
	v, err := cli.VolumeCreate(context.Background(), volume.VolumesCreateBody{
		Driver:     "local",
		DriverOpts: map[string]string{},
		Labels:     map[string]string{},
		Name:       t.Name(),
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Created volume %v", t.Name())
	return v
}

func removeVolume(t *testing.T, cli *client.Client, name string) {
	t.Logf("Removing volume %v", name)
	err := cli.VolumeRemove(context.Background(), name, true)
	if err != nil {
		t.Fatal(err)
	}
}

func inspectLogs(t *testing.T, cli *client.Client, ID string) string {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	reader, err := cli.ContainerLogs(ctx, ID, types.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
	})
	if err != nil {
		log.Fatal(err)
	}

	buf := new(bytes.Buffer)
	buf.ReadFrom(reader)
	return buf.String()
}
