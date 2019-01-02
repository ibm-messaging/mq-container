/*
© Copyright IBM Corporation 2018

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
	"os"
	"path/filepath"
	"strings"

	"github.com/ibm-messaging/mq-container/internal/command"
)

// Verifies that we are the main or only instance of this program
func verifySingleProcess() error {
	programName, err := determineExecutable()
	if err != nil {
		return fmt.Errorf("Failed to determine name of this program - %v", err)
	}

	// Verify that there is only one runmqserver
	_, err = verifyOnlyOne(programName)
	if err != nil {
		return fmt.Errorf("You cannot run more than one instance of this program")
	}

	return nil
}

// Verifies that there is only one instance running of the given program name.
func verifyOnlyOne(programName string) (int, error) {
	out, _, _ := command.Run("ps", "-e", "--format", "cmd")
	//if this goes wrong then assume we are the only one
	numOfProg := strings.Count(out, programName)
	if numOfProg != 1 {
		return numOfProg, fmt.Errorf("Expected there to be only 1 instance of %s but found %d", programName, numOfProg)
	}
	return numOfProg, nil
}

// Determines the name of the currently running executable.
func determineExecutable() (string, error) {
	file, err := os.Executable()
	if err != nil {
		return "", err
	}

	_, exec := filepath.Split(file)
	return exec, nil
}
