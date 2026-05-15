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
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/ibm-messaging/mq-container/pkg/logger"
)

func newTestLogger(t *testing.T) (*logger.Logger, *bytes.Buffer) {
	t.Helper()

	var buf bytes.Buffer
	log, err := logger.NewLogger(&buf, false, false, "testQM")
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}

	return log, &buf
}

func newTestSocket(t *testing.T) (*ProbeLoggingSocket, *bytes.Buffer) {
	t.Helper()

	log, buf := newTestLogger(t)

	return &ProbeLoggingSocket{
		logger:            log,
		lastLog:           make(map[string]string),
		probeLoggingState: NewProbeState(),
	}, buf
}

// TestNewProbeLoggingSocket verifies socket configuration initialization.
func TestNewProbeLoggingSocket(t *testing.T) {
	state := NewProbeState()
	log, _ := newTestLogger(t)

	socket := NewProbeLoggingSocket("testQM", "", state, log)

	if socket == nil {
		t.Fatalf("expected ProbeLoggingSocket to be initialized")
	}

	if socket.logger != log {
		t.Errorf("expected logger to be set")
	}

	if socket.probeLoggingState != state {
		t.Errorf("expected probe logging state to be set")
	}

	if socket.lastLog == nil {
		t.Errorf("expected lastLog map to be initialized")
	}

	expectedSockets := []string{LivenessProbeSockPath, StartupProbeSockPath}

	if len(socket.sockets) != len(expectedSockets) {
		t.Errorf("expected socket paths %+v, got %+v", expectedSockets, socket.sockets)
	}

	for i, expectedSocket := range expectedSockets {
		if socket.sockets[i] != expectedSocket {
			t.Errorf("expected socket [%d] to be %s, got %s", i, expectedSocket, socket.sockets[i])
		}
	}
}

// Test values - Liveness probe state
var handleLivenessProbeTests = []struct {
	testNum             int
	events              []*ProbeLoggingExecution
	expectedProbeStatus ProbeStatus
	expectedLogMessage  string
	expectedPrev0       string
	expectedPrev1       string
}{
	{
		1,
		[]*ProbeLoggingExecution{
			{ProbeType: LivenessProbe, StartTime: timePtr(testStartTime), Status: ProbeIncomplete},
			{ProbeType: LivenessProbe, EndTime: timePtr(testEndTime), Status: ProbePassed, Duration: int64Ptr(25), LogMessage: "passed", LogLevel: INFO},
		},
		ProbePassed,
		"passed",
		"",
		"",
	},
	{
		2,
		[]*ProbeLoggingExecution{
			{ProbeType: LivenessProbe, StartTime: timePtr(testStartTime), Status: ProbeIncomplete},
			{ProbeType: LivenessProbe, EndTime: timePtr(testEndTime), Status: ProbeFailed, Duration: int64Ptr(25), LogMessage: "failed", LogLevel: ERROR},
		},
		ProbeFailed,
		"failed",
		"",
		"",
	},
	{
		3,
		[]*ProbeLoggingExecution{
			{ProbeType: LivenessProbe, StartTime: timePtr(testStartTime), Status: ProbeIncomplete},
			{ProbeType: LivenessProbe, Status: ProbePassed, Duration: int64Ptr(10), LogMessage: "run1", LogLevel: INFO},
			{ProbeType: LivenessProbe, StartTime: timePtr(testStartTime.Add(10 * time.Second)), Status: ProbeIncomplete},
			{ProbeType: LivenessProbe, Status: ProbePassed, Duration: int64Ptr(11), LogMessage: "run2", LogLevel: INFO},
			{ProbeType: LivenessProbe, StartTime: timePtr(testStartTime.Add(20 * time.Second)), Status: ProbeIncomplete},
			{ProbeType: LivenessProbe, Status: ProbePassed, Duration: int64Ptr(12), LogMessage: "run3", LogLevel: INFO},
		},
		ProbePassed,
		"run3",
		"run2",
		"run1",
	},
}

