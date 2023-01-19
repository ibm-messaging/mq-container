/*
© Copyright IBM Corporation 2023

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
package fips

import (
	"os"
	"strings"

	"github.com/ibm-messaging/mq-container/internal/command"
	"github.com/ibm-messaging/mq-container/pkg/logger"
)

var (
	FIPSEnabledType int
)

// FIPS has been turned off either because OS is not FIPS enabled or
// MQ_ENABLE_FIPS environment variable is set to "false"
const FIPS_ENABLED_OFF = 0

// FIPS is turned ON
const FIPS_ENABLED_ON = 1

// FIPS enabled at operating system level
const FIPS_ENABLED_PLATFORM = 1

// FIPS enabled via environment variable
const FIPS_ENABLED_ENV_VAR = 2

// Get FIPS enabled type.
func ProcessFIPSType(logs *logger.Logger) {
	// Run "sysctl crypto.fips_enabled" command to determine if FIPS has been enabled
	// on OS.
	FIPSEnabledType = FIPS_ENABLED_OFF

	out, _, err := command.Run("sysctl", "crypto.fips_enabled")
	if err == nil {
		// Check the output of the command for expected output
		if strings.Contains(out, "crypto.fips_enabled = 1") {
			FIPSEnabledType = FIPS_ENABLED_PLATFORM
		}
	}

	// Check if we have been asked to override FIPS cryptography
	fipsOverride, fipsOverrideSet := os.LookupEnv("MQ_ENABLE_FIPS")
	if fipsOverrideSet {
		if strings.EqualFold(fipsOverride, "false") || strings.EqualFold(fipsOverride, "0") {
			FIPSEnabledType = FIPS_ENABLED_OFF
		} else if strings.EqualFold(fipsOverride, "true") || strings.EqualFold(fipsOverride, "1") {
			// This is the case where OS is not FIPS compliant but we have been asked to run MQ
			// queue manager, web server in FIPS mode. This case can be used when running docker tests.
			FIPSEnabledType = FIPS_ENABLED_ENV_VAR
		} else if strings.EqualFold(fipsOverride, "auto") {
			// This is the default case. Leave it to the OS default as determine above
		} else {
			// We don't recognise the value specified. Log a warning and carry on.
			if logs != nil {
				logs.Printf("Invalid value '%s' was specified for MQ_ENABLE_FIPS. The value has been ignored.\n", fipsOverride)
			}
		}
	}
}

func IsFIPSEnabled() bool {
	return FIPSEnabledType > FIPS_ENABLED_OFF
}

// Log a message on the console to indicate FIPS certified
// cryptography being used.
func PostInit(log *logger.Logger) {
	message := "FIPS cryptography is not enabled."
	if FIPSEnabledType == FIPS_ENABLED_PLATFORM {
		message = "FIPS cryptography is enabled. FIPS cryptography setting on the host is 'true'."
	} else if FIPSEnabledType == FIPS_ENABLED_ENV_VAR {
		message = "FIPS cryptography is enabled. FIPS cryptography setting on the host is 'false'."
	}

	log.Println(message)
}
