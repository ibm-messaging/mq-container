/*
© Copyright IBM Corporation 2017, 2023

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
)

func createVolume(dataPath string) error {
	_, err := os.Stat(dataPath)
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
	return nil
}

// Delete files/directories from specified path
func cleanVolume(cleanPath string) error {
	// #nosec G304
	dirContents, err := os.ReadDir(cleanPath)
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
	}

	for _, name := range dirContents {
		err = os.RemoveAll(filepath.Join(cleanPath, name.Name()))
		if err != nil {
			return err
		}
	}

	return nil
}
