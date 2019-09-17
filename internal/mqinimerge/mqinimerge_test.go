/*
© Copyright IBM Corporation 2018, 2019

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
package mqinimerge

import (
	"bufio"
	"io/ioutil"
	"strings"
	"testing"
)

func TestIniFileStanzas(t *testing.T) {
	PopulateAllAvailableStanzas()

	checkReturns("ApiExitLocal", true, true, t)
	checkReturns("Channels", true, true, t)
	checkReturns("TCP", true, true, t)
	checkReturns("ServiceComponent", true, true, t)
	checkReturns("Service", true, true, t)
	checkReturns("AccessMode", true, true, t)
	checkReturns("RestrictedMode", true, true, t)
	checkReturns("XAResourceManager", true, true, t)
	checkReturns("SSL", true, true, t)
	checkReturns("Security", true, true, t)
	checkReturns("TuningParameters", true, true, t)
	checkReturns("ABC", false, false, t)
	checkReturns("#1234ABD", true, false, t)
	checkReturns("AllActivityTrace", false, true, t)
	checkReturns("ApplicationTrace", false, true, t)
	checkReturns("xyz123abvc", false, false, t)
}

func TestIniFile1Update(t *testing.T) {
	iniFileBytes, err := ioutil.ReadFile("test1qm.ini")
	if err != nil {
		t.Errorf("Unexpected error: [%s]\n", err.Error())
	}
	userconfig := string(iniFileBytes)
	qmConfig, atConfig, err := PrepareConfigStanzasToWrite(userconfig)
	if err != nil {
		t.Errorf("Unexpected error: [%s]\n", err.Error())
	}
	if len(atConfig) == 0 {
		t.Errorf("Unexpected stanza file update: mqat.ini[%s]\n", atConfig)
	}
	if len(qmConfig) == 0 {
		t.Errorf("Expected stanza file not found: qm.ini\n")
	}

	scanner := bufio.NewScanner(strings.NewReader(userconfig))
	scanner.Split(bufio.ScanLines)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.Contains(qmConfig, line) {
			t.Errorf("Expected stanza line not found in updated string. line=%s\n, Stanza:%s\n", line, qmConfig)
			break
		}
	}
}

func TestIniFile2Update(t *testing.T) {
	iniFileBytes, err := ioutil.ReadFile("test2qm.ini")
	if err != nil {
		t.Errorf("Unexpected error: [%s]\n", err.Error())
	}
	userconfig := string(iniFileBytes)
	qmConfig, atConfig, err := PrepareConfigStanzasToWrite(userconfig)
	if err != nil {
		t.Errorf("Unexpected error: [%s]\n", err.Error())
	}
	if len(atConfig) == 0 {
		t.Errorf("Expected stanza file not found: mqat.ini\n")
	}
	if len(qmConfig) == 0 {
		t.Errorf("Expected stanza file not found: qm.ini\n")
	}

	scanner := bufio.NewScanner(strings.NewReader(userconfig))
	scanner.Split(bufio.ScanLines)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.Contains(atConfig, line) {
			t.Errorf("Expected stanza line not found in updated string. line=%s\n, Stanza:%s\n", line, qmConfig)
			break
		}
	}
}

func TestIniFile3Update(t *testing.T) {
	i := 0
	iniFileBytes, err := ioutil.ReadFile("test3qm.ini")
	if err != nil {
		t.Errorf("Unexpected error: [%s]\n", err.Error())
	}
	userconfig := string(iniFileBytes)
	qmConfig, atConfig, err := PrepareConfigStanzasToWrite(userconfig)
	if err != nil {
		t.Errorf("Unexpected error: [%s]\n", err.Error())
	}
	if len(qmConfig) == 0 {
		t.Errorf("Unexpected stanza file update: qm.ini[%s]\n", atConfig)
	}
	if len(atConfig) == 0 {
		t.Errorf("Expected stanza file not found: mqat.ini\n")
	}

	scanner := bufio.NewScanner(strings.NewReader(userconfig))
	scanner.Split(bufio.ScanLines)
	for scanner.Scan() {
		line := scanner.Text()
		i++
		//first 20 lines of test3qm.ini shall go into qm.ini file and rest into mqat.ini file.
		if i < 20 {
			if !strings.Contains(qmConfig, line) {
				t.Errorf("Expected stanza line not found in updated string. line=%s\n, Stanza:%s\n", line, qmConfig)
			}
		} else if i > 20 {
			if !strings.Contains(atConfig, line) {
				t.Errorf("Expected stanza line not found in updated string. line=%s\n, Stanza:%s\n", line, qmConfig)
			}
		}
	}
}

func checkReturns(stanza string, isqmini bool, shouldexist bool, t *testing.T) {
	exists, filename := ValidateStanzaToWrite(stanza)
	if exists != shouldexist {
		t.Errorf("Stanza should exist %t but found was %t", shouldexist, exists)
	}

	if shouldexist {
		if isqmini {
			if filename != "qm.ini" {
				t.Errorf("Expected filename:qm.ini for stanza:%s. But got %s", stanza, filename)
			}
		} else {
			if filename != "mqat.ini" {
				t.Errorf("Expected filename:mqat.ini for stanza:%s. But got %s", stanza, filename)
			}
		}
	}
}
