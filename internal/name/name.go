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

// Package name contains code to manage the queue manager name
package name

import (
	"os"
	"regexp"
)

// sanitizeQueueManagerName removes any invalid characters from a queue manager name
func sanitizeQueueManagerName(name string) string {
	var re = regexp.MustCompile("[^a-zA-Z0-9._%/]")
	return re.ReplaceAllString(name, "")
}

// GetQueueManagerName resolves the queue manager name to use.  Resolved from
// either an environment variable, or the hostname.
func GetQueueManagerName() (string, error) {
	var name string
	var err error
	name, ok := os.LookupEnv("MQ_QMGR_NAME")
	if !ok || name == "" {
		name, err = os.Hostname()
		if err != nil {
			return "", err
		}
		name = sanitizeQueueManagerName(name)
	}
	// TODO: What if the specified env variable is an invalid name?
	return name, nil
}
