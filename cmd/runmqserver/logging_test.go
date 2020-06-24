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
package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

var formatBasicTests = []struct {
	in          []byte
	outContains string
}{
	{
		[]byte("{\"ibm_datetime\":\"2020/06/24 00:00:00\",\"message\":\"Hello world\"}"),
		"Hello",
	},
	{
		[]byte("{\"ibm_datetime\":\"2020/06/24 00:00:00\",\"message\":\"Hello world\", \"ibm_commentInsert1\":\"foo\"}"),
		"CommentInsert1(foo)",
	},
	{
		[]byte("{\"ibm_datetime\":\"2020/06/24 00:00:00\",\"message\":\"Hello world\", \"ibm_arithInsert1\":1}"),
		"ArithInsert1(1)",
	},
}

func TestFormatBasic(t *testing.T) {
	for i, table := range formatBasicTests {
		t.Run(fmt.Sprintf("%v", i), func(t *testing.T) {
			var inObj map[string]interface{}
			json.Unmarshal(table.in, &inObj)
			t.Logf("Unmarshalled: %+v", inObj)
			out := formatBasic(inObj)
			if !strings.Contains(out, table.outContains) {
				t.Errorf("formatBasic() with input=%v - expected output to contain %v, got %v", string(table.in), table.outContains, out)
			}
		})
	}
}
