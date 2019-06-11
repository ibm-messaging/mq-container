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

	"golang.org/x/sys/unix"

	"github.com/ibm-messaging/mq-container/internal/command"
	containerruntime "github.com/ibm-messaging/mq-container/internal/containerruntime"
	"github.com/ibm-messaging/mq-container/internal/mqscredact"
	"github.com/ibm-messaging/mq-container/internal/ready"
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

// configureOwnership recursively handles ownership of files within the given filepath
func configureOwnership(paths []string) error {
	uid, gid, err := command.LookupMQM()
	if err != nil {
		return err
	}
	var fileInfo *unix.Stat_t
	fileInfo = new(unix.Stat_t)
	for _, root := range paths {
		_, err = os.Stat(root)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return err
		}
		err = filepath.Walk(root, func(from string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			to := fmt.Sprintf("%v%v", root, from[len(root):])
			err = unix.Stat(to, fileInfo)
			if err != nil {
				return err
			}
			fileUID := fmt.Sprint(fileInfo.Uid)
			if strings.Compare(fileUID, "999") == 0 {
				err = os.Chown(to, uid, gid)
				if err != nil {
					return err
				}
			}
			return nil
		})
		if err != nil {
			return err
		}
	}
	return nil
}

// createQueueManager creates a queue manager, if it doesn't already exist.
// It returns true if one was created (or a standby was created), or false if one already existed
func createQueueManager(name string) (bool, error) {
	log.Printf("Creating queue manager %v", name)

	// Run 'dspmqinf' to check if 'mqs.ini' configuration file exists
	// If command succeeds, the queue manager (or standby queue manager) has already been created
	_, _, err := command.Run("dspmqinf", name)
	if err == nil {
		log.Printf("Detected existing queue manager %v", name)
		return false, nil
	}

	mounts, err := containerruntime.GetMounts()
	if err != nil {
		log.Printf("Error getting mounts for queue manager")
		return false, err
	}

	// Check if 'qm.ini' configuration file exists for the queue manager
	// TODO : handle possible race condition - use a file lock?
	dataDir := getQueueManagerDataDir(mounts, name)
	_, err = os.Stat(filepath.Join(dataDir, "qm.ini"))
	if err != nil {
		// If 'qm.ini' is not found - run 'crtmqm' to create a new queue manager
		args := getCreateQueueManagerArgs(mounts, name)
		out, rc, err := command.Run("crtmqm", args...)
		if err != nil {
			log.Printf("Error %v creating queue manager: %v", rc, string(out))
			return false, err
		}
	} else {
		// If 'qm.ini' is found - run 'addmqinf' to create a standby queue manager with existing configuration
		args := getCreateStandbyQueueManagerArgs(name)
		out, rc, err := command.Run("addmqinf", args...)
		if err != nil {
			log.Printf("Error %v creating standby queue manager: %v", rc, string(out))
			return false, err
		}
		log.Println("Created standby queue manager")
		return true, nil
	}
	log.Println("Created queue manager")
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

func startQueueManager(name string) error {
	log.Println("Starting queue manager")
	out, rc, err := command.Run("strmqm", "-x", name)
	if err != nil {
		// 30=standby queue manager started, which is fine
		if rc == 30 {
			log.Printf("Started standby queue manager")
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
			// #nosec G204 - command is fixed, no injection vector
			cmd := exec.Command("runmqsc")
			// Read mqsc file into variable
			// #nosec G304 - filename variable is derived from contents of 'configDir' which is a defined constant
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
	isStandby, err := ready.IsRunningAsStandbyQM(name)
	if err != nil {
		log.Printf("Error getting status for queue manager %v: ", name, err.Error())
		return err
	}
	args := []string{"-w", "-r", name}
	if os.Getenv("MQ_MULTI_INSTANCE") == "true" {
		if isStandby {
			args = []string{"-x", name}
		} else {
			args = []string{"-s", "-w", "-r", name}
		}
	}
	out, rc, err := command.Run("endmqm", args...)
	if err != nil {
		log.Printf("Error %v stopping queue manager: %v", rc, string(out))
		return err
	}
	if isStandby {
		log.Printf("Stopped standby queue manager")
	} else {
		log.Println("Stopped queue manager")
	}
	return nil
}

func formatMQSCOutput(out string) string {
	// redact sensitive information
	out, _ = mqscredact.Redact(out)

	// add tab characters to make it more readable as part of the log
	return strings.Replace(string(out), "\n", "\n\t", -1)
}

func isStandbyQueueManager(name string) (bool, error) {
	out, rc, err := command.Run("dspmq", "-n", "-m", name)
	if err != nil {
		log.Printf("Error %v getting status for queue manager %v: %v", rc, name, string(out))
		return false, err
	}
	if strings.Contains(string(out), "(RUNNING AS STANDBY)") {
		return true, nil
	}
	return false, nil
}

func getQueueManagerDataDir(mounts map[string]string, name string) string {
	dataDir := filepath.Join("/var/mqm/qmgrs", name)
	if _, ok := mounts["/mnt/mqm-data"]; ok {
		dataDir = filepath.Join("/mnt/mqm-data/qmgrs", name)
	}
	return dataDir
}

func getCreateQueueManagerArgs(mounts map[string]string, name string) []string {
	args := []string{"-q", "-p", "1414"}
	if _, ok := mounts["/mnt/mqm-log"]; ok {
		args = append(args, "-ld", "/mnt/mqm-log/log")
	}
	if _, ok := mounts["/mnt/mqm-data"]; ok {
		args = append(args, "-md", "/mnt/mqm-data/qmgrs")
	}
	args = append(args, name)
	return args
}

func getCreateStandbyQueueManagerArgs(name string) []string {
	args := []string{"-s", "QueueManager"}
	args = append(args, "-v", fmt.Sprintf("Name=%v", name))
	args = append(args, "-v", fmt.Sprintf("Directory=%v", name))
	args = append(args, "-v", "Prefix=/var/mqm")
	args = append(args, "-v", fmt.Sprintf("DataPath=/mnt/mqm-data/qmgrs/%v", name))
	return args
}
