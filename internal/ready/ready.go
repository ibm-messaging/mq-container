/*
© Copyright IBM Corporation 2018

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

// Package ready contains code to provide a ready signal mechanism between processes
package ready

import (
	"io/ioutil"
	"os"

	log "github.com/sirupsen/logrus"
)

const fileName string = "/run/runmqserver/ready"

func fileExists() (bool, error) {
	_, err := os.Stat(fileName)
	if err != nil {
		if !os.IsNotExist(err) {
			return false, err
		}
		return false, nil
	}
	return true, nil
}

// Clear ensures that any readiness state is cleared
func Clear() error {
	log.Debug("Clear()")
	exist, err := fileExists()
	if err != nil {
		return err
	}
	if exist {
		return os.Remove(fileName)
	}
	return nil
}

// Set lets any subsequent calls to `CheckReady` know that the queue
// manager has finished its configuration step
func Set() error {
	log.Debug("Set()")
	return ioutil.WriteFile(fileName, []byte("1"), 0770)
}

// Check checks whether or not the queue manager has finished its
// configuration steps
func Check() (bool, error) {
	exists, err := fileExists()
	if err != nil {
		log.Debug("Check() -> false")
		return false, err
	}
	log.Debugf("Check() -> %v", exists)
	return exists, nil
}
