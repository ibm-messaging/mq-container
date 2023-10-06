/*
The MIT License (MIT)

Copyright (c) 2018 Jessica Frazelle

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
*/

/*
  The code for amicontained.go is forked from
  https://github.com/genuinetools/bpfd/blob/434b609b3d4a5aeb461109b1167b68e000b72f69/proc/proc.go

  The code was forked when the latest details are as "Latest commit 871fc34 on Sep 18, 2018"

*/

// Adding IBM Copyright since the forked code had to be modified to remove deprecated ioutil package
/*
© Copyright IBM Corporation 2023
*/

package containerruntime

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/syndtr/gocapability/capability"
	"golang.org/x/sys/unix"
)

// ContainerRuntime is the type for the various container runtime strings.
type ContainerRuntime string

// SeccompMode is the type for the various seccomp mode strings.
type SeccompMode string

const (
	// RuntimeDocker is the string for the docker runtime.
	RuntimeDocker ContainerRuntime = "docker"
	// RuntimeRkt is the string for the rkt runtime.
	RuntimeRkt ContainerRuntime = "rkt"
	// RuntimeNspawn is the string for the systemd-nspawn runtime.
	RuntimeNspawn ContainerRuntime = "systemd-nspawn"
	// RuntimeLXC is the string for the lxc runtime.
	RuntimeLXC ContainerRuntime = "lxc"
	// RuntimeLXCLibvirt is the string for the lxc-libvirt runtime.
	RuntimeLXCLibvirt ContainerRuntime = "lxc-libvirt"
	// RuntimeOpenVZ is the string for the openvz runtime.
	RuntimeOpenVZ ContainerRuntime = "openvz"
	// RuntimeKubernetes is the string for the kubernetes runtime.
	RuntimeKubernetes ContainerRuntime = "kube"
	// RuntimeGarden is the string for the garden runtime.
	RuntimeGarden ContainerRuntime = "garden"
	// RuntimePodman is the string for the podman runtime.
	RuntimePodman ContainerRuntime = "podman"
	// RuntimeNotFound is the string for when no container runtime is found.
	RuntimeNotFound ContainerRuntime = "not-found"

	// SeccompModeDisabled is equivalent to "0" in the /proc/{pid}/status file.
	SeccompModeDisabled SeccompMode = "disabled"
	// SeccompModeStrict is equivalent to "1" in the /proc/{pid}/status file.
	SeccompModeStrict SeccompMode = "strict"
	// SeccompModeFiltering is equivalent to "2" in the /proc/{pid}/status file.
	SeccompModeFiltering SeccompMode = "filtering"

	apparmorUnconfined = "unconfined"

	uint32Max = 4294967295

	statusFileValue = ":(.*)"
)

var (
	// ContainerRuntimes contains all the container runtimes.
	ContainerRuntimes = []ContainerRuntime{
		RuntimeDocker,
		RuntimeRkt,
		RuntimeNspawn,
		RuntimeLXC,
		RuntimeLXCLibvirt,
		RuntimeOpenVZ,
		RuntimeKubernetes,
		RuntimeGarden,
		RuntimePodman,
	}

	seccompModes = map[string]SeccompMode{
		"0": SeccompModeDisabled,
		"1": SeccompModeStrict,
		"2": SeccompModeFiltering,
	}

	statusFileValueRegex = regexp.MustCompile(statusFileValue)
)

// GetContainerRuntime returns the container runtime the process is running in.
// If pid is less than one, it returns the runtime for "self".
func GetContainerRuntime(tgid, pid int) ContainerRuntime {
	file := "/proc/self/cgroup"
	if pid > 0 {
		if tgid > 0 {
			file = fmt.Sprintf("/proc/%d/task/%d/cgroup", tgid, pid)
		} else {
			file = fmt.Sprintf("/proc/%d/cgroup", pid)
		}
	}

	// read the cgroups file
	a := readFileString(file)
	runtime := getContainerRuntime(a)
	if runtime != RuntimeNotFound {
		return runtime
	}

	// /proc/vz exists in container and outside of the container, /proc/bc only outside of the container.
	if fileExists("/proc/vz") && !fileExists("/proc/bc") {
		return RuntimeOpenVZ
	}

	a = os.Getenv("container")
	runtime = getContainerRuntime(a)
	if runtime != RuntimeNotFound {
		return runtime
	}

	// PID 1 might have dropped this information into a file in /run.
	// Read from /run/systemd/container since it is better than accessing /proc/1/environ,
	// which needs CAP_SYS_PTRACE
	a = readFileString("/run/systemd/container")
	runtime = getContainerRuntime(a)
	if runtime != RuntimeNotFound {
		return runtime
	}

	return RuntimeNotFound
}

