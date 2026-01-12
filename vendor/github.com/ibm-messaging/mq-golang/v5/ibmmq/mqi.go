/*
Package ibmmq provides a wrapper to the IBM MQ procedural interface (the MQI).

The verbs are given mixed case names without MQ - Open instead
of MQOPEN etc.

For more information on the MQI, including detailed descriptions of the functions,
constants and structures, see the MQ Documentation
at https://www.ibm.com/docs/en/ibm-mq/latest?topic=reference-developing-applications

If an MQI call returns MQCC_FAILED or MQCC_WARNING, a custom error
type is returned containing the MQCC/MQRC values as well as
a formatted string. Use 'mqreturn:= err(*ibmmq.MQReturn)' to access
the particular MQRC or MQCC values.

The build directives assume the default MQ installation path
which is in /opt/mqm (Linux) and c:\Program Files\IBM\MQ (Windows).
If you use a non-default path for the installation, you can set
environment variables CGO_CFLAGS and CGO_LDFLAGS to reference those
directories.
*/
package ibmmq

/*
  Copyright (c) IBM Corporation 2016, 2024

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
#cgo !windows,!aix CFLAGS: -I/opt/mqm/inc -D_REENTRANT
#cgo  aix          CFLAGS: -I/usr/mqm/inc -D_REENTRANT
#cgo  windows      CFLAGS:  -I"C:/Program Files/IBM/MQ/Tools/c/include" -D_WIN64
#cgo !windows,!aix,!darwin LDFLAGS: -L/opt/mqm/lib64 -lmqm_r -Wl,-rpath,/opt/mqm/lib64 -Wl,-rpath,/usr/lib64
#cgo darwin                LDFLAGS: -L/opt/mqm/lib64 -lmqm_r -Wl,-rpath,/opt/mqm/lib64 -Wl,-rpath,/usr/lib64
#cgo aix                   LDFLAGS: -L/usr/mqm/lib64 -lmqm_r
#cgo windows               LDFLAGS: -L "C:/Program Files/IBM/MQ/bin64" -lmqm

#include <stdlib.h>
#include <string.h>
#include <cmqc.h>
#include <cmqxc.h>

// Compatibility with older versions - -1 will never be a match
#if defined(MQCA_INITIAL_KEY)
#define GOCA_INITIAL_KEY MQCA_INITIAL_KEY
#else
#define GOCA_INITIAL_KEY (-1)
#endif

*/
import "C"

import (
	"encoding/binary"
	_ "fmt"
	"io"
	"os"
	"strings"
	"time"
	"unsafe"
)

/*
   This file contains the C wrappers, calling out to structure-specific
   functions where necessary.

   Define some basic types to hold the
   references to MQ objects - hconn, hobj - and
   a simple way to pass the combination of MQCC/MQRC
   returned from MQI verbs

   The object name is copied into the structures only
   for convenience. It's not really needed, but
   it can sometimes be nice to print which queue an hObj
   refers to during debug.
*/

/*
MQQueueManager contains the connection to the queue manager
*/
type MQQueueManager struct {
	hConn C.MQHCONN
	Name  string
}

/*
MQObject contains a reference to an open object and the associated
queue manager
*/
type MQObject struct {
	hObj C.MQHOBJ
	qMgr *MQQueueManager
	Name string
}

/*
 * MQMessageHandle is a wrapper for the C message handle
 * type. Unlike the C MQI, a valid hConn is required to create
 * the message handle.
 */
type MQMessageHandle struct {
	hMsg C.MQHMSG
	qMgr *MQQueueManager
}

/*
MQReturn holds the MQRC and MQCC values returned from an MQI verb. It
implements the Error() function so is returned as the specific error
from the verbs. See the sample programs for how to access the
MQRC/MQCC values in this returned error.
*/
type MQReturn struct {
	MQCC int32
	MQRC int32
	verb string
}

func (e *MQReturn) Error() string {
	return mqstrerror(e.verb, C.MQLONG(e.MQCC), C.MQLONG(e.MQRC))
}

func IsUsableHObj(o MQObject) bool {
	rc := false
	if o.qMgr == nil {
		rc = false
	} else if o.hObj != C.MQHO_UNUSABLE_HOBJ {
		rc = true
	} else {
		rc = false
	}
	// logTrace("IsUsableHObj hObj:%v rc:%v", o, rc)
	return rc
}

func IsUsableHandle(mh MQMessageHandle) bool {
	rc := false
	if mh.qMgr == nil {
		rc = false
	} else if mh.hMsg != C.MQHM_NONE && mh.hMsg != C.MQHM_UNUSABLE_HMSG {
		rc = true
	}
	return rc
}

// There may be times when we want to inspect the actual value of some of the
// types
func (handle *MQMessageHandle) GetValue() int64 {
	return int64(handle.hMsg)
}

func (x *MQQueueManager) GetValue() int32 {
	return int32(x.hConn)
}

func (hObj *MQObject) GetValue() int32 {
	return int32(hObj.hObj)
}

func (hObj *MQObject) GetHConn() *MQQueueManager {
	return hObj.qMgr
}

var endian binary.ByteOrder // Used by structure formatters such as MQCFH
const space4 = "    "
const space8 = "        "
const (
	mqDateTimeFormat = "20060102150405.00 MST" // Used as the way to parse a string into a time.Time type with magic values
	mqDateFormat     = "20060102"
	mqTimeFormat     = "150405.00"
)

// This function is executed once before any other code in the package
func init() {
	if C.MQENC_NATIVE%2 == 0 {
		endian = binary.LittleEndian
	} else {
		endian = binary.BigEndian
	}

	if os.Getenv("MQIGO_TRACE") != "" {
		SetTrace(true)
	}
}

/*
 * Copy a Go string in "strings"
 * to a fixed-size C char array such as MQCHAR12
 * Once the string has been copied, it can be immediately freed
 * Empty strings have first char set to 0 in MQI structures
 */
func setMQIString(a *C.char, v string, l int) {
	if len(v) > 0 {
		p := C.CString(v)
		C.strncpy(a, p, (C.size_t)(l))
		C.free(unsafe.Pointer(p))
	} else {
		*a = 0
	}
}

/*
 * The C.GoStringN function can return strings that include
 * NUL characters (which is not really what is expected for a C string-related
 * function). So we have a utility function to remove any trailing nulls and spaces
 */
func trimStringN(c *C.char, l C.int) string {
	var rc string
	s := C.GoStringN(c, l)
	i := strings.IndexByte(s, 0)
	if i == -1 {
		rc = s
	} else {
		rc = s[0:i]
	}
	return strings.TrimSpace(rc)
}

