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
	"testing"

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
	waitForWebReady(t, cli, id, insecureTLSConfig)

	// Stop the container cleanly
	stopContainer(t, cli, id)
}
