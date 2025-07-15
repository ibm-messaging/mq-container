/*
© Copyright IBM Corporation 2021, 2024

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

// This is a developer only configuration and not recommended for production usage.

#include <stdbool.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <cmqec.h>
#include "log.h"
#include "simpleauth.h"

// Declare the internal functions that implement the interface
MQZ_INIT_AUTHORITY MQStart;
static MQZ_AUTHENTICATE_USER mqsimpleauth_authenticate_user;
static MQZ_FREE_USER mqsimpleauth_free_user;
static MQZ_TERM_AUTHORITY mqsimpleauth_terminate;

#define LOG_FILE "/var/mqm/errors/simpleauth.json"
#define NAME "MQ Advanced for Developers custom authentication service"

/**
 * Initialization and entrypoint for the dynamically loaded
 * authorization installable service. It registers the addresses of the
 * other functions which are to be called by the queue manager.
 *
 * This function is called whenever the module is loaded.  The Options
 * field will show whether it's a PRIMARY (i.e. during qmgr startup) or
 * SECONDARY.
 */
void MQENTRY MQStart(
    MQHCONFIG hc,
    MQLONG Options,
    MQCHAR48 QMgrName,
    MQLONG ComponentDataLength,
    PMQBYTE ComponentData,
    PMQLONG Version,
    PMQLONG pCompCode,
    PMQLONG pReason)
{
  MQLONG CC = MQCC_OK;
  MQLONG Reason = MQRC_NONE;
  int log_rc = 0;

  if (Options == MQZIO_PRIMARY)
  {
    // Reset the log file.  The file could still get large if debug is turned on,
    // but this is a simpler solution for now.
    log_rc = log_init_reset(LOG_FILE);
  }
  else
  {
    log_rc = log_init(LOG_FILE);
  }

  if (log_rc != 0)
  {
    CC = MQCC_FAILED;
    Reason = MQRC_INITIALIZATION_FAILED;
  }

  if (Options == MQZIO_PRIMARY)
  {
    log_infof("Initializing %s", NAME);
  }

  log_debugf("MQStart options=%s qmgr=%.*s", ((Options == MQZIO_SECONDARY) ? "Secondary" : "Primary"), trimmed_len(QMgrName, MQ_Q_MGR_NAME_LENGTH), QMgrName);

  // Initialize the functions to use for each entry point
  if (CC == MQCC_OK)
  {
    hc->MQZEP_Call(hc, MQZID_INIT_AUTHORITY, (PMQFUNC)MQStart, &CC, &Reason);
  }
  if (CC == MQCC_OK)
  {
    hc->MQZEP_Call(hc, MQZID_TERM_AUTHORITY, (PMQFUNC)mqsimpleauth_terminate, &CC, &Reason);
  }
  if (CC == MQCC_OK)
  {
    hc->MQZEP_Call(hc, MQZID_AUTHENTICATE_USER, (PMQFUNC)mqsimpleauth_authenticate_user, &CC, &Reason);
  }
  if (CC == MQCC_OK)
  {
    hc->MQZEP_Call(hc, MQZID_FREE_USER, (PMQFUNC)mqsimpleauth_free_user, &CC, &Reason);
  }
  *Version = MQZAS_VERSION_6;
  *pCompCode = CC;
  *pReason = Reason;
  return;
}

/**
 * Called during the connection of any application which supplies an MQCSP (Connection Security Parameters).
 * This is the usual case.
 * See https://www.ibm.com/support/knowledgecenter/SSFKSJ_latest/com.ibm.mq.ref.dev.doc/q095610_.html
 */
