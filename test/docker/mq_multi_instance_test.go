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

	if status := getQueueManagerStatus(t, cli, qm1a, "QM1"); strings.Compare(status, "Running") != 0 {
		t.Fatalf("Expected QM1 to be running as active queue manager, dspmq returned status of %v", status)
	} 
	if status := getQueueManagerStatus(t, cli, qm1b, "QM1"); strings.Compare(status, "Running as standby") != 0 {
		t.Fatalf("Expected QM1 to be running as standby queue manager, dspmq returned status of %v", status)
	}
}

// TestMultiInstanceStop starts 2 containers in a multi instance queue manager configuration,	
// stops the active queue manager, then checks to ensure the backup queue manager becomes active
func TestMultiInstanceStop(t *testing.T) {
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

	if status := getQueueManagerStatus(t, cli, qm1a, "QM1"); strings.Compare(status, "Running") != 0 {
		t.Fatalf("Expected QM1 to be running as active queue manager, dspmq returned status of %v", status)
	} 
	if status := getQueueManagerStatus(t, cli, qm1b, "QM1"); strings.Compare(status, "Running as standby") != 0 {
		t.Fatalf("Expected QM1 to be running as standby queue manager, dspmq returned status of %v", status)
	}

	stopContainer(t, cli, qm1a)

	if status := getQueueManagerStatus(t, cli, qm1b, "QM1"); strings.Compare(status, "Running") != 0 {
		t.Fatalf("Expected QM1 to be running as standby queue manager, dspmq returned status of %v", status)
	}

}