/*
Conn is the function to connect to a queue manager
*/
func Conn(goQMgrName string) (MQQueueManager, error) {
	traceEntry("Conn")
	qm, err := Connx(goQMgrName, nil)
	traceExitErr("Conn", 0, err)
	return qm, err
}

/*
Connx is the extended function to connect to a queue manager.
*/
func Connx(goQMgrName string, gocno *MQCNO) (MQQueueManager, error) {
	var mqrc C.MQLONG
	var mqcc C.MQLONG
	var mqcno C.MQCNO

	traceEntry("Connx")

	// MQ normally sets signal handlers that turn out to
	// get in the way of Go programs. In particular SEGV.
	// Setting this environment variable should make it easier
	// to get stack traces out of Go programs in the event of
	// errors. For this particular variable, any value will make it
	// effective.
	os.Setenv("MQS_NO_SYNC_SIGNAL_HANDLING", "true")

	qMgr := MQQueueManager{}
	qMgr.Name = goQMgrName
	mqQMgrName := unsafe.Pointer(C.CString(goQMgrName))
	defer C.free(mqQMgrName)

	logTrace("QMgrName: %s", goQMgrName)

	// Set up a default CNO if not provided.
	if gocno == nil {
		// Because Go programs are always threaded, and we cannot
		// tell on which thread we might get dispatched, allow handles always to
		// be shareable.
		gocno = NewMQCNO()
		gocno.Options = MQCNO_HANDLE_SHARE_NO_BLOCK
	} else {
		if (gocno.Options & (MQCNO_HANDLE_SHARE_NO_BLOCK |
			MQCNO_HANDLE_SHARE_BLOCK)) == 0 {
			gocno.Options |= MQCNO_HANDLE_SHARE_NO_BLOCK
		}
	}
	copyCNOtoC(&mqcno, gocno)

	C.MQCONNX((*C.MQCHAR)(mqQMgrName), &mqcno, &qMgr.hConn, &mqcc, &mqrc)

	if gocno != nil {
		copyCNOfromC(&mqcno, gocno)
	}

	mqreturn := &MQReturn{MQCC: int32(mqcc),
		MQRC: int32(mqrc),
		verb: "MQCONNX",
	}

	if mqcc != C.MQCC_OK {
		traceExitErr("Connx", 1, mqreturn)
		return qMgr, mqreturn
	}

	traceExit("Connx")
	return qMgr, nil
}

/*
Disc is the function to disconnect from the queue manager
*/
func (x *MQQueueManager) Disc() error {
	var mqrc C.MQLONG
	var mqcc C.MQLONG

	traceEntry("Disc")

	// Cleanup any allocated message Handles. If the
	// MQDISC fails (unusual), that's still OK because we would reallocate
	// the handles on any subsequent use.
	f := otelFuncs.Disc
	if f != nil {
		f(x)
	}

	savedConn := x.hConn
	C.MQDISC(&x.hConn, &mqcc, &mqrc)
	mqreturn := MQReturn{MQCC: int32(mqcc),
		MQRC: int32(mqrc),
		verb: "MQDISC",
	}

	if int32(mqrc) != C.MQRC_HCONN_ERROR {
		cbRemoveConnection(savedConn)
	}

	if mqcc != C.MQCC_OK {
		traceExitErr("Disc", 1, &mqreturn)
		return &mqreturn
	}

	traceExit("Disc")
	return nil
}

/*
Open an object such as a queue or topic
*/
func (x *MQQueueManager) Open(good *MQOD, goOpenOptions int32) (MQObject, error) {
	var mqrc C.MQLONG
	var mqcc C.MQLONG
	var mqod C.MQOD
	var mqOpenOptions C.MQLONG

	traceEntry("Open")

	object := MQObject{
		Name: good.ObjectName,
		qMgr: x,
	}

	logTrace("Object: %s %s", good.ObjectName, good.ObjectQMgrName)

	copyODtoC(&mqod, good)
	mqOpenOptions = C.MQLONG(goOpenOptions) | C.MQOO_FAIL_IF_QUIESCING

	C.MQOPEN(x.hConn,
		(C.PMQVOID)(unsafe.Pointer(&mqod)),
		mqOpenOptions,
		&object.hObj,
		&mqcc,
		&mqrc)

	copyODfromC(&mqod, good)

	mqreturn := MQReturn{MQCC: int32(mqcc),
		MQRC: int32(mqrc),
		verb: "MQOPEN",
	}

	if mqcc != C.MQCC_OK {
		traceExitErr("Open", 1, &mqreturn)
		return object, &mqreturn
	}

	f := otelFuncs.Open
	if f != nil {
		f(&object, good, goOpenOptions)
	}

	// ObjectName may have changed because it's a model queue
	object.Name = good.ObjectName
	if good.ObjectType == C.MQOT_TOPIC {
		object.Name = good.ObjectString
	}

	traceExit("Open")
	return object, nil

}

/*
Close the object
*/
func (object *MQObject) Close(goCloseOptions int32) error {
	var mqrc C.MQLONG
	var mqcc C.MQLONG
	var mqCloseOptions C.MQLONG

	traceEntry("Close")

	mqCloseOptions = C.MQLONG(goCloseOptions)

	if !IsUsableHObj(*object) {
		err := &MQReturn{MQCC: MQCC_FAILED,
			MQRC: MQRC_HOBJ_ERROR,
			verb: "MQCLOSE",
		}
		traceExitErr("Close", 2, err)
		return err
	}

	savedHConn := object.qMgr.hConn
	savedHObj := object.hObj

	C.MQCLOSE(object.qMgr.hConn, &object.hObj, mqCloseOptions, &mqcc, &mqrc)

	mqreturn := MQReturn{MQCC: int32(mqcc),
		MQRC: int32(mqrc),
		verb: "MQCLOSE",
	}

	if mqcc != C.MQCC_OK {
		traceExitErr("Close", 1, &mqreturn)
		return &mqreturn
	}

	f := otelFuncs.Close
	if f != nil {
		f(object)
	}
	cbRemoveHandle(savedHConn, savedHObj)
	traceExit("Close")
	return nil

}

