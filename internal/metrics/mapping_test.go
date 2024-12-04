/*
© Copyright IBM Corporation 2018, 2024

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
package metrics

import "testing"

func TestGenerateMetricNamesMap(t *testing.T) {

	metricNamesMap := generateMetricNamesMap()

	if len(metricNamesMap) != 96 {
		t.Errorf("Expected mapping-size=%d; actual %d", 93, len(metricNamesMap))
	}

	actual, ok := metricNamesMap[testKey1]

	if !ok {
		t.Errorf("No metric name mapping found for %s", testKey1)
	} else {
		if actual.name != testElement1Name {
			t.Errorf("Expected metric name=%s; actual %s", testElement1Name, actual.name)
		}
	}
}
