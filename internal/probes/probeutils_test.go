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
	"fmt"
	"strings"
	"testing"
	"time"
)

var testStartTime = time.Date(2026, 4, 28, 8, 0, 0, 0, time.UTC)
var testEndTime = testStartTime.Add(25 * time.Millisecond)

func int64Ptr(v int64) *int64 {
	return &v
}

func timePtr(v time.Time) *time.Time {
	return &v
}

// Test values - Probe summary output
var writeProbeSummaryTests = []struct {
	testNum          int
	state            *ProbeLoggingState
	expectedContains []string
	expectedEmpty    bool
}{
	{1, nil, nil, true},
	{2, &ProbeLoggingState{}, nil, true},
	{3, &ProbeLoggingState{
		LivenessProbeLoggingState: &LivenessProbeLoggingState{
			CurrentRun: &ProbeLoggingExecution{
				ProbeType:  LivenessProbe,
				StartTime:  timePtr(testStartTime),
				EndTime:    timePtr(testEndTime),
				Status:     ProbePassed,
				Duration:   int64Ptr(25),
				LogMessage: "Liveness Probe Passed: Output: QMNAME(testQM) STATUS(RUNNING)",
				LogLevel:   INFO,
			},
		},
	},
		[]string{
			"----- Start Liveness Probe Summary -----",
			"Last Run: Passed",
			"Duration=25ms",
			"Note: Duration measures chkmqhealthy execution time and excludes Kubernetes probe scheduling overhead",
			"----- End Liveness Probe Summary -----",
		}, false,
	},
	{4, &ProbeLoggingState{
		LivenessProbeLoggingState: &LivenessProbeLoggingState{
			CurrentRun: &ProbeLoggingExecution{
				ProbeType: LivenessProbe,
				StartTime: timePtr(testStartTime),
				Status:    ProbeIncomplete,
				Duration:  int64Ptr(5133),
			},
			PreviousRuns: [2]ProbeLoggingExecution{
				{
					ProbeType:  LivenessProbe,
					StartTime:  timePtr(testStartTime.Add(-10 * time.Second)),
					EndTime:    timePtr(testStartTime.Add(-9 * time.Second)),
					Status:     ProbePassed,
					Duration:   int64Ptr(1000),
					LogMessage: "Liveness Probe Passed: Output: QMNAME(testQM) STATUS(RUNNING)",
					LogLevel:   INFO,
				},
				{
					ProbeType:  LivenessProbe,
					StartTime:  timePtr(testStartTime.Add(-20 * time.Second)),
					EndTime:    timePtr(testStartTime.Add(-19 * time.Second)),
					Status:     ProbeFailed,
					Duration:   int64Ptr(1000),
					LogMessage: "Liveness Probe Passed: chkmqhealthy Error",
					LogLevel:   ERROR,
				},
			},
		},
	},
		[]string{
			"----- Start Liveness Probe Summary -----",
			"Last Run: Incomplete",
			"Previous Run 1: Passed",
			"Previous Run 2: Failed",
			"Duration=5133ms",
			"Duration=1000ms",
			"Note: Duration measures chkmqhealthy execution time",
			"----- End Liveness Probe Summary -----",
		}, false,
	}, {
		5,
		&ProbeLoggingState{
			LivenessProbeLoggingState: &LivenessProbeLoggingState{
				CurrentRun: nil,
			},
			StartupProbeLoggingState: &StartupProbeLoggingState{
				CurrentRun: &ProbeLoggingExecution{
					ProbeType:    StartupProbe,
					Status:       ProbeFailed,
					LogMessage:   "Startup Probe Failed: QueueManager is not started",
					LogLevel:     ERROR,
					AttemptCount: 24,
				},
			},
		},
		[]string{
			"----- Start Startup Probe Summary -----",
			"Last State: Failed (Attempts=24 Details=Startup Probe Failed: QueueManager is not started)",
			"----- End Startup Probe Summary -----",
		}, false,
	}, {
		6,
		&ProbeLoggingState{
			LivenessProbeLoggingState: &LivenessProbeLoggingState{
				CurrentRun: nil,
			},
			StartupProbeLoggingState: &StartupProbeLoggingState{
				CurrentRun: &ProbeLoggingExecution{
					ProbeType:    StartupProbe,
					Status:       ProbeIncomplete,
					AttemptCount: 12,
				},
			},
		},
		[]string{
			"----- Start Startup Probe Summary -----",
			"Last State: Incomplete (Attempts=12)",
			"----- End Startup Probe Summary -----",
		}, false,
	},
	{
		7,
		&ProbeLoggingState{
			LivenessProbeLoggingState: &LivenessProbeLoggingState{
				CurrentRun: &ProbeLoggingExecution{
					ProbeType:  LivenessProbe,
					StartTime:  timePtr(testStartTime),
					EndTime:    timePtr(testEndTime),
					Status:     ProbePassed,
					Duration:   int64Ptr(25),
					LogMessage: "Liveness Probe Passed",
					LogLevel:   INFO,
				},
			},
			StartupProbeLoggingState: &StartupProbeLoggingState{
				CurrentRun: &ProbeLoggingExecution{
					ProbeType:    StartupProbe,
					Status:       ProbePassed,
					LogMessage:   "Startup Probe Passed",
					LogLevel:     INFO,
					AttemptCount: 10,
				},
			},
		},
		[]string{
			"----- Start Liveness Probe Summary -----",
			"Last Run: Passed",
			"----- End Liveness Probe Summary -----",
		}, false,
	},
}

