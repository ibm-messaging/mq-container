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
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
)

const devAdminPassword string = "passw0rd"
const devAppPassword string = "passw0rd"

// Disable TLS verification (server uses a self-signed certificate by default,
// so verification isn't useful anyway)
var insecureTLSConfig *tls.Config = &tls.Config{
	InsecureSkipVerify: true,
}

func waitForWebReady(t *testing.T, cli *client.Client, ID string, tlsConfig *tls.Config) {
	httpClient := http.Client{
		Timeout: time.Duration(3 * time.Second),
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
		},
	}
	url := fmt.Sprintf("https://localhost:%s/ibmmq/rest/v1/admin/installation", getWebPort(t, cli, ID))
	for {
		req, err := http.NewRequest("GET", url, nil)
		req.SetBasicAuth("admin", devAdminPassword)
		resp, err := httpClient.Do(req)
		if err == nil && resp.StatusCode == http.StatusOK {
			t.Log("MQ web server is ready")
			return
		}
		// conn, err := tls.Dial("tcp", a, &config)
		// if err == nil {
		// 	conn.Close()
		// 	// Extra sleep to allow web apps to start
		// 	time.Sleep(5 * time.Second)
		// 	t.Log("MQ web server is ready")
		// 	return
		// }
		time.Sleep(1 * time.Second)
	}
}

// tlsDir returns the host directory where the test certificate(s) are located
func tlsDir(t *testing.T, unixPath bool) string {
	return filepath.Join(getCwd(t, unixPath), "../tls")
}

// runJMSTests runs a container with a JMS client, which connects to the queue manager container with the specified ID
func runJMSTests(t *testing.T, cli *client.Client, ID string, tls bool, user, password string) {
	containerConfig := container.Config{
		// -e MQ_PORT_1414_TCP_ADDR=9.145.14.173 -e MQ_USERNAME=app -e MQ_PASSWORD=passw0rd -e MQ_CHANNEL=DEV.APP.SVRCONN -e MQ_TLS_KEYSTORE=/tls/test.p12 -e MQ_TLS_PASSPHRASE=passw0rd -v /Users/arthurbarr/go/src/github.com/ibm-messaging/mq-container/test/tls:/tls msgtest
		Env: []string{
			"MQ_PORT_1414_TCP_ADDR=" + getIPAddress(t, cli, ID),
			"MQ_USERNAME=" + user,
			"MQ_CHANNEL=DEV.APP.SVRCONN",
		},
		Image: imageNameDevJMS(),
	}
	// Set a password for the client to use, if one is specified
	if password != "" {
		containerConfig.Env = append(containerConfig.Env, "MQ_PASSWORD="+password)
	}
	if tls {
		t.Log("Using TLS from JMS client")
		containerConfig.Env = append(containerConfig.Env, []string{
			"MQ_TLS_TRUSTSTORE=/var/tls/client-trust.jks",
			"MQ_TLS_PASSPHRASE=passw0rd",
		}...)
	}
	hostConfig := container.HostConfig{
		Binds: []string{
			coverageBind(t),
			tlsDir(t, false) + ":/var/tls",
		},
	}
	networkingConfig := network.NetworkingConfig{}
	ctr, err := cli.ContainerCreate(context.Background(), &containerConfig, &hostConfig, &networkingConfig, strings.Replace(t.Name()+"JMS", "/", "", -1))
	if err != nil {
		t.Fatal(err)
	}
	startContainer(t, cli, ctr.ID)
	rc := waitForContainer(t, cli, ctr.ID, 10)
	if rc != 0 {
		t.Errorf("JUnit container failed with rc=%v", rc)
	}
	defer cleanContainer(t, cli, ctr.ID)
}

// createTLSConfig creates a tls.Config which trusts the specified certificate
func createTLSConfig(t *testing.T, certFile, password string) *tls.Config {
	// Get the SystemCertPool, continue with an empty pool on error
	certs, err := x509.SystemCertPool()
	if err != nil {
		t.Fatal(err)
	}
	// Read in the cert file
	cert, err := ioutil.ReadFile(certFile)
	if err != nil {
		t.Fatal(err)
	}
	// Append our cert to the system pool
	ok := certs.AppendCertsFromPEM(cert)
	if !ok {
		t.Fatal("No certs appended")
	}
	// Trust the augmented cert pool in our client
	return &tls.Config{
		InsecureSkipVerify: false,
		RootCAs:            certs,
	}
}

func testREST(t *testing.T, cli *client.Client, ID string, tlsConfig *tls.Config) {
	httpClient := http.Client{
		Timeout: time.Duration(30 * time.Second),
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
		},
	}

	url := fmt.Sprintf("https://localhost:%s/ibmmq/rest/v1/admin/installation", getWebPort(t, cli, ID))
	req, err := http.NewRequest("GET", url, nil)
	req.SetBasicAuth("admin", devAdminPassword)
	resp, err := httpClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected HTTP status code %v from 'GET installation'; got %v", http.StatusOK, resp.StatusCode)
	}
}
