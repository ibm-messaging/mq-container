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
	"io/ioutil"
	"testing"
	"path/filepath"
	"strings"
)

var getQueueManagerTests = []struct {
	file        string
	name        string
	prefix      string
	directory   string
	errorLogDir string
	inifile     string
}{
	{"dspmqinf1.txt", "foo", "/var/mqm", "foo", "/var/mqm/qmgrs/foo/errors","sample1qm.ini"},
	{"dspmqinf2.txt", "a/b", "/var/mqm", "a&b", "/var/mqm/qmgrs/a&b/errors","sample2qm.ini"},
	{"dspmqinf3.txt", "..", "/var/mqm", "!!", "/var/mqm/qmgrs/!!/errors","sample3qm.ini"},
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

			SetUnitTestFlag()

			//qm.ini
			path := GetIniFilePath("qm.ini", qm)
			if path != filepath.Join(qm.Prefix, "qmgrs", qm.Directory, "qm.ini") {
				t.Errorf("Unexpected directory=%s", path)
			}

			path = GetIniFilePath("/etc/mqm/1/2/3/4/5/qm.ini", qm)
			if path != filepath.Join(qm.Prefix, "qmgrs",qm.Directory, "qm.ini") {
				t.Errorf("Unexpected directory=%s", path)
			}

			path = GetIniFilePath("/var/mqm/qm.ini", qm)
			if path != filepath.Join("/var/mqm/qmgrs", qm.Directory, "qm.ini") {
				t.Errorf("Unexpected directory=%s", path)
			}

			//mqs.ini
			path = GetIniFilePath("mqs.ini", qm)
			if path != filepath.Join("/var/mqm/", "mqs.ini") {
				t.Errorf("Unexpected directory=%s", path)
			}

			path = GetIniFilePath("/etc/mqm/a/x/b/c/y/mqs.ini", qm)
			if path != filepath.Join(qm.Prefix, "mqs.ini") {
				t.Errorf("Unexpected directory=%s", path)
			}

			//mqat.ini
			path = GetIniFilePath("mqat.ini", qm)
			if path != filepath.Join("/var/mqm/", "mqat.ini") {
				t.Errorf("Unexpected directory=%s", path)
			}

			userconfigbytes, err := ioutil.ReadFile(table.inifile)
			if err != nil {
				t.Errorf("Error occured while reading userconfig file=%s", err.Error())
			}
			userconfigstring := string(userconfigbytes)

			qmconfigbytes, err := ioutil.ReadFile("qm.ini")
			if err != nil {
				t.Errorf("Error occured while reading qmconfig file=%s", err.Error())
			}
			qmconfigstring := string(qmconfigbytes)
			SetQMConfigStr(qmconfigstring)
			WriteToIniFile(userconfigstring,"")
			qmconfigstring = GetQMConfigStr()

			if !strings.ContainsAny(qmconfigstring,userconfigstring ) {
				t.Errorf("Expected stanza not found. \n qmconfig:\n%s\n userconfig:%s\n", qmconfigstring, userconfigstring)
			}
		})
	}
}
