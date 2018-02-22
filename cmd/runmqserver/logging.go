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

	"github.com/ibm-messaging/mq-container/internal/mqini"
	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
)

var debug = false

// timestampFormat matches the format used by MQ messages (includes milliseconds)
const timestampFormat string = "2006-01-02T15:04:05.000Z07:00"

type simpleTextFormatter struct {
}

func (f *simpleTextFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	// If debugging, and a prefix, but only for this formatter.
	if entry.Level == logrus.DebugLevel {
		entry.Message = "DEBUG: " + entry.Message
	}
	// Use a simple format, with a timestamp
	return []byte(formatSimple(entry.Time.Format(timestampFormat), entry.Message)), nil
}

func logDebug(args ...interface{}) {
	if debug {
		log.Debug(args)
	}
}

func logDebugf(format string, args ...interface{}) {
	if debug {
		log.Debugf(format, args...)
	}
}

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

func mirrorLogs(ctx context.Context, wg *sync.WaitGroup, name string, fromStart bool, mf mirrorFunc) (chan error, error) {
	// Always use the JSON log as the source
	// Put the queue manager name in quotes to handle cases like name=..
	qm, err := mqini.GetQueueManager(name)
	if err != nil {
		logDebug(err)
		return nil, err
	}
	f := filepath.Join(mqini.GetErrorLogDirectory(qm), "AMQERR01.json")
	return mirrorLog(ctx, wg, f, fromStart, mf)
}

func configureDebugLogger() {
	debugEnv, ok := os.LookupEnv("DEBUG")
	if ok && (debugEnv == "true" || debugEnv == "1") {
		debug = true
		logrus.SetLevel(logrus.DebugLevel)
		logDebug("Debug mode enabled")
	}
}

func configureLogger() (mirrorFunc, error) {
	// Set the simple formatter by default
	log.SetFormatter(new(simpleTextFormatter))
	f := getLogFormat()
	switch f {
	case "json":
		formatter := logrus.JSONFormatter{
			FieldMap: logrus.FieldMap{
				logrus.FieldKeyMsg:   "message",
				logrus.FieldKeyLevel: "ibm_level",
				logrus.FieldKeyTime:  "ibm_datetime",
			},
			TimestampFormat: timestampFormat,
		}
		logrus.SetFormatter(&formatter)
		return func(msg string) {
			fmt.Println(msg)
		}, nil
	case "simple":
		return func(msg string) {
			// Parse the JSON message, and print a simplified version
			var obj map[string]interface{}
			json.Unmarshal([]byte(msg), &obj)
			fmt.Printf(formatSimple(obj["ibm_datetime"].(string), obj["message"].(string)))
		}, nil
	default:
		return nil, fmt.Errorf("invalid value for LOG_FORMAT: %v", f)
	}
}
