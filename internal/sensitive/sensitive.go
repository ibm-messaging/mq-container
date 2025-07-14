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
	"runtime"
	"unsafe"
)

// Sensitive ensures that memory used for sensitive strings/byte buffers is cleared (overwritten with zeroes) after garbage collection occurs
type Sensitive struct {
	*meta
}

type meta struct {
	buf []byte
	pin *runtime.Pinner
}

// New creates a new Sensitive object
//
// NOTE: Ownership of the buffer should transfer to the Sensitive object and the buffer passed should not continue to be used by the caller
func New(buf []byte) *Sensitive {
	s := &Sensitive{}
	s.setMeta(buf)
	return s
}

// Write appends the byte slice to the underlying buffer.
// If the buffer needs to grow and is moved, the original location will be zeroed as part of the Write call.
func (s *Sensitive) Write(b []byte) error {
	newBuf := append(s.buf, b...)
	if &newBuf[0] != &s.buf[0] {
		// buffer starts at a new address - must have moved to a new memory block
		zeroMeta(s.meta)
		s.setMeta(newBuf)
		return nil
	}
	// buffer hasn't moved, don't zero out current buffer as it's still in use, but do update the buffer to capture new length etc.
	s.buf = newBuf
	return nil
}

func (s Sensitive) Len() int {
	return len(s.buf)
}

func (s *Sensitive) Append(other *Sensitive) error {
	return s.Write(other.buf)
}

// Clear triggers an immediate clear of the underlying buffer without waiting for garbage collection to occur
func (s *Sensitive) Clear() {
	zeroMeta(s.meta)
}

// String returns a string that points to the underlying managed buffer
//
// NOTE: if the Sensitive object is garbage collected, or Clear() is called, this string will be zeroed even if it remains in scope
func (s *Sensitive) String() string {
	// #nosec G103 - unsafe package is required in order to prevent memory copy during type conversion to string
	return unsafe.String(unsafe.SliceData(s.meta.buf), len(s.buf))
}

// setMeta creates a new metadata object, pinning its location in memory and registering finalizers to zero the underlying buffer on garbage collection
func (s *Sensitive) setMeta(buf []byte) {
	m := &meta{
		buf: buf,
		pin: &runtime.Pinner{},
	}
	m.pin.Pin(&buf)
	s.meta = m
	runtime.SetFinalizer(m, zeroMeta)
}

// zeroMeta zeroes the underlying buffer and removes all finalizers and memory pins
func zeroMeta(m *meta) {
	for i := range len(m.buf) {
		m.buf[i] = 0
	}
	runtime.KeepAlive(m.buf)
	m.pin.Unpin()
	runtime.SetFinalizer(m, nil)
}
