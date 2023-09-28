/*
© Copyright IBM Corporation 2017, 2023

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
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	ce "github.com/ibm-messaging/mq-container/test/container/containerengine"
)

func TestLicenseNotSet(t *testing.T) {
	t.Parallel()

	cli := ce.NewContainerClient()

	containerConfig := ce.ContainerConfig{}
	id := runContainer(t, cli, &containerConfig)
	defer cleanContainer(t, cli, id)
	rc := waitForContainer(t, cli, id, 30*time.Second)
	if rc != 1 {
		t.Errorf("Expected rc=1, got rc=%v", rc)
	}
	expectTerminationMessage(t, cli, id)
}

// Start container with LICENSE environment variable set to view.
// Check that container starts and display license text
func TestLicenseView(t *testing.T) {
	t.Parallel()

	cli := ce.NewContainerClient()

	containerConfig := ce.ContainerConfig{
		Env: []string{"LICENSE=view"},
	}
	id := runContainer(t, cli, &containerConfig)
	defer cleanContainer(t, cli, id)
	rc := waitForContainer(t, cli, id, 30*time.Second)
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
func goldenPath(t *testing.T, metrics bool) {
	cli := ce.NewContainerClient()

	containerConfig := ce.ContainerConfig{
		Env: []string{"LICENSE=accept", "MQ_QMGR_NAME=qm1"},
	}
	if metrics {
		containerConfig.Env = append(containerConfig.Env, "MQ_ENABLE_METRICS=true")
	}

	id := runContainer(t, cli, &containerConfig)
	defer cleanContainer(t, cli, id)
	waitForReady(t, cli, id)

	//By default AMQ5041I,AMQ5052I,AMQ5051I,AMQ5037I,AMQ5975I are excluded
	jsonLogs := inspectLogs(t, cli, id)

	isMessageFound := scanForExcludedEntries(jsonLogs)

	if isMessageFound == true {
		t.Errorf("Expected  to exclude messageId by default; but messageId \"%v\" is present", jsonLogs)
	}

	t.Run("Validate Default LogFilePages", func(t *testing.T) {
		testLogFilePages(t, cli, id, "qm1", "4096")
	})
	// Stop the container cleanly
	stopContainer(t, cli, id)
}

func utilTestNoQueueManagerName(t *testing.T, hostName string, expectedName string) {
	search := "QMNAME(" + expectedName + ")"
	cli := ce.NewContainerClient()
	containerConfig := ce.ContainerConfig{
		Env:      []string{"LICENSE=accept"},
		Hostname: hostName,
	}
	id := runContainer(t, cli, &containerConfig)
	defer cleanContainer(t, cli, id)
	waitForReady(t, cli, id)
	_, out := execContainer(t, cli, id, "", []string{"dspmq"})
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
	cli := ce.NewContainerClient()
	vol := createVolume(t, cli, t.Name())
	defer removeVolume(t, cli, vol)
	containerConfig := ce.ContainerConfig{
		Image: imageName(),
		Env:   []string{"LICENSE=accept", "MQ_QMGR_NAME=qm1"},
	}
	if metric {
		containerConfig.Env = append(containerConfig.Env, "MQ_ENABLE_METRICS=true")
	}

	hostConfig := ce.ContainerHostConfig{
		Binds: []string{
			coverageBind(t),
			vol + ":/mnt/mqm",
		},
	}
	networkingConfig := ce.ContainerNetworkSettings{}
	ID, err := cli.ContainerCreate(&containerConfig, &hostConfig, &networkingConfig, t.Name())
	if err != nil {
		t.Fatal(err)
	}
	startContainer(t, cli, ID)
	// TODO: If this test gets an error waiting for readiness, the first container might not get cleaned up
	waitForReady(t, cli, ID)

	// Delete the first container
	cleanContainer(t, cli, ID)

	// Start a new container with the same volume
	ID2, err := cli.ContainerCreate(&containerConfig, &hostConfig, &networkingConfig, t.Name())
	if err != nil {
		t.Fatal(err)
	}
	defer cleanContainer(t, cli, ID2)
	startContainer(t, cli, ID2)
	waitForReady(t, cli, ID2)
}

// TestWithSplitVolumesLogsData starts a queue manager with separate log/data mounts
func TestWithSplitVolumesLogsData(t *testing.T) {
	cli := ce.NewContainerClient()

	qmsharedlogs := createVolume(t, cli, "qmsharedlogs")
	defer removeVolume(t, cli, qmsharedlogs)
	qmshareddata := createVolume(t, cli, "qmshareddata")
	defer removeVolume(t, cli, qmshareddata)

	err, qmID, qmVol := startMultiVolumeQueueManager(t, cli, true, qmsharedlogs, qmshareddata, []string{"LICENSE=accept", "MQ_QMGR_NAME=qm1"}, "", "", false)
	if err != nil {
		t.Fatal(err)
	}

	defer removeVolume(t, cli, qmVol)
	defer cleanContainer(t, cli, qmID)

	waitForReady(t, cli, qmID)
}

// TestWithSplitVolumesLogsOnly starts a queue manager with a separate log mount
func TestWithSplitVolumesLogsOnly(t *testing.T) {
	cli := ce.NewContainerClient()

	qmsharedlogs := createVolume(t, cli, "qmsharedlogs")
	defer removeVolume(t, cli, qmsharedlogs)

	err, qmID, qmVol := startMultiVolumeQueueManager(t, cli, true, qmsharedlogs, "", []string{"LICENSE=accept", "MQ_QMGR_NAME=qm1"}, "", "", false)
	if err != nil {
		t.Fatal(err)
	}

	defer removeVolume(t, cli, qmVol)
	defer cleanContainer(t, cli, qmID)

	waitForReady(t, cli, qmID)
}

// TestWithSplitVolumesDataOnly starts a queue manager with a separate data mount
func TestWithSplitVolumesDataOnly(t *testing.T) {
	cli := ce.NewContainerClient()

	qmshareddata := createVolume(t, cli, "qmshareddata")
	defer removeVolume(t, cli, qmshareddata)

	err, qmID, qmVol := startMultiVolumeQueueManager(t, cli, true, "", qmshareddata, []string{"LICENSE=accept", "MQ_QMGR_NAME=qm1"}, "", "", false)
	if err != nil {
		t.Fatal(err)
	}

	defer removeVolume(t, cli, qmVol)
	defer cleanContainer(t, cli, qmID)

	waitForReady(t, cli, qmID)
}

// TestNoVolumeWithRestart ensures a queue manager container can be stopped
// and restarted cleanly
func TestNoVolumeWithRestart(t *testing.T) {
	t.Parallel()

	cli := ce.NewContainerClient()
	containerConfig := ce.ContainerConfig{
		Env: []string{"LICENSE=accept", "MQ_QMGR_NAME=qm1"},
	}
	id := runContainer(t, cli, &containerConfig)
	defer cleanContainer(t, cli, id)
	waitForReady(t, cli, id)
	stopContainer(t, cli, id)
	startContainer(t, cli, id)
	waitForReady(t, cli, id)
}

// TestVolumeRequiresRoot tests the case where only the root user can write
// to the persistent volume.  In this case, an "init container" is needed,
// where `runmqserver -i` is run to initialize the storage.  Then the
// container can be run as normal.
func TestVolumeRequiresRoot(t *testing.T) {
	cli := ce.NewContainerClient()

	// ":nocopy" requires podman version 4.2
	if cli.ContainerTool == "podman" && cli.Version < "4.2.0" {
		t.Skipf("Skipping as 'nocopy' is not available before podman version 4.2.0. Detected podman version %v", cli.Version)
	}

	vol := createVolume(t, cli, t.Name())
	defer removeVolume(t, cli, vol)

	// Set permissions on the volume to only allow root to write it
	// It's important that read and execute permissions are given to other users
	// This test was previously using nobody:nogroup and is now using nobody:nobody for compatibility
	rc, _ := runContainerOneShotWithVolume(t, cli, vol+":/mnt/mqm:nocopy", "chown", "nobody:nobody", "/mnt/mqm/")
	if rc != 0 {
		t.Fatalf("Expected one shot container to return rc=0, got rc=%v", rc)
	}
	rc, _ = runContainerOneShotWithVolume(t, cli, vol+":/mnt/mqm:nocopy", "chmod", "0755", "/mnt/mqm/")
	if rc != 0 {
		t.Fatalf("Expected one shot container to return rc=0, got rc=%v", rc)
	}

	containerConfig := ce.ContainerConfig{
		Image: imageName(),
		Env:   []string{"LICENSE=accept", "MQ_QMGR_NAME=qm1"},
	}
	hostConfig := ce.ContainerHostConfig{
		Binds: []string{
			coverageBind(t),
			vol + ":/mnt/mqm:nocopy",
		},
	}
	networkingConfig := ce.ContainerNetworkSettings{}

	// Run an "init container" as root, with the "-i" option, to initialize the volume
	containerConfig = ce.ContainerConfig{
		Image:      imageName(),
		Env:        []string{"LICENSE=accept", "MQ_QMGR_NAME=qm1", "DEBUG=true"},
		User:       "0",
		Entrypoint: []string{"runmqserver", "-i"},
	}
	initCtrID, err := cli.ContainerCreate(&containerConfig, &hostConfig, &networkingConfig, t.Name()+"Init")
	if err != nil {
		t.Fatal(err)
	}
	defer cleanContainer(t, cli, initCtrID)
	t.Logf("Init container ID=%v", initCtrID)
	startContainer(t, cli, initCtrID)
	rc = waitForContainer(t, cli, initCtrID, 30*time.Second)
	if rc != 0 {
		t.Errorf("Expected init container to exit with rc=0, got rc=%v", rc)
	}

	containerConfig = ce.ContainerConfig{
		Image: imageName(),
		Env:   []string{"LICENSE=accept", "MQ_QMGR_NAME=qm1", "DEBUG=true"},
	}
	ctrID, err := cli.ContainerCreate(&containerConfig, &hostConfig, &networkingConfig, t.Name()+"Main")
	if err != nil {
		t.Fatal(err)
	}
	defer cleanContainer(t, cli, ctrID)
	t.Logf("Main container ID=%v", ctrID)
	startContainer(t, cli, ctrID)
	waitForReady(t, cli, ctrID)
}

// TestCreateQueueManagerFail causes a failure of `crtmqm`
func TestCreateQueueManagerFail(t *testing.T) {
	t.Parallel()

	cli := ce.NewContainerClient()
	var files = []struct {
		Name, Body string
	}{
		{"Dockerfile", fmt.Sprintf(`
		FROM %v
		USER root
		RUN echo -e '#!/bin/bash\nexit 999' > /opt/mqm/bin/crtmqm
		USER 1001`, imageName())},
	}
	tag := createImage(t, cli, files)
	defer deleteImage(t, cli, tag)

	containerConfig := ce.ContainerConfig{
		Env:   []string{"LICENSE=accept", "MQ_QMGR_NAME=qm1"},
		Image: tag,
	}
	id := runContainer(t, cli, &containerConfig)
	defer cleanContainer(t, cli, id)
	rc := waitForContainer(t, cli, id, 30*time.Second)
	if rc != 1 {
		t.Errorf("Expected rc=1, got rc=%v", rc)
	}
	expectTerminationMessage(t, cli, id)
}

// TestStartQueueManagerFail causes a failure of `strmqm`
func TestStartQueueManagerFail(t *testing.T) {
	t.Parallel()

	cli := ce.NewContainerClient()
	var files = []struct {
		Name, Body string
	}{
		{"Dockerfile", fmt.Sprintf(`
		FROM %v
		USER root
		RUN echo '#!/bin/bash\ndltmqm $@ && strmqm $@' > /opt/mqm/bin/strmqm
		USER 1001`, imageName())},
	}
	tag := createImage(t, cli, files)
	defer deleteImage(t, cli, tag)

	containerConfig := ce.ContainerConfig{
		Env:   []string{"LICENSE=accept", "MQ_QMGR_NAME=qm1"},
		Image: tag,
	}
	id := runContainer(t, cli, &containerConfig)
	defer cleanContainer(t, cli, id)
	rc := waitForContainer(t, cli, id, 30*time.Second)
	if rc != 1 {
		t.Errorf("Expected rc=1, got rc=%v", rc)
	}
	expectTerminationMessage(t, cli, id)
}

// TestVolumeUnmount runs a queue manager with a volume, and then forces an
// unmount of the volume.  The health check should then fail.
// This simulates behaviour seen in some cloud environments, where network
// attached storage gets unmounted.
func TestVolumeUnmount(t *testing.T) {
	t.Parallel()

	cli := ce.NewContainerClient()
	vol := createVolume(t, cli, t.Name())
	defer removeVolume(t, cli, vol)
	containerConfig := ce.ContainerConfig{
		Image: imageName(),
		Env:   []string{"LICENSE=accept", "MQ_QMGR_NAME=qm1"},
	}
	hostConfig := ce.ContainerHostConfig{
		// SYS_ADMIN capability is required to unmount file systems
		CapAdd: []string{
			"SYS_ADMIN",
		},
		Binds: []string{
			coverageBind(t),
			vol + ":/mnt/mqm",
		},
	}
	networkingConfig := ce.ContainerNetworkSettings{}
	ctrID, err := cli.ContainerCreate(&containerConfig, &hostConfig, &networkingConfig, t.Name())
	if err != nil {
		t.Fatal(err)
	}
	startContainer(t, cli, ctrID)
	defer cleanContainer(t, cli, ctrID)
	waitForReady(t, cli, ctrID)
	// Unmount the volume as root
	rc, out := execContainer(t, cli, ctrID, "root", []string{"umount", "-l", "/mnt/mqm"})
	if rc != 0 {
		t.Fatalf("Expected umount to work with rc=0, got %v. Output was: %s", rc, out)
	}
	time.Sleep(3 * time.Second)
	rc, _ = execContainer(t, cli, ctrID, "", []string{"chkmqhealthy"})
	if rc == 0 {
		t.Errorf("Expected chkmqhealthy to fail")
		_, df := execContainer(t, cli, ctrID, "", []string{"df"})
		t.Logf(df)
		_, ps := execContainer(t, cli, ctrID, "", []string{"ps", "-ef"})
		t.Logf(ps)
	}
}

// TestZombies starts a queue manager, then causes a zombie process to be
// created, then checks that no zombies exist (runmqserver should reap them)
func TestZombies(t *testing.T) {
	t.Parallel()

	cli := ce.NewContainerClient()
	containerConfig := ce.ContainerConfig{
		Env:          []string{"LICENSE=accept", "MQ_QMGR_NAME=qm1", "DEBUG=true"},
		ExposedPorts: []string{"1414/tcp"},
	}
	id := runContainer(t, cli, &containerConfig)
	defer cleanContainer(t, cli, id)
	waitForReady(t, cli, id)
	// Kill an MQ process with children.  After it is killed, its children
	// will be adopted by PID 1, and should then be reaped when they die.
	_, out := execContainer(t, cli, id, "", []string{"pkill", "--signal", "kill", "-c", "amqzxma0"})
	if out == "0" {
		t.Log("Failed to kill process 'amqzxma0'")
		_, out := execContainer(t, cli, id, "", []string{"ps", "-lA"})
		t.Fatalf("Here is a list of currently running processes:\n%s", out)
	}
	time.Sleep(3 * time.Second)
	_, out = execContainer(t, cli, id, "", []string{"bash", "-c", "ps -lA | grep '^. Z'"})
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

	cli := ce.NewContainerClient()
	var files = []struct {
		Name, Body string
	}{
		{"Dockerfile", fmt.Sprintf(`
		  FROM %v
		  USER root
		  RUN rm -f /etc/mqm/*.mqsc
		  ADD test.mqsc /etc/mqm/
		  RUN chmod 0660 /etc/mqm/test.mqsc
		  USER 1001`, imageName())},
		{"test.mqsc", "DEFINE QLOCAL(test)"},
	}
	tag := createImage(t, cli, files)
	defer deleteImage(t, cli, tag)

	containerConfig := ce.ContainerConfig{
		Env:   []string{"LICENSE=accept", "MQ_QMGR_NAME=qm1"},
		Image: tag,
	}
	id := runContainer(t, cli, &containerConfig)
	defer cleanContainer(t, cli, id)
	waitForReady(t, cli, id)

	rc := -1
	mqscOutput := ""
	for i := 0; i < 60; i++ {
		rc, mqscOutput = execContainer(t, cli, id, "", []string{"bash", "-c", "echo 'DISPLAY QLOCAL(test)' | runmqsc"})
		if rc == 0 {
			return
		}
		time.Sleep(1 * time.Second)
	}
	if rc != 0 {
		r := regexp.MustCompile("AMQ[0-9][0-9][0-9][0-9]E")
		t.Fatalf("Expected runmqsc to exit with rc=0, got %v with error %v", rc, r.FindString(mqscOutput))
	}
}

// TestLargeMQSC creates a new image with a large MQSC file in, starts a container based
// on that image, and checks that the MQSC has been applied correctly.
func TestLargeMQSC(t *testing.T) {
	t.Parallel()

	cli := ce.NewContainerClient()
	const numQueues = 1000
	var buf bytes.Buffer
	for i := 1; i <= numQueues; i++ {
		fmt.Fprintf(&buf, "* Test processing of a large MQSC file, defining queue test%v\nDEFINE QLOCAL(test%v)\n", i, i)
	}
	var files = []struct {
		Name, Body string
	}{
		{"Dockerfile", fmt.Sprintf(`
          FROM %v
          USER root
          RUN rm -f /etc/mqm/*.mqsc
          ADD test.mqsc /etc/mqm/
          RUN chmod 0660 /etc/mqm/test.mqsc
          USER 1001`, imageName())},
		{"test.mqsc", buf.String()},
	}
	tag := createImage(t, cli, files)
	defer deleteImage(t, cli, tag)

	containerConfig := ce.ContainerConfig{
		Env:   []string{"LICENSE=accept", "MQ_QMGR_NAME=qm1"},
		Image: tag,
	}
	id := runContainer(t, cli, &containerConfig)
	defer cleanContainer(t, cli, id)
	waitForReady(t, cli, id)

	rc := -1
	mqscOutput := ""
	for i := 0; i < 60; i++ {
		rc, mqscOutput = execContainer(t, cli, id, "", []string{"bash", "-c", "echo 'DISPLAY QLOCAL(test" + strconv.Itoa(numQueues) + ")' | runmqsc"})
		if rc == 0 {
			return
		}
		time.Sleep(1 * time.Second)
	}
	if rc != 0 {
		r := regexp.MustCompile("AMQ[0-9][0-9][0-9][0-9]E")
		t.Fatalf("Expected runmqsc to exit with rc=0, got %v with error %v", rc, r.FindString(mqscOutput))
	}
}

// TestRedactValidMQSC creates a new image with a Valid MQSC file that contains sensitive information, starts a container based
// on that image, and checks that the MQSC has been redacted in the logs.
func TestRedactValidMQSC(t *testing.T) {
	t.Parallel()

	cli := ce.NewContainerClient()
	var buf bytes.Buffer
	passwords := "hippoman4567"
	sslcryp := fmt.Sprintf("GSK_PKCS11=/usr/lib/pkcs11/PKCS11_API.so;token-label;%s;SYMMETRIC_CIPHER_ON;", passwords)

	/* LDAPPWD*/
	fmt.Fprintf(&buf, "DEFINE AUTHINFO(TEST) AUTHTYPE(IDPWLDAP) CONNAME('test(24)') SHORTUSR('sn') LDAPUSER('user') LDAPPWD('%v')\n", passwords)
	fmt.Fprintf(&buf, "ALTER AUTHINFO(TEST) AUTHTYPE(IDPWLDAP) ldappwd('%v')\n", passwords)
	fmt.Fprintf(&buf, "ALTER AUTHINFO(TEST) AUTHTYPE(IDPWLDAP) lDaPpWd('%v')\n", passwords)
	fmt.Fprintf(&buf, "ALTER AUTHINFO(TEST) AUTHTYPE(IDPWLDAP) LDAPPWD \t('%v')\n", passwords)
	fmt.Fprintf(&buf, "ALTER AUTHINFO(TEST) AUTHTYPE(IDPWLDAP) +\n LDAP+\n PWD('%v')\n", passwords)
	fmt.Fprintf(&buf, "ALTER AUTHINFO(TEST) AUTHTYPE(IDPWLDAP) -\nLDAPP-\nWD('%v')\n", passwords)
	fmt.Fprintf(&buf, "ALTER AUTHINFO(TEST) AUTHTYPE(IDPWLDAP) +\n*test comment\n LDAPP-\n*test comment2\nWD('%v')\n", passwords)
	fmt.Fprintf(&buf, "ALTER AUTHINFO(TEST) AUTHTYPE(IDPWLDAP) LDAPPWD(%v)\n", passwords)

	/* PASSWORD */
	fmt.Fprintf(&buf, "DEFINE CHANNEL(TEST2) CHLTYPE(SDR) CONNAME('test(24)') XMITQ('fake') PASSWORD('%v')\n", passwords)
	fmt.Fprintf(&buf, "ALTER CHANNEL(TEST2) CHLTYPE(SDR) password('%v')\n", passwords)
	fmt.Fprintf(&buf, "ALTER CHANNEL(TEST2) CHLTYPE(SDR) pAsSwOrD('%v')\n", passwords)
	fmt.Fprintf(&buf, "ALTER CHANNEL(TEST2) CHLTYPE(SDR) PASSWORD \t('%v')\n", passwords)
	fmt.Fprintf(&buf, "ALTER CHANNEL(TEST2) +\n CHLTYPE(SDR) PASS+\n WORD+\n ('%v')\n", passwords)
	fmt.Fprintf(&buf, "ALTER CHANNEL(TEST2) -\nCHLTYPE(SDR) PASS-\nWORD-\n('%v')\n", passwords)
	fmt.Fprintf(&buf, "ALTER CHANNEL(TEST2) +\n CHLTYPE(SDR) PASS-\n*comemnt 2\nWORD+\n*test comment\n ('%v')\n", passwords)
	fmt.Fprintf(&buf, "ALTER CHANNEL(TEST2) CHLTYPE(SDR) PASSWORD(%s)\n", passwords)

	/* SSLCRYP */
	fmt.Fprintf(&buf, "ALTER QMGR SSLCRYP('%v')\n", sslcryp)
	fmt.Fprintf(&buf, "ALTER QMGR sslcryp('%v')\n", sslcryp)
	fmt.Fprintf(&buf, "ALTER QMGR SsLcRyP('%v')\n", sslcryp)
	fmt.Fprintf(&buf, "ALTER QMGR SSLCRYP \t('%v')\n", sslcryp)
	fmt.Fprintf(&buf, "ALTER QMGR +\n SSL+\n CRYP+\n ('%v')\n", sslcryp)
	fmt.Fprintf(&buf, "ALTER QMGR -\nSSLC-\nRYP-\n('%v')\n", sslcryp)
	fmt.Fprintf(&buf, "ALTER QMGR +\n*commenttime\n SSL-\n*commentagain\nCRYP+\n*last comment\n ('%v')\n", sslcryp)

	var files = []struct {
		Name, Body string
	}{
		{"Dockerfile", fmt.Sprintf(`
		  FROM %v
		  USER root
		  RUN rm -f /etc/mqm/*.mqsc
		  ADD test.mqsc /etc/mqm/
		  RUN chmod 0660 /etc/mqm/test.mqsc
		  USER 1001`, imageName())},
		{"test.mqsc", buf.String()},
	}
	tag := createImage(t, cli, files)
	defer deleteImage(t, cli, tag)

	containerConfig := ce.ContainerConfig{
		Env:   []string{"LICENSE=accept", "MQ_QMGR_NAME=qm1"},
		Image: tag,
	}
	id := runContainer(t, cli, &containerConfig)
	defer cleanContainer(t, cli, id)
	waitForReady(t, cli, id)
	stopContainer(t, cli, id)
	scanner := bufio.NewScanner(strings.NewReader(inspectLogs(t, cli, id)))
	for scanner.Scan() {
		s := scanner.Text()
		if strings.Contains(s, sslcryp) || strings.Contains(s, passwords) {
			t.Fatalf("Expected redacted MQSC output, got: %v", s)
		}
	}
	err := scanner.Err()
	if err != nil {
		t.Fatal(err)
	}
}

