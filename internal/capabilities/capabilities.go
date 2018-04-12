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

// Package capabilities allows querying of information on Linux capabilities
package capabilities

import (
	"errors"
	"strconv"
	"strings"
)

// DetectCapabilities determines Linux capabilities, based on the contents of a Linux "status" file.
// For example, the contents of file `/proc/1/status`
func DetectCapabilities(status string) ([]string, error) {
	lines := strings.Split(status, "\n")
	for _, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "CapPrm:") {
			words := strings.Fields(line)
			cap, err := strconv.ParseUint(words[1], 16, 64)
			if err != nil {
				return nil, err
			}
			return getCapabilities(cap), nil
		}
	}
	return nil, errors.New("Unable to detect capabilities")
}

// getCapabilities converts an encoded uint64 into a slice of string names of Linux capabilities
func getCapabilities(cap uint64) []string {
	caps := make([]string, 0, 37)
	if cap&0x0000000040000000 == 0x0000000040000000 {
		caps = append(caps, "AUDIT_CONTROL")
	}
	if cap&0x0000000020000000 == 0x0000000020000000 {
		caps = append(caps, "AUDIT_WRITE")
	}
	if cap&0x0000001000000000 == 0x0000001000000000 {
		caps = append(caps, "BLOCK_SUSPEND")
	}
	if cap&0x0000000000000001 == 0x0000000000000001 {
		caps = append(caps, "CHOWN")
	}
	if cap&0x0000000000000002 == 0x0000000000000002 {
		caps = append(caps, "DAC_OVERRIDE")
	}
	if cap&0x0000000000000004 == 0x0000000000000004 {
		caps = append(caps, "DAC_READ_SEARCH")
	}
	if cap&0x0000000000000008 == 0x0000000000000008 {
		caps = append(caps, "FOWNER")
	}
	if cap&0x0000000000000010 == 0x0000000000000010 {
		caps = append(caps, "FSETID")
	}
	if cap&0x0000000000004000 == 0x0000000000004000 {
		caps = append(caps, "IPC_LOCK")
	}
	if cap&0x0000000000008000 == 0x0000000000008000 {
		caps = append(caps, "IPC_OWNER")
	}
	if cap&0x0000000000000020 == 0x0000000000000020 {
		caps = append(caps, "KILL")
	}
	if cap&0x0000000010000000 == 0x0000000010000000 {
		caps = append(caps, "LEASE")
	}
	if cap&0x0000000000000200 == 0x0000000000000200 {
		caps = append(caps, "LINUX_IMMUTABLE")
	}
	if cap&0x0000000200000000 == 0x0000000200000000 {
		caps = append(caps, "MAC_ADMIN")
	}
	if cap&0x0000000100000000 == 0x0000000100000000 {
		caps = append(caps, "MAC_OVERRIDE")
	}
	if cap&0x0000000008000000 == 0x0000000008000000 {
		caps = append(caps, "MKNOD")
	}
	if cap&0x0000000000001000 == 0x0000000000001000 {
		caps = append(caps, "NET_ADMIN")
	}
	if cap&0x0000000000000400 == 0x0000000000000400 {
		caps = append(caps, "NET_BIND_SERVICE")
	}
	if cap&0x0000000000000800 == 0x0000000000000800 {
		caps = append(caps, "NET_BROADCAST")
	}
	if cap&0x0000000000002000 == 0x0000000000002000 {
		caps = append(caps, "NET_RAW")
	}
	if cap&0x0000000080000000 == 0x0000000080000000 {
		caps = append(caps, "SETFCAP")
	}
	if cap&0x0000000000000040 == 0x0000000000000040 {
		caps = append(caps, "SETGID")
	}
	if cap&0x0000000000000100 == 0x0000000000000100 {
		caps = append(caps, "SETPCAP")
	}
	if cap&0x0000000000000080 == 0x0000000000000080 {
		caps = append(caps, "SETUID")
	}
	if cap&0x0000000400000000 == 0x0000000400000000 {
		caps = append(caps, "SYSLOG")
	}
	if cap&0x0000000000200000 == 0x0000000000200000 {
		caps = append(caps, "SYS_ADMIN")
	}
	if cap&0x0000000000400000 == 0x0000000000400000 {
		caps = append(caps, "SYS_BOOT")
	}
	if cap&0x0000000000040000 == 0x0000000000040000 {
		caps = append(caps, "SYS_CHROOT")
	}
	if cap&0x0000000000010000 == 0x0000000000010000 {
		caps = append(caps, "SYS_MODULE")
	}
	if cap&0x0000000000800000 == 0x0000000000800000 {
		caps = append(caps, "SYS_NICE")
	}
	if cap&0x0000000000100000 == 0x0000000000100000 {
		caps = append(caps, "SYS_PACCT")
	}
	if cap&0x0000000000080000 == 0x0000000000080000 {
		caps = append(caps, "SYS_PTRACE")
	}
	if cap&0x0000000000020000 == 0x0000000000020000 {
		caps = append(caps, "SYS_RAWIO")
	}
	if cap&0x0000000001000000 == 0x0000000001000000 {
		caps = append(caps, "SYS_RESOURCE")
	}
	if cap&0x0000000002000000 == 0x0000000002000000 {
		caps = append(caps, "SYS_TIME")
	}
	if cap&0x0000000004000000 == 0x0000000004000000 {
		caps = append(caps, "SYS_TTY_CONFIG")
	}
	if cap&0x0000000800000000 == 0x0000000800000000 {
		caps = append(caps, "WAKE_ALARM")
	}
	return caps
}
