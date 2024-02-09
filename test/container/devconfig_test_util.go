//go:build mqdev
// +build mqdev

/*
© Copyright IBM Corporation 2018, 2024

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
	"io"
	"net/http"
	"net/http/httputil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	ce "github.com/ibm-messaging/mq-container/test/container/containerengine"
	"github.com/ibm-messaging/mq-container/test/container/pathutils"
)

const defaultAdminPassword string = "passw0rd"
const defaultAppPasswordOS string = "passw0rd"
const defaultAppPasswordWeb string = "passw0rd"

// Disable TLS verification (server uses a self-signed certificate by default,
// so verification isn't useful anyway)
var insecureTLSConfig *tls.Config = &tls.Config{
	InsecureSkipVerify: true,
}

func waitForWebReady(t *testing.T, cli ce.ContainerInterface, ID string, tlsConfig *tls.Config) {
	t.Logf("%s Waiting for web server to be ready", time.Now().Format(time.RFC3339))
	httpClient := http.Client{
		Timeout: time.Duration(10 * time.Second),
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
		},
	}
	port, err := cli.GetContainerPort(ID, 9443)
	if err != nil {
		t.Fatal(err)
	}
	url := fmt.Sprintf("https://localhost:%s/ibmmq/rest/v1/admin/installation", port)
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
	return pathutils.CleanPath(filepath.Dir(getCwd(t, unixPath)), "../tls")
}

func tlsDirWithCA(t *testing.T, unixPath bool) string {
	return pathutils.CleanPath(filepath.Dir(getCwd(t, unixPath)), "../tlscacert")
}

func tlsDirInvalid(t *testing.T, unixPath bool) string {
	return pathutils.CleanPath(filepath.Dir(getCwd(t, unixPath)), "../tlsinvalidcert")
}

// runJMSTests runs a container with a JMS client, which connects to the queue manager container with the specified ID
func runJMSTests(t *testing.T, cli ce.ContainerInterface, ID string, tls bool, user, password string, ibmjre string, cipherName string) {
	port, err := cli.GetContainerPort(ID, 1414)
	if err != nil {
		t.Error(err)
	}
	containerConfig := ce.ContainerConfig{
		// -e MQ_PORT_1414_TCP_ADDR=9.145.14.173 -e MQ_USERNAME=app -e MQ_PASSWORD=passw0rd -e MQ_CHANNEL=DEV.APP.SVRCONN -e MQ_TLS_TRUSTSTORE=/tls/test.p12 -e MQ_TLS_PASSPHRASE=passw0rd -v /Users/arthurbarr/go/src/github.com/ibm-messaging/mq-container/test/tls:/tls msgtest
		Env: []string{
			"MQ_PORT_1414_TCP_ADDR=127.0.0.1",
			"MQ_PORT_1414_OVERRIDE=" + port,
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
	hostConfig := ce.ContainerHostConfig{
		Binds: []string{
			coverageBind(t),
			tlsDir(t, false) + ":/var/tls",
		},
	}
	networkingConfig := ce.ContainerNetworkSettings{
		Networks: []string{"host"},
	}
	jmsID, err := cli.ContainerCreate(&containerConfig, &hostConfig, &networkingConfig, strings.Replace(t.Name()+"JMS", "/", "", -1))
	if err != nil {
		t.Fatal(err)
	}
	startContainer(t, cli, jmsID)
	rc := waitForContainer(t, cli, jmsID, 2*time.Minute)
	if rc != 0 {
		t.Errorf("JUnit container failed with rc=%v", rc)
	}

	// Get console output of the container and process the lines
	// to see if we have any failures
	scanner := bufio.NewScanner(strings.NewReader(inspectLogs(t, cli, jmsID)))
	for scanner.Scan() {
		s := scanner.Text()
		if processJunitLogLine(s) {
			t.Errorf("JUnit container tests failed. Reason: %s", s)
		}
	}

	defer cleanContainer(t, cli, jmsID)
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
	cert, err := os.ReadFile(certFile)
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

func testRESTAdmin(t *testing.T, cli ce.ContainerInterface, ID string, tlsConfig *tls.Config, errorExpected string) {
	httpClient := http.Client{
		Timeout: time.Duration(30 * time.Second),
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
		},
	}
	port, err := cli.GetContainerPort(ID, 9443)
	if err != nil {
		t.Fatal(err)
	}
	url := fmt.Sprintf("https://localhost:%s/ibmmq/rest/v1/admin/installation", port)
	req, err := http.NewRequest("GET", url, nil)
	req.SetBasicAuth("admin", defaultAdminPassword)
	resp, err := httpClient.Do(req)
	if err != nil {
		if len(errorExpected) > 0 {
			if !strings.Contains(err.Error(), errorExpected) {
				t.Fatal(err)
			}
		} else {
			t.Fatal(err)
		}
	}
	if resp != nil && resp.StatusCode != http.StatusOK {
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

func testRESTMessaging(t *testing.T, cli ce.ContainerInterface, ID string, tlsConfig *tls.Config, qmName string, user string, password string, errorExpected string) {
	httpClient := http.Client{
		Timeout: time.Duration(30 * time.Second),
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
		},
	}
	q := "DEV.QUEUE.1"
	port, err := cli.GetContainerPort(ID, 9443)
	if err != nil {
		t.Fatal(err)
	}
	url := fmt.Sprintf("https://localhost:%s/ibmmq/rest/v1/messaging/qmgr/%s/queue/%s/message", port, qmName, q)
	putMessage := []byte("Hello")
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(putMessage))
	req.SetBasicAuth(user, password)
	req.Header.Add("ibm-mq-rest-csrf-token", "n/a")
	req.Header.Add("Content-Type", "text/plain;charset=utf-8")
	logHTTPRequest(t, req)
	resp, err := httpClient.Do(req)
	if err != nil {
		if len(errorExpected) > 0 {
			if strings.Contains(err.Error(), errorExpected) {
				t.Logf("Error contains expected '%s' value", errorExpected)
				return
			} else {
				t.Fatal(err)
			}
		} else {
			t.Fatal(err)
		}
	}
	logHTTPResponse(t, resp)
	if resp != nil && resp.StatusCode != http.StatusCreated {
		if strings.Contains(resp.Status, errorExpected) {
			t.Logf("HTTP Response code is as expected. %s", resp.Status)
			return
		} else {
			t.Errorf("Expected HTTP status code %v from 'POST to queue'; got %v", http.StatusOK, resp.StatusCode)
			t.Logf("HTTP response: %+v", resp)
			t.Fail()
		}
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
	gotMessage, err := io.ReadAll(resp.Body)
	//gotMessage := string(b)
	if string(gotMessage) != string(putMessage) {
		t.Errorf("Expected payload to be \"%s\"; got \"%s\"", putMessage, gotMessage)
	}
}

// createTLSConfig creates a tls.Config which trusts the specified certificate
func createTLSConfigWithCipher(t *testing.T, certFile, password string, ciphers []uint16) *tls.Config {
	// Get the SystemCertPool, continue with an empty pool on error
	certs, err := x509.SystemCertPool()
	if err != nil {
		t.Fatal(err)
	}
	// Read in the cert file
	cert, err := os.ReadFile(certFile)
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
		CipherSuites:       ciphers,
	}
}
