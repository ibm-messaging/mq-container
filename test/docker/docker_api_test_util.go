/*
© Copyright IBM Corporation 2017, 2019

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

type containerDetails struct {
	ID      string
	Name    string
	Image   string
	Path    string
	Args    []string
	CapAdd  []string
	CapDrop []string
	User    string
	Env     []string
}

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

// baseImage returns the ID of the underlying base image (e.g. "ubuntu" or "rhel")
func baseImage(t *testing.T, cli *client.Client) string {
	rc, out := runContainerOneShot(t, cli, "grep", "^ID=", "/etc/os-release")
	if rc != 0 {
		t.Fatal("Couldn't determine base image")
	}
	s := strings.Split(out, "=")
	if len(s) < 2 {
		t.Fatal("Couldn't determine base image string")
	}
	return s[1]
}

// devImage returns true if the image under test is a developer image,
// determined by use of the MQ_ADMIN_PASSWORD environment variable
func devImage(t *testing.T, cli *client.Client) bool {
	rc, _ := runContainerOneShot(t, cli, "printenv", "MQ_ADMIN_PASSWORD")
	if rc == 0 {
		return true
	}
	return false
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

// terminationMessage return the termination message, or an empty string if not set
func terminationMessage(t *testing.T, cli *client.Client, ID string) string {
	r, _, err := cli.CopyFromContainer(context.Background(), ID, "/run/termination-log")
	if err != nil {
		t.Log(err)
		return ""
	}
	b, err := ioutil.ReadAll(r)
	tr := tar.NewReader(bytes.NewReader(b))
	_, err = tr.Next()
	if err != nil {
		t.Log(err)
		return ""
	}
	// read the complete content of the file h.Name into the bs []byte
	content, err := ioutil.ReadAll(tr)
	if err != nil {
		t.Log(err)
		return ""
	}
	return string(content)
}

func expectTerminationMessage(t *testing.T, cli *client.Client, ID string) {
	m := terminationMessage(t, cli, ID)
	if m == "" {
		t.Error("Expected termination message to be set")
	}
}

// logContainerDetails logs selected details about the container
func logContainerDetails(t *testing.T, cli *client.Client, ID string) {
	i, err := cli.ContainerInspect(context.Background(), ID)
	if err == nil {
		d := containerDetails{
			ID:      ID,
			Name:    i.Name,
			Image:   i.Image,
			Path:    i.Path,
			Args:    i.Args,
			CapAdd:  i.HostConfig.CapAdd,
			CapDrop: i.HostConfig.CapDrop,
			User:    i.Config.User,
			Env:     i.Config.Env,
		}
		// If you need more details, you can always just run `json.MarshalIndent(i, "", "    ")` to see everything.
		t.Logf("Container details: %+v", d)
	}
}

func cleanContainerQuiet(t *testing.T, cli *client.Client, ID string) {
	timeout := 10 * time.Second
	err := cli.ContainerStop(context.Background(), ID, &timeout)
	if err != nil {
		// Just log the error and continue
		t.Log(err)
	}
	opts := types.ContainerRemoveOptions{
		RemoveVolumes: true,
		Force:         true,
	}
	err = cli.ContainerRemove(context.Background(), ID, opts)
	if err != nil {
		t.Error(err)
	}
}

func cleanContainer(t *testing.T, cli *client.Client, ID string) {
	logContainerDetails(t, cli, ID)
	t.Logf("Stopping container: %v", ID)
	timeout := 10 * time.Second
	// Stop the container.  This allows the coverage output to be generated.
	err := cli.ContainerStop(context.Background(), ID, &timeout)
	if err != nil {
		// Just log the error and continue
		t.Log(err)
	}
	t.Log("Container stopped")

	// If a code coverage file has been generated, then rename it to match the test name
	os.Rename(filepath.Join(coverageDir(t, true), "container.cov"), filepath.Join(coverageDir(t, true), t.Name()+".cov"))
	// Log the container output for any container we're about to delete
	t.Logf("Console log from container %v:\n%v", ID, inspectTextLogs(t, cli, ID))

	m := terminationMessage(t, cli, ID)
	if m != "" {
		t.Logf("Termination message: %v", m)
	}

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
	// Always run as the "mqm" user, unless the test has specified otherwise
	if containerConfig.User == "" {
		containerConfig.User = "mqm"
	}
	// if coverage
	containerConfig.Env = append(containerConfig.Env, "COVERAGE_FILE="+t.Name()+".cov")
	containerConfig.Env = append(containerConfig.Env, "EXIT_CODE_FILE="+getExitCodeFilename(t))
	hostConfig := container.HostConfig{
		Binds: []string{
			coverageBind(t),
		},
		PortBindings: nat.PortMap{},
		CapDrop: []string{
			"ALL",
		},
	}
	if devImage(t, cli) {
		t.Logf("Detected MQ Advanced for Developers image — adding extra Linux capabilities to container")
		hostConfig.CapAdd = []string{
			"CHOWN",
			"SETUID",
			"SETGID",
			"AUDIT_WRITE",
		}
		// Only needed for a RHEL-based image
		if baseImage(t, cli) != "ubuntu" {
			hostConfig.CapAdd = append(hostConfig.CapAdd, "DAC_OVERRIDE")
		}
	} else {
		t.Logf("Detected MQ Advanced image - dropping all capabilities")
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

// runContainerOneShot runs a container with a custom entrypoint, as the root
// user and with default capabilities
func runContainerOneShot(t *testing.T, cli *client.Client, command ...string) (int64, string) {
	containerConfig := container.Config{
		Entrypoint: command,
		User:       "root",
		Image:      imageName(),
	}
	hostConfig := container.HostConfig{}
	networkingConfig := network.NetworkingConfig{}
	t.Logf("Running one shot container (%s): %v", containerConfig.Image, command)
	ctr, err := cli.ContainerCreate(context.Background(), &containerConfig, &hostConfig, &networkingConfig, t.Name()+"OneShot")
	if err != nil {
		t.Fatal(err)
	}
	startOptions := types.ContainerStartOptions{}
	err = cli.ContainerStart(context.Background(), ctr.ID, startOptions)
	if err != nil {
		t.Fatal(err)
	}
	defer cleanContainerQuiet(t, cli, ctr.ID)
	rc := waitForContainer(t, cli, ctr.ID, 20*time.Second)
	out := inspectLogs(t, cli, ctr.ID)
	t.Logf("One shot container finished with rc=%v, output=%v", rc, out)
	return rc, out
}

// runContainerOneShot runs a container with a custom entrypoint, as the root
// user, with default capabilities, and a volume mounted
func runContainerOneShotWithVolume(t *testing.T, cli *client.Client, bind string, command ...string) (int64, string) {
	containerConfig := container.Config{
		Entrypoint: command,
		User:       "root",
		Image:      imageName(),
	}
	hostConfig := container.HostConfig{
		Binds: []string{
			bind,
		},
	}
	networkingConfig := network.NetworkingConfig{}
	t.Logf("Running one shot container with volume (%s): %v", containerConfig.Image, command)
	ctr, err := cli.ContainerCreate(context.Background(), &containerConfig, &hostConfig, &networkingConfig, t.Name()+"OneShotVolume")
	if err != nil {
		t.Fatal(err)
	}
	startOptions := types.ContainerStartOptions{}
	err = cli.ContainerStart(context.Background(), ctr.ID, startOptions)
	if err != nil {
		t.Fatal(err)
	}
	defer cleanContainerQuiet(t, cli, ctr.ID)
	rc := waitForContainer(t, cli, ctr.ID, 20*time.Second)
	out := inspectLogs(t, cli, ctr.ID)
	t.Logf("One shot container finished with rc=%v, output=%v", rc, out)
	return rc, out
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

func killContainer(t *testing.T, cli *client.Client, ID string, signal string) {
	t.Logf("Killing container: %v", ID)
	err := cli.ContainerKill(context.Background(), ID, signal)
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
func waitForContainer(t *testing.T, cli *client.Client, ID string, timeout time.Duration) int64 {
	c, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	rc, err := cli.ContainerWait(c, ID)
	if err != nil {
		t.Fatal(err)
	}
	if coverage() {
		// COVERAGE: When running coverage, the exit code is written to a file,
		// to allow the coverage to be generated (which doesn't happen for non-zero
		// exit codes)
		rc = getCoverageExitCode(t, rc)
	}
	return rc
}

// execContainer runs a command in a running container, and returns the exit code and output
func execContainer(t *testing.T, cli *client.Client, ID string, user string, cmd []string) (int, string) {
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
	defer hijack.Close()
	time.Sleep(time.Millisecond * 10)
	err = cli.ContainerExecStart(context.Background(), resp.ID, types.ExecStartCheck{
		Detach: false,
		Tty:    false,
	})
	if err != nil {
		t.Fatal(err)
	}
	// Wait for the command to finish
	var exitcode int
	var outputStr string
	for {
		inspect, err := cli.ContainerExecInspect(context.Background(), resp.ID)
		if err != nil {
			t.Fatal(err)
		}
		if inspect.Running {
			continue
		}

		exitcode = inspect.ExitCode
		buf := new(bytes.Buffer)
		// Each output line has a header, which needs to be removed
		_, err = stdcopy.StdCopy(buf, buf, hijack.Reader)
		if err != nil {
			t.Fatal(err)
		}

		outputStr = strings.TrimSpace(buf.String())

		/* Commented out on 14/06/2018 as it might not be needed after adding
		 * pause between ContainerExecAttach and ContainerExecStart.
		 * TODO If intermittent failures do not occur, remove and refactor.
		 *
		 *   // Before we go let's just double check it did actually finish running
		 *   // because sometimes we get a "Exec command already running error"
		 *   alreadyRunningErr := regexp.MustCompile("Error: Exec command .* is already running")
		 *   if alreadyRunningErr.MatchString(outputStr) {
		 *   	continue
		 *   }
		 */
		break
	}

	return exitcode, outputStr
}

func waitForReady(t *testing.T, cli *client.Client, ID string) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
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

func createVolume(t *testing.T, cli *client.Client, name string) types.Volume {
	v, err := cli.VolumeCreate(context.Background(), volume.VolumesCreateBody{
		Driver:     "local",
		DriverOpts: map[string]string{},
		Labels:     map[string]string{},
		Name:       name,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Created volume %v", v.Name)
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
