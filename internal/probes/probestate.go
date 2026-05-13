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
	"os"
	"strings"
	"time"
)

type ProbeStatus string

const (
	ProbePassed     ProbeStatus = "PASSED"
	ProbeFailed     ProbeStatus = "FAILED"
	ProbeIncomplete ProbeStatus = "INCOMPLETE"

	ProbeLoggingSource = "probes"
)

type ProbeLoggingExecution struct {
	ProbeType  ProbeType   `json:"probeType"`
	StartTime  *time.Time  `json:"startTime,omitempty"`
	EndTime    *time.Time  `json:"endTime,omitempty"`
	Status     ProbeStatus `json:"status"`
	Duration   *int64      `json:"duration,omitempty"`
	LogMessage string      `json:"logMessage,omitempty"`
	LogLevel   LogLevel    `json:"logLevel,omitempty"`
}

type LivenessProbeLoggingState struct {
	CurrentRun   *ProbeLoggingExecution
	PreviousRuns [2]ProbeLoggingExecution
}

type ProbeLoggingState struct {
	LivenessProbeLoggingState *LivenessProbeLoggingState
}

// NewProbeState initializes and returns an empty ProbeLoggingState.
//   - Liveness probe state is initialized with no current run
//   - Previous runs are maintained as a fixed-size history (last two runs)
func NewProbeState() *ProbeLoggingState {
	return &ProbeLoggingState{
		LivenessProbeLoggingState: &LivenessProbeLoggingState{
			CurrentRun:   nil,
			PreviousRuns: [2]ProbeLoggingExecution{},
		},
	}
}

// IsProbeLoggingEnabled returns true if probe logging is enabled for the given probe type
// - determined by the presence of the corresponding probe socket
func IsProbeLoggingEnabled(probeType ProbeType) bool {

	switch probeType {
	case LivenessProbe:
		if _, err := os.Stat(LivenessProbeSockPath); err != nil {
			return false
		}
		return true
	default:
		return false
	}
}

// InitializeProbeLoggingExecution creates a new ProbeLoggingExecution for a probe start event.
//   - immediately emits the INCOMPLETE event for liveness probes
func InitializeProbeLoggingExecution(probeType ProbeType, probeStartTime time.Time, probeStatus ProbeStatus) *ProbeLoggingExecution {
	probeLoggingExecution := &ProbeLoggingExecution{
		ProbeType: probeType,
		StartTime: &probeStartTime,
		Status:    probeStatus,
	}

	if probeType == LivenessProbe {
		SendProbeLoggingExecution(probeLoggingExecution, LivenessProbeSockPath)
	}

	return probeLoggingExecution
}

// LogLivenessProbeMessage finalizes a liveness probe execution.
//   - computes duration based on start and end time
//   - determines probe outcome (PASSED / FAILED)
//   - builds appropriate log message and sends it to the probe socket
func (pe *ProbeLoggingExecution) LogLivenessProbeMessage(probeEndTime time.Time, healthy bool, out string, err error) {
	duration := probeEndTime.Sub(*pe.StartTime).Milliseconds()
	probeDetails := buildLivenessProbeOutputMessage(out)

	if err != nil {
		pe.logLivenessProbeMessageHelper(
			probeEndTime,
			ProbeFailed,
			duration,
			buildLivenessProbeFailureMessage(err.Error(), probeDetails),
			ERROR,
		)
		return
	}

	if !healthy {
		pe.logLivenessProbeMessageHelper(
			probeEndTime,
			ProbeFailed,
			duration,
			buildLivenessProbeFailureMessage("QueueManager is not healthy", probeDetails),
			ERROR,
		)
		return
	}

	pe.logLivenessProbeMessageHelper(
		probeEndTime,
		ProbePassed,
		duration,
		buildLivenessProbeSuccessMessage(probeDetails),
		INFO,
	)

}

// logLivenessProbeMessageHelper updates probe execution fields and emits the log.
//   - this is the internal helper used after outcome determination
//   - ensures state is fully populated before sending to socket
func (pe *ProbeLoggingExecution) logLivenessProbeMessageHelper(probeEndTime time.Time, status ProbeStatus, duration int64, message string, logLevel LogLevel) {
	pe.EndTime = &probeEndTime
	pe.Status = status
	pe.Duration = &duration
	pe.LogMessage = message
	pe.LogLevel = logLevel

	SendProbeLoggingExecution(pe, LivenessProbeSockPath)

}

func (ps ProbeStatus) getProbeStatus() string {
	switch ps {
	case ProbeIncomplete:
		return "Incomplete"
	case ProbePassed:
		return "Passed"
	case ProbeFailed:
		return "Failed"
	default:
		return "Unknown"
	}
}

// buildLivenessProbeOutputMessage formats chkmqhealthy output for logging
//   - trims whitespace
//   - returns empty string if no output is present
func buildLivenessProbeOutputMessage(out string) string {
	trimmed := strings.TrimSpace(out)
	if trimmed == "" {
		return ""
	}

	trimmed = strings.Join(strings.Fields(trimmed), " ")

	return fmt.Sprintf("Output: %s", trimmed)
}

// buildLivenessProbeFailureMessage constructs a failure log message
//   - includes failure reason
//   - optionally appends probe output details if available
func buildLivenessProbeFailureMessage(reason, details string) string {
	if details == "" {
		return fmt.Sprintf("Liveness Probe Failed: %s", reason)
	}
	return fmt.Sprintf("Liveness Probe Failed: %s, %s", reason, details)
}

// buildLivenessProbeSuccessMessage constructs a success log message
//   - includes probe output details when present
func buildLivenessProbeSuccessMessage(details string) string {
	if details == "" {
		return "Liveness Probe Passed"
	}
	return fmt.Sprintf("Liveness Probe Passed: %s", details)
}
