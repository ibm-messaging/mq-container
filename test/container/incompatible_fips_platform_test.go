//go:build arm64 || s390x

/*
© Copyright IBM Corporation 2026

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
	"strings"
	"testing"

	ce "github.com/ibm-messaging/mq-container/test/container/containerengine"
)

// When enabling FIPS on s390x or arm64 we expect a container message stating
// it was disabled and the SSLFIPS queue manager status to be NO. Due to build
// constraints test will only be run on arm64 or s390x.
func TestFipsDisabledOnPlatform(t *testing.T) {
	t.Parallel()

	cli := ce.NewContainerClient()

	containerConfig := ce.ContainerConfig{
		Env: []string{
			"LICENSE=accept",
			"MQ_QMGR_NAME=QM1",
			"MQ_ENABLE_EMBEDDED_WEB_SERVER=false",
			"MQ_ENABLE_FIPS=true",
		},
	}
	ID := runContainer(t, cli, &containerConfig)

	cleanupAfterTest(t, cli, ID, false)
	waitForReady(t, cli, ID)

	// Getting the arch of the image we are testing, due to build constraints
	// arch will only be either arm64 or s390x
	arch, err := cli.ImageInspectWithFormat("{{.Architecture}}", imageName())
	if err != nil {
		t.Fatal(err)
	}
	arch = strings.TrimSpace(arch)

	// Check for expected message on container log
	expectedFIPSMsg := fmt.Sprintf("Warning: FIPS mode was requested but is unavailable on the %s architecture.", arch)
	logs := inspectLogs(t, cli, ID)
	if !strings.Contains(logs, expectedFIPSMsg) {
		t.Errorf("Expected '%s' but got %v\n", expectedFIPSMsg, logs)
	}

	// execute runmqsc to display qmgr SSLFIPS attibute.
	_, sslFIPSOutput := execContainer(t, cli, ID, "", []string{"bash", "-c", "echo 'DISPLAY QMGR SSLFIPS' | runmqsc"})

	// Search the console output for expected values
	if !strings.Contains(sslFIPSOutput, "SSLFIPS(NO)") {
		t.Errorf("Expected SSLFIPS to be: 'NO' but it is not; got \"%v\"", sslFIPSOutput)
	}

	// Stop the container cleanly
	stopContainer(t, cli, ID)
}
