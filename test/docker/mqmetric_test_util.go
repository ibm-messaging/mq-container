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
	"time"
)

type MQMETRIC struct {
	Key    string
	Value  string
	Labels map[string]string
}

const DEFAULT_METRIC_URL = "/metrics"
const DEFAULT_METRIC_PORT = 9157
const DEFAULT_MQ_NAMESPACE = "ibmmq"

func getMetricsFromEndpoint(host string, port int) ([]MQMETRIC, error) {
	returned := []MQMETRIC{}
	if host == "" {
		return returned, fmt.Errorf("Test Error - Host was nil")
	}
	if port <= 0 {
		return returned, fmt.Errorf("Test Error - port was not above 0")
	}
	urlToUse := fmt.Sprintf("http://%s:%d%s", host, port, DEFAULT_METRIC_URL)

	resp, err := http.Get(urlToUse)
	if err != nil {
		return returned, err
	}
	defer resp.Body.Close()
	metricsRaw, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return returned, err
	}

	return convertRawMetricToMap(string(metricsRaw))
}

// Also filters out all non "ibmmq" metrics
func convertRawMetricToMap(input string) ([]MQMETRIC, error) {
	returnList := []MQMETRIC{}
	if input == "" {
		return returnList, fmt.Errorf("Test Error - Raw metric output was nil")
	}

	scanner := bufio.NewScanner(strings.NewReader(input))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "#") {
			// Comment line of HELP or TYPE. Ignore
			continue
		}
		if !strings.HasPrefix(line, DEFAULT_MQ_NAMESPACE) {
			// Not an ibmmq_ metric. Ignore
			continue
		}
		//It's an IBM MQ metric!
		key, value, labelMap, err := convertMetricLineToMetric(line)
		if err != nil {
			return returnList, fmt.Errorf("ibmmq_ metric could not be deciphered - %v", err)
		}

		toAdd := MQMETRIC{
			Key:    key,
			Value:  value,
			Labels: labelMap,
		}

		returnList = append(returnList, toAdd)
	}

	return returnList, nil
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

func waitForMetricReady(host string, port int) error {
	if host == "" {
		return fmt.Errorf("Test Error - Host was nil")
	}
	if port <= 0 {
		return fmt.Errorf("Test Error - port was not above 0")
	}
	timeout := 12 // 12 * 5 = 1 minute
	for i := 0; i < timeout; i++ {
		urlToUse := fmt.Sprintf("http://%s:%d", host, port)
		resp, err := http.Get(urlToUse)
		if err == nil {
			resp.Body.Close()
			return nil
		}

		time.Sleep(time.Second * 5)
	}

	return fmt.Errorf("Metric endpoint failed to startup in timely manner")
}
