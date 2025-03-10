package ibmmq

/*
  Copyright (c) IBM Corporation 2016,2023

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
C functions allow support of new features while still compiling
on older versions of the MQ header files. It uses the _VERSION values
to select what can be done.
*/

/*

#include <stdlib.h>
#include <string.h>
#include <cmqc.h>
#include <cmqxc.h>

void setCnoCCDTUrl(MQCNO *mqcno, PMQCHAR url, MQLONG length) {
#if defined(MQCNO_VERSION_6) && MQCNO_CURRENT_VERSION >= MQCNO_VERSION_6
  if (mqcno->Version < MQCNO_VERSION_6) {
	  mqcno->Version = MQCNO_VERSION_6;
  }
  mqcno->CCDTUrlOffset = 0;
  mqcno->CCDTUrlPtr = NULL;
  mqcno->CCDTUrlLength = length;
  if (url != NULL && length > 0) {
    mqcno->CCDTUrlPtr = url;
  }
#else
// We fail silently here, but perhaps ought to give an error in some way as you've tried to use
// a parameter that is not permitted at the version of MQ that you are building against
  if (url != NULL) {
    free(url);
  }
#endif
}

void freeCnoCCDTUrl(MQCNO *mqcno) {
#if defined(MQCNO_VERSION_6) && MQCNO_CURRENT_VERSION >= MQCNO_VERSION_6
  if (mqcno->CCDTUrlPtr != NULL) {
    free(mqcno->CCDTUrlPtr);
  }
#endif
}

void setCnoApplName(MQCNO *mqcno, PMQCHAR applName, MQLONG length) {
#if defined(MQCNO_VERSION_7) && MQCNO_CURRENT_VERSION >= MQCNO_VERSION_7
  if (applName != NULL) {
    memset(mqcno->ApplName,0,length);
    strncpy(mqcno->ApplName,applName,length);
    if (mqcno->Version < MQCNO_VERSION_7) {
      mqcno->Version = MQCNO_VERSION_7;
    }
  }
#endif

  // We can always free the string space immediately as it's not used inside the MQI. Unlike
  // the varchar elements such as CCDTUrl.
  if (applName != NULL) {
    free(applName);
  }
  return;
}

// A structure introduced in MQ 9.2.4. In order to handle builds against older versions of MQ
// we have to extract the individual fields from the Go version of the structure first. And
// we then use those as separate parameters to this function.
void setCnoBalanceParms(MQCNO *mqcno, MQLONG ApplType, MQLONG Timeout, MQLONG Options) {
#if defined(MQCNO_VERSION_8) && MQCNO_CURRENT_VERSION >= MQCNO_VERSION_8
  PMQBNO pmqbno = malloc(MQBNO_CURRENT_LENGTH); // This is freed on return from the C function
  pmqbno->Version = MQBNO_VERSION_1;
  memcpy(pmqbno->StrucId,MQBNO_STRUC_ID,4);
  pmqbno->ApplType = ApplType;
  pmqbno->Timeout = Timeout;
  pmqbno->Options = Options;
  mqcno->BalanceParmsPtr = pmqbno;
  mqcno->BalanceParmsOffset = 0;
  if (mqcno->Version < MQCNO_VERSION_8) {
    mqcno->Version = MQCNO_VERSION_8;
  }
#endif
  return;
}

void freeCnoBalanceParms(MQCNO *mqcno) {
#if defined(MQCNO_VERSION_8) && MQCNO_CURRENT_VERSION >= MQCNO_VERSION_8
  if (mqcno->Version >= MQCNO_VERSION_8 && mqcno->BalanceParmsPtr != NULL) {
    free(mqcno->BalanceParmsPtr);
  }
#endif
  return;
}

void setCspInitialKey(MQCSP *mqcsp, PMQCHAR initialKey, MQLONG length) {
#if defined(MQCSP_VERSION_2) && MQCSP_CURRENT_VERSION >= MQCSP_VERSION_2
  if (mqcsp->Version < MQCSP_VERSION_2) {
	  mqcsp->Version = MQCSP_VERSION_2;
  }
  mqcsp->InitialKeyOffset = 0;
  mqcsp->InitialKeyLength = length;
  if (initialKey != NULL && length > 0) {
    mqcsp->InitialKeyPtr = initialKey;
  } else {
    mqcsp->InitialKeyPtr = NULL;
  }
#else
// We fail silently here, but perhaps ought to give an error in some way as you've tried to use
// a parameter that is not permitted at the version of MQ that you are building against
  if (initialKey != NULL) {
    free(initialKey);
  }
#endif
}

void freeCspInitialKey(MQCSP *mqcsp) {
#if defined(MQCSP_VERSION_2) && MQCSP_CURRENT_VERSION >= MQCSP_VERSION_2
  if (mqcsp->Version >= MQCSP_VERSION_2 && mqcsp->InitialKeyPtr != NULL) {
    free(mqcsp->InitialKeyPtr);
  }
#endif
}

// If the application sets a token, then that is what we will use for
// authentication, overriding the userid/password.
void setCspToken(MQCSP *mqcsp, PMQCHAR token, MQLONG length) {
#if defined(MQCSP_VERSION_3) && MQCSP_CURRENT_VERSION >= MQCSP_VERSION_3
  if (mqcsp->Version < MQCSP_VERSION_3) {
	  mqcsp->Version = MQCSP_VERSION_3;
  }
  mqcsp->AuthenticationType = MQCSP_AUTH_ID_TOKEN;

  mqcsp->TokenOffset = 0;
  mqcsp->TokenLength = length;
  if (token != NULL && length > 0) {
    mqcsp->TokenPtr = token;
  } else {
    mqcsp->TokenPtr = NULL;
  }
#else
  if (token != NULL) {
    free(token);
  }
#endif
}

void freeCspToken(MQCSP *mqcsp) {
#if defined(MQCSP_VERSION_3) && MQCSP_CURRENT_VERSION >= MQCSP_VERSION_3
  if (mqcsp->Version >= MQCSP_VERSION_3 && mqcsp->TokenPtr != NULL) {
    free(mqcsp->TokenPtr);
  }
#endif
}

size_t getMaxCDLength() {
  size_t l;
#if defined(MQCD_VERSION_12)
  l = MQCD_LENGTH_12;
#else
  l = MQCD_LENGTH_11; // The minimum supported here
#endif
  return l;
}

size_t getMaxSCOLength() {
  size_t l;
#if defined(MQSCO_VERSION_7)
  l = MQSCO_LENGTH_7;
#elif defined(MQSCO_VERSION_6)
  l = MQSCO_LENGTH_6;
#else
  l = MQSCO_LENGTH_5; // The minimum supported here
#endif
  return l;
}

size_t getMaxCSPLength() {
  size_t l;
#if defined(MQCSP_VERSION_3)
  l = MQCSP_LENGTH_3;
#elif defined(MQCSP_VERSION_2)
  l = MQCSP_LENGTH_2;
#else
  l = MQCSP_LENGTH_1;
#endif
  return l;
}

*/
import "C"
import "unsafe"

