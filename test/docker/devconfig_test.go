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
	"crypto/tls"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
)

func TestDevGoldenPath(t *testing.T) {
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
	waitForWebReady(t, cli, id)

	timeout := time.Duration(30 * time.Second)
	// Disable TLS verification (server uses a self-signed certificate by default,
	// so verification isn't useful anyway)
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}
	httpClient := http.Client{
		Timeout:   timeout,
		Transport: tr,
	}

	url := fmt.Sprintf("https://localhost:%s/ibmmq/rest/v1/admin/installation", getWebPort(t, cli, id))
	req, err := http.NewRequest("GET", url, nil)
	req.SetBasicAuth("admin", "passw0rd")
	resp, err := httpClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected HTTP status code %v from 'GET installation'; got %v", http.StatusOK, resp.StatusCode)
	}

	// Stop the container cleanly
	stopContainer(t, cli, id)
}
