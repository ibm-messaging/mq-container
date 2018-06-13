/*
© Copyright IBM Corporation 2017, 2018

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
	"archive/tar"
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/volume"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/docker/go-connections/nat"
)

func imageName() string {
	image, ok := os.LookupEnv("TEST_IMAGE")
	if !ok {
		image = "mq-devserver:latest-x86-64"
	}
	return image
}

func imageNameDevJMS() string {
	image, ok := os.LookupEnv("DEV_JMS_IMAGE")
	if !ok {
		image = "mq-dev-jms-test"
	}
	return image
}

// isWSL return whether we are running in the Windows Subsystem for Linux
func isWSL(t *testing.T) bool {
	if runtime.GOOS == "linux" {
		uname, err := exec.Command("uname", "-r").Output()
		if err != nil {
			t.Fatal(err)
		}
		return strings.Contains(string(uname), "Microsoft")
	}
	return false
}

// getCwd returns the working directory, in an os-specific or UNIX form
func getCwd(t *testing.T, unixPath bool) string {
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if isWSL(t) {
		// Check if the cwd is a symlink
		dir, err = filepath.EvalSymlinks(dir)
		if err != nil {
			t.Fatal(err)
		}
		if !unixPath {
			dir = strings.Replace(dir, getWindowsRoot(true), getWindowsRoot(false), 1)
		}
	}
	return dir
}

// getWindowsRoot get the path of the root directory on Windows, in UNIX or OS-specific style
func getWindowsRoot(unixStylePath bool) string {
	if unixStylePath {
		return "/mnt/c/"
	}
	return "C:/"
}

func coverage() bool {
	cover := os.Getenv("TEST_COVER")
	if cover == "true" || cover == "1" {
		return true
	}
	return false
}

// coverageDir returns the host directory to use for code coverage data
func coverageDir(t *testing.T, unixStylePath bool) string {
	return filepath.Join(getCwd(t, unixStylePath), "coverage")
}

// coverageBind returns a string to use to add a bind-mounted directory for code coverage data
func coverageBind(t *testing.T) string {
	return coverageDir(t, false) + ":/var/coverage"
}

// getTempDir get the path of the tmp directory, in UNIX or OS-specific style
func getTempDir(t *testing.T, unixStylePath bool) string {
	if isWSL(t) {
		return getWindowsRoot(unixStylePath) + "Temp/"
	}
	return "/tmp/"
}

// terminationLogUnixPath returns the name of the file to use for the termination log message, with a UNIX path
func terminationLogUnixPath(t *testing.T) string {
	// Warning: this directory must be accessible to the Docker daemon,
	// in order to enable the bind mount
	return getTempDir(t, true) + t.Name() + "-termination-log"
}

// terminationLogOSPath returns the name of the file to use for the termination log message, with an OS specific path
func terminationLogOSPath(t *testing.T) string {
	// Warning: this directory must be accessible to the Docker daemon,
	// in order to enable the bind mount
	return getTempDir(t, false) + t.Name() + "-termination-log"
}

// terminationBind returns a string to use to bind-mount a termination log file.
// This is done using a bind, because you can't copy files from /dev out of the container.
func terminationBind(t *testing.T) string {
	n := terminationLogUnixPath(t)
	// Remove it if it already exists
	os.Remove(n)
	// Create the empty file
	f, err := os.OpenFile(n, os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		t.Fatal(err)
	}
	f.Close()
	return terminationLogOSPath(t) + ":/dev/termination-log"
}

// terminationMessage return the termination message, or an empty string if not set
func terminationMessage(t *testing.T) string {
	b, err := ioutil.ReadFile(terminationLogUnixPath(t))
	if err != nil {
		t.Log(err)
	}
	return string(b)
}

func expectTerminationMessage(t *testing.T) {
	m := terminationMessage(t)
	if m == "" {
		t.Error("Expected termination message to be set")
	}
}

func cleanContainer(t *testing.T, cli *client.Client, ID string) {
	i, err := cli.ContainerInspect(context.Background(), ID)
	if err == nil {
		// Log the results and continue
		t.Logf("Inspected container %v: %#v", ID, i)
		s, err := json.MarshalIndent(i, "", "    ")
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("Inspected container %v: %v", ID, string(s))
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
	os.Rename(filepath.Join(coverageDir(t, true), "container.cov"), filepath.Join(coverageDir(t, true), t.Name()+".cov"))
	// Log the container output for any container we're about to delete
	t.Logf("Console log from container %v:\n%v", ID, inspectTextLogs(t, cli, ID))

	m := terminationMessage(t)
	if m != "" {
		t.Logf("Termination message: %v", m)
	}
	os.Remove(terminationLogUnixPath(t))

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

// runContainerWithPorts creates and starts a container, exposing the specified ports on the host.
// If no image is specified in the container config, then the image name is retrieved from the TEST_IMAGE
// environment variable.
func runContainerWithPorts(t *testing.T, cli *client.Client, containerConfig *container.Config, ports []int) string {
	if containerConfig.Image == "" {
		containerConfig.Image = imageName()
	}
	// if coverage
	containerConfig.Env = append(containerConfig.Env, "COVERAGE_FILE="+t.Name()+".cov")
	containerConfig.Env = append(containerConfig.Env, "EXIT_CODE_FILE="+getExitCodeFilename(t))
	hostConfig := container.HostConfig{
		Binds: []string{
			coverageBind(t),
			terminationBind(t),
		},
		PortBindings: nat.PortMap{},
	}
	for _, p := range ports {
		port := nat.Port(fmt.Sprintf("%v/tcp", p))
		hostConfig.PortBindings[port] = []nat.PortBinding{
			{
				HostIP: "0.0.0.0",
			},
		}
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

// runContainer creates and starts a container.  If no image is specified in
// the container config, then the image name is retrieved from the TEST_IMAGE
// environment variable.
func runContainer(t *testing.T, cli *client.Client, containerConfig *container.Config) string {
	return runContainerWithPorts(t, cli, containerConfig, nil)
}

func runContainerOneShot(t *testing.T, cli *client.Client, command ...string) (int64, string) {
	containerConfig := container.Config{
		Entrypoint: command,
	}
	id := runContainer(t, cli, &containerConfig)
	defer cleanContainer(t, cli, id)
	return waitForContainer(t, cli, id, 10), inspectLogs(t, cli, id)
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

func getExitCodeFilename(t *testing.T) string {
	return t.Name() + "ExitCode"
}

func getCoverageExitCode(t *testing.T, orig int64) int64 {
	f := filepath.Join(coverageDir(t, true), getExitCodeFilename(t))
	_, err := os.Stat(f)
	if err != nil {
		t.Log(err)
		return orig
	}
	// Remove the file, ready for the next test
	defer os.Remove(f)
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
	rc, err := cli.ContainerWait(context.Background(), ID)

	if coverage() {
		// COVERAGE: When running coverage, the exit code is written to a file,
		// to allow the coverage to be generated (which doesn't happen for non-zero
		// exit codes)
		rc = getCoverageExitCode(t, rc)
	}

	if err != nil {
		t.Fatal(err)
	}
	return rc
}

// execContainer runs a command in a running container, and returns the exit code and output
func execContainer(t *testing.T, cli *client.Client, ID string, user string, cmd []string) (int, string) {
rerun:
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
	// Wait for the command to finish
	var exitcode int
	for {
		inspect, err := cli.ContainerExecInspect(context.Background(), resp.ID)
		if err != nil {
			t.Fatal(err)
		}
		if !inspect.Running {
			exitcode = inspect.ExitCode
			break
		}
	}
	buf := new(bytes.Buffer)
	// Each output line has a header, which needs to be removed
	_, err = stdcopy.StdCopy(buf, buf, hijack.Reader)
	if err != nil {
		t.Fatal(err)
	}

	outputStr := strings.TrimSpace(buf.String())

	// Before we go let's just double check it did actually run because sometimes we get a "Exec command already running error"
	alreadyRunningErr := regexp.MustCompile("Error: Exec command .* is already running")
	if alreadyRunningErr.MatchString(outputStr) {
		time.Sleep(1 * time.Second)
		goto rerun
	}

	return exitcode, outputStr
}

func waitForReady(t *testing.T, cli *client.Client, ID string) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	for {
		select {
		case <-time.After(1 * time.Second):
			rc, _ := execContainer(t, cli, ID, "mqm", []string{"chkmqready"})
			if rc == 0 {
				t.Log("MQ is ready")
				return
			}
		case <-ctx.Done():
			t.Fatal("Timed out waiting for container to become ready")
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

func inspectTextLogs(t *testing.T, cli *client.Client, ID string) string {
	jsonLogs := inspectLogs(t, cli, ID)
	scanner := bufio.NewScanner(strings.NewReader(jsonLogs))
	b := make([]byte, 64*1024)
	buf := bytes.NewBuffer(b)
	for scanner.Scan() {
		text := scanner.Text()
		if strings.HasPrefix(text, "{") {
			var e map[string]interface{}
			json.Unmarshal([]byte(text), &e)
			fmt.Fprintf(buf, "{\"message\": \"%v\"}\n", e["message"])
		} else {
			fmt.Fprintln(buf, text)
		}
	}
	err := scanner.Err()
	if err != nil {
		t.Fatal(err)
	}
	return buf.String()
}

func inspectLogs(t *testing.T, cli *client.Client, ID string) string {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	reader, err := cli.ContainerLogs(ctx, ID, types.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	buf := new(bytes.Buffer)
	// Each output line has a header, which needs to be removed
	_, err = stdcopy.StdCopy(buf, buf, reader)
	if err != nil {
		t.Fatal(err)
	}
	return buf.String()
}

// generateTAR creates a TAR-formatted []byte, with the specified files included.
func generateTAR(t *testing.T, files []struct{ Name, Body string }) []byte {
	buf := new(bytes.Buffer)
	tw := tar.NewWriter(buf)
	for _, file := range files {
		hdr := &tar.Header{
			Name: file.Name,
			Mode: 0600,
			Size: int64(len(file.Body)),
		}
		err := tw.WriteHeader(hdr)
		if err != nil {
			t.Fatal(err)
		}
		_, err = tw.Write([]byte(file.Body))
		if err != nil {
			t.Fatal(err)
		}
	}
	err := tw.Close()
	if err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// createImage creates a new Docker image with the specified files included.
func createImage(t *testing.T, cli *client.Client, files []struct{ Name, Body string }) string {
	r := bytes.NewReader(generateTAR(t, files))
	tag := strings.ToLower(t.Name())
	buildOptions := types.ImageBuildOptions{
		Context: r,
		Tags:    []string{tag},
	}
	resp, err := cli.ImageBuild(context.Background(), r, buildOptions)
	if err != nil {
		t.Fatal(err)
	}
	// resp (ImageBuildResponse) contains a series of JSON messages
	dec := json.NewDecoder(resp.Body)
	for {
		m := jsonmessage.JSONMessage{}
		err := dec.Decode(&m)
		if m.Error != nil {
			t.Fatal(m.ErrorMessage)
		}
		t.Log(strings.TrimSpace(m.Stream))
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
	}
	return tag
}

// deleteImage deletes a Docker image
func deleteImage(t *testing.T, cli *client.Client, id string) {
	cli.ImageRemove(context.Background(), id, types.ImageRemoveOptions{
		Force: true,
	})
}

func copyFromContainer(t *testing.T, cli *client.Client, id string, file string) []byte {
	reader, _, err := cli.CopyFromContainer(context.Background(), id, file)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	b, err := ioutil.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func getPort(t *testing.T, cli *client.Client, ID string, port int) string {
	i, err := cli.ContainerInspect(context.Background(), ID)
	if err != nil {
		t.Fatal(err)
	}
	portNat := nat.Port(fmt.Sprintf("%d/tcp", port))
	return i.NetworkSettings.Ports[portNat][0].HostPort
}

func countLines(t *testing.T, r io.Reader) int {
	scanner := bufio.NewScanner(r)
	count := 0
	for scanner.Scan() {
		count++
	}
	err := scanner.Err()
	if err != nil {
		t.Fatal(err)
	}
	return count
}

func countTarLines(t *testing.T, b []byte) int {
	r := bytes.NewReader(b)
	tr := tar.NewReader(r)
	total := 0
	for {
		_, err := tr.Next()
		if err == io.EOF {
			// End of TAR
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		total += countLines(t, tr)
	}
	return total
}
