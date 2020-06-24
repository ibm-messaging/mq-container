/*
© Copyright IBM Corporation 2017, 2020

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
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/ibm-messaging/mq-container/internal/command"
	"github.com/ibm-messaging/mq-container/pkg/logger"
	"github.com/ibm-messaging/mq-container/pkg/mqini"
)

// var debug = false
var log *logger.Logger

var collectDiagOnFail = false

func logTerminationf(format string, args ...interface{}) {
	logTermination(fmt.Sprintf(format, args...))
}

func logTermination(args ...interface{}) {
	msg := fmt.Sprint(args...)
	// Write the message to the termination log.  This is not the default place
	// that Kubernetes will look for termination information.
	log.Debugf("Writing termination message: %v", msg)
	err := ioutil.WriteFile("/run/termination-log", []byte(msg), 0660)
	if err != nil {
		log.Debug(err)
	}
	log.Error(msg)

	if collectDiagOnFail {
		logDiagnostics()
	}
}

func getLogFormat() string {
	return os.Getenv("LOG_FORMAT")
}

// formatBasic formats a log message parsed from JSON, as "basic" text
func formatBasic(obj map[string]interface{}) string {
	// Emulate the MQ "MessageDetail=Extended" option, by appending inserts to the message
	// This is important for certain messages, where key details are only available in the extended message content
	inserts := make([]string, 0)
	for k, v := range obj {
		if strings.HasPrefix(k, "ibm_commentInsert") {
			inserts = append(inserts, fmt.Sprintf("%s(%v)", strings.Replace(k, "ibm_comment", "Comment", 1), obj[k]))
		} else if strings.HasPrefix(k, "ibm_arithInsert") {
			if v.(float64) != 0 {
				inserts = append(inserts, fmt.Sprintf("%s(%v)", strings.Replace(k, "ibm_arith", "Arith", 1), obj[k]))
			}
		}
	}
	sort.Strings(inserts)
	if len(inserts) > 0 {
		return fmt.Sprintf("%s %s [%v]\n", obj["ibm_datetime"], obj["message"], strings.Join(inserts, ", "))
	}
	return fmt.Sprintf("%s %s\n", obj["ibm_datetime"], obj["message"])
}

// mirrorSystemErrorLogs starts a goroutine to mirror the contents of the MQ system error logs
func mirrorSystemErrorLogs(ctx context.Context, wg *sync.WaitGroup, mf mirrorFunc) (chan error, error) {
	// Always use the JSON log as the source
	return mirrorLog(ctx, wg, "/var/mqm/errors/AMQERR01.json", false, mf, false)
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
	return mirrorLog(ctx, wg, f, fromStart, mf, true)
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
		return func(msg string, isQMLog bool) bool {
			obj, err := processLogMessage(msg)
			if err == nil && isQMLog && filterQMLogMessage(obj) {
				return false
			}
			if err != nil {
				log.Printf("Failed to unmarshall JSON - %v", msg)
			} else {
				fmt.Println(msg)
			}
			return true
		}, nil
	case "basic":
		log, err = logger.NewLogger(os.Stderr, d, false, name)
		if err != nil {
			return nil, err
		}
		return func(msg string, isQMLog bool) bool {
			// Parse the JSON message, and print a simplified version
			obj, err := processLogMessage(msg)
			if err == nil && isQMLog && filterQMLogMessage(obj) {
				return false
			}
			if err != nil {
				log.Printf("Failed to unmarshall JSON - %v", err)
			} else {
				fmt.Printf(formatBasic(obj))
				// fmt.Printf(formatSimple(obj["ibm_datetime"].(string), obj["message"].(string)))
			}
			return true
		}, nil
	default:
		log, err = logger.NewLogger(os.Stdout, d, false, name)
		if err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("invalid value for LOG_FORMAT: %v", f)
	}
}

func processLogMessage(msg string) (map[string]interface{}, error) {
	var obj map[string]interface{}
	err := json.Unmarshal([]byte(msg), &obj)
	return obj, err
}

func filterQMLogMessage(obj map[string]interface{}) bool {
	hostname, err := os.Hostname()
	if os.Getenv("MQ_MULTI_INSTANCE") == "true" && err == nil && !strings.Contains(obj["host"].(string), hostname) {
		return true
	}
	return false
}

func logDiagnostics() {
	if getDebug() {
		log.Debug("--- Start Diagnostics ---")

		// show the directory ownership/permissions
		// #nosec G104
		out, _, _ := command.Run("ls", "-l", "/mnt/")
		log.Debugf("/mnt/:\n%s", out)
		// #nosec G104
		out, _, _ = command.Run("ls", "-l", "/mnt/mqm")
		log.Debugf("/mnt/mqm:\n%s", out)
		// #nosec G104
		out, _, _ = command.Run("ls", "-l", "/mnt/mqm/data")
		log.Debugf("/mnt/mqm/data:\n%s", out)
		// #nosec G104
		out, _, _ = command.Run("ls", "-l", "/mnt/mqm-log/log")
		log.Debugf("/mnt/mqm-log/log:\n%s", out)
		// #nosec G104
		out, _, _ = command.Run("ls", "-l", "/mnt/mqm-data/qmgrs")
		log.Debugf("/mnt/mqm-data/qmgrs:\n%s", out)
		// #nosec G104
		out, _, _ = command.Run("ls", "-l", "/var/mqm")
		log.Debugf("/var/mqm:\n%s", out)
		// #nosec G104
		out, _, _ = command.Run("ls", "-l", "/var/mqm/errors")
		log.Debugf("/var/mqm/errors:\n%s", out)
		// #nosec G104
		out, _, _ = command.Run("ls", "-l", "/etc/mqm")
		log.Debugf("/etc/mqm:\n%s", out)

		// Print out summary of any FDCs
		// #nosec G204
		cmd := exec.Command("/opt/mqm/bin/ffstsummary")
		cmd.Dir = "/var/mqm/errors"
		// #nosec G104
		outB, _ := cmd.CombinedOutput()
		log.Debugf("ffstsummary:\n%s", string(outB))

		log.Debug("---  End Diagnostics  ---")
	}
}
