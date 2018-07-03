// +build mqdev

/*
© Copyright IBM Corporation 2018

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

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
)

// TestDevGoldenPath tests using the default values for the default developer config.
// Note: This test requires a separate container image to be available for the JMS tests.
func TestDevGoldenPath(t *testing.T) {
	t.Parallel()
	cli, err := client.NewEnvClient()
	if err != nil {
		t.Fatal(err)
	}
	containerConfig := container.Config{
		Env: []string{
			"LICENSE=accept",
			"MQ_QMGR_NAME=qm1",
		},
	}
	id := runContainerWithPorts(t, cli, &containerConfig, []int{9443})
	defer cleanContainer(t, cli, id)
	waitForReady(t, cli, id)
	waitForWebReady(t, cli, id, insecureTLSConfig)
	t.Run("JMS", func(t *testing.T) {
		// Run the JMS tests, with no password specified
		runJMSTests(t, cli, id, false, "app", "")
	})
	// Stop the container cleanly
	stopContainer(t, cli, id)
}

// TestDevSecure tests the default developer config using the a custom TLS key store and password.
// Note: This test requires a separate container image to be available for the JMS tests
func TestDevSecure(t *testing.T) {
	t.Parallel()
	cli, err := client.NewEnvClient()
	if err != nil {
		t.Fatal(err)
	}
	const tlsPassPhrase string = "passw0rd"
	containerConfig := container.Config{
		Env: []string{
			"LICENSE=accept",
			"MQ_QMGR_NAME=qm1",
			"MQ_APP_PASSWORD=" + devAppPassword,
			"MQ_TLS_KEYSTORE=/var/tls/server.p12",
			"MQ_TLS_PASSPHRASE=" + tlsPassPhrase,
			"DEBUG=1",
		},
		Image: imageName(),
	}
	hostConfig := container.HostConfig{
		Binds: []string{
			coverageBind(t),
			tlsDir(t, false) + ":/var/tls",
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
	runJMSTests(t, cli, ctr.ID, true, "app", devAppPassword)
	// Stop the container cleanly
	stopContainer(t, cli, ctr.ID)
}

func TestDevWebDisabled(t *testing.T) {
	t.Parallel()
	cli, err := client.NewEnvClient()
	if err != nil {
		t.Fatal(err)
	}
	containerConfig := container.Config{
		Env: []string{
			"LICENSE=accept",
			"MQ_QMGR_NAME=qm1",
			"MQ_DISABLE_WEB_CONSOLE=true",
		},
	}
	id := runContainer(t, cli, &containerConfig)
	defer cleanContainer(t, cli, id)
	waitForReady(t, cli, id)
	t.Run("Web", func(t *testing.T) {
		_, dspmqweb := execContainer(t, cli, id, "mqm", []string{"dspmqweb"})
		if !strings.Contains(dspmqweb, "Server mqweb is not running.") && !strings.Contains(dspmqweb, "MQWB1125I") {
			t.Errorf("Expected dspmqweb to say 'Server is not running' or 'MQWB1125I'; got \"%v\"", dspmqweb)
		}
	})
	t.Run("JMS", func(t *testing.T) {
		// Run the JMS tests, with no password specified
		runJMSTests(t, cli, id, false, "app", "")
	})
	// Stop the container cleanly
	stopContainer(t, cli, id)
}

func TestDevConfigDisabled(t *testing.T) {
	t.Parallel()
	cli, err := client.NewEnvClient()
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
	rc, _ := execContainer(t, cli, id, "mqm", []string{"bash", "-c", "echo 'display qlocal(DEV*)' | runmqsc"})
	if rc == 0 {
		t.Errorf("Expected DEV queues to be missing")
	}
	// Stop the container cleanly
	stopContainer(t, cli, id)
}