// TestHandleLivenessProbe verifies state transitions for incomplete,
// completed, failed, and history-retention liveness probe events.
func TestHandleLivenessProbe(t *testing.T) {
	for _, test := range handleLivenessProbeTests {
		socket, _ := newTestSocket(t)

		for _, event := range test.events {
			socket.handleLivenessProbe(event, LivenessProbeSockPath)
		}

		state := socket.probeLoggingState.LivenessProbeLoggingState

		if state.CurrentRun == nil || state.CurrentRun.Status != test.expectedProbeStatus {
			t.Errorf("handleLivenessProbe() : Test%v\nExpected current status:\t%v\nGot:\t\t%v",
				test.testNum, test.expectedProbeStatus, state.CurrentRun)
			continue
		}

		if state.CurrentRun.LogMessage != test.expectedLogMessage {
			t.Errorf("handleLivenessProbe() : Test%v\nExpected current log:\t%v\nGot:\t\t%v",
				test.testNum, test.expectedLogMessage, state.CurrentRun.LogMessage)
		}

		if state.PreviousRuns[0].LogMessage != test.expectedPrev0 {
			t.Errorf("handleLivenessProbe() : Test%v\nExpected previous[0]:\t%v\nGot:\t\t%v",
				test.testNum, test.expectedPrev0, state.PreviousRuns[0].LogMessage)
		}

		if state.PreviousRuns[1].LogMessage != test.expectedPrev1 {
			t.Errorf("handleLivenessProbe() : Test%v\nExpected previous[1]:\t%v\nGot:\t\t%v",
				test.testNum, test.expectedPrev1, state.PreviousRuns[1].LogMessage)
		}
	}
}

// TestHandleLivenessProbeNil verifies that nil liveness probe execution and nil
// liveness state are ignored safely.
func TestHandleLivenessProbeNil(t *testing.T) {
	socket, _ := newTestSocket(t)

	socket.handleLivenessProbe(nil, LivenessProbeSockPath)

	state := socket.probeLoggingState.LivenessProbeLoggingState

	if state.CurrentRun != nil {
		t.Errorf("Expected CurrentRun to remain nil for nil input, got: %v", state.CurrentRun)
	}

	if state.PreviousRuns[0].StartTime != nil || state.PreviousRuns[1].StartTime != nil {
		t.Errorf("Expected PreviousRuns to remain empaty for nil input, got: %+v", state.PreviousRuns)
	}

	socket.probeLoggingState.LivenessProbeLoggingState = nil

	socket.handleLivenessProbe(&ProbeLoggingExecution{
		ProbeType: LivenessProbe,
		Status:    ProbeIncomplete,
	}, LivenessProbeSockPath)

	if socket.probeLoggingState.LivenessProbeLoggingState != nil {
		t.Errorf("Expected LivenessProbeLoggingState to remain nil, got: %+v", socket.probeLoggingState.LivenessProbeLoggingState)
	}
}

