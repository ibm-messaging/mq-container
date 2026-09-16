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
package kubeapi

import (
	"strings"
	"testing"

	"github.com/ibm-messaging/mq-container/internal/ready"
)

func TestResolveLicenseAnnotations(t *testing.T) {
	const (
		activeAnnotations  = `{"productChargedContainers":"All","productID":"1234"}`
		standbyAnnotations = `{"productChargedContainers":"None","productID":"1234"}`
		replicaAnnotations = `{"productChargedContainers":"None","productID":"5678"}`
	)

	tests := []struct {
		name   string
		status ready.QMStatus
		// annotations is the downward-API annotations file content
		// nil represents a file read error.
		annotations []byte
		// wantAnnotations are the exepcted returned annotations.
		wantAnnotations map[string]string
		wantErr         bool
		wantErrContains string
	}{
		{
			name:   "StatusActiveQM - resolves annotationWhenActive",
			status: ready.StatusActiveQM,
			annotations: downwardAPILines(map[string]string{
				annotationWhenActive: activeAnnotations,
			}),
			wantAnnotations: map[string]string{
				"productChargedContainers": "All",
				"productID":                "1234",
			},
		},
		{
			name:   "StatusActiveQM - no active annotations present",
			status: ready.StatusActiveQM,
			annotations: downwardAPILines(map[string]string{
				annotationWhenStandby: standbyAnnotations,
			}),
			wantAnnotations: nil,
		},
		{
			name:   "StatusStandbyQM - resolves annotationWhenStandby",
			status: ready.StatusStandbyQM,
			annotations: downwardAPILines(map[string]string{
				annotationWhenStandby: standbyAnnotations,
			}),
			wantAnnotations: map[string]string{
				"productChargedContainers": "None",
				"productID":                "1234",
			},
		},
		{
			name:   "StatusReplicaQM - resolves annotationWhenReplica",
			status: ready.StatusReplicaQM,
			annotations: downwardAPILines(map[string]string{
				annotationWhenReplica: replicaAnnotations,
			}),
			wantAnnotations: map[string]string{
				"productChargedContainers": "None",
				"productID":                "5678",
			},
		},
		{
			// Recovery and active share annotations
			name:   "StatusRecoveryLeaderQM - resolves annotationWhenActive",
			status: ready.StatusRecoveryLeaderQM,
			annotations: downwardAPILines(map[string]string{
				annotationWhenActive: activeAnnotations,
			}),
			wantAnnotations: map[string]string{
				"productChargedContainers": "All",
				"productID":                "1234",
			},
		},
		{
			name:            "StatusStarting - no meta-key, no fallback annotation, returns nil",
			status:          ready.StatusStarting,
			annotations:     downwardAPILines(map[string]string{}),
			wantAnnotations: nil,
		},
		{
			name:   "StatusStarting - falls back to annotationWhenReplica",
			status: ready.StatusStarting,
			annotations: downwardAPILines(map[string]string{
				annotationWhenReplica: replicaAnnotations,
			}),
			wantAnnotations: map[string]string{
				"productChargedContainers": "None",
				"productID":                "5678",
			},
		},
		{
			name:   "StatusStarting - falls back to annotationWhenStandby",
			status: ready.StatusStarting,
			annotations: downwardAPILines(map[string]string{
				annotationWhenStandby: standbyAnnotations,
			}),
			wantAnnotations: map[string]string{
				"productChargedContainers": "None",
				"productID":                "1234",
			},
		},
		{
			name:            "StatusUnknown - no meta-key, no fallback annotation, returns nil",
			status:          ready.StatusUnknown,
			annotations:     downwardAPILines(map[string]string{}),
			wantAnnotations: nil,
		},
		{
			name:   "StatusUnknown - falls back to annotationWhenReplica",
			status: ready.StatusUnknown,
			annotations: downwardAPILines(map[string]string{
				annotationWhenReplica: replicaAnnotations,
			}),
			wantAnnotations: map[string]string{
				"productChargedContainers": "None",
				"productID":                "5678",
			},
		},
		{
			name:   "StatusUnknown - falls back to annotationWhenStandby",
			status: ready.StatusUnknown,
			annotations: downwardAPILines(map[string]string{
				annotationWhenStandby: standbyAnnotations,
			}),
			wantAnnotations: map[string]string{
				"productChargedContainers": "None",
				"productID":                "1234",
			},
		},
		{
			name:            "invalid QMStatus (99) - no meta-key, no fallback annotation, returns nil",
			status:          ready.QMStatus(99),
			annotations:     downwardAPILines(map[string]string{}),
			wantAnnotations: nil,
		},
		{
			name:   "invalid QMStatus (99) - falls back to annotationWhenReplica",
			status: ready.QMStatus(99),
			annotations: downwardAPILines(map[string]string{
				annotationWhenReplica: replicaAnnotations,
			}),
			wantAnnotations: map[string]string{
				"productChargedContainers": "None",
				"productID":                "5678",
			},
		},
		{
			name:            "StatusActiveQM - meta-annotation absent, returns nil",
			status:          ready.StatusActiveQM,
			annotations:     downwardAPILines(map[string]string{}),
			wantAnnotations: nil,
		},
		{
			name:            "annotations file unreadable",
			status:          ready.StatusActiveQM,
			annotations:     nil,
			wantErr:         true,
			wantErrContains: "reading pod annotations",
		},
		{
			name:   "StatusActiveQM - malformed JSON in meta-annotation",
			status: ready.StatusActiveQM,
			annotations: downwardAPILines(map[string]string{
				annotationWhenActive: `not-valid-json`,
			}),
			wantErr:         true,
			wantErrContains: annotationWhenActive,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create reader for annotations
			m := metaReaderWith(tt.annotations, downwardAPILines(map[string]string{}))

			got, err := m.resolveLicenseAnnotations(tt.status)

			if tt.wantErr {
				if err == nil {
					t.Errorf("resolveLicenseAnnotations() expected an error, got nil")
					return
				}
				if tt.wantErrContains != "" && !strings.Contains(err.Error(), tt.wantErrContains) {
					t.Errorf("resolveLicenseAnnotations() error = %q, want it to contain %q", err.Error(), tt.wantErrContains)
				}
				return
			}

			if err != nil {
				t.Errorf("resolveLicenseAnnotations() unexpected error: %v", err)
				return
			}

			// Verify the returned map matches the expected annotations exactly.
			if len(got) != len(tt.wantAnnotations) {
				t.Errorf("resolveLicenseAnnotations() returned %d annotations, want %d; got=%v", len(got), len(tt.wantAnnotations), got)
				return
			}
			for k, wantVal := range tt.wantAnnotations {
				gotVal, ok := got[k]
				if !ok || gotVal != wantVal {
					t.Errorf("resolveLicenseAnnotations() annotation[%q] = %q, want %q", k, gotVal, wantVal)
				}
			}
		})
	}
}

