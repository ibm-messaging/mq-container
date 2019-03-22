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
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/ibm-messaging/mq-container/internal/command"
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
			// #nosec G204
			cmd := exec.Command("runmqsc")
			// Read mqsc file into variable
			mqsc, err := ioutil.ReadFile(abs)
			if err != nil {
				log.Printf("Error reading file %v: %v", abs, err)
			}
			// Write mqsc to buffer
			var buffer bytes.Buffer
			_, err = buffer.Write(mqsc)
			if err != nil {
				log.Printf("Error writing mqsc file %v to buffer: %v", abs, err)
			}
			// Buffer mqsc to stdin of runmqsc
			cmd.Stdin = &buffer
			// Run runmqsc command
			out, err := cmd.CombinedOutput()
			if err != nil {
				log.Errorf("Error running MQSC file %v (%v):\n\t%v", file.Name(), err, strings.Replace(string(out), "\n", "\n\t", -1))
			}
			// Print the runmqsc output, adding tab characters to make it more readable as part of the log
			log.Printf("Output for \"runmqsc\" with %v:\n\t%v", abs, strings.Replace(string(out), "\n", "\n\t", -1))
		}
	}
	return nil
}

func stopQueueManager(name string) error {
	log.Println("Stopping queue manager")
	out, _, err := command.Run("endmqm", "-w", "-r", name)
	if err != nil {
		log.Printf("Error stopping queue manager: %v", string(out))
		return err
	}
	log.Println("Stopped queue manager")
	return nil
}
