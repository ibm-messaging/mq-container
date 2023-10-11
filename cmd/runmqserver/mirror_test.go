/*
© Copyright IBM Corporation 2018, 2023

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
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestMirrorLogWithoutRotation(t *testing.T) {
	// Repeat the test multiple times, to help identify timing problems
	for i := 0; i < 10; i++ {
		t.Run(t.Name()+strconv.Itoa(i), func(t *testing.T) {
			// Use just the sub-test name in the file name
			tmp, err := os.CreateTemp("", strings.Split(t.Name(), "/")[1])
			if err != nil {
				t.Fatal(err)
			}
			t.Log(tmp.Name())
			defer os.Remove(tmp.Name())
			count := 0
			ctx, cancel := context.WithCancel(context.Background())
			var wg sync.WaitGroup
			_, err = mirrorLog(ctx, &wg, tmp.Name(), true, func(msg string, isQMLog bool) bool {
				count++
				return true
			}, false)
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
			cancel()
			wg.Wait()
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
			tmp, err := os.CreateTemp("", strings.Split(t.Name(), "/")[1])
			if err != nil {
				t.Fatal(err)
			}
			t.Log(tmp.Name())
			defer func() {
				t.Log("Removing file")
				os.Remove(tmp.Name())
			}()
			count := 0
			ctx, cancel := context.WithCancel(context.Background())
			var wg sync.WaitGroup
			_, err = mirrorLog(ctx, &wg, tmp.Name(), true, func(msg string, isQMLog bool) bool {
				count++
				return true
			}, false)
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
			cancel()
			wg.Wait()

			if count != 5 {
				t.Fatalf("Expected 5 log entries; got %v", count)
			}
		})
	}
}

func testMirrorLogExistingFile(t *testing.T, newQM bool) int {
	tmp, err := os.CreateTemp("", t.Name())
	if err != nil {
		t.Fatal(err)
	}
	t.Log(tmp.Name())
	log.Println("Logging 1 message before we start")
	os.WriteFile(tmp.Name(), []byte("{\"message\"=\"A\"}\n"), 0600)
	defer os.Remove(tmp.Name())
	count := 0
	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	_, err = mirrorLog(ctx, &wg, tmp.Name(), newQM, func(msg string, isQMLog bool) bool {
		count++
		return true
	}, false)
	if err != nil {
		t.Fatal(err)
	}
	f, err := os.OpenFile(tmp.Name(), os.O_APPEND|os.O_WRONLY, 0700)
	if err != nil {
		t.Fatal(err)
	}
	log.Println("Logging 2 new JSON messages")
	fmt.Fprintln(f, "{\"message\"=\"B\"}")
	fmt.Fprintln(f, "{\"message\"=\"C\"}")
	f.Close()
	cancel()
	wg.Wait()
	return count
}

// TestMirrorLogExistingFile tests that we only get new log messages, if the
// log file already exists
func TestMirrorLogExistingFile(t *testing.T) {
	count := testMirrorLogExistingFile(t, false)
	if count != 2 {
		t.Fatalf("Expected 2 log entries; got %v", count)
	}
}

// TestMirrorLogExistingFileButNewQueueManager tests that we only get all log
// messages, even if the file exists, if we tell it we want all messages
func TestMirrorLogExistingFileButNewQueueManager(t *testing.T) {
	count := testMirrorLogExistingFile(t, true)
	if count != 3 {
		t.Fatalf("Expected 3 log entries; got %v", count)
	}
}

func TestMirrorLogCancelWhileWaiting(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	defer func() {
		cancel()
		wg.Wait()
	}()
	_, err := mirrorLog(ctx, &wg, "fake.log", true, func(msg string, isQMLog bool) bool {
		return true
	}, false)
	if err != nil {
		t.Error(err)
	}
	time.Sleep(time.Second * 3)
	cancel()
	wg.Wait()
	// No need to assert anything.  If it didn't work, the code would have hung (TODO: not ideal)
}
