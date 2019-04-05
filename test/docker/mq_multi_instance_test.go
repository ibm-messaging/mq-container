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
	"time"

	"github.com/docker/docker/client"
)



// TestMultiInstanceStartup creates 2 containers in a multi instance queue manager configuration
// and starts/stop them checking we always have an active and standby
func TestMultiInstanceStartStop(t *testing.T) {
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
	killContainer(t, cli, qm1a, "SIGTERM")
	time.Sleep(2 * time.Second)

	if status := getQueueManagerStatus(t, cli, qm1b, "QM1"); strings.Compare(status, "Running") != 0 {
		t.Fatalf("Expected QM1 to be running as active queue manager, dspmq returned status of %v", status)
	}

	startContainer(t, cli, qm1a)
	waitForReady(t, cli, qm1a)

	if status := getQueueManagerStatus(t, cli, qm1a, "QM1"); strings.Compare(status, "Running as standby") != 0 {
		t.Fatalf("Expected QM1 to be running as standby queue manager, dspmq returned status of %v", status)
	}

}

// TestMultiInstanceContainerStop starts 2 containers in a multi instance queue manager configuration,	
// stops the active queue manager, then checks to ensure the backup queue manager becomes active
func TestMultiInstanceContainerStop(t *testing.T) {
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

// TestMultiInstanceRace starts 2 containers in separate goroutines in a multi instance queue manager 
// configuration, then checks to ensure that both an active and standby queue manager have been started
// func TestMultiInstanceRace(t *testing.T) {
// 	t.Parallel()
// 	cli, err := client.NewEnvClient()
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	qmsharedlogs := createVolume(t, cli, "qmsharedlogs")
// 	defer removeVolume(t, cli, qmsharedlogs.Name)
// 	qmshareddata := createVolume(t, cli, "qmshareddata")
// 	defer removeVolume(t, cli, qmshareddata.Name)

// 	qmsChannel := make(chan QMChan)

// 	go singleInstance(t, cli, qmsharedlogs.Name, qmshareddata.Name, qmsChannel)
// 	// time.Sleep(1 * time.Second)
// 	go singleInstance(t, cli, qmsharedlogs.Name, qmshareddata.Name, qmsChannel)

// 	qm1a := <- qmsChannel
// 	if qm1a.Error != nil {
// 		t.Fatal(qm1a.Error)
// 	}

// 	qm1b := <- qmsChannel
// 	if qm1b.Error != nil {
// 		t.Fatal(qm1b.Error)
// 	}

// 	close(qmsChannel)

// 	qm1aId, qm1aData := qm1a.QMId, qm1a.QMData
// 	qm1bId, qm1bData := qm1b.QMId, qm1b.QMData

// 	defer removeVolume(t, cli, qm1aData)
// 	defer removeVolume(t, cli, qm1bData)
// 	defer cleanContainer(t, cli, qm1aId)
// 	defer cleanContainer(t, cli, qm1bId)

// 	err, _, _ = getActiveStandbyQueueManager(t, cli, qm1aId, qm1bId)
// 	if err != nil {
// 		t.Fatal(err)
// 	}
// }


