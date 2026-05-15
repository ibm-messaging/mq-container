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

package probes

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// TestNewProbeState verifies the initial in-memory probe state.
func TestNewProbeState(t *testing.T) {
	state := NewProbeState()

	if state == nil {
		t.Fatalf("expected ProbeLoggingState to be initialized")
	}

	if state.LivenessProbeLoggingState == nil {
		t.Fatal("expected LivenessProbeLoggingState to be initialized")
	}

	if state.LivenessProbeLoggingState.CurrentRun != nil {
		t.Errorf("expected CurrentRun to be nil, go %+v", state.LivenessProbeLoggingState.CurrentRun)
	}

	if state.LivenessProbeLoggingState.PreviousRuns[0].StartTime != nil ||
		state.LivenessProbeLoggingState.PreviousRuns[1].StartTime != nil {
		t.Errorf("expected PrevioisRuns to be empty, got %+v", state.LivenessProbeLoggingState.PreviousRuns)
	}

	if state.StartupProbeLoggingState == nil {
		t.Fatal("expected StartupProbeLoggingState to be initialized")
	}

	if state.StartupProbeLoggingState.CurrentRun != nil {
		t.Errorf("expected StartupProbeLoggingState.CurrentRun to be nil, got %+v", state.StartupProbeLoggingState.CurrentRun)
	}
}

// Test values - Probe logging enablement
var isProbeLoggingEnabledTests = []struct {
	testNum        int
	probeType      ProbeType
	createSocket   bool
	expectedResult bool
}{
	{1, LivenessProbe, true, true},
	{2, LivenessProbe, false, false},
	{3, StartupProbe, true, true},
	{4, StartupProbe, false, false},
	{5, ProbeType("UNKNOWN"), true, false},
}

// TestIsProbeLoggingEnabled verifies probe logging enablement based on
// the presence of the corresponding probe socket.
func TestIsProbeLoggingEnabled(t *testing.T) {

	for _, test := range isProbeLoggingEnabledTests {

		tempDir := t.TempDir()

		var sockPath string

		switch test.probeType {
		case LivenessProbe:
			originalPath := LivenessProbeSockPath
			sockPath = filepath.Join(tempDir, "liveness.sock")
			LivenessProbeSockPath = sockPath

			defer func() {
				LivenessProbeSockPath = originalPath
			}()

		case StartupProbe:
			originalPath := StartupProbeSockPath
			sockPath = filepath.Join(tempDir, "startup.sock")
			StartupProbeSockPath = sockPath

			defer func() {
				StartupProbeSockPath = originalPath
			}()
		}

		if test.createSocket && sockPath != "" {
			file, err := os.Create(sockPath)
			if err != nil {
				t.Fatalf("IsProbeLoggingEnabled() : Test%v\nFailed to create socket file %v: %v",
					test.testNum, sockPath, err)
			}
			_ = file.Close()
		}

		result := IsProbeLoggingEnabled(test.probeType)

		if result != test.expectedResult {
			t.Errorf("IsProbeLoggingEnabled() : Test%v\nExpected:\t%v\nGot:\t\t%v", test.testNum, test.expectedResult, result)
		}

	}
}

// Test values - Probe logging initialization
var initializeProbeLoggingExecutionTests = []struct {
	testNum             int
	probeType           ProbeType
	probeStatus         ProbeStatus
	expectedProbeType   ProbeType
	expectedProbeStatus ProbeStatus
}{
	{1, LivenessProbe, ProbeIncomplete, LivenessProbe, ProbeIncomplete},
	{2, StartupProbe, ProbeIncomplete, StartupProbe, ProbeIncomplete},
	{3, ProbeType("UNKNOWN"), ProbeIncomplete, ProbeType("UNKNOWN"), ProbeIncomplete},
}

// TestInitializeProbeLoggingExecution verifies initial probe execution metadata.
func TestInitializeProbeLoggingExecution(t *testing.T) {
	for _, test := range initializeProbeLoggingExecutionTests {
		execution := InitializeProbeLoggingExecution(test.probeType, testStartTime, test.probeStatus)

		if execution == nil {
			t.Fatalf("InitializeProbeLoggingExecution() : Test%v\nExpected probe logging execution to be initialized", test.testNum)
		}

		if execution.ProbeType != test.expectedProbeType {
			t.Errorf("InitializeProbeLoggingExecution() : Test%v\nExpected probe type:\t%v\nGot:\t\t%v", test.testNum, test.expectedProbeType, execution.ProbeType)
		}

		if execution.Status != test.expectedProbeStatus {
			t.Errorf("InitializeProbeLoggingExecution() : Test%v\nExpected status:\t%v\nGot:\t\t%v", test.testNum, test.expectedProbeStatus, execution.Status)
		}

		if execution.StartTime == nil || !execution.StartTime.Equal(testStartTime) {
			t.Errorf("InitializeProbeLoggingExecution() : Test%v\nExpected start time:\t%v\nGot:\t\t%v", test.testNum, testStartTime, execution.StartTime)
		}
	}
}

