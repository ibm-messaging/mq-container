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
	"crypto/tls"
	"path/filepath"
	"testing"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
)

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
			// TODO: Use default password (not set) here
			"MQ_APP_PASSWORD=" + devAppPassword,
		},
	}
	id := runContainer(t, cli, &containerConfig)

	defer cleanContainer(t, cli, id)
	waitForReady(t, cli, id)
	waitForWebReady(t, cli, id)

	t.Run("REST", func(t *testing.T) {
		// Disable TLS verification (server uses a self-signed certificate by default,
		// so verification isn't useful anyway)
		testREST(t, cli, id, &tls.Config{
			InsecureSkipVerify: true,
		})
	})
	t.Run("JMS", func(t *testing.T) {
		runJMSTests(t, cli, id, false)
	})

	// Stop the container cleanly
	stopContainer(t, cli, id)
}

func TestDevTLS(t *testing.T) {
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
			tlsDir(t) + ":/var/tls",
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
	waitForWebReady(t, cli, ctr.ID)

	t.Run("REST", func(t *testing.T) {
		// Use the correct certificate for the HTTPS connection
		cert := filepath.Join(tlsDir(t), "server.crt")
		testREST(t, cli, ctr.ID, createTLSConfig(t, cert, tlsPassPhrase))
	})
	t.Run("JMS", func(t *testing.T) {
		runJMSTests(t, cli, ctr.ID, true)
	})

	// Stop the container cleanly
	stopContainer(t, cli, ctr.ID)
}
