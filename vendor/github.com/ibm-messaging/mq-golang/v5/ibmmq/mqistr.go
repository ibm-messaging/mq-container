package ibmmq

/*
****************************************************************
*
*
*                     IBM MQ for Go on all platforms
* FILE NAME:      mqistr.go
*
* This file contains the MQI definitions needed for a
* Go interface. Only 64-bit applications are supported by this
* package.
* The definitions are given directly with no additional explanation
* for each value; those can be found in other header files such as
* cmqc.h.
****************************************************************
* Copyright (c) IBM Corporation 1993, 2025
*
* Licensed under the Apache License, Version 2.0 (the "License");
* you may not use this file except in compliance with the License.
* You may obtain a copy of the License at
*
* http://www.apache.org/licenses/LICENSE-2.0
*
* Unless required by applicable law or agreed to in writing, software
* distributed under the License is distributed on an "AS IS" BASIS,
* WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
* See the License for the specific language governing permissions and
* limitations under the License.
****************************************************************
*
 */

/*
#include <stdlib.h>
#include <string.h>
#include <cmqc.h>
#include <cmqstrc.h>
*/
import "C"

import (
	"fmt"
	"strings"
)

/*
Convert MQCC/MQRC values into readable text using
the functions introduced in cmqstrc.h in MQ V8004
*/
func mqstrerror(verb string, mqcc C.MQLONG, mqrc C.MQLONG) string {
	return fmt.Sprintf("%s: MQCC = %s [%d] MQRC = %s [%d]", verb,
		C.GoString(C.MQCC_STR(mqcc)), mqcc,
		C.GoString(C.MQRC_STR(mqrc)), mqrc)
}

func MQItoStringStripPrefix(class string, value int) string {
	s := MQItoString(class, value)
	c := ""
	if strings.HasPrefix(class, "MQ") {
		c = class
	} else {
		c = "MQ" + class
	}
	if strings.HasPrefix(s, c) {
		l := strings.IndexRune(s, '_')
		if l > 0 {
			s = s[l:]
		}

	}
	return s
}