// Test Values - Startup probe test
var handleStartupProbeTests = []struct {
	testNum              int
	events               []*ProbeLoggingExecution
	expectedProbeStatus  ProbeStatus
	expectedAttemptCount int
	expectedLogMessage   string
}{
	{1,
		[]*ProbeLoggingExecution{
			{ProbeType: StartupProbe, StartTime: timePtr(testStartTime), Status: ProbeIncomplete},
			{ProbeType: StartupProbe, Status: ProbeFailed, LogMessage: "Startup Probe Failed: QueueManager is not started", LogLevel: ERROR},
		},
		ProbeFailed,
		1,
		"Startup Probe Failed: QueueManager is not started",
	},
	{
		2,
		[]*ProbeLoggingExecution{
			{ProbeType: StartupProbe, StartTime: timePtr(testStartTime), Status: ProbeIncomplete},
			{ProbeType: StartupProbe, Status: ProbePassed, LogMessage: "Startup Probe Passed: QueueManager started successfully", LogLevel: INFO},
		},
		ProbePassed,
		1,
		"Startup Probe Passed: QueueManager started successfully",
	},
	{
		3,
		[]*ProbeLoggingExecution{
			{ProbeType: StartupProbe, StartTime: timePtr(testStartTime), Status: ProbeIncomplete},
			{ProbeType: StartupProbe, Status: ProbeFailed, LogMessage: "failed1", LogLevel: ERROR},
			{ProbeType: StartupProbe, StartTime: timePtr(testStartTime.Add(10 * time.Second)), Status: ProbeIncomplete},
			{ProbeType: StartupProbe, Status: ProbeFailed, LogMessage: "failed2", LogLevel: ERROR},
			{ProbeType: StartupProbe, StartTime: timePtr(testStartTime.Add(20 * time.Second)), Status: ProbeIncomplete},
			{ProbeType: StartupProbe, Status: ProbePassed, LogMessage: "passed", LogLevel: INFO},
		},
		ProbePassed,
		3,
		"passed",
	},
}

// TestHandleStartupProbe verifies state transitions for incomplete,
// completed, failed, and attempt-count startup probe events.
func TestHandleStartupProbe(t *testing.T) {

	for _, test := range handleStartupProbeTests {
		socket, _ := newTestSocket(t)

		for _, event := range test.events {
			socket.handleStartupProbe(event, StartupProbeSockPath)
		}

		state := socket.probeLoggingState.StartupProbeLoggingState

		if state.CurrentRun == nil || state.CurrentRun.Status != test.expectedProbeStatus {
			t.Errorf("handleStartupProbe() : Test%v\nExpected current status:\t%v\nGot:\t\t%v", test.testNum, test.expectedProbeStatus, state.CurrentRun)
		}

		if state.CurrentRun.AttemptCount != test.expectedAttemptCount {
			t.Errorf("handleStartupProbe() : Test%v\nExpected attempt count:\t%v\nGot:\t\t%v", test.testNum, test.expectedAttemptCount, state.CurrentRun.AttemptCount)
		}

		if state.CurrentRun.LogMessage != test.expectedLogMessage {
			t.Errorf("handleStartupProbe() : Test%v\nExpected current log:\t%v\nGot:\t\t%v", test.testNum, test.expectedLogMessage, state.CurrentRun.LogMessage)
		}
	}
}

// TestHandleStartupProbeNil verifies that nil startup probe execution and nil
// startup state are ignored safely.
func TestHandleStartupProbeNil(t *testing.T) {
	socket, _ := newTestSocket(t)

	socket.handleStartupProbe(nil, StartupProbeSockPath)

	state := socket.probeLoggingState.StartupProbeLoggingState

	if state.CurrentRun != nil {
		t.Errorf("Expected CurrentRun to remain nil for nil input, got: %v", state.CurrentRun)
	}

	socket.probeLoggingState.StartupProbeLoggingState = nil

	socket.handleStartupProbe(&ProbeLoggingExecution{
		ProbeType: StartupProbe,
		Status:    ProbeIncomplete,
	}, StartupProbeSockPath)

	if socket.probeLoggingState.StartupProbeLoggingState != nil {
		t.Errorf("Expected StartupProbeLoggingState to remain nil, got: %+v", socket.probeLoggingState.StartupProbeLoggingState)
	}
}

