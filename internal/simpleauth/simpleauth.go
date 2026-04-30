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

//This is a developer only configuration and not recommended for production usage.

package simpleauth

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ibm-messaging/mq-container/internal/securityutility"
	"github.com/ibm-messaging/mq-container/internal/sensitive"
	"github.com/ibm-messaging/mq-container/pkg/logger"
)

const MQ_APP_PWD_ENV = "MQ_APP_PASSWORD"
const MQ_APP_PWD_SECURE_ENV = "MQ_APP_PASSWORD_SECURE"
const MQ_ADMIN_PWD_ENV = "MQ_ADMIN_PASSWORD"
const MQ_ADMIN_PWD_SECURE_ENV = "MQ_ADMIN_PASSWORD_SECURE"
const MQ_CONNAUTH_USE_HTP_ENV = "MQ_CONNAUTH_USE_HTP"

// #nosec G101
const MQ_APP_USER_SECRET_PATH = "/run/secrets/mqAppPassword"

// #nosec G101
const MQ_ADMIN_USER_SECRET_PATH = "/run/secrets/mqAdminPassword"

// IsEnabled will return a boolean value if the MQ_CONNAUTH_USER_HTP_ENV is set to true and if the app/admin
// user passwords are set as environment variables or set as secrets
func IsEnabled() bool {
	mqSimpleAuthEnabled := false
	enableHtPwd, set := os.LookupEnv(MQ_CONNAUTH_USE_HTP_ENV)
	adminPassword, adminPwdSet := os.LookupEnv(MQ_ADMIN_PWD_ENV)
	adminSecret, adminSecretSet := os.LookupEnv(MQ_ADMIN_PWD_SECURE_ENV)
	appPassword, appPwdSet := os.LookupEnv(MQ_APP_PWD_ENV)
	appSecret, appSecretSet := os.LookupEnv(MQ_APP_PWD_SECURE_ENV)

	if set && strings.EqualFold(enableHtPwd, "true") &&
		(adminPwdSet && len(strings.TrimSpace(adminPassword)) > 0 || appPwdSet && len(strings.TrimSpace(appPassword)) > 0 ||
			appSecretSet && len(strings.TrimSpace(appSecret)) > 0 || adminSecretSet && len(strings.TrimSpace(adminSecret)) > 0) {
		mqSimpleAuthEnabled = true
	}
	return mqSimpleAuthEnabled
}

// CheckForPasswords checks if the user has provided the app & admin user passwords via the environment variable
// or via the secrets. The secrets will be in /run/secrets path
func CheckForPasswords(log *logger.Logger) error {
	adminPassword, adminPwdSet := os.LookupEnv(MQ_ADMIN_PWD_ENV)
	appPassword, appPwdSet := os.LookupEnv(MQ_APP_PWD_ENV)

	// Setting the Admin Password Block, where the precedence is first given to secrets, and if both secrets and environment variables are present, appropriate deprecation warning is logged
	if _, err := os.Stat(MQ_ADMIN_USER_SECRET_PATH); err == nil {
		// First check if the environment variable has also been set. If yes, print the deprecation message.
		if adminPwdSet && len(strings.TrimSpace(adminPassword)) > 0 {
			log.Printf("Environment variable MQ_ADMIN_PASSWORD and the file /run/secrets/mqAdminPassword are both present. MQ_ADMIN_PASSWORD is deprecated, will be ignored, and should be removed.")
		}

		// Continue with the mounted secrets taking precedence to set the Admin Password
		encodedAdminSecret, err := readMQSecrets(MQ_ADMIN_USER_SECRET_PATH)
		if err != nil {
			return fmt.Errorf("encoding mqAdminPassword secret for web server failed with error %v", err)
		}

		if len(encodedAdminSecret) > 0 {
			err = os.Setenv(MQ_ADMIN_PWD_SECURE_ENV, encodedAdminSecret)
			if err != nil {
				return fmt.Errorf("setting encoded admin user password to environment variable failed with error %v", err)
			}
		}
	} else if adminPwdSet && len(strings.TrimSpace(adminPassword)) > 0 { // Falling back to environment variables if secrets have not been mounted
		adminPasswordSensitive := sensitive.New([]byte(adminPassword))
		encodedAdminPassword, err := securityutility.EncodeSecrets(adminPasswordSensitive, false)
		if err != nil {
			return fmt.Errorf("encoding Admin password for web server failed with error %v", err)
		}
		err = os.Setenv(MQ_ADMIN_PWD_SECURE_ENV, encodedAdminPassword)
		if err != nil {
			return fmt.Errorf("setting encoded admin user password to environment variable failed with error %v", err)
		}
		log.Printf("Environment variable MQ_ADMIN_PASSWORD is deprecated, use secrets to set the passwords")
	}

	// Setting the App Password Block, where the precedence is first given to secrets, and if both secrets and environment variables are present, appropriate deprecation warning is logged
	if _, err := os.Stat(MQ_APP_USER_SECRET_PATH); err == nil {
		// First check if the environment variable has also been set. If yes, print the deprecation message.
		if appPwdSet && len(strings.TrimSpace(appPassword)) > 0 {
			log.Printf("Environment variable MQ_APP_PASSWORD and the file /run/secrets/mqAppPassword are both present. MQ_APP_PASSWORD is deprecated, will be ignored, and should be removed.")
		}

		// Continue with the mounted secrets taking precedence to set the App Password
		encodedAppSecret, err := readMQSecrets(MQ_APP_USER_SECRET_PATH)
		if err != nil {
			return fmt.Errorf("encoding mqAppPassword secret for web server failed with error %v", err)

		}

		if len(encodedAppSecret) > 0 {
			err = os.Setenv(MQ_APP_PWD_SECURE_ENV, encodedAppSecret)
			if err != nil {
				return fmt.Errorf("setting encoded app user password to environment variable failed with error %v", err)
			}
		}
	} else if appPwdSet && len(strings.TrimSpace(appPassword)) > 0 { // Falling back to environment variables if secrets not mounted
		appPasswordSensitive := sensitive.New([]byte(appPassword))
		encodedAppPassword, err := securityutility.EncodeSecrets(appPasswordSensitive, false)
		if err != nil {
			return fmt.Errorf("encoding App password for web server failed with error %v", err)
		}
		err = os.Setenv(MQ_APP_PWD_SECURE_ENV, encodedAppPassword)
		if err != nil {
			return fmt.Errorf("setting encoded app user password to environment variable failed with error %v", err)
		}
		log.Printf("Environment variable MQ_APP_PASSWORD is deprecated, use secrets to set the passwords")
	}

	return nil
}

// readMQSecrets takes the secret file as an input and encodes the secret and returns an encoded password
func readMQSecrets(secretName string) (string, error) {
	passwordBuf, err := os.ReadFile(filepath.Clean(secretName))
	if err != nil {
		return "", err
	}
	passwordSensitive := sensitive.New(passwordBuf)
	if passwordSensitive.Len() > 256 {
		err = fmt.Errorf("the length of the password cannot be more than 256 characters, length of the password was %v", passwordSensitive.Len())
		return "", err
	}
	encodedPassword, err := securityutility.EncodeSecrets(passwordSensitive, false)
	if err != nil {
		return "", err
	}
	return encodedPassword, nil
}