/*
MQCNO is a structure containing the MQ Connection Options (MQCNO)
Note that only a subset of the real structure is exposed in this
version.
*/
type MQCNO struct {
	Version       int32
	Options       int32
	SecurityParms *MQCSP
	CCDTUrl       string
	ClientConn    *MQCD
	SSLConfig     *MQSCO
	ApplName      string
	BalanceParms  *MQBNO
}

/*
MQCSP is a structure containing the MQ Security Parameters (MQCSP)
*/
type MQCSP struct {
	AuthenticationType int32
	UserId             string
	Password           string
	InitialKey         string
	Token              string
}

/*
MQBNO is a structure to allow an application provision of balancing options
*/
type MQBNO struct {
	ApplType int32
	Timeout  int32
	Options  int32
}

/*
NewMQCNO fills in default values for the MQCNO structure
*/
func NewMQCNO() *MQCNO {

	cno := new(MQCNO)
	cno.Version = int32(C.MQCNO_VERSION_1)
	cno.Options = int32(C.MQCNO_NONE)
	cno.SecurityParms = nil
	cno.ClientConn = nil
	cno.CCDTUrl = ""
	cno.ApplName = ""

	return cno
}

/*
NewMQCSP fills in default values for the MQCSP structure
*/
func NewMQCSP() *MQCSP {

	csp := new(MQCSP)
	csp.AuthenticationType = int32(C.MQCSP_AUTH_NONE)
	csp.UserId = ""
	csp.Password = ""
	csp.InitialKey = ""
	csp.Token = ""

	return csp
}

