//go:build linux
// +build linux

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
© Copyright IBM Corporation 2023, 2025
*/

package containerruntime

import (
	"fmt"

	"golang.org/x/sys/unix"
)

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
