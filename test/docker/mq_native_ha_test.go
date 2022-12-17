/*
© Copyright IBM Corporation 2021, 2022

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
	"github.com/docker/docker/client"
	"strings"
	"testing"
)

// TestNativeHABasic creates 3 containers in a Native HA queue manager configuration
// and ensures the queue manger and replicas start as expected
func TestNativeHABasic(t *testing.T) {
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		t.Fatal(err)
	}

	version, err := getMQVersion(t, cli)
	if err != nil {
		t.Fatal(err)
	}
	if version < "9.2.2.0" {
		t.Skipf("Skipping %s as test requires at least MQ 9.2.2.0, but image is version %s", t.Name(), version)
	}

	containerNames := [3]string{"QM1_1", "QM1_2", "QM1_3"}
	qmReplicaIDs := [3]string{}
	qmVolumes := []string{}
	qmNetwork, err := createBridgeNetwork(cli, t)
	if err != nil {
		t.Fatal(err)
	}
	defer removeBridgeNetwork(cli, qmNetwork.ID)

	for i := 0; i <= 2; i++ {
		vol := createVolume(t, cli, containerNames[i])
		defer removeVolume(t, cli, vol.Name)
		qmVolumes = append(qmVolumes, vol.Name)

		containerConfig := getNativeHAContainerConfig(containerNames[i], containerNames, defaultHAPort)
		hostConfig := getHostConfig(t, 1, "", "", vol.Name)
		networkingConfig := getNativeHANetworkConfig(qmNetwork.ID)

		ctr := runContainerWithAllConfig(t, cli, &containerConfig, &hostConfig, &networkingConfig, containerNames[i])
		defer cleanContainer(t, cli, ctr)
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

	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		t.Fatal(err)
	}

	version, err := getMQVersion(t, cli)
	if err != nil {
		t.Fatal(err)
	}
	if version < "9.2.2.0" {
		t.Skipf("Skipping %s as test requires at least MQ 9.2.2.0, but image is version %s", t.Name(), version)
	}

	containerNames := [3]string{"QM1_1", "QM1_2", "QM1_3"}
	qmReplicaIDs := [3]string{}
	qmVolumes := []string{}
	qmNetwork, err := createBridgeNetwork(cli, t)
	if err != nil {
		t.Fatal(err)
	}
	defer removeBridgeNetwork(cli, qmNetwork.ID)

	for i := 0; i <= 2; i++ {
		vol := createVolume(t, cli, containerNames[i])
		defer removeVolume(t, cli, vol.Name)
		qmVolumes = append(qmVolumes, vol.Name)

		containerConfig := getNativeHAContainerConfig(containerNames[i], containerNames, defaultHAPort)
		hostConfig := getHostConfig(t, 1, "", "", vol.Name)
		networkingConfig := getNativeHANetworkConfig(qmNetwork.ID)

		ctr := runContainerWithAllConfig(t, cli, &containerConfig, &hostConfig, &networkingConfig, containerNames[i])
		defer cleanContainer(t, cli, ctr)
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
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		t.Fatal(err)
	}

	version, err := getMQVersion(t, cli)
	if err != nil {
		t.Fatal(err)
	}
	if version < "9.2.2.0" {
		t.Skipf("Skipping %s as test requires at least MQ 9.2.2.0, but image is version %s", t.Name(), version)
	}

	containerNames := [3]string{"QM1_1", "QM1_2", "QM1_3"}
	qmReplicaIDs := [3]string{}
	qmNetwork, err := createBridgeNetwork(cli, t)
	if err != nil {
		t.Fatal(err)
	}
	defer removeBridgeNetwork(cli, qmNetwork.ID)

	for i := 0; i <= 2; i++ {
		containerConfig := getNativeHAContainerConfig(containerNames[i], containerNames, defaultHAPort)
		containerConfig.Env = append(containerConfig.Env, "MQ_NATIVE_HA_TLS=true")
		hostConfig := getNativeHASecureHostConfig(t)
		networkingConfig := getNativeHANetworkConfig(qmNetwork.ID)

		ctr := runContainerWithAllConfig(t, cli, &containerConfig, &hostConfig, &networkingConfig, containerNames[i])
		defer cleanContainer(t, cli, ctr)
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
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		t.Fatal(err)
	}

	version, err := getMQVersion(t, cli)
	if err != nil {
		t.Fatal(err)
	}
	if version < "9.2.2.0" {
		t.Skipf("Skipping %s as test requires at least MQ 9.2.2.0, but image is version %s", t.Name(), version)
	}

	containerNames := [3]string{"QM1_1", "QM1_2", "QM1_3"}
	qmReplicaIDs := [3]string{}
	qmNetwork, err := createBridgeNetwork(cli, t)
	if err != nil {
		t.Fatal(err)
	}
	defer removeBridgeNetwork(cli, qmNetwork.ID)

	for i := 0; i <= 2; i++ {
		containerConfig := getNativeHAContainerConfig(containerNames[i], containerNames, defaultHAPort)
		containerConfig.Env = append(containerConfig.Env, "MQ_NATIVE_HA_TLS=true", "MQ_NATIVE_HA_CIPHERSPEC=TLS_AES_256_GCM_SHA384")
		hostConfig := getNativeHASecureHostConfig(t)
		networkingConfig := getNativeHANetworkConfig(qmNetwork.ID)

		ctr := runContainerWithAllConfig(t, cli, &containerConfig, &hostConfig, &networkingConfig, containerNames[i])
		defer cleanContainer(t, cli, ctr)
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
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		t.Fatal(err)
	}

	version, err := getMQVersion(t, cli)
	if err != nil {
		t.Fatal(err)
	}
	if version < "9.2.2.0" {
		t.Skipf("Skipping %s as test requires at least MQ 9.2.2.0, but image is version %s", t.Name(), version)
	}

	containerNames := [3]string{"QM1_1", "QM1_2", "QM1_3"}
	qmReplicaIDs := [3]string{}
	qmNetwork, err := createBridgeNetwork(cli, t)
	if err != nil {
		t.Fatal(err)
	}
	defer removeBridgeNetwork(cli, qmNetwork.ID)

	for i := 0; i <= 2; i++ {
		containerConfig := getNativeHAContainerConfig(containerNames[i], containerNames, defaultHAPort)
		// MQ_NATIVE_HA_CIPHERSPEC is set a FIPS compliant cipherspec.
		containerConfig.Env = append(containerConfig.Env, "MQ_NATIVE_HA_TLS=true", "MQ_NATIVE_HA_CIPHERSPEC=TLS_RSA_WITH_AES_128_GCM_SHA256", "MQ_ENABLE_FIPS=true")
		hostConfig := getNativeHASecureHostConfig(t)
		networkingConfig := getNativeHANetworkConfig(qmNetwork.ID)

		ctr := runContainerWithAllConfig(t, cli, &containerConfig, &hostConfig, &networkingConfig, containerNames[i])
		defer cleanContainer(t, cli, ctr)
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
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		t.Fatal(err)
	}

	version, err := getMQVersion(t, cli)
	if err != nil {
		t.Fatal(err)
	}
	if version < "9.2.2.0" {
		t.Skipf("Skipping %s as test requires at least MQ 9.2.2.0, but image is version %s", t.Name(), version)
	}

	containerNames := [3]string{"QM1_1", "QM1_2", "QM1_3"}
	qmReplicaIDs := [3]string{}
	qmNetwork, err := createBridgeNetwork(cli, t)
	if err != nil {
		t.Fatal(err)
	}
	defer removeBridgeNetwork(cli, qmNetwork.ID)

	for i := 0; i <= 2; i++ {
		containerConfig := getNativeHAContainerConfig(containerNames[i], containerNames, defaultHAPort)
		// MQ_NATIVE_HA_CIPHERSPEC is set a FIPS non-compliant cipherspec - SSL_ECDHE_ECDSA_WITH_RC4_128_SHA
		containerConfig.Env = append(containerConfig.Env, "MQ_NATIVE_HA_TLS=true", "MQ_NATIVE_HA_CIPHERSPEC=TLS_RSA_WITH_AES_128_GCM_SHA256", "MQ_ENABLE_FIPS=true")
		hostConfig := getNativeHASecureHostConfig(t)
		networkingConfig := getNativeHANetworkConfig(qmNetwork.ID)

		ctr, err := runContainerWithAllConfigError(t, cli, &containerConfig, &hostConfig, &networkingConfig, containerNames[i])
		defer cleanContainer(t, cli, ctr)
		// We expect container to fail in this case because the cipher is non-FIPS and we have asked for FIPS compliance
		// by setting MQ_ENABLE_FIPS=true
		if err == nil {
			t.Logf("Container start expected to fail but did not. %v", err)
		}
		qmReplicaIDs[i] = ctr
	}
}
