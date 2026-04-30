/*
© Copyright IBM Corporation 2020, 2026

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

package simpleauth

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/ibm-messaging/mq-container/pkg/logger"
)

// TestIsEnabled tests the IsEnabled() function with various scenarios
func TestIsEnabled(t *testing.T) {
	t.Run("MQ_CONNAUTH_USE_HTP_ENV env var set to true and the app and admin passwords are set via env vars", func(t *testing.T) {
		t.Setenv("MQ_ADMIN_PASSWORD", "adminpassw0rd")
		t.Setenv("MQ_APP_PASSWORD", "apppassw0rd")
		t.Setenv("MQ_CONNAUTH_USE_HTP", "true")

		result := IsEnabled()
		expected := true
		if result != expected {
			t.Errorf("expected %t, but got %t", expected, result)
		}
	})

	t.Run("MQ_CONNAUTH_USE_HTP_ENV env var set to true and the app and admin passwords are set via secrets", func(t *testing.T) {
		t.Setenv(MQ_ADMIN_PWD_SECURE_ENV, "dummySecretsPath")
		t.Setenv(MQ_APP_PWD_SECURE_ENV, "dummySecretsPath")
		t.Setenv("MQ_CONNAUTH_USE_HTP", "true")

		result := IsEnabled()
		expected := true
		if result != expected {
			t.Errorf("expected %t, but got %t", expected, result)
		}

	})

	t.Run("MQ_CONNAUTH_USE_HTP_ENV env var set to false", func(t *testing.T) {
		t.Setenv("MQ_CONNAUTH_USE_HTP", "false")

		result := IsEnabled()
		expected := false
		if result != expected {
			t.Errorf("expected %t, but got %t", expected, result)
		}
	})
}

// TestCheckForPasswords tests the CheckForPasswords() function with various scenarios
// Note: Since AES encoding is randomized, we can only verify that:
// 1. Something has been written to the environment variable
// 2. Expected log messages appear (or don't appear) based on the code path taken
// Full end-to-end verification of secrets would require Docker integration tests
func TestCheckForPasswords(t *testing.T) {
	// Skip if securityUtility is not available
	if !isSecurityUtilityAvailable() {
		t.Skip("securityUtility not available, skipping TestCheckForPasswords tests")
	}

	t.Run("Admin password set via secret file", func(t *testing.T) {
		var buf bytes.Buffer

		// Create a test logger
		testLogger, err := logger.NewLogger(&buf, false, false, "TestCheckForPasswords")
		if err != nil {
			t.Fatalf("failed to create test logger: %v", err)
		}

		// Create the secret file
		testFile := MQ_ADMIN_USER_SECRET_PATH
		err = os.MkdirAll(filepath.Dir(testFile), 0755)
		if err != nil {
			t.Fatalf("failed to create secrets directory: %v", err)
		}
		err = os.WriteFile(testFile, []byte("secretPassw0rd"), 0600)
		if err != nil {
			t.Fatalf("failed to create test secret file: %v", err)
		}
		defer os.Remove(testFile)

		// Call CheckForPasswords
		err = CheckForPasswords(testLogger)
		if err != nil {
			t.Errorf("CheckForPasswords failed: %v", err)
		}

		// Verify that encoded password environment variable was set
		encodedPassword := os.Getenv(MQ_ADMIN_PWD_SECURE_ENV)
		if encodedPassword == "" {
			t.Error("expected MQ_ADMIN_PASSWORD_SECURE to be set, but it was empty")
		}

		// Verify NO deprecation warning appears in logs (secret file path was taken)
		if bytes.Contains(buf.Bytes(), []byte("deprecated")) {
			t.Error("did not expect deprecation warning when using secret file")
		}
	})

	t.Run("App password set via secret file - no deprecation warning", func(t *testing.T) {
		var buf bytes.Buffer

		// Create a test logger
		testLogger, err := logger.NewLogger(&buf, false, false, "TestCheckForPasswords")
		if err != nil {
			t.Fatalf("failed to create test logger: %v", err)
		}

		// Create the secret file
		testFile := MQ_APP_USER_SECRET_PATH
		err = os.MkdirAll(filepath.Dir(testFile), 0755)
		if err != nil {
			t.Fatalf("failed to create secrets directory: %v", err)
		}
		err = os.WriteFile(testFile, []byte("secretPassw0rd"), 0600)
		if err != nil {
			t.Fatalf("failed to create test secret file: %v", err)
		}
		defer os.Remove(testFile)

		// Call CheckForPasswords
		err = CheckForPasswords(testLogger)
		if err != nil {
			t.Errorf("CheckForPasswords failed: %v", err)
		}

		// Verify that encoded password environment variable was set
		encodedPassword := os.Getenv(MQ_APP_PWD_SECURE_ENV)
		if encodedPassword == "" {
			t.Error("expected MQ_APP_PASSWORD_SECURE to be set, but it was empty")
		}

		// Verify NO deprecation warning appears in logs (secret file path was taken)
		if bytes.Contains(buf.Bytes(), []byte("deprecated")) {
			t.Error("did not expect deprecation warning when using secret file")
		}
	})

	t.Run("Both Admin and App passwords set via secret files - no deprecation warnings", func(t *testing.T) {
		var buf bytes.Buffer

		// Create a test logger
		testLogger, err := logger.NewLogger(&buf, false, false, "TestCheckForPasswords")
		if err != nil {
			t.Fatalf("failed to create test logger: %v", err)
		}

		// Create the admin secret file
		adminFile := MQ_ADMIN_USER_SECRET_PATH
		err = os.MkdirAll(filepath.Dir(adminFile), 0755)
		if err != nil {
			t.Fatalf("failed to create secrets directory: %v", err)
		}
		err = os.WriteFile(adminFile, []byte("adminSecretPassw0rd"), 0600)
		if err != nil {
			t.Fatalf("failed to create admin secret file: %v", err)
		}
		defer os.Remove(adminFile)

		// Create the app secret file
		appFile := MQ_APP_USER_SECRET_PATH
		err = os.WriteFile(appFile, []byte("appSecretPassw0rd"), 0600)
		if err != nil {
			t.Fatalf("failed to create app secret file: %v", err)
		}
		defer os.Remove(appFile)

		// Call CheckForPasswords
		err = CheckForPasswords(testLogger)
		if err != nil {
			t.Errorf("CheckForPasswords failed: %v", err)
		}

		// Verify that both encoded password environment variables were set
		adminEncodedPassword := os.Getenv(MQ_ADMIN_PWD_SECURE_ENV)
		if adminEncodedPassword == "" {
			t.Error("expected MQ_ADMIN_PASSWORD_SECURE to be set, but it was empty")
		}

		appEncodedPassword := os.Getenv(MQ_APP_PWD_SECURE_ENV)
		if appEncodedPassword == "" {
			t.Error("expected MQ_APP_PASSWORD_SECURE to be set, but it was empty")
		}

		// Verify NO deprecation warnings appear in logs (secret file path was taken)
		if bytes.Contains(buf.Bytes(), []byte("deprecated")) {
			t.Error("did not expect deprecation warning when using secret files")
		}
	})

	t.Run("Both Secrets and Environment variables used for Admin Password", func(t *testing.T) {
		var buf bytes.Buffer

		// Create a test logger
		testLogger, err := logger.NewLogger(&buf, false, false, "TestCheckForPasswords")
		if err != nil {
			t.Fatalf("failed to create test logger: %v", err)
		}

		// Setting the environment variable
		t.Setenv(MQ_ADMIN_PWD_ENV, "adminPassw0rd")

		// Create the secret file
		testFile := MQ_ADMIN_USER_SECRET_PATH
		err = os.MkdirAll(filepath.Dir(testFile), 0755)
		if err != nil {
			t.Fatalf("failed to create secrets directory: %v", err)
		}
		err = os.WriteFile(testFile, []byte("secretPassw0rd"), 0600)
		if err != nil {
			t.Fatalf("failed to create test secret file: %v", err)
		}
		defer os.Remove(testFile)

		// Call CheckForPasswords
		err = CheckForPasswords(testLogger)
		if err != nil {
			t.Errorf("CheckForPasswords failed: %v", err)
		}

		if !bytes.Contains(buf.Bytes(), []byte("Environment variable MQ_ADMIN_PASSWORD and the file /run/secrets/mqAdminPassword are both present. MQ_ADMIN_PASSWORD is deprecated, will be ignored, and should be removed.")) {
			t.Error("expected deprecation warning to appear in logs")
		}
	})

	t.Run("Both Secrets and Environment variables used for App Password", func(t *testing.T) {
		var buf bytes.Buffer

		// Create a test logger
		testLogger, err := logger.NewLogger(&buf, false, false, "TestCheckForPasswords")
		if err != nil {
			t.Fatalf("failed to create test logger: %v", err)
		}

		// Setting the environment variable
		t.Setenv(MQ_APP_PWD_ENV, "appPassw0rd")

		// Create the secret file
		testFile := MQ_APP_USER_SECRET_PATH
		err = os.MkdirAll(filepath.Dir(testFile), 0755)
		if err != nil {
			t.Fatalf("failed to create secrets directory: %v", err)
		}
		err = os.WriteFile(testFile, []byte("secretPassw0rd"), 0600)
		if err != nil {
			t.Fatalf("failed to create test secret file: %v", err)
		}
		defer os.Remove(testFile)

		// Call CheckForPasswords
		err = CheckForPasswords(testLogger)
		if err != nil {
			t.Errorf("CheckForPasswords failed: %v", err)
		}

		if !bytes.Contains(buf.Bytes(), []byte("Environment variable MQ_APP_PASSWORD and the file /run/secrets/mqAppPassword are both present. MQ_APP_PASSWORD is deprecated, will be ignored, and should be removed.")) {
			t.Error("expected deprecation warning to appear in logs")
		}
	})

	t.Run("Admin password set via environment variable - deprecation warning logged", func(t *testing.T) {
		var buf bytes.Buffer

		// Create a test logger to capture log output
		testLogger, err := logger.NewLogger(&buf, false, false, "TestCheckForPasswords")
		if err != nil {
			t.Fatalf("failed to create logger: %v", err)
		}

		// Set only the environment variable (no secret file)
		t.Setenv(MQ_ADMIN_PWD_ENV, "adminPassw0rd")

		// Ensure secret file does NOT exist
		os.Remove(MQ_ADMIN_USER_SECRET_PATH)

		// Call CheckForPasswords
		err = CheckForPasswords(testLogger)
		if err != nil {
			t.Errorf("CheckForPasswords failed: %v", err)
		}

		// Verify that encoded password environment variable was set (something was written)
		encodedPassword := os.Getenv(MQ_ADMIN_PWD_SECURE_ENV)
		if encodedPassword == "" {
			t.Error("expected MQ_ADMIN_PASSWORD_SECURE to be set, but it was empty")
		}

		// Verify deprecation warning appears in logs (env var path was taken)
		if !bytes.Contains(buf.Bytes(), []byte("MQ_ADMIN_PASSWORD is deprecated")) {
			t.Error("expected deprecation warning for MQ_ADMIN_PASSWORD, but it was not found in logs")
		}
	})

	t.Run("App password set via environment variable - deprecation warning logged", func(t *testing.T) {
		var buf bytes.Buffer

		// Create a test logger to capture log output
		testLogger, err := logger.NewLogger(&buf, false, false, "TestCheckForPasswords")
		if err != nil {
			t.Fatalf("failed to create logger: %v", err)
		}

		// Set only the environment variable (no secret file)
		t.Setenv(MQ_APP_PWD_ENV, "appPassw0rd")

		// Ensure secret file does NOT exist
		os.Remove(MQ_APP_USER_SECRET_PATH)

		// Call CheckForPasswords
		err = CheckForPasswords(testLogger)
		if err != nil {
			t.Errorf("CheckForPasswords failed: %v", err)
		}

		// Verify that encoded password environment variable was set (something was written)
		encodedPassword := os.Getenv(MQ_APP_PWD_SECURE_ENV)
		if encodedPassword == "" {
			t.Error("expected MQ_APP_PASSWORD_SECURE to be set, but it was empty")
		}

		// Verify deprecation warning appears in logs (env var path was taken)
		if !bytes.Contains(buf.Bytes(), []byte("MQ_APP_PASSWORD is deprecated")) {
			t.Error("expected deprecation warning for MQ_APP_PASSWORD, but it was not found in logs")
		}
	})

	t.Run("Admin password exceeds 256 characters - error thrown", func(t *testing.T) {
		var buf bytes.Buffer

		// Create a test logger
		testLogger, err := logger.NewLogger(&buf, false, false, "TestCheckForPasswords")
		if err != nil {
			t.Fatalf("failed to create test logger: %v", err)
		}

		// Create a password that exceeds 256 characters
		longPassword := make([]byte, 257)
		for i := range longPassword {
			longPassword[i] = 'a'
		}

		// Create the secret file with long password
		testFile := MQ_ADMIN_USER_SECRET_PATH
		err = os.MkdirAll(filepath.Dir(testFile), 0755)
		if err != nil {
			t.Fatalf("failed to create secrets directory: %v", err)
		}
		err = os.WriteFile(testFile, longPassword, 0600)
		if err != nil {
			t.Fatalf("failed to create test secret file: %v", err)
		}
		defer os.Remove(testFile)

		// Call CheckForPasswords - should return an error
		err = CheckForPasswords(testLogger)
		if err == nil {
			t.Error("expected error for password exceeding 256 characters, but got nil")
		}

		// Verify error message mentions the length constraint
		if err != nil && !bytes.Contains([]byte(err.Error()), []byte("256")) {
			t.Errorf("expected error message to mention 256 character limit, got: %v", err)
		}
	})

	t.Run("App password exceeds 256 characters - error thrown", func(t *testing.T) {
		var buf bytes.Buffer

		// Create a test logger
		testLogger, err := logger.NewLogger(&buf, false, false, "TestCheckForPasswords")
		if err != nil {
			t.Fatalf("failed to create test logger: %v", err)
		}

		// Create a password that exceeds 256 characters
		longPassword := make([]byte, 257)
		for i := range longPassword {
			longPassword[i] = 'b'
		}

		// Create the secret file with long password
		testFile := MQ_APP_USER_SECRET_PATH
		err = os.MkdirAll(filepath.Dir(testFile), 0755)
		if err != nil {
			t.Fatalf("failed to create secrets directory: %v", err)
		}
		err = os.WriteFile(testFile, longPassword, 0600)
		if err != nil {
			t.Fatalf("failed to create test secret file: %v", err)
		}
		defer os.Remove(testFile)

		// Call CheckForPasswords - should return an error
		err = CheckForPasswords(testLogger)
		if err == nil {
			t.Error("expected error for password exceeding 256 characters, but got nil")
		}

		// Verify error message mentions the length constraint
		if err != nil && !bytes.Contains([]byte(err.Error()), []byte("256")) {
			t.Errorf("expected error message to mention 256 character limit, got: %v", err)
		}
	})
}

// isSecurityUtilityAvailable checks if the security utility is available for testing.
func isSecurityUtilityAvailable() bool {
	// Check if both files exist
	if _, err := os.Stat("/usr/local/bin/security-utility.sh"); err != nil {
		return false
	}
	if _, err := os.Stat("/opt/mqm/web/bin/securityUtility"); err != nil {
		return false
	}
	return true
}
