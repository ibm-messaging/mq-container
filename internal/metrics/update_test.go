/*
© Copyright IBM Corporation 2018, 2025

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

import (
	"os"
	"testing"

	"github.com/ibm-messaging/mq-container/pkg/logger"
	"github.com/ibm-messaging/mq-golang/v5/mqmetric"
)

const (
	testClassName           = "CPU"
	testTypeName            = "SystemSummary"
	testElement1Name        = "cpu_load_five_minute_average_percentage"
	testElement2Name        = "cpu_load_fifteen_minute_average_percentage"
	testElement1Description = "CPU load - five minute average"
	testElement2Description = "CPU load - fifteen minute average"
	testKey1                = testClassName + "/" + testTypeName + "/" + testElement1Description
	testKey2                = testClassName + "/" + testTypeName + "/" + testElement2Description
)

func TestInitialiseMetrics(t *testing.T) {

	teardownTestCase := setupTestCase(false)
	defer teardownTestCase()

	metrics, err := initialiseMetrics(getTestLogger())
	metric, ok := metrics[testKey1]

	if err != nil {
		t.Errorf("Unexpected error %s", err.Error())
	}
	if !ok {
		t.Error("Expected metric not found in map")
	} else {
		if metric.name != testElement1Name {
			t.Errorf("Expected name=%s; actual %s", testElement1Name, metric.name)
		}
		if metric.description != testElement1Description {
			t.Errorf("Expected description=%s; actual %s", testElement1Description, metric.description)
		}
		if metric.objectType != false {
			t.Errorf("Expected objectType=%v; actual %v", false, metric.objectType)
		}
		if len(metric.values) != 0 {
			t.Errorf("Expected values-size=%d; actual %d", 0, len(metric.values))
		}
	}
	_, ok = metrics[testKey2]
	if ok {
		t.Errorf("Unexpected metric found in map, %%s object topics should be ignored")
	}

	if len(metrics) != 1 {
		t.Errorf("Map contains unexpected metrics, map size=%d", len(metrics))
	}
}

func TestInitialiseMetrics_UnexpectedKey(t *testing.T) {

	teardownTestCase := setupTestCase(false)
	defer teardownTestCase()

	mqmetric.Metrics.Classes[0].Types[0].Elements[0].Description = "New Metric"
	_, err := initialiseMetrics(getTestLogger())

	if err == nil {
		t.Error("Expected skipping metric error")
	}
}

func TestInitialiseMetrics_DuplicateKeys(t *testing.T) {

	teardownTestCase := setupTestCase(true)
	defer teardownTestCase()

	_, err := initialiseMetrics(getTestLogger())

	if err == nil {
		t.Error("Expected duplicate keys error")
	}
}

func TestUpdateMetrics(t *testing.T) {

	teardownTestCase := setupTestCase(false)
	defer teardownTestCase()

	metrics, _ := initialiseMetrics(getTestLogger())
	updateMetrics(metrics)

	metric, _ := metrics[testKey1]
	actual, ok := metric.values[qmgrLabelValue]

	if !ok {
		t.Error("No metric values found for queue manager label")
	} else {
		if actual != float64(1) {
			t.Errorf("Expected metric value=%f; actual %f", float64(1), actual)
		}
		if len(metric.values) != 1 {
			t.Errorf("Expected values-size=%d; actual %d", 1, len(metric.values))
		}
	}

	if len(mqmetric.Metrics.Classes[0].Types[0].Elements[0].Values) != 0 {
		t.Error("Unexpected cached value; publication data should have been reset")
	}

	updateMetrics(metrics)

	if len(metric.values) != 0 {
		t.Errorf("Unexpected metric value; data should have been cleared")
	}
}

func TestMakeKey(t *testing.T) {

	teardownTestCase := setupTestCase(false)
	defer teardownTestCase()

	expected := testKey1
	actual := makeKey(mqmetric.Metrics.Classes[0].Types[0].Elements[0])
	if actual != expected {
		t.Errorf("Expected value=%s; actual %s", expected, actual)
	}
}

func setupTestCase(duplicateKey bool) func() {
	populateTestMetrics(1, duplicateKey)
	return func() {
		cleanTestMetrics()
	}
}

func populateTestMetrics(testValue int, duplicateKey bool) {

	metricClass := new(mqmetric.MonClass)
	metricType1 := new(mqmetric.MonType)
	metricType2 := new(mqmetric.MonType)
	metricElement1 := new(mqmetric.MonElement)
	metricElement2 := new(mqmetric.MonElement)

	metricClass.Name = testClassName
	metricType1.Name = testTypeName
	metricType2.Name = testTypeName
	metricElement1.MetricName = "Element1Name"
	metricElement1.Description = testElement1Description
	metricElement1.Values = make(map[string]int64)
	metricElement1.Values[qmgrLabelValue] = int64(testValue)
	metricElement2.MetricName = "Element2Name"
	metricElement2.Description = testElement2Description
	metricElement2.Values = make(map[string]int64)
	metricType1.ObjectTopic = "ObjectTopic"
	metricType2.ObjectTopic = "%s"
	metricElement1.Parent = metricType1
	metricElement2.Parent = metricType2
	metricType1.Parent = metricClass
	metricType2.Parent = metricClass

	metricType1.Elements = make(map[int]*mqmetric.MonElement)
	metricType2.Elements = make(map[int]*mqmetric.MonElement)
	metricType1.Elements[0] = metricElement1
	if duplicateKey {
		metricType1.Elements[1] = metricElement1
	}
	metricType2.Elements[0] = metricElement2
	metricClass.Types = make(map[int]*mqmetric.MonType)
	metricClass.Types[0] = metricType1
	metricClass.Types[1] = metricType2
	mqmetric.Metrics.Classes = make(map[int]*mqmetric.MonClass)
	mqmetric.Metrics.Classes[0] = metricClass
}

func cleanTestMetrics() {
	mqmetric.Metrics.Classes = make(map[int]*mqmetric.MonClass)
}

func getTestLogger() *logger.Logger {
	log, _ := logger.NewLogger(os.Stdout, false, false, "test")
	return log
}
