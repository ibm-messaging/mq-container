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

package ha

import (
	"bytes"
	"embed"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/ibm-messaging/mq-container/pkg/logger"
)

//go:embed test_fixtures
var testFixtures embed.FS

func TestConfigFromEnv(t *testing.T) {

	tests := []struct {
		TestName  string
		env       map[string]string
		overrides testOverrides
		expected  haConfig
	}{
		{
			TestName: "Minimal config",
			env: map[string]string{
				"HOSTNAME":                                    "minimal-config",
				"MQ_NATIVE_HA_INSTANCE_0_NAME":                "minimal-config-instance0",
				"MQ_NATIVE_HA_INSTANCE_1_NAME":                "minimal-config-instance1",
				"MQ_NATIVE_HA_INSTANCE_2_NAME":                "minimal-config-instance2",
				"MQ_NATIVE_HA_INSTANCE_0_REPLICATION_ADDRESS": "minimal-config-instance0(9145)",
				"MQ_NATIVE_HA_INSTANCE_1_REPLICATION_ADDRESS": "minimal-config-instance1(9145)",
				"MQ_NATIVE_HA_INSTANCE_2_REPLICATION_ADDRESS": "minimal-config-instance2(9145)",
			},
			expected: haConfig{
				Name: "minimal-config",
				Instances: [3]haInstance{
					{"minimal-config-instance0", "minimal-config-instance0(9145)"},
					{"minimal-config-instance1", "minimal-config-instance1(9145)"},
					{"minimal-config-instance2", "minimal-config-instance2(9145)"},
				},
			},
		},
		{
			TestName: "Full TLS config",
			env: map[string]string{
				"HOSTNAME":                                    "tls-config",
				"MQ_NATIVE_HA_INSTANCE_0_NAME":                "tls-config-instance0",
				"MQ_NATIVE_HA_INSTANCE_1_NAME":                "tls-config-instance1",
				"MQ_NATIVE_HA_INSTANCE_2_NAME":                "tls-config-instance2",
				"MQ_NATIVE_HA_INSTANCE_0_REPLICATION_ADDRESS": "tls-config-instance0(9145)",
				"MQ_NATIVE_HA_INSTANCE_1_REPLICATION_ADDRESS": "tls-config-instance1(9145)",
				"MQ_NATIVE_HA_INSTANCE_2_REPLICATION_ADDRESS": "tls-config-instance2(9145)",
				"MQ_NATIVE_HA_TLS":                            "true",
				"MQ_NATIVE_HA_CIPHERSPEC":                     "a-cipher-spec",
				"MQ_NATIVE_HA_KEY_REPOSITORY":                 "/path/to/repository",
			},
			overrides: testOverrides{
				certificateLabel: asRef("cert-label-here"),
				fips:             asRef(false),
			},
			expected: haConfig{
				Name: "tls-config",
				Instances: [3]haInstance{
					{"tls-config-instance0", "tls-config-instance0(9145)"},
					{"tls-config-instance1", "tls-config-instance1(9145)"},
					{"tls-config-instance2", "tls-config-instance2(9145)"},
				},
				haTLSEnabled:  true,
				CipherSpec:    "a-cipher-spec",
				keyRepository: "/path/to/repository",

				CertificateLabel: "cert-label-here", // From override
				fipsAvailable:    false,             // From override
			},
		},

		{
			TestName: "Group TLS (live plain) config",
			env: map[string]string{
				"HOSTNAME":                                    "group-live-plain-config",
				"MQ_NATIVE_HA_INSTANCE_0_NAME":                "group-live-plain-config0",
				"MQ_NATIVE_HA_INSTANCE_1_NAME":                "group-live-plain-config1",
				"MQ_NATIVE_HA_INSTANCE_2_NAME":                "group-live-plain-config2",
				"MQ_NATIVE_HA_INSTANCE_0_REPLICATION_ADDRESS": "group-live-plain-config0(9145)",
				"MQ_NATIVE_HA_INSTANCE_1_REPLICATION_ADDRESS": "group-live-plain-config1(9145)",
				"MQ_NATIVE_HA_INSTANCE_2_REPLICATION_ADDRESS": "group-live-plain-config2(9145)",
				"MQ_NATIVE_HA_CIPHERSPEC":                     "NULL",
				"MQ_NATIVE_HA_KEY_REPOSITORY":                 "/path/to/repository",

				"MQ_NATIVE_HA_GROUP_RECOVERY_ENABLED":    "true",
				"MQ_NATIVE_HA_GROUP_LOCAL_NAME":          "alpha",
				"MQ_NATIVE_HA_GROUP_RECOVERY_NAME":       "beta",
				"MQ_NATIVE_HA_GROUP_CIPHERSPEC":          "ANY_TLS",
				"MQ_NATIVE_HA_GROUP_ROLE":                "Live",
				"MQ_NATIVE_HA_GROUP_LOCAL_ADDRESS":       "(4445)",
				"MQ_NATIVE_HA_GROUP_REPLICATION_ADDRESS": "beta-address(4445)",
			},
			overrides: testOverrides{
				groupCertificateLabel: asRef("recovery-cert-label-here"),
				fips:                  asRef(false),
			},
			expected: haConfig{
				Name: "group-live-plain-config",
				Instances: [3]haInstance{
					{"group-live-plain-config0", "group-live-plain-config0(9145)"},
					{"group-live-plain-config1", "group-live-plain-config1(9145)"},
					{"group-live-plain-config2", "group-live-plain-config2(9145)"},
				},
				Group: haGroupConfig{
					Local: haLocalGroupConfig{
						Name:    "alpha",
						Role:    "Live",
						Address: "(4445)",
					},
					Recovery: haRecoveryGroupConfig{
						Name:    "beta",
						Enabled: true,
						Address: "beta-address(4445)",
					},
					CertificateLabel: "recovery-cert-label-here", // From override
					CipherSpec:       "ANY_TLS",
				},
				CipherSpec:    "NULL",
				keyRepository: "/path/to/repository",

				fipsAvailable: false, // From override
			},
		},
	}

	for _, test := range tests {
		t.Run(test.TestName, func(t *testing.T) {
			// Set environment for test
			savedEnv := make([]string, len(os.Environ()))
			copy(savedEnv, os.Environ())
			defer func() {
				os.Clearenv()
				for _, env := range savedEnv {
					parts := strings.SplitN(env, "=", 2)
					os.Setenv(parts[0], parts[1])
				}
			}()

			for key, value := range test.env {
				os.Setenv(key, value)
			}

			testLogger, logBuffer, err := newTestLogger(test.TestName)
			if err != nil {
				t.Fatalf("Failed to create test logger: %s", err.Error())
			}

			// Load config from env
			cfg, err := loadConfigFromEnv(testLogger)
			t.Log(logBuffer.String())
			if err != nil {
				t.Fatalf("Loading config failed: %s", err.Error())
			}

			test.overrides.apply(cfg)

			// Validate
			if *cfg != test.expected {
				t.Fatalf("Configuration does not match expected:\n\tExpected: %#v\n\tActual: %#v\n", test.expected, *cfg)
			}
		})
	}
}

