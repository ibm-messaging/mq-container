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
	"strconv"
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

	currentVRMF, err := parseVRMF(currentVersion)
	if err != nil {
		return 0, err
	}
	compareVRMF, err := parseVRMF(checkVersion)
	if err != nil {
		return 0, fmt.Errorf("failed to parse compare version: %w", err)
	}
	return currentVRMF.compare(*compareVRMF), nil
}

type vrmf [4]int

func (v vrmf) String() string {
	return fmt.Sprintf("%d.%d.%d.%d", v[0], v[1], v[2], v[3])
}

func (v vrmf) compare(to vrmf) int {
	for idx := 0; idx < 4; idx++ {
		if v[idx] < to[idx] {
			return -1
		}
		if v[idx] > to[idx] {
			return 1
		}
	}
	return 0
}

func parseVRMF(vrmfString string) (*vrmf, error) {
	versionParts := strings.Split(vrmfString, ".")
	if len(versionParts) != 4 {
		return nil, fmt.Errorf("incorrect number of parts to version string: expected 4, got %d", len(versionParts))
	}
	vmrfPartNames := []string{"version", "release", "minor", "fix"}
	parsed := vrmf{}
	for idx, value := range versionParts {
		partName := vmrfPartNames[idx]
		if value == "" {
			return nil, fmt.Errorf("empty %s found in VRMF", partName)
		}
		val, err := strconv.Atoi(value)
		if err != nil {
			return nil, fmt.Errorf("non-numeric %s found in VRMF", partName)
		}
		if val < 0 {
			return nil, fmt.Errorf("negative %s found in VRMF", partName)
		}
		if idx == 0 && val == 0 {
			return nil, fmt.Errorf("zero value for version not allowed")
		}
		parsed[idx] = val
	}
	return &parsed, nil
}
