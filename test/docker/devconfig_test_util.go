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
	"testing"
	"time"

	"github.com/docker/docker/client"
)

func waitForWebReady(t *testing.T, cli *client.Client, ID string) {
	config := tls.Config{InsecureSkipVerify: true}
	a := fmt.Sprintf("localhost:%s", getWebPort(t, cli, ID))
	for {
		conn, err := tls.Dial("tcp", a, &config)
		if err == nil {
			conn.Close()
			// Extra sleep to allow web apps to start
			time.Sleep(3 * time.Second)
			t.Log("MQ web server is ready")
			return
		}
	}
}
