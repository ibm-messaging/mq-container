/*
© Copyright IBM Corporation 2017, 2024

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
	"path"
	"sync"

	"github.com/ibm-messaging/mq-container/internal/copy"
	"github.com/ibm-messaging/mq-container/internal/fips"
	"github.com/ibm-messaging/mq-container/internal/ha"
	"github.com/ibm-messaging/mq-container/internal/metrics"
	"github.com/ibm-messaging/mq-container/internal/ready"
	"github.com/ibm-messaging/mq-container/internal/simpleauth"
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

	// Create a startup context to be used by the signalHandler to ensure the final reap of zombie processes only occurs after all startup processes are spawned
	startupCtx, markStartupComplete := context.WithCancel(context.Background())
	var startupMarkedComplete bool
	// If the main thread returns before completing startup, cancel the startup context to unblock the signalHandler
	defer func() {
		if !startupMarkedComplete {
			markStartupComplete()
		}
	}()
	// Start signal handler
	signalControl := signalHandler(name, startupCtx)
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

	// Delete contents of /run/scratch directory.
	err = cleanVolume("/run/scratch")
	if err != nil {
		logTermination(err)
		return err
	}

	// Delete contents of /run/mqm directory.
	err = cleanVolume("/run/mqm")
	if err != nil {
		logTermination(err)
		return err
	}

	// Create ephemeral volumes
	err = createVolume("/run/scratch/runmqserver")
	if err != nil {
		logTermination(err)
		return err
	}

	// Queue manager i.e crtmqm command creates socket and
	// others files in /run/mqm directory.
	err = createVolume("/run/mqm")
	if err != nil {
		logTermination(err)
		return err
	}

	// Initialise 15-tls.mqsc file on ephemeral volume
	// #nosec G306 - its a read by owner/s group, and pose no harm.
	err = os.WriteFile("/run/15-tls.mqsc", []byte(""), 0660)
	if err != nil {
		logTermination(err)
		return err
	}

	// Initialise native-ha ini files file on ephemeral volume
	nativeHAINIs := []string{
		"10-native-ha.ini",
		"10-native-ha-instance.ini",
		"10-native-ha-keystore.ini",
	}
	for _, iniFile := range nativeHAINIs {
		// #nosec G306 - its a read by owner/s group, and pose no harm.
		err = os.WriteFile(path.Join("/run", iniFile), []byte(""), 0660)
		if err != nil {
			logTermination(err)
			return err
		}
	}

	// Copy default mqwebcontainer.xml file to ephemeral volume
	if *devFlag && os.Getenv("MQ_DEV") == "true" {
		err = copy.CopyFile("/etc/mqm/web/installations/Installation1/servers/mqweb/mqwebcontainer.xml.dev", "/run/mqwebcontainer.xml")
		if err != nil {
			logTermination(err)
			return err
		}
	} else {
		err = copy.CopyFile("/etc/mqm/web/installations/Installation1/servers/mqweb/mqwebcontainer.xml.default", "/run/mqwebcontainer.xml")
		if err != nil {
			logTermination(err)
			return err
		}
	}

	// Copy default tls.xml file to ephemeral volume
	err = copy.CopyFile("/etc/mqm/web/installations/Installation1/servers/mqweb/tls.xml.default", "/run/tls.xml")
	if err != nil {
		logTermination(err)
		return err
	}

	// Copy default jvm.options file to ephemeral volume
	err = copy.CopyFile("/etc/mqm/web/installations/Installation1/servers/mqweb/configDropins/defaults/jvm.options.default", "/run/jvm.options")
	if err != nil {
		logTermination(err)
		return err
	}

	enableTraceCrtmqdir := os.Getenv("MQ_ENABLE_TRACE_CRTMQDIR")
	if enableTraceCrtmqdir == "true" || enableTraceCrtmqdir == "1" {
		err = startMQTrace()
		if err != nil {
			logTermination(err)
			return err
		}
	}

	err = createDirStructure()
	if err != nil {
		logTermination(err)
		return err
	}

	if enableTraceCrtmqdir == "true" || enableTraceCrtmqdir == "1" {
		err = endMQTrace()
		if err != nil {
			logTermination(err)
			return err
		}
	}

	// If init flag is set, exit now
	if *initFlag {
		return nil
	}

	// Print out versioning information
	logVersionInfo()

	// Determine FIPS compliance level
	fips.ProcessFIPSType(log)

	keyLabel, defaultCmsKeystore, defaultP12Truststore, err := tls.ConfigureDefaultTLSKeystores(log)
	if err != nil {
		logTermination(err)
		return err
	}

	err = tls.ConfigureTLS(keyLabel, defaultCmsKeystore, *devFlag, log)
	if err != nil {
		logTermination(err)
		return err
	}

	//Validate MQ_LOG_CONSOLE_SOURCE variable
	if !isLogConsoleSourceValid() {
		log.Println("One or more invalid value is provided for MQ_LOGGING_CONSOLE_SOURCE. Allowed values are 'qmgr','web' and 'mqsc' in csv format")
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

	//For mirroring web server logs if source variable is set
	if checkLogSourceForMirroring("web") {
		// Always log from the end of the web server messages.log, because the log rotation should happen as soon as the web server starts
		_, err = mirrorWebServerLogs(ctx, &wg, name, false, mf)
		if err != nil {
			logTermination(err)
			return err
		}
	}

	err = postInit(name, keyLabel, defaultP12Truststore)
	if err != nil {
		logTermination(err)
		return err
	}

	if os.Getenv("MQ_NATIVE_HA") == "true" {
		err = ha.ConfigureNativeHA(log)
		if err != nil {
			logTermination(err)
			return err
		}
	}

	// Post FIPS initialization processing
	fips.PostInit(log)

	enableTraceCrtmqm := os.Getenv("MQ_ENABLE_TRACE_CRTMQM")
	if enableTraceCrtmqm == "true" || enableTraceCrtmqm == "1" {
		err = startMQTrace()
		if err != nil {
			logTermination(err)
			return err
		}
	}

	newQM, err := createQueueManager(name, *devFlag)
	if err != nil {
		logTermination(err)
		return err
	}

	if enableTraceCrtmqm == "true" || enableTraceCrtmqm == "1" {
		err = endMQTrace()
		if err != nil {
			logTermination(err)
			return err
		}
	}

	//For mirroring mq system logs and qm logs, if environment variable is set
	if checkLogSourceForMirroring("qmgr") {
		//Mirror MQ system logs
		_, err = mirrorSystemErrorLogs(ctx, &wg, mf)
		if err != nil {
			logTermination(err)
			return err
		}

		//Mirror queue manager logs
		_, err = mirrorQueueManagerErrorLogs(ctx, &wg, name, newQM, mf)
		if err != nil {
			logTermination(err)
			return err
		}
	}

	if *devFlag && simpleauth.IsEnabled() {
		_, err = mirrorMQSimpleAuthLogs(ctx, &wg, name, newQM, mf)
		if err != nil {
			logTermination(err)
			return err
		}
	}

	err = updateCommandLevel()
	if err != nil {
		logTermination(err)
		return err
	}

	enableTraceStrmqm := os.Getenv("MQ_ENABLE_TRACE_STRMQM")
	if enableTraceStrmqm == "true" || enableTraceStrmqm == "1" {
		err = startMQTrace()
		if err != nil {
			logTermination(err)
			return err
		}
	}

	// This is a developer image only change
	// This workaround should be removed and handled via <crtmqm -ii>, when inimerge is ready to handle stanza ordering
	if *devFlag && simpleauth.IsEnabled() {
		err = updateQMini(name)
		if err != nil {
			logTermination(err)
			return err
		}
	}

	err = startQueueManager(name)
	if err != nil {
		logTermination(err)
		return err
	}

	//If the queue manager has started successfully, reflect mqsc logs when enabled
	if checkLogSourceForMirroring("mqsc") {
		_, err = mirrorMQSCLogs(ctx, &wg, name, mf)
		if err != nil {
			logTermination(err)
			return err
		}
	}

	if enableTraceStrmqm == "true" || enableTraceStrmqm == "1" {
		err = endMQTrace()
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

	startupMarkedComplete = true
	markStartupComplete()

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
