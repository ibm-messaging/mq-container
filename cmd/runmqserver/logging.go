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

func logDebug(msg string) {
	if debug {
		log.Debugln(msg)
	}
}

func logDebugf(format string, args ...interface{}) {
	if debug {
		log.Debugf(format, args...)
	}
}

func jsonLogs() bool {
	e := os.Getenv("MQ_ALPHA_JSON_LOGS")
	if e == "true" || e == "1" {
		return true
	}
	return false
}

func mirrorToStdout(msg string) {
	fmt.Println(msg)
}

func formatSimple(datetime string, message string) string {
	return fmt.Sprintf("%v %v\n", datetime, message)
}

func mirrorLogs(ctx context.Context, wg *sync.WaitGroup, name string, fromStart bool) (chan error, error) {
	// Always use the JSON log as the source
	// Put the queue manager name in quotes to handle cases like name=..
	qm, err := mqini.GetQueueManager(name)
	if err != nil {
		logDebugf("%v", err)
		return nil, err
	}
	f := filepath.Join(mqini.GetErrorLogDirectory(qm), "AMQERR01.json")
	// f := fmt.Sprintf("/var/mqm/qmgrs/\"%v\"/errors/AMQERR01.json", name)
	if jsonLogs() {
		return mirrorLog(ctx, wg, f, fromStart, mirrorToStdout)
	}
	return mirrorLog(ctx, wg, f, fromStart, func(msg string) {
		// Parse the JSON message, and print a simplified version
		var obj map[string]interface{}
		json.Unmarshal([]byte(msg), &obj)
		fmt.Printf(formatSimple(obj["ibm_datetime"].(string), obj["message"].(string)))
	})
}

func configureDebugLogger() {
	debugEnv, ok := os.LookupEnv("DEBUG")
	if ok && (debugEnv == "true" || debugEnv == "1") {
		debug = true
		logrus.SetLevel(logrus.DebugLevel)
		logDebug("Debug mode enabled")
	}
}

func configureLogger() {
	if jsonLogs() {
		formatter := logrus.JSONFormatter{
			FieldMap: logrus.FieldMap{
				logrus.FieldKeyMsg:   "message",
				logrus.FieldKeyLevel: "ibm_level",
				logrus.FieldKeyTime:  "ibm_datetime",
			},
			TimestampFormat: timestampFormat,
		}
		logrus.SetFormatter(&formatter)
	} else {
		log.SetFormatter(new(simpleTextFormatter))
	}
}
