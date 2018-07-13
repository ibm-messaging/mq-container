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
package mqmetric

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/ibm-messaging/mq-golang/ibmmq"
)

func TestNormalise(t *testing.T) {
	testCases := []struct {
		dataType     int32
		dataTypeName string
		value        int64
		expected     float64
	}{
		{ibmmq.MQIAMO_MONITOR_PERCENT, "MQIAMO_MONITOR_PERCENT", 1, 0.01},
		{ibmmq.MQIAMO_MONITOR_PERCENT, "MQIAMO_MONITOR_PERCENT", 1000000, 10000},
		{ibmmq.MQIAMO_MONITOR_HUNDREDTHS, "MQIAMO_MONITOR_HUNDREDTHS", 1, 0.01},
		{ibmmq.MQIAMO_MONITOR_HUNDREDTHS, "MQIAMO_MONITOR_HUNDREDTHS", 1000000, 10000},
		{ibmmq.MQIAMO_MONITOR_MB, "MQIAMO_MONITOR_MB", 1000000, 1048576000000},
		{ibmmq.MQIAMO_MONITOR_GB, "MQIAMO_MONITOR_GB", 1000000, 1073741824000000},
		{ibmmq.MQIAMO_MONITOR_MICROSEC, "MQIAMO_MONITOR_MICROSEC", 1000000, 1},
		{ibmmq.MQIAMO_MONITOR_MICROSEC, "MQIAMO_MONITOR_MICROSEC", 1, 0.000001},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("%s/%d", tc.dataTypeName, tc.value), func(t *testing.T) {
			test := MonElement{Datatype: tc.dataType}
			returned := Normalise(&test, "", tc.value)
			if returned != tc.expected {
				t.Logf("Gave %s, %d. Expected: %f, Got: %f", tc.dataTypeName, tc.value, tc.expected, returned)
				t.Fail()
			}
		})
	}
}

func TestReadPatterns(t *testing.T) {
	const filename = "testFile"
	testCases := []struct {
		name     string
		value    string
		expected string
	}{
		{"golden", "test1=yes\ntest2=no\n", "test1=yes,test2=no"},
		{"nolf", "test1=yes\ntest2=no", "test1=yes,test2=no"},
		{"crlf", "test1=yes\r\ntest2=no\r\n", "test1=yes,test2=no"},
		{"oneliner", "test1=yes,test2=no\ntest3=maybe", "test1=yes,test2=no,test3=maybe"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			//Create dummy test file
			err := ioutil.WriteFile(filename, []byte(tc.value), 0644)
			if err != nil {
				t.Fatalf("Could not create test file - %v", err)
			}
			defer os.Remove(filename)

			returned, err := ReadPatterns(filename)
			if err != nil {
				t.Logf("Got error while running ReadPatterns - %v", err)
				t.Fail()
			} else if returned != tc.expected {
				t.Logf("File was not parsed correctly. Expected: %s. Got: %s", tc.expected, returned)
				t.Fail()
			}
		})
	}
}
func TestFormatDescription(t *testing.T) {
	testCases := []struct {
		value    string
		expected string
	}{
		{"hello", "hello_count"},
		{"no space", "no_space_count"},
		{"no/slash", "no_slash_count"},
		{"no-dash", "no_dash_count"},
		{"single___underscore", "single_underscore_count"},
		{"single__underscore__multiplace", "single_underscore_multiplace_count"},
		{"ALLLOWER", "alllower_count"},
		{"this_bytes_written_switch", "this_written_switch_count"},
		{"this_byte_max_switch", "this_max_switch_count"},
		{"this_seconds_in_use_switch", "this_in_use_switch_count"},
		{"this messages_expired_switch", "this_expired_messages_switch_count"},
		{"this_seconds_max_switch", "this_max_switch_count"},
		{"this_count_max_switch", "this_max_switch_count"},
		{"this_percentage_max_switch", "this_max_switch_count"},
	}

	for _, tc := range testCases {
		t.Run(tc.value, func(t *testing.T) {
			elem := MonElement{
				Description: tc.value,
			}
			returned := formatDescription(&elem)
			if returned != tc.expected {
				t.Logf("Gave %s. Expected: %s, Got: %s", tc.value, tc.expected, returned)
				t.Fail()
			}
		})
	}
}

