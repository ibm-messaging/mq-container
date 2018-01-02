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
	"flag"
	"io/ioutil"
	"log"
	"strconv"
	"testing"
)

var test *bool

const filename = "/var/coverage/exitCode"

func init() {
	test = flag.Bool("test", false, "Set to true when running tests for coverage")
}

// Test started when the test binary is started. Only calls main.
func TestSystem(t *testing.T) {
	if *test {
		var oldExit = osExit
		defer func() {
			osExit = oldExit
		}()
		osExit = func(rc int) {
			// Write the exit code to a file instead
			log.Printf("Writing exit code %v to file %v", strconv.Itoa(rc), filename)
			err := ioutil.WriteFile(filename, []byte(strconv.Itoa(rc)), 0644)
			if err != nil {
				log.Print(err)
			}
		}
		main()
	}
}
