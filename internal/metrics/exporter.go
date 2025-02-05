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

// Package metrics contains code to provide metrics for the queue manager
package metrics

import (
	"strings"

	"github.com/ibm-messaging/mq-container/pkg/logger"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	namespace         = "ibmmq"
	qmgrPrefix        = "qmgr"
	qmgrLabel         = "qmgr"
	objectPrefix      = "object"
	objectLabel       = "object"
	nhaInstancePrefix = "nha"
	nhaInstanceLabel  = "instance"
)

type exporter struct {
	qmName       string
	gaugeMap     map[string]*prometheus.GaugeVec
	counterMap   map[string]*prometheus.CounterVec
	firstCollect bool
	log          *logger.Logger
}

func newExporter(qmName string, log *logger.Logger) *exporter {
	return &exporter{
		qmName:       qmName,
		gaugeMap:     make(map[string]*prometheus.GaugeVec),
		counterMap:   make(map[string]*prometheus.CounterVec),
		firstCollect: true,
		log:          log,
	}
}

// Describe provides details of all available metrics
func (e *exporter) Describe(ch chan<- *prometheus.Desc) {

	requestChannel <- false
	response := <-responseChannel

	for key, metric := range response {

		if metric.isDelta {
			// For delta type metrics - allocate a Prometheus Counter
			counterVec := createCounterVec(metric.name, metric.description, metric.objectType, metric.nhaType)
			e.counterMap[key] = counterVec

			// Describe metric
			counterVec.Describe(ch)

		} else {
			// For non-delta type metrics - allocate a Prometheus Gauge
			gaugeVec := createGaugeVec(metric.name, metric.description, metric.objectType, metric.nhaType)
			e.gaugeMap[key] = gaugeVec

			// Describe metric
			gaugeVec.Describe(ch)
		}
	}
}

// Collect is called at regular intervals to provide the current metric data
func (e *exporter) Collect(ch chan<- prometheus.Metric) {

	requestChannel <- true
	response := <-responseChannel

	for key, metric := range response {

		if metric.isDelta {
			// For delta type metrics - update their Prometheus Counter
			counterVec := e.counterMap[key]

			// Populate Prometheus Counter with metric values
			// - Skip on first collect to avoid build-up of accumulated values
			if !e.firstCollect {
				for label, value := range metric.values {
					var err error
					var counter prometheus.Counter

					if label == qmgrLabelValue {
						counter, err = counterVec.GetMetricWithLabelValues(e.qmName)
					} else if strings.HasPrefix(label, nhaLabelValue) {
						nhaInstance := strings.ReplaceAll(label, nhaLabelValue, "")
						counter, err = counterVec.GetMetricWithLabelValues(nhaInstance, e.qmName)
					} else {
						counter, err = counterVec.GetMetricWithLabelValues(label, e.qmName)
					}
					if err == nil {
						counter.Add(value)
					} else {
						e.log.Errorf("Metrics Error: %s", err.Error())
					}
				}
			}

			// Collect metric
			counterVec.Collect(ch)

		} else {
			// For non-delta type metrics - reset their Prometheus Gauge
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
					} else if strings.HasPrefix(label, nhaLabelValue) {
						nhaInstance := strings.ReplaceAll(label, nhaLabelValue, "")
						gauge, err = gaugeVec.GetMetricWithLabelValues(nhaInstance, e.qmName)
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
	}

	if e.firstCollect {
		e.firstCollect = false
	}
}

// createCounterVec returns a Prometheus CounterVec populated with metric details
func createCounterVec(name, description string, objectType bool, nhaType bool) *prometheus.CounterVec {

	prefix, labels := getVecDetails(objectType, nhaType)

	counterVec := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      prefix + "_" + name,
			Help:      description,
		},
		labels,
	)
	return counterVec
}

// createGaugeVec returns a Prometheus GaugeVec populated with metric details
func createGaugeVec(name, description string, objectType bool, nhaType bool) *prometheus.GaugeVec {

	prefix, labels := getVecDetails(objectType, nhaType)

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

// getVecDetails returns the required prefix and labels for a metric
func getVecDetails(objectType bool, nhaType bool) (prefix string, labels []string) {
	switch true {
	case objectType:
		return objectPrefix, []string{objectLabel, qmgrLabel}
	case nhaType:
		return nhaInstancePrefix, []string{nhaInstanceLabel, qmgrLabel}
	default:
		return qmgrPrefix, []string{qmgrLabel}
	}

}