/*
Sub is the function to subscribe to a topic
*/
func (x *MQQueueManager) Sub(gosd *MQSD, qObject *MQObject) (MQObject, error) {
	var mqrc C.MQLONG
	var mqcc C.MQLONG
	var mqsd C.MQSD

	traceEntry("Sub")

	subObject := MQObject{
		Name: gosd.ObjectName + "[" + gosd.ObjectString + "]",
		qMgr: x,
	}

	logTrace("Object: %s", subObject.Name)

	err := checkSD(gosd, "MQSUB")
	if err != nil {
		return subObject, err
	}

	copySDtoC(&mqsd, gosd)

	C.MQSUB(x.hConn,
		(C.PMQVOID)(unsafe.Pointer(&mqsd)),
		&qObject.hObj,
		&subObject.hObj,
		&mqcc,
		&mqrc)

	copySDfromC(&mqsd, gosd)

	mqreturn := MQReturn{MQCC: int32(mqcc),
		MQRC: int32(mqrc),
		verb: "MQSUB",
	}

	if mqcc != C.MQCC_OK {
		traceExitErr("Sub", 1, &mqreturn)
		return subObject, &mqreturn
	}

	qObject.qMgr = x // Force the correct hConn for managed objects

	traceExit("Sub")
	return subObject, nil

}

/*
Subrq is the function to request retained publications
*/
func (subObject *MQObject) Subrq(gosro *MQSRO, action int32) error {
	var mqrc C.MQLONG
	var mqcc C.MQLONG
	var mqsro C.MQSRO

	traceEntry("Subrq")

	if !IsUsableHObj(*subObject) {
		err := &MQReturn{MQCC: MQCC_FAILED,
			MQRC: MQRC_HOBJ_ERROR,
			verb: "MQSUBRQ",
		}
		traceExitErr("Subrq", 2, err)
		return err
	}

	copySROtoC(&mqsro, gosro)

	C.MQSUBRQ(subObject.qMgr.hConn,
		subObject.hObj,
		C.MQLONG(action),
		(C.PMQVOID)(unsafe.Pointer(&mqsro)),
		&mqcc,
		&mqrc)

	copySROfromC(&mqsro, gosro)

	mqreturn := MQReturn{MQCC: int32(mqcc),
		MQRC: int32(mqrc),
		verb: "MQSUBRQ",
	}

	if mqcc != C.MQCC_OK {
		traceExitErr("Subrq", 1, &mqreturn)
		return &mqreturn
	}

	traceExit("Subrq")
	return nil
}

/*
Begin is the function to start a two-phase XA transaction coordinated by MQ
*/
func (x *MQQueueManager) Begin(gobo *MQBO) error {
	var mqrc C.MQLONG
	var mqcc C.MQLONG
	var mqbo C.MQBO

	traceEntry("Begin")

	copyBOtoC(&mqbo, gobo)

	C.MQBEGIN(x.hConn, (C.PMQVOID)(unsafe.Pointer(&mqbo)), &mqcc, &mqrc)

	copyBOfromC(&mqbo, gobo)

	mqreturn := MQReturn{MQCC: int32(mqcc),
		MQRC: int32(mqrc),
		verb: "MQBEGIN",
	}

	if mqcc != C.MQCC_OK {
		traceExitErr("Beqin", 1, &mqreturn)
		return &mqreturn
	}

	traceExit("Begin")
	return nil

}

/*
Cmit is the function to commit an in-flight transaction
*/
func (x *MQQueueManager) Cmit() error {
	var mqrc C.MQLONG
	var mqcc C.MQLONG

	traceEntry("Cmit")

	C.MQCMIT(x.hConn, &mqcc, &mqrc)

	mqreturn := MQReturn{MQCC: int32(mqcc),
		MQRC: int32(mqrc),
		verb: "MQCMIT",
	}

	if mqcc != C.MQCC_OK {
		traceExitErr("Cmit", 1, &mqreturn)
		return &mqreturn
	}

	traceExit("Cmit")
	return nil

}

/*
Back is the function to backout an in-flight transaction
*/
func (x *MQQueueManager) Back() error {
	var mqrc C.MQLONG
	var mqcc C.MQLONG

	traceEntry("Back")

	C.MQBACK(x.hConn, &mqcc, &mqrc)

	mqreturn := MQReturn{MQCC: int32(mqcc),
		MQRC: int32(mqrc),
		verb: "MQBACK",
	}

	if mqcc != C.MQCC_OK {
		traceExitErr("Back", 1, &mqreturn)
		return &mqreturn
	}

	traceExit("Back")
	return nil

}

/*
Stat is the function to check the status after using the asynchronous put
across a client channel
*/
func (x *MQQueueManager) Stat(statusType int32, gosts *MQSTS) error {
	var mqrc C.MQLONG
	var mqcc C.MQLONG

	var mqsts C.MQSTS

	traceEntry("Stat")

	copySTStoC(&mqsts, gosts)

	C.MQSTAT(x.hConn, C.MQLONG(statusType), (C.PMQVOID)(unsafe.Pointer(&mqsts)), &mqcc, &mqrc)

	mqreturn := MQReturn{MQCC: int32(mqcc),
		MQRC: int32(mqrc),
		verb: "MQSTAT",
	}

	copySTSfromC(&mqsts, gosts)

	if mqcc != C.MQCC_OK {
		traceExitErr("Stat", 1, &mqreturn)
		return &mqreturn
	}

	traceExit("Stat")
	return nil

}

/*
Put a message to a queue or publish to a topic
*/
func (object MQObject) Put(gomd *MQMD,
	gopmo *MQPMO, buffer []byte) error {

	var mqrc C.MQLONG
	var mqcc C.MQLONG
	var mqmd C.MQMD
	var mqpmo C.MQPMO
	var ptr C.PMQVOID

	traceEntry("Put")

	if !IsUsableHObj(object) {
		err := &MQReturn{MQCC: MQCC_FAILED,
			MQRC: MQRC_HOBJ_ERROR,
			verb: "MQPUT",
		}
		traceExitErr("Put", 2, err)
		return err
	}

	err := checkMD(gomd, "MQPUT")
	if err != nil {
		traceExitErr("Put", 1, err)
		return err
	}

	bufflen := len(buffer)
	logTrace("BufferLength: %d", bufflen)

	opts := gopmo.OtelOpts
	f1 := otelFuncs.PutTraceBefore
	if f1 != nil {
		f1(opts, object.qMgr, gomd, gopmo, buffer)
	}

	copyMDtoC(&mqmd, gomd)
	copyPMOtoC(&mqpmo, gopmo)

	if bufflen > 0 {
		ptr = (C.PMQVOID)(unsafe.Pointer(&buffer[0]))
	} else {
		ptr = nil
	}

	C.MQPUT(object.qMgr.hConn, object.hObj, (C.PMQVOID)(unsafe.Pointer(&mqmd)),
		(C.PMQVOID)(unsafe.Pointer(&mqpmo)),
		(C.MQLONG)(bufflen),
		ptr,
		&mqcc, &mqrc)

	copyMDfromC(&mqmd, gomd)
	copyPMOfromC(&mqpmo, gopmo)

	f2 := otelFuncs.PutTraceAfter
	if f2 != nil {
		f2(opts, object.qMgr, gopmo)
	}

	mqreturn := MQReturn{MQCC: int32(mqcc),
		MQRC: int32(mqrc),
		verb: "MQPUT",
	}

	if mqcc != C.MQCC_OK {
		traceExitErr("Put", 2, &mqreturn)
		return &mqreturn
	}

	traceExit("Put")
	return nil
}

