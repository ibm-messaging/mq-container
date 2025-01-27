/*
© Copyright IBM Corporation 2025

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

// Package metrics contains code to provide metrics for the queue manager
package metrics

import (
	"os"
	"testing"

	"github.com/ibm-messaging/mq-container/internal/pathutils"
	"github.com/ibm-messaging/mq-container/pkg/logger"
)

var isHTTPSMetricsEnabledTests = []struct {
	testNum       int
	createDir     bool
	files         []string
	enabled       bool
	expectedError bool
}{
	{1, false, []string{}, false, false},
	{2, true, []string{}, false, false},
	{3, true, []string{"tls.key", "tls.crt"}, true, false},
	{4, true, []string{"tls.key", "tls.key"}, false, true},
	{5, true, []string{"tls.crt", "tls.crt"}, false, true},
	{6, true, []string{"tls.key"}, false, true},
	{7, true, []string{"tls.crt"}, false, true},
	{8, true, []string{"tls.key", "random.crt"}, false, true},
	{9, true, []string{"random.key", "tls.crt"}, false, true},
	{10, true, []string{"tls.txt", "tls.crt"}, false, true},
	{11, true, []string{"tls.key", "tls.txt"}, false, true},
	{12, true, []string{"tls.key", "tls.crt", "random.key", "random.crt"}, true, false},
	{13, true, []string{"tls.key", "tls.crt", "random.txt"}, true, false},
}

func TestIsHTTPSMetricsEnabled(t *testing.T) {

	log, _ := logger.NewLogger(os.Stdout, true, false, "test")

	for _, isHTTPSMetricsEnabledTest := range isHTTPSMetricsEnabledTests {

		var tmpDir string
		var err error

		if isHTTPSMetricsEnabledTest.createDir {
			tmpDir, err = os.MkdirTemp("", "tmp")
			if err != nil {
				t.Fatalf("Failed to create temporary directory: %v", err)
			}
		}
		defer os.RemoveAll(tmpDir)

		for _, file := range isHTTPSMetricsEnabledTest.files {
			f, err := os.Create(pathutils.CleanPath(tmpDir, file))
			if err != nil {
				t.Fatalf("Failed to create file %s: %v", file, err)
			}
			// #nosec G307 - local to this function, pose no harm.
			defer f.Close()
		}

		httpsMetricsEnabled, err := isHTTPSMetricsEnabled(log, tmpDir)

		if isHTTPSMetricsEnabledTest.enabled != httpsMetricsEnabled {
			t.Errorf("Test %d : Function isHTTPSMetricsEnabled() : expected enabled [%v], got [%v]\n", isHTTPSMetricsEnabledTest.testNum, isHTTPSMetricsEnabledTest.enabled, httpsMetricsEnabled)
		}

		if isHTTPSMetricsEnabledTest.expectedError && err == nil {
			t.Errorf("Test %d : Function isHTTPSMetricsEnabled() : expected error [%v], got error [nil]\n", isHTTPSMetricsEnabledTest.testNum, isHTTPSMetricsEnabledTest.expectedError)
		}

		if !isHTTPSMetricsEnabledTest.expectedError && err != nil {
			t.Errorf("Test %d : Function isHTTPSMetricsEnabled() : expected error [nil], got error [%v]\n", isHTTPSMetricsEnabledTest.testNum, err)
		}
	}
}