static void MQENTRY mqsimpleauth_authenticate_user_csp(
    PMQCHAR pQMgrName,
    PMQCSP pSecurityParms,
    PMQZAC pApplicationContext,
    PMQZIC pIdentityContext,
    PMQPTR pCorrelationPtr,
    PMQBYTE pComponentData,
    PMQLONG pContinuation,
    PMQLONG pCompCode,
    PMQLONG pReason)
{
  char *csp_user = NULL;
  char *csp_pass = NULL;

  // Firstly, create null-terminated strings from the user credentials in the MQ CSP object
  csp_user = malloc(pSecurityParms->CSPUserIdLength + 1);
  if (!csp_user)
  {
    log_errorf("%s is unable to allocate memory for a user", NAME);
    *pCompCode = MQCC_FAILED;
    *pReason = MQRC_SERVICE_ERROR;
    return;
  }
  strncpy(csp_user, pSecurityParms->CSPUserIdPtr, pSecurityParms->CSPUserIdLength);
  csp_user[pSecurityParms->CSPUserIdLength] = 0;
  csp_pass = malloc((pSecurityParms->CSPPasswordLength + 1));
  if (!csp_pass)
  {
    log_errorf("%s is unable to allocate memory for a password", NAME);
    *pCompCode = MQCC_FAILED;
    *pReason = MQRC_SERVICE_ERROR;
    if (csp_user)
    {
      memset(csp_user, 0, pSecurityParms->CSPUserIdLength);
      free(csp_user);
    }
    return;
  }
  strncpy(csp_pass, pSecurityParms->CSPPasswordPtr, pSecurityParms->CSPPasswordLength);
  csp_pass[pSecurityParms->CSPPasswordLength] = 0;
  log_debugf("%s with CSP user set. user=%s", __func__, csp_user);
  int auth_result = simpleauth_authenticate_user(csp_user, csp_pass);

  if (auth_result == SIMPLEAUTH_VALID)
  {
    // An OK completion code means MQ will accept this user is authenticated
    *pCompCode = MQCC_OK;
    *pReason = MQRC_NONE;
    // Tell the queue manager to stop trying other authorization services.
    *pContinuation = MQZCI_STOP;
    memcpy(pIdentityContext->UserIdentifier, csp_user, sizeof(pIdentityContext->UserIdentifier));
    log_debugf("Authenticated user=%s", pIdentityContext->UserIdentifier);
  }
  // If the simpleauth file does not have an entry for this user
  else if (auth_result == SIMPLEAUTH_INVALID_USER)
  {
    *pCompCode = MQCC_WARNING;
    *pReason = MQRC_NONE;
    // Tell the queue manager to continue trying other authorization services, as they might have the user.
    *pContinuation = MQZCI_CONTINUE;
    log_debugf(
        "User authentication failed due to invalid user.  user=%.*s effuser=%.*s applname=%.*s csp_user=%s cc=%d reason=%d",
        trimmed_len(pIdentityContext->UserIdentifier, MQ_USER_ID_LENGTH),
        pIdentityContext->UserIdentifier,
        trimmed_len(pApplicationContext->EffectiveUserID, MQ_USER_ID_LENGTH),
        pApplicationContext->EffectiveUserID,
        trimmed_len(pApplicationContext->ApplName, MQ_APPL_NAME_LENGTH),
        pApplicationContext->ApplName,
        csp_user,
        *pCompCode,
        *pReason);
  }
  // If the simpleauth file has an entry for this user, but the password supplied is incorrect
  else if (auth_result == SIMPLEAUTH_INVALID_PASSWORD)
  {
    *pCompCode = MQCC_WARNING;
    *pReason = MQRC_NOT_AUTHORIZED;
    // Tell the queue manager to stop trying other authorization services.
    *pContinuation = MQZCI_STOP;
    log_debugf(
        "User authentication failed due to invalid password.  user=%.*s effuser=%.*s applname=%.*s csp_user=%s cc=%d reason=%d",
        trimmed_len(pIdentityContext->UserIdentifier, MQ_USER_ID_LENGTH),
        pIdentityContext->UserIdentifier,
        trimmed_len(pApplicationContext->EffectiveUserID, MQ_USER_ID_LENGTH),
        pApplicationContext->EffectiveUserID,
        trimmed_len(pApplicationContext->ApplName, MQ_APPL_NAME_LENGTH),
        pApplicationContext->ApplName,
        csp_user,
        *pCompCode,
        *pReason);
  }
  if (csp_user)
  {
    memset(csp_user, 0, pSecurityParms->CSPUserIdLength);
    free(csp_user);
  }
  if (csp_pass)
  {
     memset(csp_pass, 0, pSecurityParms->CSPPasswordLength);
    free(csp_pass);
  }
  return;
}

/**
 * Called during the connection of any application.
 * For more information on the parameters, see https://www.ibm.com/support/knowledgecenter/en/SSFKSJ_latest/com.ibm.mq.ref.dev.doc/q110090_.html
 */