func TestTemplatingFromConfig(t *testing.T) {
	tests := []struct {
		TestName           string
		config             haConfig
		expectedResultName string
	}{
		{
			TestName:           "MinimalConfig",
			config:             haConfig{},
			expectedResultName: "minimal-config.ini",
		},
		{
			TestName: "Base TLS config (no FIPS)",
			config: haConfig{
				haTLSEnabled:     true,
				CertificateLabel: "baseTLS",
				fipsAvailable:    false,
			},
			expectedResultName: "tls-basic.ini",
		},
		{
			TestName: "Base TLS config (with FIPS)",
			config: haConfig{
				haTLSEnabled:     true,
				CertificateLabel: "baseTLS",
				fipsAvailable:    true,
			},
			expectedResultName: "tls-basic-fips.ini",
		},
		{
			TestName: "Full TLS config (no-fips)",
			config: haConfig{
				haTLSEnabled:     true,
				CertificateLabel: "baseTLS",
				CipherSpec:       "some-cipher",
				keyRepository:    "/a/non/existant/path",
				fipsAvailable:    false,
			},
			expectedResultName: "tls-full.ini",
		},
		{
			TestName: "TLS config but not enabled",
			config: haConfig{
				haTLSEnabled:     false,
				CertificateLabel: "baseTLS",
				CipherSpec:       "some-cipher",
				keyRepository:    "/a/non/existant/path",
				fipsAvailable:    false,
			},
			expectedResultName: "minimal-config.ini",
		},
		{
			TestName: "Minimal live config",
			config: haConfig{
				Group: haGroupConfig{
					Local: haLocalGroupConfig{
						Name: "alpha",
					},
					Recovery: haRecoveryGroupConfig{
						Name:    "beta",
						Enabled: true,
						Address: "beta-address(4445)",
					},
					CertificateLabel: "recoveryTLS",
				},
			},
			expectedResultName: "group-live-minimal.ini",
		},
		{
			TestName: "Minimal recovery config",
			config: haConfig{
				Group: haGroupConfig{
					Local: haLocalGroupConfig{
						Name: "beta",
						Role: "Recovery",
					},
					Recovery: haRecoveryGroupConfig{
						Name:    "alpha",
						Enabled: true,
						Address: "alpha-address(4445)",
					},
					CertificateLabel: "recoveryTLS",
				},
			},
			expectedResultName: "group-recovery-minimal.ini",
		},
		{
			TestName: "Group TLS (live plain) config",
			config: haConfig{
				Group: haGroupConfig{
					Local: haLocalGroupConfig{
						Name:    "alpha",
						Role:    "Live",
						Address: "(4445)",
					},
					Recovery: haRecoveryGroupConfig{
						Name:    "beta",
						Enabled: true,
						Address: "beta-address(4445)",
					},
					CertificateLabel: "recoveryTLS",
					CipherSpec:       "ANY_TLS",
				},
				CipherSpec: "NULL",
			},
			expectedResultName: "group-live-plain-ha.ini",
		},
	}

	templateFile := "../../ha/native-ha.ini.tpl"

	for _, test := range tests {
		t.Run(test.TestName, func(t *testing.T) {
			t.Logf(`Runing templating test "%s"`, test.TestName)
			t.Logf(`Expected to match template "%s"`, test.expectedResultName)
			testLogger, logBuffer, err := newTestLogger(test.TestName)
			if err != nil {
				t.Fatalf("Failed to create test logger: %s", err.Error())
			}

			// Load test config
			cfg := applyTestDefaults(test.config)

			// Generate template
			tempOutputPath, err := os.CreateTemp("", "")
			if err != nil {
				t.Fatalf("Failed to create temporary output file: %s", err.Error())
			}
			defer func() { _ = os.Remove(tempOutputPath.Name()) }()
			err = cfg.generate(templateFile, tempOutputPath.Name(), testLogger)
			t.Log(logBuffer.String())
			if err != nil {
				t.Fatalf("Processing template to config failed: %s", err.Error())
			}
			actual, err := os.ReadFile(tempOutputPath.Name())
			if err != nil {
				t.Fatalf("Failed to read '%s': %s", test.TestName, err.Error())
			}

			// Validate
			assertIniMatch(t, string(actual), test.expectedResultName)
		})
	}
}

