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
	"bufio"
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
)

const defaultAdminPassword string = "passw0rd"
const defaultAppPasswordOS string = ""
const defaultAppPasswordWeb string = "passw0rd"

// Disable TLS verification (server uses a self-signed certificate by default,
// so verification isn't useful anyway)
var insecureTLSConfig *tls.Config = &tls.Config{
	InsecureSkipVerify: true,
}

func waitForWebReady(t *testing.T, cli *client.Client, ID string, tlsConfig *tls.Config) {
	t.Logf("%s Waiting for web server to be ready", time.Now().Format(time.RFC3339))
	httpClient := http.Client{
		Timeout: time.Duration(10 * time.Second),
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
		},
	}
	url := fmt.Sprintf("https://localhost:%s/ibmmq/rest/v1/admin/installation", getPort(t, cli, ID, 9443))
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	for {
		select {
		case <-time.After(1 * time.Second):
			req, err := http.NewRequest("GET", url, nil)
			req.SetBasicAuth("admin", defaultAdminPassword)
			resp, err := httpClient.Do(req)
			if err == nil && resp.StatusCode == http.StatusOK {
				t.Logf("%s MQ web server is ready", time.Now().Format(time.RFC3339))
				return
			}
		case <-ctx.Done():
			t.Fatalf("%s Timed out waiting for web server to become ready", time.Now().Format(time.RFC3339))
		}
	}
}

// tlsDir returns the host directory where the test certificate(s) are located
func tlsDir(t *testing.T, unixPath bool) string {
	return filepath.Join(getCwd(t, unixPath), "../tls")
}

// runJMSTests runs a container with a JMS client, which connects to the queue manager container with the specified ID
func runJMSTests(t *testing.T, cli *client.Client, ID string, tls bool, user, password string, ibmjre string, cipherName string) {
	containerConfig := container.Config{
		// -e MQ_PORT_1414_TCP_ADDR=9.145.14.173 -e MQ_USERNAME=app -e MQ_PASSWORD=passw0rd -e MQ_CHANNEL=DEV.APP.SVRCONN -e MQ_TLS_TRUSTSTORE=/tls/test.p12 -e MQ_TLS_PASSPHRASE=passw0rd -v /Users/arthurbarr/go/src/github.com/ibm-messaging/mq-container/test/tls:/tls msgtest
		Env: []string{
			"MQ_PORT_1414_TCP_ADDR=" + getIPAddress(t, cli, ID),
			"MQ_USERNAME=" + user,
			"MQ_CHANNEL=DEV.APP.SVRCONN",
			"IBMJRE=" + ibmjre,
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
			"MQ_TLS_CIPHER=" + cipherName,
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
	rc := waitForContainer(t, cli, ctr.ID, 2*time.Minute)
	if rc != 0 {
		t.Errorf("JUnit container failed with rc=%v", rc)
	}

	// Get console output of the container and process the lines
	// to see if we have any failures
	scanner := bufio.NewScanner(strings.NewReader(inspectLogs(t, cli, ctr.ID)))
	for scanner.Scan() {
		s := scanner.Text()
		if processJunitLogLine(s) {
			t.Errorf("JUnit container tests failed. Reason: %s", s)
		}
	}

	defer cleanContainer(t, cli, ctr.ID)
}

// Parse JUnit log line and return true if line contains failed or aborted tests
func processJunitLogLine(outputLine string) bool {
	var failedLine bool
	// Sample JUnit test run output
	//[         2 containers found      ]
	//[         0 containers skipped    ]
	//[         2 containers started    ]
	//[         0 containers aborted    ]
	//[         2 containers successful ]
	//[         0 containers failed     ]
	//[         0 tests found           ]
	//[         0 tests skipped         ]
	//[         0 tests started         ]
	//[         0 tests aborted         ]
	//[         0 tests successful      ]
	//[         0 tests failed          ]

	// Consider only those lines that begin with '[' and with ']'
	if strings.HasPrefix(outputLine, "[") && strings.HasSuffix(outputLine, "]") {
		// Strip off [] and whitespaces
		trimmed := strings.Trim(outputLine, "[] ")
		if strings.Contains(trimmed, "aborted") || strings.Contains(trimmed, "failed") {
			// Tokenize on whitespace
			tokens := strings.Split(trimmed, " ")
			// Determine the count of aborted or failed tests
			count, err := strconv.Atoi(tokens[0])
			if err == nil {
				if count > 0 {
					failedLine = true
				}
			}
		}
	}

	return failedLine
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

func testRESTAdmin(t *testing.T, cli *client.Client, ID string, tlsConfig *tls.Config) {
	httpClient := http.Client{
		Timeout: time.Duration(30 * time.Second),
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
		},
	}
	url := fmt.Sprintf("https://localhost:%s/ibmmq/rest/v1/admin/installation", getPort(t, cli, ID, 9443))
	req, err := http.NewRequest("GET", url, nil)
	req.SetBasicAuth("admin", defaultAdminPassword)
	resp, err := httpClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected HTTP status code %v from 'GET installation'; got %v", http.StatusOK, resp.StatusCode)
	}
}

// curl -i -k https://localhost:1234/ibmmq/rest/v1/messaging/qmgr/qm1/queue/DEV.QUEUE.1/message -X POST -u app -H “ibm-mq-rest-csrf-token: N/A” -H “Content-Type: text/plain;charset=utf-8" -d “Hello World”

func logHTTPRequest(t *testing.T, req *http.Request) {
	d, err := httputil.DumpRequestOut(req, true)
	if err != nil {
		t.Error(err)
	}
	t.Logf("HTTP request: %v", string(d))
}

func logHTTPResponse(t *testing.T, resp *http.Response) {
	d, err := httputil.DumpResponse(resp, true)
	if err != nil {
		t.Error(err)
	}
	t.Logf("HTTP response: %v", string(d))
}

func testRESTMessaging(t *testing.T, cli *client.Client, ID string, tlsConfig *tls.Config, qmName string, user string, password string) {
	httpClient := http.Client{
		Timeout: time.Duration(30 * time.Second),
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
		},
	}
	q := "DEV.QUEUE.1"
	url := fmt.Sprintf("https://localhost:%s/ibmmq/rest/v1/messaging/qmgr/%s/queue/%s/message", getPort(t, cli, ID, 9443), qmName, q)
	putMessage := []byte("Hello")
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(putMessage))
	req.SetBasicAuth(user, password)
	req.Header.Add("ibm-mq-rest-csrf-token", "n/a")
	req.Header.Add("Content-Type", "text/plain;charset=utf-8")
	logHTTPRequest(t, req)
	resp, err := httpClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	logHTTPResponse(t, resp)
	if resp.StatusCode != http.StatusCreated {
		t.Errorf("Expected HTTP status code %v from 'POST to queue'; got %v", http.StatusOK, resp.StatusCode)
		t.Logf("HTTP response: %+v", resp)
		t.Fail()
	}

	req, err = http.NewRequest("DELETE", url, nil)
	req.Header.Add("ibm-mq-rest-csrf-token", "n/a")
	req.SetBasicAuth(user, password)
	logHTTPRequest(t, req)
	resp, err = httpClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	logHTTPResponse(t, resp)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected HTTP status code %v from 'DELETE from queue'; got %v", http.StatusOK, resp.StatusCode)
		t.Logf("HTTP response: %+v", resp)
		t.Fail()
	}
	gotMessage, err := ioutil.ReadAll(resp.Body)
	//gotMessage := string(b)
	if string(gotMessage) != string(putMessage) {
		t.Errorf("Expected payload to be \"%s\"; got \"%s\"", putMessage, gotMessage)
	}
}