/*
NewMQBNO fills in default values for the MQBNO structure. We
use the constant values directly as the #define macros may not be
available when building against older levels of the MQ client code.
*/
func NewMQBNO() *MQBNO {
	bno := new(MQBNO)
	bno.ApplType = 0 /* MQBNO_BALTYPE_SIMPLE */
	bno.Timeout = -1 /* MQBNO_TIMEOUT_AS_DEFAULT */
	bno.Options = 0  /* MQBNO_OPTIONS_NONE */

	return bno
}

func copyCNOtoC(mqcno *C.MQCNO, gocno *MQCNO) {
	var i int
	var mqcsp C.PMQCSP
	var mqcd C.PMQCD
	var mqsco C.PMQSCO

	setMQIString((*C.char)(&mqcno.StrucId[0]), "CNO ", 4)
	mqcno.Version = C.MQLONG(gocno.Version)
	mqcno.Options = C.MQLONG(gocno.Options)

	for i = 0; i < C.MQ_CONN_TAG_LENGTH; i++ {
		mqcno.ConnTag[i] = 0
	}
	for i = 0; i < C.MQ_CONNECTION_ID_LENGTH; i++ {
		mqcno.ConnectionId[i] = 0
	}

	mqcno.ClientConnOffset = 0
	if gocno.ClientConn != nil {
		gocd := gocno.ClientConn
		l := C.getMaxCDLength()
		mqcd = C.PMQCD(C.malloc(l))
		C.memset((unsafe.Pointer)(mqcd), 0, C.size_t(l))
		copyCDtoC(mqcd, gocd)
		mqcno.ClientConnPtr = C.MQPTR(mqcd)
		if gocno.Version < 2 {
			mqcno.Version = C.MQCNO_VERSION_2
		}
	} else {
		mqcno.ClientConnPtr = nil
	}

	mqcno.SSLConfigOffset = 0
	if gocno.SSLConfig != nil {
		gosco := gocno.SSLConfig
		l := C.getMaxSCOLength()
		mqsco = C.PMQSCO(C.malloc(l))
		C.memset((unsafe.Pointer)(mqsco), 0, C.size_t(l))
		copySCOtoC(mqsco, gosco)
		mqcno.SSLConfigPtr = C.PMQSCO(mqsco)
		if gocno.Version < 4 {
			mqcno.Version = C.MQCNO_VERSION_4
		}
	} else {
		mqcno.SSLConfigPtr = nil
	}

	mqcno.SecurityParmsOffset = 0
	if gocno.SecurityParms != nil {
		gocsp := gocno.SecurityParms

		l := C.getMaxCSPLength()
		mqcsp = C.PMQCSP(C.malloc(l))
		C.memset((unsafe.Pointer)(mqcsp), 0, C.size_t(l))
		setMQIString((*C.char)(&mqcsp.StrucId[0]), "CSP ", 4)
		mqcsp.Version = C.MQCSP_VERSION_1
		mqcsp.AuthenticationType = C.MQLONG(gocsp.AuthenticationType)
		mqcsp.CSPUserIdOffset = 0
		mqcsp.CSPPasswordOffset = 0

		if gocsp.UserId != "" {
			// If you've set a non-blank userid, then you MUST be asking for userid/pwd checking.
			if mqcsp.AuthenticationType == C.MQCSP_AUTH_NONE {
				mqcsp.AuthenticationType = C.MQLONG(C.MQCSP_AUTH_USER_ID_AND_PWD)
			}
			mqcsp.CSPUserIdPtr = C.MQPTR(unsafe.Pointer(C.CString(gocsp.UserId)))
			mqcsp.CSPUserIdLength = C.MQLONG(len(gocsp.UserId))
		} else {
			mqcsp.CSPUserIdPtr = nil
			mqcsp.CSPUserIdLength = 0
		}

		if gocsp.Password != "" {
			mqcsp.CSPPasswordPtr = C.MQPTR(unsafe.Pointer(C.CString(gocsp.Password)))
			mqcsp.CSPPasswordLength = C.MQLONG(len(gocsp.Password))
		} else {
			mqcsp.CSPPasswordPtr = nil
			mqcsp.CSPPasswordLength = 0
		}

		if gocsp.InitialKey != "" {
			// This C function will bump the CSP version to V2 if necessary
			C.setCspInitialKey(mqcsp, C.PMQCHAR(C.CString(gocsp.InitialKey)), C.MQLONG(len(gocsp.InitialKey)))
		}

		if gocsp.Token != "" {
			// This C function will bump the CSP version to V3 if necessary
			// It also overrides the AuthenticationType
			C.setCspToken(mqcsp, C.PMQCHAR(C.CString(gocsp.Token)), C.MQLONG(len(gocsp.Token)))
		}

		mqcno.SecurityParmsPtr = C.PMQCSP(mqcsp)
		if gocno.Version < 5 {
			mqcno.Version = C.MQCNO_VERSION_5
		}

	} else {
		mqcno.SecurityParmsPtr = nil
	}

	// The CCDT URL option was introduced in MQ V9. To compile against older
	// versions of MQ, setting of it has been moved to a C function that can use
	// the pre-processor to decide whether it's needed.
	if gocno.CCDTUrl != "" {
		C.setCnoCCDTUrl(mqcno, C.PMQCHAR(C.CString(gocno.CCDTUrl)), C.MQLONG(len(gocno.CCDTUrl)))
	}

	// The ApplName option to the CNO was introduced in MQ V9.1.2. To compile against
	// older versions of MQ, setting of it is in a C function. The function
	// will free() the CString-allocated buffer regardless of MQ version.
	if gocno.ApplName != "" {
		C.setCnoApplName(mqcno, C.PMQCHAR(C.CString(gocno.ApplName)), C.MQ_APPL_NAME_LENGTH)
	}

	// The BalanceParms structure was added to the CNO in MQ 9.2.4. To compile against
	// older versions of MQ, setting is in a C function.
	if gocno.BalanceParms != nil {
		bno := gocno.BalanceParms
		C.setCnoBalanceParms(mqcno, C.MQLONG(bno.ApplType), C.MQLONG(bno.Timeout), C.MQLONG(bno.Options))
	}

	return
}

