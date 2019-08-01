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
package mqini

import (
	"bufio"
	"io/ioutil"
	"strings"
	"testing"
)

var getQueueManagerTests = []struct {
	file        string
	name        string
	prefix      string
	directory   string
	errorLogDir string
}{
	{"dspmqinf1.txt", "foo", "/var/mqm", "foo", "/var/mqm/qmgrs/foo/errors"},
	{"dspmqinf2.txt", "a/b", "/var/mqm", "a&b", "/var/mqm/qmgrs/a&b/errors"},
	{"dspmqinf3.txt", "..", "/var/mqm", "!!", "/var/mqm/qmgrs/!!/errors"},
}

func TestGetQueueManager(t *testing.T) {
	for _, table := range getQueueManagerTests {
		t.Run(table.file, func(t *testing.T) {
			b, err := ioutil.ReadFile(table.file)
			if err != nil {
				t.Fatal(err)
			}
			qm, err := getQueueManagerFromStanza(string(b))
			if err != nil {
				t.Fatal(err)
			}
			t.Logf("%#v", qm)
			if qm.Name != table.name {
				t.Errorf("Expected name=%v; got %v", table.name, qm.Name)
			}
			if qm.Prefix != table.prefix {
				t.Errorf("Expected prefix=%v; got %v", table.prefix, qm.Prefix)
			}
			if qm.Directory != table.directory {
				t.Errorf("Expected directory=%v; got %v", table.directory, qm.Directory)
			}

			// Test
			d := GetErrorLogDirectory(qm)
			if d != table.errorLogDir {
				t.Errorf("Expected error log directory=%v; got %v", table.errorLogDir, d)
			}
		})
	}
}

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

	iniFileBytes, err := ioutil.ReadFile("sample1qm.ini")
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

	iniFileBytes, err := ioutil.ReadFile("sample2qm.ini")
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

func TestIniFile3Update(t *testing.T) {

	iniFileBytes, err := ioutil.ReadFile("sample3qm.ini")
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

func TestIniFile4Update(t *testing.T) {

	iniFileBytes, err := ioutil.ReadFile("sample4qm.ini")
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
		if !strings.Contains(qmConfig, line) && !strings.Contains(atConfig, line) {
			t.Errorf("Expected stanza line not found in updated string. line=%s\n, Stanza:%s\n", line, qmConfig)
			break
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
