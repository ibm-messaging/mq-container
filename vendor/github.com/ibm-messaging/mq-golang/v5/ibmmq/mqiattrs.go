package ibmmq

/*
  Copyright (c) IBM Corporation 2018, 2023

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

/* We have to cope with character attributes that might not be
   available in older versions of MQ. Maps cannot be passed to C
   functions so we use some arrays with ifdefs to control it and then
   add them to the Go length map.
*/

/*

#include <stdlib.h>
#include <string.h>
#include <cmqc.h>

#define MAX_NEW_MQCA_ATTRS 32 // Running amqsinq.go will check this is large enough

int addNewCharAttrs(MQLONG a[],MQLONG l[]) {
  int i = 0;
#if defined(MQCA_SSL_KEY_REPO_PASSWORD)
  a[i] = MQCA_SSL_KEY_REPO_PASSWORD;
  l[i] = MQ_SSL_ENCRYP_KEY_REPO_PWD_LEN;
  if (++i > MAX_NEW_MQCA_ATTRS) return -1;
#endif

#if defined(MQCA_INITIAL_KEY)
  a[i] = MQCA_INITIAL_KEY;
  l[i] = MQ_INITIAL_KEY_LENGTH;
  if (++i > MAX_NEW_MQCA_ATTRS) return -1;
#endif

#if defined(MQCA_STREAM_QUEUE_NAME)
  a[i] = MQCA_STREAM_QUEUE_NAME;
  l[i] = MQ_Q_NAME_LENGTH;
  if (++i > MAX_NEW_MQCA_ATTRS) return -1;
#endif

  return i;
}

*/
import "C"
import (
	"os"
	"sync"
)

/*
 * This file deals with the lengths of attributes that may be processed
 * by the MQSET/MQINQ calls. Only a small set of the object attributes
 * are supported by MQINQ (and even fewer for MQSET) so it's reasonable
 * to list them all here
 */