/*
Put1 puts a single messsage to a queue or topic. Typically used for one-shot
replies where it can be cheaper than multiple Open/Put/Close
sequences
*/
func (x *MQQueueManager) Put1(good *MQOD, gomd *MQMD,
	gopmo *MQPMO, buffer []byte) error {

	var mqrc C.MQLONG
	var mqcc C.MQLONG
	var mqmd C.MQMD
	var mqpmo C.MQPMO
	var mqod C.MQOD
	var ptr C.PMQVOID

	traceEntry("Put1")

	err := checkMD(gomd, "MQPUT1")
	if err != nil {
		traceExitErr("Put1", 1, err)
		return err
	}

	opts := gopmo.OtelOpts
	f1 := otelFuncs.PutTraceBefore
	if f1 != nil {
		f1(opts, x, gomd, gopmo, buffer)
	}

	copyODtoC(&mqod, good)
	copyMDtoC(&mqmd, gomd)
	copyPMOtoC(&mqpmo, gopmo)

	bufflen := len(buffer)
	logTrace("BufferLength: %d", bufflen)

	if bufflen > 0 {
		ptr = (C.PMQVOID)(unsafe.Pointer(&buffer[0]))
	} else {
		ptr = nil
	}

	C.MQPUT1(x.hConn, (C.PMQVOID)(unsafe.Pointer(&mqod)),
		(C.PMQVOID)(unsafe.Pointer(&mqmd)),
		(C.PMQVOID)(unsafe.Pointer(&mqpmo)),
		(C.MQLONG)(bufflen),
		ptr,
		&mqcc, &mqrc)

	copyODfromC(&mqod, good)
	copyMDfromC(&mqmd, gomd)
	copyPMOfromC(&mqpmo, gopmo)

	f2 := otelFuncs.PutTraceAfter
	if f2 != nil {
		f2(opts, x, gopmo)
	}

	mqreturn := MQReturn{MQCC: int32(mqcc),
		MQRC: int32(mqrc),
		verb: "MQPUT1",
	}

	if mqcc != C.MQCC_OK {
		traceExitErr("Put1", 2, &mqreturn)
		return &mqreturn
	}

	traceExit("Put1")
	return nil

}

/*
Get a message from a queue
The length of the retrieved message is returned.
*/
func (object MQObject) Get(gomd *MQMD,
	gogmo *MQGMO, buffer []byte) (int, error) {

	traceEntry("Get")
	datalen, removed, err := object.getInternal(gomd, gogmo, buffer, false)
	if removed > 0 {
		copy(buffer, buffer[removed:])
		datalen -= removed
	}
	traceExitErr("Get", 0, err)
	return datalen, err
}

/*
GetSlice is the same as Get except that the buffer gets returned
ready-sliced based on the message length instead of just returning the
length. The real length is also still returned in case of truncation.
*/
func (object MQObject) GetSlice(gomd *MQMD,
	gogmo *MQGMO, buffer []byte) ([]byte, int, error) {

	traceEntry("GetSlice")
	realDatalen, removed, err := object.getInternal(gomd, gogmo, buffer, true)

	// The datalen will be set even if the buffer is too small - there
	// will be one of MQRC_TRUNCATED_MSG_ACCEPTED or _FAILED depending on the
	// GMO options. In any case, we return the available data along with the
	// error code but need to make sure that the real untruncated
	// message length is also returned. Also ensure we don't try to read past the
	// end of the buffer. The realDatalen value still includes any removed RFH2 block
	// so you can tell how big a buffer you will need on any retry after truncation.
	datalen := realDatalen
	if datalen > cap(buffer) {
		datalen = cap(buffer)
	}

	traceExitErr("GetSlice", 0, err)
	return buffer[removed:datalen], realDatalen, err
}

/*
This is the real function that calls MQGET.
*/
func (object MQObject) getInternal(gomd *MQMD,
	gogmo *MQGMO, buffer []byte, useCap bool) (int, int, error) {

	var mqrc C.MQLONG
	var mqcc C.MQLONG
	var mqmd C.MQMD
	var mqgmo C.MQGMO
	var datalen C.MQLONG
	var ptr C.PMQVOID

	removed := 0
	traceEntry("getInternal")

	err := checkMD(gomd, "MQGET")
	if err != nil {
		traceExitErr("getInternal", 1, err)
		return 0, removed, err
	}
	err = checkGMO(gogmo, "MQGET")
	if err != nil {
		traceExitErr("getInternal", 2, err)
		return 0, removed, err
	}

	if !IsUsableHObj(object) {
		err = &MQReturn{MQCC: MQCC_FAILED,
			MQRC: MQRC_HOBJ_ERROR,
			verb: "MQGET",
		}
		traceExitErr("getInternal", 3, err)
		return 0, removed, err
	}

	bufflen := 0
	if useCap {
		bufflen = cap(buffer)
		logTrace("BufferCapacity: %d", bufflen)
	} else {
		bufflen = len(buffer)
		logTrace("BufferLength: %d", bufflen)
	}

	opts := gogmo.OtelOpts
	f1 := otelFuncs.GetTraceBefore
	if f1 != nil {
		f1(opts, object.qMgr, &object, gogmo, false)
	}
	copyMDtoC(&mqmd, gomd)
	copyGMOtoC(&mqgmo, gogmo)

	if bufflen > 0 {
		// There has to be something in the buffer for CGO to be able to
		// find its address. We know there's space backing the buffer so just
		// set the first byte to something.
		if useCap && len(buffer) == 0 {
			buffer = append(buffer, 0)
		}
		ptr = (C.PMQVOID)(unsafe.Pointer(&buffer[0]))
	} else {
		ptr = nil
	}

	C.MQGET(object.qMgr.hConn, object.hObj, (C.PMQVOID)(unsafe.Pointer(&mqmd)),
		(C.PMQVOID)(unsafe.Pointer(&mqgmo)),
		(C.MQLONG)(bufflen),
		ptr,
		&datalen,
		&mqcc, &mqrc)

	godatalen := int(datalen)
	logTrace("Returned datalen: %d", godatalen)

	copyMDfromC(&mqmd, gomd)
	copyGMOfromC(&mqgmo, gogmo)

	mqreturn := MQReturn{MQCC: int32(mqcc),
		MQRC: int32(mqrc),
		verb: "MQGET",
	}

	// Only process OTEL tracing if we actually got a message
	if mqcc == C.MQCC_OK || mqrc == C.MQRC_TRUNCATED_MSG_ACCEPTED {
		f2 := otelFuncs.GetTraceAfter
		if f2 != nil {
			removed = f2(opts, &object, gogmo, gomd, buffer[0:datalen], false)
			logTrace("Removed: %d Datalen: %d BufLen: %d %+v", removed, datalen, bufflen, buffer)
		}
	}

	if mqcc != C.MQCC_OK {
		traceExitErr("getInternal", 3, &mqreturn)
		return godatalen, removed, &mqreturn
	}

	traceExit("getInternal")
	return godatalen, removed, nil

}

