/*
© Copyright IBM Corporation 2017, 2018

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
	"io/ioutil"
	"os/user"
	"runtime"
	"strings"

	"github.com/ibm-messaging/mq-container/internal/capabilities"
)

func logBaseImage() error {
	buf, err := ioutil.ReadFile("/etc/os-release")
	if err != nil {
		return err
	}
	lines := strings.Split(string(buf), "\n")
	for _, l := range lines {
		if strings.HasPrefix(l, "PRETTY_NAME=") {
			words := strings.Split(l, "\"")
			if len(words) >= 2 {
				log.Printf("Base image detected: %v", words[1])
				return nil
			}
		}
	}
	return nil
}

func logUser() {
	u, err := user.Current()
	if err == nil {
		log.Printf("Running as user ID %v (%v) with primary group %v", u.Uid, u.Name, u.Gid)
	}
}

func logCapabilities() {
	status, err := readProc("/proc/1/status")
	if err != nil {
		// Ignore
		return
	}
	caps, err := capabilities.DetectCapabilities(status)
	if err == nil {
		log.Printf("Detected capabilities: %v", strings.Join(caps, ","))
	}
}

func readProc(filename string) (value string, err error) {
	buf, err := ioutil.ReadFile(filename)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(buf)), nil
}

func readMounts() error {
	all, err := readProc("/proc/mounts")
	if err != nil {
		log.Print("Error: Couldn't read /proc/mounts")
		return err
	}
	lines := strings.Split(all, "\n")
	detected := false
	for i := range lines {
		parts := strings.Split(lines[i], " ")
		//dev := parts[0]
		mountPoint := parts[1]
		fsType := parts[2]
		if strings.Contains(mountPoint, "/mnt") {
			log.Printf("Detected '%v' volume mounted to %v", fsType, mountPoint)
			detected = true
		}
	}
	if !detected {
		log.Print("No volume detected. Persistent messages may be lost")
	} else {
		checkFS("/mnt/mqm")
	}
	return nil
}

func logConfig() {
	log.Printf("CPU architecture: %v", runtime.GOARCH)
	if runtime.GOOS == "linux" {
		var err error
		osr, err := readProc("/proc/sys/kernel/osrelease")
		if err != nil {
			log.Print(err)
		} else {
			log.Printf("Linux kernel version: %v", osr)
		}
		logBaseImage()
		fileMax, err := readProc("/proc/sys/fs/file-max")
		if err != nil {
			log.Print(err)
		} else {
			log.Printf("Maximum file handles: %v", fileMax)
		}
		logUser()
		logCapabilities()
		readMounts()
	} else {
		log.Fatalf("Unsupported platform: %v", runtime.GOOS)
	}
}
