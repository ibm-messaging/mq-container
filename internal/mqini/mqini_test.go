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
			qm, err := getQueueManagerFromStanza(b)
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
