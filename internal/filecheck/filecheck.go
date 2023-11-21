/*
© Copyright IBM Corporation 2019, 2023

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

package filecheck

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/ibm-messaging/mq-container/internal/pathutils"
)

// CheckFileSource checks the filename is valid
func CheckFileSource(fileName string) error {

	absFile, _ := filepath.Abs(fileName)

	prefixes := []string{"bin", "boot", "dev", "lib", "lib64", "proc", "sbin", "sys"}
	for _, prefix := range prefixes {
		if strings.HasPrefix(absFile, pathutils.CleanPath("/", prefix)) {
			return fmt.Errorf("Filename resolves to invalid path '%v'", absFile)
		}
	}
	return nil
}
