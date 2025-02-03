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
This file deals with asynchronous delivery of MQ messages via the MQCTL/MQCB verbs.
*/

/*
#include <stdlib.h>
#include <string.h>
#include <cmqc.h>

extern void MQCALLBACK_Go(MQHCONN, MQMD *, MQGMO *, PMQVOID, MQCBC *);
extern void MQCALLBACK_C(MQHCONN hc,MQMD *md,MQGMO *gmo,PMQVOID buf,MQCBC *cbc);
*/
import "C"
import (
	"fmt"
	"strings"
	"sync"
	"unsafe"
)

// The user's callback function must match this signature
type MQCB_FUNCTION func(*MQQueueManager, *MQObject, *MQMD, *MQGMO, []byte, *MQCBC, *MQReturn)

// Need to keep references to the user's callback function and some other
// structure elements which do not map to the C functions, or do not need to
// be passed onwards
type cbInfo struct {
	hObj             *MQObject
	callbackFunction MQCB_FUNCTION
	callbackArea     interface{}
	connectionArea   interface{}
	otelOpts         OtelOpts
}

// This map is indexed by a combination of the hConn and hObj values
var cbMap = make(map[string]*cbInfo)

// Add a mutex to control access to it as there may be several threads going for different qmgrs
var mutex sync.Mutex

/*
MQCALLBACK_Go is a wrapper callback function that will invoke the user-supplied callback
after converting the C structures into the corresponding Go format.

The "export" directive makes the function available through the CGo processing to be
accessible from a C function. See mqicb_c.go for the proxy/gateway C function that in turn calls this one
*/
//export MQCALLBACK_Go
func MQCALLBACK_Go(hConn C.MQHCONN, mqmd *C.MQMD, mqgmo *C.MQGMO, mqBuffer C.PMQVOID, mqcbc *C.MQCBC) {

	var cbHObj *MQObject

	// Find the real callback function and invoke it
	// Invoked function should match signature of the MQCB_FUNCTION type
	gogmo := NewMQGMO()
	gomd := NewMQMD()
	gocbc := NewMQCBC()

	traceEntry("Callback")

	// For EVENT callbacks, the GMO and MD may be NULL
	if mqgmo != (C.PMQGMO)(C.NULL) {
		copyGMOfromC(mqgmo, gogmo)
	}

	if mqmd != (C.PMQMD)(C.NULL) {
		copyMDfromC(mqmd, gomd)
	}

	// This should never be NULL
	copyCBCfromC(mqcbc, gocbc)

	mqrc := int32(mqcbc.Reason)
	mqcc := int32(mqcbc.CompCode)
	mqreturn := &MQReturn{MQCC: mqcc,
		MQRC: mqrc,
		verb: "MQCALLBACK",
	}

	key := makeKey(hConn, mqcbc.Hobj)
	mapLock()
	info, ok := cbMap[key]
	mapUnlock()

	// The MQ Client libraries sometimes call us with an EVENT that is
	// not associated with a particular hObj.
	// The way I've chosen is to find the first entry in
	// the map associated with the hConn and call its registered function with
	// a dummy hObj.
	if !ok {
		if gocbc.CallType == MQCBCT_EVENT_CALL && mqcbc.Hobj == C.MQHO_NONE {
			key = makePartialKey(hConn)
			mapLock()
			for k, i := range cbMap {
				if strings.HasPrefix(k, key) {
					ok = true
					info = i
					cbHObj = &MQObject{qMgr: info.hObj.qMgr, Name: ""}
					// Only care about finding one match in the table
					break
				}
			}
			mapUnlock()
		}
	} else {
		cbHObj = info.hObj
	}

	if ok {
		if gogmo.MsgHandle.hMsg != C.MQHM_NONE {
			gogmo.MsgHandle.qMgr = cbHObj.qMgr
		}

		// Set the context elements that we stashed before, and which are
		// not used in the C structure
		gocbc.CallbackArea = info.callbackArea
		gocbc.ConnectionArea = info.connectionArea
		gocbc.OtelOpts.Context = info.otelOpts.Context

		removed := 0
		// Get the data
		b := C.GoBytes(unsafe.Pointer(mqBuffer), C.int(mqcbc.DataLength))

		// Only process OTEL tracing if we actually got a message
		if mqcc == C.MQCC_OK || mqrc == C.MQRC_TRUNCATED_MSG_ACCEPTED {
			f2 := otelFuncs.GetTraceAfter
			if f2 != nil {
				removed = f2(info.otelOpts, cbHObj, gogmo, gomd, b, true)
			}
		}

		// And finally call the user function
		logTrace("Calling user function with %d bytes", len(b))
		info.callbackFunction(cbHObj.qMgr, cbHObj, gomd, gogmo, b[removed:], gocbc, mqreturn)

	}

	if mqreturn.MQCC != C.MQCC_OK {
		traceExitErr("Callback", 1, mqreturn)
	} else {
		traceExit("Callback")
	}
}