// TestRedactValidMQSC creates a new image with a Invalid MQSC file that contains sensitive information, starts a container based
// on that image, and checks that the MQSC has been redacted in the logs.
func TestRedactInvalidMQSC(t *testing.T) {
	t.Parallel()

	cli := ce.NewContainerClient()
	var buf bytes.Buffer
	passwords := "hippoman4567"
	sslcryp := fmt.Sprintf("GSK_PKCS11=/usr/lib/pkcs11/PKCS11_API.so;token-label;%s;SYMMETRIC_CIPHER_ON;", passwords)

	/* LDAPPWD*/
	fmt.Fprintf(&buf, "DEFINE AUTHINFO(TEST) AUTHTYPE(IDPWLDAP) CONNAME('test(24)') SHORTUSR('sn') LDAPUSER('user') LDAPPWD('%v')\n", passwords)
	fmt.Fprintf(&buf, "ALTER AUTHINFO(TEST) AUTHTYPE(IDPWLDAP) LDAPPPPPPP('%v')\n", passwords)
	fmt.Fprintf(&buf, "ALTER AUTHINFO(TEST) AUTHTYPE(IDPWLDAP) LDAPPWD['%v']\n", passwords)
	fmt.Fprintf(&buf, "ALTER AUTHINFO(TEST) AUTHTYPE(ARGHHH) LDAPPWD('%v')\n", passwords)
	fmt.Fprintf(&buf, "ALTER AUTHINFO(TEST) LDAPPWD('%v')\n", passwords)
	fmt.Fprintf(&buf, "ALTER AUTHINFO(TEST) ARGHAHA(IDPWLDAP) LDAPPWD('%v')\n", passwords)
	fmt.Fprintf(&buf, "ALTER AUTHINFO(TEST) AUTHTYPE(IDPWLDAP) LDAPPWD '%v'\n", passwords)
	fmt.Fprintf(&buf, "ALTER AUTHINFO(TEST) AUTHTYPE(IDPWLDAP) LDAPPWD('%v') badvalues\n", passwords)
	fmt.Fprintf(&buf, "ALTER AUTHINFO(TEST) AUTHTYPE(IDPWLDAP) badvales LDAPPWD('%v')\n", passwords)
	fmt.Fprintf(&buf, "ALTER AUTHINFO(TEST) AUTHTYPE(IDPWLDAP) LDAPPWD{'%v'}\n", passwords)
	fmt.Fprintf(&buf, "ALTER AUTHINFO(TEST) AUTHTYPE(IDPWLDAP) LDAPPWD<'%v'>\n", passwords)
	fmt.Fprintf(&buf, "ALTER AUTHINFO(TEST) AUTHTYPE(IDPWLDAP) LDAPPWD('%v'+\n p['il6])\n", passwords)
	fmt.Fprintf(&buf, "ALTER AUTHINFO(TEST) AUTHTYPE(IDPWLDAP) LDAPPWD('%v'/653***)\n", passwords)
	fmt.Fprintf(&buf, "ALTER AUTHINFO(TEST) LDAPPWD('%v'\n DISPLAY QMGR", passwords)
	fmt.Fprintf(&buf, "ALTER AUTHINFO(TEST) LDAPPWD('%v💩')\n", passwords)
	fmt.Fprintf(&buf, "ALTER AUTHINFO(TEST) LDAPPWD💩('%v')\n", passwords)
	fmt.Fprintf(&buf, "ALTER AUTHINFO(TEST) LDAP+\n 💩PWD('%v')\n", passwords)
	fmt.Fprintf(&buf, "ALTER AUTHINFO(TEST) 💩 LDAPPWD('%v')\n", passwords)
	fmt.Fprintf(&buf, "ALTER AUTHINFO(TEST) LDAPPWD 💩 ('%v')\n", passwords)
	fmt.Fprintf(&buf, "ALTER AUTHINFO(TEST) LDAPPWD('%v') 💩\n", passwords)
	fmt.Fprintf(&buf, "ALTER 💩 AUTHINFO(TEST) LDAPPWD('%v')\n", passwords)

	var files = []struct {
		Name, Body string
	}{
		{"Dockerfile", fmt.Sprintf(`
		  FROM %v
		  USER root
		  RUN rm -f /etc/mqm/*.mqsc
		  ADD test.mqsc /etc/mqm/
		  RUN chmod 0660 /etc/mqm/test.mqsc
		  USER 1001`, imageName())},
		{"test.mqsc", buf.String()},
	}
	tag := createImage(t, cli, files)
	defer deleteImage(t, cli, tag)

	containerConfig := ce.ContainerConfig{
		Env:   []string{"LICENSE=accept", "MQ_QMGR_NAME=qm1"},
		Image: tag,
	}
	id := runContainer(t, cli, &containerConfig)
	defer cleanContainer(t, cli, id)
	rc := waitForContainer(t, cli, id, 30*time.Second)
	if rc != 1 {
		t.Errorf("Expected rc=1, got rc=%v", rc)
	}
	scanner := bufio.NewScanner(strings.NewReader(inspectLogs(t, cli, id)))
	for scanner.Scan() {
		s := scanner.Text()
		if strings.Contains(s, sslcryp) || strings.Contains(s, passwords) {
			t.Fatalf("Expected redacted MQSC output, got: %v", s)
		}
	}
	err := scanner.Err()
	if err != nil {
		t.Fatal(err)
	}
}

