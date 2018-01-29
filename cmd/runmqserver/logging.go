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
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

// waitForFile waits until the specified file exists
func waitForFile(path string) (os.FileInfo, error) {
	var fi os.FileInfo
	var err error
	// Wait for file to exist
	for {
		fi, err = os.Stat(path)
		if err != nil {
			if os.IsNotExist(err) {
				time.Sleep(500 * time.Millisecond)
				continue
			} else {
				return nil, err
			}
		}
		break
	}
	log.Debugf("File exists: %v, %v", path, fi.Size())
	return fi, nil
}

// mirrorAvailableMessages prints lines from the file, until no more are available
func mirrorAvailableMessages(f *os.File, w io.Writer) {
	scanner := bufio.NewScanner(f)
	count := 0
	for scanner.Scan() {
		t := scanner.Text()
		if strings.HasPrefix(t, "{") {
			// Assume JSON, so just print it
			fmt.Fprintln(w, t)
		} else if strings.HasPrefix(t, "AMQ") {
			// Only print MQ messages with AMQnnnn codes
			fmt.Fprintln(w, t)
		}
		count++
	}
	log.Debugf("Mirrored %v log entries", count)
	err := scanner.Err()
	if err != nil {
		log.Errorf("Error reading file: %v", err)
		return
	}
}

// mirrorLog tails the specified file, and logs each line to stdout.
// This is useful for usability, as the container console log can show
// messages from the MQ error logs.
func mirrorLog(path string, w io.Writer) (chan bool, error) {
	lifecycle := make(chan bool)
	var offset int64 = -1
	var f *os.File
	var err error
	var fi os.FileInfo
	// Need to check if the file exists before returning, otherwise we have a
	// race to see if the new file get created before we can test for it
	fi, err = os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			// File doesn't exist, so ensure we start at the beginning
			offset = 0
		} else {
			return nil, err
		}
	} else {
		// If the file exists, open it now, before we return.  This makes sure
		// the file is open before the queue manager is created or started.
		// Otherwise, there would be the potential for a nearly-full file to
		// rotate before the goroutine had a chance to open it.
		f, err = os.OpenFile(path, os.O_RDONLY, 0)
		if err != nil {
			return nil, err
		}
		// File already exists, so start reading at the end
		offset = fi.Size()
	}

	go func() {
		if f == nil {
			// File didn't exist, so need to wait for it
			fi, err = waitForFile(path)
			if err != nil {
				log.Errorln(err)
				lifecycle <- true
				return
			}
			f, err = os.OpenFile(path, os.O_RDONLY, 0)
			if err != nil {
				log.Errorln(err)
				lifecycle <- true
				return
			}
		}

		fi, err = f.Stat()
		if err != nil {
			log.Errorln(err)
			lifecycle <- true
			return
		}
		// The file now exists.  If it didn't exist before we started, offset=0
		if offset != 0 {
			log.Debugf("Seeking %v", offset)
			f.Seek(offset, 0)
		}
		closing := false
		for {
			log.Debugln("Start of loop")
			// If there's already data there, mirror it now.
			mirrorAvailableMessages(f, w)
			log.Debugf("Stat %v", path)
			newFI, err := os.Stat(path)
			if err != nil {
				log.Error(err)
				lifecycle <- true
				return
			}
			if !os.SameFile(fi, newFI) {
				log.Debugln("Not the same file!")
				// WARNING: There is a possible race condition here.  If *another*
				// log rotation happens before we can open the new file, then we
				// could skip all those messages.  This could happen with a very small
				// MQ error log size.
				mirrorAvailableMessages(f, w)
				f.Close()
				// Re-open file
				log.Debugln("Re-opening error log file")
				// Used to work with this: f, err = waitForFile2(path)
				f, err = os.OpenFile(path, os.O_RDONLY, 0)
				if err != nil {
					fmt.Printf("ERROR: %v", err)
					return
				}
				fi = newFI
				// Don't seek this time, because we know it's a new file
				mirrorAvailableMessages(f, w)
			}
			log.Debugln("Check for lifecycle event")
			select {
			// Have we been asked to shut down?
			case <-lifecycle:
				// Set a flag, to allow one more time through the loop
				closing = true
			default:
				if closing {
					log.Debugln("Shutting down mirror")
					lifecycle <- true
					return
				}
				time.Sleep(500 * time.Millisecond)
			}
		}
	}()
	return lifecycle, nil
}