/*
© Copyright IBM Corporation 2024, 2026

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

// Package securityutility contains code to use the securityUtility tool from /opt/mqm/web/bin directory
// to encode passwords and generate AES keys.
package securityutility

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/ibm-messaging/mq-container/internal/sensitive"
)

const (
	// Path to the AES key file
	aesKeyFile = "/dev/shm/liberty-aes-key.xml"
	// Path to the security utility wrapper script
	securityUtilityWrapper = "/usr/local/bin/security-utility.sh"
	// Path to the securityUtility binary
	securityUtilityPath = "/opt/mqm/web/bin/securityUtility"
	// Java home directory for running securityUtility
	javaHome = "/opt/mqm/java/jre64/jre"
	// Minimum length for secrets/passwords
	minSecretLength = 16
	// Maximum length for secrets/passwords
	maxSecretLength = 256
)

// EncodeSecrets takes a secret/password as an input. If useAES is true, the AES key will be used to encrypt
// the password using securityUtility. If useAES is false, the password will be hashed using securityUtility.
// The encoded password or an error is returned.
func EncodeSecrets(secret *sensitive.Sensitive, useAES bool) (string, error) {
	if err := validateSecurityUtility(); err != nil {
		return "", err
	}

	if secret.Len() > maxSecretLength {
		return "", fmt.Errorf("length of password is greater than the maximum length of %d characters, actual length is %d", maxSecretLength, secret.Len())
	}

	// Select the appropriate command arguments based on encoding type
	var args []string
	if useAES {
		args = []string{"encode", "--encoding=aes", "--aesConfigFile=" + aesKeyFile}
	} else {
		args = []string{"encode", "--encoding=hash"}
	}

	// Execute the securityUtility command
	out, err := executeSecurityUtility(args, secret)
	if err != nil {
		return "", fmt.Errorf("securityUtility encode failed: %w, output: %s", err, string(out))
	}

	// Parse the output to extract the encoded secret
	encodedSecret, err := extractEncodedSecret(string(out))
	if err != nil {
		return "", fmt.Errorf("failed to extract encoded secret from output: %w, output: %s", err, string(out))
	}

	return strings.TrimSpace(encodedSecret), nil
}

// GenerateLibertyAESKeyFile generates the AES key file for Liberty from the initialKey.
// The key file is created at /dev/shm/liberty-aes-key.xml.
// If the file already exists (e.g., from a previous container run in the same pod),
// it will be removed before generating a new one.
func GenerateLibertyAESKeyFile(initialKey *sensitive.Sensitive) error {
	if err := validateSecurityUtility(); err != nil {
		return err
	}

	if initialKey.Len() < minSecretLength {
		return fmt.Errorf("length of initialKey is less than the minimum length of %d characters, actual length is %d", minSecretLength, initialKey.Len())
	}

	if initialKey.Len() > maxSecretLength {
		return fmt.Errorf("length of initialKey is greater than the maximum length of %d characters, actual length is %d", maxSecretLength, initialKey.Len())
	}

	// Remove existing AES key file if it exists (e.g., from a previous container run)
	// This handles the case where /dev/shm is shared at pod level and persists across container restarts
	if _, err := os.Stat(aesKeyFile); err == nil {
		if err := os.Remove(aesKeyFile); err != nil {
			return fmt.Errorf("failed to remove existing AES key file at %s: %w", aesKeyFile, err)
		}
	}

	// Build the --key= argument with the secret
	keyArg := sensitive.New([]byte("--key="))
	if err := keyArg.Append(initialKey); err != nil {
		return fmt.Errorf("failed to append secret to key argument: %w", err)
	}

	// Execute the securityUtility command
	out, err := executeSecurityUtility([]string{"generateAESKey", "--createConfigFile=" + aesKeyFile}, keyArg)
	if err != nil {
		return fmt.Errorf("securityUtility generateAESKey failed: %w, output: %s", err, string(out))
	}

	// Verify the key file was created
	if _, err := os.Stat(aesKeyFile); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("failed to create AES key file at %s: %s", aesKeyFile, string(out))
		}
		return fmt.Errorf("failed to verify AES key file at %s: %w", aesKeyFile, err)
	}

	return nil
}

// validateSecurityUtility checks if both the security utility wrapper script and
// the underlying securityUtility binary exist and are accessible.
func validateSecurityUtility() error {
	// Check wrapper script
	if _, err := os.Stat(securityUtilityWrapper); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("security utility wrapper not found at %s", securityUtilityWrapper)
		}
		return fmt.Errorf("failed to access security utility wrapper at %s: %w", securityUtilityWrapper, err)
	}

	// Check underlying securityUtility binary
	if _, err := os.Stat(securityUtilityPath); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("securityUtility not found at %s", securityUtilityPath)
		}
		return fmt.Errorf("failed to access securityUtility at %s: %w", securityUtilityPath, err)
	}

	return nil
}

// executeSecurityUtility runs the securityUtility wrapper script with the provided arguments and secret.
// The secret is passed as a command parameter (not interpolated into a shell command), which eliminates
// shell injection vulnerabilities. The wrapper script handles environment setup and executes securityUtility.
func executeSecurityUtility(args []string, secret *sensitive.Sensitive) ([]byte, error) {
	// Build the full argument list: wrapper script args + secret
	cmdArgs := make([]string, len(args)+1)
	copy(cmdArgs, args)
	cmdArgs[len(args)] = secret.String()

	// Execute the wrapper script with arguments passed as parameters
	// No shell escaping needed - arguments are passed directly to the script
	// #nosec G204 -- securityUtilityWrapper is a hardcoded constant path to a trusted wrapper script.
	// User input is only passed as command arguments (cmdArgs), never as part of the command path.
	// The wrapper script uses "$@" to safely pass arguments without shell interpretation.
	cmd := exec.Command(securityUtilityWrapper, cmdArgs...)
	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, fmt.Sprintf("JAVA_HOME=%s", javaHome))

	return cmd.CombinedOutput()
}

// extractEncodedSecret parses the securityUtility output and extracts the encoded secret.
// When the JVM is in FIPS 140-2 mode and the IBMJCEPlusFIPS provider is used, additional
// messages may be displayed. This function filters those out and returns only the encoded secret.
func extractEncodedSecret(output string) (string, error) {
	cmdOutput := strings.Split(output, "\n")

	// Look for lines containing the encoded secret markers
	for _, line := range cmdOutput {
		if strings.Contains(line, "{aes}") || strings.Contains(line, "{hash}") {
			return line, nil
		}
	}

	return "", fmt.Errorf("securityUtility did not return an encoded secret")
}
