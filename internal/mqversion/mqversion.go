/*
© Copyright IBM Corporation 2020

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

package mqversion

import (
	"fmt"
	"strings"

	"github.com/ibm-messaging/mq-container/internal/command"
)

// Get will return the current MQ version
func Get() (string, error) {
	mqVersion, _, err := command.Run("dspmqver", "-b", "-f", "2")
	if err != nil {
		return "", fmt.Errorf("Error Getting MQ version: %v", err)
	}
	return strings.TrimSpace(mqVersion), nil
}

// Compare returns an integer comparing two MQ version strings lexicographically. The result will be 0 if currentVersion==checkVersion, -1 if currentVersion < checkVersion, and +1 if currentVersion > checkVersion
func Compare(checkVersion string) (int, error) {
	currentVersion, err := Get()
	if err != nil {
		return 0, err
	}
	// trim any suffix from MQ version x.x.x.x
	currentVersion = currentVersion[0:7]
	if currentVersion < checkVersion {
		return -1, nil
	} else if currentVersion == checkVersion {
		return 0, nil
	} else if currentVersion > checkVersion {
		return 1, nil
	}
	return 0, fmt.Errorf("Failed to compare MQ versions")
}