/*
Inq is the function to inquire on an attribute of an object

This has a much simpler API than the original implementation.
Simply pass in the list of selectors for the object
and the return value consists of a map whose elements are
a) accessed via the selector
b) varying datatype (integer, string, string array) based on the selector
*/
func (object MQObject) Inq(goSelectors []int32) (map[int32]interface{}, error) {
	var mqrc C.MQLONG
	var mqcc C.MQLONG
	var mqCharAttrs C.PMQCHAR
	var goCharAttrs []byte
	var goIntAttrs []int32
	var ptr C.PMQLONG
	var charOffset int
	var charLength int

	traceEntry("Inq")

	if !IsUsableHObj(object) {
		err := &MQReturn{MQCC: MQCC_FAILED,
			MQRC: MQRC_HOBJ_ERROR,
			verb: "MQINQ",
		}
		traceExitErr("Inq", 2, err)
		return nil, err
	}

	intAttrCount, _, charAttrLen := getAttrInfo(goSelectors)

	if intAttrCount > 0 {
		goIntAttrs = make([]int32, intAttrCount)
		ptr = (C.PMQLONG)(unsafe.Pointer(&goIntAttrs[0]))
	} else {
		ptr = nil
	}
	if charAttrLen > 0 {
		mqCharAttrs = (C.PMQCHAR)(C.malloc(C.size_t(charAttrLen)))
		defer C.free(unsafe.Pointer(mqCharAttrs))
	} else {
		mqCharAttrs = nil
	}

	// Pass in the selectors
	C.MQINQ(object.qMgr.hConn, object.hObj,
		C.MQLONG(len(goSelectors)),
		C.PMQLONG(unsafe.Pointer(&goSelectors[0])),
		C.MQLONG(intAttrCount),
		ptr,
		C.MQLONG(charAttrLen),
		mqCharAttrs,
		&mqcc, &mqrc)

	mqreturn := MQReturn{MQCC: int32(mqcc),
		MQRC: int32(mqrc),
		verb: "MQINQ",
	}

	if mqcc != C.MQCC_OK {
		traceExitErr("Inq", 1, &mqreturn)
		return nil, &mqreturn
	}

	// Create a map of the selectors to the returned values
	returnedMap := make(map[int32]interface{})

	// Get access to the returned character data
	if charAttrLen > 0 {
		goCharAttrs = C.GoBytes(unsafe.Pointer(mqCharAttrs), C.int(charAttrLen))
	}

	// Walk through the returned data to build a map of responses. Go through
	// the integers first to ensure that the map includes MQIA_NAME_COUNT if that
	// had been requested
	intAttr := 0
	for i := 0; i < len(goSelectors); i++ {
		s := goSelectors[i]
		if s >= C.MQIA_FIRST && s <= C.MQIA_LAST {
			returnedMap[s] = goIntAttrs[intAttr]
			intAttr++
		}
	}

	// Now we can walk through the list again for the character attributes
	// and build the map entries. Getting the list of NAMES from a NAMELIST
	// is a bit complicated ...
	charLength = 0
	charOffset = 0
	for i := 0; i < len(goSelectors); i++ {
		s := goSelectors[i]
		if s >= C.MQCA_FIRST && s <= C.MQCA_LAST {
			if s == C.MQCA_NAMES {
				count, ok := returnedMap[C.MQIA_NAME_COUNT]
				if ok {
					c := int(count.(int32))
					charLength = C.MQ_OBJECT_NAME_LENGTH
					names := make([]string, c)
					for j := 0; j < c; j++ {
						name := string(goCharAttrs[charOffset : charOffset+charLength])
						idx := strings.IndexByte(name, 0)
						if idx != -1 {
							name = name[0:idx]
						}
						names[j] = strings.TrimSpace(name)
						charOffset += charLength
					}
					returnedMap[s] = names
				} else {
					charLength = 0
				}
			} else {
				charLength = getAttrLength(s)
				name := string(goCharAttrs[charOffset : charOffset+charLength])
				idx := strings.IndexByte(name, 0)
				if idx != -1 {
					name = name[0:idx]
				}

				if s == C.GOCA_INITIAL_KEY && strings.TrimSpace(name) != "" {
					// The actual return is something unprintable so set it to the same thing as a PCF command response
					returnedMap[s] = "********"
				} else {
					returnedMap[s] = strings.TrimSpace(name)
				}
				charOffset += charLength
			}
		}
	}

	traceExit("Inq")
	return returnedMap, nil
}

/*
The InqMap function was the migration path when the original Inq was
deprecated. It is kept here as a temporary wrapper to the new Inq() version.
*/
func (object MQObject) InqMap(goSelectors []int32) (map[int32]interface{}, error) {
	return object.Inq(goSelectors)
}

