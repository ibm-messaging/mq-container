/*
© Copyright IBM Corporation 2018

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
	"testing"

	"github.com/ibm-messaging/mq-golang/mqmetric"
)

func TestInitialiseMetrics(t *testing.T) {

	teardownTestCase := setupTestCase()
	defer teardownTestCase()

	metrics := initialiseMetrics()
	metric, ok := metrics["ClassName/Type1Name/Element1Name"]

	if !ok {
		t.Error("Expected metric not found in map")
	} else {
		if metric.name != "Element1Name" {
			t.Errorf("Expected name=%s; actual %s", "Element1Name", metric.name)
		}
		if metric.description != "Element1Description" {
			t.Errorf("Expected description=%s; actual %s", "Element1Description", metric.description)
		}
		if metric.objectType != false {
			t.Errorf("Expected objectType=%v; actual %v", false, metric.objectType)
		}
		if len(metric.values) != 0 {
			t.Errorf("Expected values-size=%d; actual %d", 0, len(metric.values))
		}
	}
	_, ok = metrics["ClassName/Type2Name/Element2Name"]
	if ok {
		t.Errorf("Unexpected metric found in map, %%s object topics should be ignored")
	}

	if len(metrics) != 1 {
		t.Errorf("Map contains unexpected metrics, map size=%d", len(metrics))
	}
}

func TestUpdateMetrics(t *testing.T) {

	teardownTestCase := setupTestCase()
	defer teardownTestCase()

	metrics := initialiseMetrics()
	updateMetrics(metrics)

	metric, _ := metrics["ClassName/Type1Name/Element1Name"]
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

	teardownTestCase := setupTestCase()
	defer teardownTestCase()

	expected := "ClassName/Type1Name/Element1Name"
	actual := makeKey(mqmetric.Metrics.Classes[0].Types[0].Elements[0])
	if actual != expected {
		t.Errorf("Expected value=%s; actual %s", expected, actual)
	}
}

func setupTestCase() func() {
	populateTestMetrics(1)
	return func() {
		cleanTestMetrics()
	}
}

func populateTestMetrics(testValue int) {

	metricClass := new(mqmetric.MonClass)
	metricType1 := new(mqmetric.MonType)
	metricType2 := new(mqmetric.MonType)
	metricElement1 := new(mqmetric.MonElement)
	metricElement2 := new(mqmetric.MonElement)

	metricClass.Name = "ClassName"
	metricType1.Name = "Type1Name"
	metricType2.Name = "Type2Name"
	metricElement1.MetricName = "Element1Name"
	metricElement1.Description = "Element1Description"
	metricElement1.Values = make(map[string]int64)
	metricElement1.Values[qmgrLabelValue] = int64(testValue)
	metricElement2.MetricName = "Element2Name"
	metricElement2.Description = "Element2Description"
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
