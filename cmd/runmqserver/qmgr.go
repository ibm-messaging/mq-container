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
package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/ibm-messaging/mq-container/internal/command"
	"github.com/ibm-messaging/mq-container/internal/mqscredact"
)

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
	out, rc, err := command.Run("dspmqinf", name)
	if err != nil {
		// TODO : handle single instance queue manager with log & data volumes
		dataDir := filepath.Join("/var/mqm/qmgrs", name)
		if os.Getenv("MQ_MULTI_INSTANCE") == "true" {
			dataDir = filepath.Join("/mnt/mqm-data/data", name)
		}
		if _, err := os.Stat(filepath.Join(dataDir, "qm.ini")); err != nil {
			// TODO : tidy-up & test setting log & data when not mounted
			if os.Getenv("MQ_MULTI_INSTANCE") == "true" {
				log.Println("Creating active queue manager")
				out, rc, err = command.Run("crtmqm", "-q", "-p", "1414", "-ld", "/mnt/mqm-log/data", "-md", "/mnt/mqm-data/data", name)
			} else {
				out, rc, err = command.Run("crtmqm", "-q", "-p", "1414", name)
			}
			if err != nil {
				log.Printf("crtmqm returned %v : %v", rc, string(out))
				return false, err
			}
			// Return true or false?
			return true, nil
		} else {
			log.Println("Creating standby queue manager")
			qmName := fmt.Sprintf("Name=%v", name)
			qmDirectory := fmt.Sprintf("Directory=%v", name)
			qmPrefix := "Prefix=/var/mqm"
			qmDataPath := fmt.Sprintf("DataPath=/mnt/mqm-data/data/%v", name)
			out, rc, err := command.Run("addmqinf", "-s", "QueueManager", "-v", qmName, "-v", qmDirectory, "-v", qmPrefix, "-v", qmDataPath)
			if err != nil {
				log.Printf("addmqinf returned %v : %v", rc, string(out))
				return false, err
			}
			// Return true or false?
			return true, nil
		}
	} else {
		log.Printf("Detected existing queue manager %v", name)
		return false, nil
	}
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

func startQueueManager(name string) error {
	log.Println("Starting queue manager")
	out, rc, err := command.Run("strmqm", "-x", name)
	if err != nil {
		// 30=Standby queue manager started, which is fine
		if rc == 30 {
			log.Printf("Detected standby queue manager %v", name)
			return nil
		}
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
			// #nosec G204
			verify := exec.Command("runmqsc", "-v", "-e")
			cmd := exec.Command("runmqsc")
			// Read mqsc file into variable
			mqsc, err := ioutil.ReadFile(abs)
			if err != nil {
				log.Printf("Error reading file %v: %v", abs, err)
				continue
			}
			// Write mqsc to buffer
			var buffer bytes.Buffer
			_, err = buffer.Write(mqsc)
			if err != nil {
				log.Printf("Error writing MQSC file %v to buffer: %v", abs, err)
				continue
			}
			verifyBuffer := buffer

			// Buffer mqsc to stdin of runmqsc
			cmd.Stdin = &buffer
			verify.Stdin = &verifyBuffer

			// Verify the MQSC commands
			out, err := verify.CombinedOutput()
			if err != nil {
				log.Errorf("Error verifying MQSC file %v (%v):\n\t%v", file.Name(), err, formatMQSCOutput(string(out)))
				return fmt.Errorf("Error verifying MQSC file %v (%v):\n\t%v", file.Name(), err, formatMQSCOutput(string(out)))
			}

			// Run runmqsc command
			out, err = cmd.CombinedOutput()
			if err != nil {
				log.Errorf("Error running MQSC file %v (%v):\n\t%v", file.Name(), err, formatMQSCOutput(string(out)))
				continue
			} else {
				// Print the runmqsc output, adding tab characters to make it more readable as part of the log
				log.Printf("Output for \"runmqsc\" with %v:\n\t%v", abs, formatMQSCOutput(string(out)))
			}
		}
	}
	return nil
}

func stopQueueManager(name string) error {
	log.Println("Stopping queue manager")

	// TODO : tidy-up code
	isStandby, err := isStandbyQueueManager(name)
	if err != nil {
		return err
	}
	arg := "-s"
	if isStandby {
		arg = "-x"
	}
	out, _, err := command.Run("endmqm", arg, "-w", "-r", name)
	if err != nil {
		log.Printf("Error stopping queue manager: %v", string(out))
		return err
	}
	log.Println("Stopped queue manager")
	return nil
}

func formatMQSCOutput(out string) string {
	// redact sensitive information
	out, _ = mqscredact.Redact(out)

	// add tab characters to make it more readable as part of the log
	return strings.Replace(string(out), "\n", "\n\t", -1)
}

func isStandbyQueueManager(name string) (bool, error) {
	out, _, err := command.Run("dspmq", "-n", "-m", name)
	if err != nil {
		log.Printf("Error getting status for queue manager: %v", string(out))
		return false, err
	}
	if strings.Contains(string(out), "(RUNNING AS STANDBY)") {
		return true, nil
	}
	return false, nil
}
