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

package logger

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestJSONLogger(t *testing.T) {
	buf := new(bytes.Buffer)
	l, err := NewLogger(buf, true, true, t.Name())
	if err != nil {
		t.Fatal(err)
	}
	s := "Hello world"
	l.Print(s)
	var e map[string]interface{}
	err = json.Unmarshal(buf.Bytes(), &e)
	if err != nil {
		t.Error(err)
	}
	if s != e["message"] {
		t.Errorf("Expected JSON to contain message=%v; got %v", s, buf.String())
	}
}

func TestSimpleLogger(t *testing.T) {
	buf := new(bytes.Buffer)
	l, err := NewLogger(buf, true, false, t.Name())
	if err != nil {
		t.Fatal(err)
	}
	s := "Hello world"
	l.Print(s)
	if !strings.Contains(buf.String(), s) {
		t.Errorf("Expected log output to contain %v; got %v", s, buf.String())
	}
}
