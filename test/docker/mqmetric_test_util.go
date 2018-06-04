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
	"bufio"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/docker/docker/api/types/container"
)

type mqmetric struct {
	Key    string
	Value  string
	Labels map[string]string
}

const defaultMetricURL = "/metrics"
const defaultMetricPort = 9157
const defaultMQNamespace = "ibmmq"
const defaultMetricQMName = "qm1"

func getMetrics(t *testing.T, port string) []mqmetric {
	returned := []mqmetric{}
	urlToUse := fmt.Sprintf("http://localhost:%s%s", port, defaultMetricURL)
	resp, err := http.Get(urlToUse)
	if err != nil {
		t.Fatalf("Error from HTTP GET for metrics: %v", err)
		return returned
	}
	defer resp.Body.Close()
	metricsRaw, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Error reading metrics data: %v", err)
		return returned
	}
	return convertRawMetricToMap(t, string(metricsRaw))
}

// Also filters out all non "ibmmq" metrics
func convertRawMetricToMap(t *testing.T, input string) []mqmetric {
	returnList := []mqmetric{}
	scanner := bufio.NewScanner(strings.NewReader(input))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "#") {
			// Comment line of HELP or TYPE. Ignore
			continue
		}
		if !strings.HasPrefix(line, defaultMQNamespace) {
			// Not an ibmmq_ metric. Ignore
			continue
		}
		//It's an IBM MQ metric!
		key, value, labelMap, err := convertMetricLineToMetric(line)
		if err != nil {
			t.Fatalf("ibmmq_ metric could not be deciphered - %v", err)
		}

		toAdd := mqmetric{
			Key:    key,
			Value:  value,
			Labels: labelMap,
		}

		returnList = append(returnList, toAdd)
	}

	return returnList
}

func convertMetricLineToMetric(input string) (string, string, map[string]string, error) {
	// Lines are in the form "<key>{<labels>}<value>" or "<key> <value>"
	// Get the key and value while skipping the label
	var key, value string
	labelMap := make(map[string]string)
	if strings.Contains(input, "{") {
		// Get key
		splitted := strings.Split(input, "{")
		if len(splitted) != 2 {
			return "", "", labelMap, fmt.Errorf("Could not split by { Expected 2 but got %d - %s", len(splitted), input)
		}
		key = strings.TrimSpace(splitted[0])

		// Get value
		splitted = strings.Split(splitted[1], "}")
		if len(splitted) != 2 {
			return "", "", labelMap, fmt.Errorf("Could not split by } Expected 2 but got %d - %s", len(splitted), input)
		}
		value = strings.TrimSpace(splitted[1])

		// Get labels
		allLabels := strings.Split(splitted[0], ",")
		for _, e := range allLabels {
			labelPair := strings.Split(e, "=")
			if len(labelPair) != 2 {
				return "", "", labelMap, fmt.Errorf("Could not split label by '=' Expected 2 but got %d - %s", len(labelPair), e)
			}
			lkey := strings.TrimSpace(labelPair[0])
			lvalue := strings.TrimSpace(labelPair[1])
			lvalue = strings.Trim(lvalue, "\"")
			labelMap[lkey] = lvalue
		}

	} else {
		splitted := strings.Split(input, " ")
		if len(splitted) != 2 {
			return "", "", labelMap, fmt.Errorf("Could not split by ' ' Expected 2 but got %d - %s", len(splitted), input)
		}
		key = strings.TrimSpace(splitted[0])
		value = strings.TrimSpace(splitted[1])
	}
	return key, value, labelMap, nil
}

func waitForMetricReady(t *testing.T, port string) {
	timeout := 12 // 12 * 5 = 1 minute
	for i := 0; i < timeout; i++ {
		urlToUse := fmt.Sprintf("http://localhost:%s", port)
		resp, err := http.Get(urlToUse)
		if err == nil {
			resp.Body.Close()
			return
		}

		time.Sleep(time.Second * 10)
	}
	t.Fatalf("Metric endpoint failed to startup in timely manner")
}

func metricsContainerConfig() *container.Config {
	return &container.Config{
		Env: []string{
			"LICENSE=accept",
			"MQ_QMGR_NAME=" + defaultMetricQMName,
			"MQ_ENABLE_METRICS=true",
		},
	}
}
