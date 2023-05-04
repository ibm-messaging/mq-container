/*
© Copyright IBM Corporation 2017

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

http://www.apache.org/licenses/LICENSE_2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package main

import "testing"

func TestCountLinesWith(t *testing.T) {
	lines := []string{
		"/usr/bin/qemu-x86_x64 /usr/local/bin/runmqdevserver runmqdevserver",
		"date -u",
	}
	got := countLinesWith(lines, "runmqdevserver")
	if got != 1 {
		t.Fatalf("got %d occurances", got)
	}
}