/*
CB is the function to register/unregister a callback function for an object, based on
criteria in the message descriptor and get-message-options. There are 2 variations of
the function - one for queue-based and one for an hConn-wide event handler that does not
require an hObj.
*/

func (object *MQObject) CB(goOperation int32, gocbd *MQCBD, gomd *MQMD, gogmo *MQGMO) error {
	var mqrc C.MQLONG
	var mqcc C.MQLONG
	var mqOperation C.MQLONG
	var mqcbd C.MQCBD
	var mqmd C.MQMD
	var mqgmo C.MQGMO

	traceEntry("CB(Q)")

	err := checkMD(gomd, "MQCB")
	if err != nil {
		traceExitErr("CB(Q)", 1, err)
		return err
	}
	err = checkGMO(gogmo, "MQCB")
	if err != nil {
		traceExitErr("CB(Q)", 2, err)
		return err
	}

	mqOperation = C.MQLONG(goOperation)

	f1 := otelFuncs.GetTraceBefore
	if f1 != nil {
		f1(gogmo.OtelOpts, object.qMgr, object, gogmo, true)
	}
	copyCBDtoC(&mqcbd, gocbd)
	copyMDtoC(&mqmd, gomd)
	copyGMOtoC(&mqgmo, gogmo)

	key := makeKey(object.qMgr.hConn, object.hObj)

	// The callback function is a C function that is a proxy for the MQCALLBACK_Go function
	// defined here. And that in turn will call the user's callback function
	mqcbd.CallbackFunction = (C.MQPTR)(unsafe.Pointer(C.MQCALLBACK_C))

	C.MQCB(object.qMgr.hConn, mqOperation, (C.PMQVOID)(unsafe.Pointer(&mqcbd)),
		object.hObj,
		(C.PMQVOID)(unsafe.Pointer(&mqmd)), (C.PMQVOID)(unsafe.Pointer(&mqgmo)),
		&mqcc, &mqrc)

	mqreturn := MQReturn{MQCC: int32(mqcc),
		MQRC: int32(mqrc),
		verb: "MQCB",
	}

	if mqcc != C.MQCC_OK {
		traceExitErr("CB", 3, &mqreturn)
		return &mqreturn
	}

	// Add or remove the control information in the map used by the callback routines
	switch mqOperation {
	case C.MQOP_DEREGISTER:
		mapLock()
		delete(cbMap, key)
		mapUnlock()
	case C.MQOP_REGISTER:
		// Stash the hObj and real function to be called
		info := &cbInfo{hObj: object,
			callbackFunction: gocbd.CallbackFunction,
			connectionArea:   nil,
			callbackArea:     gocbd.CallbackArea,
		}
		info.otelOpts.Context = gogmo.OtelOpts.Context
		info.otelOpts.RemoveRFH2 = gogmo.OtelOpts.RemoveRFH2

		mapLock()
		cbMap[key] = info
		mapUnlock()

	default: // Other values leave the map alone
	}

	traceExit("CB(Q)")
	return nil
}

