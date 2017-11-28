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
package capabilities

import (
	"reflect"
	"testing"
)

var capTests = []struct {
	in  uint64
	out []string
}{
	{0x0000000040000000, []string{"AUDIT_CONTROL"}},
	// Default values when you run a Docker container without changing capabilities:
	{0x00000000a80425fb, []string{"AUDIT_WRITE", "CHOWN", "DAC_OVERRIDE", "FOWNER", "FSETID", "KILL", "MKNOD", "NET_BIND_SERVICE", "NET_RAW", "SETFCAP", "SETGID", "SETPCAP", "SETUID", "SYS_CHROOT"}},
}

func TestGetCapabilities(t *testing.T) {
	for _, table := range capTests {
		caps := getCapabilities(table.in)
		if !reflect.DeepEqual(caps, table.out) {
			t.Errorf("getCapabilities(%v) - expected %v, got %v", table.in, table.out, caps)
		}
	}
}