func copyCNOfromC(mqcno *C.MQCNO, gocno *MQCNO) {

	if mqcno.SecurityParmsPtr != nil {
		if mqcno.SecurityParmsPtr.CSPUserIdPtr != nil {
			C.free(unsafe.Pointer(mqcno.SecurityParmsPtr.CSPUserIdPtr))
		}
		// Set memory to 0 for area that held a password
		if mqcno.SecurityParmsPtr.CSPPasswordPtr != nil {
			C.memset((unsafe.Pointer)(mqcno.SecurityParmsPtr.CSPPasswordPtr), 0, C.size_t(mqcno.SecurityParmsPtr.CSPPasswordLength))
			C.free(unsafe.Pointer(mqcno.SecurityParmsPtr.CSPPasswordPtr))
		}

		C.freeCspInitialKey(mqcno.SecurityParmsPtr) // The code in this function checks validity
		C.freeCspToken(mqcno.SecurityParmsPtr)      // The code in this function checks validity

		C.free(unsafe.Pointer(mqcno.SecurityParmsPtr))
	}

	if mqcno.ClientConnPtr != nil {
		copyCDfromC(C.PMQCD(mqcno.ClientConnPtr), gocno.ClientConn)
		C.free(unsafe.Pointer(mqcno.ClientConnPtr))
	}

	if mqcno.SSLConfigPtr != nil {
		copySCOfromC(C.PMQSCO(mqcno.SSLConfigPtr), gocno.SSLConfig)
		C.free(unsafe.Pointer(mqcno.SSLConfigPtr))
	}

	// Do any freeing up of control blocks malloced by the C functions used to permit
	// compilation against older versions of MQ.
	C.freeCnoCCDTUrl(mqcno)
	C.freeCnoBalanceParms(mqcno)
	// ApplName is input-only so we don't need to do any version-specific processing
	// for it in this function.

	return
}