// TestInvalidMQSC creates a new image with an MQSC file containing invalid MQSC,
// tries to start a container based on that image, and checks that container terminates
func TestInvalidMQSC(t *testing.T) {
	t.Parallel()
	cli := ce.NewContainerClient()
	var files = []struct {
		Name, Body string
	}{
		{"Dockerfile", fmt.Sprintf(`
		FROM %v
		USER root
		RUN rm -f /etc/mqm/*.mqsc
		ADD mqscTest.mqsc /etc/mqm/
		RUN chmod 0660 /etc/mqm/mqscTest.mqsc
		USER 1001`, imageName())},
		{"mqscTest.mqsc", "DEFINE INVALIDLISTENER('TEST.LISTENER.TCP') TRPTYPE(TCP) PORT(1414) CONTROL(QMGR) REPLACE"},
	}
	tag := createImage(t, cli, files)
	defer deleteImage(t, cli, tag)

	containerConfig := ce.ContainerConfig{
		Env:   []string{"LICENSE=accept", "MQ_QMGR_NAME=qm1"},
		Image: tag,
	}
	id := runContainer(t, cli, &containerConfig)
	defer cleanContainer(t, cli, id)
	rc := waitForContainer(t, cli, id, 60*time.Second)
	if rc != 1 {
		t.Errorf("Expected rc=1, got rc=%v", rc)
	}
	expectTerminationMessage(t, cli, id)
}