/*
Set is the function that wraps MQSET. The single parameter is a map whose
elements contain an MQIA/MQCA selector with either a string or an int32 for
the value.
*/
func (object MQObject) Set(goSelectors map[int32]interface{}) error {
	var mqrc C.MQLONG
	var mqcc C.MQLONG

	var charAttrs []byte
	var charAttrsPtr C.PMQCHAR
	var intAttrs []int32
	var intAttrsPtr C.PMQLONG
	var charOffset int
	var charLength int

	traceEntry("Set")

	if !IsUsableHObj(object) {
		err := &MQReturn{MQCC: MQCC_FAILED,
			MQRC: MQRC_HOBJ_ERROR,
			verb: "MQSET",
		}
		traceExitErr("Set", 2, err)
		return err
	}

	// Pass through the map twice. First time lets us
	// create an array of selector names from map keys which is then
	// used to calculate the character buffer that's needed
	selectors := make([]int32, len(goSelectors))
	i := 0
	for k, _ := range goSelectors {
		selectors[i] = k
		i++
	}

	intAttrCount, _, charAttrLen := getAttrInfo(selectors)

	// Create the areas to be used for the separate char and int values
	if intAttrCount > 0 {
		intAttrs = make([]int32, intAttrCount)
		intAttrsPtr = (C.PMQLONG)(unsafe.Pointer(&intAttrs[0]))
	} else {
		intAttrsPtr = nil
	}

	if charAttrLen > 0 {
		charAttrs = make([]byte, charAttrLen)
		charAttrsPtr = (C.PMQCHAR)(unsafe.Pointer(&charAttrs[0]))
	} else {
		charAttrsPtr = nil
	}

	// Walk through the map a second time
	charOffset = 0
	intAttr := 0
	for i := 0; i < len(selectors); i++ {
		s := selectors[i]
		if s >= C.MQCA_FIRST && s <= C.MQCA_LAST {
			// The character processing is a bit OTT since there is in reality
			// only a single attribute that can ever be SET. But a general purpose
			// function looks more like the MQINQ operation
			v := goSelectors[s].(string)
			charLength = getAttrLength(s)
			vBytes := []byte(v)
			b := byte(0)
			for j := 0; j < charLength; j++ {
				if j < len(vBytes) {
					b = vBytes[j]
				} else {
					b = 0
				}
				charAttrs[charOffset+j] = b
			}
			charOffset += charLength
		} else if s >= C.MQIA_FIRST && s <= C.MQIA_LAST {
			vv := int32(0)
			v := goSelectors[s]
			// Force the returned value from the map to be int32 because we
			// can't check it at compile time.
			if _, ok := v.(int32); ok {
				vv = v.(int32)
			} else if _, ok := v.(int); ok {
				vv = int32(v.(int))
			}
			intAttrs[intAttr] = vv
			intAttr++
		}
	}

	// Pass in the selectors
	C.MQSET(object.qMgr.hConn, object.hObj,
		C.MQLONG(len(selectors)),
		C.PMQLONG(unsafe.Pointer(&selectors[0])),
		C.MQLONG(intAttrCount),
		intAttrsPtr,
		C.MQLONG(charAttrLen),
		charAttrsPtr,
		&mqcc, &mqrc)

	mqreturn := MQReturn{MQCC: int32(mqcc),
		MQRC: int32(mqrc),
		verb: "MQSET",
	}

	if mqcc != C.MQCC_OK {
		traceExitErr("Set", 1, &mqreturn)
		return &mqreturn
	}

	traceExit("Set")
	return nil
}

/*********** Message Handles and Properties  ****************/

/*
CrtMH is the function to create a message handle for holding properties
*/
func (x *MQQueueManager) CrtMH(gocmho *MQCMHO) (MQMessageHandle, error) {
	var mqrc C.MQLONG
	var mqcc C.MQLONG

	var mqcmho C.MQCMHO
	var mqhmsg C.MQHMSG

	traceEntry("CrtMH")

	copyCMHOtoC(&mqcmho, gocmho)

	C.MQCRTMH(x.hConn,
		(C.PMQVOID)(unsafe.Pointer(&mqcmho)),
		(C.PMQHMSG)(unsafe.Pointer(&mqhmsg)),
		&mqcc,
		&mqrc)

	mqreturn := MQReturn{MQCC: int32(mqcc),
		MQRC: int32(mqrc),
		verb: "MQCRTMH",
	}

	copyCMHOfromC(&mqcmho, gocmho)
	msgHandle := MQMessageHandle{hMsg: mqhmsg, qMgr: x}

	if mqcc != C.MQCC_OK {
		traceExitErr("CrtMH", 1, &mqreturn)
		return msgHandle, &mqreturn
	}

	traceExit("CrtMH")
	return msgHandle, nil

}

/*
DltMH is the function to delete a message handle holding properties
*/
func (handle *MQMessageHandle) DltMH(godmho *MQDMHO) error {
	var mqrc C.MQLONG
	var mqcc C.MQLONG

	var mqdmho C.MQDMHO

	traceEntry("DltMH")

	if !IsUsableHandle(*handle) {
		err := &MQReturn{MQCC: MQCC_FAILED,
			MQRC: MQRC_HMSG_ERROR,
			verb: "MQDLTMH",
		}
		traceExitErr("DltMh", 2, err)
		return err
	}
	copyDMHOtoC(&mqdmho, godmho)

	C.MQDLTMH(handle.qMgr.hConn,
		(C.PMQHMSG)(unsafe.Pointer(&handle.hMsg)),
		(C.PMQVOID)(unsafe.Pointer(&mqdmho)),
		&mqcc,
		&mqrc)

	mqreturn := MQReturn{MQCC: int32(mqcc),
		MQRC: int32(mqrc),
		verb: "MQDLTMH",
	}

	copyDMHOfromC(&mqdmho, godmho)

	if mqcc != C.MQCC_OK {
		traceExitErr("DltMH", 1, &mqreturn)
		return &mqreturn
	}

	handle.hMsg = C.MQHM_NONE
	traceExit("DltMH")
	return nil
}

