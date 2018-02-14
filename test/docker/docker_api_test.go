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
	"fmt"
	"io"
	"path/filepath"
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

// TestGoldenPath starts a queue manager successfully
func TestGoldenPath(t *testing.T) {
	t.Parallel()
	cli, err := client.NewEnvClient()
	if err != nil {
		t.Fatal(err)
	}
	containerConfig := container.Config{
		Env: []string{"LICENSE=accept", "MQ_QMGR_NAME=qm1"},
		// ExposedPorts: nat.PortSet{
		// 	"1414/tcp": struct{}{},
		// },
	}
	id := runContainer(t, cli, &containerConfig)
	defer cleanContainer(t, cli, id)
	waitForReady(t, cli, id)
}

// TestSecurityVulnerabilities checks for any vulnerabilities in the image, as reported
// by Ubuntu
func TestSecurityVulnerabilities(t *testing.T) {
	t.Parallel()
	cli, err := client.NewEnvClient()
	if err != nil {
		t.Fatal(err)
	}
	// containerConfig := container.Config{
	// 	// Override the entrypoint to make "apt" only receive security updates, then check for updates
	// 	Entrypoint: []string{"bash", "-c", "source /etc/os-release && echo \"deb http://security.ubuntu.com/ubuntu/ ${VERSION_CODENAME}-security main restricted\" > /etc/apt/sources.list && apt-get update 2>&1 >/dev/null && apt-get --simulate -qq upgrade"},
	// }
	// id := runContainer(t, cli, &containerConfig)
	// defer cleanContainer(t, cli, id)
	// // rc is the return code from apt-get
	// rc := waitForContainer(t, cli, id, 10)

	rc, _ := runContainerOneShot(t, cli, "bash", "-c", "test -d /etc/apt")
	if rc != 0 {
		t.Skip("Skipping test because container is not Ubuntu-based")
	}
	// Override the entrypoint to make "apt" only receive security updates, then check for updates
	rc, log := runContainerOneShot(t, cli, "bash", "-c", "source /etc/os-release && echo \"deb http://security.ubuntu.com/ubuntu/ ${VERSION_CODENAME}-security main restricted\" > /etc/apt/sources.list && apt-get update 2>&1 >/dev/null && apt-get --simulate -qq upgrade")
	if rc != 0 {
		t.Fatalf("Expected success, got %v", rc)
	}
	lines := strings.Split(strings.TrimSpace(log), "\n")
	if len(lines) > 0 && lines[0] != "" {
		t.Errorf("Expected no vulnerabilities, found the following:\n%v", log)
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
	out := execContainerWithOutput(t, cli, id, "mqm", []string{"dspmq"})
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
// container and starts a new one with same volume.
func TestWithVolume(t *testing.T) {
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
}

// TestStartQueueManagerFail causes a failure of `strmqm`
func TestStartQueueManagerFail(t *testing.T) {
	t.Parallel()
	cli, err := client.NewEnvClient()
	if err != nil {
		t.Fatal(err)
	}
	img, _, err := cli.ImageInspectWithRaw(context.Background(), imageName())
	oldEntrypoint := strings.Join(img.Config.Entrypoint, " ")
	containerConfig := container.Config{
		Env: []string{"LICENSE=accept", "MQ_QMGR_NAME=qm1"},
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
	out := execContainerWithOutput(t, cli, id, "mqm", []string{"pkill", "--signal", "kill", "-c", "amqzxma0"})
	if out == "0" {
		t.Fatalf("Expected pkill to kill a process, got %v", out)
	}
	time.Sleep(3 * time.Second)
	out = execContainerWithOutput(t, cli, id, "mqm", []string{"bash", "-c", "ps -lA | grep '^. Z'"})
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
	rc := execContainerWithExitCode(t, cli, id, "mqm", []string{"bash", "-c", "echo 'DISPLAY QLOCAL(test)' | runmqsc"})
	if rc != 0 {
		t.Fatalf("Expected runmqsc to exit with rc=0, got %v", rc)
	}
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
	// fmt.Fprintf(&b, "world!")

	// b := make([]byte, 32768)
	// buf := bytes.NewBuffer(b)
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
	t.Log(execContainerWithOutput(t, cli, id, "root", []string{"cat", "/etc/mqm/test.mqsc"}))
	for {
		readyRC := execContainerWithExitCode(t, cli, id, "mqm", []string{"chkmqready"})
		queueCheckRC := execContainerWithExitCode(t, cli, id, "mqm", []string{"bash", "-c", queueCheckCommand})
		t.Logf("readyRC=%v,queueCheckRC=%v\n", readyRC, queueCheckRC)
		if readyRC == 0 {
			if queueCheckRC != 0 {
				t.Fatalf("chkmqready returned %v when MQSC had not finished", readyRC)
			} else {
				// chkmqready says OK, and the last queue exists, so return
				t.Log(execContainerWithOutput(t, cli, id, "root", []string{"bash", "-c", "echo 'DISPLAY QLOCAL(test1)' | runmqsc"}))
				return
			}
		}
	}
}

func countLines(t *testing.T, r io.Reader) int {
	scanner := bufio.NewScanner(r)
	count := 0
	for scanner.Scan() {
		// if scanner.Text() != "" {
		count++
		// fmt.Printf("%v: %v\n", count, scanner.Text())
		// }
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
	// totals := make([]int, 3)
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
		// totals = append(totals, countLines(t, tr))
	}
	return total
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
			"MQ_ALPHA_JSON_LOGS=true",
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
	// execContainerWithOutput(t, cli, id, "fred", []string{"bash", "-c", "for i in {1..30} ; do /opt/mqm/samp/bin/amqsput FAKE; done"})
	execContainerWithOutput(t, cli, id, "root", []string{"useradd", "fred"})
	for {
		execContainerWithOutput(t, cli, id, "fred", []string{"bash", "-c", "/opt/mqm/samp/bin/amqsput FAKE"})
		amqerr02size, err := strconv.Atoi(execContainerWithOutput(t, cli, id, "mqm", []string{"bash", "-c", "wc -c < " + filepath.Join(dir, "AMQERR02.json")}))
		if err != nil {
			t.Fatal(err)
		}
		if amqerr02size > 0 {
			// We've done enough to cause log rotation
			break
		}
	}
	out := execContainerWithOutput(t, cli, id, "root", []string{"ls", "-l", dir})
	t.Log(out)
	stopContainer(t, cli, id)
	// Wait to allow the logs to be mirrored
	//time.Sleep(5 * time.Second)
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
	// log := strings.Split(inspectLogs(t, cli, id), "\n")
	// total := 0
	// for _, v := range totals {
	// 	total += v
	// }
	//log := countLines(t, []byte(inspectLogs(t, cli, id)))
	total := amqerr01 + amqerr02 + amqerr03
	//total := len(amqerr01) + len(amqerr02) + len(amqerr03)
	// totalMirrored := 0
	// for _, line := range log {
	// 	// Look for something which is only in the MQ log lines, and not those from runmqserver
	// 	if strings.Contains(line, "ibm_threadId") {
	// 		totalMirrored++
	// 	}
	// }
	if totalMirrored != total {
		t.Fatalf("Expected %v (%v + %v + %v) mirrored log entries; got %v", total, amqerr01, amqerr02, amqerr03, totalMirrored)
		//t.Fatalf("Expected %v mirrored log entries; got %v (AMQERR01=%v entries; AMQERR02=%v entries; AMQERR03=%v entries)", total, totalMirrored, totals[0], totals[1], totals[2])
	} else {
		t.Logf("Found %v (%v + %v + %v) mirrored log entries", totalMirrored, amqerr01, amqerr02, amqerr03)
	}
}