func TestChangesRequired(t *testing.T) {
	const (
		wantLabel  = "Active"
		otherLabel = "Standby"
	)

	var (
		activeAnnotations = map[string]string{
			"productChargedContainers": "All",
			"productID":                "1234",
		}

		// Active annotations on downward API
		matchingAnnotationsFile = downwardAPILines(map[string]string{
			"productChargedContainers": "All",
			"productID":                "1234",
		})

		// Standby annotations on downward API
		mismatchedAnnotationsFile = downwardAPILines(map[string]string{
			"productChargedContainers": "None",
			"productID":                "1234",
		})

		// Live label file whose value matches the desired label.
		matchingLabelsFile = downwardAPILines(map[string]string{
			labelMQStatus: wantLabel,
		})

		// Live label file whose value differs from the desired label.
		mismatchedLabelsFile = downwardAPILines(map[string]string{
			labelMQStatus: otherLabel,
		})
	)

	tests := []struct {
		name                string
		wantAnnotations     map[string]string
		annotationsData     []byte
		labelsData          []byte
		wantNeeded          bool
		wantReasonContains  []string
	}{
		{
			name:            "label and annotations match",
			wantAnnotations: activeAnnotations,
			labelsData:      matchingLabelsFile,
			annotationsData: matchingAnnotationsFile,
			wantNeeded:      false,
		},
		{
			name:               "label matches, annotations mismatch",
			wantAnnotations:    activeAnnotations,
			labelsData:         matchingLabelsFile,
			annotationsData:    mismatchedAnnotationsFile,
			wantNeeded:         true,
			wantReasonContains: []string{"productChargedContainers"},
		},
		{
			name:               "label mismatch",
			wantAnnotations:    activeAnnotations,
			labelsData:         mismatchedLabelsFile,
			annotationsData:    nil,
			wantNeeded:         true,
			wantReasonContains: []string{"'Standby' to 'Active'"},
		},
		{
			name:            "annotations are nil",
			wantAnnotations: nil,
			labelsData:      matchingLabelsFile,
			annotationsData: nil,
			wantNeeded:      false,
		},
		{
			// The labels file cannot be read — state is unknown, PATCH required.
			name:               "labels file unreadable",
			wantAnnotations:    activeAnnotations,
			labelsData:         nil, // nil causes metaReaderWith to return a read error
			annotationsData:    matchingAnnotationsFile,
			wantNeeded:         true,
			wantReasonContains: []string{"failed to read existing label"},
		},
		{
			// The label matches but the annotations file cannot be read — PATCH required.
			name:               "annotations file unreadable",
			wantAnnotations:    activeAnnotations,
			labelsData:         matchingLabelsFile,
			annotationsData:    nil, // nil causes metaReaderWith to return a read error
			wantNeeded:         true,
			wantReasonContains: []string{"failed to read current annotations"},
		},
		{
			// Both label and annotation stale — reason must mention both.
			name:               "label and annotations both stale",
			wantAnnotations:    activeAnnotations,
			labelsData:         mismatchedLabelsFile,
			annotationsData:    mismatchedAnnotationsFile,
			wantNeeded:         true,
			wantReasonContains: []string{"'Standby' to 'Active'", "productChargedContainers"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := metaReaderWith(tt.annotationsData, tt.labelsData)

			needed, reason := m.changesRequired(wantLabel, tt.wantAnnotations)

			if needed != tt.wantNeeded {
				t.Errorf("changesRequired() needed = %v, want %v (reason: %q)", needed, tt.wantNeeded, reason)
			}
			if tt.wantNeeded && reason == "" {
				t.Errorf("changesRequired() reason is empty, want non-empty")
			}
			if !tt.wantNeeded && reason != "" {
				t.Errorf("changesRequired() reason = %q, want empty", reason)
			}
			for _, fragment := range tt.wantReasonContains {
				if !strings.Contains(reason, fragment) {
					t.Errorf("changesRequired() reason = %q, want it to contain %q", reason, fragment)
				}
			}
		})
	}
}