// TestWriteProbeSummary verifies summary output for nil, empty,
// current-only, and current-with-history probe states.
func TestWriteProbeSummary(t *testing.T) {
	for _, test := range writeProbeSummaryTests {

		log, buf := newTestLogger(t)

		WriteProbeSummary(test.state, log)

		output := buf.String()

		if test.expectedEmpty && output != "" {
			t.Errorf("WriteProbeSummary() : Test%v\nExpected empty output\nGot:\t%v", test.testNum, output)
		}

		for _, expected := range test.expectedContains {
			if !strings.Contains(output, expected) {
				t.Errorf("WriteProbeSummary() : Test%v\nExpected output to contain:\t%v\nGot:\t\t\t%v", test.testNum, expected, output)
			}
		}

		livenessStarted := test.state != nil &&
			test.state.LivenessProbeLoggingState != nil &&
			test.state.LivenessProbeLoggingState.CurrentRun != nil

		startupStatePresent := test.state != nil &&
			test.state.StartupProbeLoggingState != nil &&
			test.state.StartupProbeLoggingState.CurrentRun != nil

		if livenessStarted && startupStatePresent && strings.Contains(output, "Startup Probe Summary") {
			t.Errorf("WriteProbeSummary() : Test%v\nExpected startup summary to be suppressed when liveness has started\nGot:\t%v", test.testNum, output)
		}

	}
}

// Test values - Summary duration
var formatDurationSummaryTests = []struct {
	testNum   int
	duration  *int64
	startTime *time.Time
	status    ProbeStatus
	expected  string
}{
	{1, int64Ptr(0), timePtr(testStartTime), ProbePassed, "0ms"},
	{2, int64Ptr(15), timePtr(testStartTime), ProbePassed, "15ms"},
	{3, int64Ptr(4600), timePtr(testStartTime), ProbePassed, "4600ms"},
	{4, int64Ptr(20), timePtr(testStartTime), ProbeFailed, "20ms"},
	{5, nil, timePtr(testStartTime), ProbePassed, "N/A"},
	{6, nil, timePtr(testStartTime), ProbeFailed, "N/A"},
	{7, nil, nil, ProbeIncomplete, "N/A"},
	{8, int64Ptr(5003), timePtr(testStartTime), ProbeIncomplete, "5003ms"},
}

// TestFormatDurationForSummary verifies duration formatting for completed,
// incomplete, zero-duration, and missing-duration values.
func TestFormatDurationForSummary(t *testing.T) {
	for _, test := range formatDurationSummaryTests {
		result := formatDurationForSummary(test.duration, test.startTime, test.status)

		if result != test.expected {
			t.Errorf("formatDurationForSummary() : Test%v\nExpected:\t%v\nGot:\t\t%v",
				test.testNum, test.expected, result)
		}
	}
}

// Test values - Summary duration with no recorded duration
var formatDurationSummaryElapsedTests = []struct {
	testNum       int
	startOffset   time.Duration
	minDurationMs int64
	maxDurationMs int64
}{
	{1, -50 * time.Millisecond, 1, 500},
	{2, -1500 * time.Millisecond, 1000, 2500},
}

// TestFormatDurationForSummaryElapsed verifies that incomplete probes without
// recorded duration use elapsed time from their start time.
func TestFormatDurationForSummaryElapsed(t *testing.T) {
	for _, test := range formatDurationSummaryElapsedTests {
		startTime := time.Now().Add(test.startOffset)

		result := formatDurationForSummary(nil, &startTime, ProbeIncomplete)

		if !strings.HasSuffix(result, "ms") {
			t.Errorf("formatDurationForSummary() : Test%v\nExpected duration suffix:\tms\nGot:\t\t\t%v",
				test.testNum, result)
			continue
		}

		var durationMs int64
		_, err := fmt.Sscanf(result, "%dms", &durationMs)
		if err != nil {
			t.Errorf("formatDurationForSummary() : Test%v\nFailed to parse duration:\t%v\nGot:\t\t\t%v",
				test.testNum, err, result)
			continue
		}

		if durationMs < test.minDurationMs || durationMs > test.maxDurationMs {
			t.Errorf("formatDurationForSummary() : Test%v\nExpected range:\t%v-%vms\nGot:\t\t%vms",
				test.testNum, test.minDurationMs, test.maxDurationMs, durationMs)
		}
	}
}

