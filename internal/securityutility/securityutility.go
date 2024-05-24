/*
© Copyright IBM Corporation 2024

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

// Look for go standards for package comment
// Package securityUtility contains code to use securityUtility tool from opt/mqm/web/bin directory
// to encode the passwords
package securityutility

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// EncodeSecrets takes a secret/password as an input and encodes the password using securityUtility
// and returns the encoded password
func EncodeSecrets(secret string) (string, error) {
	_, err := os.Stat("/opt/mqm/web/bin/securityUtility")
	if err != nil && os.IsNotExist(err) {
		return "", err
	}

	if len(secret) > 256 {
		return "", fmt.Errorf("length of password is greater than the maximum length of 256 characters, length of password is %d", len(secret))
	}
	// Set the java environment required for running securityUtility tool and then run the securityUtility tool
	// to encode the password using "aes" encoding
	// #nosec G204
	cmd := exec.Command("/bin/sh", "-c", "source setmqenv -s;/opt/mqm/web/bin/server; /opt/mqm/web/bin/securityUtility encode --encoding=aes "+secret)
	cmd.Env = os.Environ()

	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", err
	}
	encodedSecret := ""
	cmdOutput := strings.Split(string(out), "\n")
	// When the JVM is in FIPS 140-2 mode and the IBMJCEPlusFIPS provider is used the following message is displayed
	// The IBMJCEPlusFIPS provider is configured for FIPS 140-2. Please note that the 140-2 configuration may be removed in the future.
	// Hence read only the encoded password and ignore the above message
	for _, line := range cmdOutput {
		if strings.Contains(line, "{aes}") {
			encodedSecret = line
		}
	}
	return strings.TrimSpace(encodedSecret), nil
}