// Test values - Liveness probe completion
var logLivenessProbeMessageTests = []struct {
	testNum             int
	healthy             bool
	out                 string
	err                 error
	expectedProbeStatus ProbeStatus
	expectedLogLevel    LogLevel
	expectedLog         string
	expectedDuration    int64
}{
	{1, true, "QMNAME(testQM) STATUS(RUNNING)\n", nil, ProbePassed, INFO, "Liveness Probe Passed: Output: QMNAME(testQM) STATUS(RUNNING)", 25},
	{2, false, "QMNAME(testQM) STATUS(ENDED)\n", nil, ProbeFailed, ERROR, "Liveness Probe Failed: QueueManager is not healthy, Output: QMNAME(testQM) STATUS(ENDED)", 25},
	{3, false, "", errors.New("chkmqhealthy error"), ProbeFailed, ERROR, "Liveness Probe Failed: chkmqhealthy error", 25},
}

// TestLogLivenessProbeMessage verifies completion handling for passed,
// unhealthy, and error outcomes.
func TestLogLivenessProbeMessage(t *testing.T) {
	for _, test := range logLivenessProbeMessageTests {
		execution := &ProbeLoggingExecution{
			ProbeType: LivenessProbe,
			StartTime: timePtr(testStartTime),
			Status:    ProbeIncomplete,
		}

		execution.LogLivenessProbeMessage(testEndTime, test.healthy, test.out, test.err)

		if execution.Status != test.expectedProbeStatus {
			t.Errorf("LogLivenessProbeMessage() : Test%v\nExpected status:\t%v\nGot:\t\t%v",
				test.testNum, test.expectedProbeStatus, execution.Status)
		}

		if execution.LogLevel != test.expectedLogLevel {
			t.Errorf("LogLivenessProbeMessage() : Test%v\nExpected log level:\t%v\nGot:\t\t%v",
				test.testNum, test.expectedLogLevel, execution.LogLevel)
		}

		if execution.LogMessage != test.expectedLog {
			t.Errorf("LogLivenessProbeMessage() : Test%v\nExpected log:\t%v\nGot:\t\t%v",
				test.testNum, test.expectedLog, execution.LogMessage)
		}

		if execution.Duration == nil || *execution.Duration != test.expectedDuration {
			t.Errorf("LogLivenessProbeMessage() : Test%v\nExpected duration:\t%v\nGot:\t\t%v",
				test.testNum, test.expectedDuration, execution.Duration)
		}

		if execution.EndTime == nil || !execution.EndTime.Equal(testEndTime) {
			t.Errorf("LogLivenessProbeMessage() : Test%v\nExpected end time:\t%v\nGot:\t\t%v",
				test.testNum, testEndTime, execution.EndTime)
		}
	}
}

// Test values - Liveness probe helper update
var logLivenessProbeMessageHelperTests = []struct {
	testNum             int
	status              ProbeStatus
	duration            int64
	message             string
	logLevel            LogLevel
	expectedProbeStatus ProbeStatus
	expectedLogMessage  string
	expectedLevel       LogLevel
	expectedDuration    int64
}{
	{1, ProbePassed, 25, "passed", INFO, ProbePassed, "passed", INFO, 25},
	{2, ProbeFailed, 30, "failed", ERROR, ProbeFailed, "failed", ERROR, 30},
}

// TestLogLivenessProbeMessageHelper verifies direct field updates before socket emission.
func TestLogLivenessProbeMessageHelper(t *testing.T) {
	for _, test := range logLivenessProbeMessageHelperTests {

		execution := &ProbeLoggingExecution{
			ProbeType: LivenessProbe,
			StartTime: timePtr(testStartTime),
			Status:    ProbeIncomplete,
		}

		execution.logLivenessProbeMessageHelper(testEndTime, test.status, test.duration, test.message, test.logLevel)

		if execution.Status != test.expectedProbeStatus {
			t.Errorf("logLivenessProbeMessageHelper() : Test%v\nExpected status:\t%v\nGot:\t\t%v", test.testNum, test.expectedProbeStatus, execution.Status)
		}

		if execution.LogMessage != test.expectedLogMessage {
			t.Errorf("logLivenessProbeMessageHelper() : Test%v\nExpected log:\t%v\nGot:\t\t%v", test.testNum, test.expectedLogMessage, execution.LogMessage)
		}

		if execution.LogLevel != test.expectedLevel {
			t.Errorf("logLivenessProbeMessageHelper() : Test%v\nExpected log level:\t%v\nGot:\t\t%v", test.testNum, test.expectedLevel, execution.LogLevel)
		}

		if execution.Duration == nil || *execution.Duration != test.expectedDuration {
			t.Errorf("logLivenessProbeMessageHelper() : Test%v\nExpected duration:\t%v\nGot:\t\t%v", test.testNum, test.expectedDuration, execution.Duration)
		}

		if execution.EndTime == nil || !execution.EndTime.Equal(testEndTime) {
			t.Errorf("logLivenessProbeMessageHelper() : Test%v\nExpected end time:\t%v\nGot:\t\t%v", test.testNum, testEndTime, execution.EndTime)
		}
	}
}