func TestSuffixes(t *testing.T) {
	baseDescription := "test_suffix"
	testCases := []struct {
		name     string
		value    int32
		expected string
	}{
		{"MQIAMO_MONITOR_MB", ibmmq.MQIAMO_MONITOR_MB, baseDescription + "_bytes"},
		{"MQIAMO_MONITOR_GB", ibmmq.MQIAMO_MONITOR_GB, baseDescription + "_bytes"},
		{"MQIAMO_MONITOR_MICROSEC", ibmmq.MQIAMO_MONITOR_MICROSEC, baseDescription + "_seconds"},
		{"MQIAMO_MONITOR_PERCENT", ibmmq.MQIAMO_MONITOR_PERCENT, baseDescription + "_percentage"},
		{"MQIAMO_MONITOR_HUNDREDTHS", ibmmq.MQIAMO_MONITOR_HUNDREDTHS, baseDescription + "_percentage"},
		{"0", 0, baseDescription + "_count"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			elem := MonElement{
				Description: baseDescription,
				Datatype:    tc.value,
			}
			returned := formatDescription(&elem)
			if returned != tc.expected {
				t.Logf("Gave %s/%d Expected: %s, Got: %s", baseDescription, tc.value, tc.expected, returned)
				t.Fail()
			}
		})
	}

	// special case log_bytes
	t.Run("log_bytes", func(t *testing.T) {
		elem := MonElement{
			Description: "log_test_suffix",
			Datatype:    0,
		}
		returned := formatDescription(&elem)
		if returned != "log_test_suffix_bytes" {
			t.Logf("Gave log_test_suffix/0 Expected: %s, Got: %s", "log_test_suffix_bytes", returned)
			t.Fail()
		}
	})

	// special case log_total
	t.Run("log_bytes", func(t *testing.T) {
		elem := MonElement{
			Description: "log_total_suffix",
			Datatype:    0,
		}
		returned := formatDescription(&elem)
		if returned != "log_suffix_total" {
			t.Logf("Gave log_total_suffix/0 Expected: %s, Got: %s", "log_suffix_total", returned)
			t.Fail()
		}
	})
}

func TestParsePCFResponse(t *testing.T) {
	testCases := []struct {
		name   string
		params []ibmmq.PCFParameter
	}{
		{
			"noParams",
			make([]ibmmq.PCFParameter, 0),
		},
		{
			"oneParam",
			[]ibmmq.PCFParameter{
				ibmmq.PCFParameter{
					Type:           ibmmq.MQCFT_STRING, // String
					Parameter:      ibmmq.MQCACF_APPL_NAME,
					String:         []string{"HELLOTEST"},
					ParameterCount: 1,
				},
			},
		},
		{
			"twoParams",
			[]ibmmq.PCFParameter{
				ibmmq.PCFParameter{
					Type:           ibmmq.MQCFT_STRING, // String
					Parameter:      ibmmq.MQCACF_APPL_NAME,
					String:         []string{"HELLOTEST"},
					ParameterCount: 1,
				},
				ibmmq.PCFParameter{
					Type:           ibmmq.MQCFT_STRING, // String
					Parameter:      ibmmq.MQCACF_APPL_NAME,
					String:         []string{"FIRST"},
					ParameterCount: 1,
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cfh := ibmmq.NewMQCFH()
			cfh.Type = ibmmq.MQCFT_RESPONSE
			cfh.ParameterCount = int32(len(tc.params))
			headerbytes := cfh.Bytes()

			parmbytes := []byte{}
			for _, parm := range tc.params {
				parmbytes = append(parmbytes, parm.Bytes()...)
			}
			messagebytes := append(headerbytes, parmbytes...)

			returned, last := parsePCFResponse(messagebytes)

			if len(returned) != len(tc.params) && !last {
				t.Logf("Gave header and parameter. Expected: 1, false , Got: %d, %t", len(returned), last)
				t.Fail()
			} else {
				for i := range returned {
					t.Logf("Checking param %d", i)
					checkParamsMatch(returned[i], &tc.params[i], t)
				}
			}
		})
	}
}

func checkParamsMatch(returned *ibmmq.PCFParameter, expected *ibmmq.PCFParameter, t *testing.T) {
	if returned.Type != expected.Type {
		t.Logf("Returned parameter 'Type' did not match. Expected: %d, Got: %d", expected.Type, returned.Type)
		t.Fail()
	}
	if returned.Parameter != expected.Parameter {
		t.Logf("Returned parameter 'Parameter' did not match. Expected: %d, Got: %d", expected.Parameter, returned.Parameter)
		t.Fail()
	}
	if len(returned.String) != len(expected.String) {
		t.Logf("Length of Returned parameter 'String' did not match. Expected: %d, Got: %d", len(expected.String), len(returned.String))
		t.Fail()
	} else {
		for i := range returned.String {
			if returned.String[i] != expected.String[i] {
				t.Logf("Returned parameter 'String[%d]' did not match. Expected: %s, Got: %s", i, expected.String[i], returned.String[i])
				t.Fail()
			}
		}
	}
	if len(returned.Int64Value) != len(expected.Int64Value) {
		t.Logf("Length of Returned parameter 'Int64Value' did not match. Expected: %d, Got: %d", len(expected.Int64Value), len(returned.Int64Value))
		t.Fail()
	} else {
		for i := range returned.Int64Value {
			if returned.Int64Value[i] != expected.Int64Value[i] {
				t.Logf("Returned parameter 'Int64Value[%d]' did not match. Expected: %d, Got: %d", i, expected.Int64Value[i], returned.Int64Value[i])
				t.Fail()
			}
		}
	}
	if len(returned.GroupList) != len(expected.GroupList) {
		t.Logf("Length of Returned parameter 'GroupList' did not match. Expected: %d, Got: %d", len(expected.GroupList), len(returned.GroupList))
		t.Fail()
	} else {
		for i := range returned.GroupList {
			checkParamsMatch(returned.GroupList[i], expected.GroupList[i], t)
		}
	}
}
