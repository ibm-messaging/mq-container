/*
© Copyright IBM Corporation 2017, 2026

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
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/ibm-messaging/mq-container/pkg/logger"
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

// cleanVolumeBestEffort deletes files/directories from the specified path, skipping deletion attempts for specified skipPaths and their parent directories.
// Files and sub-directories within a given skipPath will still be cleaned up. Deletion failures are logged as warnings.
func cleanVolumeBestEffort(cleanPath string, skipPaths []string, log *logger.Logger) {
	skipPathsMap := map[string]bool{
		cleanPath: true,
	}

	for _, skipPath := range skipPaths {
		// Skip the specified skipPath as well as any parent directories
		for ; skipPath != "." && strings.HasPrefix(skipPath, cleanPath); skipPath = filepath.Dir(skipPath) {
			skipPathsMap[skipPath] = true
		}
	}

	log.Debug("Begin cleaning path: ", cleanPath)
	// #nosec G104
	_ = filepath.WalkDir(cleanPath, func(path string, d fs.DirEntry, walkErr error) error {
		if skipPathsMap[path] {
			log.Debug("Skipping deletion of: ", path)
			return nil
		}
		log.Debug("Removing: ", path)
		err := os.RemoveAll(path)
		if err != nil {
			log.Printf("Warning: volume cleanup could not remove '%s': %v", path, err.Error())
			return nil
		}
		if d.IsDir() {
			return filepath.SkipDir
		}
		return nil
	})
	log.Debug("Finished cleaning path: ", cleanPath)
	return
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
