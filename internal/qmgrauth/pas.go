/*
© Copyright IBM Corporation 2020

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

//This is a developer only configuration and not recommended for production usage.
package main

/*
#cgo !windows CFLAGS: -I/opt/mqm/lib64 -D_REENTRANT
#cgo !windows,!darwin LDFLAGS: -L/opt/mqm/lib64 -lmqm_r -Wl,-rpath,/opt/mqm/lib64 -Wl,-rpath,/usr/lib64
#cgo darwin   LDFLAGS:         -L/opt/mqm/lib64 -lmqm_r -Wl,-rpath,/opt/mqm/lib64 -Wl,-rpath,/usr/lib64
#cgo  windows CFLAGS:  -I"C:/Program Files/IBM/MQ/Tools/c/include"
#cgo windows LDFLAGS: -L "C:/Program Files/IBM/MQ/bin64" -lmqm
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <cmqc.h>
#include <cmqxc.h>
#include <cmqzc.h>
#include <cmqec.h>
#include <time.h>
static MQZ_INIT_AUTHORITY           PASStart;
static MQZ_AUTHENTICATE_USER        OAAuthUser;
static MQZ_FREE_USER                OAFreeUser;
static MQZ_TERM_AUTHORITY           OATermAuth;
extern int Authenticate(char *, char *);
extern int CheckAuthority(char *);
static char *OAEnvStr(MQLONG);
static void FindSize();
static void PrintDateTime();
static FILE *fp = NULL;
static int primary_process = 0;

static void MQENTRY PASStart(
  MQHCONFIG hc,
  MQLONG    Options,
  MQCHAR48   QMgrName,
  MQLONG    ComponentDataLength,
  PMQBYTE   ComponentData,
  PMQLONG   Version,
  PMQLONG   pCompCode,
  PMQLONG   pReason) {
  MQLONG CC       = MQCC_OK;
  MQLONG Reason   = MQRC_NONE;

  if ((Options & MQZIO_PRIMARY) == MQZIO_PRIMARY)
    primary_process = 1;

  fp=fopen("/var/mqm/errors/amqpasdev.log","a");

  if (CC == MQCC_OK)
    hc->MQZEP_Call(hc, MQZID_INIT_AUTHORITY,(PMQFUNC)PASStart,&CC,&Reason);

  if (CC == MQCC_OK)
    hc->MQZEP_Call(hc,MQZID_TERM_AUTHORITY,(PMQFUNC)OATermAuth,&CC,&Reason);

  if (CC == MQCC_OK)
    hc->MQZEP_Call(hc,MQZID_AUTHENTICATE_USER,(PMQFUNC)OAAuthUser,&CC,&Reason);

  if (CC == MQCC_OK)
    hc->MQZEP_Call(hc,MQZID_FREE_USER,(PMQFUNC)OAFreeUser,&CC,&Reason);

  *Version   = MQZAS_VERSION_5;
  *pCompCode = CC;
  *pReason   = Reason;

  PrintDateTime();
  fprintf(fp, "Pluggable OAM Initialized.\n");
  fprintf(fp, "THIS IS A DEVELOPER ONLY CONFIGURATION AND NOT RECOMMENDED FOR PRODUCTION USAGE");
  return;
}

static char *authuserfmt =
  "\tUser    : \"%12.12s\"\n"\
  "\tEffUser : \"%12.12s\"\n"\
  "\tAppName : \"%28.28s\"\n"\
  "\tApIdDt  : \"%32.32s\"\n"\
  "\tEnv     : \"%s\"\n"\
  "\tApp Pid : %d\n"\
  "\tApp Tid : %d\n"\
  ;
static void MQENTRY OAAuthUser (
     PMQCHAR  pQMgrName,
     PMQCSP   pSecurityParms,
     PMQZAC   pApplicationContext,
     PMQZIC   pIdentityContext,
     PMQPTR   pCorrelationPtr,
     PMQBYTE  pComponentData,
     PMQLONG  pContinuation,
     PMQLONG  pCompCode,
     PMQLONG  pReason)
{
  char *spuser = NULL;
  char *sppass = NULL;
  int gorc = MQRC_NOT_AUTHORIZED;

  if ((pSecurityParms->CSPUserIdLength) > 0) {
    //Grab the user creds from csp.
    spuser = malloc(pSecurityParms->CSPUserIdLength+1);
    strncpy(spuser,pSecurityParms->CSPUserIdPtr,pSecurityParms->CSPUserIdLength);
    spuser[pSecurityParms->CSPUserIdLength]=0;
    sppass =  malloc(pSecurityParms->CSPPasswordLength+1);
    strncpy(sppass,pSecurityParms->CSPPasswordPtr,pSecurityParms->CSPPasswordLength);
    sppass[pSecurityParms->CSPPasswordLength]=0;
    gorc = Authenticate(spuser,sppass);

    if (gorc == MQRC_NONE) {
      *pCompCode = MQCC_OK;
      *pReason = MQRC_NONE;
      *pContinuation = MQZCI_CONTINUE;
      memcpy( pIdentityContext->UserIdentifier
        , spuser
        , sizeof(pIdentityContext->UserIdentifier) );
    } else {
      *pCompCode = MQCC_WARNING;
      *pReason = MQRC_NONE;
      *pContinuation = MQZCI_CONTINUE;
      //we print to error file only if error'd
      PrintDateTime();
      if (fp) {
        fprintf(fp, authuserfmt,
          pIdentityContext->UserIdentifier,
          pApplicationContext->EffectiveUserID,
          pApplicationContext->ApplName,
          pIdentityContext->ApplIdentityData,
          OAEnvStr(pApplicationContext->Environment),
          pApplicationContext->ProcessId,
          pApplicationContext->ThreadId);

          fprintf(fp,"\tCSP UserId  : %s\n", spuser);
          fprintf(fp,"\tCSP Password : %s\n", "****..");
          fprintf(fp,"\tPAS-Compcode:%d\n",*pCompCode);
          fprintf(fp,"\tPAS-Reasoncode:%d\n",*pReason);
      }
    }
    free(spuser);
    free(sppass);
  } else {
    //this is only a normal UID authentication.
    spuser = malloc(sizeof(PMQCHAR12));
    strncpy(spuser,pApplicationContext->EffectiveUserID,strlen(pApplicationContext->EffectiveUserID));
    spuser[sizeof(PMQCHAR12)]=0;
    gorc = CheckAuthority(spuser);
    if (gorc == MQRC_NONE){
        *pCompCode = MQCC_OK;
        *pReason = MQRC_NONE;
        *pContinuation = MQZCI_CONTINUE;
        memcpy( pIdentityContext->UserIdentifier
                , spuser
                , sizeof(pIdentityContext->UserIdentifier) );
    } else {
        *pCompCode = MQCC_WARNING;
        *pReason = MQRC_NONE;
        *pContinuation = MQZCI_CONTINUE;
        //we print only if error'd
        PrintDateTime();
        if (fp)
        {
          fprintf(fp, authuserfmt,
	        pIdentityContext->UserIdentifier,
	        pApplicationContext->EffectiveUserID,
	        pApplicationContext->ApplName,
	        pIdentityContext->ApplIdentityData,
	        OAEnvStr(pApplicationContext->Environment),
	        pApplicationContext->ProcessId,
	        pApplicationContext->ThreadId
	        );
          fprintf(fp,"\tUID  : %s\n", spuser);
          fprintf(fp,"\tPAS-Compcode:%d\n",*pCompCode);
          fprintf(fp,"\tPAS-Reasoncode:%d\n",*pReason);
        }
    }
  }
  return;
}

static void MQENTRY OAFreeUser (
     PMQCHAR  pQMgrName,
     PMQZFP   pFreeParms,
     PMQBYTE  pComponentData,
     PMQLONG  pContinuation,

     PMQLONG  pCompCode,
     PMQLONG  pReason)
{
  *pCompCode = MQCC_WARNING;
  *pReason   = MQRC_NONE;
  *pContinuation = MQZCI_CONTINUE;
  return;
}

static void MQENTRY OATermAuth(
  MQHCONFIG  hc,
  MQLONG     Options,
  PMQCHAR    pQMgrName,
  PMQBYTE    pComponentData,
  PMQLONG    pCompCode,
  PMQLONG    pReason)
{
  if ((primary_process) && ((Options & MQZTO_PRIMARY) == MQZTO_PRIMARY) ||
	                   ((Options & MQZTO_SECONDARY) == MQZTO_SECONDARY))
  {
    if (fp)
    {
      fclose(fp);
      fp = NULL;
    }
  }
  *pCompCode = MQCC_OK;
  *pReason   = MQRC_NONE;
}

static void PrintDateTime() {
  FindSize();
  struct tm *local;
    time_t t;
    t = time(NULL);
    local = localtime(&t);
    if (fp) {
      fprintf(fp, "-------------------------------------------------\n");
      fprintf(fp, "Local time: %s", asctime(local));
      local = gmtime(&t);
      fprintf(fp, "UTC time: %s", asctime(local));
    }
    return;
}

static char *OAEnvStr(MQLONG x)
 {
   switch (x)
   {
   case MQXE_OTHER:          return "Application";
   case MQXE_MCA:            return "Channel";
   case MQXE_MCA_SVRCONN:    return "Channel SvrConn";
   case MQXE_COMMAND_SERVER: return "Command Server";
   case MQXE_MQSC:           return "MQSC";
   default:                  return "Invalid Environment";
   }
 }

static void FindSize()
{
  int sz = 0;
  int prev=ftell(fp);
  fseek(fp, 0L, SEEK_END);
  sz=ftell(fp);
  //if log file size goes over 1mb, rewind it.
  if (sz > 1000000) {
    rewind(fp);
  } else {
    fseek(fp, prev, SEEK_SET);
  }
}

*/
import "C"
import "github.com/ibm-messaging/mq-container/internal/htpasswd"

//export MQStart
func MQStart(hc C.MQHCONFIG, Options C.MQLONG, QMgrName C.PMQCHAR, ComponentDataLength C.MQLONG, ComponentData C.PMQBYTE, Version C.PMQLONG, pCompCode C.PMQLONG, pReason C.PMQLONG) {
	C.PASStart(hc, Options, QMgrName, ComponentDataLength, ComponentData, Version, pCompCode, pReason)
}

//export Authenticate
func Authenticate(x *C.char, y *C.char) C.int {
	user := C.GoString(x)
	pwd := C.GoString(y)
	found, ok, err := htpasswd.AuthenticateUser(user, pwd, false)

	if !found || !ok || err != nil {
		return C.MQRC_UNKNOWN_OBJECT_NAME
	}
	return C.MQRC_NONE
}

//export CheckAuthority
func CheckAuthority(x *C.char) C.int {
	user := C.GoString(x)
	found, err := htpasswd.ValidateUser(user, false)
	if !found || err != nil {
		return C.MQRC_UNKNOWN_OBJECT_NAME
	}
	return C.MQRC_NONE

}

func main() {}
