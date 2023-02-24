/*
© Copyright IBM Corporation 2020

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

package mqversion

import (
	"fmt"
	"testing"
)

func TestCompareLower(t *testing.T) {
	checkVersion := "99.99.99.99"
	mqVersionCheck, err := Compare(checkVersion)
	if err != nil {
		t.Fatalf("Failed to compare MQ versions: %v", err)
	}
	if mqVersionCheck != -1 {
		t.Errorf("MQ version compare result failed. Expected -1, Got %v", mqVersionCheck)
	}
}

func TestCompareHigher(t *testing.T) {
	checkVersion := "1.1.1.1"
	mqVersionCheck, err := Compare(checkVersion)
	if err != nil {
		t.Fatalf("Failed to compare MQ versions: %v", err)
	}
	if mqVersionCheck != 1 {
		t.Errorf("MQ version compare result failed. Expected 1, Got %v", mqVersionCheck)
	}
}

func TestCompareEqual(t *testing.T) {
	checkVersion, err := Get()
	if err != nil {
		t.Fatalf("Failed to get current MQ version: %v", err)
	}
	mqVersionCheck, err := Compare(checkVersion)
	if err != nil {
		t.Fatalf("Failed to compare MQ versions: %v", err)
	}
	if mqVersionCheck != 0 {
		t.Errorf("MQ version compare result failed. Expected 0, Got %v", mqVersionCheck)
	}
}

func TestVersionValid(t *testing.T) {
	checkVersion, err := Get()
	if err != nil {
		t.Fatalf("Failed to get current MQ version: %v", err)
	}
	_, err = parseVRMF(checkVersion)
	if err != nil {
		t.Fatalf("Validation of MQ version failed: %v", err)
	}
}

func TestValidVRMF(t *testing.T) {
	validVRMFs := map[string]vrmf{
		"1.0.0.0":         {1, 0, 0, 0},
		"10.0.0.0":        {10, 0, 0, 0},
		"1.10.0.0":        {1, 10, 0, 0},
		"1.0.10.0":        {1, 0, 10, 0},
		"1.0.0.10":        {1, 0, 0, 10},
		"999.998.997.996": {999, 998, 997, 996},
	}
	for test, expect := range validVRMFs {
		t.Run(test, func(t *testing.T) {
			parsed, err := parseVRMF(test)
			if err != nil {
				t.Fatalf("Unexpectedly failed to parse VRMF '%s': %s", test, err.Error())
			}
			if *parsed != expect {
				t.Fatalf("VRMF not parsed as expected. Expected '%v', got '%v'", parsed, expect)
			}
		})
	}
}

func TestInvalidVRMF(t *testing.T) {
	invalidVRMFs := []string{
		"not-a-number",
		"9.8.7.string",
		"0.1.2.3",
		"1.0.0.-10",
	}
	for _, test := range invalidVRMFs {
		t.Run(test, func(t *testing.T) {
			parsed, err := parseVRMF(test)
			if err == nil {
				t.Fatalf("Expected error when parsing VRMF '%s', but got none.  VRMF returned: %v", test, parsed)
			}
		})
	}
}

func TestCompare(t *testing.T) {
	tests := []struct {
		current string
		compare string
		expect  int
	}{
		{"1.0.0.1", "1.0.0.1", 0},
		{"1.0.0.1", "1.0.0.0", 1},
		{"1.0.0.1", "1.0.0.2", -1},
		{"9.9.9.9", "10.0.0.0", -1},
		{"9.9.9.9", "9.10.0.0", -1},
		{"9.9.9.9", "9.9.10.0", -1},
		{"9.9.9.9", "9.9.9.10", -1},
	}
	for _, test := range tests {
		t.Run(fmt.Sprintf("%s-%s", test.current, test.compare), func(t *testing.T) {
			baseVRMF, err := parseVRMF(test.current)
			if err != nil {
				t.Fatalf("Could not parse base version '%s': %s", test.current, err.Error())
			}
			compareVRMF, err := parseVRMF(test.compare)
			if err != nil {
				t.Fatalf("Could not parse current version '%s': %s", test.current, err.Error())
			}
			result := baseVRMF.compare(*compareVRMF)
			if result != test.expect {
				t.Fatalf("Expected %d but got %d when comparing '%s' with '%s'", test.expect, result, test.current, test.compare)
			}
			if test.expect == 0 {
				return
			}
			resultReversed := compareVRMF.compare(*baseVRMF)
			if resultReversed != test.expect*-1 {
				t.Fatalf("Expected %d but got %d when comparing '%s' with '%s'", test.expect*-1, resultReversed, test.compare, test.current)
			}
		})
	}
}
