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

// Package command contains code to run external commands
package command

import (
	"fmt"
	"os/exec"
	"runtime"
	"syscall"
)

// Run runs an OS command.  On Linux it waits for the command to
// complete and returns the exit status (return code).
// Do not use this function to run shell built-ins (like "cd"), because
// the error handling works differently
func Run(name string, arg ...string) (string, int, error) {
	cmd := exec.Command(name, arg...)
	// Run the command and wait for completion
	out, err := cmd.CombinedOutput()
	if err != nil {
		// Assert that this is an ExitError
		exiterr, ok := err.(*exec.ExitError)
		// If the type assertion was correct, and we're on Linux
		if ok && runtime.GOOS == "linux" {
			status, ok := exiterr.Sys().(syscall.WaitStatus)
			if ok {
				return string(out), status.ExitStatus(), fmt.Errorf("%v: %v", name, err)
			}
		}
		return string(out), -1, err
	}
	return string(out), 0, nil
}
