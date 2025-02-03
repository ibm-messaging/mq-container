package ibmmq

/*
  Copyright (c) IBM Corporation 2024

  Licensed under the Apache License, Version 2.0 (the "License");
  you may not use this file except in compliance with the License.
  You may obtain a copy of the License at

  http://www.apache.org/licenses/LICENSE-2.0

  Unless required by applicable law or agreed to in writing, software
  distributed under the License is distributed on an "AS IS" BASIS,
  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
  See the License for the specific language governing permissions and
  limitations under the License.

   Contributors:
     Mark Taylor - Initial Contribution
*/

/*
#include <stdlib.h>
#include <cmqc.h>
#include <cmqcfc.h>
*/
import "C"

import (
	"bytes"
	"encoding/binary"
)

type MQRFH2 struct {
	StrucLength    int32
	Encoding       int32
	CodedCharSetId int32
	Format         string
	Flags          int32
	NameValueCCSID int32
}

// This file manipulates MQRFH2 structures. Most of the time, I would
// expect people to use message properties. Those are generally much easier
// to work with. But there might be times when you want to go to the raw level.
//
// Functions are included to create the header, to set the body name/value strings,
// and then to extract the header and strings.
func NewMQRFH2(md *MQMD) *MQRFH2 {

	rfh2 := new(MQRFH2)

	rfh2.CodedCharSetId = MQCCSI_INHERIT

	rfh2.Format = ""
	rfh2.Flags = MQRFH_NONE

	rfh2.StrucLength = int32(MQRFH_STRUC_LENGTH_FIXED_2)
	rfh2.NameValueCCSID = 1208
	if md != nil {

		rfh2.Encoding = md.Encoding
		if md.CodedCharSetId == MQCCSI_DEFAULT {
			rfh2.CodedCharSetId = MQCCSI_INHERIT
		} else {
			rfh2.CodedCharSetId = md.CodedCharSetId
		}
		rfh2.Format = md.Format

		md.Format = MQFMT_RF_HEADER_2
		md.CodedCharSetId = MQCCSI_Q_MGR
	}

	if (C.MQENC_NATIVE % 2) == 0 {
		endian = binary.LittleEndian
	} else {
		endian = binary.BigEndian
	}

	return rfh2
}

// Return a byte array based on the contents of the RFH2 header
// This builds one of the pieces needed for the complete element when you are
// PUTting a message.  An application will not need to call this, but should
// instead use the Get() function to return the full byte array - header and strings
func (rfh2 *MQRFH2) bytes() []byte {
	buf := make([]byte, MQRFH_STRUC_LENGTH_FIXED_2)
	offset := 0

	copy(buf[offset:], "RFH ")
	offset += 4
	endian.PutUint32(buf[offset:], uint32(MQRFH_VERSION_2))
	offset += 4
	endian.PutUint32(buf[offset:], uint32(rfh2.StrucLength))
	offset += 4

	endian.PutUint32(buf[offset:], uint32(rfh2.Encoding))
	offset += 4
	endian.PutUint32(buf[offset:], uint32(rfh2.CodedCharSetId))
	offset += 4

	// Make sure the format is space padded to the correct length
	copy(buf[offset:], (rfh2.Format + space8)[0:8])
	offset += int(MQ_FORMAT_LENGTH)

	endian.PutUint32(buf[offset:], uint32(rfh2.Flags))
	offset += 4
	endian.PutUint32(buf[offset:], uint32(rfh2.NameValueCCSID))
	offset += 4

	return buf
}

/*
We have a byte array for the message contents. The start of that buffer
is the MQRFH2 structure. We read the bytes from that fixed header to match
the C structure definition for each field. We will assume use of RFH v2

This function is not called directly by applications. They can use the GetHeader
function instead, which covers both RFH2 and DLH structures.
*/
func getHeaderRFH2(md *MQMD, buf []byte) (*MQRFH2, int, error) {

	var version int32

	rfh2 := NewMQRFH2(nil)

	r := bytes.NewBuffer(buf)
	_ = readStringFromFixedBuffer(r, 4) // StrucId
	binary.Read(r, endian, &version)
	binary.Read(r, endian, &rfh2.StrucLength)

	binary.Read(r, endian, &rfh2.Encoding)
	binary.Read(r, endian, &rfh2.CodedCharSetId)

	rfh2.Format = readStringFromFixedBuffer(r, MQ_FORMAT_LENGTH)
	binary.Read(r, endian, &rfh2.Flags)
	binary.Read(r, endian, &rfh2.NameValueCCSID)

	return rfh2, int(rfh2.StrucLength), nil
}

// Split the name/value strings in the RFH2 into a string array
// In the message body , each consists of a length/string duple, so we read the length
// and then the string. And repeat until the buffer is exhausted.
// Each returned string will be a complete XML-like element which requires
// further parsing to extract individual properties.
func (hdr *MQRFH2) Get(buf []byte) []string {
	var l int32
	props := make([]string, 0)
	r := bytes.NewBuffer(buf[MQRFH_STRUC_LENGTH_FIXED_2:])

	propsLen := r.Len() // binary.Read modifies the buffer length so get it at the start of the loop
	for offset := 0; offset < propsLen; {
		binary.Read(r, endian, &l)
		offset += 4
		s := readStringFromFixedBuffer(r, l)
		props = append(props, s)
		offset += int(l)
	}
	return props
}

// Add a set of name/value strings to the RFH2. Return
// the byte array of the combined header and values, and modify
// the input RFH2 structure to have the correct length
func (hdr *MQRFH2) Set(p []string) []byte {
	var b []byte

	for i := 0; i < len(p); i++ {
		s := p[i]
		l := roundTo4(int32(len(s)))
		s = (s + space4)[0:l] // Pad with spaces to rounded length

		w := new(bytes.Buffer)
		binary.Write(w, endian, l)

		b = append(b, w.Bytes()...)
		b = append(b, s...)

	}

	// Now we know the length of the combined name/value strings, add it
	// to the header structure and make that the first part of the bytes response
	hdr.StrucLength = int32(len(b)) + MQRFH_STRUC_LENGTH_FIXED_2
	b = append(hdr.bytes(), b...)
	return b
}
