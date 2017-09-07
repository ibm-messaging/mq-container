/*
© Copyright IBM Corporation 2017

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
	"log"
	"os"
	"path/filepath"
	"runtime"
	"syscall"
)

//const mainDir string := "/mnt/mqm"
const mqmUID uint32 = 999
const mqmGID uint32 = 999

func createVolume(path string) error {
	// fi, err := os.Stat(path)
	// if err != nil {
	// 	if os.IsNotExist(err) {
	// 		// TODO: Should this be fatal?
	// 		//log.Warnf("No volume found under %v", path)
	// 		return nil
	// 	} else {
	// 		return err
	// 	}
	// }
	//log.Printf("%v details: %v", path, fi.Sys())
	dataPath := filepath.Join(path, "data")
	fi, err := os.Stat(dataPath)
	if err != nil {
		if os.IsNotExist(err) {
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
		// log.Printf("Checking UID/GID for %v", dataPath)
		//log.Debugf("Checking UID/GID for %v", dataPath)
		stat := sys.(*syscall.Stat_t)
		if stat.Uid != mqmUID || stat.Gid != mqmGID {
			err = os.Chown(dataPath, int(mqmUID), int(mqmGID))
			if err != nil {
				log.Printf("Error: Unable to change ownership of %v", dataPath)
				return err
			}
		}
	}
	return nil
}

// If /mnt/mqm exists
// 	If /mnt/mqm contains a "data" directory AND data is owned by mqm:mqm AND data is writeable by mqm:mqm then
// 		Create Symlink from /var/mqm to /mnt/mqm/data
// 	Else
// 		// Try to sort it out
// 		Create directory /mnt/mqm/data
// 		If directory not already owned by mqm:mqm
// 			chown mqm:mqm
// 			if error
// 				delete directory again if empty
// 			if directory not already 0755
// 				chmod 0755
// 				if error
// 					delete directory again if empty
