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
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"

	"github.com/ibm-messaging/mq-container/internal/logger"
	"github.com/ibm-messaging/mq-container/internal/mqini"
)

// var debug = false
var log *logger.Logger

func logTerminationf(format string, args ...interface{}) {
	logTermination(fmt.Sprintf(format, args))
}

func logTermination(args ...interface{}) {
	msg := fmt.Sprint(args)
	// Write the message to the termination log.  This is the default place
	// that Kubernetes will look for termination information.
	log.Debugf("Writing termination message: %v", msg)
	err := ioutil.WriteFile("/dev/termination-log", []byte(msg), 0660)
	if err != nil {
		log.Debug(err)
	}
	log.Error(msg)
}

func getLogFormat() string {
	return os.Getenv("LOG_FORMAT")
}

func formatSimple(datetime string, message string) string {
	return fmt.Sprintf("%v %v\n", datetime, message)
}

// mirrorSystemErrorLogs starts a goroutine to mirror the contents of the MQ system error logs
func mirrorSystemErrorLogs(ctx context.Context, wg *sync.WaitGroup, mf mirrorFunc) (chan error, error) {
	// Always use the JSON log as the source
	return mirrorLog(ctx, wg, "/var/mqm/errors/AMQERR01.json", false, mf)
}

// mirrorQueueManagerErrorLogs starts a goroutine to mirror the contents of the MQ queue manager error logs
func mirrorQueueManagerErrorLogs(ctx context.Context, wg *sync.WaitGroup, name string, fromStart bool, mf mirrorFunc) (chan error, error) {
	// Always use the JSON log as the source
	qm, err := mqini.GetQueueManager(name)
	if err != nil {
		log.Debug(err)
		return nil, err
	}
	f := filepath.Join(mqini.GetErrorLogDirectory(qm), "AMQERR01.json")
	return mirrorLog(ctx, wg, f, fromStart, mf)
}

func getDebug() bool {
	debug := os.Getenv("DEBUG")
	if debug == "true" || debug == "1" {
		return true
	}
	return false
}

func configureLogger(name string) (mirrorFunc, error) {
	var err error
	f := getLogFormat()
	d := getDebug()
	switch f {
	case "json":
		log, err = logger.NewLogger(os.Stderr, d, true, name)
		if err != nil {
			return nil, err
		}
		return log.LogDirect, nil
	case "basic":
		log, err = logger.NewLogger(os.Stderr, d, false, name)
		if err != nil {
			return nil, err
		}
		return func(msg string) {
			// Parse the JSON message, and print a simplified version
			var obj map[string]interface{}
			json.Unmarshal([]byte(msg), &obj)
			fmt.Printf(formatSimple(obj["ibm_datetime"].(string), obj["message"].(string)))
		}, nil
	default:
		log, err = logger.NewLogger(os.Stdout, d, false, name)
		if err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("invalid value for LOG_FORMAT: %v", f)
	}
}