func TestSimpleMQIniMerge(t *testing.T) {
	t.Parallel()
	cli := ce.NewContainerClient()
	var files = []struct {
		Name, Body string
	}{
		{"Dockerfile", fmt.Sprintf(`
		FROM %v
		USER root
		ADD test1.ini /etc/mqm/
		RUN chmod 0660 /etc/mqm/test1.ini
		USER 1001`, imageName())},
		{"test1.ini",
			"Log:\n   LogSecondaryFiles=28"},
	}
	tag := createImage(t, cli, files)
	defer deleteImage(t, cli, tag)

	containerConfig := ce.ContainerConfig{
		Env:   []string{"LICENSE=accept", "MQ_QMGR_NAME=qm1"},
		Image: tag,
	}
	id := runContainer(t, cli, &containerConfig)
	defer cleanContainer(t, cli, id)
	waitForReady(t, cli, id)

	catIniFileCommand := fmt.Sprintf("cat /var/mqm/qmgrs/qm1/qm.ini")
	_, test := execContainer(t, cli, id, "", []string{"bash", "-c", catIniFileCommand})
	merged := strings.Contains(test, "LogSecondaryFiles=28")

	if !merged {
		t.Error("ERROR: The Files are not merged correctly")
	}

}
func TestMultipleIniMerge(t *testing.T) {
	t.Parallel()
	cli := ce.NewContainerClient()
	var files = []struct {
		Name, Body string
	}{
		{"Dockerfile", fmt.Sprintf(`
		FROM %v
		USER root
		ADD test1.ini /etc/mqm/
		ADD test2.ini /etc/mqm/
		ADD test3.ini /etc/mqm/
		RUN chmod 0660 /etc/mqm/test1.ini
		RUN chmod 0660 /etc/mqm/test2.ini
		RUN chmod 0660 /etc/mqm/test3.ini
		USER 1001`, imageName())},
		{"test1.ini",
			"Log:\n LogSecondaryFiles=28"},
		{"test2.ini",
			"Log:\n LogSecondaryFiles=28"},
		{"test3.ini",
			"ApplicationTrace:\n   ApplName=amqsact*\n   Trace=OFF"},
	}
	tag := createImage(t, cli, files)
	defer deleteImage(t, cli, tag)

	containerConfig := ce.ContainerConfig{
		Env:   []string{"LICENSE=accept", "MQ_QMGR_NAME=qm1"},
		Image: tag,
	}
	id := runContainer(t, cli, &containerConfig)
	defer cleanContainer(t, cli, id)
	waitForReady(t, cli, id)

	catIniFileCommand := fmt.Sprintf("cat /var/mqm/qmgrs/qm1/qm.ini")
	_, test := execContainer(t, cli, id, "", []string{"bash", "-c", catIniFileCommand})

	//checks that no duplicates are created by adding 2 ini files with the same line
	numberOfDuplicates := strings.Count(test, "LogSecondaryFiles=28")

	newStanza := strings.Contains(test, "ApplicationTrace:\n   ApplName=amqsact*")

	if (numberOfDuplicates > 1) || !newStanza {
		t.Error("ERROR: The Files are not merged correctly")
	}
}

func TestMQIniMergeOnTheSameVolumeButTwoContainers(t *testing.T) {
	cli := ce.NewContainerClient()

	var filesFirstContainer = []struct {
		Name, Body string
	}{
		{"Dockerfile", fmt.Sprintf(`
		FROM %v
		USER root
		ADD test1.ini /etc/mqm/
		RUN chmod 0660 /etc/mqm/test1.ini
		USER 1001`, imageName())},
		{"test1.ini",
			"ApplicationTrace:\n   ApplName=amqsact*\n   Trace=OFF"},
	}
	firstImage := createImage(t, cli, filesFirstContainer)
	defer deleteImage(t, cli, firstImage)
	vol := createVolume(t, cli, t.Name())
	defer removeVolume(t, cli, vol)

	containerConfig := ce.ContainerConfig{
		Image: firstImage,
		Env:   []string{"LICENSE=accept", "MQ_QMGR_NAME=qm1"},
	}

	hostConfig := ce.ContainerHostConfig{
		Binds: []string{
			coverageBind(t),
			vol + ":/mnt/mqm",
		},
	}
	networkingConfig := ce.ContainerNetworkSettings{}
	ctr1ID, err := cli.ContainerCreate(&containerConfig, &hostConfig, &networkingConfig, t.Name())
	if err != nil {
		t.Fatal(err)
	}

	startContainer(t, cli, ctr1ID)
	waitForReady(t, cli, ctr1ID)

	catIniFileCommand := fmt.Sprintf("cat /var/mqm/qmgrs/qm1/qm.ini")
	_, test := execContainer(t, cli, ctr1ID, "", []string{"bash", "-c", catIniFileCommand})
	addedStanza := strings.Contains(test, "ApplicationTrace:\n   ApplName=amqsact*\n   Trace=OFF")

	if addedStanza != true {
		t.Error("ERROR: The Files are not merged correctly")
	}
	// Delete the first container
	cleanContainer(t, cli, ctr1ID)

	var filesSecondContainer = []struct {
		Name, Body string
	}{
		{"Dockerfile", fmt.Sprintf(`
		FROM %v
		USER root
		ADD test1.ini /etc/mqm/
		RUN chmod 0660 /etc/mqm/test1.ini
		USER 1001`, imageName())},
		{"test1.ini",
			"Log:\n   LogBufferPages=128"},
	}

	secondImage := createImage(t, cli, filesSecondContainer)
	defer deleteImage(t, cli, secondImage)

	containerConfig2 := ce.ContainerConfig{
		Image: secondImage,
		Env:   []string{"LICENSE=accept", "MQ_QMGR_NAME=qm1"},
	}

	ctr2ID, err := cli.ContainerCreate(&containerConfig2, &hostConfig, &networkingConfig, t.Name())
	if err != nil {
		t.Fatal(err)
	}
	defer cleanContainer(t, cli, ctr2ID)
	startContainer(t, cli, ctr2ID)
	waitForReady(t, cli, ctr2ID)

	_, test2 := execContainer(t, cli, ctr2ID, "", []string{"bash", "-c", catIniFileCommand})
	changedStanza := strings.Contains(test2, "LogBufferPages=128")
	//check if stanza that was merged in the first container doesnt exist in this one.
	firstMergedStanza := strings.Contains(test2, "ApplicationTrace:\n   ApplName=amqsact*\n   Trace=OFF")

	if !changedStanza || firstMergedStanza {
		t.Error("ERROR: The Files are not merged correctly after removing first container")
	}

}