/*
SetMP is the function to set a message property. This function allows the
property value to be (almost) any basic datatype - string, int32, int64, []byte
and converts it into the appropriate format for the C MQI.
*/
func (handle *MQMessageHandle) SetMP(gosmpo *MQSMPO, name string, gopd *MQPD, value interface{}) error {
	var mqrc C.MQLONG
	var mqcc C.MQLONG

	var mqsmpo C.MQSMPO
	var mqpd C.MQPD
	var mqName C.MQCHARV

	var propertyLength C.MQLONG
	var propertyType C.MQLONG
	var propertyPtr C.PMQVOID

	var propertyInt32 C.MQLONG
	var propertyInt64 C.MQINT64
	var propertyBool C.MQLONG
	var propertyInt8 C.MQINT8
	var propertyInt16 C.MQINT16
	var propertyFloat32 C.MQFLOAT32
	var propertyFloat64 C.MQFLOAT64

	traceEntry("SetMP")

	if !IsUsableHandle(*handle) {
		err := &MQReturn{MQCC: MQCC_FAILED,
			MQRC: MQRC_HMSG_ERROR,
			verb: "MQSETMH",
		}
		traceExitErr("SetMh", 2, err)
		return err
	}

	mqName.VSLength = (C.MQLONG)(len(name))
	mqName.VSCCSID = C.MQCCSI_APPL
	if mqName.VSLength > 0 {
		mqName.VSPtr = (C.MQPTR)(C.CString(name))
		mqName.VSBufSize = mqName.VSLength
	}

	propertyType = -1
	if v, ok := value.(int32); ok {
		propertyInt32 = (C.MQLONG)(v)
		propertyType = C.MQTYPE_INT32
		propertyLength = 4
		propertyPtr = (C.PMQVOID)(&propertyInt32)
	} else if v, ok := value.(int64); ok {
		propertyInt64 = (C.MQINT64)(v)
		propertyType = C.MQTYPE_INT64
		propertyLength = 8
		propertyPtr = (C.PMQVOID)(&propertyInt64)
	} else if v, ok := value.(int); ok {
		propertyInt64 = (C.MQINT64)(v)
		propertyType = C.MQTYPE_INT64
		propertyLength = 8
		propertyPtr = (C.PMQVOID)(&propertyInt64)
	} else if v, ok := value.(int8); ok {
		propertyInt8 = (C.MQINT8)(v)
		propertyType = C.MQTYPE_INT8
		propertyLength = 1
		propertyPtr = (C.PMQVOID)(&propertyInt8)
	} else if v, ok := value.(byte); ok { // Separate for int8 and byte (alias uint8)
		propertyInt8 = (C.MQINT8)(v)
		propertyType = C.MQTYPE_INT8
		propertyLength = 1
		propertyPtr = (C.PMQVOID)(&propertyInt8)
	} else if v, ok := value.(int16); ok {
		propertyInt16 = (C.MQINT16)(v)
		propertyType = C.MQTYPE_INT16
		propertyLength = 2
		propertyPtr = (C.PMQVOID)(&propertyInt16)
	} else if v, ok := value.(float32); ok {
		propertyFloat32 = (C.MQFLOAT32)(v)
		propertyType = C.MQTYPE_FLOAT32
		propertyLength = C.sizeof_MQFLOAT32
		propertyPtr = (C.PMQVOID)(&propertyFloat32)
	} else if v, ok := value.(float64); ok {
		propertyFloat64 = (C.MQFLOAT64)(v)
		propertyType = C.MQTYPE_FLOAT64
		propertyLength = C.sizeof_MQFLOAT64
		propertyPtr = (C.PMQVOID)(&propertyFloat64)
	} else if v, ok := value.(string); ok {
		propertyType = C.MQTYPE_STRING
		propertyLength = (C.MQLONG)(len(v))
		propertyPtr = (C.PMQVOID)(C.CString(v))
	} else if v, ok := value.(bool); ok {
		propertyType = C.MQTYPE_BOOLEAN
		propertyLength = 4
		if v {
			propertyBool = 1
		} else {
			propertyBool = 0
		}
		propertyPtr = (C.PMQVOID)(&propertyBool)
	} else if v, ok := value.([]byte); ok {
		propertyType = C.MQTYPE_BYTE_STRING
		propertyLength = (C.MQLONG)(len(v))
		propertyPtr = (C.PMQVOID)(C.malloc(C.size_t(len(v))))
		copy((*[1 << 31]byte)(propertyPtr)[0:len(v)], v)
	} else if v == nil {
		propertyType = C.MQTYPE_NULL
		propertyLength = 0
		propertyPtr = (C.PMQVOID)(C.NULL)
	} else {
		// Unknown datatype - return an error immediately
		mqreturn := MQReturn{MQCC: C.MQCC_FAILED,
			MQRC: C.MQRC_PROPERTY_TYPE_ERROR,
			verb: "MQSETMP",
		}
		traceExitErr("SetMP", 1, &mqreturn)
		return &mqreturn
	}

	copySMPOtoC(&mqsmpo, gosmpo)
	copyPDtoC(&mqpd, gopd)

	C.MQSETMP(handle.qMgr.hConn,
		handle.hMsg,
		(C.PMQVOID)(unsafe.Pointer(&mqsmpo)),
		(C.PMQVOID)(unsafe.Pointer(&mqName)),
		(C.PMQVOID)(unsafe.Pointer(&mqpd)),
		propertyType,
		propertyLength,
		propertyPtr,
		&mqcc,
		&mqrc)

	if len(name) > 0 {
		C.free(unsafe.Pointer(mqName.VSPtr))
	}

	if propertyType == C.MQTYPE_STRING || propertyType == C.MQTYPE_BYTE_STRING {
		C.free(unsafe.Pointer(propertyPtr))
	}

	mqreturn := MQReturn{MQCC: int32(mqcc),
		MQRC: int32(mqrc),
		verb: "MQSETMP",
	}

	copySMPOfromC(&mqsmpo, gosmpo)
	copyPDfromC(&mqpd, gopd)

	if mqcc != C.MQCC_OK {
		traceExitErr("SetMP", 2, &mqreturn)
		return &mqreturn
	}

	traceExit("SetMP")
	return nil
}

/*
DltMP is the function to remove a message property.
*/
func (handle *MQMessageHandle) DltMP(godmpo *MQDMPO, name string) error {
	var mqrc C.MQLONG
	var mqcc C.MQLONG

	var mqdmpo C.MQDMPO
	var mqName C.MQCHARV

	traceEntry("DltMP")

	if !IsUsableHandle(*handle) {
		err := &MQReturn{MQCC: MQCC_FAILED,
			MQRC: MQRC_HMSG_ERROR,
			verb: "MQDLTMP",
		}
		traceExitErr("DltMp", 2, err)
		return err
	}

	mqName.VSLength = (C.MQLONG)(len(name))
	mqName.VSCCSID = C.MQCCSI_APPL
	if mqName.VSLength > 0 {
		mqName.VSPtr = (C.MQPTR)(C.CString(name))
		mqName.VSBufSize = mqName.VSLength
	}

	copyDMPOtoC(&mqdmpo, godmpo)

	C.MQDLTMP(handle.qMgr.hConn,
		handle.hMsg,
		(C.PMQVOID)(unsafe.Pointer(&mqdmpo)),
		(C.PMQVOID)(unsafe.Pointer(&mqName)),
		&mqcc,
		&mqrc)

	if len(name) > 0 {
		C.free(unsafe.Pointer(mqName.VSPtr))
	}

	mqreturn := MQReturn{MQCC: int32(mqcc),
		MQRC: int32(mqrc),
		verb: "MQDLTMP",
	}

	copyDMPOfromC(&mqdmpo, godmpo)

	if mqcc != C.MQCC_OK {
		traceExitErr("DltMP", 1, &mqreturn)
		return &mqreturn
	}

	traceExit("DltMP")
	return nil
}

