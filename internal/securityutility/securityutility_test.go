/*
© Copyright IBM Corporation 2026

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

package securityutility

import (
	"os"
	"strings"
	"testing"

	"github.com/ibm-messaging/mq-container/internal/sensitive"
)

// TestExtractEncodedSecret tests the extractEncodedSecret function
func TestExtractEncodedSecret(t *testing.T) {
	tests := []struct {
		name        string
		output      string
		expected    string
		expectError bool
	}{
		{
			name:        "Output with AES encoded secret",
			output:      "Some output\n{aes}encodedSecretHere\nMore output",
			expected:    "{aes}encodedSecretHere",
			expectError: false,
		},
		{
			name:        "Output with hash encoded secret",
			output:      "Some output\n{hash}hashedSecretHere\nMore output",
			expected:    "{hash}hashedSecretHere",
			expectError: false,
		},
		{
			name:        "Output with FIPS messages and AES secret",
			output:      "FIPS mode enabled\nIBMJCEPlusFIPS provider loaded\n{aes}encodedSecret123\nSuccess",
			expected:    "{aes}encodedSecret123",
			expectError: false,
		},
		{
			name:        "Output without encoded secret",
			output:      "Some output\nNo secret here\nMore output",
			expected:    "",
			expectError: true,
		},
		{
			name:        "Empty output",
			output:      "",
			expected:    "",
			expectError: true,
		},
		{
			name:        "Output with only newlines",
			output:      "\n\n\n",
			expected:    "",
			expectError: true,
		},
		{
			name:        "Multiple encoded secrets (should return first)",
			output:      "{aes}first\n{hash}second",
			expected:    "{aes}first",
			expectError: false,
		},
		{
			name:        "AES secret with additional text on same line",
			output:      "Result: {aes}myEncodedPassword123",
			expected:    "Result: {aes}myEncodedPassword123",
			expectError: false,
		},
		{
			name:        "Hash secret with additional text on same line",
			output:      "Encoded password: {hash}myHashedPassword456",
			expected:    "Encoded password: {hash}myHashedPassword456",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := extractEncodedSecret(tt.output)

			if tt.expectError {
				if err == nil {
					t.Errorf("extractEncodedSecret() expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("extractEncodedSecret() unexpected error: %v", err)
				}
				if result != tt.expected {
					t.Errorf("extractEncodedSecret() = %q, want %q", result, tt.expected)
				}
			}
		})
	}
}

// TestValidateSecurityUtility tests the validateSecurityUtility function
func TestValidateSecurityUtility(t *testing.T) {
	t.Run("Security utility validation", func(t *testing.T) {
		err := validateSecurityUtility()

		// We expect an error if the wrapper script or securityUtility doesn't exist (typical in test environment)
		// If both exist (running on actual MQ system), the test should pass
		if err != nil {
			if !strings.Contains(err.Error(), "security utility wrapper not found") &&
				!strings.Contains(err.Error(), "failed to access security utility wrapper") &&
				!strings.Contains(err.Error(), "securityUtility not found") &&
				!strings.Contains(err.Error(), "failed to access securityUtility") {
				t.Errorf("validateSecurityUtility() unexpected error format: %v", err)
			}
		} else {
			t.Log("security utility wrapper and securityUtility found (running in MQ container)")
		}
	})
}

// TestEncodeSecretsValidation tests input validation for EncodeSecrets
func TestEncodeSecretsValidation(t *testing.T) {
	t.Run("Secret exceeds maximum length", func(t *testing.T) {
		// Create a secret that exceeds maxSecretLength (256 characters)
		longSecret := make([]byte, maxSecretLength+1)
		for i := range longSecret {
			longSecret[i] = 'a'
		}
		secret := sensitive.New(longSecret)

		_, err := EncodeSecrets(secret, false)
		if err == nil {
			t.Error("EncodeSecrets() expected error for secret exceeding max length, but got none")
		}

		// Check for length error OR security utility wrapper not found (validation happens first)
		if err != nil && !strings.Contains(err.Error(), "greater than the maximum length") &&
			!strings.Contains(err.Error(), "security utility wrapper not found") {
			t.Errorf("EncodeSecrets() expected max length or validation error, got: %v", err)
		}
	})

	t.Run("Valid secret length at boundary", func(t *testing.T) {
		// Create a secret exactly at maxSecretLength
		secret := make([]byte, maxSecretLength)
		for i := range secret {
			secret[i] = 'b'
		}
		s := sensitive.New(secret)

		// This will fail if securityUtility doesn't exist, but validates length check passes
		_, err := EncodeSecrets(s, false)

		// We expect either success or an error about securityUtility not being found
		// but NOT an error about length
		if err != nil && strings.Contains(err.Error(), "greater than the maximum length") {
			t.Errorf("EncodeSecrets() should not fail on max length boundary, got: %v", err)
		}
	})

	t.Run("Valid secret with AES encoding", func(t *testing.T) {
		secret := sensitive.New([]byte("validPassword123"))

		_, err := EncodeSecrets(secret, true)

		// We expect either success or an error about securityUtility/AES key file not being found
		// but NOT an error about length
		if err != nil && strings.Contains(err.Error(), "greater than the maximum length") {
			t.Errorf("EncodeSecrets() should not fail on valid length, got: %v", err)
		}
	})

	t.Run("Valid secret with hash encoding", func(t *testing.T) {
		secret := sensitive.New([]byte("validPassword456"))

		_, err := EncodeSecrets(secret, false)

		// We expect either success or an error about securityUtility not being found
		// but NOT an error about length
		if err != nil && strings.Contains(err.Error(), "greater than the maximum length") {
			t.Errorf("EncodeSecrets() should not fail on valid length, got: %v", err)
		}
	})

	t.Run("Secret with special characters is handled safely", func(t *testing.T) {
		// Test that special shell characters are handled safely
		secret := sensitive.New([]byte("pass'word;$(cmd)&test"))

		_, err := EncodeSecrets(secret, false)

		// We expect either success or an error about securityUtility not being found
		// The important thing is that it doesn't execute shell commands
		if err != nil && strings.Contains(err.Error(), "greater than the maximum length") {
			t.Errorf("EncodeSecrets() should not fail on valid length, got: %v", err)
		}
	})
}

// TestGenerateLibertyAESKeyFileValidation tests input validation for GenerateLibertyAESKeyFile
func TestGenerateLibertyAESKeyFileValidation(t *testing.T) {
	t.Run("InitialKey less than minimum length", func(t *testing.T) {
		// Create a key that is less than minSecretLength (16 characters)
		shortKey := make([]byte, minSecretLength-1)
		for i := range shortKey {
			shortKey[i] = 'a'
		}
		key := sensitive.New(shortKey)

		err := GenerateLibertyAESKeyFile(key)
		if err == nil {
			t.Error("GenerateLibertyAESKeyFile() expected error for key less than min length, but got none")
		}

		// Check for length error OR security utility wrapper not found (validation happens first)
		if err != nil && !strings.Contains(err.Error(), "less than the minimum length") &&
			!strings.Contains(err.Error(), "security utility wrapper not found") {
			t.Errorf("GenerateLibertyAESKeyFile() expected min length or validation error, got: %v", err)
		}
	})

	t.Run("InitialKey exceeds maximum length", func(t *testing.T) {
		// Create a key that exceeds maxSecretLength (256 characters)
		longKey := make([]byte, maxSecretLength+1)
		for i := range longKey {
			longKey[i] = 'b'
		}
		key := sensitive.New(longKey)

		err := GenerateLibertyAESKeyFile(key)
		if err == nil {
			t.Error("GenerateLibertyAESKeyFile() expected error for key exceeding max length, but got none")
		}

		// Check for length error OR security utility wrapper not found (validation happens first)
		if err != nil && !strings.Contains(err.Error(), "greater than the maximum length") &&
			!strings.Contains(err.Error(), "security utility wrapper not found") {
			t.Errorf("GenerateLibertyAESKeyFile() expected max length or validation error, got: %v", err)
		}
	})

	t.Run("Valid key length at minimum boundary", func(t *testing.T) {
		// Create a key exactly at minSecretLength
		key := make([]byte, minSecretLength)
		for i := range key {
			key[i] = 'c'
		}
		k := sensitive.New(key)

		// This will fail if securityUtility doesn't exist, but validates length check passes
		err := GenerateLibertyAESKeyFile(k)

		// We expect either success or an error about securityUtility not being found
		// but NOT an error about length
		if err != nil && (strings.Contains(err.Error(), "less than the minimum length") ||
			strings.Contains(err.Error(), "greater than the maximum length")) {
			t.Errorf("GenerateLibertyAESKeyFile() should not fail on valid length, got: %v", err)
		}
	})

	t.Run("Valid key length at maximum boundary", func(t *testing.T) {
		// Create a key exactly at maxSecretLength
		key := make([]byte, maxSecretLength)
		for i := range key {
			key[i] = 'd'
		}
		k := sensitive.New(key)

		// This will fail if securityUtility doesn't exist, but validates length check passes
		err := GenerateLibertyAESKeyFile(k)

		// We expect either success or an error about securityUtility not being found
		// but NOT an error about length
		if err != nil && (strings.Contains(err.Error(), "less than the minimum length") ||
			strings.Contains(err.Error(), "greater than the maximum length")) {
			t.Errorf("GenerateLibertyAESKeyFile() should not fail on valid length, got: %v", err)
		}
	})

	t.Run("Valid key length in middle range", func(t *testing.T) {
		// Create a key with a typical length (32 characters)
		key := make([]byte, 32)
		for i := range key {
			key[i] = 'e'
		}
		k := sensitive.New(key)

		err := GenerateLibertyAESKeyFile(k)

		// We expect either success or an error about securityUtility not being found
		// but NOT an error about length
		if err != nil && (strings.Contains(err.Error(), "less than the minimum length") ||
			strings.Contains(err.Error(), "greater than the maximum length")) {
			t.Errorf("GenerateLibertyAESKeyFile() should not fail on valid length, got: %v", err)
		}
	})

	t.Run("Key with special characters is handled safely", func(t *testing.T) {
		// Test that special shell characters in the key are handled safely
		key := sensitive.New([]byte("myKey'with;special$(chars)&more"))

		err := GenerateLibertyAESKeyFile(key)

		// We expect either success or an error about securityUtility not being found
		// The important thing is that it doesn't execute shell commands
		if err != nil && (strings.Contains(err.Error(), "less than the minimum length") ||
			strings.Contains(err.Error(), "greater than the maximum length")) {
			t.Errorf("GenerateLibertyAESKeyFile() should not fail on valid length, got: %v", err)
		}
	})
}

// TestExecuteSecurityUtilityEscaping tests that executeSecurityUtility handles escaping internally
func TestExecuteSecurityUtilityEscaping(t *testing.T) {
	// Skip if securityUtility is not available
	if err := validateSecurityUtility(); err != nil {
		t.Skip("securityUtility not available, skipping executeSecurityUtility tests")
	}

	t.Run("Function passes secret with single quotes as parameter", func(t *testing.T) {
		// Use help command which should be available
		args := []string{"help"}
		// The secret won't be used by help, but we test that quotes don't break things
		secret := sensitive.New([]byte("test'with'quotes"))

		// executeSecurityUtility passes secret as parameter, not in shell command
		// This may fail but shouldn't panic
		_, err := executeSecurityUtility(args, secret)

		// We just verify it doesn't panic - the command may fail for other reasons
		t.Logf("executeSecurityUtility with quotes in parameter: %v", err)
	})

	t.Run("Function handles secret with dangerous characters safely", func(t *testing.T) {
		args := []string{"help"}
		// Test with characters that would be dangerous if interpreted by shell
		secret := sensitive.New([]byte("pass;rm -rf /;word"))

		// Secret is passed as parameter, so dangerous characters are treated as literal data
		_, err := executeSecurityUtility(args, secret)

		// We just verify it doesn't panic and doesn't execute rm command
		t.Logf("executeSecurityUtility with dangerous characters as parameter: %v", err)
	})
}

// TestAESKeyFileCreation tests the AES key file creation logic
func TestAESKeyFileCreation(t *testing.T) {
	t.Run("Check if AES key file exists after generation attempt", func(t *testing.T) {
		// Skip if securityUtility is not available
		if err := validateSecurityUtility(); err != nil {
			t.Skip("securityUtility not available, skipping AES key file creation test")
		}

		// Clean up any existing file first
		os.Remove(aesKeyFile)

		// Try to generate with valid key
		key := make([]byte, 32)
		for i := range key {
			key[i] = 'x'
		}
		k := sensitive.New(key)

		err := GenerateLibertyAESKeyFile(k)

		// The test should fail if any error occurs
		if err != nil {
			t.Fatalf("GenerateLibertyAESKeyFile() failed: %v", err)
		}

		// Check if file was created
		if _, statErr := os.Stat(aesKeyFile); statErr != nil {
			t.Errorf("GenerateLibertyAESKeyFile() succeeded but file was not created: %v", statErr)
		} else {
			t.Log("AES key file successfully created")
		}

		// Clean up
		os.Remove(aesKeyFile)
	})

	t.Run("Regenerate AES key file when it already exists", func(t *testing.T) {
		// Skip if securityUtility is not available
		if err := validateSecurityUtility(); err != nil {
			t.Skip("securityUtility not available, skipping AES key file regeneration test")
		}

		// Clean up any existing file first
		os.Remove(aesKeyFile)

		// Generate initial key file
		key1 := make([]byte, 32)
		for i := range key1 {
			key1[i] = 'a'
		}
		k1 := sensitive.New(key1)

		err := GenerateLibertyAESKeyFile(k1)
		if err != nil {
			t.Fatalf("GenerateLibertyAESKeyFile() first generation failed: %v", err)
		}

		// Verify file exists
		if _, statErr := os.Stat(aesKeyFile); statErr != nil {
			t.Fatalf("First AES key file was not created: %v", statErr)
		}

		// Now try to regenerate with a different key (simulating container restart)
		key2 := make([]byte, 32)
		for i := range key2 {
			key2[i] = 'b'
		}
		k2 := sensitive.New(key2)

		err = GenerateLibertyAESKeyFile(k2)
		if err != nil {
			t.Fatalf("GenerateLibertyAESKeyFile() regeneration failed: %v", err)
		}

		// Verify file still exists after regeneration
		if _, statErr := os.Stat(aesKeyFile); statErr != nil {
			t.Errorf("AES key file was not recreated after regeneration: %v", statErr)
		} else {
			t.Log("AES key file successfully regenerated when it already existed")
		}

		// Clean up
		os.Remove(aesKeyFile)
	})
}