func applyTestDefaults(testConfig haConfig) haConfig {
	baseName := "test-config"
	setIfBlank(&testConfig.Name, baseName)
	for i := 0; i < 3; i++ {
		instName := fmt.Sprintf("%s-instance%d", baseName, i)
		replAddress := fmt.Sprintf("%s(9145)", instName)
		setIfBlank(&testConfig.Instances[i].Name, instName)
		setIfBlank(&testConfig.Instances[i].ReplicationAddress, replAddress)
	}
	return testConfig
}

func setIfBlank[T comparable](setting *T, val T) {
	var zero T
	if *setting == zero {
		*setting = val
	}
}

func assertIniMatch(t *testing.T, actual string, expectedResultName string) {
	expectedContent, err := testFixtures.ReadFile(fmt.Sprintf("test_fixtures/%s", expectedResultName))
	if err != nil {
		t.Fatalf("Failed to read expected results file (%s): %s", expectedResultName, err.Error())
	}
	expectedLines := strings.Split(string(expectedContent), "\n")
	actualLines := strings.Split(actual, "\n")

	filterBlank := func(lines *[]string) {
		n := 0
		for i := 0; i < len(*lines); i++ {
			if strings.TrimSpace((*lines)[i]) == "" {
				continue
			}
			(*lines)[n] = (*lines)[i]
			n++
		}
		*lines = (*lines)[0:n]
	}
	filterBlank(&expectedLines)
	filterBlank(&actualLines)

	maxLine := len(expectedLines)
	if len(actualLines) > maxLine {
		maxLine = len(actualLines)
	}

	for i := 0; i < maxLine; i++ {
		actLine, expLine := "", ""
		if i < len(actualLines) {
			actLine = actualLines[i]
		}
		if i < len(expectedLines) {
			expLine = expectedLines[i]
		}
		if actLine != expLine {
			t.Fatalf("Template does not match\n\nFirst difference at line %d:\n\tExpected: %s\n\tActual  : %s\n\nExpected:\n\t%s\n\nActual:\n\t%s", i+1, expLine, actLine, strings.Join(expectedLines, "\n\t"), strings.Join(actualLines, "\n\t"))
		}
	}
}

func newTestLogger(name string) (*logger.Logger, *bytes.Buffer, error) {
	buffer := new(bytes.Buffer)
	l, err := logger.NewLogger(buffer, true, false, name)
	return l, buffer, err
}

type testOverrides struct {
	certificateLabel      *string
	groupCertificateLabel *string
	fips                  *bool
}

func (t testOverrides) apply(cfg *haConfig) {
	if t.certificateLabel != nil {
		cfg.CertificateLabel = *t.certificateLabel
	}
	if t.groupCertificateLabel != nil {
		cfg.Group.CertificateLabel = *t.groupCertificateLabel
	}
	if t.fips != nil {
		cfg.fipsAvailable = *t.fips
	}
}

func asRef[T any](val T) *T {
	ref := &val
	return ref
}
