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
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/ibm-messaging/mq-container/internal/ready"
	"github.com/ibm-messaging/mq-container/pkg/logger"
	"github.com/ibm-messaging/mq-container/pkg/logrotation"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	defaultPort = "9157"

	// keyDirMetrics is the location of the TLS keys to use for HTTPS metrics
	keyDirMetrics = "/etc/mqm/metrics/pki/keys"

	auditLogDirectory      = "/var/mqm/errors"
	auditLogFilenameFormat = "metricaudit%02d.json"
	auditLogMaxBytes       = 4 * 1024 * 1024
	auditLogNumFiles       = 3
)

var (
	metricsEnabled = false
	// #nosec G112 - this code is changing soon to use https.
	// for now we will ignore the gosec.
	metricsServer = &http.Server{Addr: ":" + defaultPort}
)

// GatherMetrics gathers metrics for the queue manager
func GatherMetrics(qmName string, log *logger.Logger) {

	// If running in standby mode - wait until the queue manager becomes active
	for {
		status, _ := ready.Status(context.Background(), qmName)
		if status.ActiveQM() {
			break
		}
		time.Sleep(requestTimeout * time.Second)
	}

	metricsEnabled = true

	err := startMetricsGathering(qmName, log)
	if err != nil {
		log.Errorf("Metrics Error: %s", err.Error())
		StopMetricsGathering(log)
	}
}

// startMetricsGathering starts gathering metrics for the queue manager
func startMetricsGathering(qmName string, log *logger.Logger) error {

	defer func() {
		if r := recover(); r != nil {
			log.Errorf("Metrics Error: %v", r)
		}
	}()

	// Check if TLS keys have been provided for enabling HTTPS metrics
	httpsMetricsEnabled, err := isHTTPSMetricsEnabled(log, keyDirMetrics)
	if err != nil {
		return fmt.Errorf("Failed to validate HTTPS metrics configuration: %v", err)
	}

	// Generate appropriate audit log wrapper based on configuration
	auditWrapper := passthroughHandlerFuncWrapper
	if os.Getenv("MQ_LOGGING_METRICS_AUDIT_ENABLED") == "true" {
		auditLog := logrotation.NewRotatingLogger(auditLogDirectory, auditLogFilenameFormat, auditLogMaxBytes, auditLogNumFiles)
		err := auditLog.Init()
		if err != nil {
			return fmt.Errorf("Failed to set up metric audit log: %v", err)
		}
		auditWrapper = newAuditingHandlerFuncWrapper(qmName, auditLog)
	}

	if httpsMetricsEnabled {
		log.Println("Starting HTTPS metrics gathering")
	} else {
		log.Println("Starting HTTP (insecure) metrics gathering")
	}

	// Start processing metrics
	go processMetrics(log, qmName)

	// Wait for metrics to be ready before starting the Prometheus handler
	<-startChannel

	// Register metrics
	metricsExporter := newExporter(qmName, log)
	err = prometheus.Register(metricsExporter)
	if err != nil {
		return fmt.Errorf("Failed to register metrics: %v", err)
	}

	var tlsWatcher *certificateMonitor
	if httpsMetricsEnabled {
		tlsWatcher, err = loadAndWatchCertificates(context.Background(), keyDirMetrics, log)
		if err != nil {
			return fmt.Errorf("failed to set up TLS certificate monitor: %w", err)
		}
		tlsConfig := tls.Config{
			MinVersion: tls.VersionTLS12,
			GetCertificate: func(chi *tls.ClientHelloInfo) (*tls.Certificate, error) {
				cert := tlsWatcher.latestCert()
				if cert == nil {
					return nil, fmt.Errorf("no certificate loaded")
				}
				return cert, nil
			},
		}
		metricsServer.TLSConfig = &tlsConfig
	}

	// Setup HTTP server to handle requests from Prometheus
	http.Handle("/metrics", wrapHandler(promhttp.Handler(), auditWrapper))
	http.HandleFunc("/", auditWrapper(
		func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(200)
			// #nosec G104
			w.Write([]byte("Status: METRICS ACTIVE"))
		},
	))

	go func() {
		var err error
		if httpsMetricsEnabled {
			defer tlsWatcher.stop()
			// No certificates provided here as these are dynamically controlled by the GetCertificate call
			err = metricsServer.ListenAndServeTLS("", "")
		} else {
			err = metricsServer.ListenAndServe()
		}
		if err != nil && err != http.ErrServerClosed {
			log.Errorf("Metrics Error: Failed to handle metrics request: %v", err)
			StopMetricsGathering(log)
		}
	}()

	return nil
}

// StopMetricsGathering stops gathering metrics for the queue manager
func StopMetricsGathering(log *logger.Logger) {

	if metricsEnabled {

		// Stop processing metrics
		stopChannel <- true

		// Shutdown HTTP server
		timeout, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		err := metricsServer.Shutdown(timeout)
		if err != nil {
			log.Errorf("Failed to shutdown metrics server: %v", err)
		}
	}
}

// isHTTPSMetricsEnabled checks if TLS keys have been provided for enabling HTTPS metrics
func isHTTPSMetricsEnabled(log *logger.Logger, keyDirectory string) (bool, error) {

	// Read files in the location required for metrics TLS keys
	files, err := os.ReadDir(keyDirectory)
	if err != nil && strings.Contains(err.Error(), "no such file or directory") {
		return false, nil
	} else if err != nil {
		return false, fmt.Errorf("Unable to read files in '%s': %v", keyDirectory, err)
	}

	certFile := false
	keyFile := false

	// Validate if we have the required metrics TLS keys (tls.crt & tls.key)
	if len(files) > 0 {
		for _, file := range files {
			if file.Name() == "tls.crt" {
				certFile = true
			} else if file.Name() == "tls.key" {
				keyFile = true
			}
		}
		if certFile && keyFile {
			return true, nil
		}

		if !certFile {
			log.Errorf("Metrics Error: Unable to find required file 'tls.crt' in '%s'", keyDirectory)
		}
		if !keyFile {
			log.Errorf("Metrics Error: Unable to find required file 'tls.key' in '%s'", keyDirectory)
		}
		return false, fmt.Errorf("Missing files required for HTTPS")

	}

	return false, nil
}
