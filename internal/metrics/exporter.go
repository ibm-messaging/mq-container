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

// Package metrics contains code to provide metrics for the queue manager
package metrics

import (
	"github.com/ibm-messaging/mq-container/internal/logger"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	namespace    = "ibmmq"
	qmgrPrefix   = "qmgr"
	qmgrLabel    = "qmgr"
	objectPrefix = "object"
	objectLabel  = "object"
)

type exporter struct {
	qmName       string
	gaugeMap     map[string]*prometheus.GaugeVec
	firstCollect bool
	log          *logger.Logger
}

func newExporter(qmName string, log *logger.Logger) *exporter {
	return &exporter{
		qmName:       qmName,
		gaugeMap:     make(map[string]*prometheus.GaugeVec),
		firstCollect: true,
		log:          log,
	}
}

// Describe provides details of all available metrics
func (e *exporter) Describe(ch chan<- *prometheus.Desc) {

	requestChannel <- false
	response := <-responseChannel

	for key, metric := range response {

		// Allocate a Prometheus Gauge for each available metric
		gaugeVec := createGaugeVec(metric.name, metric.description, metric.objectType)
		e.gaugeMap[key] = gaugeVec

		// Describe metric
		gaugeVec.Describe(ch)
	}
}

// Collect is called at regular intervals to provide the current metric data
func (e *exporter) Collect(ch chan<- prometheus.Metric) {

	requestChannel <- true
	response := <-responseChannel

	for key, metric := range response {

		// Reset Prometheus Gauge
		gaugeVec := e.gaugeMap[key]
		gaugeVec.Reset()

		// Populate Prometheus Gauge with metric values
		// - Skip on first collect to avoid build-up of accumulated values
		if !e.firstCollect {
			for label, value := range metric.values {
				var err error
				var gauge prometheus.Gauge

				if label == qmgrLabelValue {
					gauge, err = gaugeVec.GetMetricWithLabelValues(e.qmName)
				} else {
					gauge, err = gaugeVec.GetMetricWithLabelValues(label, e.qmName)
				}
				if err == nil {
					gauge.Set(value)
				} else {
					e.log.Errorf("Metrics Error: %s", err.Error())
				}
			}
		}

		// Collect metric
		gaugeVec.Collect(ch)
	}

	if e.firstCollect {
		e.firstCollect = false
	}
}

// createGaugeVec returns a Prometheus GaugeVec populated with metric details
func createGaugeVec(name, description string, objectType bool) *prometheus.GaugeVec {

	prefix := qmgrPrefix
	labels := []string{qmgrLabel}

	if objectType {
		prefix = objectPrefix
		labels = []string{objectLabel, qmgrLabel}
	}

	gaugeVec := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      prefix + "_" + name,
			Help:      description,
		},
		labels,
	)
	return gaugeVec
}
