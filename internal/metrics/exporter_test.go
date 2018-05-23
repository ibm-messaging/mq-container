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

	"github.com/prometheus/client_golang/prometheus"
)

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
