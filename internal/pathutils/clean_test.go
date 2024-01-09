/*
© Copyright IBM Corporation 2023

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

package pathutils

import (
	"strings"
	"testing"
)

func TestClean(t *testing.T) {

	tests := []struct {
		uncleaned string
		filepath      string
		cleaned   string
	}{
		{"/a/rooted/path", "some.file", "/a/rooted/path/some.file"},
		{"../../../a/relative/path", "abc.txt", "a/relative/path/abc.txt"},
		{"a/path" + ".p12", "some.file", "a/path.p12/some.file"},
		{"/", "bin", "/bin"},
		{"abc/def", "../../a/relative/path", "abc/def/a/relative/path"},
	}

	for _, test := range tests {
		cleaned := CleanPath(test.uncleaned, test.filepath)

		if !strings.EqualFold(cleaned, test.cleaned) {
			t.Fatalf("file path sanitisation failed. Expected %s but got %s\n", test.cleaned, cleaned)
		}
	}
}
