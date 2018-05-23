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
	"net/http"
	"time"

	"github.com/ibm-messaging/mq-container/internal/logger"
	"github.com/ibm-messaging/mq-golang/mqmetric"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	defaultPort = "9157"
	retryCount  = 3
	retryWait   = 5
)

// GatherMetrics gathers metrics for the queue manager
func GatherMetrics(qmName string, log *logger.Logger) {

	for i := 0; i <= retryCount; i++ {
		err := startMetricsGathering(qmName, log)
		if err != nil {
			log.Errorf("Metrics Error: %s", err.Error())
		}
		if i != retryCount {
			log.Printf("Waiting %d seconds before retrying metrics gathering", retryWait)
			time.Sleep(retryWait * time.Second)
		} else {
			log.Println("Unable to gather metrics - metrics are now disabled")
		}
	}
}

// startMetricsGathering starts gathering metrics for the queue manager
func startMetricsGathering(qmName string, log *logger.Logger) error {

	defer func() {
		if r := recover(); r != nil {
			log.Errorf("Metrics Error: %v", r)
		}
	}()

	log.Println("Starting metrics gathering")

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
	defer mqmetric.EndConnection()

	// Discover available metrics for the queue manager and subscribe to them
	err = mqmetric.DiscoverAndSubscribe("", true, "")
	if err != nil {
		return fmt.Errorf("Failed to discover and subscribe to metrics: %v", err)
	}

	// Start processing metrics
	go processMetrics(log)

	// Register metrics
	prometheus.MustRegister(newExporter(qmName))

	// Setup HTTP server to handle requests from Prometheus
	http.Handle("/metrics", prometheus.Handler())
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte("Status: METRICS ACTIVE"))
	})
	err = http.ListenAndServe(":"+defaultPort, nil)
	return fmt.Errorf("Failed to handle metrics request: %v", err)
}
