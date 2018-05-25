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
package main

import (
	"strings"
	"testing"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
)

func TestGoldenPathMetric(t *testing.T) {
	t.Parallel()
	cli, err := client.NewEnvClient()
	if err != nil {
		t.Fatal(err)
	}

	containerConfig := container.Config{
		Env: []string{
			"LICENSE=accept",
			"MQ_QMGR_NAME=qm1",
			"MQ_ENABLE_METRICS=true",
		},
	}
	id := runContainer(t, cli, &containerConfig)
	defer cleanContainer(t, cli, id)

	hostname := getIPAddress(t, cli, id)
	port := DEFAULT_METRIC_PORT

	// Now the container is ready we prod the prometheus endpoint until it's up.
	waitForMetricReady(hostname, port)

	// Call once as mq_prometheus 'ignores' the first call and will not return any metrics
	_, err = getMetricsFromEndpoint(hostname, port)
	if err != nil {
		t.Logf("Failed to call metric endpoint - %v", err)
		t.FailNow()
	}

	time.Sleep(15 * time.Second)
	metrics, err := getMetricsFromEndpoint(hostname, port)
	if err != nil {
		t.Logf("Failed to call metric endpoint - %v", err)
		t.FailNow()
	}

	if len(metrics) <= 0 {
		t.Log("Expected some metrics to be returned but had none...")
		t.Fail()
	}

	// Stop the container cleanly
	stopContainer(t, cli, id)
}

func TestMetricNames(t *testing.T) {
	t.Parallel()

	approvedSuffixes := []string{"bytes", "seconds", "percentage", "count", "total"}
	cli, err := client.NewEnvClient()
	if err != nil {
		t.Fatal(err)
	}

	containerConfig := container.Config{
		Env: []string{
			"LICENSE=accept",
			"MQ_QMGR_NAME=qm1",
			"MQ_ENABLE_METRICS=true",
		},
	}
	id := runContainer(t, cli, &containerConfig)
	defer cleanContainer(t, cli, id)

	hostname := getIPAddress(t, cli, id)
	port := DEFAULT_METRIC_PORT

	// Now the container is ready we prod the prometheus endpoint until it's up.
	waitForMetricReady(hostname, port)

	// Call once as mq_prometheus 'ignores' the first call
	_, err = getMetricsFromEndpoint(hostname, port)
	if err != nil {
		t.Logf("Failed to call metric endpoint - %v", err)
		t.FailNow()
	}

	time.Sleep(15 * time.Second)
	metrics, err := getMetricsFromEndpoint(hostname, port)
	if err != nil {
		t.Logf("Failed to call metric endpoint - %v", err)
		t.FailNow()
	}

	if len(metrics) <= 0 {
		t.Log("Expected some metrics to be returned but had none...")
		t.Fail()
	}

	okMetrics := []string{}
	badMetrics := []string{}

	for _, metric := range metrics {
		ok := false
		for _, e := range approvedSuffixes {
			if strings.HasSuffix(metric.Key, e) {
				ok = true
				break
			}
		}

		if !ok {
			t.Logf("Metric '%s' does not have an approved suffix", metric.Key)
			badMetrics = append(badMetrics, metric.Key)
			t.Fail()
		} else {
			okMetrics = append(okMetrics, metric.Key)
		}
	}

	// Stop the container cleanly
	stopContainer(t, cli, id)
}

