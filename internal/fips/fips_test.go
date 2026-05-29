/*
© Copyright IBM Corporation 2022, 2026

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
	"os"
	"runtime"
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

	// FIPS is disabled on s390x or arm64
	arch := runtime.GOARCH
	if arch == "s390x" || arch == "arm64" {
		t.Skipf("Skipping as FIPS is not compatible with the %s arch", arch)
	}

	// Test MQ_ENABLE_FIPS=true
	os.Setenv("MQ_ENABLE_FIPS", "true")
	t.Log(os.Getenv("MQ_ENABLE_FIPS"))
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

func TestFIPSDisabledOnS390x(t *testing.T) {
	// Test FIPS disabled on s390x
	if runtime.GOARCH != "s390x" {
		t.Skip("Skipping as test is only for s390x arch")
	}

	os.Setenv("MQ_ENABLE_FIPS", "true")
	ProcessFIPSType(nil)
	fipsType := IsFIPSEnabled()
	if fipsType {
		t.Errorf("Expected FIPS OFF but got %v\n", fipsType)
	}
}

func TestFIPSDisabledOnArm64(t *testing.T) {
	// Test FIPS disabled on arm64
	if runtime.GOARCH != "arm64" {
		t.Skip("Skipping as test is only for arm64 arch")
	}

	os.Setenv("MQ_ENABLE_FIPS", "true")
	ProcessFIPSType(nil)
	fipsType := IsFIPSEnabled()
	if fipsType {
		t.Errorf("Expected FIPS OFF but got %v\n", fipsType)
	}
}
