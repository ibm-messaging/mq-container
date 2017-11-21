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
package command

import (
	"runtime"
	"testing"
)

var commandTests = []struct {
	name string
	arg  []string
	rc   int
}{
	{"ls", []string{}, 0},
	{"ls", []string{"madeup"}, 2},
	{"bash", []string{"-c", "exit 99"}, 99},
}

func TestRun(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Skipping tests for package which only works on Linux")
	}
	for _, table := range commandTests {
		arg := table.arg
		_, rc, err := Run(table.name, arg...)
		if rc != table.rc {
			t.Errorf("Run(%v,%v) - expected %v, got %v", table.name, table.arg, table.rc, rc)
		}
		if rc != 0 && err == nil {
			t.Errorf("Run(%v,%v) - expected error for non-zero return code (rc=%v)", table.name, table.arg, rc)
		}
	}
}
