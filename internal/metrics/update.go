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
	"fmt"
	"strings"
	"time"

	"github.com/ibm-messaging/mq-container/internal/logger"
	"github.com/ibm-messaging/mq-golang/mqmetric"
)

const (
	qmgrLabelValue = mqmetric.QMgrMapKey
	requestTimeout = 10
)

var (
	startChannel    = make(chan bool)
	stopChannel     = make(chan bool, 2)
	requestChannel  = make(chan bool)
	responseChannel = make(chan map[string]*metricData)
)

type metricData struct {
	name        string
	description string
	objectType  bool
	values      map[string]float64
}

// processMetrics processes publications of metric data and handles describe/collect/stop requests
func processMetrics(log *logger.Logger, qmName string) {

	var err error
	var firstConnect = true
	var metrics map[string]*metricData

	for {
		// Connect to queue manager and discover available metrics
		err = doConnect(qmName)
		if err == nil {
			if firstConnect {
				firstConnect = false
				startChannel <- true
			}
			metrics, _ = initialiseMetrics(log)
		}

		// Now loop until something goes wrong
		for err == nil {

			// Process publications of metric data
			// TODO: If we have a large number of metrics to process, then we could be blocked from responding to stop requests
			err = mqmetric.ProcessPublications()

			// Handle describe/collect/stop requests
			if err == nil {
				select {
				case collect := <-requestChannel:
					if collect {
						updateMetrics(metrics)
					}
					responseChannel <- metrics
				case <-stopChannel:
					log.Println("Stopping metrics gathering")
					mqmetric.EndConnection()
					return
				case <-time.After(requestTimeout * time.Second):
					log.Debugf("Metrics: No requests received within timeout period (%d seconds)", requestTimeout)
				}
			}
		}
		log.Errorf("Metrics Error: %s", err.Error())

		// Close the connection
		mqmetric.EndConnection()

		// Handle stop requests
		select {
		case <-stopChannel:
			log.Println("Stopping metrics gathering")
			return
		case <-time.After(requestTimeout * time.Second):
			log.Println("Retrying metrics gathering")
		}
	}
}

// doConnect connects to the queue manager and discovers available metrics
func doConnect(qmName string) error {

	// Set connection configuration
	var connConfig mqmetric.ConnectionConfig
	connConfig.ClientMode = false
	connConfig.UserId = ""
	connConfig.Password = ""

	// Connect to the queue manager - open the command and dynamic reply queues
	err := mqmetric.InitConnectionStats(qmName, "SYSTEM.DEFAULT.MODEL.QUEUE", "", &connConfig)
	if err != nil {
		return fmt.Errorf("Failed to connect to queue manager %s: %v", qmName, err)
	}

	// Discover available metrics for the queue manager and subscribe to them
	err = mqmetric.DiscoverAndSubscribe("", true, "")
	if err != nil {
		return fmt.Errorf("Failed to discover and subscribe to metrics: %v", err)
	}

	return nil
}

// initialiseMetrics sets initial details for all available metrics
func initialiseMetrics(log *logger.Logger) (map[string]*metricData, error) {

	metrics := make(map[string]*metricData)
	validMetrics := true
	metricNamesMap := generateMetricNamesMap()

	for _, metricClass := range mqmetric.Metrics.Classes {
		for _, metricType := range metricClass.Types {
			if !strings.Contains(metricType.ObjectTopic, "%s") {
				for _, metricElement := range metricType.Elements {

					// Get unique metric key
					key := makeKey(metricElement)

					// Get metric name from mapping
					if metricLookup, found := metricNamesMap[key]; found {

						// Check if metric is enabled
						if metricLookup.enabled {

							// Set metric details
							metric := metricData{
								name:        metricLookup.name,
								description: metricElement.Description,
							}

							// Add metric
							if _, exists := metrics[key]; !exists {
								metrics[key] = &metric
							} else {
								log.Errorf("Metrics Error: Found duplicate metric key [%s]", key)
								validMetrics = false
							}
						} else {
							log.Debugf("Metrics: Skipping metric, metric is not enabled for key [%s]", key)
						}
					} else {
						log.Errorf("Metrics Error: Skipping metric, unexpected key [%s]", key)
						validMetrics = false
					}
				}
			}
		}
	}

	if !validMetrics {
		return metrics, fmt.Errorf("Invalid metrics data")
	}
	return metrics, nil
}

// updateMetrics updates values for all available metrics
func updateMetrics(metrics map[string]*metricData) {

	for _, metricClass := range mqmetric.Metrics.Classes {
		for _, metricType := range metricClass.Types {
			if !strings.Contains(metricType.ObjectTopic, "%s") {
				for _, metricElement := range metricType.Elements {

					// Unexpected metric elements (with no defined mapping) are handled in 'initialiseMetrics'
					// - if any exist, they are logged as errors and skipped (they are not added to the metrics map)
					// Therefore we can ignore handling any unexpected metric elements found here
					// - this avoids us logging excessive errors, as this function is called frequently
					metric, ok := metrics[makeKey(metricElement)]
					if ok {
						// Clear existing metric values
						metric.values = make(map[string]float64)

						// Update metric with cached values of publication data
						for label, value := range metricElement.Values {
							normalisedValue := mqmetric.Normalise(metricElement, label, value)
							metric.values[label] = normalisedValue
						}
					}

					// Reset cached values of publication data for this metric
					metricElement.Values = make(map[string]int64)
				}
			}
		}
	}
}

// makeKey builds a unique key for each metric
func makeKey(metricElement *mqmetric.MonElement) string {
	return metricElement.Parent.Parent.Name + "/" + metricElement.Parent.Name + "/" + metricElement.Description
}