// Test values - Liveness probe output messages
var buildLivenessProbeOutputMessageTests = []struct {
	testNum  int
	out      string
	expected string
}{
	{1, "QMNAME(testQM) STATUS(RUNNING)", "Output: QMNAME(testQM) STATUS(RUNNING)"},
	{2, "  QMNAME(testQM) STATUS(RUNNING)\n", "Output: QMNAME(testQM) STATUS(RUNNING)"},
	{3, "", ""},
	{4, "   \n\t", ""},
}

// TestBuildLivenessProbeOutputMessage verifies chkmqhealthy output formatting
// for normal, whitespace-padded, empty, and whitespace-only output.
func TestBuildLivenessProbeOutputMessage(t *testing.T) {
	for _, test := range buildLivenessProbeOutputMessageTests {
		result := buildLivenessProbeOutputMessage(test.out)

		if result != test.expected {
			t.Errorf("BuildLivenessProbeOutputMessage() : Test%v\nExpected:\t%v\nGot:\t\t%v",
				test.testNum, test.expected, result)
		}
	}
}

// Test values - Liveness probe failure messages
var buildLivenessProbeFailureMessageTests = []struct {
	testNum  int
	reason   string
	details  string
	expected string
}{
	{1, "chkmqhealthy error", "", "Liveness Probe Failed: chkmqhealthy error"},
	{2, "QueueManager is not healthy", "Output: QMNAME(testQM) STATUS(ENDED)", "Liveness Probe Failed: QueueManager is not healthy, Output: QMNAME(testQM) STATUS(ENDED)"},
	{3, "", "", "Liveness Probe Failed: "},
}

// TestBuildLivenessProbeFailureMessage verifies failure log message formatting
// with and without probe output details.
func TestBuildLivenessProbeFailureMessage(t *testing.T) {
	for _, test := range buildLivenessProbeFailureMessageTests {
		result := buildLivenessProbeFailureMessage(test.reason, test.details)

		if result != test.expected {
			t.Errorf("BuildLivenessProbeFailureMessage() : Test%v\nExpected:\t%v\nGot:\t\t%v",
				test.testNum, test.expected, result)
		}
	}
}

// Test values - Liveness probe success messages
var buildLivenessProbeSuccessMessageTests = []struct {
	testNum  int
	details  string
	expected string
}{
	{1, "Output: QMNAME(testQM) STATUS(RUNNING)", "Liveness Probe Passed: Output: QMNAME(testQM) STATUS(RUNNING)"},
	{2, "", "Liveness Probe Passed"},
}

// TestBuildLivenessProbeSuccessMessage verifies success log message formatting
// with and without probe output details.
func TestBuildLivenessProbeSuccessMessage(t *testing.T) {
	for _, test := range buildLivenessProbeSuccessMessageTests {
		result := buildLivenessProbeSuccessMessage(test.details)

		if result != test.expected {
			t.Errorf("BuildLivenessProbeSuccessMessage() : Test%v\nExpected:\t%v\nGot:\t\t%v",
				test.testNum, test.expected, result)
		}
	}
}

// Test values - Startup probe completion
var logStartupProbeMessageTests = []struct {
	testNum             int
	started             bool
	err                 error
	expectedProbeStatus ProbeStatus
	expectedLogLevel    LogLevel
	expectedLog         string
}{
	{1, true, nil, ProbePassed, INFO, "Startup Probe Passed: QueueManager started successfully"},
	{2, false, nil, ProbeFailed, ERROR, "Startup Probe Failed: QueueManager is not started"},
	{3, false, errors.New("startup probe error"), ProbeFailed, ERROR, "Startup Probe Failed: startup probe error"},
}

