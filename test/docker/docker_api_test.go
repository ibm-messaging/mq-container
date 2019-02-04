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
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"

	"github.com/ibm-messaging/mq-container/internal/command"
)

func TestLicenseNotSet(t *testing.T) {
	t.Parallel()
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
	expectTerminationMessage(t)
}

func TestLicenseView(t *testing.T) {
	t.Parallel()
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

// TestGoldenPath starts a queue manager successfully when metrics are enabled
func TestGoldenPathWithMetrics(t *testing.T) {
	t.Parallel()
	goldenPath(t, true)
}

// TestGoldenPath starts a queue manager successfully when metrics are disabled
func TestGoldenPathNoMetrics(t *testing.T) {
	t.Parallel()
	goldenPath(t, false)
}

// Actual test function for TestGoldenPathNoMetrics & TestGoldenPathWithMetrics
func goldenPath(t *testing.T, metric bool) {
	cli, err := client.NewEnvClient()
	if err != nil {
		t.Fatal(err)
	}
	containerConfig := container.Config{
		Env: []string{"LICENSE=accept", "MQ_QMGR_NAME=qm1"},
	}
	if metric {
		containerConfig.Env = append(containerConfig.Env, "MQ_ENABLE_METRICS=true")
	}

	id := runContainer(t, cli, &containerConfig)
	defer cleanContainer(t, cli, id)
	waitForReady(t, cli, id)
	// Stop the container cleanly
	stopContainer(t, cli, id)
}

// TestSecurityVulnerabilitiesUbuntu checks for any vulnerabilities in the image, as reported
// by Ubuntu
func TestSecurityVulnerabilitiesUbuntu(t *testing.T) {
	t.Parallel()
	cli, err := client.NewEnvClient()
	if err != nil {
		t.Fatal(err)
	}
	rc, _ := runContainerOneShot(t, cli, "bash", "-c", "test -d /etc/apt")
	if rc != 0 {
		t.Skip("Skipping test because container is not Ubuntu-based")
	}
	// Override the entrypoint to make "apt" only receive security updates, then check for updates
	var url string
	if runtime.GOARCH == "amd64" {
		url = "http://security.ubuntu.com/ubuntu/"
	} else {
		url = "http://ports.ubuntu.com/ubuntu-ports/"
	}
	rc, log := runContainerOneShot(t, cli, "bash", "-c", "source /etc/os-release && echo \"deb "+url+" ${VERSION_CODENAME}-security main restricted\" > /etc/apt/sources.list && apt-get update 2>&1 >/dev/null && apt-get --simulate -qq upgrade")
	if rc != 0 {
		t.Fatalf("Expected success, got %v", rc)
	}
	lines := strings.Split(strings.TrimSpace(log), "\n")
	if len(lines) > 0 && lines[0] != "" {
		t.Errorf("Expected no vulnerabilities, found the following:\n%v", log)
	}
}

// TestSecurityVulnerabilitiesRedHat checks for any vulnerabilities in the image, as reported
// by Red Hat
func TestSecurityVulnerabilitiesRedHat(t *testing.T) {
	t.Parallel()
	cli, err := client.NewEnvClient()
	if err != nil {
		t.Fatal(err)
	}
	_, ret, _ := command.Run("bash", "-c", "test -f /etc/redhat-release")
	if ret != 0 {
		t.Skip("Skipping test because host is not RedHat-based")
	}
	rc, _ := runContainerOneShot(t, cli, "bash", "-c", "test -f /etc/redhat-release")
	if rc != 0 {
		t.Skip("Skipping test because container is not RedHat-based")
	}
	id, _, err := command.Run("buildah", "from", imageName())
	if err != nil {
		t.Fatal(err)
	}
	id = strings.TrimSpace(id)
	defer command.Run("buildah", "rm", id)
	mnt, _, err := command.Run("buildah", "mount", id)
	if err != nil {
		t.Fatal(err)
	}
	mnt = strings.TrimSpace(mnt)
	_, _, err = command.Run("bash", "-c", "cp /etc/yum.repos.d/* "+ filepath.Join(mnt, "/etc/yum.repos.d/"))
	if err != nil {
		t.Fatal(err)
	}
	out, ret, _ := command.Run("bash", "-c", "yum --installroot="+mnt+" updateinfo list sec | grep /Sec")
	if ret != 1{
		t.Errorf("Expected no vulnerabilities, found the following:\n%v", out)
	}
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
	}
	id := runContainer(t, cli, &containerConfig)
	defer cleanContainer(t, cli, id)
	waitForReady(t, cli, id)
	_, out := execContainer(t, cli, id, "mqm", []string{"dspmq"})
	if !strings.Contains(out, search) {
		t.Errorf("Expected result of running dspmq to contain name=%v, got name=%v", search, out)
	}
}
func TestNoQueueManagerName(t *testing.T) {
	t.Parallel()
	utilTestNoQueueManagerName(t, "test", "test")
}

