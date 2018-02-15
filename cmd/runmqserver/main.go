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
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"

	"github.com/ibm-messaging/mq-container/internal/command"
	"github.com/ibm-messaging/mq-container/internal/mqini"
	"github.com/ibm-messaging/mq-container/internal/name"
	"github.com/ibm-messaging/mq-container/internal/ready"
)

var debug = false

func logDebug(msg string) {
	if debug {
		log.Debug(msg)
	}
}

func logDebugf(format string, args ...interface{}) {
	if debug {
		log.Debugf(format, args...)
	}
}

// createDirStructure creates the default MQ directory structure under /var/mqm
func createDirStructure() error {
	out, _, err := command.Run("/opt/mqm/bin/crtmqdir", "-f", "-s")
	if err != nil {
		log.Printf("Error creating directory structure: %v\n", string(out))
		return err
	}
	log.Println("Created directory structure under /var/mqm")
	return nil
}

// createQueueManager creates a queue manager, if it doesn't already exist.
// It returns true if one was created, or false if one already existed
func createQueueManager(name string) (bool, error) {
	log.Printf("Creating queue manager %v", name)
	out, rc, err := command.Run("crtmqm", "-q", "-p", "1414", name)
	if err != nil {
		// 8=Queue manager exists, which is fine
		if rc == 8 {
			log.Printf("Detected existing queue manager %v", name)
			return false, nil
		}
		log.Printf("crtmqm returned %v", rc)
		log.Println(string(out))
		return false, err
	}
	return true, nil
}

func updateCommandLevel() error {
	level, ok := os.LookupEnv("MQ_CMDLEVEL")
	if ok && level != "" {
		log.Printf("Setting CMDLEVEL to %v", level)
		out, rc, err := command.Run("strmqm", "-e", "CMDLEVEL="+level)
		if err != nil {
			log.Printf("Error %v setting CMDLEVEL: %v", rc, string(out))
			return err
		}
	}
	return nil
}

func startQueueManager() error {
	log.Println("Starting queue manager")
	out, rc, err := command.Run("strmqm")
	if err != nil {
		log.Printf("Error %v starting queue manager: %v", rc, string(out))
		return err
	}
	log.Println("Started queue manager")
	return nil
}

func configureQueueManager() error {
	const configDir string = "/etc/mqm"
	files, err := ioutil.ReadDir(configDir)
	if err != nil {
		log.Println(err)
		return err
	}

	for _, file := range files {
		if strings.HasSuffix(file.Name(), ".mqsc") {
			abs := filepath.Join(configDir, file.Name())
			cmd := exec.Command("runmqsc")
			stdin, err := cmd.StdinPipe()
			if err != nil {
				log.Println(err)
				return err
			}
			// Open the MQSC file for reading
			f, err := os.Open(abs)
			if err != nil {
				log.Printf("Error opening %v: %v", abs, err)
			}
			// Copy the contents to stdin of the runmqsc process
			_, err = io.Copy(stdin, f)
			if err != nil {
				log.Printf("Error reading %v: %v", abs, err)
			}
			f.Close()
			stdin.Close()
			// Run the command and wait for completion
			out, err := cmd.CombinedOutput()
			if err != nil {
				log.Println(err)
			}
			// Print the runmqsc output, adding tab characters to make it more readable as part of the log
			log.Printf("Output for \"runmqsc\" with %v:\n\t%v", abs, strings.Replace(string(out), "\n", "\n\t", -1))
		}
	}
	return nil
}

func stopQueueManager(name string) error {
	log.Println("Stopping queue manager")
	out, _, err := command.Run("endmqm", "-w", name)
	if err != nil {
		log.Printf("Error stopping queue manager: %v", string(out))
		return err
	}
	log.Println("Stopped queue manager")
	return nil
}

func jsonLogs() bool {
	e := os.Getenv("MQ_ALPHA_JSON_LOGS")
	if e == "true" || e == "1" {
		return true
	}
	return false
}

func mirrorLogs(name string, fromStart bool) (chan bool, error) {
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
		return mirrorLog(f, fromStart, func(msg string) {
			// Print the message straight to stdout
			fmt.Println(msg)
		})
	}
	return mirrorLog(f, fromStart, func(msg string) {
		// Parse the JSON message, and print a simplified version
		var obj map[string]interface{}
		json.Unmarshal([]byte(msg), &obj)
		fmt.Printf("%v %v\n", obj["ibm_datetime"], obj["message"])
	})
}

type simpleTextFormatter struct {
}

const timestampFormat string = "2006-01-02T15:04:05.000Z07:00"

func (f *simpleTextFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	// If debugging, and a prefix, but only for this formatter.
	if entry.Level == logrus.DebugLevel {
		entry.Message = "DEBUG: " + entry.Message
	}
	// Use a simple format, with a timestamp
	return []byte(fmt.Sprintf("%v %v\n", entry.Time.Format(timestampFormat), entry.Message)), nil
}

func configureLogger() {
	if jsonLogs() {
		formatter := logrus.JSONFormatter{
			FieldMap: logrus.FieldMap{
				logrus.FieldKeyMsg:   "message",
				logrus.FieldKeyLevel: "ibm_level",
				logrus.FieldKeyTime:  "ibm_datetime",
			},
			// Match time stamp format used by MQ messages (includes milliseconds)
			TimestampFormat: timestampFormat,
		}
		logrus.SetFormatter(&formatter)
	} else {
		log.SetFormatter(new(simpleTextFormatter))
	}
}

func doMain() error {
	configureLogger()
	err := ready.Clear()
	if err != nil {
		return err
	}
	debugEnv, ok := os.LookupEnv("DEBUG")
	if ok && (debugEnv == "true" || debugEnv == "1") {
		debug = true
		logrus.SetLevel(logrus.DebugLevel)
		logDebug("Debug mode enabled")
	}
	name, err := name.GetQueueManagerName()
	if err != nil {
		log.Println(err)
		return err
	}
	accepted, err := checkLicense()
	if err != nil {
		return err
	}
	if !accepted {
		return errors.New("License not accepted")
	}
	log.Printf("Using queue manager name: %v", name)

	// Start signal handler
	signalControl := signalHandler(name)

	logConfig()
	err = createVolume("/mnt/mqm")
	if err != nil {
		log.Println(err)
		return err
	}
	err = createDirStructure()
	if err != nil {
		return err
	}
	newQM, err := createQueueManager(name)
	if err != nil {
		return err
	}
	mirrorLifecycle, err := mirrorLogs(name, newQM)
	if err != nil {
		return err
	}
	err = updateCommandLevel()
	if err != nil {
		return err
	}
	err = startQueueManager()
	if err != nil {
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
	// Tell the mirroring goroutine to shutdown
	mirrorLifecycle <- true
	// Wait for the mirroring goroutine to finish cleanly
	<-mirrorLifecycle
	return nil
}

var osExit = os.Exit

func main() {
	err := doMain()
	if err != nil {
		osExit(1)
	}
}
