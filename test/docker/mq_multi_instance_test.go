/*
© Copyright IBM Corporation 2018, 2019

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
	"strings"

	"github.com/docker/docker/client"
)

// TestMultiInstanceStartup creates 2 containers in a multi instance queue manager configuration,	
// checks to ensure both active and standby queue managers are started
func TestMultiInstanceStartup(t *testing.T) {
	t.Parallel()
	cli, err := client.NewEnvClient()
	if err != nil {
		t.Fatal(err)
	}
	err, qm1a, qm1b, volumes := configureMultiInstance(t, cli)
	if err != nil {
		t.Fatal(err)
	}
	for _, volume := range volumes {
		defer removeVolume(t, cli, volume)
	}
	defer cleanContainer(t, cli, qm1a)
	defer cleanContainer(t, cli, qm1b)
	_, dspmqOut := execContainer(t, cli, qm1a, "mqm", []string{"bash", "-c", "dspmq", "-m", "QM1"})
	if strings.Contains(dspmqOut, "STATUS(Running)") == false {
		t.Fatalf("Expected QM1 to be running on active queue manager, dspmq returned %v", dspmqOut)
	}
	_, dspmqOut = execContainer(t, cli, qm1b, "mqm", []string{"bash", "-c", "dspmq", "-m", "QM1"})
	if strings.Contains(dspmqOut, "STATUS(Running as standby)") == false {
		t.Fatalf("Expected QM1 to be running as standby on standby queue manager, dspmq returned %v", dspmqOut)
	}
}
