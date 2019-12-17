/*
© Copyright IBM Corporation 2017, 2019

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
	"flag"
	"os"
	"sync"

	"github.com/ibm-messaging/mq-container/internal/metrics"
	"github.com/ibm-messaging/mq-container/internal/ready"
	"github.com/ibm-messaging/mq-container/internal/tls"
	"github.com/ibm-messaging/mq-container/pkg/containerruntimelogger"
	"github.com/ibm-messaging/mq-container/pkg/name"
)

func doMain() error {
	var initFlag = flag.Bool("i", false, "initialize volume only, then exit")
	var infoFlag = flag.Bool("info", false, "Display debug info, then exit")
	var noLogRuntimeFlag = flag.Bool("nologruntime", false, "used when running this program from another program, to control log output")
	var devFlag = flag.Bool("dev", false, "used when running this program from runmqdevserver to control how TLS is configured")
	flag.Parse()

	name, nameErr := name.GetQueueManagerName()
	mf, err := configureLogger(name)
	if err != nil {
		logTermination(err)
		return err
	}

	// Check whether they only want debug info
	if *infoFlag {
		logVersionInfo()
		err = containerruntimelogger.LogContainerDetails(log)
		if err != nil {
			log.Printf("Error displaying container details: %v", err)
		}
		return nil
	}

	err = verifySingleProcess()
	if err != nil {
		// We don't do the normal termination here as it would create a termination file.
		log.Error(err)
		return err
	}

	if nameErr != nil {
		logTermination(err)
		return err
	}
	err = ready.Clear()
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
	// Enable diagnostic collecting on failure
	collectDiagOnFail = true

	if *noLogRuntimeFlag == false {
		err = containerruntimelogger.LogContainerDetails(log)
		if err != nil {
			logTermination(err)
			return err
		}
	}

	err = createVolume("/mnt/mqm/data")
	if err != nil {
		logTermination(err)
		return err
	}
	err = createVolume("/mnt/mqm-log/log")
	if err != nil {
		logTermination(err)
		return err
	}
	err = createVolume("/mnt/mqm-data/qmgrs")
	if err != nil {
		logTermination(err)
		return err
	}

	err = createDirStructure()
	if err != nil {
		logTermination(err)
		return err
	}

	// If init flag is set, exit now
	if *initFlag {
		return nil
	}

	// Print out versioning information
	logVersionInfo()

	keyLabel, cmsKeystore, p12Truststore, err := tls.ConfigureTLSKeystores()
	if err != nil {
		logTermination(err)
		return err
	}

	err = tls.ConfigureTLS(keyLabel, cmsKeystore, *devFlag, log)
	if err != nil {
		logTermination(err)
		return err
	}

	err = postInit(name, keyLabel, p12Truststore)
	if err != nil {
		logTermination(err)
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
	_, err = mirrorSystemErrorLogs(ctx, &wg, mf)
	if err != nil {
		logTermination(err)
		return err
	}
	_, err = mirrorQueueManagerErrorLogs(ctx, &wg, name, newQM, mf)
	if err != nil {
		logTermination(err)
		return err
	}
	err = updateCommandLevel()
	if err != nil {
		logTermination(err)
		return err
	}

	err = startQueueManager(name)
	if err != nil {
		logTermination(err)
		return err
	}
	if standby, _ := ready.IsRunningAsStandbyQM(name); !standby {
		err = configureQueueManager()
		if err != nil {
			logTermination(err)
			return err
		}
	}

	enableMetrics := os.Getenv("MQ_ENABLE_METRICS")
	if enableMetrics == "true" || enableMetrics == "1" {
		go metrics.GatherMetrics(name, log)
	} else {
		log.Println("Metrics are disabled")
	}

	// Start reaping zombies from now on.
	// Start this here, so that we don't reap any sub-processes created
	// by this process (e.g. for crtmqm or strmqm)
	signalControl <- startReaping
	// Reap zombies now, just in case we've already got some
	signalControl <- reapNow
	// Write a file to indicate that chkmqready should now work as normal
	err = ready.Set()
	if err != nil {
		logTermination(err)
		return err
	}
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
