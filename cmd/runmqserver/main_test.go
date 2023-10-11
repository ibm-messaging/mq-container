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
	"flag"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/ibm-messaging/mq-container/pkg/logger"
)

var test *bool

func init() {
	test = flag.Bool("test", false, "Set to true when running tests for coverage")
	log, _ = logger.NewLogger(os.Stdout, true, false, "test")
}

// Test started when the test binary is started. Only calls main.
func TestSystem(t *testing.T) {
	if *test {
		var oldExit = osExit
		defer func() {
			osExit = oldExit
		}()

		filename, ok := os.LookupEnv("EXIT_CODE_FILE")
		if !ok {
			filename = "/var/coverage/exitCode"
		} else {
			filename = filepath.Join("/var/coverage/", filename)
		}

		osExit = func(rc int) {
			// Write the exit code to a file instead
			log.Printf("Writing exit code %v to file %v", strconv.Itoa(rc), filename)
			err := os.WriteFile(filename, []byte(strconv.Itoa(rc)), 0644)
			if err != nil {
				log.Print(err)
			}
		}
		main()
	}
}
