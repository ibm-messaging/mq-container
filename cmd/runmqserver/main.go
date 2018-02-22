/*
© Copyright IBM Corporation 2017, 2018

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

// runmqserver initializes, creates and starts a queue manager, as PID 1 in a container
package main

import (
	"context"
	"errors"
	"os"
	"sync"

	"github.com/ibm-messaging/mq-container/internal/name"
	"github.com/ibm-messaging/mq-container/internal/ready"
)

func doMain() error {
	mf, err := configureLogger()
	if err != nil {
		logTermination(err)
		return err
	}
	err = ready.Clear()
	if err != nil {
		logTermination(err)
		return err
	}
	name, err := name.GetQueueManagerName()
	if err != nil {
		logTermination(err)
		return err
	}
	accepted, err := checkLicense()
	if err != nil {
		logTerminationf("Error checking license acceptance: %v", err)
		return err
	}
	if !accepted {
		err = errors.New("License not accepted")
		logTermination(err)
		return err
	}
	log.Printf("Using queue manager name: %v", name)

	// Start signal handler
	signalControl := signalHandler(name)

	logConfig()
	err = createVolume("/mnt/mqm")
	if err != nil {
		logTermination(err)
		return err
	}
	err = createDirStructure()
	if err != nil {
		return err
	}
	newQM, err := createQueueManager(name)
	if err != nil {
		logTermination(err)
		return err
	}
	var wg sync.WaitGroup
	defer func() {
		log.Debug("Waiting for log mirroring to complete")
		wg.Wait()
	}()
	ctx, cancelMirror := context.WithCancel(context.Background())
	defer func() {
		log.Debug("Cancel log mirroring")
		cancelMirror()
	}()
	// TODO: Use the error channel
	_, err = mirrorLogs(ctx, &wg, name, newQM, mf)
	if err != nil {
		logTermination(err)
		return err
	}
	err = updateCommandLevel()
	if err != nil {
		logTermination(err)
		return err
	}
	err = startQueueManager()
	if err != nil {
		logTermination(err)
		return err
	}
	configureQueueManager()
	// Start reaping zombies from now on.
	// Start this here, so that we don't reap any sub-processes created
	// by this process (e.g. for crtmqm or strmqm)
	signalControl <- startReaping
	// Reap zombies now, just in case we've already got some
	signalControl <- reapNow
	// Write a file to indicate that chkmqready should now work as normal
	ready.Set()
	// Wait for terminate signal
	<-signalControl
	return nil
}

var osExit = os.Exit

func main() {
	err := doMain()
	if err != nil {
		osExit(1)
	}
}
