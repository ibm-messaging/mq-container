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
	"sync"
	"time"

	"github.com/ibm-messaging/mq-container/internal/logger"
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
	var wg sync.WaitGroup

	defer func() {
		if r := recover(); r != nil {
			log.Errorf("Metrics Error: %v", r)
		}
	}()

	// Start processing metrics
	wg.Add(1)
	go processMetrics(log, qmName, &wg)

	// Wait for metrics to be ready before starting the prometheus handler
	wg.Wait()

	// Register metrics
	prometheus.MustRegister(newExporter(qmName))

	// Setup HTTP server to handle requests from Prometheus
	http.Handle("/metrics", prometheus.Handler())
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte("Status: METRICS ACTIVE"))
	})

	err := http.ListenAndServe(":"+defaultPort, nil)
	if err != nil {
		return fmt.Errorf("Failed to handle metrics request: %v", err)
	}

	return nil
}