// TestReadiness creates a new image with large amounts of MQSC in, to
// ensure that the readiness check doesn't pass until configuration has finished.
// WARNING: This test is sensitive to the speed of the machine it's running on.
func TestReadiness(t *testing.T) {
	t.Parallel()

	cli := ce.NewContainerClient()
	const numQueues = 3
	var buf bytes.Buffer
	for i := 1; i <= numQueues; i++ {
		fmt.Fprintf(&buf, "* Defining queue test %v\nDEFINE QLOCAL(test%v)\n", i, i)
	}
	var files = []struct {
		Name, Body string
	}{
		{"Dockerfile", fmt.Sprintf(`
		FROM %v
		USER root
		RUN rm -f /etc/mqm/*.mqsc
		ADD test.mqsc /etc/mqm/
		RUN chmod 0660 /etc/mqm/test.mqsc
		USER 1001`, imageName())},
		{"test.mqsc", buf.String()},
	}
	tag := createImage(t, cli, files)
	defer deleteImage(t, cli, tag)

	containerConfig := ce.ContainerConfig{
		Env:   []string{"LICENSE=accept", "MQ_QMGR_NAME=qm1", "DEBUG=1"},
		Image: tag,
	}
	id := runContainer(t, cli, &containerConfig)
	defer cleanContainer(t, cli, id)
	queueCheckCommand := fmt.Sprintf("echo 'DISPLAY QLOCAL(test%v)' | runmqsc", numQueues)
	_, mqsc := execContainer(t, cli, id, "", []string{"cat", "/etc/mqm/test.mqsc"})
	t.Log(mqsc)
	for {
		readyRC, _ := execContainer(t, cli, id, "", []string{"chkmqready"})
		if readyRC == 0 {
			queueCheckRC := -1
			queueCheckOut := ""
			for i := 1; i < 60; i++ {
				queueCheckRC, queueCheckOut = execContainer(t, cli, id, "", []string{"bash", "-c", queueCheckCommand})
				t.Logf("readyRC=%v,queueCheckRC=%v\n", readyRC, queueCheckRC)
				if queueCheckRC == 0 {
					break
				}
				time.Sleep(1 * time.Second)
			}
			if queueCheckRC != 0 {
				r := regexp.MustCompile("AMQ[0-9][0-9][0-9][0-9]E")
				t.Fatalf("Runmqsc returned %v with error %v. chkmqready returned %v when MQSC had not finished", queueCheckRC, r.FindString(queueCheckOut), readyRC)
			} else {
				// chkmqready says OK, and the last queue exists, so return
				_, runmqsc := execContainer(t, cli, id, "", []string{"bash", "-c", "echo 'DISPLAY QLOCAL(test1)' | runmqsc"})
				t.Log(runmqsc)
				return
			}
		}
	}
}

