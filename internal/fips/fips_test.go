/*
© Copyright IBM Corporation 2022

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

// Package keystore contains code to create and update keystores
package fips

import (
	"fmt"
	"os"
	"testing"
)

func TestEnableFIPSAuto(t *testing.T) {
	ProcessFIPSType(nil)
	// Test default "auto"
	fipsType := IsFIPSEnabled()
	if fipsType {
		t.Errorf("Expected FIPS OFF but got %v\n", fipsType)
	}
}

func TestEnableFIPSTrue(t *testing.T) {
	// Test MQ_ENABLE_FIPS=true
	os.Setenv("MQ_ENABLE_FIPS", "true")
	fmt.Println(os.Getenv("MQ_ENABLE_FIPS"))
	ProcessFIPSType(nil)
	fipsType := IsFIPSEnabled()
	if !fipsType {
		t.Errorf("Expected FIPS ON but got %v\n", fipsType)
	}
}

func TestEnableFIPSFalse(t *testing.T) {
	// Test MQ_ENABLE_FIPS=false
	os.Setenv("MQ_ENABLE_FIPS", "false")
	ProcessFIPSType(nil)
	fipsType := IsFIPSEnabled()
	if fipsType {
		t.Errorf("Expected FIPS OFF but got %v\n", fipsType)
	}

}

func TestEnableFIPSInvalid(t *testing.T) {
	// Test MQ_ENABLE_FIPS with invalid value
	os.Setenv("MQ_ENABLE_FIPS", "falseOff")
	ProcessFIPSType(nil)
	fipsType := IsFIPSEnabled()
	if fipsType {
		t.Errorf("Expected FIPS OFF but got %v\n", fipsType)
	}
}
