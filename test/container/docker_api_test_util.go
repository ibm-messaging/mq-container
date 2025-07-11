/*
© Copyright IBM Corporation 2017, 2024

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
	"errors"
	"fmt"
	"io"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	ce "github.com/ibm-messaging/mq-container/test/container/containerengine"
	"github.com/ibm-messaging/mq-container/test/container/pathutils"
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

// baseImage returns the ID of the underlying base image (e.g. "ubuntu" or "rhel")
func baseImage(t *testing.T, cli ce.ContainerInterface) string {
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
func devImage(t *testing.T, cli ce.ContainerInterface) bool {
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

// isARM returns whether we are running an arm64 MacOS machine
func isARM(t *testing.T) bool {
	return runtime.GOARCH == "arm64"
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
	return pathutils.CleanPath(getCwd(t, unixStylePath), "coverage")
}

// coverageBind returns a string to use to add a bind-mounted directory for code coverage data
func coverageBind(t *testing.T) string {
	return coverageDir(t, false) + ":/var/coverage"
}

func addCoverageBindIfAvailable(t *testing.T, cfg *ce.ContainerHostConfig) {
	info, err := os.Stat(coverageDir(t, false))
	if err != nil || !info.IsDir() {
		return
	}
	cfg.Binds = append(cfg.Binds, coverageBind(t))
}

// getTempDir get the path of the tmp directory, in UNIX or OS-specific style
func getTempDir(t *testing.T, unixStylePath bool) string {
	if isWSL(t) {
		return getWindowsRoot(unixStylePath) + "Temp/"
	}
	return "/tmp/"
}

// terminationMessage return the termination message, or an empty string if not set
func terminationMessage(t *testing.T, cli ce.ContainerInterface, ID string) string {
	r, err := cli.CopyFromContainer(ID, "/run/termination-log")
	if err != nil {
		t.Log(err)
		t.Log(string(r))
		return ""
	}
	return string(r)
}

func expectTerminationMessage(t *testing.T, cli ce.ContainerInterface, ID string) {
	m := terminationMessage(t, cli, ID)
	if m == "" {
		t.Error("Expected termination message to be set")
	}
}

// logContainerDetails logs selected details about the container
func logContainerDetails(t *testing.T, cli ce.ContainerInterface, ID string) {
	i, err := cli.ContainerInspect(ID)
	if err == nil {
		d := ce.ContainerDetailsLogging{
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

func cleanContainerQuiet(t *testing.T, cli ce.ContainerInterface, ID string) {
	timeout := 10 * time.Second
	err := cli.ContainerStop(ID, &timeout)
	if err != nil {
		// Just log the error and continue
		t.Log(err)
	}
	opts := ce.ContainerRemoveOptions{
		RemoveVolumes: true,
		Force:         true,
	}
	err = cli.ContainerRemove(ID, opts)
	if err != nil {
		t.Error(err)
	}
}

func cleanContainer(t *testing.T, cli ce.ContainerInterface, ID string, ignoreFDCs bool) {
	logContainerDetails(t, cli, ID)

	if !ignoreFDCs {
		summaries, err := getQMFDCSummaries(cli, ID)
		if err != nil {
			t.Logf("WARN - Not checking for FDCs: failed to get FDC summaries: %v", err)
		} else {
			if len(summaries) > 0 {
				t.Errorf("%d FDC(s) found in the queue manager!", len(summaries))
				for _, summary := range summaries {
					t.Log("\n" + summary)
				}
			} else {
				t.Log("No FDCs found in the queue manager")
			}
		}
	}

	t.Logf("Stopping container: %v", ID)
	timeout := 10 * time.Second
	// Stop the container.  This allows the coverage output to be generated.
	err := cli.ContainerStop(ID, &timeout)
	if err != nil {
		// Just log the error and continue
		t.Log(err)
	}
	t.Log("Container stopped")

	// If a code coverage file has been generated, then rename it to match the test name
	os.Rename(pathutils.CleanPath(coverageDir(t, true), "container.cov"), pathutils.CleanPath(coverageDir(t, true), t.Name()+".cov"))
	// Log the container output for any container we're about to delete
	t.Logf("Console log from container %v:\n%v", ID, inspectTextLogs(t, cli, ID))

	m := terminationMessage(t, cli, ID)
	if m != "" {
		t.Logf("Termination message: %v", m)
	}

	t.Logf("Removing container: %s", ID)
	opts := ce.ContainerRemoveOptions{
		RemoveVolumes: true,
		Force:         true,
	}
	err = cli.ContainerRemove(ID, opts)
	if err != nil {
		t.Error(err)
	}
}

// getQMFDCSummaries returns a slice of the summary blocks from each of the .FDC files in /var/mqm/errors
func getQMFDCSummaries(cli ce.ContainerInterface, ID string) ([]string, error) {
	summaries := []string{}
	// Get a list of any FDC files
	tmpDir, err := os.MkdirTemp("", "tmp-"+ID[:8]+"-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tmpDir)
	err = cli.CopyFromContainerToDir(ID, "/var/mqm/errors/", tmpDir+"/")
	if err != nil {
		return nil, fmt.Errorf("failed to copy errors directory from the container: %v", err)
	}

	tmpDir += "/errors/"
	entries, err := os.ReadDir(tmpDir)
	if err != nil {
		return nil, fmt.Errorf("failed to list errors directory: %v", err)
	}

	fdcPaths := []string{}
	for _, entry := range entries {
		split := strings.Split(entry.Name(), ".")
		if split[len(split)-1] == "FDC" {
			fdcPaths = append(fdcPaths, tmpDir+"/"+entry.Name())
		}
	}

	var fileOpenErr error
	for _, filePath := range fdcPaths {
		// For each FDC file, read the file until we grab the summary block
		file, err := os.Open(filePath)
		if err != nil {
			summaries = append(summaries, fmt.Sprintf("failed to get FDC summary for %s: %s", filePath, err))
			fileOpenErr = errors.New("failed to open one or more FDC files, check logs for specific errors")
			continue
		}
		defer file.Close()

		fdcSummary := ""
		summaryOpeningFound := false
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := scanner.Text() + "\n"
			fdcSummary += line
			// Check for the end of the summary block
			if strings.Contains(line, "--+") {
				// the pattern appears at the start and end of the summary so
				// we need to make sure it is the second instance
				if summaryOpeningFound {
					break
				} else {
					summaryOpeningFound = true
				}
			}
		}
		summaries = append(summaries, fdcSummary)
	}

	return summaries, fileOpenErr
}

func generateRandomUID() string {
	rand.Seed(time.Now().UnixNano())
	min := 1000
	max := 9999
	return fmt.Sprint(rand.Intn(max-min) + min)
}

// getDefaultHostConfig creates a HostConfig and populates it with the defaults used in testing
func getDefaultHostConfig(t *testing.T, cli ce.ContainerInterface) *ce.ContainerHostConfig {
	hostConfig := ce.ContainerHostConfig{
		PortBindings: []ce.PortBinding{},
		CapDrop: []string{
			"ALL",
		},
		Privileged: false,
	}
	if coverage() {
		hostConfig.Binds = append(hostConfig.Binds, coverageBind(t))
	}
	if devImage(t, cli) {
		// Only needed for a RHEL-based image
		if baseImage(t, cli) != "ubuntu" {
			hostConfig.CapAdd = append(hostConfig.CapAdd, "DAC_OVERRIDE")
		}
	} else {
		t.Logf("Detected MQ Advanced image - dropping all capabilities")
	}
	return &hostConfig
}

// runContainerWithHostConfig creates and starts a container, using the supplied HostConfig.
// Note that a default HostConfig can be created using getDefaultHostConfig.
func runContainerWithHostConfig(t *testing.T, cli ce.ContainerInterface, containerConfig *ce.ContainerConfig, hostConfig *ce.ContainerHostConfig) string {
	if containerConfig.Image == "" {
		containerConfig.Image = imageName()
	}
	// Always run as a random user, unless the test has specified otherwise
	if containerConfig.User == "" {
		containerConfig.User = generateRandomUID()
	}
	if coverage() {
		containerConfig.Env = append(containerConfig.Env, "COVERAGE_FILE="+t.Name()+".cov")
		containerConfig.Env = append(containerConfig.Env, "EXIT_CODE_FILE="+getExitCodeFilename(t))
	}
	networkingConfig := ce.ContainerNetworkSettings{}
	t.Logf("Running container (%s)", containerConfig.Image)
	ID, err := cli.ContainerCreate(containerConfig, hostConfig, &networkingConfig, t.Name())
	if err != nil {
		t.Fatal(err)
	}
	startContainer(t, cli, ID)
	return ID
}

// runContainerWithAllConfig creates and starts a container, using the supplied ContainerConfig, HostConfig,
// NetworkingConfig, and container name (or the value of t.Name if containerName="").
func runContainerWithAllConfig(t *testing.T, cli ce.ContainerInterface, containerConfig *ce.ContainerConfig, hostConfig *ce.ContainerHostConfig, networkingConfig *ce.ContainerNetworkSettings, containerName string) string {
	if containerName == "" {
		containerName = t.Name()
	}
	if containerConfig.Image == "" {
		containerConfig.Image = imageName()
	}
	// Always run as a random user, unless the test has specified otherwise
	if containerConfig.User == "" {
		containerConfig.User = generateRandomUID()
	}
	if coverage() {
		containerConfig.Env = append(containerConfig.Env, "COVERAGE_FILE="+t.Name()+".cov")
		containerConfig.Env = append(containerConfig.Env, "EXIT_CODE_FILE="+getExitCodeFilename(t))
	}
	t.Logf("Running container (%s)", containerConfig.Image)
	ID, err := cli.ContainerCreate(containerConfig, hostConfig, networkingConfig, containerName)
	if err != nil {
		t.Fatal(err)
	}
	startContainer(t, cli, ID)
	return ID
}

// runContainerWithPorts creates and starts a container, exposing the specified ports on the host.
// If no image is specified in the container config, then the image name is retrieved from the TEST_IMAGE
// environment variable.
func runContainerWithPorts(t *testing.T, cli ce.ContainerInterface, containerConfig *ce.ContainerConfig, ports []int) string {
	hostConfig := getDefaultHostConfig(t, cli)
	var binding ce.PortBinding
	for _, p := range ports {
		port := fmt.Sprintf("%v/tcp", p)
		binding = ce.PortBinding{
			ContainerPort: port,
			HostIP:        "0.0.0.0",
		}
		hostConfig.PortBindings = append(hostConfig.PortBindings, binding)
	}
	return runContainerWithHostConfig(t, cli, containerConfig, hostConfig)
}

// runContainer creates and starts a container.  If no image is specified in
// the container config, then the image name is retrieved from the TEST_IMAGE
// environment variable.
func runContainer(t *testing.T, cli ce.ContainerInterface, containerConfig *ce.ContainerConfig) string {
	return runContainerWithPorts(t, cli, containerConfig, nil)
}

// runContainerOneShot runs a container with a custom entrypoint, as the root
// user and with default capabilities
func runContainerOneShot(t *testing.T, cli ce.ContainerInterface, command ...string) (int64, string) {
	containerConfig := ce.ContainerConfig{
		Entrypoint: command,
		User:       "root",
		Image:      imageName(),
	}
	hostConfig := ce.ContainerHostConfig{}
	networkingConfig := ce.ContainerNetworkSettings{}
	t.Logf("Running one shot container (%s): %v", containerConfig.Image, command)
	ID, err := cli.ContainerCreate(&containerConfig, &hostConfig, &networkingConfig, t.Name()+"OneShot")
	if err != nil {
		t.Fatal(err)
	}
	startOptions := ce.ContainerStartOptions{}
	err = cli.ContainerStart(ID, startOptions)
	if err != nil {
		t.Fatal(err)
	}
	defer cleanContainerQuiet(t, cli, ID)
	rc := waitForContainer(t, cli, ID, 20*time.Second)
	out := inspectLogs(t, cli, ID)
	t.Logf("One shot container finished with rc=%v, output=%v", rc, out)
	return rc, out
}

// runContainerOneShot runs a container with a custom entrypoint, as the root
// user, with default capabilities, and a volume mounted
func runContainerOneShotWithVolume(t *testing.T, cli ce.ContainerInterface, bind string, command ...string) (int64, string) {
	containerConfig := ce.ContainerConfig{
		Entrypoint: command,
		User:       "root",
		Image:      imageName(),
	}
	hostConfig := ce.ContainerHostConfig{
		Binds: []string{
			bind,
		},
	}
	networkingConfig := ce.ContainerNetworkSettings{}
	t.Logf("Running one shot container with volume (%s): %v", containerConfig.Image, command)
	ID, err := cli.ContainerCreate(&containerConfig, &hostConfig, &networkingConfig, t.Name()+"OneShotVolume")
	if err != nil {
		t.Fatal(err)
	}
	startOptions := ce.ContainerStartOptions{}
	err = cli.ContainerStart(ID, startOptions)
	if err != nil {
		t.Fatal(err)
	}
	defer cleanContainerQuiet(t, cli, ID)
	rc := waitForContainer(t, cli, ID, 20*time.Second)
	out := inspectLogs(t, cli, ID)
	t.Logf("One shot container finished with rc=%v, output=%v", rc, out)
	return rc, out
}

func startMultiVolumeQueueManager(t *testing.T, cli ce.ContainerInterface, dataVol bool, qmsharedlogs string, qmshareddata string, env []string, qmRun string, qmTmp string, readOnlyRootFs bool) (error, string, string) {
	id := strconv.FormatInt(time.Now().UnixNano(), 10)
	volume := createVolume(t, cli, id)
	containerConfig := ce.ContainerConfig{
		Image: imageName(),
		Env:   env,
	}
	var hostConfig ce.ContainerHostConfig

	if !dataVol {
		hostConfig = ce.ContainerHostConfig{}
		if readOnlyRootFs {
			hostConfig.Binds = append(hostConfig.Binds, qmRun+":/run")
			hostConfig.Binds = append(hostConfig.Binds, qmTmp+":/tmp")
			hostConfig.ReadOnlyRootfs = true
		}
	} else if qmsharedlogs == "" && qmshareddata == "" {
		hostConfig = getHostConfig(t, 1, "", "", volume, qmRun, qmTmp, readOnlyRootFs)
	} else if qmsharedlogs == "" {
		hostConfig = getHostConfig(t, 2, "", qmshareddata, volume, qmRun, qmTmp, readOnlyRootFs)
	} else if qmshareddata == "" {
		hostConfig = getHostConfig(t, 3, qmsharedlogs, "", volume, qmRun, qmTmp, readOnlyRootFs)
	} else {
		hostConfig = getHostConfig(t, 4, qmsharedlogs, qmshareddata, volume, qmRun, qmTmp, readOnlyRootFs)
	}
	networkingConfig := ce.ContainerNetworkSettings{}
	qmID, err := cli.ContainerCreate(&containerConfig, &hostConfig, &networkingConfig, t.Name()+id)
	if err != nil {
		return err, "", ""
	}
	startContainer(t, cli, qmID)

	return nil, qmID, volume
}

func getHostConfig(t *testing.T, mounts int, qmsharedlogs string, qmshareddata string, qmdata string, qmRun string, qmTmp string, readOnlyRootFS bool) ce.ContainerHostConfig {

	var hostConfig ce.ContainerHostConfig

	switch mounts {
	case 1:
		hostConfig = ce.ContainerHostConfig{
			Binds: []string{
				qmdata + ":/mnt/mqm",
			},
		}
	case 2:
		hostConfig = ce.ContainerHostConfig{
			Binds: []string{
				qmdata + ":/mnt/mqm",
				qmshareddata + ":/mnt/mqm-data",
			},
		}
	case 3:
		hostConfig = ce.ContainerHostConfig{
			Binds: []string{
				qmdata + ":/mnt/mqm",
				qmsharedlogs + ":/mnt/mqm-log",
			},
		}
	case 4:
		hostConfig = ce.ContainerHostConfig{
			Binds: []string{
				qmdata + ":/mnt/mqm",
				qmsharedlogs + ":/mnt/mqm-log",
				qmshareddata + ":/mnt/mqm-data",
			},
		}
	}
	if coverage() {
		hostConfig.Binds = append(hostConfig.Binds, coverageBind(t))
	}
	if readOnlyRootFS {
		hostConfig.Binds = append(hostConfig.Binds, qmRun+":/run")
		hostConfig.Binds = append(hostConfig.Binds, qmTmp+":/tmp")
		hostConfig.ReadOnlyRootfs = true
	}
	return hostConfig
}

func startContainer(t *testing.T, cli ce.ContainerInterface, ID string) {
	t.Logf("Starting container: %v", ID)
	startOptions := ce.ContainerStartOptions{}
	err := cli.ContainerStart(ID, startOptions)
	if err != nil {
		t.Fatal(err)
	}
}

func stopContainer(t *testing.T, cli ce.ContainerInterface, ID string) {
	t.Logf("Stopping container: %v", ID)
	timeout := 10 * time.Second
	err := cli.ContainerStop(ID, &timeout)
	if err != nil {
		// Just log the error and continue
		t.Log(err)
	}
}

func killContainer(t *testing.T, cli ce.ContainerInterface, ID string, signal string) {
	t.Logf("Killing container: %v", ID)
	err := cli.ContainerKill(ID, signal)
	if err != nil {
		t.Fatal(err)
	}
}

func getExitCodeFilename(t *testing.T) string {
	return t.Name() + "ExitCode"
}

func getCoverageExitCode(t *testing.T, orig int64) int64 {
	f := pathutils.CleanPath(coverageDir(t, true), getExitCodeFilename(t))
	_, err := os.Stat(f)
	if err != nil {
		t.Log(err)
		return orig
	}
	// Remove the file, ready for the next test
	defer os.Remove(f)
	buf, err := os.ReadFile(f)
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
func waitForContainer(t *testing.T, cli ce.ContainerInterface, ID string, timeout time.Duration) int64 {
	c, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	t.Logf("Waiting for container for %s", timeout)
	okC, errC := cli.ContainerWait(c, ID, ce.ContainerStateNotRunning)
	var rc int64
	select {
	case err := <-errC:
		t.Fatal(err)
	case ok := <-okC:
		rc = ok
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
func execContainer(t *testing.T, cli ce.ContainerInterface, ID string, user string, cmd []string) (int, string) {
	t.Logf("Container %s - Running command: %v", ID[:8], cmd)
	exitcode, outputStr := cli.ExecContainer(ID, user, cmd)
	return exitcode, outputStr
}

func waitForReady(t *testing.T, cli ce.ContainerInterface, ID string) {

	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()

	for {
		select {
		case <-time.After(1 * time.Second):
			rc, _ := execContainer(t, cli, ID, "", []string{"chkmqready"})

			if rc == 0 {
				t.Log("MQ is ready")
				return
			} else if rc == 10 {
				t.Log("MQ Readiness: Queue Manager Running as Standby")
				return
			} else if rc == 20 {
				t.Log("MQ Readiness: Queue Manager Running as Replica")
				return
			}
		case <-ctx.Done():
			t.Fatal("Timed out waiting for container to become ready")
		}
	}
}

func waitForWebConsoleReady(t *testing.T, cli ce.ContainerInterface, ID string) {

	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()

	for {
		select {
		case <-time.After(1 * time.Second):
			rc, out := execContainer(t, cli, ID, "", []string{"dspmqweb"})

			t.Logf("value of rc = %v, output: %v", rc, out)
			if rc == 0 {
				t.Log("MQ Web console is ready")
				return
			}
		case <-ctx.Done():
			rc, out := execContainer(t, cli, ID, "", []string{"dspmqweb"})
			t.Logf("value of rc = %v, output: %v", rc, out)
			t.Fatal("Timed out waiting for container webconsole to become ready")
		}
	}
}

func createNetwork(t *testing.T, cli ce.ContainerInterface) string {
	name := "test"
	t.Logf("Creating network: %v", name)
	opts := ce.NetworkCreateOptions{}
	netID, err := cli.NetworkCreate(name, opts)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Created network %v with ID %v", name, netID)
	return netID
}

func removeNetwork(t *testing.T, cli ce.ContainerInterface, ID string) {
	t.Logf("Removing network ID: %v", ID)
	err := cli.NetworkRemove(ID)
	if err != nil {
		t.Fatal(err)
	}
}

func createVolume(t *testing.T, cli ce.ContainerInterface, name string) string {
	v, err := cli.VolumeCreate(ce.VolumeCreateOptions{
		Driver: "local",
		Name:   name,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Created volume %v", v)
	return v
}

func removeVolume(t *testing.T, cli ce.ContainerInterface, name string) {
	t.Logf("Removing volume %v", name)
	err := cli.VolumeRemove(name, true)
	if err != nil {
		t.Fatal(err)
	}
}

func inspectTextLogs(t *testing.T, cli ce.ContainerInterface, ID string) string {
	jsonLogs := inspectLogs(t, cli, ID)
	scanner := bufio.NewScanner(strings.NewReader(jsonLogs))
	b := make([]byte, 64*1024)
	buf := bytes.NewBuffer(b)
	for scanner.Scan() {
		text := scanner.Text()
		if strings.HasPrefix(text, "{") {
			// If it's a JSON log message, it makes it hard to debug the test, as the JSON
			// is embedded in the long test output.  So just summarize the JSON instead.
			var e map[string]interface{}
			json.Unmarshal([]byte(text), &e)
			fmt.Fprintf(buf, "{\"ibm_datetime\": \"%v\", \"message\": \"%v\", ...}\n", e["ibm_datetime"], e["message"])
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

func inspectLogs(t *testing.T, cli ce.ContainerInterface, ID string) string {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	logs, err := cli.GetContainerLogs(ctx, ID, ce.ContainerLogsOptions{})
	if err != nil {
		t.Fatal(err)
	}
	return logs
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
func createImage(t *testing.T, cli ce.ContainerInterface, files []struct{ Name, Body string }) string {
	r := bytes.NewReader(generateTAR(t, files))
	tag := strings.ToLower(t.Name())

	tmpDir, err := os.MkdirTemp("", "tmp")
	if err != nil {
		t.Fatal(err)
	}

	defer os.RemoveAll(tmpDir)

	//Write files to temp directory
	for _, file := range files {
		//Add tag to file name to allow parallel testing
		f, err := os.Create(pathutils.CleanPath(tmpDir, file.Name))
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()

		body := []byte(file.Body)
		_, err = f.Write(body)
		if err != nil {
			t.Fatal(err)
		}
	}
	_, err = cli.ImageBuild(r, tag, pathutils.CleanPath(tmpDir, files[0].Name))
	if err != nil {
		t.Fatal(err)
	}
	return tag
}

// deleteImage deletes a Docker image
func deleteImage(t *testing.T, cli ce.ContainerInterface, id string) {
	cli.ImageRemove(id, ce.ImageRemoveOptions{
		Force: true,
	})
}

func copyFromContainer(t *testing.T, cli ce.ContainerInterface, id string, file string) []byte {
	b, err := cli.CopyFromContainer(id, file)
	if err != nil {
		t.Fatal(err)
	}
	return b
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

// scanForExcludedEntries scans for default excluded messages. Returns the message code if found.
// this checks for the default excluded messages MQ_LOGGING_CONSOLE_EXCLUDE_ID : https://www.ibm.com/docs/en/ibm-mq/9.4?topic=reference-mq-advanced-container-image
func scanForExcludedEntries(msg string) string {
	excludedEntries := []string{"AMQ5041I", "AMQ5052I", "AMQ5051I", "AMQ5037I", "AMQ5975I"}
	for _, ee := range excludedEntries {
		if strings.Contains(msg, ee) {
			return ee
		}
	}
	return ""
}

// checkLogForValidJSON checks if the message is in Json format
func checkLogForValidJSON(jsonLogs string) bool {
	scanner := bufio.NewScanner(strings.NewReader(jsonLogs))
	for scanner.Scan() {
		var obj map[string]interface{}
		s := scanner.Text()
		err := json.Unmarshal([]byte(s), &obj)
		if err != nil {
			return false
		}
	}
	return true
}

// runContainerWithAllConfig creates and starts a container, using the supplied ContainerConfig, HostConfig,
// NetworkingConfig, and container name (or the value of t.Name if containerName="").
func runContainerWithAllConfigError(t *testing.T, cli ce.ContainerInterface, containerConfig *ce.ContainerConfig, hostConfig *ce.ContainerHostConfig, networkingConfig *ce.ContainerNetworkSettings, containerName string) (string, error) {
	if containerName == "" {
		containerName = t.Name()
	}
	if containerConfig.Image == "" {
		containerConfig.Image = imageName()
	}
	// Always run as a random user, unless the test has specified otherwise
	if containerConfig.User == "" {
		containerConfig.User = generateRandomUID()
	}
	if coverage() {
		containerConfig.Env = append(containerConfig.Env, "COVERAGE_FILE="+t.Name()+".cov")
		containerConfig.Env = append(containerConfig.Env, "EXIT_CODE_FILE="+getExitCodeFilename(t))
	}
	t.Logf("Running container (%s)", containerConfig.Image)
	ID, err := cli.ContainerCreate(containerConfig, hostConfig, networkingConfig, containerName)
	if err != nil {
		return "", err
	}
	err = startContainerError(t, cli, ID)
	if err != nil {
		return "", err
	}
	return ID, nil
}

func startContainerError(t *testing.T, cli ce.ContainerInterface, ID string) error {
	t.Logf("Starting container: %v", ID)
	startOptions := ce.ContainerStartOptions{}
	err := cli.ContainerStart(ID, startOptions)
	if err != nil {
		return err
	}

	return nil
}

// testLogFilePages validates that the specified number of logFilePages is present in the qm.ini file.
func testLogFilePages(t *testing.T, cli ce.ContainerInterface, id string, qmName string, expectedLogFilePages string) {
	catIniFileCommand := fmt.Sprintf("%s", "cat /var/mqm/qmgrs/" + qmName + "/qm.ini")
	_, iniContent := execContainer(t, cli, id, "", []string{"bash", "-c", catIniFileCommand})

	if !strings.Contains(iniContent, "LogFilePages="+expectedLogFilePages) {
		t.Errorf("Expected qm.ini to contain LogFilePages="+expectedLogFilePages+"; got qm.ini \"%v\"", iniContent)
	}
}

// waitForMessageInLog will check for a particular message with wait
func waitForMessageInLog(t *testing.T, cli ce.ContainerInterface, id string, expectedMessageId string) (string, error) {
	timeout := time.NewTimer(120 * time.Second)
	var jsonLogs string
	for {
		select {
		case <-timeout.C:
			return "", fmt.Errorf("timeout waiting for expected message ID %q to be logged", expectedMessageId)
		default:
			jsonLogs = inspectLogs(t, cli, id)
			if strings.Contains(jsonLogs, expectedMessageId) {
				return jsonLogs, nil
			}
		}
	}
}

// waitForMessageCountInLog will check for a particular message with wait and must occur exact number of times in log as specified by count
func waitForMessageCountInLog(t *testing.T, cli ce.ContainerInterface, id string, expectedMessageId string, count int) (string, error) {
	timeout := time.NewTimer(120 * time.Second)
	var jsonLogs string
	for {
		select {
		case <-timeout.C:
			return "", fmt.Errorf("timeout waiting for expected message ID %q to be logged %d times", expectedMessageId, count)
		default:
			jsonLogs = inspectLogs(t, cli, id)
			if strings.Count(jsonLogs, expectedMessageId) == count {
				return jsonLogs, nil
			}
		}
	}
}

// Returns fully qualified path
func tlsDirDN(t *testing.T, unixPath bool, certPath string) string {
	return pathutils.CleanPath(filepath.Dir(getCwd(t, unixPath)), certPath)
}
