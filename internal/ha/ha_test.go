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
				tlsEnabled:    true,
				cipherSpec:    "a-cipher-spec",
				keyRepository: "/path/to/repository",

				certificateLabel: "cert-label-here", // From override
				fipsAvailable:    false,             // From override
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

			testLogger, logBuffer, err := newTestLogger(t, test.TestName)
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
				tlsEnabled:       true,
				certificateLabel: "baseTLS",
				fipsAvailable:    false,
			},
			expectedResultName: "tls-basic.ini",
		},
		{
			TestName: "Base TLS config (with FIPS)",
			config: haConfig{
				tlsEnabled:       true,
				certificateLabel: "baseTLS",
				fipsAvailable:    true,
			},
			expectedResultName: "tls-basic-fips.ini",
		},
		{
			TestName: "Full TLS config (no-fips)",
			config: haConfig{
				tlsEnabled:       true,
				certificateLabel: "baseTLS",
				cipherSpec:       "some-cipher",
				keyRepository:    "/a/non/existant/path",
				fipsAvailable:    false,
			},
			expectedResultName: "tls-full.ini",
		},
		{
			TestName: "TLS config but not enabled",
			config: haConfig{
				tlsEnabled:       false,
				certificateLabel: "baseTLS",
				cipherSpec:       "some-cipher",
				keyRepository:    "/a/non/existant/path",
				fipsAvailable:    false,
			},
			expectedResultName: "minimal-config.ini",
		},
	}

	templateFile := "../../ha/native-ha.ini.tpl"

	for _, test := range tests {
		t.Run(test.TestName, func(t *testing.T) {
			testLogger, logBuffer, err := newTestLogger(t, test.TestName)
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
			t.Fatalf("Template does not match\n\nExpected:\n\t%s\n\nActual:\n\t%s\n\nFirst difference at line %d:\n\tExpected: %s\n\tActual  : %s", strings.Join(expectedLines, "\n\t"), strings.Join(actualLines, "\n\t"), i+1, expLine, actLine)
		}
	}
}

func newTestLogger(t *testing.T, name string) (*logger.Logger, *bytes.Buffer, error) {
	buffer := new(bytes.Buffer)
	l, err := logger.NewLogger(buffer, true, false, name)
	return l, buffer, err
}

type testOverrides struct {
	certificateLabel *string
	fips             *bool
}

func (t testOverrides) apply(cfg *haConfig) {
	if t.certificateLabel != nil {
		cfg.certificateLabel = *t.certificateLabel
	}
	if t.fips != nil {
		cfg.fipsAvailable = *t.fips
	}
}

func asRef[T any](val T) *T {
	ref := &val
	return ref
}
