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
	"fmt"
	"testing"
	"time"
	"net/http"
	"crypto/tls"

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