/*
InqMP is the function to inquire about the value of a message property.
*/
func (handle *MQMessageHandle) InqMP(goimpo *MQIMPO, gopd *MQPD, name string) (string, interface{}, error) {
	var mqrc C.MQLONG
	var mqcc C.MQLONG

	var mqimpo C.MQIMPO
	var mqpd C.MQPD
	var mqName C.MQCHARV

	var propertyLength C.MQLONG
	var propertyType C.MQLONG
	var propertyPtr C.PMQVOID
	var propertyValue interface{}

	const namebufsize = 1024
	const propbufsize = 10240

	traceEntry("InqMP")

	if !IsUsableHandle(*handle) {
		err := &MQReturn{MQCC: MQCC_FAILED,
			MQRC: MQRC_HMSG_ERROR,
			verb: "MQINQMP",
		}
		traceExitErr("InqMp", 2, err)
		return "", nil, err
	}

	mqName.VSLength = (C.MQLONG)(len(name))
	mqName.VSCCSID = C.MQCCSI_APPL
	if mqName.VSLength > 0 {
		mqName.VSPtr = (C.MQPTR)(C.CString(name))
		mqName.VSBufSize = mqName.VSLength
	} else {
		mqName.VSPtr = (C.MQPTR)(C.malloc(namebufsize))
		mqName.VSBufSize = namebufsize
	}
	// VSPtr is either explicit malloc or comes from CString which does a
	// malloc. Either way, the buffer should be freed at the end.
	defer C.free(unsafe.Pointer(mqName.VSPtr))

	copyIMPOtoC(&mqimpo, goimpo)
	copyPDtoC(&mqpd, gopd)

	// Use a local buffer instead of something global so we don't
	// have to worry about multiple threads accessing it.
	propertyPtr = C.PMQVOID(C.malloc(propbufsize))
	defer C.free(unsafe.Pointer(propertyPtr))

	bufferLength := C.MQLONG(namebufsize)

	C.MQINQMP(handle.qMgr.hConn,
		handle.hMsg,
		(C.PMQVOID)(unsafe.Pointer(&mqimpo)),
		(C.PMQVOID)(unsafe.Pointer(&mqName)),
		(C.PMQVOID)(unsafe.Pointer(&mqpd)),
		(C.PMQLONG)(unsafe.Pointer(&propertyType)),
		bufferLength,
		propertyPtr,
		(C.PMQLONG)(unsafe.Pointer(&propertyLength)),
		&mqcc,
		&mqrc)

	mqreturn := MQReturn{MQCC: int32(mqcc),
		MQRC: int32(mqrc),
		verb: "MQINQMP",
	}

	copyIMPOfromC(&mqimpo, goimpo)
	copyPDfromC(&mqpd, gopd)

	if mqcc != C.MQCC_OK {
		traceExitErr("InqMP", 1, &mqreturn)
		return "", nil, &mqreturn
	}

	switch propertyType {
	case C.MQTYPE_INT8:
		p := (*C.MQBYTE)(propertyPtr)
		propertyValue = (int8)(*p)
	case C.MQTYPE_INT16:
		p := (*C.MQINT16)(propertyPtr)
		propertyValue = (int16)(*p)
	case C.MQTYPE_INT32:
		p := (*C.MQINT32)(propertyPtr)
		propertyValue = (int32)(*p)
	case C.MQTYPE_INT64:
		p := (*C.MQINT64)(propertyPtr)
		propertyValue = (int64)(*p)
	case C.MQTYPE_FLOAT32:
		p := (*C.MQFLOAT32)(propertyPtr)
		propertyValue = (float32)(*p)
	case C.MQTYPE_FLOAT64:
		p := (*C.MQFLOAT64)(propertyPtr)
		propertyValue = (float64)(*p)
	case C.MQTYPE_BOOLEAN:
		p := (*C.MQLONG)(propertyPtr)
		b := (int32)(*p)
		if b == 0 {
			propertyValue = false
		} else {
			propertyValue = true
		}
	case C.MQTYPE_STRING:
		propertyValue = C.GoStringN((*C.char)(propertyPtr), (C.int)(propertyLength))
	case C.MQTYPE_BYTE_STRING:
		ba := make([]byte, propertyLength)
		p := (*C.MQBYTE)(propertyPtr)
		copy(ba[:], C.GoBytes(unsafe.Pointer(p), (C.int)(propertyLength)))
		propertyValue = ba
	case C.MQTYPE_NULL:
		propertyValue = nil
	}

	traceExit("InqMP")
	return goimpo.ReturnedName, propertyValue, nil
}

/*
GetHeader returns a structure containing a parsed-out version of an MQI
message header such as the MQDLH (which is currently the only structure
supported). Other structures like the RFH2 could follow.

The caller of this function needs to cast the returned structure to the
specific type in order to reference the fields.
*/
func GetHeader(md *MQMD, buf []byte) (interface{}, int, error) {
	switch md.Format {
	case MQFMT_DEAD_LETTER_HEADER:
		return getHeaderDLH(md, buf)
	case MQFMT_RF_HEADER_2:
		return getHeaderRFH2(md, buf)
	}

	mqreturn := &MQReturn{MQCC: int32(MQCC_FAILED),
		MQRC: int32(MQRC_FORMAT_NOT_SUPPORTED),
	}

	return nil, 0, mqreturn
}

func readStringFromFixedBuffer(r io.Reader, l int32) string {
	tmpBuf := make([]byte, l)
	binary.Read(r, endian, tmpBuf)

	s := string(tmpBuf)
	i := strings.IndexByte(s, 0)
	if i >= 0 {
		s = s[0:i]
	}
	return strings.TrimSpace(s)
}

// The date/time fields are being taken from a valid MQMD but they still might not be
// "real" timestamps if the putting application has overridden context setting. If the values
// are invalid, then we return an empty Go value
func createGoDateTime(d string, t string) time.Time {

	if len(d) == int(MQ_PUT_DATE_LENGTH) && len(t) == int(MQ_PUT_TIME_LENGTH) {
		// Combine the MQI strings into a single parseable string
		s := d + t[:6] + "." + t[6:] + " UTC" // MQ times are always given as UTC
		goTime, err := time.Parse(mqDateTimeFormat, s)
		if err != nil {
			return time.Time{}
		} else {
			return goTime
		}
	} else {
		return time.Time{}
	}
}

// If the application has set a Go timestamp, it ought to be valid. So we can try to convert it
// to the MQ separate string formats.
func createCDateTime(goTime time.Time) (string, string) {
	d := goTime.Format(mqDateFormat) // These magic values tell Go how to parse/format between Times and strings
	t := goTime.Format(mqTimeFormat)
	t = t[:6] + t[7:] // Strip the '.'
	return d, t
}
