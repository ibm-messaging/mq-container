//go:build mqdev
// +build mqdev

/*
© Copyright IBM Corporation 2018, 2022

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
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
)

// TestDevGoldenPath tests using the default values for the default developer config.
// Note: This test requires a separate container image to be available for the JMS tests.
func TestDevGoldenPath(t *testing.T) {
	t.Parallel()

	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		t.Fatal(err)
	}
	qm := "qm1"
	containerConfig := container.Config{
		Env: []string{
			"LICENSE=accept",
			"MQ_QMGR_NAME=" + qm,
			"DEBUG=true",
		},
	}
	id := runContainerWithPorts(t, cli, &containerConfig, []int{9443})
	defer cleanContainer(t, cli, id)
	waitForReady(t, cli, id)
	waitForWebReady(t, cli, id, insecureTLSConfig)
	t.Run("JMS", func(t *testing.T) {
		// Run the JMS tests, with no password specified.
		// Use OpenJDK JRE for running testing, pass false for 7th parameter.
		// Last parameter is blank as the test doesn't use TLS.
		runJMSTests(t, cli, id, false, "app", defaultAppPasswordOS, "false", "")
	})
	t.Run("REST admin", func(t *testing.T) {
		testRESTAdmin(t, cli, id, insecureTLSConfig)
	})
	t.Run("REST messaging", func(t *testing.T) {
		testRESTMessaging(t, cli, id, insecureTLSConfig, qm, "app", defaultAppPasswordWeb)
	})
	// Stop the container cleanly
	stopContainer(t, cli, id)
}

// TestDevSecure tests the default developer config using the a custom TLS key store and password.
// Note: This test requires a separate container image to be available for the JMS tests
func TestDevSecure(t *testing.T) {
	t.Parallel()

	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		t.Fatal(err)
	}

	const tlsPassPhrase string = "passw0rd"
	qm := "qm1"
	appPassword := "differentPassw0rd"
	containerConfig := container.Config{
		Env: []string{
			"LICENSE=accept",
			"MQ_QMGR_NAME=" + qm,
			"MQ_APP_PASSWORD=" + appPassword,
			"DEBUG=1",
			"WLP_LOGGING_MESSAGE_FORMAT=JSON",
			"MQ_ENABLE_EMBEDDED_WEB_SERVER_LOG=true",
		},
		Image: imageName(),
	}
	hostConfig := container.HostConfig{
		Binds: []string{
			coverageBind(t),
			tlsDir(t, false) + ":/etc/mqm/pki/keys/default",
		},
		// Assign a random port for the web server on the host
		// TODO: Don't do this for all tests
		PortBindings: nat.PortMap{
			"9443/tcp": []nat.PortBinding{
				{
					HostIP: "0.0.0.0",
				},
			},
		},
	}
	networkingConfig := network.NetworkingConfig{}
	ctr, err := cli.ContainerCreate(context.Background(), &containerConfig, &hostConfig, &networkingConfig, t.Name())
	if err != nil {
		t.Fatal(err)
	}
	defer cleanContainer(t, cli, ctr.ID)
	startContainer(t, cli, ctr.ID)
	waitForReady(t, cli, ctr.ID)
	cert := filepath.Join(tlsDir(t, true), "server.crt")
	waitForWebReady(t, cli, ctr.ID, createTLSConfig(t, cert, tlsPassPhrase))

	t.Run("JMS", func(t *testing.T) {
		// OpenJDK is used for running tests, hence pass "false" for 7th parameter.
		// Cipher name specified is compliant with non-IBM JRE naming.
		runJMSTests(t, cli, ctr.ID, true, "app", appPassword, "false", "TLS_RSA_WITH_AES_256_CBC_SHA256")
	})
	t.Run("REST admin", func(t *testing.T) {
		testRESTAdmin(t, cli, ctr.ID, insecureTLSConfig)
	})
	t.Run("REST messaging", func(t *testing.T) {
		testRESTMessaging(t, cli, ctr.ID, insecureTLSConfig, qm, "app", appPassword)
	})

	// Stop the container cleanly
	stopContainer(t, cli, ctr.ID)
}

func TestDevWebDisabled(t *testing.T) {
	t.Parallel()

	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		t.Fatal(err)
	}
	containerConfig := container.Config{
		Env: []string{
			"LICENSE=accept",
			"MQ_QMGR_NAME=qm1",
			"MQ_ENABLE_EMBEDDED_WEB_SERVER=false",
		},
	}
	id := runContainer(t, cli, &containerConfig)
	defer cleanContainer(t, cli, id)
	waitForReady(t, cli, id)
	t.Run("Web", func(t *testing.T) {
		_, dspmqweb := execContainer(t, cli, id, "", []string{"dspmqweb"})
		if !strings.Contains(dspmqweb, "Server mqweb is not running.") && !strings.Contains(dspmqweb, "MQWB1125I") {
			t.Errorf("Expected dspmqweb to say 'Server is not running' or 'MQWB1125I'; got \"%v\"", dspmqweb)
		}
	})
	t.Run("JMS", func(t *testing.T) {
		// Run the JMS tests, with no password specified
		// OpenJDK is used for running tests, hence pass "false" for 7th parameter.
		// Last parameter is blank as the test doesn't use TLS.
		runJMSTests(t, cli, id, false, "app", defaultAppPasswordOS, "false", "")
	})
	// Stop the container cleanly
	stopContainer(t, cli, id)
}

func TestDevConfigDisabled(t *testing.T) {
	t.Parallel()

	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		t.Fatal(err)
	}
	containerConfig := container.Config{
		Env: []string{
			"LICENSE=accept",
			"MQ_QMGR_NAME=qm1",
			"MQ_DEV=false",
		},
	}
	id := runContainerWithPorts(t, cli, &containerConfig, []int{9443})
	defer cleanContainer(t, cli, id)
	waitForReady(t, cli, id)
	waitForWebReady(t, cli, id, insecureTLSConfig)
	rc, _ := execContainer(t, cli, id, "", []string{"bash", "-c", "echo 'display qlocal(DEV*)' | runmqsc"})
	if rc == 0 {
		t.Errorf("Expected DEV queues to be missing")
	}
	// Stop the container cleanly
	stopContainer(t, cli, id)
}

// Test if SSLKEYR and CERTLABL attributes are not set when key and certificate
// are not supplied.
func TestSSLKEYRBlank(t *testing.T) {
	t.Parallel()

	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		t.Fatal(err)
	}
	containerConfig := container.Config{
		Env: []string{
			"LICENSE=accept",
			"MQ_QMGR_NAME=QM1",
			"MQ_ENABLE_EMBEDDED_WEB_SERVER=false",
		},
	}
	id := runContainerWithPorts(t, cli, &containerConfig, []int{9443})
	defer cleanContainer(t, cli, id)
	waitForReady(t, cli, id)

	// execute runmqsc to display qmgr SSLKEYR and CERTLABL attibutes.
	// Search the console output for exepcted values
	_, sslkeyROutput := execContainer(t, cli, id, "", []string{"bash", "-c", "echo 'DISPLAY QMGR SSLKEYR CERTLABL' | runmqsc"})
	if !strings.Contains(sslkeyROutput, "SSLKEYR( )") || !strings.Contains(sslkeyROutput, "CERTLABL( )") {
		// Although queue manager is ready, it may be that MQSC scripts have not been applied yet.
		// Hence wait for a second and retry few times before giving up.
		waitCount := 30
		var i int
		for i = 0; i < waitCount; i++ {
			time.Sleep(1 * time.Second)
			_, sslkeyROutput = execContainer(t, cli, id, "", []string{"bash", "-c", "echo 'DISPLAY QMGR SSLKEYR CERTLABL' | runmqsc"})
			if strings.Contains(sslkeyROutput, "SSLKEYR( )") && strings.Contains(sslkeyROutput, "CERTLABL( )") {
				break
			}
		}
		// Failed to get expected output? dump the contents of mqsc files.
		if i == waitCount {
			_, tls15mqsc := execContainer(t, cli, id, "", []string{"cat", "/etc/mqm/15-tls.mqsc"})
			_, autoMQSC := execContainer(t, cli, id, "", []string{"cat", "/mnt/mqm/data/qmgrs/QM1/autocfg/cached.mqsc"})
			t.Errorf("Expected SSLKEYR to be blank but it is not; got \"%v\"\n AutoConfig MQSC file contents %v\n 15-tls: %v", sslkeyROutput, autoMQSC, tls15mqsc)
		}
	}

	// Stop the container cleanly
	stopContainer(t, cli, id)
}

// Test if SSLKEYR and CERTLABL attributes are set when key and certificate
// are supplied.
func TestSSLKEYRWithSuppliedKeyAndCert(t *testing.T) {
	t.Parallel()

	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		t.Fatal(err)
	}

	containerConfig := container.Config{
		Env: []string{
			"LICENSE=accept",
			"MQ_QMGR_NAME=QM1",
			"MQ_ENABLE_EMBEDDED_WEB_SERVER=false",
		},
		Image: imageName(),
	}
	hostConfig := container.HostConfig{
		Binds: []string{
			coverageBind(t),
			tlsDir(t, false) + ":/etc/mqm/pki/keys/default",
		},
	}
	networkingConfig := network.NetworkingConfig{}
	ctr, err := cli.ContainerCreate(context.Background(), &containerConfig, &hostConfig, &networkingConfig, t.Name())
	if err != nil {
		t.Fatal(err)
	}
	defer cleanContainer(t, cli, ctr.ID)
	startContainer(t, cli, ctr.ID)
	waitForReady(t, cli, ctr.ID)

	// execute runmqsc to display qmgr SSLKEYR and CERTLABL attibutes.
	// Search the console output for exepcted values
	_, sslkeyROutput := execContainer(t, cli, ctr.ID, "", []string{"bash", "-c", "echo 'DISPLAY QMGR SSLKEYR CERTLABL' | runmqsc"})
	if !strings.Contains(sslkeyROutput, "SSLKEYR(/run/runmqserver/tls/key)") || !strings.Contains(sslkeyROutput, "CERTLABL(default)") {
		// Although queue manager is ready, it may be that MQSC scripts have not been applied yet.
		// Hence wait for a second and retry few times before giving up.
		waitCount := 30
		var i int
		for i = 0; i < waitCount; i++ {
			time.Sleep(1 * time.Second)
			_, sslkeyROutput = execContainer(t, cli, ctr.ID, "", []string{"bash", "-c", "echo 'DISPLAY QMGR SSLKEYR CERTLABL' | runmqsc"})
			if strings.Contains(sslkeyROutput, "SSLKEYR(/run/runmqserver/tls/key)") && strings.Contains(sslkeyROutput, "CERTLABL(default)") {
				break
			}
		}
		// Failed to get expected output? dump the contents of mqsc files.
		if i == waitCount {
			_, tls15mqsc := execContainer(t, cli, ctr.ID, "", []string{"cat", "/etc/mqm/15-tls.mqsc"})
			_, autoMQSC := execContainer(t, cli, ctr.ID, "", []string{"cat", "/mnt/mqm/data/qmgrs/QM1/autocfg/cached.mqsc"})
			t.Errorf("Expected SSLKEYR to be '/run/runmqserver/tls/key' but it is not; got \"%v\" \n AutoConfig MQSC file contents %v\n 15-tls: %v", sslkeyROutput, autoMQSC, tls15mqsc)
		}
	}

	// Stop the container cleanly
	stopContainer(t, cli, ctr.ID)
}

// Test with CA cert
func TestSSLKEYRWithCACert(t *testing.T) {
	t.Parallel()

	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		t.Fatal(err)
	}

	containerConfig := container.Config{
		Env: []string{
			"LICENSE=accept",
			"MQ_QMGR_NAME=QM1",
			"MQ_ENABLE_EMBEDDED_WEB_SERVER=false",
		},
		Image: imageName(),
	}
	hostConfig := container.HostConfig{
		Binds: []string{
			coverageBind(t),
			tlsDirWithCA(t, false) + ":/etc/mqm/pki/keys/QM1CA",
		},
		// Assign a random port for the web server on the host
		PortBindings: nat.PortMap{
			"9443/tcp": []nat.PortBinding{
				{
					HostIP: "0.0.0.0",
				},
			},
		},
	}
	networkingConfig := network.NetworkingConfig{}
	ctr, err := cli.ContainerCreate(context.Background(), &containerConfig, &hostConfig, &networkingConfig, t.Name())
	if err != nil {
		t.Fatal(err)
	}
	defer cleanContainer(t, cli, ctr.ID)
	startContainer(t, cli, ctr.ID)
	waitForReady(t, cli, ctr.ID)

	// execute runmqsc to display qmgr SSLKEYR and CERTLABL attibutes.
	// Search the console output for exepcted values
	_, sslkeyROutput := execContainer(t, cli, ctr.ID, "", []string{"bash", "-c", "echo 'DISPLAY QMGR SSLKEYR CERTLABL' | runmqsc"})
	if !strings.Contains(sslkeyROutput, "SSLKEYR(/run/runmqserver/tls/key)") {
		// Although queue manager is ready, it may be that MQSC scripts have not been applied yet.
		// Hence wait for a second and retry few times before giving up.
		waitCount := 30
		var i int
		for i = 0; i < waitCount; i++ {
			time.Sleep(1 * time.Second)
			_, sslkeyROutput = execContainer(t, cli, ctr.ID, "", []string{"bash", "-c", "echo 'DISPLAY QMGR SSLKEYR CERTLABL' | runmqsc"})
			if strings.Contains(sslkeyROutput, "SSLKEYR(/run/runmqserver/tls/key)") {
				break
			}
		}
		// Failed to get expected output? dump the contents of mqsc files.
		if i == waitCount {
			_, tls15mqsc := execContainer(t, cli, ctr.ID, "", []string{"cat", "/etc/mqm/15-tls.mqsc"})
			_, autoMQSC := execContainer(t, cli, ctr.ID, "", []string{"cat", "/mnt/mqm/data/qmgrs/QM1/autocfg/cached.mqsc"})
			t.Errorf("Expected SSLKEYR to be '/run/runmqserver/tls/key' but it is not; got \"%v\"\n AutoConfig MQSC file contents %v\n 15-tls: %v", sslkeyROutput, autoMQSC, tls15mqsc)
		}
	}

	if !strings.Contains(sslkeyROutput, "CERTLABL(QM1CA)") {
		_, autoMQSC := execContainer(t, cli, ctr.ID, "", []string{"cat", "/etc/mqm/15-tls.mqsc"})
		t.Errorf("Expected CERTLABL to be 'QM1CA' but it is not; got \"%v\" \n MQSC File contents %v", sslkeyROutput, autoMQSC)
	}

	// Stop the container cleanly
	stopContainer(t, cli, ctr.ID)
}
