/*
© Copyright IBM Corporation 2018, 2023

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
	"os"
	"syscall"

	"github.com/ibm-messaging/mq-container/internal/htpasswd"
	"github.com/ibm-messaging/mq-container/pkg/containerruntimelogger"
	"github.com/ibm-messaging/mq-container/pkg/logger"
	"github.com/ibm-messaging/mq-container/pkg/name"
)

var log *logger.Logger

func getLogFormat() string {
	return os.Getenv("LOG_FORMAT")
}

func getDebug() bool {
	debug := os.Getenv("DEBUG")
	if debug == "true" || debug == "1" {
		return true
	}
	return false
}

func configureLogger() error {
	var err error
	f := getLogFormat()
	d := getDebug()
	n, err := name.GetQueueManagerName()
	if err != nil {
		return err
	}
	switch f {
	case "json":
		log, err = logger.NewLogger(os.Stderr, d, true, n)
		if err != nil {
			return err
		}
	case "basic":
		log, err = logger.NewLogger(os.Stderr, d, false, n)
		if err != nil {
			return err
		}
	default:
		log, err = logger.NewLogger(os.Stdout, d, false, n)
		return fmt.Errorf("invalid value for LOG_FORMAT: %v", f)
	}
	return nil
}

func logTerminationf(format string, args ...interface{}) {
	logTermination(fmt.Sprintf(format, args...))
}

// TODO: Duplicated code
func logTermination(args ...interface{}) {
	msg := fmt.Sprint(args...)
	// Write the message to the termination log.  This is not the default place
	// that Kubernetes will look for termination information.
	log.Debugf("Writing termination message: %v", msg)
	// #nosec G306 - its a read by owner/s group, and pose no harm.
	err := os.WriteFile("/run/termination-log", []byte(msg), 0660)
	if err != nil {
		log.Debug(err)
	}
	log.Error(msg)
}

func doMain() error {
	err := configureLogger()
	if err != nil {
		logTermination(err)
		return err
	}

	err = containerruntimelogger.LogContainerDetails(log)
	if err != nil {
		logTermination(err)
		return err
	}

	adminPassword, set := os.LookupEnv("MQ_ADMIN_PASSWORD")
	if !set {
		adminPassword = "passw0rd"
		err = os.Setenv("MQ_ADMIN_PASSWORD", adminPassword)
		if err != nil {
			logTerminationf("Error setting admin password variable: %v", err)
			return err
		}
	}
	err = htpasswd.SetPassword("admin", adminPassword, false)
	if err != nil {
		logTerminationf("Error setting admin password: %v", err)
		return err
	}

	appPassword, set := os.LookupEnv("MQ_APP_PASSWORD")
	if set {
		err = htpasswd.SetPassword("app", appPassword, false)
		if err != nil {
			logTerminationf("Error setting app password: %v", err)
			return err
		}
	}

	err = updateMQSC(set)
	if err != nil {
		logTerminationf("Error updating MQSC: %v", err)
		return err
	}

	return nil
}

var osExit = os.Exit

func main() {
	err := doMain()
	if err != nil {
		osExit(1)
	} else {
		// Replace this process with runmqserver
		// #nosec G204
		err = syscall.Exec("/usr/local/bin/runmqserver", []string{"runmqserver", "-nologruntime", "-dev"}, os.Environ())
		if err != nil {
			log.Errorf("Error replacing this process with runmqserver: %v", err)
		}
	}
}
