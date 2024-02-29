/*

© Copyright IBM Corporation 2024


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

// Package pathutils contains code to provide sanitised file paths
package pathutils

import (
	"path"
	"path/filepath"
)

// CleanPath returns the result of joining a series of sanitised file paths (preventing directory traversal for each path)
// If the first path is relative, a relative path is returned
func CleanPath(paths ...string) string {
	if len(paths) == 0 {
		return ""
	}
	var combined string
	if !path.IsAbs(paths[0]) {
		combined = "./"
	}
	for _, part := range paths {
		combined = filepath.Join(combined, filepath.FromSlash(path.Clean("/"+part)))
	}
	return combined
}
