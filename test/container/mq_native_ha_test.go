/*
© Copyright IBM Corporation 2021, 2026

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
	"strings"
	"testing"
	"time"

	ce "github.com/ibm-messaging/mq-container/test/container/containerengine"
)

// TestNativeHABasic creates 3 containers in a Native HA queue manager configuration
// and ensures the queue manger and replicas start as expected
func TestNativeHABasic(t *testing.T) {
	cli := ce.NewContainerClient()

	version, err := cli.GetMQVersion(imageName())
	if err != nil {
		t.Fatal(err)
	}
	if version < "9.2.2.0" {
		t.Skipf("Skipping %s as test requires at least MQ 9.2.2.0, but image is version %s", t.Name(), version)
	}

	containerNames := [3]string{"QM1_1", "QM1_2", "QM1_3"}
	qmReplicaIDs := [3]string{}
	qmVolumes := []string{}
	//Each native HA qmgr instance is exposed on subsequent ports on the host starting with basePort
	//If the qmgr exposes more than one port (tests do not do this currently) then they are offset by +50
	basePort := 14551
	for i := 0; i <= 2; i++ {
		nhaPort := basePort + i
		vol := createVolume(t, cli, containerNames[i])
		cleanupVolume(t, cli, vol)
		qmVolumes = append(qmVolumes, vol)
		containerConfig := getNativeHAContainerConfig(containerNames[i], containerNames, basePort)
		hostConfig := getHostConfig(t, 1, "", "", vol, "", "", false)
		hostConfig = populateNativeHAPortBindings([]int{9414}, nhaPort, hostConfig)
		networkingConfig := getNativeHANetworkConfig("host")
		ctr := runContainerWithAllConfig(t, cli, &containerConfig, &hostConfig, &networkingConfig, containerNames[i])
		cleanupAfterTest(t, cli, ctr, false)
		qmReplicaIDs[i] = ctr
	}

	waitForReadyHA(t, cli, qmReplicaIDs)

	_, err = getActiveReplicaInstances(t, cli, qmReplicaIDs)
	if err != nil {
		t.Fatal(err)
	}

}

// TestNativeHAFailover creates 3 containers in a Native HA queue manager configuration,
// stops the active queue manager, checks a replica becomes active, and ensures the stopped
// queue manager comes back as a replica
func TestNativeHAFailover(t *testing.T) {

	cli := ce.NewContainerClient()

	version, err := cli.GetMQVersion(imageName())
	if err != nil {
		t.Fatal(err)
	}
	if version < "9.2.2.0" {
		t.Skipf("Skipping %s as test requires at least MQ 9.2.2.0, but image is version %s", t.Name(), version)
	}

	containerNames := [3]string{"QM1_1", "QM1_2", "QM1_3"}
	qmReplicaIDs := [3]string{}
	qmVolumes := []string{}
	//Each native HA qmgr instance is exposed on subsequent ports on the host starting with basePort
	//If the qmgr exposes more than one port (tests do not do this currently) then they are offset by +50
	basePort := 14551
	for i := 0; i <= 2; i++ {
		nhaPort := basePort + i
		vol := createVolume(t, cli, containerNames[i])
		cleanupVolume(t, cli, vol)
		qmVolumes = append(qmVolumes, vol)
		containerConfig := getNativeHAContainerConfig(containerNames[i], containerNames, basePort)
		hostConfig := getHostConfig(t, 1, "", "", vol, "", "", false)
		hostConfig = populateNativeHAPortBindings([]int{9414}, nhaPort, hostConfig)
		networkingConfig := getNativeHANetworkConfig("host")
		ctr := runContainerWithAllConfig(t, cli, &containerConfig, &hostConfig, &networkingConfig, containerNames[i])
		cleanupAfterTest(t, cli, ctr, false)
		qmReplicaIDs[i] = ctr
	}

	waitForReadyHA(t, cli, qmReplicaIDs)

	haStatus, err := getActiveReplicaInstances(t, cli, qmReplicaIDs)
	if err != nil {
		t.Fatal(err)
	}

	stopContainer(t, cli, haStatus.Active)
	waitForFailoverHA(t, cli, haStatus.Replica)
	startContainer(t, cli, haStatus.Active)
	waitForReady(t, cli, haStatus.Active)

	_, err = getActiveReplicaInstances(t, cli, qmReplicaIDs)
	if err != nil {
		t.Fatal(err)
	}

}

// TestNativeHASecure creates 3 containers in a Native HA queue manager configuration
// with HA TLS enabled, and ensures the queue manger and replicas start as expected
func TestNativeHASecure(t *testing.T) {
	cli := ce.NewContainerClient()

	version, err := cli.GetMQVersion(imageName())
	if err != nil {
		t.Fatal(err)
	}
	if version < "9.2.2.0" {
		t.Skipf("Skipping %s as test requires at least MQ 9.2.2.0, but image is version %s", t.Name(), version)
	}
	if isARM(t) {
		t.Skip("Skipping as an issue has been identified for the arm64 MQ image")
	}

	containerNames := [3]string{"QM1_1", "QM1_2", "QM1_3"}
	qmReplicaIDs := [3]string{}
	//Each native HA qmgr instance is exposed on subsequent ports on the host starting with basePort
	//If the qmgr exposes more than one port (tests do not do this currently) then they are offset by +50
	basePort := 14551
	for i := 0; i <= 2; i++ {
		nhaPort := basePort + i
		containerConfig := getNativeHAContainerConfig(containerNames[i], containerNames, defaultHAPort)
		containerConfig.Env = append(containerConfig.Env, "MQ_NATIVE_HA_TLS=true")
		hostConfig := getNativeHASecureHostConfig(t)
		hostConfig = populateNativeHAPortBindings([]int{9414}, nhaPort, hostConfig)
		networkingConfig := getNativeHANetworkConfig("host")
		ctr := runContainerWithAllConfig(t, cli, &containerConfig, &hostConfig, &networkingConfig, containerNames[i])
		cleanupAfterTest(t, cli, ctr, false)
		qmReplicaIDs[i] = ctr
	}

	waitForReadyHA(t, cli, qmReplicaIDs)

	_, err = getActiveReplicaInstances(t, cli, qmReplicaIDs)
	if err != nil {
		t.Fatal(err)
	}

}

// TestNativeHASecure creates 3 containers in a Native HA queue manager configuration
// with HA TLS enabled, overrides the default CipherSpec, and ensures the queue manger
// and replicas start as expected
func TestNativeHASecureCipherSpec(t *testing.T) {
	cli := ce.NewContainerClient()

	version, err := cli.GetMQVersion(imageName())
	if err != nil {
		t.Fatal(err)
	}
	if version < "9.2.2.0" {
		t.Skipf("Skipping %s as test requires at least MQ 9.2.2.0, but image is version %s", t.Name(), version)
	}

	containerNames := [3]string{"QM1_1", "QM1_2", "QM1_3"}
	qmReplicaIDs := [3]string{}
	//Each native HA qmgr instance is exposed on subsequent ports on the host starting with basePort
	//If the qmgr exposes more than one port (tests do not do this currently) then they are offset by +50
	basePort := 14551
	for i := 0; i <= 2; i++ {
		nhaPort := basePort + i
		containerConfig := getNativeHAContainerConfig(containerNames[i], containerNames, defaultHAPort)
		containerConfig.Env = append(containerConfig.Env, "MQ_NATIVE_HA_TLS=true", "MQ_NATIVE_HA_CIPHERSPEC=TLS_AES_256_GCM_SHA384")
		hostConfig := getNativeHASecureHostConfig(t)
		hostConfig = populateNativeHAPortBindings([]int{9414}, nhaPort, hostConfig)
		networkingConfig := getNativeHANetworkConfig("host")
		ctr := runContainerWithAllConfig(t, cli, &containerConfig, &hostConfig, &networkingConfig, containerNames[i])
		cleanupAfterTest(t, cli, ctr, false)
		qmReplicaIDs[i] = ctr
	}

	waitForReadyHA(t, cli, qmReplicaIDs)

	_, err = getActiveReplicaInstances(t, cli, qmReplicaIDs)
	if err != nil {
		t.Fatal(err)
	}

}

// TestNativeHASecure creates 3 containers in a Native HA queue manager configuration
// with HA TLS FIPS enabled, overrides the default CipherSpec, and ensures the queue manger
// and replicas start as expected. This test uses FIPS compliant cipher.
func TestNativeHASecureCipherSpecFIPS(t *testing.T) {
	cli := ce.NewContainerClient()
	skipIfFIPSCryptoUnavailable(t, cli)

	version, err := cli.GetMQVersion(imageName())
	if err != nil {
		t.Fatal(err)
	}
	if version < "9.2.2.0" {
		t.Skipf("Skipping %s as test requires at least MQ 9.2.2.0, but image is version %s", t.Name(), version)
	}

	containerNames := [3]string{"QM1_1", "QM1_2", "QM1_3"}
	qmReplicaIDs := [3]string{}
	//Each native HA qmgr instance is exposed on subsequent ports on the host starting with basePort
	//If the qmgr exposes more than one port (tests do not do this currently) then they are offset by +50
	basePort := 14551
	for i := 0; i <= 2; i++ {
		nhaPort := basePort + i
		containerConfig := getNativeHAContainerConfig(containerNames[i], containerNames, defaultHAPort)
		// MQ_NATIVE_HA_CIPHERSPEC is set a FIPS compliant cipherspec.
		containerConfig.Env = append(containerConfig.Env, "MQ_NATIVE_HA_TLS=true", "MQ_NATIVE_HA_CIPHERSPEC=ANY_TLS12_OR_HIGHER", "MQ_ENABLE_FIPS=true")
		hostConfig := getNativeHASecureHostConfig(t)
		hostConfig = populateNativeHAPortBindings([]int{9414}, nhaPort, hostConfig)
		networkingConfig := getNativeHANetworkConfig("host")
		ctr := runContainerWithAllConfig(t, cli, &containerConfig, &hostConfig, &networkingConfig, containerNames[i])
		cleanupAfterTest(t, cli, ctr, false)
		qmReplicaIDs[i] = ctr
	}

	waitForReadyHA(t, cli, qmReplicaIDs)
	// Display the contents of qm.ini
	_, qmini := execContainer(t, cli, qmReplicaIDs[0], "", []string{"cat", "/var/mqm/qmgrs/QM1/qm.ini"})
	if !strings.Contains(qmini, "SSLFipsRequired=Yes") {
		t.Errorf("Expected SSLFipsRequired=Yes but it is not; got \"%v\"", qmini)
	}

	_, err = getActiveReplicaInstances(t, cli, qmReplicaIDs)
	if err != nil {
		t.Fatal(err)
	}
}

// TestNativeHASecure creates 3 containers in a Native HA queue manager configuration
// with HA TLS FIPS enabled with non-FIPS cipher, overrides the default CipherSpec, and
// ensures the queue manger and replicas don't start as expected
func TestNativeHASecureCipherSpecNonFIPSCipher(t *testing.T) {
	cli := ce.NewContainerClient()
	skipIfFIPSCryptoUnavailable(t, cli)

	version, err := cli.GetMQVersion(imageName())
	if err != nil {
		t.Fatal(err)
	}
	if version < "9.2.2.0" {
		t.Skipf("Skipping %s as test requires at least MQ 9.2.2.0, but image is version %s", t.Name(), version)
	}

	containerNames := [3]string{"QM1_1", "QM1_2", "QM1_3"}
	qmReplicaIDs := [3]string{}
	//Each native HA qmgr instance is exposed on subsequent ports on the host starting with basePort
	//If the qmgr exposes more than one port (tests do not do this currently) then they are offset by +50
	basePort := 14551
	for i := 0; i <= 2; i++ {
		nhaPort := basePort + i
		containerConfig := getNativeHAContainerConfig(containerNames[i], containerNames, defaultHAPort)
		// MQ_NATIVE_HA_CIPHERSPEC is set a FIPS non-compliant cipherspec - SSL_ECDHE_ECDSA_WITH_RC4_128_SHA
		containerConfig.Env = append(containerConfig.Env, "MQ_NATIVE_HA_TLS=true", "MQ_NATIVE_HA_CIPHERSPEC=SSL_ECDHE_ECDSA_WITH_RC4_128_SHA", "MQ_ENABLE_FIPS=true")
		hostConfig := getNativeHASecureHostConfig(t)
		hostConfig = populateNativeHAPortBindings([]int{9414}, nhaPort, hostConfig)
		networkingConfig := getNativeHANetworkConfig("host")
		ctr := runContainerWithAllConfig(t, cli, &containerConfig, &hostConfig, &networkingConfig, containerNames[i])
		cleanupAfterTest(t, cli, ctr, false)
		// We expect container to fail in this case because the cipher is non-FIPS and we have asked for FIPS compliance
		// by setting MQ_ENABLE_FIPS=true
		qmReplicaIDs[i] = ctr
	}
	for i := 0; i <= 2; i++ {
		waitForTerminationMessage(t, cli, qmReplicaIDs[i], "/opt/mqm/bin/strmqm: exit status 23", 60*time.Second)
	}
}

// TestNativeHAFailover creates 3 containers in a Native HA queue manager configuration,
// stops the active queue manager, checks a replica becomes active, and ensures the stopped
// queue manager comes back as a replica
func TestNativeHAFailoverWithRoRFs(t *testing.T) {

	cli := ce.NewContainerClient()

	version, err := cli.GetMQVersion(imageName())
	if err != nil {
		t.Fatal(err)
	}
	if version < "9.2.2.0" {
		t.Skipf("Skipping %s as test requires at least MQ 9.2.2.0, but image is version %s", t.Name(), version)
	}

	containerNames := [3]string{"QM1_1", "QM1_2", "QM1_3"}
	qmReplicaIDs := [3]string{}
	qmVolumes := []string{}
	//Each native HA qmgr instance is exposed on subsequent ports on the host starting with basePort
	//If the qmgr exposes more than one port (tests do not do this currently) then they are offset by +50
	basePort := 14551
	for i := 0; i <= 2; i++ {
		nhaPort := basePort + i
		vol := createVolume(t, cli, containerNames[i])
		cleanupVolume(t, cli, vol)
		volRun := createVolume(t, cli, "ephRun"+containerNames[i])
		cleanupVolume(t, cli, volRun)
		volTmp := createVolume(t, cli, "ephTmp"+containerNames[i])
		cleanupVolume(t, cli, volTmp)

		qmVolumes = append(qmVolumes, vol)
		qmVolumes = append(qmVolumes, volRun)
		qmVolumes = append(qmVolumes, volTmp)

		containerConfig := getNativeHAContainerConfig(containerNames[i], containerNames, basePort)
		hostConfig := getHostConfig(t, 1, "", "", vol, volRun, volTmp, true)
		hostConfig = populateNativeHAPortBindings([]int{9414}, nhaPort, hostConfig)
		networkingConfig := getNativeHANetworkConfig("host")
		ctr := runContainerWithAllConfig(t, cli, &containerConfig, &hostConfig, &networkingConfig, containerNames[i])
		cleanupAfterTest(t, cli, ctr, false)
		qmReplicaIDs[i] = ctr
	}

	waitForReadyHA(t, cli, qmReplicaIDs)

	haStatus, err := getActiveReplicaInstances(t, cli, qmReplicaIDs)
	if err != nil {
		t.Fatal(err)
	}

	stopContainer(t, cli, haStatus.Active)
	waitForFailoverHA(t, cli, haStatus.Replica)
	startContainer(t, cli, haStatus.Active)
	waitForReady(t, cli, haStatus.Active)

	_, err = getActiveReplicaInstances(t, cli, qmReplicaIDs)
	if err != nil {
		t.Fatal(err)
	}
}

// TestNativeHaProbeLoggingGoldenPath tests probe logging in Native HA configuration.
// Verifies first successful probe is logged on all three instances (active and replicas), and SIGTERM summary is generated on the active instance.
func TestNativeHaProbeLoggingGoldenPath(t *testing.T) {
	cli := ce.NewContainerClient()

	containerNames := [3]string{"QM1_1", "QM1_2", "QM1_3"}
	qmReplicaIds := [3]string{}
	qmVolumes := []string{}
	// Each native HA qmgr instance is exposed on subsequent ports on the host starting with basePort
	// If the qmgr exposes more than one port (tests do not do this currently) then they are offset by +50
	basePort := 14551
	for i := 0; i <= 2; i++ {
		nhaPort := basePort + i
		vol := createVolume(t, cli, containerNames[i])
		cleanupVolume(t, cli, vol)
		qmVolumes = append(qmVolumes, vol)
		containerConfig := getNativeHAContainerConfig(containerNames[i], containerNames, basePort)
		hostConfig := getHostConfig(t, 1, "", "", vol, "", "", false)
		hostConfig = populateNativeHAPortBindings([]int{9414}, nhaPort, hostConfig)
		networkConfig := getNativeHANetworkConfig("host")
		ctr := runContainerWithAllConfig(t, cli, &containerConfig, &hostConfig, &networkConfig, containerNames[i])
		cleanupAfterTest(t, cli, ctr, false)
		qmReplicaIds[i] = ctr
	}

	waitForReadyHA(t, cli, qmReplicaIds)

	haStatus, err := getActiveReplicaInstances(t, cli, qmReplicaIds)
	if err != nil {
		t.Fatal(err)
	}

	// Execute chkmqstarted and chkmqhealthy on all three instances to trigger first pass logging
	for _, id := range qmReplicaIds {
		// Execute the chkmqstarted and chkmqhealthy multiple times
		for i := 1; i <= 3; i++ {

			// Only check startup on the active instance here. waitForReadyHA() returns once a single Native HA instance has successfully started, but replicas may
			// still be progressing through recovery, log replay, or the in-sync timeout path used by chkmqstarted(), which can legitimately return non-zero.
			if id == haStatus.Active {
				rc, _ := execContainer(t, cli, id, "", []string{"chkmqstarted"})
				if rc != 0 {
					t.Errorf("Expected startup probe to pass with rc=0, got rc=%d", rc)
				}
			}

			rc, _ := execContainer(t, cli, id, "", []string{"chkmqhealthy"})
			if rc != 0 {
				t.Errorf("Expected liveness probe to pass with rc=0, got rc=%d", rc)
			}
		}
	}

	// Verify first successful startup probe and liveness probe were logged once only.
	// Additional identical successful probes should be suppressed by deduplication.
	for _, id := range qmReplicaIds {
		containerLogs := inspectLogs(t, cli, id)

		// Only check startup on the active instance here. waitForReadyHA() returns once a single Native HA instance has successfully started, but replicas may
		// still be progressing through recovery, log replay, or the in-sync timeout path used by chkmqstarted(), which can legitimately return non-zero.
		if id == haStatus.Active {
			startupPassedCount := strings.Count(containerLogs, "Startup Probe Passed")
			if startupPassedCount != 1 {
				t.Errorf("Expected exactly one startup probe pass log due to deduplication, got %d", startupPassedCount)
			}
		}

		livenessPassedCount := strings.Count(containerLogs, "Liveness Probe Passed")
		if livenessPassedCount != 1 {
			t.Errorf("Expected exactly one liveness probe pass log due to deduplication, got %d", livenessPassedCount)
		}
	}

	// Kill the active replica by sending SIGTERM
	killContainer(t, cli, haStatus.Active, "SIGTERM")

	// Verify the active replica has the probe summary logged
	containerLogs := inspectLogs(t, cli, haStatus.Active)

	// Since the liveness probe has been executed, startup probe summary should not be logged
	if strings.Contains(containerLogs, "----- Start Startup Probe Summary -----") {
		t.Errorf("Startup probe summary logged at SIGTERM, even when liveness probe has been executed")
	}

	if !strings.Contains(containerLogs, "----- Start Liveness Probe Summary -----") {
		t.Errorf("Expected liveness probe summary at SIGTERM")
	}
}

// TestNativeHALivenessProbeLoggingFailureRecovery tests liveness probe logging deduplication on the active Native HA instance.
// Verifies pass-fail-fail-pass pattern logging and SIGTERM summary generation.
func TestNativeHaLivenessProbeLoggingFailureRecovery(t *testing.T) {
	cli := ce.NewContainerClient()

	containerNames := [3]string{"QM1_1", "QM1_2", "QM1_3"}
	qmReplicaIds := [3]string{}
	qmVolumes := []string{}
	// Each native HA qmgr instance is exposed on subsequent ports on the host starting with basePort
	// If the qmgr exposes more than one port (tests do not do this currently) then they are offset by +50
	basePort := 14551
	for i := 0; i <= 2; i++ {
		nhaPort := basePort + i
		vol := createVolume(t, cli, containerNames[i])
		cleanupVolume(t, cli, vol)
		qmVolumes = append(qmVolumes, vol)
		containerConfig := getNativeHAContainerConfig(containerNames[i], containerNames, basePort)
		hostConfig := getHostConfig(t, 1, "", "", vol, "", "", false)
		hostConfig = populateNativeHAPortBindings([]int{9414}, nhaPort, hostConfig)
		networkConfig := getNativeHANetworkConfig("host")
		ctr := runContainerWithAllConfig(t, cli, &containerConfig, &hostConfig, &networkConfig, containerNames[i])
		cleanupAfterTest(t, cli, ctr, false)
		qmReplicaIds[i] = ctr
	}

	waitForReadyHA(t, cli, qmReplicaIds)

	haStatus, err := getActiveReplicaInstances(t, cli, qmReplicaIds)
	if err != nil {
		t.Fatal(err)
	}
	qmActiveReplicaId := haStatus.Active

	// First success
	rc, _ := execContainer(t, cli, qmActiveReplicaId, "", []string{"chkmqhealthy"})
	if rc != 0 {
		t.Errorf("Expected liveness probe to pass with rc=0, got rc=%d", rc)
	}

	// Stop the QueueManager
	execContainer(t, cli, qmActiveReplicaId, "", []string{"endmqm", "-i", "QM1"})
	time.Sleep(2 * time.Second)

	// Execute the chkmqhealthy command, it will now fail
	rc, _ = execContainer(t, cli, qmActiveReplicaId, "", []string{"chkmqhealthy"})
	if rc == 0 {
		t.Errorf("Expected liveness probe to fail")
	}

	rc, _ = execContainer(t, cli, qmActiveReplicaId, "", []string{"chkmqhealthy"})
	if rc == 0 {
		t.Errorf("Expected liveness probe to fail")
	}

	// Start the QueueManager
	execContainer(t, cli, qmActiveReplicaId, "", []string{"strmqm", "QM1"})
	waitForReadyHA(t, cli, qmReplicaIds)

	// Execute the chkmqhealthy command
	rc, _ = execContainer(t, cli, qmActiveReplicaId, "", []string{"chkmqhealthy"})
	if rc != 0 {
		t.Errorf("Expected liveness probe to pass with rc=0, got rc=%d", rc)
	}

	containerLogs := inspectLogs(t, cli, qmActiveReplicaId)

	// Verify: 1st pass, 2 fails, recovery pass all logged
	passedRuntimeLogCount := strings.Count(containerLogs, "Liveness Probe Passed")
	failedRuntimeLogCount := strings.Count(containerLogs, "Liveness Probe Failed")

	if passedRuntimeLogCount != 2 {
		t.Errorf("Expected 2 liveness probe pass logs (first + recovery), got %d", passedRuntimeLogCount)
	}

	if failedRuntimeLogCount != 2 {
		t.Errorf("Expected 2 liveness probe failure logs, got %d", passedRuntimeLogCount)
	}
}

// TestNativeHaStartupProbeLoggingSummaryOnSigterm tests that when the startup probe
// has not passed, SIGTERM logs the startup probe summary with the correct attempt count.
func TestNativeHaStartupProbeLoggingOnSigterm(t *testing.T) {
	cli := ce.NewContainerClient()

	containerNames := [3]string{"QM1_1", "QM1_2", "QM1_3"}
	qmReplicaIds := [3]string{}
	qmVolumes := []string{}
	// Each native HA qmgr instance is exposed on subsequent ports on the host starting with basePort
	// If the qmgr exposes more than one port (tests do not do this currently) then they are offset by +50
	basePort := 14551
	for i := 0; i <= 2; i++ {
		nhaPort := basePort + i
		vol := createVolume(t, cli, containerNames[i])
		cleanupVolume(t, cli, vol)
		qmVolumes = append(qmVolumes, vol)
		containerConfig := getNativeHAContainerConfig(containerNames[i], containerNames, basePort)
		hostConfig := getHostConfig(t, 1, "", "", vol, "", "", false)
		hostConfig = populateNativeHAPortBindings([]int{9414}, nhaPort, hostConfig)
		networkConfig := getNativeHANetworkConfig("host")
		ctr := runContainerWithAllConfig(t, cli, &containerConfig, &hostConfig, &networkConfig, containerNames[i])
		cleanupAfterTest(t, cli, ctr, false)
		qmReplicaIds[i] = ctr
	}

	waitForReadyHA(t, cli, qmReplicaIds)

	haStatus, err := getActiveReplicaInstances(t, cli, qmReplicaIds)
	if err != nil {
		t.Fatal(err)
	}
	qmActiveReplicaId := haStatus.Active

	// Stop the QueueManager
	execContainer(t, cli, qmActiveReplicaId, "", []string{"endmqm", "-i", "QM1"})
	time.Sleep(2 * time.Second)

	// Execute the chkmqstarted command multiple times
	for i := 1; i <= 5; i++ {
		rc, _ := execContainer(t, cli, qmActiveReplicaId, "", []string{"chkmqstarted"})
		if rc == 0 {
			t.Errorf("Expected startup probe to fail on attempt %d", i)
		}
	}

	// Kill the active replica by sending SIGTERM
	killContainer(t, cli, qmActiveReplicaId, "SIGTERM")

	containerLogs := inspectLogs(t, cli, qmActiveReplicaId)

	if !strings.Contains(containerLogs, "----- Start Startup Probe Summary -----") {
		t.Errorf("Expected startup probe summary at SIGTERM")
	}

	if strings.Contains(containerLogs, "----- Start Liveness Probe Summary -----") {
		t.Errorf("Did not expect liveness probe summary at SIGTERM when liveness probe has not been executed")
	}

	if !strings.Contains(containerLogs, "Last State: Failed") {
		t.Errorf("Expected startup probe summary to show failed last run, logs were: %s", containerLogs)
	}

}
