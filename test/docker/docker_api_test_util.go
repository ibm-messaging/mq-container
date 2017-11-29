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
	"strconv"
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

// coverageDir returns the host directory to use for code coverage data
func coverageDir(t *testing.T) string {
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	return filepath.Join(dir, "coverage")
}

// coverageBind returns a string to use to add a bind-mounted directory for code coverage data
func coverageBind(t *testing.T) string {
	return coverageDir(t) + ":/var/coverage"
}

func cleanContainer(t *testing.T, cli *client.Client, ID string) {
	i, err := cli.ContainerInspect(context.Background(), ID)
	if err == nil {
		// Log the results and continue
		t.Logf("Inspected container %v: %#v", ID, i)
	}
	t.Logf("Stopping container: %v", ID)
	timeout := 10 * time.Second
	// Stop the container.  This allows the coverage output to be generated.
	err = cli.ContainerStop(context.Background(), ID, &timeout)
	if err != nil {
		// Just log the error and continue
		t.Log(err)
	}
	t.Log("Container stopped")
	// If a code coverage file has been generated, then rename it to match the test name
	os.Rename(filepath.Join(coverageDir(t), "container.cov"), filepath.Join(coverageDir(t), t.Name()+".cov"))
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
	// if coverage
	containerConfig.Env = append(containerConfig.Env, "COVERAGE_FILE="+t.Name()+".cov")
	hostConfig := container.HostConfig{
		PortBindings: nat.PortMap{
			"1414/tcp": []nat.PortBinding{
				{
					HostIP:   "0.0.0.0",
					HostPort: "1414",
				},
			},
		},
		Binds: []string{
			coverageBind(t),
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

func getCoverageExitCode(t *testing.T, orig int64) int64 {
	f := filepath.Join(coverageDir(t), "exitCode")
	_, err := os.Stat(f)
	if err != nil {
		t.Log(err)
		return orig
	}
	// Remove the file, ready for the next test
	//defer os.Remove(f)
	buf, err := ioutil.ReadFile(f)
	if err != nil {
		t.Log(err)
		return orig
	}
	rc, err := strconv.Atoi(string(buf))
	if err != nil {
		t.Log(err)
		return orig
	}
	t.Logf("Retrieved exit code %v from file", rc)
	return int64(rc)
}

// waitForContainer waits until a container has exited
func waitForContainer(t *testing.T, cli *client.Client, ID string, timeout int64) int64 {
	//ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
	//defer cancel()
	rc, err := cli.ContainerWait(context.Background(), ID)

	// COVERAGE: When running coverage, the exit code is written to a file,
	// to allow the coverage to be generated (which doesn't happen for non-zero
	// exit codes)
	rc = getCoverageExitCode(t, rc)

	//	err := <-errC
	if err != nil {
		t.Fatal(err)
	}
	//	wait := <-waitC
	return rc
}

// execContainerWithExitCode runs a command in a running container, and returns the exit code
// Note: due to a bug in Docker/Moby code, you always get an exit code of 0 if you attach to the
// container to get output.  This is why these are two separate commands.
func execContainerWithExitCode(t *testing.T, cli *client.Client, ID string, user string, cmd []string) int {
	config := types.ExecConfig{
		User:        user,
		Privileged:  false,
		Tty:         false,
		AttachStdin: false,
		// Note that you still need to attach stdout/stderr, even though they're not wanted
		AttachStdout: true,
		AttachStderr: true,
		Detach:       false,
		Cmd:          cmd,
	}
	resp, err := cli.ContainerExecCreate(context.Background(), ID, config)
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
	return inspect.ExitCode
}

// execContainerWithOutput runs a command in a running container, and returns the output from stdout/stderr
// Note: due to a bug in Docker/Moby code, you always get an exit code of 0 if you attach to the
// container to get output.  This is why these are two separate commands.
func execContainerWithOutput(t *testing.T, cli *client.Client, ID string, user string, cmd []string) string {
	config := types.ExecConfig{
		User:         user,
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
	// TODO: For some reason, each line seems to start with an extra, random character
	buf, err := ioutil.ReadAll(hijack.Reader)
	if err != nil {
		t.Fatal(err)
	}
	hijack.Close()
	return string(buf)
}

func waitForReady(t *testing.T, cli *client.Client, ID string) {
	for {
		rc := execContainerWithExitCode(t, cli, ID, "mqm", []string{"chkmqready"})
		if rc == 0 {
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
