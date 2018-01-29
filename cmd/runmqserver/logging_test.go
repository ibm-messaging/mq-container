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
package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"testing"

	log "github.com/sirupsen/logrus"
)

func TestMirrorLogWithoutRotation(t *testing.T) {
	// Repeat the test multiple times, to help identify timing problems
	for i := 0; i < 10; i++ {
		t.Run(t.Name()+strconv.Itoa(i), func(t *testing.T) {
			// Use just the sub-test name in the file name
			tmp, err := ioutil.TempFile("", strings.Split(t.Name(), "/")[1])
			if err != nil {
				t.Fatal(err)
			}
			t.Log(tmp.Name())
			defer os.Remove(tmp.Name())
			count := 0
			lifecycle, err := mirrorLog(tmp.Name(), func(msg string) {
				count++
			})
			if err != nil {
				t.Fatal(err)
			}
			f, err := os.OpenFile(tmp.Name(), os.O_WRONLY, 0700)
			if err != nil {
				t.Fatal(err)
			}
			log.Println("Logging 3 JSON messages")
			fmt.Fprintln(f, "{\"message\"=\"A\"}")
			fmt.Fprintln(f, "{\"message\"=\"B\"}")
			fmt.Fprintln(f, "{\"message\"=\"C\"}")
			f.Close()
			lifecycle <- true
			<-lifecycle

			if count != 3 {
				t.Fatalf("Expected 3 log entries; got %v", count)
			}
		})
	}
}

func TestMirrorLogWithRotation(t *testing.T) {
	// Repeat the test multiple times, to help identify timing problems
	for i := 0; i < 5; i++ {
		t.Run(t.Name()+strconv.Itoa(i), func(t *testing.T) {
			// Use just the sub-test name in the file name
			tmp, err := ioutil.TempFile("", strings.Split(t.Name(), "/")[1])
			if err != nil {
				t.Fatal(err)
			}
			t.Log(tmp.Name())
			defer func() {
				t.Log("Removing file")
				os.Remove(tmp.Name())
			}()
			count := 0
			lifecycle, err := mirrorLog(tmp.Name(), func(msg string) {
				count++
			})
			if err != nil {
				t.Fatal(err)
			}
			f, err := os.OpenFile(tmp.Name(), os.O_WRONLY, 0700)
			if err != nil {
				t.Fatal(err)
			}
			t.Log("Logging 3 JSON messages")
			fmt.Fprintln(f, "{\"message\"=\"A\"}")
			fmt.Fprintln(f, "{\"message\"=\"B\"}")
			fmt.Fprintln(f, "{\"message\"=\"C\"}")
			f.Close()

			// Rotate the file, by renaming it
			rotated := tmp.Name() + ".1"
			os.Rename(tmp.Name(), rotated)
			defer os.Remove(rotated)
			// Open a new file, with the same name as before
			f, err = os.OpenFile(tmp.Name(), os.O_WRONLY|os.O_CREATE, 0700)
			if err != nil {
				t.Fatal(err)
			}
			t.Log("Logging 2 more JSON messages")
			fmt.Fprintln(f, "{\"message\"=\"D\"}")
			fmt.Fprintln(f, "{\"message\"=\"E\"}")
			f.Close()

			// Shut the mirroring down
			lifecycle <- true
			// Wait until it's finished
			<-lifecycle

			if count != 5 {
				t.Fatalf("Expected 5 log entries; got %v", count)
			}
		})
	}
}

func init() {
	log.SetLevel(log.DebugLevel)
}
