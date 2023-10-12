/*
© Copyright IBM Corporation 2019,2023

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
package containerruntime

import (
	"fmt"
	"os"
	"strings"
)

func DetectContainerRuntime() ContainerRuntime {
	return GetContainerRuntime(0, 1)
}

func GetBaseImage() (string, error) {
	buf, err := os.ReadFile("/etc/os-release")
	if err != nil {
		return "", fmt.Errorf("Failed to read /etc/os-release: %v", err)
	}
	lines := strings.Split(string(buf), "\n")
	for _, l := range lines {
		if strings.HasPrefix(l, "PRETTY_NAME=") {
			words := strings.Split(l, "\"")
			if len(words) >= 2 {
				return words[1], nil
			}
		}
	}
	return "unknown", nil
}

// GetCapabilities gets the Linux capabilities (e.g. setuid, setgid).  See https://docs.docker.com/engine/reference/run/#runtime-privilege-and-linux-capabilities
func GetContainerCapabilities() (map[string][]string, error) {
	//passing the pid as 1 since runmqserver initializes, creates and starts a queue manager, as PID 1 in a container
	return GetCapabilities(1)
}

// GetSeccomp gets the seccomp enforcing mode, which affects which kernel calls can be made
func GetSeccomp() SeccompMode {
	//passing the pid as 1 since runmqserver initializes, creates and starts a queue manager, as PID 1 in a container
	return GetSeccompEnforcingMode(1)
}

// GetSecurityAttributes gets the security attributes of the current process.
// The security attributes indicate whether AppArmor or SELinux are being used,
// and what the level of confinement is.
func GetSecurityAttributes() string {
	a, err := readProc("/proc/self/attr/current")
	// On some systems, if AppArmor or SELinux are not installed, you get an
	// error when you try and read `/proc/self/attr/current`, even though the
	// file exists.
	if err != nil || a == "" {
		a = "none"
	}
	return a
}

func readProc(filename string) (value string, err error) {
	// #nosec G304
	buf, err := os.ReadFile(filename)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(buf)), nil
}

func GetMounts() (map[string]string, error) {
	all, err := readProc("/proc/mounts")
	if err != nil {
		return nil, fmt.Errorf("Couldn't read /proc/mounts")
	}
	result := make(map[string]string)
	lines := strings.Split(all, "\n")
	for i := range lines {
		parts := strings.Split(lines[i], " ")
		//dev := parts[0]
		mountPoint := parts[1]
		fsType := parts[2]
		if strings.Contains(mountPoint, "/mnt/mqm") {
			result[mountPoint] = fsType
		}
	}
	return result, nil
}

func GetKernelVersion() (string, error) {
	return readProc("/proc/sys/kernel/osrelease")
}

func GetMaxFileHandles() (string, error) {
	return readProc("/proc/sys/fs/file-max")
}

// SupportedFilesystem returns true if the supplied filesystem type is supported for MQ data
func SupportedFilesystem(fsType string) bool {
	switch fsType {
	case "aufs", "overlayfs", "tmpfs":
		return false
	default:
		return true
	}
}

// ValidMultiInstanceFilesystem returns true if the supplied filesystem type is valid for a multi-instance queue manager
func ValidMultiInstanceFilesystem(fsType string) bool {
	if !SupportedFilesystem(fsType) {
		return false
	}
	// TODO : check for non-shared filesystems & shared filesystems which are known not to work
	return true
}