// Test values - Dedup log messages
var dedupLogTests = []struct {
	testNum    int
	probeType  string
	socketPath string
	entries    []struct {
		level   string
		message string
	}
	expectedPass  int
	expectedError int
}{
	{
		1,
		LivenessProbe.getProbeType(),
		LivenessProbeSockPath,
		[]struct {
			level   string
			message string
		}{
			{INFO.getLogLevel(), "pass"},
			{INFO.getLogLevel(), "pass"},
		},
		1,
		0,
	},
	{
		2,
		LivenessProbe.getProbeType(),
		LivenessProbeSockPath,
		[]struct {
			level   string
			message string
		}{
			{ERROR.getLogLevel(), "failure"},
			{ERROR.getLogLevel(), "failure"},
		},
		0,
		2,
	},
	{
		3,
		LivenessProbe.getProbeType(),
		LivenessProbeSockPath,
		[]struct {
			level   string
			message string
		}{
			{INFO.getLogLevel(), "pass"},
			{INFO.getLogLevel(), "pass"},
			{ERROR.getLogLevel(), "failure"},
			{INFO.getLogLevel(), "pass"},
		},
		2,
		1,
	},
	{
		4,
		StartupProbe.getProbeType(),
		StartupProbeSockPath,
		[]struct {
			level   string
			message string
		}{
			{INFO.getLogLevel(), "pass"},
			{INFO.getLogLevel(), "pass"},
		},
		1,
		0,
	}, {
		5,
		StartupProbe.getProbeType(),
		StartupProbeSockPath,
		[]struct {
			level   string
			message string
		}{
			{ERROR.getLogLevel(), "failure"},
			{ERROR.getLogLevel(), "failure"},
		},
		0,
		0,
	},
	{
		6,
		StartupProbe.getProbeType(),
		StartupProbeSockPath,
		[]struct {
			level   string
			message string
		}{
			{ERROR.getLogLevel(), "failure"},
			{ERROR.getLogLevel(), "failure"},
			{INFO.getLogLevel(), "pass"},
			{INFO.getLogLevel(), "pass"},
		},
		1,
		0,
	},
}

// TestDedupLog verifies INFO deduplication, ERROR logging,
// and recovery-success logging after a failure.
func TestDedupLog(t *testing.T) {
	for _, test := range dedupLogTests {
		socket, logs := newTestSocket(t)

		for _, entry := range test.entries {
			socket.dedupLog(test.socketPath, entry.level, entry.message, test.probeType)
		}

		passCount := strings.Count(logs.String(), "pass")
		errorCount := strings.Count(logs.String(), "failure")

		if passCount != test.expectedPass {
			t.Errorf("dedupLog() : Test%v\nExpected pass count:\t%v\nGot:\t\t%v\nLogs:\t\t%v",
				test.testNum, test.expectedPass, passCount, logs.String())
		}

		if errorCount != test.expectedError {
			t.Errorf("dedupLog() : Test%v\nExpected error count:\t%v\nGot:\t\t%v\nLogs:\t\t%v",
				test.testNum, test.expectedError, errorCount, logs.String())
		}
	}
}

// Test values - Log level formatting
var getLogLevelTests = []struct {
	testNum  int
	level    LogLevel
	expected string
}{
	{1, INFO, "INFO"},
	{2, ERROR, "ERROR"},
	{3, LogLevel(99), "INFO"},
}

// TestGetLogLevel verifies log level string mapping.
func TestGetLogLevel(t *testing.T) {
	for _, test := range getLogLevelTests {
		result := test.level.getLogLevel()

		if result != test.expected {
			t.Errorf("getLogLevel() : Test%v\nExpected:\t%v\nGot:\t\t%v",
				test.testNum, test.expected, result)
		}
	}
}

// Test values - Probe type formatting
var getProbeTypeTests = []struct {
	testNum  int
	probe    ProbeType
	expected string
}{
	{1, LivenessProbe, "LIVENESS"},
	{2, StartupProbe, "STARTUP"},
	{3, ProbeType("UNKNOWN"), ""},
}

// TestGetProbeType verifies probe type string mapping.
func TestGetProbeType(t *testing.T) {
	for _, test := range getProbeTypeTests {
		result := test.probe.getProbeType()

		if result != test.expected {
			t.Errorf("getProbeType() : Test%v\nExpected:\t%v\nGot:\t\t%v",
				test.testNum, test.expected, result)
		}
	}
}