func TestErrorLogRotation(t *testing.T) {
	t.Skipf("Skipping %v until test defect fixed", t.Name())
	t.Parallel()

	cli := ce.NewContainerClient()

	logsize := 65536

	rc, _ := runContainerOneShot(t, cli, "bash", "-c", "test -d /etc/apt")
	if rc != 0 {
		// RHEL
		logsize = 32768
	}

	qmName := "qm1"
	containerConfig := ce.ContainerConfig{
		Env: []string{
			"LICENSE=accept",
			"MQ_QMGR_NAME=" + qmName,
			fmt.Sprintf("MQMAXERRORLOGSIZE=%d", logsize),
			"LOG_FORMAT=json",
			fmt.Sprintf("AMQ_EXTRA_QM_STANZAS=QMErrorLog:ErrorLogSize=%d", logsize),
		},
		ExposedPorts: []string{"1414/tcp"},
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

		_, atoiStr := execContainer(t, cli, id, "", []string{"bash", "-c", "wc -c < " + filepath.Join(dir, "AMQERR02.json")})
		amqerr02size, _ := strconv.Atoi(atoiStr)

		if amqerr02size > 0 {
			// We've done enough to cause log rotation
			break
		}
	}
	_, out := execContainer(t, cli, id, "", []string{"ls", "-l", dir})
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
	err := scanner.Err()
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
	cli := ce.NewContainerClient()
	containerConfig := ce.ContainerConfig{
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
	err := scanner.Err()
	if err != nil {
		t.Fatal(err)
	}
}

// TestMQJSONDisabled tests the case where MQ's JSON logging feature is
// specifically disabled (which will disable log mirroring)
func TestMQJSONDisabled(t *testing.T) {
	t.SkipNow()
	t.Parallel()
	cli := ce.NewContainerClient()
	containerConfig := ce.ContainerConfig{
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

	cli := ce.NewContainerClient()

	containerConfig := ce.ContainerConfig{
		Env: []string{"LICENSE=accept"},
	}
	id := runContainer(t, cli, &containerConfig)
	defer cleanContainer(t, cli, id)
	waitForReady(t, cli, id)

	rc, license := execContainer(t, cli, id, "", []string{"dspmqver", "-f", "8192", "-b"})
	if rc != 0 {
		t.Fatalf("Failed to get license string. RC=%d. Output=%s", rc, license)
	}
	license = ce.SanitizeString(license)

	if license != expectedLicense {
		t.Errorf("Expected license to be '%s' but was '%s", expectedLicense, license)
	}
}

func TestVersioning(t *testing.T) {
	t.Parallel()

	cli := ce.NewContainerClient()

	containerConfig := ce.ContainerConfig{
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
	// foundRevision := false
	// foundSource := false
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

		// if strings.Contains(line, "Image revision:") && !foundRevision {
		// 	total--
		// 	foundRevision = true
		// 	dataAr := strings.Split(line, " ")
		// 	data := dataAr[len(dataAr)-1]

		// 	// Verify revision
		// 	pattern := regexp.MustCompile("^[a-fA-F0-9]{40}$")
		// 	if !pattern.MatchString(data) {
		// 		t.Errorf("Failed to validate revision (%v)", data)
		// 	}
		// }

		// if strings.Contains(line, "Image source:") && !foundSource {
		// 	total--
		// 	foundSource = true
		// 	dataAr := strings.Split(line, " ")
		// 	data := dataAr[len(dataAr)-1]

		// 	// Verify source
		// 	if !strings.Contains(data, "github") {
		// 		t.Errorf("Failed to validate source (%v)", data)
		// 	}
		// }

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

	// if !foundCreated || !foundRevision || !foundSource || !foundMQVersion || !foundMQLevel || !foundMQLicense {
	if !foundCreated || !foundMQVersion || !foundMQLevel || !foundMQLicense {

		// t.Errorf("Failed to find one or more version strings: created(%v) revision(%v) source(%v) mqversion(%v) mqlevel(%v) mqlicense(%v)", foundCreated, foundRevision, foundSource, foundMQVersion, foundMQLevel, foundMQLicense)
		t.Errorf("Failed to find one or more version strings: created(%v) mqversion(%v) mqlevel(%v) mqlicense(%v)", foundCreated, foundMQVersion, foundMQLevel, foundMQLicense)

	}
}

func TestTraceStrmqm(t *testing.T) {
	t.Parallel()

	cli := ce.NewContainerClient()

	containerConfig := ce.ContainerConfig{
		Env: []string{
			"LICENSE=accept",
			"MQ_ENABLE_TRACE_STRMQM=1",
		},
	}
	id := runContainer(t, cli, &containerConfig)
	defer cleanContainer(t, cli, id)
	waitForReady(t, cli, id)

	rc, _ := execContainer(t, cli, id, "", []string{"bash", "-c", "ls -A /var/mqm/trace | grep .TRC"})
	if rc != 0 {
		t.Fatalf("No trace files found in trace directory /var/mqm/trace. RC=%d.", rc)
	}
}

// utilTestHealthCheck is used by TestHealthCheck* to run a container with
// privileges enabled or disabled.  Otherwise the same as the golden path tests.
func utilTestHealthCheck(t *testing.T, nonewpriv bool) {
	t.Parallel()
	cli := ce.NewContainerClient()
	containerConfig := ce.ContainerConfig{
		Env: []string{"LICENSE=accept", "MQ_QMGR_NAME=qm1"},
	}
	hostConfig := getDefaultHostConfig(t, cli)
	hostConfig.SecurityOpt = append(hostConfig.SecurityOpt, fmt.Sprintf("no-new-privileges:%v", nonewpriv))
	id := runContainerWithHostConfig(t, cli, &containerConfig, hostConfig)
	defer cleanContainer(t, cli, id)
	waitForReady(t, cli, id)
	rc, out := execContainer(t, cli, id, "", []string{"chkmqhealthy"})
	t.Log(out)
	if rc != 0 {
		t.Errorf("Expected chkmqhealthy to return with exit code 0; got \"%v\"", rc)
		t.Logf("Output from chkmqhealthy:\n%v", out)
	}
	// Stop the container cleanly
	stopContainer(t, cli, id)
}

// TestHealthCheckWithNoNewPrivileges tests golden path start/stop plus
// chkmqhealthy, when running in a container where no new privileges are
// allowed (i.e. setuid is disabled)
func TestHealthCheckWithNoNewPrivileges(t *testing.T) {
	utilTestHealthCheck(t, true)
}

// TestHealthCheckWithNewPrivileges tests golden path start/stop plus
// chkmqhealthy when running in a container where new privileges are
// allowed (i.e. setuid is allowed)
// See https://github.com/ibm-messaging/mq-container/issues/428
func TestHealthCheckWithNewPrivileges(t *testing.T) {
	utilTestHealthCheck(t, false)
}

// utilTestStartedCheck is used by TestStartedCheck* to run a container with
// privileges enabled or disabled.  Otherwise the same as the golden path tests.
func utilTestStartedCheck(t *testing.T, nonewpriv bool) {
	t.Parallel()
	cli := ce.NewContainerClient()
	containerConfig := ce.ContainerConfig{
		Env: []string{"LICENSE=accept", "MQ_QMGR_NAME=qm1"},
	}
	hostConfig := getDefaultHostConfig(t, cli)
	hostConfig.SecurityOpt = append(hostConfig.SecurityOpt, fmt.Sprintf("no-new-privileges:%v", nonewpriv))
	id := runContainerWithHostConfig(t, cli, &containerConfig, hostConfig)
	defer cleanContainer(t, cli, id)
	waitForReady(t, cli, id)
	rc, out := execContainer(t, cli, id, "", []string{"chkmqstarted"})
	t.Log(out)
	if rc != 0 {
		t.Errorf("Expected chkmqstarted to return with exit code 0; got \"%v\"", rc)
		t.Logf("Output from chkmqstarted:\n%v", out)
	}
	// Stop the container cleanly
	stopContainer(t, cli, id)
}

// TestStartedCheckWithNoNewPrivileges tests golden path start/stop plus
// chkmqstarted, when running in a container where no new privileges are
// allowed (i.e. setuid is disabled)
func TestStartedCheckWithNoNewPrivileges(t *testing.T) {
	utilTestStartedCheck(t, true)
}

// TestStartedCheckWithNewPrivileges tests golden path start/stop plus
// chkmqstarted when running in a container where new privileges are
// allowed (i.e. setuid is allowed)
// See https://github.com/ibm-messaging/mq-container/issues/428
func TestStartedCheckWithNewPrivileges(t *testing.T) {
	utilTestStartedCheck(t, false)
}

// Start a container with qm grace set to x seconds
// Check that when the container is stopped that the command endmqm has option -tp and x
func TestEndMQMOpts(t *testing.T) {
	t.Parallel()
	cli := ce.NewContainerClient()
	containerConfig := ce.ContainerConfig{
		Env: []string{"LICENSE=accept", "MQ_GRACE_PERIOD=27"},
	}

	id := runContainer(t, cli, &containerConfig)
	defer cleanContainer(t, cli, id)
	waitForReady(t, cli, id)
	killContainer(t, cli, id, "SIGTERM")
	_, out := execContainer(t, cli, id, "", []string{"bash", "-c", "ps -ef | grep 'endmqm -w -r -tp 27'"})
	t.Log(out)
	if !strings.Contains(out, "endmqm -w -r -tp 27") {
		t.Errorf("Expected endmqm options endmqm -w -r -tp 27; got \"%v\"", out)
	}
}

// TestCustomLogFilePages starts a qmgr with a custom number of logfilepages set.
// Check that the number of logfilepages matches.
func TestCustomLogFilePages(t *testing.T) {
	t.Parallel()
	cli := ce.NewContainerClient()
	containerConfig := ce.ContainerConfig{
		Env: []string{"LICENSE=accept", "MQ_QMGR_LOG_FILE_PAGES=8192", "MQ_QMGR_NAME=qmlfp"},
	}

	id := runContainer(t, cli, &containerConfig)
	defer cleanContainer(t, cli, id)
	waitForReady(t, cli, id)

	testLogFilePages(t, cli, id, "qmlfp", "8192")
}

// TestLoggingConsoleSource tests default behavior which is
// MQ_LOGGING_CONSOLE_SOURCE set to qmgr,web
func TestLoggingConsoleSource(t *testing.T) {

	t.Parallel()
	cli := ce.NewContainerClient()
	containerConfig := ce.ContainerConfig{
		Env: []string{
			"LICENSE=accept",
			"MQ_QMGR_NAME=qm1",
			"MQ_ENABLE_EMBEDDED_WEB_SERVER=true",
		},
	}
	id := runContainer(t, cli, &containerConfig)
	defer cleanContainer(t, cli, id)
	waitForReady(t, cli, id)

	jsonLogs, errJson := waitForMessageInLog(t, cli, id, "AMQ6206I")
	if errJson != nil {
		t.Errorf("%v", errJson)
	}
	
	//Check for web server logs existence in console logs since its visibility is default along with qmgr logs
	jsonLogs, errJson = waitForMessageInLog(t, cli, id, "CWWKF0011I")
	if errJson != nil {
		t.Errorf("%v", errJson)
	}

	isMessageFound := scanForExcludedEntries(jsonLogs)

	if isMessageFound == true {
		t.Errorf("Expected  to exclude messageId by default; but messageId \"%v\" is present", jsonLogs)
	}

	// Stop the container cleanly
	stopContainer(t, cli, id)
}

// TestOldBehaviorWebConsole sets LOG_FORMAT to json and verify logs are indeed in json format
func TestOldBehaviorWebConsole(t *testing.T) {

	t.Parallel()
	cli := ce.NewContainerClient()
	containerConfig := ce.ContainerConfig{
		Env: []string{
			"LICENSE=accept",
			"MQ_QMGR_NAME=qm1",
			"LOG_FORMAT=json",
		},
	}
	id := runContainer(t, cli, &containerConfig)
	defer cleanContainer(t, cli, id)
	waitForReady(t, cli, id)
	jsonLogs := inspectLogs(t, cli, id)

	isMessageFound := scanForExcludedEntries(jsonLogs)

	if isMessageFound == true {
		t.Errorf("Expected  to exclude messageId by default; but messageId \"%v\" is present", jsonLogs)
	}

	if strings.Contains(jsonLogs, "Environment variable LOG_FORMAT is deprecated. Use MQ_LOGGING_CONSOLE_FORMAT instead.") {
		t.Logf("Expected Message stating LOG_FORMAT is deprecated is present in the log")
	} else {
		t.Errorf("Expected Message stating LOG_FORMAT is deprecated is not in the log")
	}

	isValidJSON := checkLogForValidJSON(jsonLogs)

	if !isValidJSON {
		t.Fatalf("Expected all log lines to be valid JSON.  Logs: %v ", jsonLogs)
	}

	// Stop the container cleanly
	stopContainer(t, cli, id)
}

// TestLoggingConsoleWithContRestart restarts the container and checks
// that setting of env variable persists
func TestLoggingConsoleWithContRestart(t *testing.T) {

	t.Parallel()

	cli := ce.NewContainerClient()
	containerConfig := ce.ContainerConfig{
		Env: []string{
			"LICENSE=accept",
			"MQ_QMGR_NAME=qm1",
			"MQ_LOGGING_CONSOLE_SOURCE=qmgr",
		},
	}
	id := runContainer(t, cli, &containerConfig)

	defer cleanContainer(t, cli, id)
	waitForReady(t, cli, id)

	jsonLogs, errJson := waitForMessageInLog(t, cli, id, "AMQ6206I")
	if errJson != nil {
		t.Errorf("%v", errJson)
	}

	isMessageFound := scanForExcludedEntries(jsonLogs)

	if isMessageFound == true {
		t.Errorf("Expected  to exclude messageId by default; but messageId \"%v\" is present", jsonLogs)
	}

	stopContainer(t, cli, id)
	startContainer(t, cli, id)
	waitForReady(t, cli, id)

	jsonLogs = inspectLogs(t, cli, id)

	if !strings.Contains(jsonLogs, "Stopped queue manager") || strings.Contains(jsonLogs, "CWWKF0011I") {
		t.Errorf("CWWKF0011I which is not expected is present!!!!!")
	}

	isMessageFoundAfterRestart := scanForExcludedEntries(jsonLogs)

	if isMessageFoundAfterRestart == true {
		t.Errorf("Expected  to exclude messageId by default; but messageId \"%v\" is present", jsonLogs)
	}

	// Stop the container cleanly
	stopContainer(t, cli, id)
}

// TestLoggingWithQmgrAndExcludeId tests MQ_LOGGING_CONSOLE_SOURCE set to qmgr
// and  exclude ID set to amq7230I.

func TestLoggingWithQmgrAndExcludeId(t *testing.T) {
	qmgrName := "qm1"

	t.Parallel()

	cli := ce.NewContainerClient()
	containerConfig := ce.ContainerConfig{
		Env: []string{
			"LICENSE=accept",
			"MQ_QMGR_NAME=" + qmgrName,
			"MQ_LOGGING_CONSOLE_SOURCE=qmgr",
			"MQ_LOGGING_CONSOLE_FORMAT=json",
			"MQ_LOGGING_CONSOLE_EXCLUDE_ID=amq7230I",
		},
	}

	dir := "/var/mqm/qmgrs/" + qmgrName + "/errors"

	id := runContainer(t, cli, &containerConfig)
	defer cleanContainer(t, cli, id)
	waitForReady(t, cli, id)

	jsonLogs, errJson := waitForMessageInLog(t, cli, id, "AMQ6206I")
	if errJson != nil {
		t.Errorf("%v", errJson)
	}

	isValidJSON := checkLogForValidJSON(jsonLogs)

	if !isValidJSON {
		t.Fatalf("Expected all log lines to be valid JSON.  Logs: %v ", jsonLogs)
	}

	if strings.Contains(jsonLogs, "AMQ7230I") || strings.Contains(jsonLogs, "CWWKF0011I") {
		t.Errorf("Expected to exclude messageId by default; but messageId \"%v\" is present", jsonLogs)
	}

	stopContainer(t, cli, id)

	//checking that message is only excluded from the console log, but not from the MQ error log
	b := copyFromContainer(t, cli, id, filepath.Join(dir, "AMQERR01.json"))

	foundInLog := 0
	r := bytes.NewReader(b)
	scannernew := bufio.NewScanner(r)
	for scannernew.Scan() {
		textData := scannernew.Text()

		if strings.Contains(textData, "AMQ7230I") {
			foundInLog = 1
		}
	}
	if foundInLog == 0 {
		t.Errorf("mesageID AMQ7230I is not present in MQ LOG!!!!")
	}

}

// TestLoggingConsoleSetToWeb tests MQ_LOGGING_CONSOLE_SOURCE set to web
func TestLoggingConsoleSetToWeb(t *testing.T) {

	t.Parallel()
	cli := ce.NewContainerClient()
	containerConfig := ce.ContainerConfig{
		Env: []string{
			"LICENSE=accept",
			"MQ_QMGR_NAME=qm1",
			"MQ_ENABLE_EMBEDDED_WEB_SERVER=true",
			"MQ_LOGGING_CONSOLE_SOURCE=web",
			"MQ_LOGGING_CONSOLE_EXCLUDE_ID=CWWKG0028A,CWWKS4105I",
			"MQ_LOGGING_CONSOLE_FORMAT=json",
		},
	}
	id := runContainer(t, cli, &containerConfig)
	defer cleanContainer(t, cli, id)
	waitForReady(t, cli, id)

	jsonLogs, errJson := waitForMessageInLog(t, cli, id, "CWWKF0011I")
	if errJson != nil {
		t.Errorf("%v", errJson)
	}

	if strings.Contains(jsonLogs, "AMQ6206I") {
		t.Errorf("Logging source is set to web, Qmgr message  \"%v\" should be excluded!!!", jsonLogs)
	}

	if strings.Contains(jsonLogs, "AMQ5041I") || strings.Contains(jsonLogs, "AMQ5052I") ||
		strings.Contains(jsonLogs, "AMQ5051I") || strings.Contains(jsonLogs, "AMQ5037I") ||
		strings.Contains(jsonLogs, "AMQ5975I") || strings.Contains(jsonLogs, "CWWKG0028A") ||
		strings.Contains(jsonLogs, "CWWKS4105I") {
		t.Errorf("Expected  to exclude messageId by default; but messageId \"%v\" is present", jsonLogs)
	}

	// Stop the container cleanly
	stopContainer(t, cli, id)
}

// TestLoggingConsoleSetToQmgr test sets LOG_FORMAT to BASIC and MQ_LOGGING_CONSOLE_FORMAT to
// json and check that log is in json format
func TestLoggingConsoleSetToQmgr(t *testing.T) {
	t.Parallel()
	cli := ce.NewContainerClient()
	containerConfig := ce.ContainerConfig{
		Env: []string{
			"LICENSE=accept",
			"MQ_QMGR_NAME=qm1",
			"MQ_ENABLE_EMBEDDED_WEB_SERVER=false",
			"MQ_LOGGING_CONSOLE_SOURCE=qmgr",
			"LOG_FORMAT=BASIC",
			"MQ_LOGGING_CONSOLE_FORMAT=json",
		},
	}
	id := runContainer(t, cli, &containerConfig)
	defer cleanContainer(t, cli, id)
	waitForReady(t, cli, id)

	jsonLogs, errJson := waitForMessageInLog(t, cli, id, "AMQ6206I")
	if errJson != nil {
		t.Errorf("%v", errJson)
	}

	isMessageFound := scanForExcludedEntries(jsonLogs)

	if isMessageFound == true {
		t.Errorf("Expected  to exclude messageId by default; but messageId \"%v\" is present", jsonLogs)
	}

	isValidJSON := checkLogForValidJSON(jsonLogs)

	if !isValidJSON {
		t.Fatalf("Expected all log lines to be valid JSON.  Logs: %v ", jsonLogs)
	}

	// Stop the container cleanly
	stopContainer(t, cli, id)
}

func TestWebLogsHeaderRotation(t *testing.T) {

	t.Parallel()
	cli := ce.NewContainerClient()
	containerConfig := ce.ContainerConfig{
		Env: []string{
			"LICENSE=accept",
			"MQ_QMGR_NAME=qm1",
			"MQ_ENABLE_EMBEDDED_WEB_SERVER=true",
			"MQ_LOGGING_CONSOLE_SOURCE=qmgr,web",
		},
	}
	id := runContainer(t, cli, &containerConfig)
	defer cleanContainer(t, cli, id)
	waitForReady(t, cli, id)

	consoleLogs, errJson := waitForMessageInLog(t, cli, id, "CWWKF0011I")
	if errJson != nil {
		t.Errorf("%v", errJson)
		//If there is error jump to last step and stop the container, else continue
	} else {
		//The below variable represents the first message in messages.log of web server, considered as the header message
		webLogheader := "product = WebSphere Application Server"

		if !strings.Contains(consoleLogs, webLogheader) {
			t.Errorf("Console log is without web server header message\n \"%v\"", consoleLogs)
			//If there is error jump to last step and stop the container, else continue
		} else {
			// Stop the container cleanly
			stopContainer(t, cli, id)
			startContainer(t, cli, id)
			waitForReady(t, cli, id)

			consoleLogs2, errJson := waitForMessageCountInLog(t, cli, id, "CWWKF0011I", 2)
			if errJson != nil {
				t.Errorf("%v", errJson)
				//If there is error jump to last step and stop the container, else continue
			} else {
				t.Logf("Total headers found is %v", strings.Count(consoleLogs2, webLogheader))
				if strings.Count(consoleLogs2, webLogheader) != 2 {
					t.Errorf("Console logs do not contain header message after restart \"%v\"", consoleLogs2)
				}
			}
		}
	}
	// Stop the container cleanly
	stopContainer(t, cli, id)
}

// Test queue manager with both personal and CA certificate having the same DN
func TestSameSubDNError(t *testing.T) {
	expectedOutput := "Error: The Subject DN of the Issuer Certificate and the Queue Manager are same"
	utilSubDNTest(t, "../tlssamesubdn", "true", expectedOutput, false)
}

// Test queue manager with both personal and CA certificate having the same DN
// but override the changed behavior via environment variable
func TestSameSubDNErrorOverride(t *testing.T) {
	expectedOutput := "Failed to relabel certificate for"
	utilSubDNTest(t, "../tlssamesubdn", "false", expectedOutput, false)
}

// Test queue manager with root CA certificate
func TestWithCASignedCerts(t *testing.T) {
	expectedOutput := "Creating queue manager MQQM"
	utilSubDNTest(t, "../tlsdifferentsubdn", "true", expectedOutput, true)
}

// Test queue manager with intermediate CA certificate
func TestWithIntermediateCASignedCerts(t *testing.T) {
	expectedOutput := "Creating queue manager MQQM"
	utilSubDNTest(t, "../tlsintermediateca", "true", expectedOutput, true)
}

// Scan the console output for required content.
func scanForText(output string, prefix string, findText string) (int, bool) {
	var count int
	var found bool
	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		s := scanner.Text()
		if strings.HasPrefix(s, prefix) {
			count++
		}
		if strings.Contains(s, findText) {
			found = true
		}
	}
	return count, found
}

// Utility function to test Certificate relabel issues.
func utilSubDNTest(t *testing.T, certPath string, overrideFlag string, expecteOutPut string, waitLong bool) {
	t.Parallel()

	cli := ce.NewContainerClient()

	containerConfig := ce.ContainerConfig{
		Env: []string{
			"LICENSE=accept",
			"MQ_QMGR_NAME=QM1",
			"MQ_ENABLE_CERT_VALIDATION=" + overrideFlag,
		},
		Image: imageName(),
	}
	hostConfig := ce.ContainerHostConfig{
		Binds: []string{
			coverageBind(t),
			tlsDirDN(t, false, certPath) + ":/etc/mqm/pki/keys/QM1",
		},
	}

	networkingConfig := ce.ContainerNetworkSettings{}
	ctrID, err := cli.ContainerCreate(&containerConfig, &hostConfig, &networkingConfig, t.Name())
	if err != nil {
		t.Fatal(err)
	}
	defer cleanContainer(t, cli, ctrID)
	startContainer(t, cli, ctrID)

	if waitLong {
		waitForReady(t, cli, ctrID)
		_, output := execContainer(t, cli, ctrID, "", []string{"bash", "-c", "echo 'DISPLAY QMGR SSLKEYR CERTLABL SSLFIPS' | runmqsc"})
		if !strings.Contains(output, "SSLKEYR(/run/runmqserver/tls/key)") {
			t.Errorf("Expected SSLKEYR to be '/run/runmqserver/tls/key' but it is not; got \"%v\"", output)
		}

		if !strings.Contains(output, "CERTLABL(QM1)") {
			t.Errorf("Expected CERTLABL to be 'default' but it is not; got \"%v\"", output)
		}
		_, output = execContainer(t, cli, ctrID, "", []string{"bash", "-c", "runmqakm -cert -list -type cms -db /run/runmqserver/tls/key.kdb -stashed"})
		if strings.EqualFold(t.Name(), "TestWithCASignedCerts") {
			// There should be one personal certificate and one trusted certificate.
			count, found := scanForText(output, "!", "CN=MQMFTQM,OU=ISL,O=IBM,L=BLR,ST=KA,C=IN")
			if count != 1 && !found {
				t.Errorf("Expected 1 trusted certificate with name containing CN=MQMFTQM. But found %v", output)
			}
			// One personal certificate that relabeld as QM1
			count, found = scanForText(output, "-", "QM1")
			if count != 1 && !found {
				t.Errorf("Expected 1 personal certificate with name containing QM1. But found %v", output)
			}
		} else if strings.EqualFold(t.Name(), "TestWithIntermediateCASignedCerts") {
			// There should be one personal certificate and two trusted certificates
			// an intermediate CA and the root CA.
			count, found := scanForText(output, "!", "ST=HANTS,C=GB")
			if count != 2 && !found {
				t.Errorf("Expected 2 trusted certificate with name containing 'ST=HANTS,C=GB'. But found %v", output)
			}
			// One personal certificate that is correctly relabeld as QM1
			count, found = scanForText(output, "-", "QM1")
			if count != 1 && !found {
				t.Errorf("Expected 1 personal certificate with name containing QM1. But found %v", output)
			}
		}
	} else {
		rc := waitForContainer(t, cli, ctrID, 30*time.Second)
		// Expect return code 1 if container failed to create.
		if rc == 1 {
			// Get container logs and search for specific message.
			logs := inspectLogs(t, cli, ctrID)
			if !strings.Contains(logs, expecteOutPut) {
				t.Errorf("Container creating failed because of invalid certifates")
			}
		} else {
			// Some other error occurred
			t.Errorf("Some other error occurred %v", rc)
		}
	}
}

// Attempt to run container with read-only root filesystem. Container
// should fail with a "read-only file system" error message.
func TestReadOnlyRootFilesystem(t *testing.T) {
	t.Parallel()

	cli := ce.NewContainerClient()
	containerConfig := ce.ContainerConfig{
		Image: imageName(),
		Env:   []string{"LICENSE=accept", "MQ_QMGR_NAME=QM1"},
	}
	hostConfig := ce.ContainerHostConfig{
		Binds: []string{
			coverageBind(t),
		},
		ReadOnlyRootfs: true,
	}
	networkingConfig := ce.ContainerNetworkSettings{}
	ctrID, err := cli.ContainerCreate(&containerConfig, &hostConfig, &networkingConfig, t.Name())
	if err != nil {
		t.Fatal(err)
	}
	startContainer(t, cli, ctrID)
	defer cleanContainer(t, cli, ctrID)

	rc := waitForContainer(t, cli, ctrID, 30*time.Second)
	if rc != 1 {
		t.Errorf("Expected rc=1, got rc=%v", rc)
	}

	messageToSearch := "read-only file system"
	l := inspectLogs(t, cli, ctrID)
	if !strings.Contains(strings.ToLower(l), messageToSearch) {
		t.Fatalf("Expected 'read-only file system' in the logs but was not found. The output was: %s", l)
	}
}

// Verify symlinks have been correctly created
func TestRORFSVerifySymLinks(t *testing.T) {
	t.Parallel()

	cli := ce.NewContainerClient()

	const tlsPassPhrase string = "passw0rd"
	qm := "QM1"
	appPassword := "differentPassw0rd"
	containerConfig := ce.ContainerConfig{
		Env: []string{
			"LICENSE=accept",
			"MQ_QMGR_NAME=" + qm,
			"MQ_APP_PASSWORD=" + appPassword,
			"DEBUG=1",
			"WLP_LOGGING_MESSAGE_FORMAT=JSON",
			"MQ_ENABLE_EMBEDDED_WEB_SERVER_LOG=true",
			"MQ_ENABLE_FIPS=true",
		},
		Image: imageName(),
	}

	ephData := createVolume(t, cli, "ephData"+t.Name())
	defer removeVolume(t, cli, ephData)
	ephRun := createVolume(t, cli, "ephRun"+t.Name())
	defer removeVolume(t, cli, ephRun)
	ephTmp := createVolume(t, cli, "ephTmp"+t.Name())
	defer removeVolume(t, cli, ephTmp)
	hostConfig := ce.ContainerHostConfig{
		Binds: []string{
			coverageBind(t),
			ephRun + ":/run",
			ephTmp + ":/tmp",
			ephData + ":/mnt/mqm",
			tlsDirDN(t, false, "../tls") + ":/etc/mqm/pki/keys/default",
			tlsDirDN(t, false, "../tls") + ":/etc/mqm/pki/trust/default",
		},
		ReadOnlyRootfs: true,
	}

	// Assign a random port for the web server on the host
	var binding ce.PortBinding
	ports := []int{9443}
	for _, p := range ports {
		port := fmt.Sprintf("%v/tcp", p)
		binding = ce.PortBinding{
			ContainerPort: port,
			HostIP:        "0.0.0.0",
		}
		hostConfig.PortBindings = append(hostConfig.PortBindings, binding)
	}
	networkingConfig := ce.ContainerNetworkSettings{}
	ID, err := cli.ContainerCreate(&containerConfig, &hostConfig, &networkingConfig, t.Name())
	if err != nil {
		t.Fatal(err)
	}
	defer cleanContainer(t, cli, ID)

	startContainer(t, cli, ID)
	waitForReady(t, cli, ID)

	// Check both qmgr keystore and web keystores are created in /run/runmqserver directory
	// Check there are valid symlinks created.
	var symLinks = []struct {
		origin      string
		symLinkName string
	}{
		{
			origin:      "/etc/mqm/web/installations/Installation1/servers/mqweb/mqwebexternal.xml",
			symLinkName: "-> /run/mqwebexternal.xml",
		},
		{
			origin:      "/etc/mqm/web/installations/Installation1/servers/mqweb/tls.xml",
			symLinkName: "-> /run/tls.xml",
		},
		{
			origin:      "/etc/mqm/web/installations/Installation1/servers/mqweb/configDropins/defaults/jvm.options",
			symLinkName: "-> /run/jvm.options",
		},
		{
			origin:      "/etc/mqm/15-tls.mqsc",
			symLinkName: "-> /run/15-tls.mqsc",
		},
		{
			origin:      "/etc/mqm/native-ha.ini",
			symLinkName: "-> /run/native-ha.ini",
		},
		{
			origin:      "/run/runmqserver",
			symLinkName: "-> /run/scratch/runmqserver",
		},
	}

	for _, symLink := range symLinks {
		_, out := execContainer(t, cli, ID, "", []string{"ls", "-l", symLink.origin})
		if !strings.Contains(out, symLink.symLinkName) {
			t.Errorf("Expected symlink =%v, but did not get. Got %v", symLink.origin, out)
		}
	}

	// Verify keystore and trust stores are created as expected
	var fileNamesAndPermissions = []struct {
		fileName    string
		permissions string
	}{
		{
			fileName:    "/run/runmqserver/tls/default.p12",
			permissions: "-rw-r--r--",
		},
		{
			fileName:    "/run/runmqserver/tls/key.crl",
			permissions: "-rw-------",
		},
		{
			fileName:    "/run/runmqserver/tls/key.kdb",
			permissions: "-rw-------",
		},
		{
			fileName:    "/run/runmqserver/tls/key.rdb",
			permissions: "-rw-------",
		},
		{
			fileName:    "/run/runmqserver/tls/key.sth",
			permissions: "-rw-------",
		},
		{
			fileName:    "/run/runmqserver/tls/trust.p12",
			permissions: "-rw-------",
		},
		{
			fileName:    "/run/runmqserver/tls/trust.sth",
			permissions: "-rw-------",
		},
	}

	for _, filePerm := range fileNamesAndPermissions {
		_, out := execContainer(t, cli, ID, "", []string{"ls", "-lR", filePerm.fileName})
		if !strings.Contains(out, filePerm.fileName) || !strings.Contains(out, filePerm.permissions) {
			t.Errorf("Expected file=%v or permisions =%v was not found, Got %v", filePerm.fileName, filePerm.permissions, out)
		}
	}

	// Stop the container cleanly
	stopContainer(t, cli, ID)
}
