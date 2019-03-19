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
	"os"
	"path/filepath"
	"runtime"
	"syscall"

	"github.com/ibm-messaging/mq-container/internal/command"
)

func createVolume(path string) error {
	dataPath := filepath.Join(path, "data")
	fi, err := os.Stat(dataPath)
	if err != nil {
		if os.IsNotExist(err) {
			// #nosec G301
			err = os.MkdirAll(dataPath, 0755)
			if err != nil {
				return err
			}
		} else {
			return err
		}
	}
	fi, err = os.Stat(dataPath)
	if err != nil {
		return err
	}
	sys := fi.Sys()
	if sys != nil && runtime.GOOS == "linux" {
		stat := sys.(*syscall.Stat_t)
		mqmUID, mqmGID, err := command.LookupMQM()
		if err != nil {
			return err
		}
		log.Debugf("mqm user is %v (%v)", mqmUID, mqmGID)
		if int(stat.Uid) != mqmUID || int(stat.Gid) != mqmGID {
			err = os.Chown(dataPath, mqmUID, mqmGID)
			if err != nil {
				log.Printf("Error: Unable to change ownership of %v", dataPath)
				return err
			}
		}
	}
	return nil
}

func createWebConsoleTLSDirStructure() error {
	// Create tls directory
	dir := "/run/tls"
	_, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			err = os.MkdirAll(dir, 0770)
			if err != nil {
				return err
			}
			mqmUID, mqmGID, err := command.LookupMQM()
			if err != nil {
				log.Error(err)
				return err
			}
			err = os.Chown(dir, mqmUID, mqmGID)
			if err != nil {
				log.Error(err)
				return err
			}
		} else {
			return err
		}
	}

	return nil
}

func createDevTLSDir() error {
	// TODO: Use a persisted file (on the volume) instead?
	par := "/run/runmqdevserver"
	dir := filepath.Join(par, "tls")

	_, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			// #nosec G301
			err = os.MkdirAll(dir, 0770)
			if err != nil {
				return err
			}
			mqmUID, mqmGID, err := command.LookupMQM()
			if err != nil {
				log.Error(err)
				return err
			}
			err = os.Chown(dir, mqmUID, mqmGID)
			if err != nil {
				log.Error(err)
				return err
			}
			err = os.Chown(par, mqmUID, mqmGID)
			if err != nil {
				log.Error(err)
				return err
			}

		} else {
			return err
		}
	}
	return nil
}
