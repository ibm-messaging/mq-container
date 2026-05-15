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
	"time"

	"github.com/ibm-messaging/mq-container/pkg/logger"
)

// WriteProbeSummary logs the probe execution summary.
//   - liveness summary includes the current run and up to two previous runs
//   - startup summary includes the latest startup probe state when liveness has not started
func WriteProbeSummary(probeLoggingState *ProbeLoggingState, log *logger.Logger) {
	if probeLoggingState != nil {

		// prefer liveness probe summary once liveness probe has started; otherwise fall back to startup probe summary
		if probeLoggingState.LivenessProbeLoggingState != nil && probeLoggingState.LivenessProbeLoggingState.CurrentRun != nil {
			livenessState := probeLoggingState.LivenessProbeLoggingState

			log.Println("----- Start Liveness Probe Summary -----")

			if livenessState.CurrentRun != nil {
				log.Printf("%s", formatProbeSummary("Last Run", livenessState.CurrentRun, LivenessProbe))
			}

			for i, probeHistory := range livenessState.PreviousRuns {
				if probeHistory.StartTime != nil {
					history := probeHistory
					log.Printf("%s", formatProbeSummary(fmt.Sprintf("Previous Run %d", i+1), &history, LivenessProbe))
				}
			}

			log.Println("Note: Duration measures chkmqhealthy execution time and excludes Kubernetes probe scheduling overhead")

			log.Println("----- End Liveness Probe Summary -----")
		} else if probeLoggingState.StartupProbeLoggingState != nil && probeLoggingState.StartupProbeLoggingState.CurrentRun != nil {
			startupState := probeLoggingState.StartupProbeLoggingState

			log.Println("----- Start Startup Probe Summary -----")

			if startupState.CurrentRun != nil {
				log.Printf("%s", formatProbeSummary("Last State", startupState.CurrentRun, StartupProbe))
			}

			log.Println("----- End Startup Probe Summary -----")
		}
	}
}

// formatProbeSummary formats a single probe execution entry for summary output.
func formatProbeSummary(prefix string, probeLoggingExecution *ProbeLoggingExecution, probeType ProbeType) string {
	if probeLoggingExecution == nil {
		return ""
	}

	status := probeLoggingExecution.Status.getProbeStatus()

	if probeLoggingExecution.Status == ProbeIncomplete {
		if probeType == StartupProbe {
			return fmt.Sprintf(
				"%s: %s (Attempts=%d)",
				prefix,
				status,
				probeLoggingExecution.AttemptCount,
			)
		} else if probeType == LivenessProbe {
			return fmt.Sprintf(
				"%s: %s (Started=%s Duration=%s)",
				prefix,
				status,
				timePtrToString(probeLoggingExecution.StartTime),
				formatDurationForSummary(probeLoggingExecution.Duration, probeLoggingExecution.StartTime, probeLoggingExecution.Status),
			)
		}
	}

	if probeType == StartupProbe {
		return fmt.Sprintf(
			"%s: %s (Attempts=%d Details=%s)",
			prefix,
			status,
			probeLoggingExecution.AttemptCount,
			strings.TrimSpace(probeLoggingExecution.LogMessage),
		)
	}
	return fmt.Sprintf(
		"%s: %s (Started=%s Completed=%s Duration=%s Details=%s)",
		prefix,
		status,
		timePtrToString(probeLoggingExecution.StartTime),
		timePtrToString(probeLoggingExecution.EndTime),
		formatDurationForSummary(probeLoggingExecution.Duration, probeLoggingExecution.StartTime, probeLoggingExecution.Status),
		strings.TrimSpace(probeLoggingExecution.LogMessage),
	)
}

// timePtrToString safely formats a time pointer into RFC3339 format
//   - returns "N/A" when time is not available
func timePtrToString(t *time.Time) string {
	if t == nil {
		return "N/A"
	}
	return t.Format(time.RFC3339)
}

// int64PtrToString safely formats an int64 pointer
//   - returns "N/A" when value is not available
func int64PtrToString(i *int64) string {
	if i == nil {
		return "N/A"
	}
	return fmt.Sprintf("%d", *i)
}

// formatDurationForSummary returns probe duration in milliseconds for summary output.
func formatDurationForSummary(duration *int64, startTime *time.Time, status ProbeStatus) string {
	if duration != nil {
		return fmt.Sprintf("%dms", *duration)
	}

	if status == ProbeIncomplete {
		if startTime == nil {
			return "N/A"
		}
		return fmt.Sprintf("%dms", time.Since(*startTime).Milliseconds())
	}

	return "N/A"
}
