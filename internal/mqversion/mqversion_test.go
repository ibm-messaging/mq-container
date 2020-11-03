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

import "testing"

func TestCompareLower(t *testing.T) {
	checkVersion := "9.9.9.9"
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