/*
MQItoString returns a string representation of the MQI #define.
Some of the sets are aggregated, so that "RC" will return something from either the MQRC
or MQRCCF sets. These sets are related and do not overlap values.
All the other sets are coded here directly, rather than rely on the contents of cmqstrc.h
so that we do not try to use new functions when running on older versions of MQ.
*/
func MQItoString(class string, value int) string {
	s := ""
	v := C.MQLONG(value)
	if strings.HasPrefix(class, "MQ_") {
		class = class[3:]
	} else if strings.HasPrefix(class, "MQ") {
		class = class[2:]
	} else if strings.HasPrefix(class, "_") {
		class = class[1:]
	}
	switch class {
	case "BACF":
		s = C.GoString(C.MQBACF_STR(v))

	case "CA":
		s = C.GoString(C.MQCA_STR(v))
		if s == "" {
			s = C.GoString(C.MQCACF_STR(v))
		}
		if s == "" {
			s = C.GoString(C.MQCACH_STR(v))
		}
		if s == "" {
			s = C.GoString(C.MQCAMO_STR(v))
		}

	case "CC":
		s = C.GoString(C.MQCC_STR(v))
	case "CHT":
		s = C.GoString(C.MQCHT_STR(v))
	case "CMD":
		s = C.GoString(C.MQCMD_STR(v))

	case "IA":
		s = C.GoString(C.MQIA_STR(v))
		if s == "" {
			s = C.GoString(C.MQIACF_STR(v))
		}
		if s == "" {
			s = C.GoString(C.MQIACH_STR(v))
		}
		if s == "" {
			s = C.GoString(C.MQIAMO_STR(v))
		}
		if s == "" {
			s = C.GoString(C.MQIAMO64_STR(v))
		}

	case "OT":
		s = C.GoString(C.MQOT_STR(v))

	case "PL":
		s = C.GoString(C.MQPL_STR(v))

	case "RC":
		s = C.GoString(C.MQRC_STR(v))
		if s == "" {
			s = C.GoString(C.MQRCCF_STR(v))
		}

	case "ACTIVE":
		switch v {
		case 0:
			s = "MQACTIVE_NO"
		case 1:
			s = "MQACTIVE_YES"
		default:
			s = ""
		}

	case "ACTP":
		switch v {
		case 0:
			s = "MQACTP_NEW"
		case 1:
			s = "MQACTP_FORWARD"
		case 2:
			s = "MQACTP_REPLY"
		case 3:
			s = "MQACTP_REPORT"
		default:
			s = ""
		}

	case "ACTV":
		switch v {
		case 1:
			s = "MQACTV_DETAIL_LOW"
		case 2:
			s = "MQACTV_DETAIL_MEDIUM"
		case 3:
			s = "MQACTV_DETAIL_HIGH"
		default:
			s = ""
		}

	case "ACT":
		switch v {
		case 1:
			s = "MQACT_FORCE_REMOVE"
		case 2:
			s = "MQACT_ADVANCE_LOG"
		case 3:
			s = "MQACT_COLLECT_STATISTICS"
		case 4:
			s = "MQACT_PUBSUB"
		case 5:
			s = "MQACT_ADD"
		case 6:
			s = "MQACT_REPLACE"
		case 7:
			s = "MQACT_REMOVE"
		case 8:
			s = "MQACT_REMOVEALL"
		case 9:
			s = "MQACT_FAIL"
		case 10:
			s = "MQACT_REDUCE_LOG"
		case 11:
			s = "MQACT_ARCHIVE_LOG"
		default:
			s = ""
		}

	case "ADOPT_CHECK":
		switch v {
		case 0:
			s = "MQADOPT_CHECK_NONE"
		case 1:
			s = "MQADOPT_CHECK_ALL"
		case 2:
			s = "MQADOPT_CHECK_Q_MGR_NAME"
		case 4:
			s = "MQADOPT_CHECK_NET_ADDR"
		case 8:
			s = "MQADOPT_CHECK_CHANNEL_NAME"
		default:
			s = ""
		}

	case "ADOPT_TYPE":
		switch v {
		case 0:
			s = "MQADOPT_TYPE_NO"
		case 1:
			s = "MQADOPT_TYPE_ALL"
		case 2:
			s = "MQADOPT_TYPE_SVR"
		case 4:
			s = "MQADOPT_TYPE_SDR"
		case 8:
			s = "MQADOPT_TYPE_RCVR"
		case 16:
			s = "MQADOPT_TYPE_CLUSRCVR"
		default:
			s = ""
		}

	case "ADPCTX":
		switch v {
		case 0:
			s = "MQADPCTX_NO"
		case 1:
			s = "MQADPCTX_YES"
		default:
			s = ""
		}

	case "AIT":
		switch v {
		case 0:
			s = "MQAIT_ALL"
		case 1:
			s = "MQAIT_CRL_LDAP"
		case 2:
			s = "MQAIT_OCSP"
		case 3:
			s = "MQAIT_IDPW_OS"
		case 4:
			s = "MQAIT_IDPW_LDAP"
		default:
			s = ""
		}

	case "APPL":
		switch v {
		case 0:
			s = "MQAPPL_IMMOVABLE"
		case 1:
			s = "MQAPPL_MOVABLE"
		default:
			s = ""
		}

	case "AS":
		switch v {
		case 0:
			s = "MQAS_NONE"
		case 1:
			s = "MQAS_STARTED"
		case 2:
			s = "MQAS_START_WAIT"
		case 3:
			s = "MQAS_STOPPED"
		case 4:
			s = "MQAS_SUSPENDED"
		case 5:
			s = "MQAS_SUSPENDED_TEMPORARY"
		case 6:
			s = "MQAS_ACTIVE"
		case 7:
			s = "MQAS_INACTIVE"
		default:
			s = ""
		}

	case "AT":
		switch v {
		case -1:
			s = "MQAT_UNKNOWN"
		case 0:
			s = "MQAT_NO_CONTEXT"
		case 1:
			s = "MQAT_CICS"
		case 2:
			s = "MQAT_ZOS"
		case 3:
			s = "MQAT_IMS"
		case 4:
			s = "MQAT_OS2"
		case 5:
			s = "MQAT_DOS"
		case 6:
			s = "MQAT_UNIX"
		case 7:
			s = "MQAT_QMGR"
		case 8:
			s = "MQAT_OS400"
		case 9:
			s = "MQAT_WINDOWS"
		case 10:
			s = "MQAT_CICS_VSE"
		case 11:
			s = "MQAT_WINDOWS_NT"
		case 12:
			s = "MQAT_VMS"
		case 13:
			s = "MQAT_NSK"
		case 14:
			s = "MQAT_VOS"
		case 15:
			s = "MQAT_OPEN_TP1"
		case 18:
			s = "MQAT_VM"
		case 19:
			s = "MQAT_IMS_BRIDGE"
		case 20:
			s = "MQAT_XCF"
		case 21:
			s = "MQAT_CICS_BRIDGE"
		case 22:
			s = "MQAT_NOTES_AGENT"
		case 23:
			s = "MQAT_TPF"
		case 25:
			s = "MQAT_USER"
		case 26:
			s = "MQAT_QMGR_PUBLISH"
		case 28:
			s = "MQAT_JAVA"
		case 29:
			s = "MQAT_DQM"
		case 30:
			s = "MQAT_CHANNEL_INITIATOR"
		case 31:
			s = "MQAT_WLM"
		case 32:
			s = "MQAT_BATCH"
		case 33:
			s = "MQAT_RRS_BATCH"
		case 34:
			s = "MQAT_SIB"
		case 35:
			s = "MQAT_SYSTEM_EXTENSION"
		case 36:
			s = "MQAT_MCAST_PUBLISH"
		case 37:
			s = "MQAT_AMQP"
		default:
			s = ""
		}

	case "AUTHENTICATE":
		switch v {
		case 0:
			s = "MQAUTHENTICATE_OS"
		case 1:
			s = "MQAUTHENTICATE_PAM"
		default:
			s = ""
		}

	case "AUTHOPT":
		switch v {
		case 1:
			s = "MQAUTHOPT_ENTITY_EXPLICIT"
		case 2:
			s = "MQAUTHOPT_ENTITY_SET"
		case 16:
			s = "MQAUTHOPT_NAME_EXPLICIT"
		case 32:
			s = "MQAUTHOPT_NAME_ALL_MATCHING"
		case 64:
			s = "MQAUTHOPT_NAME_AS_WILDCARD"
		case 256:
			s = "MQAUTHOPT_CUMULATIVE"
		case 512:
			s = "MQAUTHOPT_EXCLUDE_TEMP"
		default:
			s = ""
		}

	case "AUTH":
		switch v {
		case -3:
			s = "MQAUTH_ALL_MQI"
		case -2:
			s = "MQAUTH_ALL_ADMIN"
		case -1:
			s = "MQAUTH_ALL"
		case 0:
			s = "MQAUTH_NONE"
		case 1:
			s = "MQAUTH_ALT_USER_AUTHORITY"
		case 2:
			s = "MQAUTH_BROWSE"
		case 3:
			s = "MQAUTH_CHANGE"
		case 4:
			s = "MQAUTH_CLEAR"
		case 5:
			s = "MQAUTH_CONNECT"
		case 6:
			s = "MQAUTH_CREATE"
		case 7:
			s = "MQAUTH_DELETE"
		case 8:
			s = "MQAUTH_DISPLAY"
		case 9:
			s = "MQAUTH_INPUT"
		case 10:
			s = "MQAUTH_INQUIRE"
		case 11:
			s = "MQAUTH_OUTPUT"
		case 12:
			s = "MQAUTH_PASS_ALL_CONTEXT"
		case 13:
			s = "MQAUTH_PASS_IDENTITY_CONTEXT"
		case 14:
			s = "MQAUTH_SET"
		case 15:
			s = "MQAUTH_SET_ALL_CONTEXT"
		case 16:
			s = "MQAUTH_SET_IDENTITY_CONTEXT"
		case 17:
			s = "MQAUTH_CONTROL"
		case 18:
			s = "MQAUTH_CONTROL_EXTENDED"
		case 19:
			s = "MQAUTH_PUBLISH"
		case 20:
			s = "MQAUTH_SUBSCRIBE"
		case 21:
			s = "MQAUTH_RESUME"
		case 22:
			s = "MQAUTH_SYSTEM"
		default:
			s = ""
		}

	case "AUTOCLUS":
		switch v {
		case 0:
			s = "MQAUTOCLUS_TYPE_NONE"
		case 1:
			s = "MQAUTOCLUS_TYPE_UNIFORM"
		default:
			s = ""
		}

	case "AUTO":
		switch v {
		case 0:
			s = "MQAUTO_START_NO"
		case 1:
			s = "MQAUTO_START_YES"
		default:
			s = ""
		}

	case "BALANCED":
		switch v {
		case 0:
			s = "MQBALANCED_NO"
		case 1:
			s = "MQBALANCED_YES"
		case 2:
			s = "MQBALANCED_NOT_APPLICABLE"
		case 3:
			s = "MQBALANCED_UNKNOWN"
		default:
			s = ""
		}

	case "BALSTATE":
		switch v {
		case 0:
			s = "MQBALSTATE_NOT_APPLICABLE"
		case 1:
			s = "MQBALSTATE_LOW"
		case 2:
			s = "MQBALSTATE_OK"
		case 3:
			s = "MQBALSTATE_HIGH"
		case 4:
			s = "MQBALSTATE_UNKNOWN"
		default:
			s = ""
		}

	case "BL":
		switch v {
		case -1:
			s = "MQBL_NULL_TERMINATED"
		default:
			s = ""
		}

	case "BMHO":
		switch v {
		case 0:
			s = "MQBMHO_NONE"
		case 1:
			s = "MQBMHO_DELETE_PROPERTIES"
		default:
			s = ""
		}

	case "BND":
		switch v {
		case 0:
			s = "MQBND_BIND_ON_OPEN"
		case 1:
			s = "MQBND_BIND_NOT_FIXED"
		case 2:
			s = "MQBND_BIND_ON_GROUP"
		default:
			s = ""
		}

	case "BNO_BALTYPE":
		switch v {
		case 0:
			s = "MQBNO_BALTYPE_SIMPLE"
		case 1:
			s = "MQBNO_BALTYPE_REQREP"
		case 65536:
			s = "MQBNO_BALTYPE_RA_MANAGED"
		default:
			s = ""
		}

	case "BNO_OPTIONS":
		switch v {
		case 0:
			s = "MQBNO_OPTIONS_NONE"
		case 1:
			s = "MQBNO_OPTIONS_IGNORE_TRANS"
		default:
			s = ""
		}

	case "BNO_TIMEOUT":
		switch v {
		case -2:
			s = "MQBNO_TIMEOUT_NEVER"
		case -1:
			s = "MQBNO_TIMEOUT_AS_DEFAULT"
		case 0:
			s = "MQBNO_TIMEOUT_IMMEDIATE"
		default:
			s = ""
		}

	case "BO":
		switch v {
		case 0:
			s = "MQBO_NONE"
		default:
			s = ""
		}

	case "BPLOCATION":
		switch v {
		case 0:
			s = "MQBPLOCATION_BELOW"
		case 1:
			s = "MQBPLOCATION_ABOVE"
		case 2:
			s = "MQBPLOCATION_SWITCHING_ABOVE"
		case 3:
			s = "MQBPLOCATION_SWITCHING_BELOW"
		default:
			s = ""
		}

	case "BT":
		switch v {
		case 1:
			s = "MQBT_OTMA"
		default:
			s = ""
		}

	case "CADSD":
		switch v {
		case 0:
			s = "MQCADSD_NONE"
		case 1:
			s = "MQCADSD_SEND"
		case 16:
			s = "MQCADSD_RECV"
		case 256:
			s = "MQCADSD_MSGFORMAT"
		default:
			s = ""
		}

	case "CAFTY":
		switch v {
		case 0:
			s = "MQCAFTY_NONE"
		case 1:
			s = "MQCAFTY_PREFERRED"
		default:
			s = ""
		}

	case "CAP":
		switch v {
		case 0:
			s = "MQCAP_NOT_SUPPORTED"
		case 1:
			s = "MQCAP_SUPPORTED"
		case 2:
			s = "MQCAP_EXPIRED"
		default:
			s = ""
		}

	case "CAUT":
		switch v {
		case 0:
			s = "MQCAUT_ALL"
		case 1:
			s = "MQCAUT_BLOCKUSER"
		case 2:
			s = "MQCAUT_BLOCKADDR"
		case 3:
			s = "MQCAUT_SSLPEERMAP"
		case 4:
			s = "MQCAUT_ADDRESSMAP"
		case 5:
			s = "MQCAUT_USERMAP"
		case 6:
			s = "MQCAUT_QMGRMAP"
		default:
			s = ""
		}

	case "CBCF":
		switch v {
		case 0:
			s = "MQCBCF_NONE"
		case 1:
			s = "MQCBCF_READA_BUFFER_EMPTY"
		default:
			s = ""
		}

	case "CBCT":
		switch v {
		case 1:
			s = "MQCBCT_START_CALL"
		case 2:
			s = "MQCBCT_STOP_CALL"
		case 3:
			s = "MQCBCT_REGISTER_CALL"
		case 4:
			s = "MQCBCT_DEREGISTER_CALL"
		case 5:
			s = "MQCBCT_EVENT_CALL"
		case 6:
			s = "MQCBCT_MSG_REMOVED"
		case 7:
			s = "MQCBCT_MSG_NOT_REMOVED"
		case 8:
			s = "MQCBCT_MC_EVENT_CALL"
		default:
			s = ""
		}

	case "CBDO":
		switch v {
		case 0:
			s = "MQCBDO_NONE"
		case 1:
			s = "MQCBDO_START_CALL"
		case 4:
			s = "MQCBDO_STOP_CALL"
		case 256:
			s = "MQCBDO_REGISTER_CALL"
		case 512:
			s = "MQCBDO_DEREGISTER_CALL"
		case 8192:
			s = "MQCBDO_FAIL_IF_QUIESCING"
		case 16384:
			s = "MQCBDO_EVENT_CALL"
		case 32768:
			s = "MQCBDO_MC_EVENT_CALL"
		default:
			s = ""
		}

	case "CBD":
		switch v {
		case -1:
			s = "MQCBD_FULL_MSG_LENGTH"
		default:
			s = ""
		}

	case "CBO":
		switch v {
		case 0:
			s = "MQCBO_NONE"
		case 1:
			s = "MQCBO_ADMIN_BAG"
		case 2:
			s = "MQCBO_LIST_FORM_ALLOWED"
		case 4:
			s = "MQCBO_REORDER_AS_REQUIRED"
		case 8:
			s = "MQCBO_CHECK_SELECTORS"
		case 16:
			s = "MQCBO_COMMAND_BAG"
		case 32:
			s = "MQCBO_SYSTEM_BAG"
		case 64:
			s = "MQCBO_GROUP_BAG"
		default:
			s = ""
		}

	case "CBT":
		switch v {
		case 1:
			s = "MQCBT_MESSAGE_CONSUMER"
		case 2:
			s = "MQCBT_EVENT_HANDLER"
		default:
			s = ""
		}

	case "CCSI":
		switch v {
		case -4:
			s = "MQCCSI_AS_PUBLISHED"
		case -3:
			s = "MQCCSI_APPL"
		case -2:
			s = "MQCCSI_INHERIT"
		case -1:
			s = "MQCCSI_EMBEDDED"
		case 0:
			s = "MQCCSI_DEFAULT"
		default:
			s = ""
		}

	case "CCT":
		switch v {
		case 0:
			s = "MQCCT_NO"
		case 1:
			s = "MQCCT_YES"
		default:
			s = ""
		}

	case "CDC":
		switch v {
		case 0:
			s = "MQCDC_NO_SENDER_CONVERSION"
		case 1:
			s = "MQCDC_SENDER_CONVERSION"
		default:
			s = ""
		}

	case "CEX":
		switch v {
		case -2:
			s = "MQCEX_AS_PARENT"
		case -1:
			s = "MQCEX_NOLIMIT"
		default:
			s = ""
		}

	case "CFACCESS":
		switch v {
		case 0:
			s = "MQCFACCESS_ENABLED"
		case 1:
			s = "MQCFACCESS_SUSPENDED"
		case 2:
			s = "MQCFACCESS_DISABLED"
		default:
			s = ""
		}

	case "CFCONLOS":
		switch v {
		case 0:
			s = "MQCFCONLOS_TERMINATE"
		case 1:
			s = "MQCFCONLOS_TOLERATE"
		case 2:
			s = "MQCFCONLOS_ASQMGR"
		default:
			s = ""
		}

	case "CFC":
		switch v {
		case 0:
			s = "MQCFC_NOT_LAST"
		case 1:
			s = "MQCFC_LAST"
		default:
			s = ""
		}

	case "CFOFFLD":
		switch v {
		case 0:
			s = "MQCFOFFLD_NONE"
		case 1:
			s = "MQCFOFFLD_SMDS"
		case 2:
			s = "MQCFOFFLD_DB2"
		case 3:
			s = "MQCFOFFLD_BOTH"
		default:
			s = ""
		}

	case "CFOP":
		switch v {
		case 1:
			s = "MQCFOP_LESS"
		case 2:
			s = "MQCFOP_EQUAL"
		case 3:
			s = "MQCFOP_NOT_GREATER"
		case 4:
			s = "MQCFOP_GREATER"
		case 5:
			s = "MQCFOP_NOT_EQUAL"
		case 6:
			s = "MQCFOP_NOT_LESS"
		case 10:
			s = "MQCFOP_CONTAINS"
		case 13:
			s = "MQCFOP_EXCLUDES"
		case 18:
			s = "MQCFOP_LIKE"
		case 21:
			s = "MQCFOP_NOT_LIKE"
		case 26:
			s = "MQCFOP_CONTAINS_GEN"
		case 29:
			s = "MQCFOP_EXCLUDES_GEN"
		default:
			s = ""
		}

	case "CFO_REFRESH":
		switch v {
		case 0:
			s = "MQCFO_REFRESH_REPOSITORY_NO"
		case 1:
			s = "MQCFO_REFRESH_REPOSITORY_YES"
		default:
			s = ""
		}

	case "CFO_REMOVE":
		switch v {
		case 0:
			s = "MQCFO_REMOVE_QUEUES_NO"
		case 1:
			s = "MQCFO_REMOVE_QUEUES_YES"
		default:
			s = ""
		}

	case "CFR":
		switch v {
		case 0:
			s = "MQCFR_NO"
		case 1:
			s = "MQCFR_YES"
		default:
			s = ""
		}

	case "CFSTATUS":
		switch v {
		case 0:
			s = "MQCFSTATUS_NOT_FOUND"
		case 1:
			s = "MQCFSTATUS_ACTIVE"
		case 2:
			s = "MQCFSTATUS_IN_RECOVER"
		case 3:
			s = "MQCFSTATUS_IN_BACKUP"
		case 4:
			s = "MQCFSTATUS_FAILED"
		case 5:
			s = "MQCFSTATUS_NONE"
		case 6:
			s = "MQCFSTATUS_UNKNOWN"
		case 7:
			s = "MQCFSTATUS_RECOVERED"
		case 8:
			s = "MQCFSTATUS_EMPTY"
		case 9:
			s = "MQCFSTATUS_NEW"
		case 20:
			s = "MQCFSTATUS_ADMIN_INCOMPLETE"
		case 21:
			s = "MQCFSTATUS_NEVER_USED"
		case 22:
			s = "MQCFSTATUS_NO_BACKUP"
		case 23:
			s = "MQCFSTATUS_NOT_FAILED"
		case 24:
			s = "MQCFSTATUS_NOT_RECOVERABLE"
		case 25:
			s = "MQCFSTATUS_XES_ERROR"
		default:
			s = ""
		}

	case "CFTYPE":
		switch v {
		case 0:
			s = "MQCFTYPE_APPL"
		case 1:
			s = "MQCFTYPE_ADMIN"
		default:
			s = ""
		}

	case "CFT":
		switch v {
		case 0:
			s = "MQCFT_NONE"
		case 1:
			s = "MQCFT_COMMAND"
		case 2:
			s = "MQCFT_RESPONSE"
		case 3:
			s = "MQCFT_INTEGER"
		case 4:
			s = "MQCFT_STRING"
		case 5:
			s = "MQCFT_INTEGER_LIST"
		case 6:
			s = "MQCFT_STRING_LIST"
		case 7:
			s = "MQCFT_EVENT"
		case 8:
			s = "MQCFT_USER"
		case 9:
			s = "MQCFT_BYTE_STRING"
		case 10:
			s = "MQCFT_TRACE_ROUTE"
		case 12:
			s = "MQCFT_REPORT"
		case 13:
			s = "MQCFT_INTEGER_FILTER"
		case 14:
			s = "MQCFT_STRING_FILTER"
		case 15:
			s = "MQCFT_BYTE_STRING_FILTER"
		case 16:
			s = "MQCFT_COMMAND_XR"
		case 17:
			s = "MQCFT_XR_MSG"
		case 18:
			s = "MQCFT_XR_ITEM"
		case 19:
			s = "MQCFT_XR_SUMMARY"
		case 20:
			s = "MQCFT_GROUP"
		case 21:
			s = "MQCFT_STATISTICS"
		case 22:
			s = "MQCFT_ACCOUNTING"
		case 23:
			s = "MQCFT_INTEGER64"
		case 25:
			s = "MQCFT_INTEGER64_LIST"
		case 26:
			s = "MQCFT_APP_ACTIVITY"
		case 27:
			s = "MQCFT_STATUS"
		default:
			s = ""
		}

	case "CF":
		switch v {
		case 0:
			s = "MQCF_NONE"
		case 1:
			s = "MQCF_DIST_LISTS"
		default:
			s = ""
		}

	case "CGWI":
		switch v {
		case -2:
			s = "MQCGWI_DEFAULT"
		default:
			s = ""
		}

	case "CHAD":
		switch v {
		case 0:
			s = "MQCHAD_DISABLED"
		case 1:
			s = "MQCHAD_ENABLED"
		default:
			s = ""
		}

	case "CHIDS":
		switch v {
		case 0:
			s = "MQCHIDS_NOT_INDOUBT"
		case 1:
			s = "MQCHIDS_INDOUBT"
		default:
			s = ""
		}

	case "CHK":
		switch v {
		case 0:
			s = "MQCHK_OPTIONAL"
		case 1:
			s = "MQCHK_NONE"
		case 2:
			s = "MQCHK_REQUIRED_ADMIN"
		case 3:
			s = "MQCHK_REQUIRED"
		case 4:
			s = "MQCHK_AS_Q_MGR"
		default:
			s = ""
		}

	case "CHLA":
		switch v {
		case 0:
			s = "MQCHLA_DISABLED"
		case 1:
			s = "MQCHLA_ENABLED"
		default:
			s = ""
		}

	case "CHLD":
		switch v {
		case -1:
			s = "MQCHLD_ALL"
		case 1:
			s = "MQCHLD_DEFAULT"
		case 2:
			s = "MQCHLD_SHARED"
		case 4:
			s = "MQCHLD_PRIVATE"
		case 5:
			s = "MQCHLD_FIXSHARED"
		default:
			s = ""
		}

	case "CHRR":
		switch v {
		case 0:
			s = "MQCHRR_RESET_NOT_REQUESTED"
		default:
			s = ""
		}

	case "CHSH":
		switch v {
		case 0:
			s = "MQCHSH_RESTART_NO"
		case 1:
			s = "MQCHSH_RESTART_YES"
		default:
			s = ""
		}

	case "CHSR":
		switch v {
		case 0:
			s = "MQCHSR_STOP_NOT_REQUESTED"
		case 1:
			s = "MQCHSR_STOP_REQUESTED"
		default:
			s = ""
		}

	case "CHSSTATE":
		switch v {
		case 0:
			s = "MQCHSSTATE_OTHER"
		case 100:
			s = "MQCHSSTATE_END_OF_BATCH"
		case 200:
			s = "MQCHSSTATE_SENDING"
		case 300:
			s = "MQCHSSTATE_RECEIVING"
		case 400:
			s = "MQCHSSTATE_SERIALIZING"
		case 500:
			s = "MQCHSSTATE_RESYNCHING"
		case 600:
			s = "MQCHSSTATE_HEARTBEATING"
		case 700:
			s = "MQCHSSTATE_IN_SCYEXIT"
		case 800:
			s = "MQCHSSTATE_IN_RCVEXIT"
		case 900:
			s = "MQCHSSTATE_IN_SENDEXIT"
		case 1000:
			s = "MQCHSSTATE_IN_MSGEXIT"
		case 1100:
			s = "MQCHSSTATE_IN_MREXIT"
		case 1200:
			s = "MQCHSSTATE_IN_CHADEXIT"
		case 1250:
			s = "MQCHSSTATE_NET_CONNECTING"
		case 1300:
			s = "MQCHSSTATE_SSL_HANDSHAKING"
		case 1400:
			s = "MQCHSSTATE_NAME_SERVER"
		case 1500:
			s = "MQCHSSTATE_IN_MQPUT"
		case 1600:
			s = "MQCHSSTATE_IN_MQGET"
		case 1700:
			s = "MQCHSSTATE_IN_MQI_CALL"
		case 1800:
			s = "MQCHSSTATE_COMPRESSING"
		default:
			s = ""
		}

	case "CHS":
		switch v {
		case 0:
			s = "MQCHS_INACTIVE"
		case 1:
			s = "MQCHS_BINDING"
		case 2:
			s = "MQCHS_STARTING"
		case 3:
			s = "MQCHS_RUNNING"
		case 4:
			s = "MQCHS_STOPPING"
		case 5:
			s = "MQCHS_RETRYING"
		case 6:
			s = "MQCHS_STOPPED"
		case 7:
			s = "MQCHS_REQUESTING"
		case 8:
			s = "MQCHS_PAUSED"
		case 9:
			s = "MQCHS_DISCONNECTED"
		case 13:
			s = "MQCHS_INITIALIZING"
		case 14:
			s = "MQCHS_SWITCHING"
		default:
			s = ""
		}

	case "CHTAB":
		switch v {
		case 1:
			s = "MQCHTAB_Q_MGR"
		case 2:
			s = "MQCHTAB_CLNTCONN"
		default:
			s = ""
		}

	case "CIH":
		switch v {
		case 0:
			s = "MQCIH_NONE"
		case 1:
			s = "MQCIH_PASS_EXPIRATION"
		case 2:
			s = "MQCIH_REPLY_WITHOUT_NULLS"
		case 4:
			s = "MQCIH_SYNC_ON_RETURN"
		default:
			s = ""
		}

	case "CIT":
		switch v {
		case 1:
			s = "MQCIT_MULTICAST"
		default:
			s = ""
		}

	case "CLCT":
		switch v {
		case 0:
			s = "MQCLCT_STATIC"
		case 1:
			s = "MQCLCT_DYNAMIC"
		default:
			s = ""
		}

	case "CLROUTE":
		switch v {
		case 0:
			s = "MQCLROUTE_DIRECT"
		case 1:
			s = "MQCLROUTE_TOPIC_HOST"
		case 2:
			s = "MQCLROUTE_NONE"
		default:
			s = ""
		}

	case "CLRS":
		switch v {
		case 1:
			s = "MQCLRS_LOCAL"
		case 2:
			s = "MQCLRS_GLOBAL"
		default:
			s = ""
		}

	case "CLRT":
		switch v {
		case 1:
			s = "MQCLRT_RETAINED"
		default:
			s = ""
		}

	case "CLST":
		switch v {
		case 0:
			s = "MQCLST_ACTIVE"
		case 1:
			s = "MQCLST_PENDING"
		case 2:
			s = "MQCLST_INVALID"
		case 3:
			s = "MQCLST_ERROR"
		default:
			s = ""
		}

	case "CLT":
		switch v {
		case 1:
			s = "MQCLT_PROGRAM"
		case 2:
			s = "MQCLT_TRANSACTION"
		default:
			s = ""
		}

	case "CLWL":
		switch v {
		case -3:
			s = "MQCLWL_USEQ_AS_Q_MGR"
		case 0:
			s = "MQCLWL_USEQ_LOCAL"
		case 1:
			s = "MQCLWL_USEQ_ANY"
		default:
			s = ""
		}

	case "CLXQ":
		switch v {
		case 0:
			s = "MQCLXQ_SCTQ"
		case 1:
			s = "MQCLXQ_CHANNEL"
		default:
			s = ""
		}

	case "CMDI":
		switch v {
		case 1:
			s = "MQCMDI_CMDSCOPE_ACCEPTED"
		case 2:
			s = "MQCMDI_CMDSCOPE_GENERATED"
		case 3:
			s = "MQCMDI_CMDSCOPE_COMPLETED"
		case 4:
			s = "MQCMDI_QSG_DISP_COMPLETED"
		case 5:
			s = "MQCMDI_COMMAND_ACCEPTED"
		case 6:
			s = "MQCMDI_CLUSTER_REQUEST_QUEUED"
		case 7:
			s = "MQCMDI_CHANNEL_INIT_STARTED"
		case 11:
			s = "MQCMDI_RECOVER_STARTED"
		case 12:
			s = "MQCMDI_BACKUP_STARTED"
		case 13:
			s = "MQCMDI_RECOVER_COMPLETED"
		case 14:
			s = "MQCMDI_SEC_TIMER_ZERO"
		case 16:
			s = "MQCMDI_REFRESH_CONFIGURATION"
		case 17:
			s = "MQCMDI_SEC_SIGNOFF_ERROR"
		case 18:
			s = "MQCMDI_IMS_BRIDGE_SUSPENDED"
		case 19:
			s = "MQCMDI_DB2_SUSPENDED"
		case 20:
			s = "MQCMDI_DB2_OBSOLETE_MSGS"
		case 21:
			s = "MQCMDI_SEC_UPPERCASE"
		case 22:
			s = "MQCMDI_SEC_MIXEDCASE"
		default:
			s = ""
		}

	case "CMDL":
		switch v {
		case 100:
			s = "MQCMDL_LEVEL_1"
		case 101:
			s = "MQCMDL_LEVEL_101"
		case 110:
			s = "MQCMDL_LEVEL_110"
		case 114:
			s = "MQCMDL_LEVEL_114"
		case 120:
			s = "MQCMDL_LEVEL_120"
		case 200:
			s = "MQCMDL_LEVEL_200"
		case 201:
			s = "MQCMDL_LEVEL_201"
		case 210:
			s = "MQCMDL_LEVEL_210"
		case 211:
			s = "MQCMDL_LEVEL_211"
		case 220:
			s = "MQCMDL_LEVEL_220"
		case 221:
			s = "MQCMDL_LEVEL_221"
		case 230:
			s = "MQCMDL_LEVEL_230"
		case 320:
			s = "MQCMDL_LEVEL_320"
		case 420:
			s = "MQCMDL_LEVEL_420"
		case 500:
			s = "MQCMDL_LEVEL_500"
		case 510:
			s = "MQCMDL_LEVEL_510"
		case 520:
			s = "MQCMDL_LEVEL_520"
		case 530:
			s = "MQCMDL_LEVEL_530"
		case 531:
			s = "MQCMDL_LEVEL_531"
		case 600:
			s = "MQCMDL_LEVEL_600"
		case 700:
			s = "MQCMDL_LEVEL_700"
		case 701:
			s = "MQCMDL_LEVEL_701"
		case 710:
			s = "MQCMDL_LEVEL_710"
		case 711:
			s = "MQCMDL_LEVEL_711"
		case 750:
			s = "MQCMDL_LEVEL_750"
		case 800:
			s = "MQCMDL_LEVEL_800"
		case 801:
			s = "MQCMDL_LEVEL_801"
		case 802:
			s = "MQCMDL_LEVEL_802"
		case 900:
			s = "MQCMDL_LEVEL_900"
		case 901:
			s = "MQCMDL_LEVEL_901"
		case 902:
			s = "MQCMDL_LEVEL_902"
		case 903:
			s = "MQCMDL_LEVEL_903"
		case 904:
			s = "MQCMDL_LEVEL_904"
		case 905:
			s = "MQCMDL_LEVEL_905"
		case 910:
			s = "MQCMDL_LEVEL_910"
		case 911:
			s = "MQCMDL_LEVEL_911"
		case 912:
			s = "MQCMDL_LEVEL_912"
		case 913:
			s = "MQCMDL_LEVEL_913"
		case 914:
			s = "MQCMDL_LEVEL_914"
		case 915:
			s = "MQCMDL_LEVEL_915"
		case 920:
			s = "MQCMDL_LEVEL_920"
		case 921:
			s = "MQCMDL_LEVEL_921"
		case 922:
			s = "MQCMDL_LEVEL_922"
		case 923:
			s = "MQCMDL_LEVEL_923"
		case 924:
			s = "MQCMDL_LEVEL_924"
		case 925:
			s = "MQCMDL_LEVEL_925"
		case 930:
			s = "MQCMDL_LEVEL_930"
		case 931:
			s = "MQCMDL_LEVEL_931"
		case 932:
			s = "MQCMDL_LEVEL_932"
		case 933:
			s = "MQCMDL_LEVEL_933"
		case 934:
			s = "MQCMDL_LEVEL_934"
		case 935:
			s = "MQCMDL_LEVEL_935"
		case 940:
			s = "MQCMDL_LEVEL_940"
		case 941:
			s = "MQCMDL_LEVEL_941"
		case 942:
			s = "MQCMDL_LEVEL_942"
		default:
			s = ""
		}

	case "CMHO":
		switch v {
		case 0:
			s = "MQCMHO_NONE"
		case 1:
			s = "MQCMHO_NO_VALIDATION"
		case 2:
			s = "MQCMHO_VALIDATE"
		default:
			s = ""
		}

	case "CNO":
		switch v {
		case 0:
			s = "MQCNO_NONE"
		case 1:
			s = "MQCNO_FASTPATH_BINDING"
		case 2:
			s = "MQCNO_SERIALIZE_CONN_TAG_Q_MGR"
		case 4:
			s = "MQCNO_SERIALIZE_CONN_TAG_QSG"
		case 8:
			s = "MQCNO_RESTRICT_CONN_TAG_Q_MGR"
		case 16:
			s = "MQCNO_RESTRICT_CONN_TAG_QSG"
		case 32:
			s = "MQCNO_HANDLE_SHARE_NONE"
		case 64:
			s = "MQCNO_HANDLE_SHARE_BLOCK"
		case 128:
			s = "MQCNO_HANDLE_SHARE_NO_BLOCK"
		case 256:
			s = "MQCNO_SHARED_BINDING"
		case 512:
			s = "MQCNO_ISOLATED_BINDING"
		case 1024:
			s = "MQCNO_LOCAL_BINDING"
		case 2048:
			s = "MQCNO_CLIENT_BINDING"
		case 4096:
			s = "MQCNO_ACCOUNTING_MQI_ENABLED"
		case 8192:
			s = "MQCNO_ACCOUNTING_MQI_DISABLED"
		case 16384:
			s = "MQCNO_ACCOUNTING_Q_ENABLED"
		case 32768:
			s = "MQCNO_ACCOUNTING_Q_DISABLED"
		case 65536:
			s = "MQCNO_NO_CONV_SHARING"
		case 262144:
			s = "MQCNO_ALL_CONVS_SHARE"
		case 524288:
			s = "MQCNO_CD_FOR_OUTPUT_ONLY"
		case 1048576:
			s = "MQCNO_USE_CD_SELECTION"
		case 2097152:
			s = "MQCNO_GENERATE_CONN_TAG"
		case 16777216:
			s = "MQCNO_RECONNECT"
		case 33554432:
			s = "MQCNO_RECONNECT_DISABLED"
		case 67108864:
			s = "MQCNO_RECONNECT_Q_MGR"
		case 134217728:
			s = "MQCNO_ACTIVITY_TRACE_ENABLED"
		case 268435456:
			s = "MQCNO_ACTIVITY_TRACE_DISABLED"
		default:
			s = ""
		}

	case "CODL":
		switch v {
		case -1:
			s = "MQCODL_AS_INPUT"
		default:
			s = ""
		}

	case "COMPRESS":
		switch v {
		case -1:
			s = "MQCOMPRESS_NOT_AVAILABLE"
		case 0:
			s = "MQCOMPRESS_NONE"
		case 1:
			s = "MQCOMPRESS_RLE"
		case 2:
			s = "MQCOMPRESS_ZLIBFAST"
		case 4:
			s = "MQCOMPRESS_ZLIBHIGH"
		case 8:
			s = "MQCOMPRESS_SYSTEM"
		case 16:
			s = "MQCOMPRESS_LZ4FAST"
		case 32:
			s = "MQCOMPRESS_LZ4HIGH"
		case 268435455:
			s = "MQCOMPRESS_ANY"
		default:
			s = ""
		}

	case "COPY":
		switch v {
		case 0:
			s = "MQCOPY_NONE"
		case 1:
			s = "MQCOPY_ALL"
		case 2:
			s = "MQCOPY_FORWARD"
		case 4:
			s = "MQCOPY_PUBLISH"
		case 8:
			s = "MQCOPY_REPLY"
		case 16:
			s = "MQCOPY_REPORT"
		case 22:
			s = "MQCOPY_DEFAULT"
		default:
			s = ""
		}

	case "CO":
		switch v {
		case 0:
			s = "MQCO_NONE"
		case 1:
			s = "MQCO_DELETE"
		case 2:
			s = "MQCO_DELETE_PURGE"
		case 4:
			s = "MQCO_KEEP_SUB"
		case 8:
			s = "MQCO_REMOVE_SUB"
		case 32:
			s = "MQCO_QUIESCE"
		default:
			s = ""
		}

	case "CQT":
		switch v {
		case 1:
			s = "MQCQT_LOCAL_Q"
		case 2:
			s = "MQCQT_ALIAS_Q"
		case 3:
			s = "MQCQT_REMOTE_Q"
		case 4:
			s = "MQCQT_Q_MGR_ALIAS"
		default:
			s = ""
		}

	case "CRC":
		switch v {
		case 0:
			s = "MQCRC_OK"
		case 1:
			s = "MQCRC_CICS_EXEC_ERROR"
		case 2:
			s = "MQCRC_MQ_API_ERROR"
		case 3:
			s = "MQCRC_BRIDGE_ERROR"
		case 4:
			s = "MQCRC_BRIDGE_ABEND"
		case 5:
			s = "MQCRC_APPLICATION_ABEND"
		case 6:
			s = "MQCRC_SECURITY_ERROR"
		case 7:
			s = "MQCRC_PROGRAM_NOT_AVAILABLE"
		case 8:
			s = "MQCRC_BRIDGE_TIMEOUT"
		case 9:
			s = "MQCRC_TRANSID_NOT_AVAILABLE"
		default:
			s = ""
		}

	case "CSP":
		switch v {
		case 0:
			s = "MQCSP_AUTH_NONE"
		case 1:
			s = "MQCSP_AUTH_USER_ID_AND_PWD"
		case 2:
			s = "MQCSP_AUTH_ID_TOKEN"
		default:
			s = ""
		}

	case "CSRV_CONVERT":
		switch v {
		case 0:
			s = "MQCSRV_CONVERT_NO"
		case 1:
			s = "MQCSRV_CONVERT_YES"
		default:
			s = ""
		}

	case "CSRV_DLQ":
		switch v {
		case 0:
			s = "MQCSRV_DLQ_NO"
		case 1:
			s = "MQCSRV_DLQ_YES"
		default:
			s = ""
		}

	case "CS":
		switch v {
		case 0:
			s = "MQCS_NONE"
		case 1:
			s = "MQCS_SUSPENDED_TEMPORARY"
		case 2:
			s = "MQCS_SUSPENDED_USER_ACTION"
		case 3:
			s = "MQCS_SUSPENDED"
		case 4:
			s = "MQCS_STOPPED"
		default:
			s = ""
		}

	case "CTES":
		switch v {
		case 0:
			s = "MQCTES_NOSYNC"
		case 256:
			s = "MQCTES_COMMIT"
		case 4352:
			s = "MQCTES_BACKOUT"
		case 65536:
			s = "MQCTES_ENDTASK"
		default:
			s = ""
		}

	case "CTLO":
		switch v {
		case 0:
			s = "MQCTLO_NONE"
		case 1:
			s = "MQCTLO_THREAD_AFFINITY"
		case 8192:
			s = "MQCTLO_FAIL_IF_QUIESCING"
		default:
			s = ""
		}

	case "CUOWC":
		switch v {
		case 16:
			s = "MQCUOWC_MIDDLE"
		case 256:
			s = "MQCUOWC_COMMIT"
		case 273:
			s = "MQCUOWC_ONLY"
		case 4352:
			s = "MQCUOWC_BACKOUT"
		case 65536:
			s = "MQCUOWC_CONTINUE"
		default:
			s = ""
		}

	case "DCC":
		switch v {
		case 0:
			s = "MQDCC_NONE"
		case 1:
			s = "MQDCC_DEFAULT_CONVERSION"
		case 2:
			s = "MQDCC_FILL_TARGET_BUFFER"
		case 4:
			s = "MQDCC_INT_DEFAULT_CONVERSION"
		case 16:
			s = "MQDCC_SOURCE_ENC_NORMAL"
		case 32:
			s = "MQDCC_SOURCE_ENC_REVERSED"
		case 240:
			s = "MQDCC_SOURCE_ENC_MASK"
		case 256:
			s = "MQDCC_TARGET_ENC_NORMAL"
		case 512:
			s = "MQDCC_TARGET_ENC_REVERSED"
		case 3840:
			s = "MQDCC_TARGET_ENC_MASK"
		default:
			s = ""
		}

	case "DC":
		switch v {
		case 1:
			s = "MQDC_MANAGED"
		case 2:
			s = "MQDC_PROVIDED"
		default:
			s = ""
		}

	case "DELO":
		switch v {
		case 0:
			s = "MQDELO_NONE"
		case 4:
			s = "MQDELO_LOCAL"
		default:
			s = ""
		}

	case "DHF":
		switch v {
		case 0:
			s = "MQDHF_NONE"
		case 1:
			s = "MQDHF_NEW_MSG_IDS"
		default:
			s = ""
		}

	case "DISCONNECT":
		switch v {
		case 0:
			s = "MQDISCONNECT_NORMAL"
		case 1:
			s = "MQDISCONNECT_IMPLICIT"
		case 2:
			s = "MQDISCONNECT_Q_MGR"
		default:
			s = ""
		}

	case "DLV":
		switch v {
		case 0:
			s = "MQDLV_AS_PARENT"
		case 1:
			s = "MQDLV_ALL"
		case 2:
			s = "MQDLV_ALL_DUR"
		case 3:
			s = "MQDLV_ALL_AVAIL"
		default:
			s = ""
		}

	case "DL":
		switch v {
		case 0:
			s = "MQDL_NOT_SUPPORTED"
		case 1:
			s = "MQDL_SUPPORTED"
		default:
			s = ""
		}

	case "DMHO":
		switch v {
		case 0:
			s = "MQDMHO_NONE"
		default:
			s = ""
		}

	case "DMPO":
		switch v {
		case 0:
			s = "MQDMPO_NONE"
		case 1:
			s = "MQDMPO_DEL_PROP_UNDER_CURSOR"
		default:
			s = ""
		}

	case "DNSWLM":
		switch v {
		case 0:
			s = "MQDNSWLM_NO"
		case 1:
			s = "MQDNSWLM_YES"
		default:
			s = ""
		}

	case "DOPT":
		switch v {
		case 0:
			s = "MQDOPT_RESOLVED"
		case 1:
			s = "MQDOPT_DEFINED"
		default:
			s = ""
		}

	case "DSB":
		switch v {
		case 0:
			s = "MQDSB_DEFAULT"
		case 1:
			s = "MQDSB_8K"
		case 2:
			s = "MQDSB_16K"
		case 3:
			s = "MQDSB_32K"
		case 4:
			s = "MQDSB_64K"
		case 5:
			s = "MQDSB_128K"
		case 6:
			s = "MQDSB_256K"
		case 7:
			s = "MQDSB_512K"
		case 8:
			s = "MQDSB_1M"
		default:
			s = ""
		}

	case "DSE":
		switch v {
		case 0:
			s = "MQDSE_DEFAULT"
		case 1:
			s = "MQDSE_YES"
		case 2:
			s = "MQDSE_NO"
		default:
			s = ""
		}

	case "EC":
		switch v {
		case 2:
			s = "MQEC_MSG_ARRIVED"
		case 3:
			s = "MQEC_WAIT_INTERVAL_EXPIRED"
		case 4:
			s = "MQEC_WAIT_CANCELED"
		case 5:
			s = "MQEC_Q_MGR_QUIESCING"
		case 6:
			s = "MQEC_CONNECTION_QUIESCING"
		default:
			s = ""
		}

	case "EI":
		switch v {
		case -1:
			s = "MQEI_UNLIMITED"
		default:
			s = ""
		}

	case "ENC":
		switch v {
		case -4096:
			s = "MQENC_RESERVED_MASK"
		case -1:
			s = "MQENC_AS_PUBLISHED"
		case 1:
			s = "MQENC_INTEGER_NORMAL"
		case 2:
			s = "MQENC_INTEGER_REVERSED"
		case 15:
			s = "MQENC_INTEGER_MASK"
		case 16:
			s = "MQENC_DECIMAL_NORMAL"
		case 32:
			s = "MQENC_DECIMAL_REVERSED"
		case 240:
			s = "MQENC_DECIMAL_MASK"
		case 256:
			s = "MQENC_FLOAT_IEEE_NORMAL"
		case 273:
			s = "MQENC_NORMAL"
		case 512:
			s = "MQENC_FLOAT_IEEE_REVERSED"
		case 546:
			s = "MQENC_REVERSED"
		case 768:
			s = "MQENC_FLOAT_S390"
		case 785:
			s = "MQENC_S390"
		case 1024:
			s = "MQENC_FLOAT_TNS"
		case 1041:
			s = "MQENC_TNS"
		case 3840:
			s = "MQENC_FLOAT_MASK"
		default:
			s = ""
		}

	case "EPH":
		switch v {
		case 0:
			s = "MQEPH_NONE"
		case 1:
			s = "MQEPH_CCSID_EMBEDDED"
		default:
			s = ""
		}

	case "ET":
		switch v {
		case 1:
			s = "MQET_MQSC"
		default:
			s = ""
		}

	case "EVO":
		switch v {
		case 0:
			s = "MQEVO_OTHER"
		case 1:
			s = "MQEVO_CONSOLE"
		case 2:
			s = "MQEVO_INIT"
		case 3:
			s = "MQEVO_MSG"
		case 4:
			s = "MQEVO_MQSET"
		case 5:
			s = "MQEVO_INTERNAL"
		case 6:
			s = "MQEVO_MQSUB"
		case 7:
			s = "MQEVO_CTLMSG"
		case 8:
			s = "MQEVO_REST"
		default:
			s = ""
		}

	case "EVR":
		switch v {
		case 0:
			s = "MQEVR_DISABLED"
		case 1:
			s = "MQEVR_ENABLED"
		case 2:
			s = "MQEVR_EXCEPTION"
		case 3:
			s = "MQEVR_NO_DISPLAY"
		case 4:
			s = "MQEVR_API_ONLY"
		case 5:
			s = "MQEVR_ADMIN_ONLY"
		case 6:
			s = "MQEVR_USER_ONLY"
		default:
			s = ""
		}

	case "EXPI":
		switch v {
		case 0:
			s = "MQEXPI_OFF"
		default:
			s = ""
		}

	case "EXTATTRS":
		switch v {
		case 0:
			s = "MQEXTATTRS_ALL"
		case 1:
			s = "MQEXTATTRS_NONDEF"
		default:
			s = ""
		}

	case "EXT":
		switch v {
		case 0:
			s = "MQEXT_ALL"
		case 1:
			s = "MQEXT_OBJECT"
		case 2:
			s = "MQEXT_AUTHORITY"
		default:
			s = ""
		}

	case "FB":
		switch v {
		case 0:
			s = "MQFB_NONE"
		case 256:
			s = "MQFB_QUIT"
		case 258:
			s = "MQFB_EXPIRATION"
		case 259:
			s = "MQFB_COA"
		case 260:
			s = "MQFB_COD"
		case 262:
			s = "MQFB_CHANNEL_COMPLETED"
		case 263:
			s = "MQFB_CHANNEL_FAIL_RETRY"
		case 264:
			s = "MQFB_CHANNEL_FAIL"
		case 265:
			s = "MQFB_APPL_CANNOT_BE_STARTED"
		case 266:
			s = "MQFB_TM_ERROR"
		case 267:
			s = "MQFB_APPL_TYPE_ERROR"
		case 268:
			s = "MQFB_STOPPED_BY_MSG_EXIT"
		case 269:
			s = "MQFB_ACTIVITY"
		case 271:
			s = "MQFB_XMIT_Q_MSG_ERROR"
		case 275:
			s = "MQFB_PAN"
		case 276:
			s = "MQFB_NAN"
		case 277:
			s = "MQFB_STOPPED_BY_CHAD_EXIT"
		case 279:
			s = "MQFB_STOPPED_BY_PUBSUB_EXIT"
		case 280:
			s = "MQFB_NOT_A_REPOSITORY_MSG"
		case 281:
			s = "MQFB_BIND_OPEN_CLUSRCVR_DEL"
		case 282:
			s = "MQFB_MAX_ACTIVITIES"
		case 283:
			s = "MQFB_NOT_FORWARDED"
		case 284:
			s = "MQFB_NOT_DELIVERED"
		case 285:
			s = "MQFB_UNSUPPORTED_FORWARDING"
		case 286:
			s = "MQFB_UNSUPPORTED_DELIVERY"
		case 291:
			s = "MQFB_DATA_LENGTH_ZERO"
		case 292:
			s = "MQFB_DATA_LENGTH_NEGATIVE"
		case 293:
			s = "MQFB_DATA_LENGTH_TOO_BIG"
		case 294:
			s = "MQFB_BUFFER_OVERFLOW"
		case 295:
			s = "MQFB_LENGTH_OFF_BY_ONE"
		case 296:
			s = "MQFB_IIH_ERROR"
		case 298:
			s = "MQFB_NOT_AUTHORIZED_FOR_IMS"
		case 299:
			s = "MQFB_DATA_LENGTH_TOO_SHORT"
		case 300:
			s = "MQFB_IMS_ERROR"
		case 401:
			s = "MQFB_CICS_INTERNAL_ERROR"
		case 402:
			s = "MQFB_CICS_NOT_AUTHORIZED"
		case 403:
			s = "MQFB_CICS_BRIDGE_FAILURE"
		case 404:
			s = "MQFB_CICS_CORREL_ID_ERROR"
		case 405:
			s = "MQFB_CICS_CCSID_ERROR"
		case 406:
			s = "MQFB_CICS_ENCODING_ERROR"
		case 407:
			s = "MQFB_CICS_CIH_ERROR"
		case 408:
			s = "MQFB_CICS_UOW_ERROR"
		case 409:
			s = "MQFB_CICS_COMMAREA_ERROR"
		case 410:
			s = "MQFB_CICS_APPL_NOT_STARTED"
		case 411:
			s = "MQFB_CICS_APPL_ABENDED"
		case 412:
			s = "MQFB_CICS_DLQ_ERROR"
		case 413:
			s = "MQFB_CICS_UOW_BACKED_OUT"
		case 501:
			s = "MQFB_PUBLICATIONS_ON_REQUEST"
		case 502:
			s = "MQFB_SUBSCRIBER_IS_PUBLISHER"
		case 503:
			s = "MQFB_MSG_SCOPE_MISMATCH"
		case 504:
			s = "MQFB_SELECTOR_MISMATCH"
		case 505:
			s = "MQFB_NOT_A_GROUPUR_MSG"
		default:
			s = ""
		}

	case "FC":
		switch v {
		case 0:
			s = "MQFC_NO"
		case 1:
			s = "MQFC_YES"
		default:
			s = ""
		}

	case "FSENC":
		switch v {
		case 0:
			s = "MQFSENC_NO"
		case 1:
			s = "MQFSENC_YES"
		case 2:
			s = "MQFSENC_UNKNOWN"
		default:
			s = ""
		}

	case "FS":
		switch v {
		case -1:
			s = "MQFS_SHARED"
		default:
			s = ""
		}

	case "FUN":
		switch v {
		case 0:
			s = "MQFUN_TYPE_UNKNOWN"
		case 1:
			s = "MQFUN_TYPE_JVM"
		case 2:
			s = "MQFUN_TYPE_PROGRAM"
		case 3:
			s = "MQFUN_TYPE_PROCEDURE"
		case 4:
			s = "MQFUN_TYPE_USERDEF"
		case 5:
			s = "MQFUN_TYPE_COMMAND"
		default:
			s = ""
		}

	case "GACF":
		switch v {
		case 8001:
			s = "MQGACF_COMMAND_CONTEXT"
		case 8002:
			s = "MQGACF_COMMAND_DATA"
		case 8003:
			s = "MQGACF_TRACE_ROUTE"
		case 8004:
			s = "MQGACF_OPERATION"
		case 8005:
			s = "MQGACF_ACTIVITY"
		case 8006:
			s = "MQGACF_EMBEDDED_MQMD"
		case 8007:
			s = "MQGACF_MESSAGE"
		case 8008:
			s = "MQGACF_MQMD"
		case 8009:
			s = "MQGACF_VALUE_NAMING"
		case 8010:
			s = "MQGACF_Q_ACCOUNTING_DATA"
		case 8011:
			s = "MQGACF_Q_STATISTICS_DATA"
		case 8012:
			s = "MQGACF_CHL_STATISTICS_DATA"
		case 8013:
			s = "MQGACF_ACTIVITY_TRACE"
		case 8014:
			s = "MQGACF_APP_DIST_LIST"
		case 8015:
			s = "MQGACF_MONITOR_CLASS"
		case 8016:
			s = "MQGACF_MONITOR_TYPE"
		case 8017:
			s = "MQGACF_MONITOR_ELEMENT"
		case 8018:
			s = "MQGACF_APPL_STATUS"
		case 8019:
			s = "MQGACF_CHANGED_APPLS"
		case 8020:
			s = "MQGACF_ALL_APPLS"
		case 8021:
			s = "MQGACF_APPL_BALANCE"
		default:
			s = ""
		}

	case "GMO":
		switch v {
		case 0:
			s = "MQGMO_NONE"
		case 1:
			s = "MQGMO_WAIT"
		case 2:
			s = "MQGMO_SYNCPOINT"
		case 4:
			s = "MQGMO_NO_SYNCPOINT"
		case 8:
			s = "MQGMO_SET_SIGNAL"
		case 16:
			s = "MQGMO_BROWSE_FIRST"
		case 32:
			s = "MQGMO_BROWSE_NEXT"
		case 64:
			s = "MQGMO_ACCEPT_TRUNCATED_MSG"
		case 128:
			s = "MQGMO_MARK_SKIP_BACKOUT"
		case 256:
			s = "MQGMO_MSG_UNDER_CURSOR"
		case 512:
			s = "MQGMO_LOCK"
		case 1024:
			s = "MQGMO_UNLOCK"
		case 2048:
			s = "MQGMO_BROWSE_MSG_UNDER_CURSOR"
		case 4096:
			s = "MQGMO_SYNCPOINT_IF_PERSISTENT"
		case 8192:
			s = "MQGMO_FAIL_IF_QUIESCING"
		case 16384:
			s = "MQGMO_CONVERT"
		case 32768:
			s = "MQGMO_LOGICAL_ORDER"
		case 65536:
			s = "MQGMO_COMPLETE_MSG"
		case 131072:
			s = "MQGMO_ALL_MSGS_AVAILABLE"
		case 262144:
			s = "MQGMO_ALL_SEGMENTS_AVAILABLE"
		case 1048576:
			s = "MQGMO_MARK_BROWSE_HANDLE"
		case 2097152:
			s = "MQGMO_MARK_BROWSE_CO_OP"
		case 4194304:
			s = "MQGMO_UNMARK_BROWSE_CO_OP"
		case 8388608:
			s = "MQGMO_UNMARK_BROWSE_HANDLE"
		case 16777216:
			s = "MQGMO_UNMARKED_BROWSE_MSG"
		case 17825808:
			s = "MQGMO_BROWSE_HANDLE"
		case 18874384:
			s = "MQGMO_BROWSE_CO_OP"
		case 33554432:
			s = "MQGMO_PROPERTIES_FORCE_MQRFH2"
		case 67108864:
			s = "MQGMO_NO_PROPERTIES"
		case 134217728:
			s = "MQGMO_PROPERTIES_IN_HANDLE"
		case 268435456:
			s = "MQGMO_PROPERTIES_COMPATIBILITY"
		default:
			s = ""
		}

	case "GUR":
		switch v {
		case 0:
			s = "MQGUR_DISABLED"
		case 1:
			s = "MQGUR_ENABLED"
		default:
			s = ""
		}

	case "HA":
		switch v {
		case 4001:
			s = "MQHA_BAG_HANDLE"
		default:
			s = ""
		}

	case "HB":
		switch v {
		case -2:
			s = "MQHB_NONE"
		case -1:
			s = "MQHB_UNUSABLE_HBAG"
		default:
			s = ""
		}

	case "HC":
		switch v {
		case -3:
			s = "MQHC_UNASSOCIATED_HCONN"
		case -1:
			s = "MQHC_UNUSABLE_HCONN"
		case 0:
			s = "MQHC_DEF_HCONN"
		default:
			s = ""
		}

	case "HM":
		switch v {
		case -1:
			s = "MQHM_UNUSABLE_HMSG"
		case 0:
			s = "MQHM_NONE"
		default:
			s = ""
		}

	case "HO":
		switch v {
		case -1:
			s = "MQHO_UNUSABLE_HOBJ"
		case 0:
			s = "MQHO_NONE"
		default:
			s = ""
		}

	case "HSTATE":
		switch v {
		case 0:
			s = "MQHSTATE_INACTIVE"
		case 1:
			s = "MQHSTATE_ACTIVE"
		default:
			s = ""
		}

	case "IAMO_MONITOR_DATATYPE":
		switch v {
		case 1:
			s = "MQIAMO_MONITOR_UNIT"
		case 2:
			s = "MQIAMO_MONITOR_DELTA"
		case 3:
			s = "MQIAMO_MONITOR_LSN"
		case 100:
			s = "MQIAMO_MONITOR_HUNDREDTHS"
		case 1024:
			s = "MQIAMO_MONITOR_KB"
		case 10000:
			s = "MQIAMO_MONITOR_PERCENT"
		case 1000000:
			s = "MQIAMO_MONITOR_MICROSEC"
		case 1048576:
			s = "MQIAMO_MONITOR_MB"
		case 100000000:
			s = "MQIAMO_MONITOR_GB"
		default:
			s = ""
		}

	case "IAMO_MONITOR_FLAGS":
		switch v {
		case 0:
			s = "MQIAMO_MONITOR_FLAGS_NONE"
		case 1:
			s = "MQIAMO_MONITOR_FLAGS_OBJNAME"
		default:
			s = ""
		}

	case "IASY":
		switch v {
		case -9:
			s = "MQIASY_VERSION"
		case -8:
			s = "MQIASY_BAG_OPTIONS"
		case -7:
			s = "MQIASY_REASON"
		case -6:
			s = "MQIASY_COMP_CODE"
		case -5:
			s = "MQIASY_CONTROL"
		case -4:
			s = "MQIASY_MSG_SEQ_NUMBER"
		case -3:
			s = "MQIASY_COMMAND"
		case -2:
			s = "MQIASY_TYPE"
		case -1:
			s = "MQIASY_CODED_CHAR_SET_ID"
		default:
			s = ""
		}

	case "IAV":
		switch v {
		case -2:
			s = "MQIAV_UNDEFINED"
		case -1:
			s = "MQIAV_NOT_APPLICABLE"
		default:
			s = ""
		}

	case "IDO":
		switch v {
		case 1:
			s = "MQIDO_COMMIT"
		case 2:
			s = "MQIDO_BACKOUT"
		default:
			s = ""
		}

	case "IEPF":
		switch v {
		case 0:
			s = "MQIEPF_NONE"
		case 1:
			s = "MQIEPF_THREADED_LIBRARY"
		case 2:
			s = "MQIEPF_LOCAL_LIBRARY"
		default:
			s = ""
		}

	case "IGQPA":
		switch v {
		case 1:
			s = "MQIGQPA_DEFAULT"
		case 2:
			s = "MQIGQPA_CONTEXT"
		case 3:
			s = "MQIGQPA_ONLY_IGQ"
		case 4:
			s = "MQIGQPA_ALTERNATE_OR_IGQ"
		default:
			s = ""
		}

	case "IGQ":
		switch v {
		case 0:
			s = "MQIGQ_DISABLED"
		case 1:
			s = "MQIGQ_ENABLED"
		default:
			s = ""
		}

	case "IIH":
		switch v {
		case 0:
			s = "MQIIH_NONE"
		case 1:
			s = "MQIIH_PASS_EXPIRATION"
		case 8:
			s = "MQIIH_REPLY_FORMAT_NONE"
		case 16:
			s = "MQIIH_IGNORE_PURG"
		case 32:
			s = "MQIIH_CM0_REQUEST_RESPONSE"
		default:
			s = ""
		}

	case "IMGRCOV":
		switch v {
		case 0:
			s = "MQIMGRCOV_NO"
		case 1:
			s = "MQIMGRCOV_YES"
		case 2:
			s = "MQIMGRCOV_AS_Q_MGR"
		default:
			s = ""
		}

	case "IMMREASON":
		switch v {
		case 0:
			s = "MQIMMREASON_NONE"
		case 1:
			s = "MQIMMREASON_NOT_CLIENT"
		case 2:
			s = "MQIMMREASON_NOT_RECONNECTABLE"
		case 3:
			s = "MQIMMREASON_MOVING"
		case 4:
			s = "MQIMMREASON_APPLNAME_CHANGED"
		case 5:
			s = "MQIMMREASON_IN_TRANSACTION"
		case 6:
			s = "MQIMMREASON_AWAITS_REPLY"
		case 7:
			s = "MQIMMREASON_NO_REDIRECT"
		default:
			s = ""
		}

	case "IMPO":
		switch v {
		case 0:
			s = "MQIMPO_NONE"
		case 2:
			s = "MQIMPO_CONVERT_TYPE"
		case 4:
			s = "MQIMPO_QUERY_LENGTH"
		case 8:
			s = "MQIMPO_INQ_NEXT"
		case 16:
			s = "MQIMPO_INQ_PROP_UNDER_CURSOR"
		case 32:
			s = "MQIMPO_CONVERT_VALUE"
		default:
			s = ""
		}

	case "INBD":
		switch v {
		case 0:
			s = "MQINBD_Q_MGR"
		case 3:
			s = "MQINBD_GROUP"
		default:
			s = ""
		}

	case "IND":
		switch v {
		case -2:
			s = "MQIND_ALL"
		case -1:
			s = "MQIND_NONE"
		default:
			s = ""
		}

	case "IPADDR":
		switch v {
		case 0:
			s = "MQIPADDR_IPV4"
		case 1:
			s = "MQIPADDR_IPV6"
		default:
			s = ""
		}

	case "IS":
		switch v {
		case 0:
			s = "MQIS_NO"
		case 1:
			s = "MQIS_YES"
		default:
			s = ""
		}

	case "IT":
		switch v {
		case 0:
			s = "MQIT_NONE"
		case 1:
			s = "MQIT_MSG_ID"
		case 2:
			s = "MQIT_CORREL_ID"
		case 4:
			s = "MQIT_MSG_TOKEN"
		case 5:
			s = "MQIT_GROUP_ID"
		default:
			s = ""
		}

	case "KAI":
		switch v {
		case -1:
			s = "MQKAI_AUTO"
		default:
			s = ""
		}

	case "KEY":
		switch v {
		case -1:
			s = "MQKEY_REUSE_UNLIMITED"
		case 0:
			s = "MQKEY_REUSE_DISABLED"
		default:
			s = ""
		}

	case "LDAPC":
		switch v {
		case 0:
			s = "MQLDAPC_INACTIVE"
		case 1:
			s = "MQLDAPC_CONNECTED"
		case 2:
			s = "MQLDAPC_ERROR"
		default:
			s = ""
		}

	case "LDAP_AUTHORMD":
		switch v {
		case 0:
			s = "MQLDAP_AUTHORMD_OS"
		case 1:
			s = "MQLDAP_AUTHORMD_SEARCHGRP"
		case 2:
			s = "MQLDAP_AUTHORMD_SEARCHUSR"
		case 3:
			s = "MQLDAP_AUTHORMD_SRCHGRPSN"
		default:
			s = ""
		}

	case "LDAP_NESTGRP":
		switch v {
		case 0:
			s = "MQLDAP_NESTGRP_NO"
		case 1:
			s = "MQLDAP_NESTGRP_YES"
		default:
			s = ""
		}

	case "LOGTYPE":
		switch v {
		case 0:
			s = "MQLOGTYPE_CIRCULAR"
		case 1:
			s = "MQLOGTYPE_LINEAR"
		case 2:
			s = "MQLOGTYPE_REPLICATED"
		default:
			s = ""
		}

	case "LR":
		switch v {
		case -2:
			s = "MQLR_MAX"
		case -1:
			s = "MQLR_AUTO"
		case 1:
			s = "MQLR_ONE"
		default:
			s = ""
		}

	case "MASTER":
		switch v {
		case 0:
			s = "MQMASTER_NO"
		case 1:
			s = "MQMASTER_YES"
		default:
			s = ""
		}

	case "MATCH":
		switch v {
		case 0:
			s = "MQMATCH_GENERIC"
		case 1:
			s = "MQMATCH_RUNCHECK"
		case 2:
			s = "MQMATCH_EXACT"
		case 3:
			s = "MQMATCH_ALL"
		default:
			s = ""
		}

	case "MCAS":
		switch v {
		case 0:
			s = "MQMCAS_STOPPED"
		case 3:
			s = "MQMCAS_RUNNING"
		default:
			s = ""
		}

	case "MCAT":
		switch v {
		case 1:
			s = "MQMCAT_PROCESS"
		case 2:
			s = "MQMCAT_THREAD"
		default:
			s = ""
		}

	case "MCB":
		switch v {
		case 0:
			s = "MQMCB_DISABLED"
		case 1:
			s = "MQMCB_ENABLED"
		default:
			s = ""
		}

	case "MCEV":
		switch v {
		case 1:
			s = "MQMCEV_PACKET_LOSS"
		case 2:
			s = "MQMCEV_HEARTBEAT_TIMEOUT"
		case 3:
			s = "MQMCEV_VERSION_CONFLICT"
		case 4:
			s = "MQMCEV_RELIABILITY"
		case 5:
			s = "MQMCEV_CLOSED_TRANS"
		case 6:
			s = "MQMCEV_STREAM_ERROR"
		case 10:
			s = "MQMCEV_NEW_SOURCE"
		case 11:
			s = "MQMCEV_RECEIVE_QUEUE_TRIMMED"
		case 12:
			s = "MQMCEV_PACKET_LOSS_NACK_EXPIRE"
		case 13:
			s = "MQMCEV_ACK_RETRIES_EXCEEDED"
		case 14:
			s = "MQMCEV_STREAM_SUSPEND_NACK"
		case 15:
			s = "MQMCEV_STREAM_RESUME_NACK"
		case 16:
			s = "MQMCEV_STREAM_EXPELLED"
		case 20:
			s = "MQMCEV_FIRST_MESSAGE"
		case 21:
			s = "MQMCEV_LATE_JOIN_FAILURE"
		case 22:
			s = "MQMCEV_MESSAGE_LOSS"
		case 23:
			s = "MQMCEV_SEND_PACKET_FAILURE"
		case 24:
			s = "MQMCEV_REPAIR_DELAY"
		case 25:
			s = "MQMCEV_MEMORY_ALERT_ON"
		case 26:
			s = "MQMCEV_MEMORY_ALERT_OFF"
		case 27:
			s = "MQMCEV_NACK_ALERT_ON"
		case 28:
			s = "MQMCEV_NACK_ALERT_OFF"
		case 29:
			s = "MQMCEV_REPAIR_ALERT_ON"
		case 30:
			s = "MQMCEV_REPAIR_ALERT_OFF"
		case 31:
			s = "MQMCEV_RELIABILITY_CHANGED"
		case 80:
			s = "MQMCEV_SHM_DEST_UNUSABLE"
		case 81:
			s = "MQMCEV_SHM_PORT_UNUSABLE"
		case 110:
			s = "MQMCEV_CCT_GETTIME_FAILED"
		case 120:
			s = "MQMCEV_DEST_INTERFACE_FAILURE"
		case 121:
			s = "MQMCEV_DEST_INTERFACE_FAILOVER"
		case 122:
			s = "MQMCEV_PORT_INTERFACE_FAILURE"
		case 123:
			s = "MQMCEV_PORT_INTERFACE_FAILOVER"
		default:
			s = ""
		}

	case "MCP":
		switch v {
		case -2:
			s = "MQMCP_COMPAT"
		case -1:
			s = "MQMCP_ALL"
		case 0:
			s = "MQMCP_NONE"
		case 1:
			s = "MQMCP_USER"
		case 2:
			s = "MQMCP_REPLY"
		default:
			s = ""
		}

	case "MC":
		switch v {
		case 0:
			s = "MQMC_AS_PARENT"
		case 1:
			s = "MQMC_ENABLED"
		case 2:
			s = "MQMC_DISABLED"
		case 3:
			s = "MQMC_ONLY"
		default:
			s = ""
		}

	case "MDEF":
		switch v {
		case 0:
			s = "MQMDEF_NONE"
		default:
			s = ""
		}

	case "MDS":
		switch v {
		case 0:
			s = "MQMDS_PRIORITY"
		case 1:
			s = "MQMDS_FIFO"
		default:
			s = ""
		}

	case "MEDIMGINTVL":
		switch v {
		case 0:
			s = "MQMEDIMGINTVL_OFF"
		default:
			s = ""
		}

	case "MEDIMGLOGLN":
		switch v {
		case 0:
			s = "MQMEDIMGLOGLN_OFF"
		default:
			s = ""
		}

	case "MEDIMGSCHED":
		switch v {
		case 0:
			s = "MQMEDIMGSCHED_MANUAL"
		case 1:
			s = "MQMEDIMGSCHED_AUTO"
		default:
			s = ""
		}

	case "MF":
		switch v {
		case -1048576:
			s = "MQMF_ACCEPT_UNSUP_MASK"
		case 0:
			s = "MQMF_NONE"
		case 1:
			s = "MQMF_SEGMENTATION_ALLOWED"
		case 2:
			s = "MQMF_SEGMENT"
		case 4:
			s = "MQMF_LAST_SEGMENT"
		case 8:
			s = "MQMF_MSG_IN_GROUP"
		case 16:
			s = "MQMF_LAST_MSG_IN_GROUP"
		case 4095:
			s = "MQMF_REJECT_UNSUP_MASK"
		case 1044480:
			s = "MQMF_ACCEPT_UNSUP_IF_XMIT_MASK"
		default:
			s = ""
		}

	case "MHBO":
		switch v {
		case 0:
			s = "MQMHBO_NONE"
		case 1:
			s = "MQMHBO_PROPERTIES_IN_MQRFH2"
		case 2:
			s = "MQMHBO_DELETE_PROPERTIES"
		default:
			s = ""
		}

	case "MLP_ENCRYPTION":
		switch v {
		case 0:
			s = "MQMLP_ENCRYPTION_ALG_NONE"
		case 1:
			s = "MQMLP_ENCRYPTION_ALG_RC2"
		case 2:
			s = "MQMLP_ENCRYPTION_ALG_DES"
		case 3:
			s = "MQMLP_ENCRYPTION_ALG_3DES"
		case 4:
			s = "MQMLP_ENCRYPTION_ALG_AES128"
		case 5:
			s = "MQMLP_ENCRYPTION_ALG_AES256"
		default:
			s = ""
		}

	case "MLP_SIGN":
		switch v {
		case 0:
			s = "MQMLP_SIGN_ALG_NONE"
		case 1:
			s = "MQMLP_SIGN_ALG_MD5"
		case 2:
			s = "MQMLP_SIGN_ALG_SHA1"
		case 3:
			s = "MQMLP_SIGN_ALG_SHA224"
		case 4:
			s = "MQMLP_SIGN_ALG_SHA256"
		case 5:
			s = "MQMLP_SIGN_ALG_SHA384"
		case 6:
			s = "MQMLP_SIGN_ALG_SHA512"
		default:
			s = ""
		}

	case "MLP_TOLERATE":
		switch v {
		case 0:
			s = "MQMLP_TOLERATE_UNPROTECTED_NO"
		case 1:
			s = "MQMLP_TOLERATE_UNPROTECTED_YES"
		default:
			s = ""
		}

	case "MMBI":
		switch v {
		case -1:
			s = "MQMMBI_UNLIMITED"
		default:
			s = ""
		}

	case "MODE":
		switch v {
		case 0:
			s = "MQMODE_FORCE"
		case 1:
			s = "MQMODE_QUIESCE"
		case 2:
			s = "MQMODE_TERMINATE"
		default:
			s = ""
		}

	case "MON_OVERRIDE":
		switch v {
		case 0:
			s = "MQMON_DISABLED"
		case 1:
			s = "MQMON_ENABLED"
		default:
			s = ""
		}

	case "MON":
		switch v {
		case -3:
			s = "MQMON_Q_MGR"
		case -1:
			s = "MQMON_NONE"
		case 0:
			s = "MQMON_OFF"
		case 1:
			s = "MQMON_ON"
		case 17:
			s = "MQMON_LOW"
		case 33:
			s = "MQMON_MEDIUM"
		case 65:
			s = "MQMON_HIGH"
		default:
			s = ""
		}

	case "MON_AVAILABILITY":
		switch v {
		case -1:
			s = "MQMON_NOT_AVAILABLE"
		default:
			s = ""
		}

	case "MO":
		switch v {
		case 0:
			s = "MQMO_NONE"
		case 1:
			s = "MQMO_MATCH_MSG_ID"
		case 2:
			s = "MQMO_MATCH_CORREL_ID"
		case 4:
			s = "MQMO_MATCH_GROUP_ID"
		case 8:
			s = "MQMO_MATCH_MSG_SEQ_NUMBER"
		case 16:
			s = "MQMO_MATCH_OFFSET"
		case 32:
			s = "MQMO_MATCH_MSG_TOKEN"
		default:
			s = ""
		}

	case "MT":
		switch v {
		case 1:
			s = "MQMT_REQUEST"
		case 2:
			s = "MQMT_REPLY"
		case 4:
			s = "MQMT_REPORT"
		case 8:
			s = "MQMT_DATAGRAM"
		case 112:
			s = "MQMT_MQE_FIELDS_FROM_MQE"
		case 113:
			s = "MQMT_MQE_FIELDS"
		default:
			s = ""
		}

	case "MULC":
		switch v {
		case 0:
			s = "MQMULC_STANDARD"
		case 1:
			s = "MQMULC_REFINED"
		default:
			s = ""
		}

	case "NC":
		switch v {
		case 256:
			s = "MQNC_MAX_NAMELIST_NAME_COUNT"
		default:
			s = ""
		}

	case "NHABACKLOG":
		switch v {
		case -1:
			s = "MQNHABACKLOG_UNKNOWN"
		default:
			s = ""
		}

	case "NHACONNACTV":
		switch v {
		case 0:
			s = "MQNHACONNACTV_NO"
		case 1:
			s = "MQNHACONNACTV_YES"
		default:
			s = ""
		}

	case "NHACONNGRP":
		switch v {
		case 0:
			s = "MQNHACONNGRP_NO"
		case 1:
			s = "MQNHACONNGRP_YES"
		case 2:
			s = "MQNHACONNGRP_SUSPENDED"
		default:
			s = ""
		}

	case "NHAGRPROLE":
		switch v {
		case 0:
			s = "MQNHAGRPROLE_UNKNOWN"
		case 1:
			s = "MQNHAGRPROLE_NOT_CONFIGURED"
		case 2:
			s = "MQNHAGRPROLE_LIVE"
		case 3:
			s = "MQNHAGRPROLE_RECOVERY"
		case 4:
			s = "MQNHAGRPROLE_PENDING_LIVE"
		case 5:
			s = "MQNHAGRPROLE_PENDING_RECOVERY"
		default:
			s = ""
		}

	case "NHAINSYNC":
		switch v {
		case 0:
			s = "MQNHAINSYNC_NO"
		case 1:
			s = "MQNHAINSYNC_YES"
		default:
			s = ""
		}

	case "NHAROLE":
		switch v {
		case 0:
			s = "MQNHAROLE_UNKNOWN"
		case 1:
			s = "MQNHAROLE_ACTIVE"
		case 2:
			s = "MQNHAROLE_REPLICA"
		case 3:
			s = "MQNHAROLE_LEADER"
		default:
			s = ""
		}

	case "NHASTATUS":
		switch v {
		case 0:
			s = "MQNHASTATUS_UNKNOWN"
		case 1:
			s = "MQNHASTATUS_NORMAL"
		case 2:
			s = "MQNHASTATUS_CHECKING"
		case 3:
			s = "MQNHASTATUS_SYNCHRONIZING"
		case 4:
			s = "MQNHASTATUS_REBASING"
		case 5:
			s = "MQNHASTATUS_DISK_FULL"
		case 6:
			s = "MQNHASTATUS_DISCONNECTED"
		case 7:
			s = "MQNHASTATUS_PARTITIONED"
		default:
			s = ""
		}

	case "NHATYPE":
		switch v {
		case -1:
			s = "MQNHATYPE_ALL"
		case 0:
			s = "MQNHATYPE_INSTANCE"
		case 1:
			s = "MQNHATYPE_GROUP"
		default:
			s = ""
		}

	case "NPMS":
		switch v {
		case 1:
			s = "MQNPMS_NORMAL"
		case 2:
			s = "MQNPMS_FAST"
		default:
			s = ""
		}

	case "NPM":
		switch v {
		case 0:
			s = "MQNPM_CLASS_NORMAL"
		case 10:
			s = "MQNPM_CLASS_HIGH"
		default:
			s = ""
		}

	case "NSH":
		switch v {
		case -1:
			s = "MQNSH_ALL"
		case 0:
			s = "MQNSH_NONE"
		default:
			s = ""
		}

	case "NT":
		switch v {
		case 0:
			s = "MQNT_NONE"
		case 1:
			s = "MQNT_Q"
		case 2:
			s = "MQNT_CLUSTER"
		case 4:
			s = "MQNT_AUTH_INFO"
		case 1001:
			s = "MQNT_ALL"
		default:
			s = ""
		}

	case "OL":
		switch v {
		case -1:
			s = "MQOL_UNDEFINED"
		default:
			s = ""
		}

	case "OM":
		switch v {
		case 0:
			s = "MQOM_NO"
		case 1:
			s = "MQOM_YES"
		default:
			s = ""
		}

	case "OO":
		switch v {
		case 0:
			s = "MQOO_READ_AHEAD_AS_Q_DEF"
		case 1:
			s = "MQOO_INPUT_AS_Q_DEF"
		case 2:
			s = "MQOO_INPUT_SHARED"
		case 4:
			s = "MQOO_INPUT_EXCLUSIVE"
		case 8:
			s = "MQOO_BROWSE"
		case 16:
			s = "MQOO_OUTPUT"
		case 32:
			s = "MQOO_INQUIRE"
		case 64:
			s = "MQOO_SET"
		case 128:
			s = "MQOO_SAVE_ALL_CONTEXT"
		case 256:
			s = "MQOO_PASS_IDENTITY_CONTEXT"
		case 512:
			s = "MQOO_PASS_ALL_CONTEXT"
		case 1024:
			s = "MQOO_SET_IDENTITY_CONTEXT"
		case 2048:
			s = "MQOO_SET_ALL_CONTEXT"
		case 4096:
			s = "MQOO_ALTERNATE_USER_AUTHORITY"
		case 8192:
			s = "MQOO_FAIL_IF_QUIESCING"
		case 16384:
			s = "MQOO_BIND_ON_OPEN"
		case 32768:
			s = "MQOO_BIND_NOT_FIXED"
		case 65536:
			s = "MQOO_RESOLVE_NAMES"
		case 131072:
			s = "MQOO_CO_OP"
		case 262144:
			s = "MQOO_RESOLVE_LOCAL_Q"
		case 524288:
			s = "MQOO_NO_READ_AHEAD"
		case 1048576:
			s = "MQOO_READ_AHEAD"
		case 2097152:
			s = "MQOO_NO_MULTICAST"
		case 4194304:
			s = "MQOO_BIND_ON_GROUP"
		default:
			s = ""
		}

	case "OPER":
		switch v {
		case 0:
			s = "MQOPER_UNKNOWN"
		case 1:
			s = "MQOPER_BROWSE"
		case 2:
			s = "MQOPER_DISCARD"
		case 3:
			s = "MQOPER_GET"
		case 4:
			s = "MQOPER_PUT"
		case 5:
			s = "MQOPER_PUT_REPLY"
		case 6:
			s = "MQOPER_PUT_REPORT"
		case 7:
			s = "MQOPER_RECEIVE"
		case 8:
			s = "MQOPER_SEND"
		case 9:
			s = "MQOPER_TRANSFORM"
		case 10:
			s = "MQOPER_PUBLISH"
		case 11:
			s = "MQOPER_EXCLUDED_PUBLISH"
		case 12:
			s = "MQOPER_DISCARDED_PUBLISH"
		default:
			s = ""
		}

	case "OPMODE":
		switch v {
		case 0:
			s = "MQOPMODE_COMPAT"
		case 1:
			s = "MQOPMODE_NEW_FUNCTION"
		default:
			s = ""
		}

	case "OP":
		switch v {
		case 1:
			s = "MQOP_START"
		case 2:
			s = "MQOP_START_WAIT"
		case 4:
			s = "MQOP_STOP"
		case 256:
			s = "MQOP_REGISTER"
		case 512:
			s = "MQOP_DEREGISTER"
		case 65536:
			s = "MQOP_SUSPEND"
		case 131072:
			s = "MQOP_RESUME"
		default:
			s = ""
		}

	case "PAGECLAS":
		switch v {
		case 0:
			s = "MQPAGECLAS_4KB"
		case 1:
			s = "MQPAGECLAS_FIXED4KB"
		default:
			s = ""
		}

	case "PA":
		switch v {
		case 1:
			s = "MQPA_DEFAULT"
		case 2:
			s = "MQPA_CONTEXT"
		case 3:
			s = "MQPA_ONLY_MCA"
		case 4:
			s = "MQPA_ALTERNATE_OR_MCA"
		default:
			s = ""
		}

	case "PD":
		switch v {
		case -1048576:
			s = "MQPD_REJECT_UNSUP_MASK"
		case 0:
			s = "MQPD_NONE"
		case 1:
			s = "MQPD_SUPPORT_OPTIONAL"
		case 1023:
			s = "MQPD_ACCEPT_UNSUP_MASK"
		case 1024:
			s = "MQPD_SUPPORT_REQUIRED_IF_LOCAL"
		case 1047552:
			s = "MQPD_ACCEPT_UNSUP_IF_XMIT_MASK"
		case 1048576:
			s = "MQPD_SUPPORT_REQUIRED"
		default:
			s = ""
		}

	case "PER":
		switch v {
		case -1:
			s = "MQPER_PERSISTENCE_AS_PARENT"
		case 0:
			s = "MQPER_NOT_PERSISTENT"
		case 1:
			s = "MQPER_PERSISTENT"
		case 2:
			s = "MQPER_PERSISTENCE_AS_Q_DEF"
		default:
			s = ""
		}

	case "PMO":
		switch v {
		case 0:
			s = "MQPMO_NONE"
		case 2:
			s = "MQPMO_SYNCPOINT"
		case 4:
			s = "MQPMO_NO_SYNCPOINT"
		case 32:
			s = "MQPMO_DEFAULT_CONTEXT"
		case 64:
			s = "MQPMO_NEW_MSG_ID"
		case 128:
			s = "MQPMO_NEW_CORREL_ID"
		case 256:
			s = "MQPMO_PASS_IDENTITY_CONTEXT"
		case 512:
			s = "MQPMO_PASS_ALL_CONTEXT"
		case 1024:
			s = "MQPMO_SET_IDENTITY_CONTEXT"
		case 2048:
			s = "MQPMO_SET_ALL_CONTEXT"
		case 4096:
			s = "MQPMO_ALTERNATE_USER_AUTHORITY"
		case 8192:
			s = "MQPMO_FAIL_IF_QUIESCING"
		case 16384:
			s = "MQPMO_NO_CONTEXT"
		case 32768:
			s = "MQPMO_LOGICAL_ORDER"
		case 65536:
			s = "MQPMO_ASYNC_RESPONSE"
		case 131072:
			s = "MQPMO_SYNC_RESPONSE"
		case 262144:
			s = "MQPMO_RESOLVE_LOCAL_Q"
		case 524288:
			s = "MQPMO_WARN_IF_NO_SUBS_MATCHED"
		case 2097152:
			s = "MQPMO_RETAIN"
		case 8388608:
			s = "MQPMO_MD_FOR_OUTPUT_ONLY"
		case 67108864:
			s = "MQPMO_SCOPE_QMGR"
		case 134217728:
			s = "MQPMO_SUPPRESS_REPLYTO"
		case 268435456:
			s = "MQPMO_NOT_OWN_SUBS"
		default:
			s = ""
		}

	case "PMRF":
		switch v {
		case 0:
			s = "MQPMRF_NONE"
		case 1:
			s = "MQPMRF_MSG_ID"
		case 2:
			s = "MQPMRF_CORREL_ID"
		case 4:
			s = "MQPMRF_GROUP_ID"
		case 8:
			s = "MQPMRF_FEEDBACK"
		case 16:
			s = "MQPMRF_ACCOUNTING_TOKEN"
		default:
			s = ""
		}

	case "PO":
		switch v {
		case 0:
			s = "MQPO_NO"
		case 1:
			s = "MQPO_YES"
		default:
			s = ""
		}

	case "PRI":
		switch v {
		case -3:
			s = "MQPRI_PRIORITY_AS_PUBLISHED"
		case -2:
			s = "MQPRI_PRIORITY_AS_PARENT"
		case -1:
			s = "MQPRI_PRIORITY_AS_Q_DEF"
		default:
			s = ""
		}

	case "PROP":
		switch v {
		case -1:
			s = "MQPROP_UNRESTRICTED_LENGTH"
		case 0:
			s = "MQPROP_COMPATIBILITY"
		case 1:
			s = "MQPROP_NONE"
		case 2:
			s = "MQPROP_ALL"
		case 3:
			s = "MQPROP_FORCE_MQRFH2"
		case 4:
			s = "MQPROP_V6COMPAT"
		default:
			s = ""
		}

	case "PROTO":
		switch v {
		case 1:
			s = "MQPROTO_MQTTV3"
		case 2:
			s = "MQPROTO_HTTP"
		case 3:
			s = "MQPROTO_AMQP"
		case 4:
			s = "MQPROTO_MQTTV311"
		default:
			s = ""
		}

	case "PRT":
		switch v {
		case 0:
			s = "MQPRT_RESPONSE_AS_PARENT"
		case 1:
			s = "MQPRT_SYNC_RESPONSE"
		case 2:
			s = "MQPRT_ASYNC_RESPONSE"
		default:
			s = ""
		}

	case "PSCLUS":
		switch v {
		case 0:
			s = "MQPSCLUS_DISABLED"
		case 1:
			s = "MQPSCLUS_ENABLED"
		default:
			s = ""
		}

	case "PSCT":
		switch v {
		case -1:
			s = "MQPSCT_NONE"
		default:
			s = ""
		}

	case "PSM":
		switch v {
		case 0:
			s = "MQPSM_DISABLED"
		case 1:
			s = "MQPSM_COMPAT"
		case 2:
			s = "MQPSM_ENABLED"
		default:
			s = ""
		}

	case "PSPROP":
		switch v {
		case 0:
			s = "MQPSPROP_NONE"
		case 1:
			s = "MQPSPROP_COMPAT"
		case 2:
			s = "MQPSPROP_RFH2"
		case 3:
			s = "MQPSPROP_MSGPROP"
		default:
			s = ""
		}

	case "PSST":
		switch v {
		case 0:
			s = "MQPSST_ALL"
		case 1:
			s = "MQPSST_LOCAL"
		case 2:
			s = "MQPSST_PARENT"
		case 3:
			s = "MQPSST_CHILD"
		default:
			s = ""
		}

	case "PS":
		switch v {
		case 0:
			s = "MQPS_STATUS_INACTIVE"
		case 1:
			s = "MQPS_STATUS_STARTING"
		case 2:
			s = "MQPS_STATUS_STOPPING"
		case 3:
			s = "MQPS_STATUS_ACTIVE"
		case 4:
			s = "MQPS_STATUS_COMPAT"
		case 5:
			s = "MQPS_STATUS_ERROR"
		case 6:
			s = "MQPS_STATUS_REFUSED"
		default:
			s = ""
		}

	case "PUBO":
		switch v {
		case 0:
			s = "MQPUBO_NONE"
		case 1:
			s = "MQPUBO_CORREL_ID_AS_IDENTITY"
		case 2:
			s = "MQPUBO_RETAIN_PUBLICATION"
		case 4:
			s = "MQPUBO_OTHER_SUBSCRIBERS_ONLY"
		case 8:
			s = "MQPUBO_NO_REGISTRATION"
		case 16:
			s = "MQPUBO_IS_RETAINED_PUBLICATION"
		default:
			s = ""
		}

	case "QA_BACKOUT":
		switch v {
		case 0:
			s = "MQQA_BACKOUT_NOT_HARDENED"
		case 1:
			s = "MQQA_BACKOUT_HARDENED"
		default:
			s = ""
		}

	case "QA_GET":
		switch v {
		case 0:
			s = "MQQA_GET_ALLOWED"
		case 1:
			s = "MQQA_GET_INHIBITED"
		default:
			s = ""
		}

	case "QA_PUT":
		switch v {
		case 0:
			s = "MQQA_PUT_ALLOWED"
		case 1:
			s = "MQQA_PUT_INHIBITED"
		default:
			s = ""
		}

	case "QA_SHAREABLE":
		switch v {
		case 0:
			s = "MQQA_NOT_SHAREABLE"
		case 1:
			s = "MQQA_SHAREABLE"
		default:
			s = ""
		}

	case "QDT":
		switch v {
		case 1:
			s = "MQQDT_PREDEFINED"
		case 2:
			s = "MQQDT_PERMANENT_DYNAMIC"
		case 3:
			s = "MQQDT_TEMPORARY_DYNAMIC"
		case 4:
			s = "MQQDT_SHARED_DYNAMIC"
		default:
			s = ""
		}

	case "QFS":
		switch v {
		case -1:
			s = "MQQFS_DEFAULT"
		default:
			s = ""
		}

	case "QF":
		switch v {
		case 1:
			s = "MQQF_LOCAL_Q"
		case 64:
			s = "MQQF_CLWL_USEQ_ANY"
		case 128:
			s = "MQQF_CLWL_USEQ_LOCAL"
		default:
			s = ""
		}

	case "QMDT":
		switch v {
		case 1:
			s = "MQQMDT_EXPLICIT_CLUSTER_SENDER"
		case 2:
			s = "MQQMDT_AUTO_CLUSTER_SENDER"
		case 3:
			s = "MQQMDT_CLUSTER_RECEIVER"
		case 4:
			s = "MQQMDT_AUTO_EXP_CLUSTER_SENDER"
		default:
			s = ""
		}

	case "QMFAC":
		switch v {
		case 1:
			s = "MQQMFAC_IMS_BRIDGE"
		case 2:
			s = "MQQMFAC_DB2"
		default:
			s = ""
		}

	case "QMF":
		switch v {
		case 2:
			s = "MQQMF_REPOSITORY_Q_MGR"
		case 8:
			s = "MQQMF_CLUSSDR_USER_DEFINED"
		case 16:
			s = "MQQMF_CLUSSDR_AUTO_DEFINED"
		case 32:
			s = "MQQMF_AVAILABLE"
		default:
			s = ""
		}

	case "QMOPT":
		switch v {
		case 0:
			s = "MQQMOPT_DISABLED"
		case 1:
			s = "MQQMOPT_ENABLED"
		case 2:
			s = "MQQMOPT_REPLY"
		default:
			s = ""
		}

	case "QMSTA":
		switch v {
		case 1:
			s = "MQQMSTA_STARTING"
		case 2:
			s = "MQQMSTA_RUNNING"
		case 3:
			s = "MQQMSTA_QUIESCING"
		case 4:
			s = "MQQMSTA_STANDBY"
		default:
			s = ""
		}

	case "QMT":
		switch v {
		case 0:
			s = "MQQMT_NORMAL"
		case 1:
			s = "MQQMT_REPOSITORY"
		default:
			s = ""
		}

	case "QO":
		switch v {
		case 0:
			s = "MQQO_NO"
		case 1:
			s = "MQQO_YES"
		default:
			s = ""
		}

	case "QSGD":
		switch v {
		case -1:
			s = "MQQSGD_ALL"
		case 0:
			s = "MQQSGD_Q_MGR"
		case 1:
			s = "MQQSGD_COPY"
		case 2:
			s = "MQQSGD_SHARED"
		case 3:
			s = "MQQSGD_GROUP"
		case 4:
			s = "MQQSGD_PRIVATE"
		case 6:
			s = "MQQSGD_LIVE"
		default:
			s = ""
		}

	case "QSGS":
		switch v {
		case 0:
			s = "MQQSGS_UNKNOWN"
		case 1:
			s = "MQQSGS_CREATED"
		case 2:
			s = "MQQSGS_ACTIVE"
		case 3:
			s = "MQQSGS_INACTIVE"
		case 4:
			s = "MQQSGS_FAILED"
		case 5:
			s = "MQQSGS_PENDING"
		default:
			s = ""
		}

	case "QSIE":
		switch v {
		case 0:
			s = "MQQSIE_NONE"
		case 1:
			s = "MQQSIE_HIGH"
		case 2:
			s = "MQQSIE_OK"
		default:
			s = ""
		}

	case "QSOT":
		switch v {
		case 1:
			s = "MQQSOT_ALL"
		case 2:
			s = "MQQSOT_INPUT"
		case 3:
			s = "MQQSOT_OUTPUT"
		default:
			s = ""
		}

	case "QSO":
		switch v {
		case 0:
			s = "MQQSO_NO"
		case 1:
			s = "MQQSO_YES"
		case 2:
			s = "MQQSO_EXCLUSIVE"
		default:
			s = ""
		}

	case "QSUM":
		switch v {
		case 0:
			s = "MQQSUM_NO"
		case 1:
			s = "MQQSUM_YES"
		default:
			s = ""
		}

	case "QT":
		switch v {
		case 1:
			s = "MQQT_LOCAL"
		case 2:
			s = "MQQT_MODEL"
		case 3:
			s = "MQQT_ALIAS"
		case 6:
			s = "MQQT_REMOTE"
		case 7:
			s = "MQQT_CLUSTER"
		case 1001:
			s = "MQQT_ALL"
		default:
			s = ""
		}

	case "RAR":
		switch v {
		case 0:
			s = "MQRAR_NO"
		case 1:
			s = "MQRAR_YES"
		default:
			s = ""
		}

	case "RCN":
		switch v {
		case 0:
			s = "MQRCN_NO"
		case 1:
			s = "MQRCN_YES"
		case 2:
			s = "MQRCN_Q_MGR"
		case 3:
			s = "MQRCN_DISABLED"
		default:
			s = ""
		}

	case "RCVTIME":
		switch v {
		case 0:
			s = "MQRCVTIME_MULTIPLY"
		case 1:
			s = "MQRCVTIME_ADD"
		case 2:
			s = "MQRCVTIME_EQUAL"
		default:
			s = ""
		}

	case "RDNS":
		switch v {
		case 0:
			s = "MQRDNS_ENABLED"
		case 1:
			s = "MQRDNS_DISABLED"
		default:
			s = ""
		}

	case "RD":
		switch v {
		case -1:
			s = "MQRD_NO_RECONNECT"
		case 0:
			s = "MQRD_NO_DELAY"
		default:
			s = ""
		}

	case "READA":
		switch v {
		case 0:
			s = "MQREADA_NO"
		case 1:
			s = "MQREADA_YES"
		case 2:
			s = "MQREADA_DISABLED"
		case 3:
			s = "MQREADA_INHIBITED"
		case 4:
			s = "MQREADA_BACKLOG"
		default:
			s = ""
		}

	case "RECAUTO":
		switch v {
		case 0:
			s = "MQRECAUTO_NO"
		case 1:
			s = "MQRECAUTO_YES"
		default:
			s = ""
		}

	case "RECORDING":
		switch v {
		case 0:
			s = "MQRECORDING_DISABLED"
		case 1:
			s = "MQRECORDING_Q"
		case 2:
			s = "MQRECORDING_MSG"
		default:
			s = ""
		}

	case "REGO":
		switch v {
		case 0:
			s = "MQREGO_NONE"
		case 1:
			s = "MQREGO_CORREL_ID_AS_IDENTITY"
		case 2:
			s = "MQREGO_ANONYMOUS"
		case 4:
			s = "MQREGO_LOCAL"
		case 8:
			s = "MQREGO_DIRECT_REQUESTS"
		case 16:
			s = "MQREGO_NEW_PUBLICATIONS_ONLY"
		case 32:
			s = "MQREGO_PUBLISH_ON_REQUEST_ONLY"
		case 64:
			s = "MQREGO_DEREGISTER_ALL"
		case 128:
			s = "MQREGO_INCLUDE_STREAM_NAME"
		case 256:
			s = "MQREGO_INFORM_IF_RETAINED"
		case 512:
			s = "MQREGO_DUPLICATES_OK"
		case 1024:
			s = "MQREGO_NON_PERSISTENT"
		case 2048:
			s = "MQREGO_PERSISTENT"
		case 4096:
			s = "MQREGO_PERSISTENT_AS_PUBLISH"
		case 8192:
			s = "MQREGO_PERSISTENT_AS_Q"
		case 16384:
			s = "MQREGO_ADD_NAME"
		case 32768:
			s = "MQREGO_NO_ALTERATION"
		case 65536:
			s = "MQREGO_FULL_RESPONSE"
		case 131072:
			s = "MQREGO_JOIN_SHARED"
		case 262144:
			s = "MQREGO_JOIN_EXCLUSIVE"
		case 524288:
			s = "MQREGO_LEAVE_ONLY"
		case 1048576:
			s = "MQREGO_VARIABLE_USER_ID"
		case 2097152:
			s = "MQREGO_LOCKED"
		default:
			s = ""
		}

	case "REORG":
		switch v {
		case 0:
			s = "MQREORG_DISABLED"
		case 1:
			s = "MQREORG_ENABLED"
		default:
			s = ""
		}

	case "RFH":
		switch v {
		case -65536:
			s = "MQRFH_FLAGS_RESTRICTED_MASK"
		case 0:
			s = "MQRFH_NONE"
		default:
			s = ""
		}

	case "RL":
		switch v {
		case -1:
			s = "MQRL_UNDEFINED"
		default:
			s = ""
		}

	case "RMHF":
		switch v {
		case 0:
			s = "MQRMHF_NOT_LAST"
		case 1:
			s = "MQRMHF_LAST"
		default:
			s = ""
		}

	case "ROUTE":
		switch v {
		case -65536:
			s = "MQROUTE_DELIVER_REJ_UNSUP_MASK"
		case 0:
			s = "MQROUTE_UNLIMITED_ACTIVITIES"
		case 2:
			s = "MQROUTE_DETAIL_LOW"
		case 8:
			s = "MQROUTE_DETAIL_MEDIUM"
		case 32:
			s = "MQROUTE_DETAIL_HIGH"
		case 256:
			s = "MQROUTE_FORWARD_ALL"
		case 512:
			s = "MQROUTE_FORWARD_IF_SUPPORTED"
		case 4096:
			s = "MQROUTE_DELIVER_YES"
		case 8192:
			s = "MQROUTE_DELIVER_NO"
		case 65539:
			s = "MQROUTE_ACCUMULATE_NONE"
		case 65540:
			s = "MQROUTE_ACCUMULATE_IN_MSG"
		case 65541:
			s = "MQROUTE_ACCUMULATE_AND_REPLY"
		default:
			s = ""
		}

	case "RO":
		switch v {
		case -270532353:
			s = "MQRO_ACCEPT_UNSUP_MASK"
		case 0:
			s = "MQRO_NONE"
		case 1:
			s = "MQRO_PAN"
		case 2:
			s = "MQRO_NAN"
		case 4:
			s = "MQRO_ACTIVITY"
		case 64:
			s = "MQRO_PASS_CORREL_ID"
		case 128:
			s = "MQRO_PASS_MSG_ID"
		case 256:
			s = "MQRO_COA"
		case 768:
			s = "MQRO_COA_WITH_DATA"
		case 1792:
			s = "MQRO_COA_WITH_FULL_DATA"
		case 2048:
			s = "MQRO_COD"
		case 6144:
			s = "MQRO_COD_WITH_DATA"
		case 14336:
			s = "MQRO_COD_WITH_FULL_DATA"
		case 16384:
			s = "MQRO_PASS_DISCARD_AND_EXPIRY"
		case 261888:
			s = "MQRO_ACCEPT_UNSUP_IF_XMIT_MASK"
		case 2097152:
			s = "MQRO_EXPIRATION"
		case 6291456:
			s = "MQRO_EXPIRATION_WITH_DATA"
		case 14680064:
			s = "MQRO_EXPIRATION_WITH_FULL_DATA"
		case 16777216:
			s = "MQRO_EXCEPTION"
		case 50331648:
			s = "MQRO_EXCEPTION_WITH_DATA"
		case 117440512:
			s = "MQRO_EXCEPTION_WITH_FULL_DATA"
		case 134217728:
			s = "MQRO_DISCARD_MSG"
		case 270270464:
			s = "MQRO_REJECT_UNSUP_MASK"
		default:
			s = ""
		}

	case "RP":
		switch v {
		case 0:
			s = "MQRP_NO"
		case 1:
			s = "MQRP_YES"
		default:
			s = ""
		}

	case "RQ":
		switch v {
		case 1:
			s = "MQRQ_CONN_NOT_AUTHORIZED"
		case 2:
			s = "MQRQ_OPEN_NOT_AUTHORIZED"
		case 3:
			s = "MQRQ_CLOSE_NOT_AUTHORIZED"
		case 4:
			s = "MQRQ_CMD_NOT_AUTHORIZED"
		case 5:
			s = "MQRQ_Q_MGR_STOPPING"
		case 6:
			s = "MQRQ_Q_MGR_QUIESCING"
		case 7:
			s = "MQRQ_CHANNEL_STOPPED_OK"
		case 8:
			s = "MQRQ_CHANNEL_STOPPED_ERROR"
		case 9:
			s = "MQRQ_CHANNEL_STOPPED_RETRY"
		case 10:
			s = "MQRQ_CHANNEL_STOPPED_DISABLED"
		case 11:
			s = "MQRQ_BRIDGE_STOPPED_OK"
		case 12:
			s = "MQRQ_BRIDGE_STOPPED_ERROR"
		case 13:
			s = "MQRQ_SSL_HANDSHAKE_ERROR"
		case 14:
			s = "MQRQ_SSL_CIPHER_SPEC_ERROR"
		case 15:
			s = "MQRQ_SSL_CLIENT_AUTH_ERROR"
		case 16:
			s = "MQRQ_SSL_PEER_NAME_ERROR"
		case 17:
			s = "MQRQ_SUB_NOT_AUTHORIZED"
		case 18:
			s = "MQRQ_SUB_DEST_NOT_AUTHORIZED"
		case 19:
			s = "MQRQ_SSL_UNKNOWN_REVOCATION"
		case 20:
			s = "MQRQ_SYS_CONN_NOT_AUTHORIZED"
		case 21:
			s = "MQRQ_CHANNEL_BLOCKED_ADDRESS"
		case 22:
			s = "MQRQ_CHANNEL_BLOCKED_USERID"
		case 23:
			s = "MQRQ_CHANNEL_BLOCKED_NOACCESS"
		case 24:
			s = "MQRQ_MAX_ACTIVE_CHANNELS"
		case 25:
			s = "MQRQ_MAX_CHANNELS"
		case 26:
			s = "MQRQ_SVRCONN_INST_LIMIT"
		case 27:
			s = "MQRQ_CLIENT_INST_LIMIT"
		case 28:
			s = "MQRQ_CAF_NOT_INSTALLED"
		case 29:
			s = "MQRQ_CSP_NOT_AUTHORIZED"
		case 30:
			s = "MQRQ_FAILOVER_PERMITTED"
		case 31:
			s = "MQRQ_FAILOVER_NOT_PERMITTED"
		case 32:
			s = "MQRQ_STANDBY_ACTIVATED"
		case 33:
			s = "MQRQ_REPLICA_ACTIVATED"
		default:
			s = ""
		}

	case "RT":
		switch v {
		case 1:
			s = "MQRT_CONFIGURATION"
		case 2:
			s = "MQRT_EXPIRY"
		case 3:
			s = "MQRT_NSPROC"
		case 4:
			s = "MQRT_PROXYSUB"
		case 5:
			s = "MQRT_SUB_CONFIGURATION"
		default:
			s = ""
		}

	case "RU":
		switch v {
		case 1:
			s = "MQRU_PUBLISH_ON_REQUEST"
		case 2:
			s = "MQRU_PUBLISH_ALL"
		default:
			s = ""
		}

	case "SCA":
		switch v {
		case 0:
			s = "MQSCA_REQUIRED"
		case 1:
			s = "MQSCA_OPTIONAL"
		case 2:
			s = "MQSCA_NEVER_REQUIRED"
		default:
			s = ""
		}

	case "SCOPE":
		switch v {
		case 0:
			s = "MQSCOPE_ALL"
		case 1:
			s = "MQSCOPE_AS_PARENT"
		case 4:
			s = "MQSCOPE_QMGR"
		default:
			s = ""
		}

	case "SCO":
		switch v {
		case 1:
			s = "MQSCO_Q_MGR"
		case 2:
			s = "MQSCO_CELL"
		default:
			s = ""
		}

	case "SCYC":
		switch v {
		case 0:
			s = "MQSCYC_UPPER"
		case 1:
			s = "MQSCYC_MIXED"
		default:
			s = ""
		}

	case "SECCOMM":
		switch v {
		case 0:
			s = "MQSECCOMM_NO"
		case 1:
			s = "MQSECCOMM_YES"
		case 2:
			s = "MQSECCOMM_ANON"
		default:
			s = ""
		}

	case "SECITEM":
		switch v {
		case 0:
			s = "MQSECITEM_ALL"
		case 1:
			s = "MQSECITEM_MQADMIN"
		case 2:
			s = "MQSECITEM_MQNLIST"
		case 3:
			s = "MQSECITEM_MQPROC"
		case 4:
			s = "MQSECITEM_MQQUEUE"
		case 5:
			s = "MQSECITEM_MQCONN"
		case 6:
			s = "MQSECITEM_MQCMDS"
		case 7:
			s = "MQSECITEM_MXADMIN"
		case 8:
			s = "MQSECITEM_MXNLIST"
		case 9:
			s = "MQSECITEM_MXPROC"
		case 10:
			s = "MQSECITEM_MXQUEUE"
		case 11:
			s = "MQSECITEM_MXTOPIC"
		default:
			s = ""
		}

	case "SECPROT":
		switch v {
		case 0:
			s = "MQSECPROT_NONE"
		case 1:
			s = "MQSECPROT_SSLV30"
		case 2:
			s = "MQSECPROT_TLSV10"
		case 4:
			s = "MQSECPROT_TLSV12"
		case 8:
			s = "MQSECPROT_TLSV13"
		default:
			s = ""
		}

	case "SECSW":
		switch v {
		case 1:
			s = "MQSECSW_PROCESS"
		case 2:
			s = "MQSECSW_NAMELIST"
		case 3:
			s = "MQSECSW_Q"
		case 4:
			s = "MQSECSW_TOPIC"
		case 6:
			s = "MQSECSW_CONTEXT"
		case 7:
			s = "MQSECSW_ALTERNATE_USER"
		case 8:
			s = "MQSECSW_COMMAND"
		case 9:
			s = "MQSECSW_CONNECTION"
		case 10:
			s = "MQSECSW_SUBSYSTEM"
		case 11:
			s = "MQSECSW_COMMAND_RESOURCES"
		case 15:
			s = "MQSECSW_Q_MGR"
		case 16:
			s = "MQSECSW_QSG"
		case 21:
			s = "MQSECSW_OFF_FOUND"
		case 22:
			s = "MQSECSW_ON_FOUND"
		case 23:
			s = "MQSECSW_OFF_NOT_FOUND"
		case 24:
			s = "MQSECSW_ON_NOT_FOUND"
		case 25:
			s = "MQSECSW_OFF_ERROR"
		case 26:
			s = "MQSECSW_ON_OVERRIDDEN"
		default:
			s = ""
		}

	case "SECTYPE":
		switch v {
		case 1:
			s = "MQSECTYPE_AUTHSERV"
		case 2:
			s = "MQSECTYPE_SSL"
		case 3:
			s = "MQSECTYPE_CLASSES"
		case 4:
			s = "MQSECTYPE_CONNAUTH"
		default:
			s = ""
		}

	case "SELTYPE":
		switch v {
		case 0:
			s = "MQSELTYPE_NONE"
		case 1:
			s = "MQSELTYPE_STANDARD"
		case 2:
			s = "MQSELTYPE_EXTENDED"
		default:
			s = ""
		}

	case "SEL_ALL":
		switch v {
		case -30003:
			s = "MQSEL_ALL_SYSTEM_SELECTORS"
		case -30002:
			s = "MQSEL_ALL_USER_SELECTORS"
		case -30001:
			s = "MQSEL_ALL_SELECTORS"
		default:
			s = ""
		}

	case "SEL_ANY":
		switch v {
		case -30003:
			s = "MQSEL_ANY_SYSTEM_SELECTOR"
		case -30002:
			s = "MQSEL_ANY_USER_SELECTOR"
		case -30001:
			s = "MQSEL_ANY_SELECTOR"
		default:
			s = ""
		}

	case "SMPO":
		switch v {
		case 0:
			s = "MQSMPO_NONE"
		case 1:
			s = "MQSMPO_SET_PROP_UNDER_CURSOR"
		case 2:
			s = "MQSMPO_SET_PROP_AFTER_CURSOR"
		case 4:
			s = "MQSMPO_APPEND_PROPERTY"
		case 8:
			s = "MQSMPO_SET_PROP_BEFORE_CURSOR"
		default:
			s = ""
		}

	case "SO":
		switch v {
		case 0:
			s = "MQSO_NONE"
		case 1:
			s = "MQSO_ALTER"
		case 2:
			s = "MQSO_CREATE"
		case 4:
			s = "MQSO_RESUME"
		case 8:
			s = "MQSO_DURABLE"
		case 16:
			s = "MQSO_GROUP_SUB"
		case 32:
			s = "MQSO_MANAGED"
		case 64:
			s = "MQSO_SET_IDENTITY_CONTEXT"
		case 128:
			s = "MQSO_NO_MULTICAST"
		case 256:
			s = "MQSO_FIXED_USERID"
		case 512:
			s = "MQSO_ANY_USERID"
		case 2048:
			s = "MQSO_PUBLICATIONS_ON_REQUEST"
		case 4096:
			s = "MQSO_NEW_PUBLICATIONS_ONLY"
		case 8192:
			s = "MQSO_FAIL_IF_QUIESCING"
		case 262144:
			s = "MQSO_ALTERNATE_USER_AUTHORITY"
		case 1048576:
			s = "MQSO_WILDCARD_CHAR"
		case 2097152:
			s = "MQSO_WILDCARD_TOPIC"
		case 4194304:
			s = "MQSO_SET_CORREL_ID"
		case 67108864:
			s = "MQSO_SCOPE_QMGR"
		case 134217728:
			s = "MQSO_NO_READ_AHEAD"
		case 268435456:
			s = "MQSO_READ_AHEAD"
		default:
			s = ""
		}

	case "SPL":
		switch v {
		case 0:
			s = "MQSPL_PASSTHRU"
		case 1:
			s = "MQSPL_REMOVE"
		case 2:
			s = "MQSPL_AS_POLICY"
		default:
			s = ""
		}

	case "SP":
		switch v {
		case 0:
			s = "MQSP_NOT_AVAILABLE"
		case 1:
			s = "MQSP_AVAILABLE"
		default:
			s = ""
		}

	case "SQQM":
		switch v {
		case 0:
			s = "MQSQQM_USE"
		case 1:
			s = "MQSQQM_IGNORE"
		default:
			s = ""
		}

	case "SRO":
		switch v {
		case 0:
			s = "MQSRO_NONE"
		case 8192:
			s = "MQSRO_FAIL_IF_QUIESCING"
		default:
			s = ""
		}

	case "SR":
		switch v {
		case 1:
			s = "MQSR_ACTION_PUBLICATION"
		default:
			s = ""
		}

	case "SSL":
		switch v {
		case 0:
			s = "MQSSL_FIPS_NO"
		case 1:
			s = "MQSSL_FIPS_YES"
		default:
			s = ""
		}

	case "STAT":
		switch v {
		case 0:
			s = "MQSTAT_TYPE_ASYNC_ERROR"
		case 1:
			s = "MQSTAT_TYPE_RECONNECTION"
		case 2:
			s = "MQSTAT_TYPE_RECONNECTION_ERROR"
		default:
			s = ""
		}

	case "STDBY":
		switch v {
		case 0:
			s = "MQSTDBY_NOT_PERMITTED"
		case 1:
			s = "MQSTDBY_PERMITTED"
		default:
			s = ""
		}

	case "ST":
		switch v {
		case 0:
			s = "MQST_BEST_EFFORT"
		case 1:
			s = "MQST_MUST_DUP"
		default:
			s = ""
		}

	case "SUBTYPE":
		switch v {
		case -2:
			s = "MQSUBTYPE_USER"
		case -1:
			s = "MQSUBTYPE_ALL"
		case 1:
			s = "MQSUBTYPE_API"
		case 2:
			s = "MQSUBTYPE_ADMIN"
		case 3:
			s = "MQSUBTYPE_PROXY"
		default:
			s = ""
		}

	case "SUB_DURABILITY":
		switch v {
		case -1:
			s = "MQSUB_DURABLE_ALL"
		case 1:
			s = "MQSUB_DURABLE_YES"
		case 2:
			s = "MQSUB_DURABLE_NO"
		default:
			s = ""
		}

	case "SUB":
		switch v {
		case 0:
			s = "MQSUB_DURABLE_AS_PARENT"
		case 1:
			s = "MQSUB_DURABLE_ALLOWED"
		case 2:
			s = "MQSUB_DURABLE_INHIBITED"
		default:
			s = ""
		}

	case "SUS":
		switch v {
		case 0:
			s = "MQSUS_NO"
		case 1:
			s = "MQSUS_YES"
		default:
			s = ""
		}

	case "SVC_CONTROL":
		switch v {
		case 0:
			s = "MQSVC_CONTROL_Q_MGR"
		case 1:
			s = "MQSVC_CONTROL_Q_MGR_START"
		case 2:
			s = "MQSVC_CONTROL_MANUAL"
		default:
			s = ""
		}

	case "SVC_STATUS":
		switch v {
		case 0:
			s = "MQSVC_STATUS_STOPPED"
		case 1:
			s = "MQSVC_STATUS_STARTING"
		case 2:
			s = "MQSVC_STATUS_RUNNING"
		case 3:
			s = "MQSVC_STATUS_STOPPING"
		case 4:
			s = "MQSVC_STATUS_RETRYING"
		default:
			s = ""
		}

	case "SVC_TYPE":
		switch v {
		case 0:
			s = "MQSVC_TYPE_COMMAND"
		case 1:
			s = "MQSVC_TYPE_SERVER"
		default:
			s = ""
		}

	case "SYNCPOINT":
		switch v {
		case 0:
			s = "MQSYNCPOINT_YES"
		case 1:
			s = "MQSYNCPOINT_IFPER"
		default:
			s = ""
		}

	case "SYSOBJ":
		switch v {
		case 0:
			s = "MQSYSOBJ_YES"
		case 1:
			s = "MQSYSOBJ_NO"
		default:
			s = ""
		}

	case "SYSP":
		switch v {
		case 0:
			s = "MQSYSP_NO"
		case 1:
			s = "MQSYSP_YES"
		case 2:
			s = "MQSYSP_EXTENDED"
		case 10:
			s = "MQSYSP_TYPE_INITIAL"
		case 11:
			s = "MQSYSP_TYPE_SET"
		case 12:
			s = "MQSYSP_TYPE_LOG_COPY"
		case 13:
			s = "MQSYSP_TYPE_LOG_STATUS"
		case 14:
			s = "MQSYSP_TYPE_ARCHIVE_TAPE"
		case 20:
			s = "MQSYSP_ALLOC_BLK"
		case 21:
			s = "MQSYSP_ALLOC_TRK"
		case 22:
			s = "MQSYSP_ALLOC_CYL"
		case 30:
			s = "MQSYSP_STATUS_BUSY"
		case 31:
			s = "MQSYSP_STATUS_PREMOUNT"
		case 32:
			s = "MQSYSP_STATUS_AVAILABLE"
		case 33:
			s = "MQSYSP_STATUS_UNKNOWN"
		case 34:
			s = "MQSYSP_STATUS_ALLOC_ARCHIVE"
		case 35:
			s = "MQSYSP_STATUS_COPYING_BSDS"
		case 36:
			s = "MQSYSP_STATUS_COPYING_LOG"
		default:
			s = ""
		}

	case "S_AVAIL":
		switch v {
		case 0:
			s = "MQS_AVAIL_NORMAL"
		case 1:
			s = "MQS_AVAIL_ERROR"
		case 2:
			s = "MQS_AVAIL_STOPPED"
		default:
			s = ""
		}

	case "S_EXPANDST":
		switch v {
		case 0:
			s = "MQS_EXPANDST_NORMAL"
		case 1:
			s = "MQS_EXPANDST_FAILED"
		case 2:
			s = "MQS_EXPANDST_MAXIMUM"
		default:
			s = ""
		}

	case "S_OPENMODE":
		switch v {
		case 0:
			s = "MQS_OPENMODE_NONE"
		case 1:
			s = "MQS_OPENMODE_READONLY"
		case 2:
			s = "MQS_OPENMODE_UPDATE"
		case 3:
			s = "MQS_OPENMODE_RECOVERY"
		default:
			s = ""
		}

	case "S_STATUS":
		switch v {
		case 0:
			s = "MQS_STATUS_CLOSED"
		case 1:
			s = "MQS_STATUS_CLOSING"
		case 2:
			s = "MQS_STATUS_OPENING"
		case 3:
			s = "MQS_STATUS_OPEN"
		case 4:
			s = "MQS_STATUS_NOTENABLED"
		case 5:
			s = "MQS_STATUS_ALLOCFAIL"
		case 6:
			s = "MQS_STATUS_OPENFAIL"
		case 7:
			s = "MQS_STATUS_STGFAIL"
		case 8:
			s = "MQS_STATUS_DATAFAIL"
		default:
			s = ""
		}

	case "TA":
		switch v {
		case 1:
			s = "MQTA_BLOCK"
		case 2:
			s = "MQTA_PASSTHRU"
		default:
			s = ""
		}

	case "TA_PROXY":
		switch v {
		case 1:
			s = "MQTA_PROXY_SUB_FORCE"
		case 2:
			s = "MQTA_PROXY_SUB_FIRSTUSE"
		default:
			s = ""
		}

	case "TA_PUB":
		switch v {
		case 0:
			s = "MQTA_PUB_AS_PARENT"
		case 1:
			s = "MQTA_PUB_INHIBITED"
		case 2:
			s = "MQTA_PUB_ALLOWED"
		default:
			s = ""
		}

	case "TA_SUB":
		switch v {
		case 0:
			s = "MQTA_SUB_AS_PARENT"
		case 1:
			s = "MQTA_SUB_INHIBITED"
		case 2:
			s = "MQTA_SUB_ALLOWED"
		default:
			s = ""
		}

	case "TCPKEEP":
		switch v {
		case 0:
			s = "MQTCPKEEP_NO"
		case 1:
			s = "MQTCPKEEP_YES"
		default:
			s = ""
		}

	case "TCPSTACK":
		switch v {
		case 0:
			s = "MQTCPSTACK_SINGLE"
		case 1:
			s = "MQTCPSTACK_MULTIPLE"
		default:
			s = ""
		}

	case "TC":
		switch v {
		case 0:
			s = "MQTC_OFF"
		case 1:
			s = "MQTC_ON"
		default:
			s = ""
		}

	case "TIME":
		switch v {
		case 0:
			s = "MQTIME_UNIT_MINS"
		case 1:
			s = "MQTIME_UNIT_SECS"
		default:
			s = ""
		}

	case "TOPT":
		switch v {
		case 0:
			s = "MQTOPT_LOCAL"
		case 1:
			s = "MQTOPT_CLUSTER"
		case 2:
			s = "MQTOPT_ALL"
		default:
			s = ""
		}

	case "TRAXSTR":
		switch v {
		case 0:
			s = "MQTRAXSTR_NO"
		case 1:
			s = "MQTRAXSTR_YES"
		default:
			s = ""
		}

	case "TRIGGER":
		switch v {
		case 0:
			s = "MQTRIGGER_RESTART_NO"
		case 1:
			s = "MQTRIGGER_RESTART_YES"
		default:
			s = ""
		}

	case "TSCOPE":
		switch v {
		case 1:
			s = "MQTSCOPE_QMGR"
		case 2:
			s = "MQTSCOPE_ALL"
		default:
			s = ""
		}

	case "TT":
		switch v {
		case 0:
			s = "MQTT_NONE"
		case 1:
			s = "MQTT_FIRST"
		case 2:
			s = "MQTT_EVERY"
		case 3:
			s = "MQTT_DEPTH"
		default:
			s = ""
		}

	case "TYPE":
		switch v {
		case 0:
			s = "MQTYPE_AS_SET"
		case 2:
			s = "MQTYPE_NULL"
		case 4:
			s = "MQTYPE_BOOLEAN"
		case 8:
			s = "MQTYPE_BYTE_STRING"
		case 16:
			s = "MQTYPE_INT8"
		case 32:
			s = "MQTYPE_INT16"
		case 64:
			s = "MQTYPE_INT32"
		case 128:
			s = "MQTYPE_INT64"
		case 256:
			s = "MQTYPE_FLOAT32"
		case 512:
			s = "MQTYPE_FLOAT64"
		case 1024:
			s = "MQTYPE_STRING"
		default:
			s = ""
		}

	case "UCI":
		switch v {
		case 0:
			s = "MQUCI_NO"
		case 1:
			s = "MQUCI_YES"
		default:
			s = ""
		}

	case "UIDSUPP":
		switch v {
		case 0:
			s = "MQUIDSUPP_NO"
		case 1:
			s = "MQUIDSUPP_YES"
		default:
			s = ""
		}

	case "UNDELIVERED":
		switch v {
		case 0:
			s = "MQUNDELIVERED_NORMAL"
		case 1:
			s = "MQUNDELIVERED_SAFE"
		case 2:
			s = "MQUNDELIVERED_DISCARD"
		case 3:
			s = "MQUNDELIVERED_KEEP"
		default:
			s = ""
		}

	case "UOWST":
		switch v {
		case 0:
			s = "MQUOWST_NONE"
		case 1:
			s = "MQUOWST_ACTIVE"
		case 2:
			s = "MQUOWST_PREPARED"
		case 3:
			s = "MQUOWST_UNRESOLVED"
		default:
			s = ""
		}

	case "UOWT":
		switch v {
		case 0:
			s = "MQUOWT_Q_MGR"
		case 1:
			s = "MQUOWT_CICS"
		case 2:
			s = "MQUOWT_RRS"
		case 3:
			s = "MQUOWT_IMS"
		case 4:
			s = "MQUOWT_XA"
		default:
			s = ""
		}

	case "USAGE_DS":
		switch v {
		case 10:
			s = "MQUSAGE_DS_OLDEST_ACTIVE_UOW"
		case 11:
			s = "MQUSAGE_DS_OLDEST_PS_RECOVERY"
		case 12:
			s = "MQUSAGE_DS_OLDEST_CF_RECOVERY"
		default:
			s = ""
		}

	case "USAGE_EXPAND":
		switch v {
		case 1:
			s = "MQUSAGE_EXPAND_USER"
		case 2:
			s = "MQUSAGE_EXPAND_SYSTEM"
		case 3:
			s = "MQUSAGE_EXPAND_NONE"
		default:
			s = ""
		}

	case "USAGE_PS":
		switch v {
		case 0:
			s = "MQUSAGE_PS_AVAILABLE"
		case 1:
			s = "MQUSAGE_PS_DEFINED"
		case 2:
			s = "MQUSAGE_PS_OFFLINE"
		case 3:
			s = "MQUSAGE_PS_NOT_DEFINED"
		case 4:
			s = "MQUSAGE_PS_SUSPENDED"
		default:
			s = ""
		}

	case "USAGE_SMDS":
		switch v {
		case 0:
			s = "MQUSAGE_SMDS_AVAILABLE"
		case 1:
			s = "MQUSAGE_SMDS_NO_DATA"
		default:
			s = ""
		}

	case "USEDLQ":
		switch v {
		case 0:
			s = "MQUSEDLQ_AS_PARENT"
		case 1:
			s = "MQUSEDLQ_NO"
		case 2:
			s = "MQUSEDLQ_YES"
		default:
			s = ""
		}

	case "USRC":
		switch v {
		case 0:
			s = "MQUSRC_MAP"
		case 1:
			s = "MQUSRC_NOACCESS"
		case 2:
			s = "MQUSRC_CHANNEL"
		default:
			s = ""
		}

	case "US":
		switch v {
		case 0:
			s = "MQUS_NORMAL"
		case 1:
			s = "MQUS_TRANSMISSION"
		default:
			s = ""
		}

	case "VL":
		switch v {
		case -1:
			s = "MQVL_NULL_TERMINATED"
		case 0:
			s = "MQVL_EMPTY_STRING"
		default:
			s = ""
		}

	case "VS":
		switch v {
		case -1:
			s = "MQVS_NULL_TERMINATED"
		default:
			s = ""
		}

	case "VU":
		switch v {
		case 1:
			s = "MQVU_FIXED_USER"
		case 2:
			s = "MQVU_ANY_USER"
		default:
			s = ""
		}

	case "WARN":
		switch v {
		case 0:
			s = "MQWARN_NO"
		case 1:
			s = "MQWARN_YES"
		default:
			s = ""
		}

	case "WIH":
		switch v {
		case 0:
			s = "MQWIH_NONE"
		default:
			s = ""
		}

	case "WI":
		switch v {
		case -1:
			s = "MQWI_UNLIMITED"
		default:
			s = ""
		}

	case "WS":
		switch v {
		case 0:
			s = "MQWS_DEFAULT"
		case 1:
			s = "MQWS_CHAR"
		case 2:
			s = "MQWS_TOPIC"
		default:
			s = ""
		}

	case "WXP":
		switch v {
		case 2:
			s = "MQWXP_PUT_BY_CLUSTER_CHL"
		default:
			s = ""
		}

	case "XACT":
		switch v {
		case 1:
			s = "MQXACT_EXTERNAL"
		case 2:
			s = "MQXACT_INTERNAL"
		default:
			s = ""
		}

	case "XCC":
		switch v {
		case -8:
			s = "MQXCC_FAILED"
		case -7:
			s = "MQXCC_REQUEST_ACK"
		case -6:
			s = "MQXCC_CLOSE_CHANNEL"
		case -5:
			s = "MQXCC_SUPPRESS_EXIT"
		case -4:
			s = "MQXCC_SEND_SEC_MSG"
		case -3:
			s = "MQXCC_SEND_AND_REQUEST_SEC_MSG"
		case -2:
			s = "MQXCC_SKIP_FUNCTION"
		case -1:
			s = "MQXCC_SUPPRESS_FUNCTION"
		case 0:
			s = "MQXCC_OK"
		default:
			s = ""
		}

	case "XC":
		switch v {
		case 1:
			s = "MQXC_MQOPEN"
		case 2:
			s = "MQXC_MQCLOSE"
		case 3:
			s = "MQXC_MQGET"
		case 4:
			s = "MQXC_MQPUT"
		case 5:
			s = "MQXC_MQPUT1"
		case 6:
			s = "MQXC_MQINQ"
		case 8:
			s = "MQXC_MQSET"
		case 9:
			s = "MQXC_MQBACK"
		case 10:
			s = "MQXC_MQCMIT"
		case 42:
			s = "MQXC_MQSUB"
		case 43:
			s = "MQXC_MQSUBRQ"
		case 44:
			s = "MQXC_MQCB"
		case 45:
			s = "MQXC_MQCTL"
		case 46:
			s = "MQXC_MQSTAT"
		case 48:
			s = "MQXC_CALLBACK"
		default:
			s = ""
		}

	case "XDR":
		switch v {
		case 0:
			s = "MQXDR_OK"
		case 1:
			s = "MQXDR_CONVERSION_FAILED"
		default:
			s = ""
		}

	case "XEPO":
		switch v {
		case 0:
			s = "MQXEPO_NONE"
		default:
			s = ""
		}

	case "XE":
		switch v {
		case 0:
			s = "MQXE_OTHER"
		case 1:
			s = "MQXE_MCA"
		case 2:
			s = "MQXE_MCA_SVRCONN"
		case 3:
			s = "MQXE_COMMAND_SERVER"
		case 4:
			s = "MQXE_MQSC"
		case 5:
			s = "MQXE_MCA_CLNTCONN"
		default:
			s = ""
		}

	case "XF":
		switch v {
		case 1:
			s = "MQXF_INIT"
		case 2:
			s = "MQXF_TERM"
		case 3:
			s = "MQXF_CONN"
		case 4:
			s = "MQXF_CONNX"
		case 5:
			s = "MQXF_DISC"
		case 6:
			s = "MQXF_OPEN"
		case 7:
			s = "MQXF_CLOSE"
		case 8:
			s = "MQXF_PUT1"
		case 9:
			s = "MQXF_PUT"
		case 10:
			s = "MQXF_GET"
		case 11:
			s = "MQXF_DATA_CONV_ON_GET"
		case 12:
			s = "MQXF_INQ"
		case 13:
			s = "MQXF_SET"
		case 14:
			s = "MQXF_BEGIN"
		case 15:
			s = "MQXF_CMIT"
		case 16:
			s = "MQXF_BACK"
		case 18:
			s = "MQXF_STAT"
		case 19:
			s = "MQXF_CB"
		case 20:
			s = "MQXF_CTL"
		case 21:
			s = "MQXF_CALLBACK"
		case 22:
			s = "MQXF_SUB"
		case 23:
			s = "MQXF_SUBRQ"
		case 24:
			s = "MQXF_XACLOSE"
		case 25:
			s = "MQXF_XACOMMIT"
		case 26:
			s = "MQXF_XACOMPLETE"
		case 27:
			s = "MQXF_XAEND"
		case 28:
			s = "MQXF_XAFORGET"
		case 29:
			s = "MQXF_XAOPEN"
		case 30:
			s = "MQXF_XAPREPARE"
		case 31:
			s = "MQXF_XARECOVER"
		case 32:
			s = "MQXF_XAROLLBACK"
		case 33:
			s = "MQXF_XASTART"
		case 34:
			s = "MQXF_AXREG"
		case 35:
			s = "MQXF_AXUNREG"
		default:
			s = ""
		}

	case "XPT":
		switch v {
		case -1:
			s = "MQXPT_ALL"
		case 0:
			s = "MQXPT_LOCAL"
		case 1:
			s = "MQXPT_LU62"
		case 2:
			s = "MQXPT_TCP"
		case 3:
			s = "MQXPT_NETBIOS"
		case 4:
			s = "MQXPT_SPX"
		case 5:
			s = "MQXPT_DECNET"
		case 6:
			s = "MQXPT_UDP"
		default:
			s = ""
		}

	case "XR2":
		switch v {
		case 0:
			s = "MQXR2_DEFAULT_CONTINUATION"
		case 1:
			s = "MQXR2_PUT_WITH_DEF_USERID"
		case 2:
			s = "MQXR2_PUT_WITH_MSG_USERID"
		case 4:
			s = "MQXR2_USE_EXIT_BUFFER"
		case 8:
			s = "MQXR2_CONTINUE_CHAIN"
		case 16:
			s = "MQXR2_SUPPRESS_CHAIN"
		case 32:
			s = "MQXR2_DYNAMIC_CACHE"
		default:
			s = ""
		}

	case "XR":
		switch v {
		case 1:
			s = "MQXR_BEFORE"
		case 2:
			s = "MQXR_AFTER"
		case 3:
			s = "MQXR_CONNECTION"
		case 4:
			s = "MQXR_BEFORE_CONVERT"
		case 11:
			s = "MQXR_INIT"
		case 12:
			s = "MQXR_TERM"
		case 13:
			s = "MQXR_MSG"
		case 14:
			s = "MQXR_XMIT"
		case 15:
			s = "MQXR_SEC_MSG"
		case 16:
			s = "MQXR_INIT_SEC"
		case 17:
			s = "MQXR_RETRY"
		case 18:
			s = "MQXR_AUTO_CLUSSDR"
		case 19:
			s = "MQXR_AUTO_RECEIVER"
		case 20:
			s = "MQXR_CLWL_OPEN"
		case 21:
			s = "MQXR_CLWL_PUT"
		case 22:
			s = "MQXR_CLWL_MOVE"
		case 23:
			s = "MQXR_CLWL_REPOS"
		case 24:
			s = "MQXR_CLWL_REPOS_MOVE"
		case 25:
			s = "MQXR_END_BATCH"
		case 26:
			s = "MQXR_ACK_RECEIVED"
		case 27:
			s = "MQXR_AUTO_SVRCONN"
		case 28:
			s = "MQXR_AUTO_CLUSRCVR"
		case 29:
			s = "MQXR_SEC_PARMS"
		case 30:
			s = "MQXR_PUBLICATION"
		case 31:
			s = "MQXR_PRECONNECT"
		default:
			s = ""
		}

	case "XT":
		switch v {
		case 1:
			s = "MQXT_API_CROSSING_EXIT"
		case 2:
			s = "MQXT_API_EXIT"
		case 11:
			s = "MQXT_CHANNEL_SEC_EXIT"
		case 12:
			s = "MQXT_CHANNEL_MSG_EXIT"
		case 13:
			s = "MQXT_CHANNEL_SEND_EXIT"
		case 14:
			s = "MQXT_CHANNEL_RCV_EXIT"
		case 15:
			s = "MQXT_CHANNEL_MSG_RETRY_EXIT"
		case 16:
			s = "MQXT_CHANNEL_AUTO_DEF_EXIT"
		case 20:
			s = "MQXT_CLUSTER_WORKLOAD_EXIT"
		case 21:
			s = "MQXT_PUBSUB_ROUTING_EXIT"
		case 22:
			s = "MQXT_PUBLISH_EXIT"
		case 23:
			s = "MQXT_PRECONNECT_EXIT"
		default:
			s = ""
		}

	case "ZAET":
		switch v {
		case 0:
			s = "MQZAET_NONE"
		case 1:
			s = "MQZAET_PRINCIPAL"
		case 2:
			s = "MQZAET_GROUP"
		case 3:
			s = "MQZAET_UNKNOWN"
		default:
			s = ""
		}

	case "ZAO":
		switch v {
		case 0:
			s = "MQZAO_NONE"
		case 1:
			s = "MQZAO_CONNECT"
		case 2:
			s = "MQZAO_BROWSE"
		case 4:
			s = "MQZAO_INPUT"
		case 8:
			s = "MQZAO_OUTPUT"
		case 16:
			s = "MQZAO_INQUIRE"
		case 32:
			s = "MQZAO_SET"
		case 64:
			s = "MQZAO_PASS_IDENTITY_CONTEXT"
		case 128:
			s = "MQZAO_PASS_ALL_CONTEXT"
		case 256:
			s = "MQZAO_SET_IDENTITY_CONTEXT"
		case 512:
			s = "MQZAO_SET_ALL_CONTEXT"
		case 1024:
			s = "MQZAO_ALTERNATE_USER_AUTHORITY"
		case 2048:
			s = "MQZAO_PUBLISH"
		case 4096:
			s = "MQZAO_SUBSCRIBE"
		case 8192:
			s = "MQZAO_RESUME"
		case 16383:
			s = "MQZAO_ALL_MQI"
		case 65536:
			s = "MQZAO_CREATE"
		case 131072:
			s = "MQZAO_DELETE"
		case 262144:
			s = "MQZAO_DISPLAY"
		case 524288:
			s = "MQZAO_CHANGE"
		case 1048576:
			s = "MQZAO_CLEAR"
		case 2097152:
			s = "MQZAO_CONTROL"
		case 4194304:
			s = "MQZAO_CONTROL_EXTENDED"
		case 8388608:
			s = "MQZAO_AUTHORIZE"
		case 16646144:
			s = "MQZAO_ALL_ADMIN"
		case 16777216:
			s = "MQZAO_REMOVE"
		case 33554432:
			s = "MQZAO_SYSTEM"
		case 50216959:
			s = "MQZAO_ALL"
		case 67108864:
			s = "MQZAO_CREATE_ONLY"
		default:
			s = ""
		}

	case "ZAT":
		switch v {
		case 0:
			s = "MQZAT_INITIAL_CONTEXT"
		case 1:
			s = "MQZAT_CHANGE_CONTEXT"
		default:
			s = ""
		}

	case "ZCI":
		switch v {
		case 0:
			s = "MQZCI_CONTINUE"
		case 1:
			s = "MQZCI_STOP"
		default:
			s = ""
		}

	case "ZID_AUTHORITY":
		switch v {
		case 0:
			s = "MQZID_INIT_AUTHORITY"
		case 1:
			s = "MQZID_TERM_AUTHORITY"
		case 2:
			s = "MQZID_CHECK_AUTHORITY"
		case 3:
			s = "MQZID_COPY_ALL_AUTHORITY"
		case 4:
			s = "MQZID_DELETE_AUTHORITY"
		case 5:
			s = "MQZID_SET_AUTHORITY"
		case 6:
			s = "MQZID_GET_AUTHORITY"
		case 7:
			s = "MQZID_GET_EXPLICIT_AUTHORITY"
		case 8:
			s = "MQZID_REFRESH_CACHE"
		case 9:
			s = "MQZID_ENUMERATE_AUTHORITY_DATA"
		case 10:
			s = "MQZID_AUTHENTICATE_USER"
		case 11:
			s = "MQZID_FREE_USER"
		case 12:
			s = "MQZID_INQUIRE"
		case 13:
			s = "MQZID_CHECK_PRIVILEGED"
		default:
			s = ""
		}

	case "ZID_NAME":
		switch v {
		case 0:
			s = "MQZID_INIT_NAME"
		case 1:
			s = "MQZID_TERM_NAME"
		case 2:
			s = "MQZID_LOOKUP_NAME"
		case 3:
			s = "MQZID_INSERT_NAME"
		case 4:
			s = "MQZID_DELETE_NAME"
		default:
			s = ""
		}

	case "ZID_USERID":
		switch v {
		case 0:
			s = "MQZID_INIT_USERID"
		case 1:
			s = "MQZID_TERM_USERID"
		case 2:
			s = "MQZID_FIND_USERID"
		default:
			s = ""
		}

	case "ZID":
		switch v {
		case 0:
			s = "MQZID_INIT"
		case 1:
			s = "MQZID_TERM"
		default:
			s = ""
		}

	case "ZIO":
		switch v {
		case 0:
			s = "MQZIO_PRIMARY"
		case 1:
			s = "MQZIO_SECONDARY"
		default:
			s = ""
		}

	case "ZSE":
		switch v {
		case 0:
			s = "MQZSE_CONTINUE"
		case 1:
			s = "MQZSE_START"
		default:
			s = ""
		}

	case "ZSL":
		switch v {
		case 0:
			s = "MQZSL_NOT_RETURNED"
		case 1:
			s = "MQZSL_RETURNED"
		default:
			s = ""
		}

	case "ZTO":
		switch v {
		case 0:
			s = "MQZTO_PRIMARY"
		case 1:
			s = "MQZTO_SECONDARY"
		default:
			s = ""
		}

	case "CERT":
		switch v {
		case 0:
			s = "MQ_CERT_VAL_POLICY_ANY"
		case 1:
			s = "MQ_CERT_VAL_POLICY_RFC5280"
		case 2:
			s = "MQ_CERT_VAL_POLICY_NONE"
		default:
			s = ""
		}

	case "HTTPSCERTREV":
		switch v {
		case 0:
			s = "MQ_HTTPSCERTREV_DEFAULT"
		case 1:
			s = "MQ_HTTPSCERTREV_REQUIRED"
		case 2:
			s = "MQ_HTTPSCERTREV_DISABLED"
		case 3:
			s = "MQ_HTTPSCERTREV_OPTIONAL"
		default:
			s = ""
		}

	case "HTTPSCERTVAL":
		switch v {
		case 0:
			s = "MQ_HTTPSCERTVAL_DEFAULT"
		case 1:
			s = "MQ_HTTPSCERTVAL_ANY"
		case 2:
			s = "MQ_HTTPSCERTVAL_NONE"
		case 3:
			s = "MQ_HTTPSCERTVAL_HOSTNAMECN"
		default:
			s = ""
		}

	case "MQTT":
		switch v {
		case 65536:
			s = "MQ_MQTT_MAX_KEEP_ALIVE"
		default:
			s = ""
		}

	case "SUITE":
		switch v {
		case 0:
			s = "MQ_SUITE_B_NOT_AVAILABLE"
		case 1:
			s = "MQ_SUITE_B_NONE"
		case 2:
			s = "MQ_SUITE_B_128_BIT"
		case 4:
			s = "MQ_SUITE_B_192_BIT"
		default:
			s = ""
		}
	}
	return s
}
