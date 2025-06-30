/*
© Copyright IBM Corporation 2025

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

package sensitive

import (
	"fmt"
	"runtime"
	"testing"
)

func TestString(t *testing.T) {
	buf := []byte("initial")
	s := New(buf)
	asString := s.String()

	expectString := "initial"

	if asString != expectString {
		t.Fatalf("String does not equal provided byte slice: actual '%s' != expected '%s'", asString, expectString)
	}
}

func TestExpand(t *testing.T) {
	buf := []byte("initial")
	s := New(buf)

	// Expect writing additional data to move the underlying buffer to a new slice - original buffer should be cleared during move
	s.Write([]byte("+expanded"))
	for i := range len(buf) {
		if buf[i] != 0 {
			t.Fatalf("Byte buffer not zeroed after expansion ('%s')", string(buf))
		}
	}

	expectString := "initial+expanded"
	asString := s.String()
	if asString != expectString {
		t.Fatalf("Expanded string does not match expected: actual '%s' != expected '%s'", asString, expectString)
	}
}

func TestMultipleExpansions(t *testing.T) {
	buf := []byte("initial")
	s := New(buf)

	for i := range 10 {
		s.Write([]byte(fmt.Sprintf("-%d", i)))
		runtime.GC()
	}
	expected := "initial-0-1-2-3-4-5-6-7-8-9"
	actual := string(append([]byte(nil), []byte(s.String())...)) // Copy bytes from the string, not just a clone of the pointer

	if expected != actual {
		t.Fatalf("Final string did not match expected: actual '%s' != expected '%s'", actual, expected)
	}
}

func TestAppend(t *testing.T) {
	buf1 := []byte("secret1")
	buf2 := []byte("-secret2")

	s1 := New(buf1)
	s2 := New(buf2)

	assertEqual := func(sen *Sensitive, expected string) {
		actual := make([]byte, len(sen.buf))
		copy(actual, sen.buf)
		if expected != string(actual) {
			t.Fatalf("Buffer does not match expected: actual '%s' != expected '%s'", actual, expected)
		}
	}

	assertZero := func(buf []byte) {
		for i := range len(buf) {
			if buf[i] != 0 {
				t.Fatalf("Byte buffer not zeroed ('%s')", string(buf))
			}
		}
	}

	assertEqual(s1, "secret1")
	assertEqual(s2, "-secret2")

	s1.Append(s2)
	assertZero(buf1) // Expect zeroed after expansion
	assertEqual(s1, "secret1-secret2")
	assertEqual(s2, "-secret2")

	s2.Clear()
	assertZero(buf2)
	assertEqual(s1, "secret1-secret2") // Do not expect s1 to be affected by s2 memory being zeroed
}

func TestGCRedaction(t *testing.T) {
	generate := func(expand bool) ([]byte, []byte, string) {
		// The Sensitive struct falls out of scope at the end of generate, expect that all returned buffers should be zeroed after garbage collection occurs
		originalBuf := []byte("initial")
		s := New(originalBuf)
		if expand {
			s.Write([]byte("+expanded"))
		}
		asString := s.String()
		return originalBuf, s.meta.buf, asString
	}

	for _, expand := range []bool{false, true} {
		t.Run(fmt.Sprintf("expand=%v", expand), func(t *testing.T) {
			originalBuf, asByte, asString := generate(expand)

			if expand && &originalBuf != &asByte {
				for i := range len(originalBuf) {
					if originalBuf[i] != 0 {
						t.Fatalf("Byte buffer not zeroed move ('%s')", string(originalBuf))
					}
				}
			}

			t.Logf("Before GC: []byte='%s' (%x); string='%s' (%x)", string(asByte), asByte, asString, asString)
			runtime.GC()

			stringCopy := string(append([]byte(nil), []byte(asString)...)) // Copy bytes from the string, not just a clone of the pointer
			byteCopy := make([]byte, len(asByte))
			t.Logf("After GC: []byte='%s' (%x); string='%s' (%x)", string(byteCopy), byteCopy, stringCopy, stringCopy)

			for i := range len(byteCopy) {
				if byteCopy[i] != 0 {
					t.Fatalf("Byte buffer not zeroed after GC ('%s')", string(byteCopy))
				}
			}
			for i := range len(stringCopy) {
				if stringCopy[i] != 0 {
					t.Fatalf("String not zeroed after GC ('%s')", stringCopy)
				}
			}
		})
	}
}

func TestClear(t *testing.T) {
	buf := []byte("initial")
	s := New(buf)
	asString := s.String()

	expectString := "initial"

	if asString != expectString {
		t.Fatalf("Expected string to be unchanged before Clear() called (actual '%s' != expected '%s')", asString, expectString)
	}

	t.Logf("Before Clear: []byte='%s' (%x); string='%s' (%x)", string(buf), buf, asString, asString)
	s.Clear()

	stringCopy := string(append([]byte(nil), []byte(asString)...)) // Copy bytes from the string, not just a clone of the pointer
	byteCopy := make([]byte, len(buf))
	t.Logf("After Clear: []byte='%s' (%x); string='%s' (%x)", string(byteCopy), byteCopy, stringCopy, stringCopy)

	for i := range len(byteCopy) {
		if byteCopy[i] != 0 {
			t.Fatalf("Byte buffer not zeroed after GC ('%s')", string(byteCopy))
		}
	}
	for i := range len(stringCopy) {
		if stringCopy[i] != 0 {
			t.Fatalf("String not zeroed after GC ('%s')", stringCopy)
		}
	}
}