var mqInqLength = map[int32]int32{
	C.MQCA_ALTERATION_DATE:       C.MQ_DATE_LENGTH,
	C.MQCA_ALTERATION_TIME:       C.MQ_TIME_LENGTH,
	C.MQCA_APPL_ID:               C.MQ_PROCESS_APPL_ID_LENGTH,
	C.MQCA_BACKOUT_REQ_Q_NAME:    C.MQ_Q_NAME_LENGTH,
	C.MQCA_BASE_Q_NAME:           C.MQ_Q_NAME_LENGTH,
	C.MQCA_CERT_LABEL:            C.MQ_CERT_LABEL_LENGTH,
	C.MQCA_QSG_CERT_LABEL:        C.MQ_CERT_LABEL_LENGTH,
	C.MQCA_CF_STRUC_NAME:         C.MQ_CF_STRUC_NAME_LENGTH,
	C.MQCA_CHANNEL_AUTO_DEF_EXIT: C.MQ_EXIT_NAME_LENGTH,
	C.MQCA_CHINIT_SERVICE_PARM:   C.MQ_CHINIT_SERVICE_PARM_LENGTH,
	C.MQCA_CLUSTER_NAME:          C.MQ_CLUSTER_NAME_LENGTH,
	C.MQCA_CLUSTER_NAMELIST:      C.MQ_NAMELIST_NAME_LENGTH,
	C.MQCA_CLUSTER_WORKLOAD_DATA: C.MQ_EXIT_DATA_LENGTH,
	C.MQCA_CLUSTER_WORKLOAD_EXIT: C.MQ_EXIT_NAME_LENGTH,
	C.MQCA_CLUS_CHL_NAME:         C.MQ_OBJECT_NAME_LENGTH,
	C.MQCA_COMMAND_INPUT_Q_NAME:  C.MQ_Q_NAME_LENGTH,
	C.MQCA_COMM_INFO_NAME:        C.MQ_OBJECT_NAME_LENGTH,
	C.MQCA_CONN_AUTH:             C.MQ_AUTH_INFO_NAME_LENGTH,
	C.MQCA_CREATION_DATE:         C.MQ_DATE_LENGTH,
	C.MQCA_CREATION_TIME:         C.MQ_TIME_LENGTH,
	C.MQCA_CUSTOM:                C.MQ_CUSTOM_LENGTH,
	C.MQCA_DEAD_LETTER_Q_NAME:    C.MQ_Q_NAME_LENGTH,
	C.MQCA_DEF_XMIT_Q_NAME:       C.MQ_Q_NAME_LENGTH,
	C.MQCA_DNS_GROUP:             C.MQ_DNS_GROUP_NAME_LENGTH,
	C.MQCA_ENV_DATA:              C.MQ_PROCESS_ENV_DATA_LENGTH,
	C.MQCA_IGQ_USER_ID:           C.MQ_USER_ID_LENGTH,
	C.MQCA_INITIATION_Q_NAME:     C.MQ_Q_NAME_LENGTH,
	C.MQCA_INSTALLATION_DESC:     C.MQ_INSTALLATION_DESC_LENGTH,
	C.MQCA_INSTALLATION_NAME:     C.MQ_INSTALLATION_NAME_LENGTH,
	C.MQCA_INSTALLATION_PATH:     C.MQ_INSTALLATION_PATH_LENGTH,
	C.MQCA_LU62_ARM_SUFFIX:       C.MQ_ARM_SUFFIX_LENGTH,
	C.MQCA_LU_GROUP_NAME:         C.MQ_LU_NAME_LENGTH,
	C.MQCA_LU_NAME:               C.MQ_LU_NAME_LENGTH,
	C.MQCA_NAMELIST_DESC:         C.MQ_NAMELIST_DESC_LENGTH,
	C.MQCA_NAMELIST_NAME:         C.MQ_NAMELIST_NAME_LENGTH,
	C.MQCA_NAMES:                 C.MQ_OBJECT_NAME_LENGTH * 256, // Maximum length to allocate
	C.MQCA_PARENT:                C.MQ_Q_MGR_NAME_LENGTH,
	C.MQCA_PROCESS_DESC:          C.MQ_PROCESS_DESC_LENGTH,
	C.MQCA_PROCESS_NAME:          C.MQ_PROCESS_NAME_LENGTH,
	C.MQCA_Q_DESC:                C.MQ_Q_DESC_LENGTH,
	C.MQCA_Q_MGR_DESC:            C.MQ_Q_MGR_DESC_LENGTH,
	C.MQCA_Q_MGR_IDENTIFIER:      C.MQ_Q_MGR_IDENTIFIER_LENGTH,
	C.MQCA_Q_MGR_NAME:            C.MQ_Q_MGR_NAME_LENGTH,
	C.MQCA_Q_NAME:                C.MQ_Q_NAME_LENGTH,
	C.MQCA_QSG_NAME:              C.MQ_QSG_NAME_LENGTH,
	C.MQCA_REMOTE_Q_MGR_NAME:     C.MQ_Q_MGR_NAME_LENGTH,
	C.MQCA_REMOTE_Q_NAME:         C.MQ_Q_NAME_LENGTH,
	C.MQCA_REPOSITORY_NAME:       C.MQ_Q_MGR_NAME_LENGTH,
	C.MQCA_REPOSITORY_NAMELIST:   C.MQ_NAMELIST_NAME_LENGTH,
	C.MQCA_SSL_CRL_NAMELIST:      C.MQ_NAMELIST_NAME_LENGTH,
	C.MQCA_SSL_CRYPTO_HARDWARE:   C.MQ_SSL_CRYPTO_HARDWARE_LENGTH,
	C.MQCA_SSL_KEY_REPOSITORY:    C.MQ_SSL_KEY_REPOSITORY_LENGTH,
	C.MQCA_STORAGE_CLASS:         C.MQ_STORAGE_CLASS_LENGTH,
	C.MQCA_TCP_NAME:              C.MQ_TCP_NAME_LENGTH,
	C.MQCA_TRIGGER_DATA:          C.MQ_TRIGGER_DATA_LENGTH,
	C.MQCA_USER_DATA:             C.MQ_PROCESS_USER_DATA_LENGTH,
	C.MQCA_VERSION:               C.MQ_VERSION_LENGTH,
	C.MQCA_XMIT_Q_NAME:           C.MQ_Q_NAME_LENGTH,
}

var charAttrsAddedOnce sync.Once

/*
 * Return how many char & int attributes are in the list of selectors, and the
 * maximum length of the buffer needed to return them from the MQI
 */
func getAttrInfo(attrs []int32) (int, int, int) {
	var charAttrLength = 0
	var charAttrCount = 0
	var intAttrCount = 0

	charAttrsAddedOnce.Do(func() {
		maxNewAttrs := C.MAX_NEW_MQCA_ATTRS
		attrVals := make([]C.MQLONG, maxNewAttrs)
		attrLens := make([]C.MQLONG, maxNewAttrs)
		addedAttrs := int(C.addNewCharAttrs(&attrVals[0], &attrLens[0]))

		if addedAttrs < 0 {
			// Force an immediate exit if the arrays are not large enough.
			logError("mqiattrs.go: MAX_NEW_MQCA_ATTRS is too small. Increase from %d\n", C.MAX_NEW_MQCA_ATTRS)
			os.Exit(1)
		}
		for i := 0; i < addedAttrs; i++ {
			mqInqLength[int32(attrVals[i])] = int32(attrLens[i])
		}
	})

	for i := 0; i < len(attrs); i++ {
		if v, ok := mqInqLength[attrs[i]]; ok {
			charAttrCount++
			charAttrLength += int(v)
		} else if attrs[i] >= C.MQIA_FIRST && attrs[i] <= C.MQIA_LAST {
			intAttrCount++
		}
	}
	return intAttrCount, charAttrCount, charAttrLength
}

func getAttrLength(attr int32) int {
	if v, ok := mqInqLength[attr]; ok {
		return int(v)
	} else {
		return 0
	}

}