func (object *MQQueueManager) CB(goOperation int32, gocbd *MQCBD) error {
	var mqrc C.MQLONG
	var mqcc C.MQLONG
	var mqOperation C.MQLONG
	var mqcbd C.MQCBD

	traceEntry("CB(QM)")

	mqOperation = C.MQLONG(goOperation)
	copyCBDtoC(&mqcbd, gocbd)

	key := makeKey(object.hConn, C.MQHO_NONE)

	// The callback function is a C function that is a proxy for the MQCALLBACK_Go function
	// defined here. And that in turn will call the user's callback function
	mqcbd.CallbackFunction = (C.MQPTR)(unsafe.Pointer(C.MQCALLBACK_C))

	C.MQCB(object.hConn, mqOperation, (C.PMQVOID)(unsafe.Pointer(&mqcbd)),
		C.MQHO_NONE, nil, nil,
		&mqcc, &mqrc)

	mqreturn := MQReturn{MQCC: int32(mqcc),
		MQRC: int32(mqrc),
		verb: "MQCB",
	}

	if mqcc != C.MQCC_OK {
		traceExitErr("CB(QM)", 1, &mqreturn)
		return &mqreturn
	}

	// Add or remove the control information in the map used by the callback routines
	switch mqOperation {
	case C.MQOP_DEREGISTER:
		mapLock()
		delete(cbMap, key)
		mapUnlock()
	case C.MQOP_REGISTER:
		// Stash an hObj and real function to be called
		info := &cbInfo{hObj: &MQObject{qMgr: object, Name: ""},
			callbackFunction: gocbd.CallbackFunction,
			connectionArea:   nil,
			callbackArea:     gocbd.CallbackArea,
		}
		mapLock()
		cbMap[key] = info
		mapUnlock()
	default: // Other values leave the map alone
	}

	traceExit("CB(QM)")
	return nil
}

/*
Ctl is the function that starts/stops invocation of a registered callback.
*/
func (x *MQQueueManager) Ctl(goOperation int32, goctlo *MQCTLO) error {
	var mqrc C.MQLONG
	var mqcc C.MQLONG
	var mqOperation C.MQLONG
	var mqctlo C.MQCTLO

	traceEntry("Ctl")

	mqOperation = C.MQLONG(goOperation)
	copyCTLOtoC(&mqctlo, goctlo)

	// Need to make sure control information is available before the callback
	// is enabled. So this gets setup even if the MQCTL fails.
	key := makePartialKey(x.hConn)
	mapLock()
	for k, info := range cbMap {
		if strings.HasPrefix(k, key) {
			info.connectionArea = goctlo.ConnectionArea
		}
	}
	mapUnlock()

	C.MQCTL(x.hConn, mqOperation, (C.PMQVOID)(unsafe.Pointer(&mqctlo)), &mqcc, &mqrc)

	mqreturn := MQReturn{MQCC: int32(mqcc),
		MQRC: int32(mqrc),
		verb: "MQCTL",
	}

	if mqcc != C.MQCC_OK {
		traceExitErr("Ctl", 1, &mqreturn)
		return &mqreturn
	}

	traceExit("Ctl")
	return nil
}

// Functions below here manage the map of objects and control information so that
// the Go variables can be saved/restored from invocations to the C layer
func makeKey(hConn C.MQHCONN, hObj C.MQHOBJ) string {
	key := fmt.Sprintf("%d/%d", hConn, hObj)
	return key
}

func makePartialKey(hConn C.MQHCONN) string {
	key := fmt.Sprintf("%d/", hConn)
	return key
}

// Functions to delete any structures used to map C elements to Go
func cbRemoveConnection(hConn C.MQHCONN) {
	// Remove all of the hObj values for this hconn
	key := makePartialKey(hConn)
	mapLock()
	for k, _ := range cbMap {
		if strings.HasPrefix(k, key) {
			delete(cbMap, k)
		}
	}
	mapUnlock()
}

func cbRemoveHandle(hConn C.MQHCONN, hObj C.MQHOBJ) {
	key := makeKey(hConn, hObj)
	mapLock()
	delete(cbMap, key)
	mapUnlock()
}

func mapLock() {
	mutex.Lock()
}
func mapUnlock() {
	mutex.Unlock()
}
