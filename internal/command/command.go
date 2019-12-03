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
	"os/user"
	"strconv"
	"syscall"
)

// Run runs an OS command.  On Linux it waits for the command to
// complete and returns the exit status (return code).
// Do not use this function to run shell built-ins (like "cd"), because
// the error handling works differently
func Run(name string, arg ...string) (string, int, error) {
	// Run the command and wait for completion
	// #nosec G204
	cmd := exec.Command(name, arg...)
	out, err := cmd.CombinedOutput()
	rc := cmd.ProcessState.ExitCode()
	if err != nil {
		return string(out), rc, fmt.Errorf("%v: %v", cmd.Path, err)
	}
	return string(out), rc, nil
}

// RunAsMQM runs the specified command as the mqm user
func RunAsMQM(name string, arg ...string) (string, int, error) {
	// #nosec G204
	cmd := exec.Command(name, arg...)
	cmd.SysProcAttr = &syscall.SysProcAttr{}
	uid, gid, err := LookupMQM()
	if err != nil {
		return "", 0, err
	}
	cmd.SysProcAttr.Credential = &syscall.Credential{Uid: uint32(uid), Gid: uint32(gid)}
	return Run(name, arg...)
}

// LookupMQM looks up the UID & GID of the mqm user
func LookupMQM() (int, int, error) {
	mqm, err := user.Lookup("mqm")
	if err != nil {
		return -1, -1, err
	}
	mqmUID, err := strconv.Atoi(mqm.Uid)
	if err != nil {
		return -1, -1, err
	}
	mqmGID, err := strconv.Atoi(mqm.Gid)
	if err != nil {
		return -1, -1, err
	}
	return mqmUID, mqmGID, nil
}
