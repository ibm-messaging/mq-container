/*
© Copyright IBM Corporation 2018, 2023

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
	"fmt"
	"net"
	"strconv"
	"testing"
	"time"

	ce "github.com/ibm-messaging/mq-container/test/container/containerengine"
)

func TestGoldenPathMetric(t *testing.T) {
	t.Parallel()

	cli := ce.NewContainerClient()
	id := runContainerWithPorts(t, cli, metricsContainerConfig(), []int{defaultMetricPort})
	cleanupAfterTest(t, cli, id, false)

	port, err := cli.GetContainerPort(id, defaultMetricPort)
	if err != nil {
		t.Fatal(err)
	}
	// Now the container is ready we prod the prometheus endpoint until it's up.
	waitForMetricReady(t, port)

	// Call once as mq_prometheus 'ignores' the first call and will not return any metrics
	getMetrics(t, port)
	time.Sleep(15 * time.Second)

	// Now actually get the metrics (after waiting for some to become available)
	metrics := getMetrics(t, port)
	if len(metrics) <= 0 {
		t.Error("Expected some metrics to be returned but had none...")
	}
	// Stop the container cleanly
	stopContainer(t, cli, id)
}

func TestMetricNames(t *testing.T) {
	t.Parallel()

	cli := ce.NewContainerClient()
	id := runContainerWithPorts(t, cli, metricsContainerConfig(), []int{defaultMetricPort})
	cleanupAfterTest(t, cli, id, false)

	port, err := cli.GetContainerPort(id, defaultMetricPort)
	if err != nil {
		t.Fatal(err)
	}
	// Now the container is ready we prod the prometheus endpoint until it's up.
	waitForMetricReady(t, port)

	// Call once as mq_prometheus 'ignores' the first call
	getMetrics(t, port)
	time.Sleep(35 * time.Second)

	// Now actually get the metrics (after waiting for some to become available)
	metrics := getMetrics(t, port)
	names := metricNames()
	if len(metrics) != len(names) {
		t.Errorf("Expected %d metrics to be returned, received %d", len(names), len(metrics))
	}

	// Check all the metrics have the correct names
	for _, metric := range metrics {
		ok := false
		for _, name := range names {
			if metric.Key == "ibmmq_qmgr_"+name {
				ok = true
				break
			}
		}

		if !ok {
			t.Errorf("Metric '%s' does not have the expected name", metric.Key)
		}
	}

	// Stop the container cleanly
	stopContainer(t, cli, id)
}

func TestMetricLabels(t *testing.T) {
	t.Parallel()

	cli := ce.NewContainerClient()
	requiredLabels := []string{"qmgr"}
	id := runContainerWithPorts(t, cli, metricsContainerConfig(), []int{defaultMetricPort})
	cleanupAfterTest(t, cli, id, false)
	port, err := cli.GetContainerPort(id, defaultMetricPort)
	if err != nil {
		t.Fatal(err)
	}

	// Now the container is ready we prod the prometheus endpoint until it's up.
	waitForMetricReady(t, port)

	// Call once as mq_prometheus 'ignores' the first call
	getMetrics(t, port)
	time.Sleep(15 * time.Second)

	// Now actually get the metrics (after waiting for some to become available)
	metrics := getMetrics(t, port)
	if len(metrics) <= 0 {
		t.Error("Expected some metrics to be returned but had none")
	}

	// Check all the metrics have the required labels
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
			t.Errorf("Metric '%s' with labels %s does not have one or more required labels - %s", metric.Key, metric.Labels, requiredLabels)
		}
	}

	// Stop the container cleanly
	stopContainer(t, cli, id)
}

func TestRapidFirePrometheus(t *testing.T) {
	t.Parallel()

	cli := ce.NewContainerClient()
	id := runContainerWithPorts(t, cli, metricsContainerConfig(), []int{defaultMetricPort})
	cleanupAfterTest(t, cli, id, false)
	port, err := cli.GetContainerPort(id, defaultMetricPort)
	if err != nil {
		t.Fatal(err)
	}

	// Now the container is ready we prod the prometheus endpoint until it's up.
	waitForMetricReady(t, port)

	// Call once as mq_prometheus 'ignores' the first call and will not return any metrics
	getMetrics(t, port)

	// Rapid fire it then check we're still happy
	for i := 0; i < 30; i++ {
		getMetrics(t, port)
		time.Sleep(1 * time.Second)
	}
	time.Sleep(11 * time.Second)

	// Now actually get the metrics (after waiting for some to become available)
	metrics := getMetrics(t, port)
	if len(metrics) <= 0 {
		t.Error("Expected some metrics to be returned but had none")
	}

	// Stop the container cleanly
	stopContainer(t, cli, id)
}

func TestSlowPrometheus(t *testing.T) {
	t.Parallel()

	cli := ce.NewContainerClient()
	id := runContainerWithPorts(t, cli, metricsContainerConfig(), []int{defaultMetricPort})
	cleanupAfterTest(t, cli, id, false)
	port, err := cli.GetContainerPort(id, defaultMetricPort)
	if err != nil {
		t.Fatal(err)
	}

	// Now the container is ready we prod the prometheus endpoint until it's up.
	waitForMetricReady(t, port)

	// Call once as mq_prometheus 'ignores' the first call and will not return any metrics
	getMetrics(t, port)

	// Send a request twice over a long period and check we're still happy
	for i := 0; i < 2; i++ {
		time.Sleep(30 * time.Second)
		metrics := getMetrics(t, port)
		if len(metrics) <= 0 {
			t.Error("Expected some metrics to be returned but had none")
		}

	}

	// Stop the container cleanly
	stopContainer(t, cli, id)
}

func TestContainerRestart(t *testing.T) {
	t.Parallel()

	cli := ce.NewContainerClient()
	id := runContainerWithPorts(t, cli, metricsContainerConfig(), []int{defaultMetricPort})
	cleanupAfterTest(t, cli, id, false)
	port, err := cli.GetContainerPort(id, defaultMetricPort)
	if err != nil {
		t.Fatal(err)
	}

	// Now the container is ready we prod the prometheus endpoint until it's up.
	waitForMetricReady(t, port)

	// Call once as mq_prometheus 'ignores' the first call and will not return any metrics
	getMetrics(t, port)
	time.Sleep(15 * time.Second)

	// Now actually get the metrics (after waiting for some to become available)
	metrics := getMetrics(t, port)
	if len(metrics) <= 0 {
		t.Fatal("Expected some metrics to be returned before the restart but had none...")
	}

	// Stop the container cleanly
	stopContainer(t, cli, id)
	// Start the container cleanly
	startContainer(t, cli, id)
	port, err = cli.GetContainerPort(id, defaultMetricPort)
	if err != nil {
		t.Fatal(err)
	}

	// Now the container is ready we prod the prometheus endpoint until it's up.
	waitForMetricReady(t, port)

	// Call once as mq_prometheus 'ignores' the first call and will not return any metrics
	getMetrics(t, port)
	time.Sleep(15 * time.Second)

	// Now actually get the metrics (after waiting for some to become available)
	metrics = getMetrics(t, port)
	if len(metrics) <= 0 {
		t.Error("Expected some metrics to be returned after the restart but had none...")
	}

	// Stop the container cleanly
	stopContainer(t, cli, id)
}

func TestQMRestart(t *testing.T) {
	t.Parallel()

	cli := ce.NewContainerClient()
	id := runContainerWithPorts(t, cli, metricsContainerConfig(), []int{defaultMetricPort})
	cleanupAfterTest(t, cli, id, false)

	port, err := cli.GetContainerPort(id, defaultMetricPort)
	if err != nil {
		t.Fatal(err)
	}

	// Now the container is ready we prod the prometheus endpoint until it's up.
	waitForMetricReady(t, port)

	// Call once as mq_prometheus 'ignores' the first call and will not return any metrics
	getMetrics(t, port)
	time.Sleep(15 * time.Second)

	// Now actually get the metrics (after waiting for some to become available)
	metrics := getMetrics(t, port)
	if len(metrics) <= 0 {
		t.Fatal("Expected some metrics to be returned before the restart but had none...")
	}

	// Restart just the QM (to simulate a lost connection)
	t.Log("Stopping queue manager\n")
	rc, out := execContainer(t, cli, id, "", []string{"endmqm", "-w", "-r", defaultMetricQMName})
	if rc != 0 {
		t.Fatalf("Failed to stop the queue manager. rc=%d, err=%s", rc, out)
	}
	t.Log("starting queue manager\n")
	rc, out = execContainer(t, cli, id, "", []string{"strmqm", defaultMetricQMName})
	if rc != 0 {
		t.Fatalf("Failed to start the queue manager. rc=%d, err=%s", rc, out)
	}

	// Wait for the queue manager to come back up
	time.Sleep(10 * time.Second)

	// Now the container is ready we prod the prometheus endpoint until it's up.
	waitForMetricReady(t, port)

	// Call once as mq_prometheus 'ignores' the first call and will not return any metrics
	getMetrics(t, port)
	time.Sleep(15 * time.Second)

	// Now actually get the metrics (after waiting for some to become available)
	metrics = getMetrics(t, port)
	if len(metrics) <= 0 {
		t.Errorf("Expected some metrics to be returned after the restart but had none...")
	}

	// Stop the container cleanly
	stopContainer(t, cli, id)
}

func TestValidValues(t *testing.T) {
	t.Parallel()

	cli := ce.NewContainerClient()
	id := runContainerWithPorts(t, cli, metricsContainerConfig(), []int{defaultMetricPort})
	cleanupAfterTest(t, cli, id, false)
	// hostname := getIPAddress(t, cli, id)
	port, err := cli.GetContainerPort(id, defaultMetricPort)
	if err != nil {
		t.Fatal(err)
	}
	// Now the container is ready we prod the prometheus endpoint until it's up.
	waitForMetricReady(t, port)

	// Call once as mq_prometheus 'ignores' the first call and will not return any metrics
	getMetrics(t, port)
	time.Sleep(15 * time.Second)

	// Now actually get the metrics (after waiting for some to become available)
	metrics := getMetrics(t, port)
	if len(metrics) <= 0 {
		t.Fatal("Expected some metrics to be returned but had none...")
	}

	// Check that the values for each metric are valid numbers
	// can be either int, float or exponential - all these can be parsed by ParseFloat function
	for _, e := range metrics {
		if _, err := strconv.ParseFloat(e.Value, 64); err != nil {
			t.Errorf("Value (%s) for key (%s) is not a valid number", e.Value, e.Key)
		}
	}

	// Stop the container cleanly
	stopContainer(t, cli, id)
}

func TestChangingValues(t *testing.T) {
	t.Parallel()

	cli := ce.NewContainerClient()
	id := runContainerWithPorts(t, cli, metricsContainerConfig(), []int{1414, defaultMetricPort})
	cleanupAfterTest(t, cli, id, false)
	// hostname := getIPAddress(t, cli, id)
	port, err := cli.GetContainerPort(id, defaultMetricPort)
	if err != nil {
		t.Fatal(err)
	}
	// Now the container is ready we prod the prometheus endpoint until it's up.
	waitForMetricReady(t, port)

	// Call once as mq_prometheus 'ignores' the first call and will not return any metrics
	getMetrics(t, port)
	time.Sleep(15 * time.Second)

	// Now actually get the metrics (after waiting for some to become available)
	metrics := getMetrics(t, port)
	if len(metrics) <= 0 {
		t.Fatal("Expected some metrics to be returned but had none...")
	}

	// Check we have no FDC files to start
	for _, e := range metrics {
		if e.Key == "ibmmq_qmgr_mq_fdc_file_count" {
			if e.Value != "0" {
				t.Fatalf("Expected %s to have a value of 0 but was %s", e.Key, e.Value)
			}
		}
	}

	// Send invalid data to the MQ listener to generate a FDC
	noport, err := cli.GetContainerPort(id, 1414)
	if err != nil {
		t.Fatal(err)
	}
	listener := fmt.Sprintf("localhost:%s", noport)
	conn, err := net.Dial("tcp", listener)
	if err != nil {
		t.Fatalf("Could not connect to the listener - %v", err)
	}
	fmt.Fprintf(conn, "THIS WILL GENERATE A FDC!")
	conn.Close()

	// Now actually get the metrics (after waiting for some to become available)
	time.Sleep(25 * time.Second)
	metrics = getMetrics(t, port)
	if len(metrics) <= 0 {
		t.Fatal("Expected some metrics to be returned but had none...")
	}

	// Check that there is now 1 FDC file
	for _, e := range metrics {
		if e.Key == "ibmmq_qmgr_mq_fdc_file_count" {
			if e.Value != "1" {
				t.Fatalf("Expected %s to have a value of 1 but was %s", e.Key, e.Value)
			}
		}
	}

	// Stop the container cleanly
	stopContainer(t, cli, id)
}