func TestNoQueueManagerNameInvalidHostname(t *testing.T) {
	t.Parallel()
	utilTestNoQueueManagerName(t, "test-1", "test1")
}

// TestWithVolume runs a container with a Docker volume, then removes that
// container and starts a new one with same volume. With metrics enabled
func TestWithVolumeAndMetrics(t *testing.T) {
	t.Parallel()
	withVolume(t, true)
}

// TestWithVolume runs a container with a Docker volume, then removes that
// container and starts a new one with same volume. With metrics disabled
func TestWithVolumeNoMetrics(t *testing.T) {
	t.Parallel()
	withVolume(t, false)
}

// Actual test function for TestWithVolumeNoMetrics & TestWithVolumeAndMetrics
func withVolume(t *testing.T, metric bool) {
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
	if metric {
		containerConfig.Env = append(containerConfig.Env, "MQ_ENABLE_METRICS=true")
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
	t.Parallel()
	cli, err := client.NewEnvClient()
	if err != nil {
		t.Fatal(err)
	}
	containerConfig := container.Config{
		Env: []string{"LICENSE=accept", "MQ_QMGR_NAME=qm1"},
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
	t.Parallel()
	cli, err := client.NewEnvClient()
	if err != nil {
		t.Fatal(err)
	}
	img, _, err := cli.ImageInspectWithRaw(context.Background(), imageName())
	if err != nil {
		t.Fatal(err)
	}
	oldEntrypoint := strings.Join(img.Config.Entrypoint, " ")
	containerConfig := container.Config{
		Env: []string{"LICENSE=accept", "MQ_QMGR_NAME=qm1"},
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
	expectTerminationMessage(t)
}

// TestStartQueueManagerFail causes a failure of `strmqm`
func TestStartQueueManagerFail(t *testing.T) {
	t.Parallel()
	cli, err := client.NewEnvClient()
	if err != nil {
		t.Fatal(err)
	}
	img, _, err := cli.ImageInspectWithRaw(context.Background(), imageName())
	if err != nil {
		t.Fatal(err)
	}
	oldEntrypoint := strings.Join(img.Config.Entrypoint, " ")
	containerConfig := container.Config{
		Env: []string{"LICENSE=accept", "MQ_QMGR_NAME=qm1", "DEBUG=1"},
		// Override the entrypoint to replace `strmqm` with a script which deletes the queue manager.
		// This will cause `strmqm` to return with an exit code of 72.
		Entrypoint: []string{"bash", "-c", "echo '#!/bin/bash\ndltmqm $@ && strmqm $@' > /opt/mqm/bin/strmqm && exec " + oldEntrypoint},
	}
	id := runContainer(t, cli, &containerConfig)
	defer cleanContainer(t, cli, id)
	rc := waitForContainer(t, cli, id, 10)
	if rc != 1 {
		t.Errorf("Expected rc=1, got rc=%v", rc)
	}
	expectTerminationMessage(t)
}

// TestVolumeUnmount runs a queue manager with a volume, and then forces an
// unmount of the volume.  The health check should then fail.
// This simulates behaviour seen in some cloud environments, where network
// attached storage gets unmounted.
func TestVolumeUnmount(t *testing.T) {
	t.Parallel()
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
	rc, out := execContainer(t, cli, ctr.ID, "root", []string{"umount", "-l", "-f", "/mnt/mqm"})
	if rc != 0 {
		t.Fatalf("Expected umount to work with rc=0, got %v. Output was: %s", rc, out)
	}
	time.Sleep(3 * time.Second)
	rc, _ = execContainer(t, cli, ctr.ID, "mqm", []string{"chkmqhealthy"})
	if rc == 0 {
		t.Errorf("Expected chkmqhealthy to fail")
		_, df := execContainer(t, cli, ctr.ID, "mqm", []string{"df"})
		t.Logf(df)
		_, ps := execContainer(t, cli, ctr.ID, "mqm", []string{"ps", "-ef"})
		t.Logf(ps)
	}
}

// TestZombies starts a queue manager, then causes a zombie process to be
// created, then checks that no zombies exist (runmqserver should reap them)
func TestZombies(t *testing.T) {
	t.Parallel()
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
	_, out := execContainer(t, cli, id, "mqm", []string{"pkill", "--signal", "kill", "-c", "amqzxma0"})
	if out == "0" {
		t.Log("Failed to kill process 'amqzxma0'")
		_, out := execContainer(t, cli, id, "root", []string{"ps", "-lA"})
		t.Fatalf("Here is a list of currently running processes:\n%s", out)
	}
	time.Sleep(3 * time.Second)
	_, out = execContainer(t, cli, id, "mqm", []string{"bash", "-c", "ps -lA | grep '^. Z'"})
	if out != "" {
		count := strings.Count(out, "\n") + 1
		t.Errorf("Expected zombies=0, got %v", count)
		t.Error(out)
		t.Fail()
	}
}

// TestMQSC creates a new image with an MQSC file in, starts a container based
// on that image, and checks that the MQSC has been applied correctly.
func TestMQSC(t *testing.T) {
	t.Parallel()
	cli, err := client.NewEnvClient()
	if err != nil {
		t.Fatal(err)
	}
	var files = []struct {
		Name, Body string
	}{
		{"Dockerfile", fmt.Sprintf("FROM %v\nRUN rm -f /etc/mqm/*.mqsc\nADD test.mqsc /etc/mqm/", imageName())},
		{"test.mqsc", "DEFINE QLOCAL(test)"},
	}
	tag := createImage(t, cli, files)
	defer deleteImage(t, cli, tag)

	containerConfig := container.Config{
		Env:   []string{"LICENSE=accept", "MQ_QMGR_NAME=qm1"},
		Image: tag,
	}
	id := runContainer(t, cli, &containerConfig)
	defer cleanContainer(t, cli, id)
	waitForReady(t, cli, id)
	rc, mqscOutput := execContainer(t, cli, id, "mqm", []string{"bash", "-c", "echo 'DISPLAY QLOCAL(test)' | runmqsc"})
	if rc != 0 {
		r := regexp.MustCompile("AMQ[0-9][0-9][0-9][0-9]E")
		t.Fatalf("Expected runmqsc to exit with rc=0, got %v with error %v", rc, r.FindString(mqscOutput))
	}
}

// TestInvalidMQSC creates a new image with an MQSC file containing invalid MQSC,
// tries to start a container based on that image, and checks that container terminates
func TestInvalidMQSC(t *testing.T) {
	t.Parallel()
	cli, err := client.NewEnvClient()
	if err != nil {
		t.Fatal(err)
	}
	var files = []struct {
		Name, Body string
	}{
		{"Dockerfile", fmt.Sprintf("FROM %v\nRUN rm -f /etc/mqm/*.mqsc\nADD mqscTest.mqsc /etc/mqm/", imageName())},
		{"mqscTest.mqsc", "DEFINE INVALIDLISTENER('TEST.LISTENER.TCP') TRPTYPE(TCP) PORT(1414) CONTROL(QMGR) REPLACE"},
	}
	tag := createImage(t, cli, files)
	defer deleteImage(t, cli, tag)

	containerConfig := container.Config{
		Env:   []string{"LICENSE=accept", "MQ_QMGR_NAME=qm1"},
		Image: tag,
	}
	id := runContainer(t, cli, &containerConfig)
	defer cleanContainer(t, cli, id)
	rc := waitForContainer(t, cli, id, 5)
	if rc != 1 {
		t.Errorf("Expected rc=1, got rc=%v", rc)
	}
	expectTerminationMessage(t)
}

// TestReadiness creates a new image with large amounts of MQSC in, to
// ensure that the readiness check doesn't pass until configuration has finished.
// WARNING: This test is sensitive to the speed of the machine it's running on.
func TestReadiness(t *testing.T) {
	t.Parallel()
	cli, err := client.NewEnvClient()
	if err != nil {
		t.Fatal(err)
	}
	const numQueues = 3
	var buf bytes.Buffer
	for i := 1; i <= numQueues; i++ {
		fmt.Fprintf(&buf, "* Defining queue test %v\nDEFINE QLOCAL(test%v)\n", i, i)
	}
	var files = []struct {
		Name, Body string
	}{
		{"Dockerfile", fmt.Sprintf("FROM %v\nRUN rm -f /etc/mqm/*.mqsc\nADD test.mqsc /etc/mqm/", imageName())},
		{"test.mqsc", buf.String()},
	}
	tag := createImage(t, cli, files)
	defer deleteImage(t, cli, tag)

	containerConfig := container.Config{
		Env:   []string{"LICENSE=accept", "MQ_QMGR_NAME=qm1", "DEBUG=1"},
		Image: tag,
	}
	id := runContainer(t, cli, &containerConfig)
	defer cleanContainer(t, cli, id)
	queueCheckCommand := fmt.Sprintf("echo 'DISPLAY QLOCAL(test%v)' | runmqsc", numQueues)
	_, mqsc := execContainer(t, cli, id, "root", []string{"cat", "/etc/mqm/test.mqsc"})
	t.Log(mqsc)
	for {
		readyRC, _ := execContainer(t, cli, id, "mqm", []string{"chkmqready"})
		queueCheckRC, queueCheckOut := execContainer(t, cli, id, "mqm", []string{"bash", "-c", queueCheckCommand})
		t.Logf("readyRC=%v,queueCheckRC=%v\n", readyRC, queueCheckRC)

		if readyRC == 0 {
			if queueCheckRC != 0 {
				r := regexp.MustCompile("AMQ[0-9][0-9][0-9][0-9]E")
				t.Fatalf("Runmqsc returned %v with error %v. chkmqready returned %v when MQSC had not finished", queueCheckRC, r.FindString(queueCheckOut), readyRC)
			} else {
				// chkmqready says OK, and the last queue exists, so return
				_, runmqsc := execContainer(t, cli, id, "root", []string{"bash", "-c", "echo 'DISPLAY QLOCAL(test1)' | runmqsc"})
				t.Log(runmqsc)
				return
			}
		}
	}
}

func TestErrorLogRotation(t *testing.T) {
	t.Parallel()
	cli, err := client.NewEnvClient()
	if err != nil {
		t.Fatal(err)
	}
	qmName := "qm1"
	containerConfig := container.Config{
		Env: []string{
			"LICENSE=accept",
			"MQ_QMGR_NAME=" + qmName,
			"MQMAXERRORLOGSIZE=65536",
			"LOG_FORMAT=json",
		},
		ExposedPorts: nat.PortSet{
			"1414/tcp": struct{}{},
		},
	}
	id := runContainer(t, cli, &containerConfig)
	defer cleanContainer(t, cli, id)
	waitForReady(t, cli, id)
	dir := "/var/mqm/qmgrs/" + qmName + "/errors"
	// Generate some content for the error logs, by trying to put messages under an unauthorized user
	// execContainer(t, cli, id, "fred", []string{"bash", "-c", "for i in {1..30} ; do /opt/mqm/samp/bin/amqsput FAKE; done"})
	execContainer(t, cli, id, "root", []string{"useradd", "fred"})
	for {
		execContainer(t, cli, id, "fred", []string{"bash", "-c", "/opt/mqm/samp/bin/amqsput FAKE"})

		_, atoiStr := execContainer(t, cli, id, "mqm", []string{"bash", "-c", "wc -c < " + filepath.Join(dir, "AMQERR02.json")})
		amqerr02size, _ := strconv.Atoi(atoiStr)

		if amqerr02size > 0 {
			// We've done enough to cause log rotation
			break
		}
	}
	_, out := execContainer(t, cli, id, "root", []string{"ls", "-l", dir})
	t.Log(out)
	stopContainer(t, cli, id)
	b := copyFromContainer(t, cli, id, filepath.Join(dir, "AMQERR01.json"))
	amqerr01 := countTarLines(t, b)
	b = copyFromContainer(t, cli, id, filepath.Join(dir, "AMQERR02.json"))
	amqerr02 := countTarLines(t, b)
	b = copyFromContainer(t, cli, id, filepath.Join(dir, "AMQERR03.json"))
	amqerr03 := countTarLines(t, b)
	scanner := bufio.NewScanner(strings.NewReader(inspectLogs(t, cli, id)))
	totalMirrored := 0
	for scanner.Scan() {
		if strings.Contains(scanner.Text(), "\"message\":\"AMQ") {
			totalMirrored++
		}
	}
	err = scanner.Err()
	if err != nil {
		t.Fatal(err)
	}
	total := amqerr01 + amqerr02 + amqerr03
	if totalMirrored != total {
		t.Fatalf("Expected %v (%v + %v + %v) mirrored log entries; got %v", total, amqerr01, amqerr02, amqerr03, totalMirrored)
	} else {
		t.Logf("Found %v (%v + %v + %v) mirrored log entries", totalMirrored, amqerr01, amqerr02, amqerr03)
	}
}

// Tests the log comes out in JSON format when JSON format is enabled. With metrics enabled
func TestJSONLogFormatWithMetrics(t *testing.T) {
	t.Parallel()
	jsonLogFormat(t, true)
}

// Tests the log comes out in JSON format when JSON format is enabled. With metrics disabled
func TestJSONLogFormatNoMetrics(t *testing.T) {
	t.Parallel()
	jsonLogFormat(t, false)
}

// Actual test function for TestJSONLogFormatWithMetrics & TestJSONLogFormatNoMetrics
func jsonLogFormat(t *testing.T, metric bool) {
	cli, err := client.NewEnvClient()
	if err != nil {
		t.Fatal(err)
	}
	containerConfig := container.Config{
		Env: []string{
			"LICENSE=accept",
			"LOG_FORMAT=json",
		},
	}
	if metric {
		containerConfig.Env = append(containerConfig.Env, "MQ_ENABLE_METRICS=true")
	}

	id := runContainer(t, cli, &containerConfig)
	defer cleanContainer(t, cli, id)
	waitForReady(t, cli, id)
	stopContainer(t, cli, id)
	scanner := bufio.NewScanner(strings.NewReader(inspectLogs(t, cli, id)))
	for scanner.Scan() {
		var obj map[string]interface{}
		s := scanner.Text()
		err := json.Unmarshal([]byte(s), &obj)
		if err != nil {
			t.Fatalf("Expected all log lines to be valid JSON.  Got error %v for line %v", err, s)
		}
	}
	err = scanner.Err()
	if err != nil {
		t.Fatal(err)
	}
}

func TestBadLogFormat(t *testing.T) {
	t.Parallel()
	cli, err := client.NewEnvClient()
	if err != nil {
		t.Fatal(err)
	}
	containerConfig := container.Config{
		Env: []string{
			"LICENSE=accept",
			"LOG_FORMAT=fake",
		},
	}
	id := runContainer(t, cli, &containerConfig)
	defer cleanContainer(t, cli, id)
	rc := waitForContainer(t, cli, id, 5)
	if rc != 1 {
		t.Errorf("Expected rc=1, got rc=%v", rc)
	}
	expectTerminationMessage(t)
}

// TestMQJSONDisabled tests the case where MQ's JSON logging feature is
// specifically disabled (which will disable log mirroring)
func TestMQJSONDisabled(t *testing.T) {
	t.SkipNow()
	t.Parallel()
	cli, err := client.NewEnvClient()
	if err != nil {
		t.Fatal(err)
	}
	containerConfig := container.Config{
		Env: []string{
			"LICENSE=accept",
			"MQ_QMGR_NAME=qm1",
			"AMQ_ADDITIONAL_JSON_LOG=0",
		},
	}
	id := runContainer(t, cli, &containerConfig)
	defer cleanContainer(t, cli, id)
	waitForReady(t, cli, id)
	// Stop the container (which could hang if runmqserver is still waiting for
	// JSON logs to appear)
	stopContainer(t, cli, id)
}

func TestCorrectLicense(t *testing.T) {
	t.Parallel()

	//Check we have the license set
	expectedLicense, ok := os.LookupEnv("EXPECTED_LICENSE")
	if !ok {
		t.Fatal("Required test environment variable 'EXPECTED_LICENSE' was not set.")
	}

	cli, err := client.NewEnvClient()
	if err != nil {
		t.Fatal(err)
	}

	containerConfig := container.Config{
		Env: []string{"LICENSE=accept"},
	}
	id := runContainer(t, cli, &containerConfig)
	defer cleanContainer(t, cli, id)
	waitForReady(t, cli, id)

	rc, license := execContainer(t, cli, id, "mqm", []string{"dspmqver", "-f", "8192", "-b"})
	if rc != 0 {
		t.Fatalf("Failed to get license string. RC=%d. Output=%s", rc, license)
	}

	if license != expectedLicense {
		t.Errorf("Expected license to be '%s' but was '%s", expectedLicense, license)
	}
}

func TestVersioning(t *testing.T) {
	t.Parallel()

	cli, err := client.NewEnvClient()
	if err != nil {
		t.Fatal(err)
	}

	containerConfig := container.Config{
		Env: []string{"LICENSE=accept"},
	}
	id := runContainer(t, cli, &containerConfig)
	defer cleanContainer(t, cli, id)
	waitForReady(t, cli, id)

	// Get whole logs and check versioning system
	l := inspectLogs(t, cli, id)
	scanner := bufio.NewScanner(strings.NewReader(l))

	total := 6
	foundCreated := false
	foundRevision := false
	foundSource := false
	foundMQVersion := false
	foundMQLevel := false
	foundMQLicense := false

	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "Image created:") && !foundCreated {
			total--
			foundCreated = true
			dataAr := strings.Split(line, " ")
			data := dataAr[len(dataAr)-1]

			// Verify created is in a known timestamp format
			_, err := time.Parse(time.RFC3339, data)
			_, err2 := time.Parse("2006-01-02T15:04:05-0700", data)
			if err != nil && err2 != nil {
				t.Errorf("Failed to validate Image created stamp (%v) - %v or %v", data, time.RFC3339, "2006-01-02T15:04:05-0700")
			}
		}

		if strings.Contains(line, "Image revision:") && !foundRevision {
			total--
			foundRevision = true
			dataAr := strings.Split(line, " ")
			data := dataAr[len(dataAr)-1]

			// Verify revision
			pattern := regexp.MustCompile("^[a-fA-F0-9]{40}$")
			if !pattern.MatchString(data) {
				t.Errorf("Failed to validate revision (%v)", data)
			}
		}

		if strings.Contains(line, "Image source:") && !foundSource {
			total--
			foundSource = true
			dataAr := strings.Split(line, " ")
			data := dataAr[len(dataAr)-1]

			// Verify source
			if !strings.Contains(data, "github") {
				t.Errorf("Failed to validate source (%v)", data)
			}
		}

		if strings.Contains(line, "MQ version:") && !foundMQVersion {
			total--
			foundMQVersion = true
			dataAr := strings.Split(line, " ")
			data := dataAr[len(dataAr)-1]

			// Verify MQ version
			pattern := regexp.MustCompile("^\\d+\\.\\d+\\.\\d+\\.\\d+$")
			if !pattern.MatchString(data) {
				t.Errorf("Failed to validate mq version (%v)", data)
			}
		}

		if strings.Contains(line, "MQ level:") && !foundMQLevel {
			total--
			foundMQLevel = true
			dataAr := strings.Split(line, " ")
			data := dataAr[len(dataAr)-1]

			// Verify MQ version
			pattern := regexp.MustCompile("^p\\d{3}-.+$")
			if !pattern.MatchString(data) {
				t.Errorf("Failed to validate mq level (%v)", data)
			}
		}

		if strings.Contains(line, "MQ license:") && !foundMQLicense {
			total--
			foundMQLicense = true
			dataAr := strings.Split(line, " ")
			data := dataAr[len(dataAr)-1]

			// Verify MQ version
			if data != "Developer" && data != "Production" {
				t.Errorf("Failed to validate mq license (%v)", data)
			}
		}

		// end loop early
		if total == 0 {
			break
		}
	}

	if !foundCreated || !foundRevision || !foundSource || !foundMQVersion || !foundMQLevel || !foundMQLicense {
		t.Errorf("Failed to find one or more version strings: created(%v) revision(%v) source(%v) mqversion(%v) mqlevel(%v) mqlicense(%v)", foundCreated, foundRevision, foundSource, foundMQVersion, foundMQLevel, foundMQLicense)
	}
}