// TestLogStartupProbeMessage verifies completion handling for passed,
// not-started, and error startup probe outcomes.
func TestLogStartupProbeMessage(t *testing.T) {
	for _, test := range logStartupProbeMessageTests {

		execution := &ProbeLoggingExecution{
			ProbeType: StartupProbe,
			StartTime: timePtr(testStartTime),
			Status:    ProbeIncomplete,
		}

		execution.LogStartupProbeMessage(test.started, test.err)

		if execution.Status != test.expectedProbeStatus {
			t.Errorf("LogStartupProbeMessage() : Test%v\nExpected status:\t%v\nGot:\t\t%v", test.testNum, test.expectedProbeStatus, execution.Status)
		}

		if execution.LogLevel != test.expectedLogLevel {
			t.Errorf("LogStartupProbeMessage() : Test%v\nExpected log level:\t%v\nGot:\t\t%v", test.testNum, test.expectedLogLevel, execution.LogLevel)
		}

		if execution.LogMessage != test.expectedLog {
			t.Errorf("LogStartupProbeMessage() : Test%v\nExpected log:\t%v\nGot:\t\t%v", test.testNum, test.expectedLog, execution.LogMessage)
		}
	}
}

// Test values - Startup probe helper update
var logStartupProbeMessageHelperTests = []struct {
	testNum             int
	status              ProbeStatus
	message             string
	logLevel            LogLevel
	expectedProbeStatus ProbeStatus
	expectedLogMessage  string
	expectedLevel       LogLevel
}{
	{1, ProbePassed, "passed", INFO, ProbePassed, "passed", INFO},
	{2, ProbeFailed, "failed", ERROR, ProbeFailed, "failed", ERROR},
}

// TestLogStartupProbeMessageHelper verifies direct startup probe field updates before socket emission.
func TestLogStartupProbeMessageHelper(t *testing.T) {
	for _, test := range logStartupProbeMessageHelperTests {

		execution := &ProbeLoggingExecution{
			ProbeType: StartupProbe,
			StartTime: timePtr(testStartTime),
			Status:    ProbeIncomplete,
		}

		execution.logStartupProbeMessageHelper(test.status, test.message, test.logLevel)

		if execution.Status != test.expectedProbeStatus {
			t.Errorf("logStartupProbeMessageHelper() : Test%v\nExpected status:\t%v\nGot:\t\t%v", test.testNum, test.expectedProbeStatus, execution.Status)
		}

		if execution.LogMessage != test.expectedLogMessage {
			t.Errorf("logStartupProbeMessageHelper() : Test%v\nExpected log:\t%v\nGot:\t\t%v", test.testNum, test.expectedLogMessage, execution.LogMessage)
		}

		if execution.LogLevel != test.expectedLevel {
			t.Errorf("logStartupProbeMessageHelper() : Test%v\nExpected log level:\t%v\nGot:\t\t%v", test.testNum, test.expectedLevel, execution.LogLevel)
		}
	}
}

// Test values - Startup probe failure messages
var buildStartupProbeFailureMessageTests = []struct {
	testNum  int
	reason   string
	expected string
}{
	{1, "QueueManager is not started", "Startup Probe Failed: QueueManager is not started"},
	{2, "startup probe error", "Startup Probe Failed: startup probe error"},
}

// TestBuildStartupProbeFailureMessage verifies startup failure log formatting.
func TestBuildStartupProbeFailureMessage(t *testing.T) {
	for _, test := range buildStartupProbeFailureMessageTests {
		result := buildStartupProbeFailureMessage(test.reason)

		if result != test.expected {
			t.Errorf("BuildStartupProbeFailureMessage() : Test%v\nExpected:\t%v\nGot:\t\t%v", test.testNum, test.expected, result)
		}
	}
}

// Test values - Startup probe success messages
var buildStartupProbeSuccessMessageTests = []struct {
	testNum  int
	reason   string
	expected string
}{
	{1, "QueueManager started successfully", "Startup Probe Passed: QueueManager started successfully"},
}

// TestBuildStartupProbeSuccessMessage verifies startup success log formatting.
func TestBuildStartupProbeSuccessMessage(t *testing.T) {
	for _, test := range buildStartupProbeSuccessMessageTests {
		result := buildStartupProbeSuccessMessage(test.reason)

		if result != test.expected {
			t.Errorf("BuildStartupProbeSuccessMessage() : Test%v\nExpected:\t%v\nGot:\t\t%v",
				test.testNum, test.expected, result)
		}
	}
}

// Test values - Probe status formatting
var getProbeStatusTests = []struct {
	testNum  int
	status   ProbeStatus
	expected string
}{
	{1, ProbeIncomplete, "Incomplete"},
	{2, ProbePassed, "Passed"},
	{3, ProbeFailed, "Failed"},
	{4, ProbeStatus("UNEXPECTED"), "Unknown"},
}

// TestGetProbeStatus verifies human-readable status formatting.
func TestGetProbeStatus(t *testing.T) {
	for _, test := range getProbeStatusTests {
		result := test.status.getProbeStatus()

		if result != test.expected {
			t.Errorf("getProbeStatus() : Test%v\nExpected:\t%v\nGot:\t\t%v",
				test.testNum, test.expected, result)
		}
	}
}
