/*
© Copyright IBM Corporation 2019

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

package copy

import (
	"fmt"
	"io"
	"os"
)

func CopyFileMode(src, dest string, perm os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open %s for copy: %v", src, err)
	}
	defer in.Close()

	out, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY, perm)
	if err != nil {
		return fmt.Errorf("failed to open %s for copy: %v", dest, err)
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	if err != nil {
		return err
	}
	err = out.Close()
	return err
}

// CopyFile copies the specified file
func CopyFile(src, dest string) error {
	return CopyFileMode(src, dest, 0770)
}