static void MQENTRY mqsimpleauth_authenticate_user(
    PMQCHAR pQMgrName,
    PMQCSP pSecurityParms,
    PMQZAC pApplicationContext,
    PMQZIC pIdentityContext,
    PMQPTR pCorrelationPtr,
    PMQBYTE pComponentData,
    PMQLONG pContinuation,
    PMQLONG pCompCode,
    PMQLONG pReason)
{
  char *spuser = NULL;
  // By default, return a warning, which indicates to MQ that this
  // authorization service hasn't authenticated the user.
  *pCompCode = MQCC_WARNING;
  *pReason = MQRC_NONE;
  // By default, tell the queue manager to continue trying other
  // authorization services.
  *pContinuation = MQZCI_CONTINUE;

  if ((pSecurityParms->AuthenticationType) == MQCSP_AUTH_USER_ID_AND_PWD)
  {
    mqsimpleauth_authenticate_user_csp(pQMgrName, pSecurityParms, pApplicationContext, pIdentityContext, pCorrelationPtr, pComponentData, pContinuation, pCompCode, pReason);
  }
  else
  {
    // Password not supplied, so just check that the user ID is valid
    spuser = malloc(sizeof(PMQCHAR12) + 1);
    if (!spuser)
    {
      log_errorf("%s is unable to allocate memory to check a user", NAME);
      *pCompCode = MQCC_FAILED;
      *pReason = MQRC_SERVICE_ERROR;
      return;
    }
    strncpy(spuser, pApplicationContext->EffectiveUserID, strlen(pApplicationContext->EffectiveUserID));
    spuser[sizeof(PMQCHAR12)] = 0;
    log_debugf("%s without CSP user set.  effectiveuid=%s env=%d, callertype=%d, type=%d, accttoken=%d applidentitydata=%d", __func__, spuser, pApplicationContext->Environment, pApplicationContext->CallerType, pApplicationContext->AuthenticationType, pIdentityContext->AccountingToken, pIdentityContext->ApplIdentityData);
    if (strncmp(spuser, "mqm", 3) == 0)
    {
      // Special case: pass the "mqm" user on for validation up the chain
      // A warning in the completion code means MQ will pass this to other authorization services
      *pCompCode = MQCC_WARNING;
      *pReason = MQRC_NONE;
      *pContinuation = MQZCI_CONTINUE;
    }
    else
    {
      bool valid_user = simpleauth_valid_user(spuser);
      if (valid_user)
      {
        // An OK completion code means MQ will accept this user is authenticated
        *pCompCode = MQCC_OK;
        *pReason = MQRC_NONE;
        *pContinuation = MQZCI_STOP;
        memcpy(pIdentityContext->UserIdentifier, spuser, sizeof(pIdentityContext->UserIdentifier));
      }
      else
      {
        log_debugf(
            "User authentication failed user=%.*s effuser=%.*s applname=%.*s cspuser=%s cc=%d reason=%d",
            trimmed_len(pIdentityContext->UserIdentifier, MQ_USER_ID_LENGTH),
            pIdentityContext->UserIdentifier,
            trimmed_len(pApplicationContext->EffectiveUserID, MQ_USER_ID_LENGTH),
            pApplicationContext->EffectiveUserID,
            trimmed_len(pApplicationContext->ApplName, MQ_APPL_NAME_LENGTH),
            pApplicationContext->ApplName,
            spuser,
            *pCompCode,
            *pReason);
      }
      if (spuser)
      {
         memset(spuser, 0, sizeof(PMQCHAR12) + 1);
        free(spuser);
      }
    }
  }
  return;
}

/**
 * Called during MQDISC, as the inverse of the call to authenticate.
 */
static void MQENTRY mqsimpleauth_free_user(
    PMQCHAR pQMgrName,
    PMQZFP pFreeParms,
    PMQBYTE pComponentData,
    PMQLONG pContinuation,

    PMQLONG pCompCode,
    PMQLONG pReason)
{
  log_debugf("mqsimpleauth_freeuser()");
  *pCompCode = MQCC_WARNING;
  *pReason = MQRC_NONE;
  *pContinuation = MQZCI_CONTINUE;
}

/**
 * Called when the authorization service is terminated.
 */
static void MQENTRY mqsimpleauth_terminate(
    MQHCONFIG hc,
    MQLONG Options,
    PMQCHAR pQMgrName,
    PMQBYTE pComponentData,
    PMQLONG pCompCode,
    PMQLONG pReason)
{
  if (Options == MQZTO_PRIMARY)
  {
    log_infof("Terminating %s", NAME);
    log_close();
  }
  else {
    log_debugf("Terminating secondary");
  }
  *pCompCode = MQCC_OK;
  *pReason = MQRC_NONE;
}