func TestMetricLabels(t *testing.T) {
	t.Parallel()

	requiredLabels := []string{"qmgr"}
	cli, err := client.NewEnvClient()
	if err != nil {
		t.Fatal(err)
	}

	containerConfig := container.Config{
		Env: []string{
			"LICENSE=accept",
			"MQ_QMGR_NAME=qm1",
			"MQ_ENABLE_METRICS=true",
		},
	}
	id := runContainer(t, cli, &containerConfig)
	defer cleanContainer(t, cli, id)

	hostname := getIPAddress(t, cli, id)
	port := DEFAULT_METRIC_PORT

	// Now the container is ready we prod the prometheus endpoint until it's up.
	waitForMetricReady(hostname, port)

	// Call once as mq_prometheus 'ignores' the first call
	_, err = getMetricsFromEndpoint(hostname, port)
	if err != nil {
		t.Logf("Failed to call metric endpoint - %v", err)
		t.FailNow()
	}

	time.Sleep(15 * time.Second)
	metrics, err := getMetricsFromEndpoint(hostname, port)
	if err != nil {
		t.Logf("Failed to call metric endpoint - %v", err)
		t.FailNow()
	}

	if len(metrics) <= 0 {
		t.Log("Expected some metrics to be returned but had none...")
		t.Fail()
	}

	for _, metric := range metrics {
		found := false
		for key := range metric.Labels {
			for _, e := range requiredLabels {
				if key == e {
					found = true
					break
				}
			}
			if found {
				break
			}
		}

		if !found {
			t.Logf("Metric '%s' with labels %s does not have one or more required labels - %s", metric.Key, metric.Labels, requiredLabels)
			t.Fail()
		}
	}

	// Stop the container cleanly
	stopContainer(t, cli, id)
}

func TestRapidFirePrometheus(t *testing.T) {
	t.Parallel()

	cli, err := client.NewEnvClient()
	if err != nil {
		t.Fatal(err)
	}

	containerConfig := container.Config{
		Env: []string{
			"LICENSE=accept",
			"MQ_QMGR_NAME=qm1",
			"MQ_ENABLE_METRICS=true",
		},
	}
	id := runContainer(t, cli, &containerConfig)
	defer cleanContainer(t, cli, id)

	hostname := getIPAddress(t, cli, id)
	port := DEFAULT_METRIC_PORT

	// Now the container is ready we prod the prometheus endpoint until it's up.
	waitForMetricReady(hostname, port)

	// Call once as mq_prometheus 'ignores' the first call and will not return any metrics
	_, err = getMetricsFromEndpoint(hostname, port)
	if err != nil {
		t.Logf("Failed to call metric endpoint - %v", err)
		t.FailNow()
	}

	// Rapid fire it then check we're still happy
	for i := 0; i < 30; i++ {
		_, err := getMetricsFromEndpoint(hostname, port)
		if err != nil {
			t.Logf("Failed to call metric endpoint - %v", err)
			t.FailNow()
		}
		time.Sleep(1 * time.Second)
	}

	time.Sleep(11 * time.Second)

	metrics, err := getMetricsFromEndpoint(hostname, port)
	if err != nil {
		t.Logf("Failed to call metric endpoint - %v", err)
		t.FailNow()
	}
	if len(metrics) <= 0 {
		t.Log("Expected some metrics to be returned but had none...")
		t.Fail()
	}

	// Stop the container cleanly
	stopContainer(t, cli, id)
}

func TestSlowPrometheus(t *testing.T) {
	t.Parallel()

	cli, err := client.NewEnvClient()
	if err != nil {
		t.Fatal(err)
	}

	containerConfig := container.Config{
		Env: []string{
			"LICENSE=accept",
			"MQ_QMGR_NAME=qm1",
			"MQ_ENABLE_METRICS=true",
		},
	}
	id := runContainer(t, cli, &containerConfig)
	defer cleanContainer(t, cli, id)

	hostname := getIPAddress(t, cli, id)
	port := DEFAULT_METRIC_PORT

	// Now the container is ready we prod the prometheus endpoint until it's up.
	waitForMetricReady(hostname, port)

	// Call once as mq_prometheus 'ignores' the first call and will not return any metrics
	_, err = getMetricsFromEndpoint(hostname, port)
	if err != nil {
		t.Logf("Failed to call metric endpoint - %v", err)
		t.FailNow()
	}

	// Send a request twice over a long period and check we're still happy
	for i := 0; i < 2; i++ {
		time.Sleep(30 * time.Second)
		metrics, err := getMetricsFromEndpoint(hostname, port)
		if err != nil {
			t.Logf("Failed to call metric endpoint - %v", err)
			t.FailNow()
		}
		if len(metrics) <= 0 {
			t.Log("Expected some metrics to be returned but had none...")
			t.Fail()
		}

	}

	// Stop the container cleanly
	stopContainer(t, cli, id)
}