// Test values - Probe summary
var formatProbeSummaryTests = []struct {
	testNum   int
	prefix    string
	probe     *ProbeLoggingExecution
	probeType ProbeType
	expected  string
}{
	{1, "Last Run", nil, LivenessProbe, ""},
	{
		2,
		"Last Run",
		&ProbeLoggingExecution{ProbeType: LivenessProbe, StartTime: timePtr(testStartTime), Status: ProbeIncomplete, Duration: int64Ptr(5131)},
		LivenessProbe,
		"Last Run: Incomplete (Started=2026-04-28T08:00:00Z Duration=5131ms)",
	},
	{
		3,
		"Previous Run 1",
		&ProbeLoggingExecution{
			ProbeType:  LivenessProbe,
			StartTime:  timePtr(testStartTime),
			EndTime:    timePtr(testEndTime),
			Status:     ProbePassed,
			Duration:   int64Ptr(25),
			LogMessage: " Liveness Probe Passed: Output: QMNAME(testQM) STATUS(RUNNING) ",
			LogLevel:   INFO,
		},
		LivenessProbe,
		"Previous Run 1: Passed (Started=2026-04-28T08:00:00Z Completed=2026-04-28T08:00:00Z Duration=25ms Details=Liveness Probe Passed: Output: QMNAME(testQM) STATUS(RUNNING))",
	},
	{
		4,
		"Previous Run 2",
		&ProbeLoggingExecution{
			ProbeType:  LivenessProbe,
			StartTime:  timePtr(testStartTime),
			EndTime:    timePtr(testEndTime),
			Status:     ProbeFailed,
			Duration:   int64Ptr(20),
			LogMessage: "Liveness Probe Failed: chkmqhealthy error",
			LogLevel:   ERROR,
		},
		LivenessProbe,
		"Previous Run 2: Failed (Started=2026-04-28T08:00:00Z Completed=2026-04-28T08:00:00Z Duration=20ms Details=Liveness Probe Failed: chkmqhealthy error)",
	},
	{
		5,
		"Last Run",
		&ProbeLoggingExecution{ProbeType: LivenessProbe, Status: ProbeIncomplete, Duration: int64Ptr(5213)},
		LivenessProbe,
		"Last Run: Incomplete (Started=N/A Duration=5213ms)",
	},
	{
		6,
		"Previous Run 1",
		&ProbeLoggingExecution{ProbeType: LivenessProbe, StartTime: timePtr(testStartTime), EndTime: nil, Status: ProbePassed, Duration: nil, LogMessage: "", LogLevel: INFO},
		LivenessProbe,
		"Previous Run 1: Passed (Started=2026-04-28T08:00:00Z Completed=N/A Duration=N/A Details=)",
	},
	{
		7,
		"Last Run",
		&ProbeLoggingExecution{ProbeType: StartupProbe, Status: ProbeIncomplete, AttemptCount: 24},
		StartupProbe,
		"Last Run: Incomplete (Attempts=24)",
	},
	{
		8,
		"Last State",
		&ProbeLoggingExecution{ProbeType: StartupProbe, Status: ProbeFailed, AttemptCount: 24, LogMessage: " Startup Probe Failed: QueueManager is not started "},
		StartupProbe,
		"Last State: Failed (Attempts=24 Details=Startup Probe Failed: QueueManager is not started)",
	},
	{
		9,
		"Last Run",
		&ProbeLoggingExecution{ProbeType: StartupProbe, Status: ProbePassed, AttemptCount: 5, LogMessage: " Startup Probe Passed: QueueManager started successfully "},
		StartupProbe,
		"Last Run: Passed (Attempts=5 Details=Startup Probe Passed: QueueManager started successfully)",
	},
}

// TestFormatProbeSummary verifies summary formatting for nil, incomplete,
// passed, failed, and missing-field probe execution entries.
func TestFormatProbeSummary(t *testing.T) {
	for _, test := range formatProbeSummaryTests {
		result := formatProbeSummary(test.prefix, test.probe, test.probeType)

		if result != test.expected {
			t.Errorf("formatProbeSummary() : Test%v\nExpected:\t%v\nGot:\t\t%v",
				test.testNum, test.expected, result)
		}
	}
}

// Test values - Time pointer formatting
var timePtrToStringTests = []struct {
	testNum  int
	input    *time.Time
	expected string
}{
	{1, timePtr(testStartTime), "2026-04-28T08:00:00Z"},
	{2, nil, "N/A"},
}

// TestTimePtrToString verifies formatting of available and missing time values.
func TestTimePtrToString(t *testing.T) {
	for _, test := range timePtrToStringTests {
		result := timePtrToString(test.input)

		if result != test.expected {
			t.Errorf("timePtrToString() : Test%v\nExpected:\t%v\nGot:\t\t%v",
				test.testNum, test.expected, result)
		}
	}
}

// Test values - int64 pointer formatting
var int64PtrToStringTests = []struct {
	testNum  int
	input    *int64
	expected string
}{
	{1, int64Ptr(0), "0"},
	{2, int64Ptr(15), "15"},
	{3, int64Ptr(4600), "4600"},
	{4, nil, "N/A"},
}

// TestInt64PtrToString verifies formatting of available and missing int64 values.
func TestInt64PtrToString(t *testing.T) {
	for _, test := range int64PtrToStringTests {
		result := int64PtrToString(test.input)

		if result != test.expected {
			t.Errorf("int64PtrToString() : Test%v\nExpected:\t%v\nGot:\t\t%v",
				test.testNum, test.expected, result)
		}
	}
}
