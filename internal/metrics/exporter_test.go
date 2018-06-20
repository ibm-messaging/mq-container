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
	"time"

	"github.com/ibm-messaging/mq-golang/ibmmq"
	"github.com/ibm-messaging/mq-golang/mqmetric"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

func TestDescribe_Counter(t *testing.T) {
	testDescribe(t, true)
}

func TestDescribe_Gauge(t *testing.T) {
	testDescribe(t, false)
}

func testDescribe(t *testing.T, isDelta bool) {

	teardownTestCase := setupTestCase(false)
	defer teardownTestCase()
	log := getTestLogger()

	ch := make(chan *prometheus.Desc)
	go func() {
		exporter := newExporter("qmName", log)
		exporter.Describe(ch)
	}()

	collect := <-requestChannel
	if collect {
		t.Errorf("Received unexpected collect request")
	}

	if isDelta {
		mqmetric.Metrics.Classes[0].Types[0].Elements[0].Datatype = ibmmq.MQIAMO_MONITOR_DELTA
	}
	metrics, _ := initialiseMetrics(log)
	responseChannel <- metrics

	select {
	case prometheusDesc := <-ch:
		expected := "Desc{fqName: \"ibmmq_qmgr_" + testElement1Name + "\", help: \"" + testElement1Description + "\", constLabels: {}, variableLabels: [qmgr]}"
		actual := prometheusDesc.String()
		if actual != expected {
			t.Errorf("Expected value=%s; actual %s", expected, actual)
		}
	case <-time.After(1 * time.Second):
		t.Error("Did not receive channel response from describe")
	}
}

func TestCollect_Counter(t *testing.T) {
	testCollect(t, true)
}

func TestCollect_Gauge(t *testing.T) {
	testCollect(t, false)
}

func testCollect(t *testing.T, isDelta bool) {

	teardownTestCase := setupTestCase(false)
	defer teardownTestCase()
	log := getTestLogger()

	exporter := newExporter("qmName", log)
	if isDelta {
		exporter.counterMap[testKey1] = createCounterVec(testElement1Name, testElement1Description, false)
	} else {
		exporter.gaugeMap[testKey1] = createGaugeVec(testElement1Name, testElement1Description, false)
	}

	for i := 1; i <= 3; i++ {

		ch := make(chan prometheus.Metric)
		go func() {
			exporter.Collect(ch)
			close(ch)
		}()

		collect := <-requestChannel
		if !collect {
			t.Errorf("Received unexpected describe request")
		}

		populateTestMetrics(i, false)
		if isDelta {
			mqmetric.Metrics.Classes[0].Types[0].Elements[0].Datatype = ibmmq.MQIAMO_MONITOR_DELTA
		}
		metrics, _ := initialiseMetrics(log)
		updateMetrics(metrics)
		responseChannel <- metrics

		select {
		case <-ch:
			var actual float64
			prometheusMetric := dto.Metric{}
			if isDelta {
				exporter.counterMap[testKey1].WithLabelValues("qmName").Write(&prometheusMetric)
				actual = prometheusMetric.GetCounter().GetValue()
			} else {
				exporter.gaugeMap[testKey1].WithLabelValues("qmName").Write(&prometheusMetric)
				actual = prometheusMetric.GetGauge().GetValue()
			}

			if i == 1 {
				if actual != float64(0) {
					t.Errorf("Expected values to be zero on first collect; actual %f", actual)
				}
			} else if isDelta && i != 2 {
				if actual != float64(i+(i-1)) {
					t.Errorf("Expected value=%f; actual %f", float64(i+(i-1)), actual)
				}
			} else if actual != float64(i) {
				t.Errorf("Expected value=%f; actual %f", float64(i), actual)
			}
		case <-time.After(1 * time.Second):
			t.Error("Did not receive channel response from collect")
		}
	}
}

func TestCreateCounterVec(t *testing.T) {

	ch := make(chan *prometheus.Desc)
	counterVec := createCounterVec("MetricName", "MetricDescription", false)
	go func() {
		counterVec.Describe(ch)
	}()
	description := <-ch

	expected := "Desc{fqName: \"ibmmq_qmgr_MetricName\", help: \"MetricDescription\", constLabels: {}, variableLabels: [qmgr]}"
	actual := description.String()
	if actual != expected {
		t.Errorf("Expected value=%s; actual %s", expected, actual)
	}
}

func TestCreateCounterVec_ObjectLabel(t *testing.T) {

	ch := make(chan *prometheus.Desc)
	counterVec := createCounterVec("MetricName", "MetricDescription", true)
	go func() {
		counterVec.Describe(ch)
	}()
	description := <-ch

	expected := "Desc{fqName: \"ibmmq_object_MetricName\", help: \"MetricDescription\", constLabels: {}, variableLabels: [object qmgr]}"
	actual := description.String()
	if actual != expected {
		t.Errorf("Expected value=%s; actual %s", expected, actual)
	}
}

func TestCreateGaugeVec(t *testing.T) {

	ch := make(chan *prometheus.Desc)
	gaugeVec := createGaugeVec("MetricName", "MetricDescription", false)
	go func() {
		gaugeVec.Describe(ch)
	}()
	description := <-ch

	expected := "Desc{fqName: \"ibmmq_qmgr_MetricName\", help: \"MetricDescription\", constLabels: {}, variableLabels: [qmgr]}"
	actual := description.String()
	if actual != expected {
		t.Errorf("Expected value=%s; actual %s", expected, actual)
	}
}

func TestCreateGaugeVec_ObjectLabel(t *testing.T) {

	ch := make(chan *prometheus.Desc)
	gaugeVec := createGaugeVec("MetricName", "MetricDescription", true)
	go func() {
		gaugeVec.Describe(ch)
	}()
	description := <-ch

	expected := "Desc{fqName: \"ibmmq_object_MetricName\", help: \"MetricDescription\", constLabels: {}, variableLabels: [object qmgr]}"
	actual := description.String()
	if actual != expected {
		t.Errorf("Expected value=%s; actual %s", expected, actual)
	}
}
