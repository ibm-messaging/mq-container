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
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/ibm-messaging/mq-container/internal/ready"
)

const (
	podMetadataDir      = "/run/podmetadata"
	annotationsFilePath = podMetadataDir + "/annotations"
	labelsFilePath      = podMetadataDir + "/labels"

	labelMQStatus = "mq.ibm.com/status"

	annotationWhenActive  = "mq.ibm.com/license-when-active"
	annotationWhenStandby = "mq.ibm.com/license-when-standby"
	annotationWhenReplica = "mq.ibm.com/license-when-replica"
)

// podMetaReader reads pod metadata from the Kubernetes downward API projected
// volume files. The two read functions are overridable so that unit tests can
// inject in-memory content without touching the filesystem.
type podMetaReader struct {
	readAnnotationsFile func() ([]byte, error)
	readLabelsFile      func() ([]byte, error)
}

func newPodMetaReader() *podMetaReader {
	return &podMetaReader{
		readAnnotationsFile: fileReader(annotationsFilePath),
		readLabelsFile:      fileReader(labelsFilePath),
	}
}

// annotations returns the parsed contents of the pod annotations downward API file.
func (m *podMetaReader) annotations() (map[string]string, error) {
	data, err := m.readAnnotationsFile()
	if err != nil {
		return nil, err
	}
	return parseDownwardKeyValues(data)
}

// labels returns the parsed contents of the pod labels downward API file.
func (m *podMetaReader) labels() (map[string]string, error) {
	data, err := m.readLabelsFile()
	if err != nil {
		return nil, err
	}
	return parseDownwardKeyValues(data)
}

// resolveLicenseAnnotations returns the license annotations to apply for the
// given QM status. It reads the annotations from the downward API file and
// inspects the appropriate meta-annotation to find the target values.
// Returns nil when the relevant meta-annotation is not present.
func (m *podMetaReader) resolveLicenseAnnotations(status ready.QMStatus) (map[string]string, error) {
	podAnnotations, err := m.annotations()
	if err != nil {
		return nil, fmt.Errorf("reading pod annotations: %w", err)
	}

	var metaKey string
	switch status {
	case ready.StatusActiveQM, ready.StatusRecoveryQM:
		metaKey = annotationWhenActive
	case ready.StatusStandbyQM:
		metaKey = annotationWhenStandby
	case ready.StatusReplicaQM:
		metaKey = annotationWhenReplica
	default:
		if _, found := podAnnotations[annotationWhenReplica]; found {
			metaKey = annotationWhenReplica
		} else if _, found := podAnnotations[annotationWhenStandby]; found {
			metaKey = annotationWhenStandby
		}
	}

	raw, ok := podAnnotations[metaKey]
	if !ok {
		// Meta-annotation absent — PDB annotation switching not configured.
		return nil, nil
	}

	var annotations map[string]string
	if err := json.Unmarshal([]byte(raw), &annotations); err != nil {
		return nil, fmt.Errorf("decoding %s: %w", metaKey, err)
	}
	return annotations, nil
}

// currentMatchesDesired returns true when the current pod metadata already reflects
// the desired label and annotations, meaning no PATCH is needed. Any read or parse
// error is treated as "unknown state" and returns false so that the caller
// proceeds with the PATCH.
func (m *podMetaReader) currentMatchesDesired(wantLabel string, wantAnnotations map[string]string) bool {
	currentLabels, err := m.labels()
	if err != nil {
		return false
	}
	if currentLabels[labelMQStatus] != wantLabel {
		return false
	}

	if len(wantAnnotations) == 0 {
		return true
	}

	currentAnnotations, err := m.annotations()
	if err != nil {
		return false
	}
	for k, v := range wantAnnotations {
		if currentAnnotations[k] != v {
			return false
		}
	}
	return true
}

// fileReader returns a zero-argument read function for a fixed path. It is used
// to populate the default read functions on podMetaReader.
func fileReader(path string) func() ([]byte, error) {
	return func() ([]byte, error) {
		// #nosec G304 - path is always one of the two fixed downward API constants
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("reading downward API file %s: %w", path, err)
		}
		return data, nil
	}
}

// parseDownwardKeyValues parses the Kubernetes downward API annotations or
// labels file format. Each line has the form: key="value" where the value is
// a double-quoted Go string literal.
func parseDownwardKeyValues(data []byte) (map[string]string, error) {
	result := map[string]string{}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		eq := strings.IndexByte(line, '=')
		if eq < 0 {
			continue
		}
		key := line[:eq]
		// The value is a double-quoted Go string literal.
		val, err := strconv.Unquote(line[eq+1:])
		if err != nil {
			return nil, fmt.Errorf("parsing key %q: %w", key, err)
		}
		result[key] = val
	}
	return result, nil
}
