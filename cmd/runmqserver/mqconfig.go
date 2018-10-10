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
	"fmt"
	"io/ioutil"
	"runtime"
	"strings"

	"github.com/genuinetools/amicontained/container"
)

func logContainerRuntime() {
	r, err := container.DetectRuntime()
	if err != nil {
		log.Printf("Failed to get container runtime: %v", err)
	}
	log.Printf("Container runtime: %v", r)
}

func logBaseImage() {
	buf, err := ioutil.ReadFile("/etc/os-release")
	if err != nil {
		log.Printf("Failed to read /etc/os-release: %v", err)
	}
	lines := strings.Split(string(buf), "\n")
	for _, l := range lines {
		if strings.HasPrefix(l, "PRETTY_NAME=") {
			words := strings.Split(l, "\"")
			if len(words) >= 2 {
				log.Printf("Base image: %v", words[1])
				return
			}
		}
	}
}

// logCapabilities logs the Linux capabilities (e.g. setuid, setgid).  See https://docs.docker.com/engine/reference/run/#runtime-privilege-and-linux-capabilities
func logCapabilities() {
	caps, err := container.Capabilities()
	if err != nil {
		log.Printf("Failed to get container capabilities: %v", err)
	}
	for k, v := range caps {
		if len(v) > 0 {
			log.Printf("Capabilities (%s set): %v", strings.ToLower(k), strings.Join(v, ","))
		}
	}
}

// logSeccomp logs the seccomp enforcing mode, which affects which kernel calls can be made
func logSeccomp() {
	s, err := container.SeccompEnforcingMode()
	if err != nil {
		log.Printf("Failed to get container SeccompEnforcingMode: %v", err)
	}
	log.Printf("seccomp enforcing mode: %v", s)
	return nil
}

// logSecurityAttributes logs the security attributes of the current process.
// The security attributes indicate whether AppArmor or SELinux are being used,
// and what the level of confinement is.
func logSecurityAttributes() {
	a, err := readProc("/proc/self/attr/current")
	// On some systems, if AppArmor or SELinux are not installed, you get an
	// error when you try and read `/proc/self/attr/current`, even though the
	// file exists.
	if err != nil || a == "" {
		a = "none"
	}
	log.Printf("Process security attributes: %v", a)
}

func readProc(filename string) (value string, err error) {
	// #nosec G304
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
		if strings.Contains(mountPoint, "/mnt/mqm") {
			log.Printf("Detected '%v' volume mounted to %v", fsType, mountPoint)
			detected = true
		}
	}
	if !detected {
		log.Print("No volume detected. Persistent messages may be lost")
	} else {
		return checkFS("/mnt/mqm")
	}
	return nil
}

func logConfig() error {
	log.Printf("CPU architecture: %v", runtime.GOARCH)
	if runtime.GOOS == "linux" {
		var err error
		osr, err := readProc("/proc/sys/kernel/osrelease")
		if err != nil {
			log.Print(err)
		} else {
			log.Printf("Linux kernel version: %v", osr)
		}
		logContainerRuntime()
		logBaseImage()
		fileMax, err := readProc("/proc/sys/fs/file-max")
		if err != nil {
			log.Print(err)
		} else {
			log.Printf("Maximum file handles: %v", fileMax)
		}
		logUser()
		logCapabilities()
		logSeccomp()
		logSecurityAttributes()
		err = readMounts()
		if err != nil {
			return err
		}
	} else {
		return fmt.Errorf("Unsupported platform: %v", runtime.GOOS)
	}
	return nil
}