func getContainerRuntime(input string) ContainerRuntime {
	if len(strings.TrimSpace(input)) < 1 {
		return RuntimeNotFound
	}

	for _, runtime := range ContainerRuntimes {
		if strings.Contains(input, string(runtime)) {
			return runtime
		}
	}

	return RuntimeNotFound
}

func fileExists(file string) bool {
	if _, err := os.Stat(file); !os.IsNotExist(err) {
		return true
	}
	return false
}

func readFile(file string) []byte {
	if !fileExists(file) {
		return nil
	}
	// filepath.clean was added to resolve the gosec build failure
	// with error "Potential file inclusion via variable"
	// IBM Modified the below line to remove the deprecated ioutil dependency
	b, err := os.ReadFile(filepath.Clean(file))
	if err != nil {
		return nil
	}
	return b
}

// GetCapabilities returns the allowed capabilities for the process.
// If pid is less than one, it returns the capabilities for "self".
func GetCapabilities(pid int) (map[string][]string, error) {
	allCaps := capability.List()

	caps, err := capability.NewPid(pid)
	if err != nil {
		return nil, err
	}

	allowedCaps := map[string][]string{}
	allowedCaps["EFFECTIVE | PERMITTED | INHERITABLE"] = []string{}
	allowedCaps["BOUNDING"] = []string{}
	allowedCaps["AMBIENT"] = []string{}

	for _, cap := range allCaps {
		if caps.Get(capability.CAPS, cap) {
			allowedCaps["EFFECTIVE | PERMITTED | INHERITABLE"] = append(allowedCaps["EFFECTIVE | PERMITTED | INHERITABLE"], cap.String())
		}
		if caps.Get(capability.BOUNDING, cap) {
			allowedCaps["BOUNDING"] = append(allowedCaps["BOUNDING"], cap.String())
		}
		if caps.Get(capability.AMBIENT, cap) {
			allowedCaps["AMBIENT"] = append(allowedCaps["AMBIENT"], cap.String())
		}
	}

	return allowedCaps, nil
}

// GetSeccompEnforcingMode returns the seccomp enforcing level (disabled, filtering, strict)
// for a process.
// If pid is less than one, it returns the seccomp enforcing mode for "self".
func GetSeccompEnforcingMode(pid int) SeccompMode {
	file := "/proc/self/status"
	if pid > 0 {
		file = fmt.Sprintf("/proc/%d/status", pid)
	}

	return getSeccompEnforcingMode(readFileString(file))
}

func getSeccompEnforcingMode(input string) SeccompMode {
	mode := getStatusEntry(input, "Seccomp:")
	sm, ok := seccompModes[mode]
	if ok {
		return sm
	}

	// Pre linux 3.8, check if Seccomp is supported, via CONFIG_SECCOMP.
	if err := unix.Prctl(unix.PR_GET_SECCOMP, 0, 0, 0, 0); err != unix.EINVAL {
		// Make sure the kernel has CONFIG_SECCOMP_FILTER.
		if err := unix.Prctl(unix.PR_SET_SECCOMP, unix.SECCOMP_MODE_FILTER, 0, 0, 0); err != unix.EINVAL {
			return SeccompModeStrict
		}
	}

	return SeccompModeDisabled
}

// TODO: make this function more efficient and read the file line by line.
func getStatusEntry(input, find string) string {
	// Split status file string by line
	statusMappings := strings.Split(input, "\n")
	statusMappings = deleteEmpty(statusMappings)

	for _, line := range statusMappings {
		if strings.Contains(line, find) {
			matches := statusFileValueRegex.FindStringSubmatch(line)
			if len(matches) > 1 {
				return strings.TrimSpace(matches[1])
			}
		}
	}

	return ""
}

func deleteEmpty(s []string) []string {
	var r []string
	for _, str := range s {
		if strings.TrimSpace(str) != "" {
			r = append(r, strings.TrimSpace(str))
		}
	}
	return r
}

func readFileString(file string) string {
	b := readFile(file)
	if b == nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}
