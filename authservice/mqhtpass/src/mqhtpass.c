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

// This is a developer only configuration and not recommended for production usage.

#include <stdbool.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <cmqec.h>
#include "log.h"
#include "htpass.h"

// Declare the internal functions that implement the interface
MQZ_INIT_AUTHORITY MQStart;
static MQZ_AUTHENTICATE_USER mqhtpass_authenticate_user;
static MQZ_FREE_USER mqhtpass_free_user;
static MQZ_TERM_AUTHORITY mqhtpass_terminate;

#define LOG_FILE "/var/mqm/errors/mqhtpass.json"
#define HTPASSWD_FILE "/etc/mqm/mq.htpasswd"
#define NAME "MQ Advanced for Developers custom authentication service"

static char *trim(char *s);

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
  log_debugf("MQStart options=%s qmgr=%s", ((Options == MQZIO_SECONDARY) ? "Secondary" : "Primary"), trim(QMgrName));

  if (!htpass_valid_file(HTPASSWD_FILE))
  {
    CC = MQCC_FAILED;
    Reason = MQRC_INITIALIZATION_FAILED;
  }

  // Initialize the functions to use for each entry point
  if (CC == MQCC_OK)
  {
    hc->MQZEP_Call(hc, MQZID_INIT_AUTHORITY, (PMQFUNC)MQStart, &CC, &Reason);
  }
  if (CC == MQCC_OK)
  {
    hc->MQZEP_Call(hc, MQZID_TERM_AUTHORITY, (PMQFUNC)mqhtpass_terminate, &CC, &Reason);
  }
  if (CC == MQCC_OK)
  {
    hc->MQZEP_Call(hc, MQZID_AUTHENTICATE_USER, (PMQFUNC)mqhtpass_authenticate_user, &CC, &Reason);
  }
  if (CC == MQCC_OK)
  {
    hc->MQZEP_Call(hc, MQZID_FREE_USER, (PMQFUNC)mqhtpass_free_user, &CC, &Reason);
  }
  *Version = MQZAS_VERSION_5;
  *pCompCode = CC;
  *pReason = Reason;
  return;
}

/**
 * Called during the connection of any application.
 */
static void MQENTRY mqhtpass_authenticate_user(
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
  char *sppass = NULL;
  // By default, return a warning, which indicates to MQ that this
  // authorization service hasn't authenticated the user.
  *pCompCode = MQCC_WARNING;
  *pReason = MQRC_NONE;
  // By default, tell the queue manager to continue trying other
  // authorization services.
  *pContinuation = MQZCI_CONTINUE;

  if ((pSecurityParms->AuthenticationType) == MQCSP_AUTH_USER_ID_AND_PWD)
  {
    // Authenticating a user ID and password.

    // Firstly, create null-terminated strings from the user credentials in the MQ CSP object
    spuser = malloc(pSecurityParms->CSPUserIdLength + 1);
    if (!spuser)
    {
      log_errorf("%s is unable to allocate memory for a user", NAME);
      return;
    }
    strncpy(spuser, pSecurityParms->CSPUserIdPtr, pSecurityParms->CSPUserIdLength);
    spuser[pSecurityParms->CSPUserIdLength] = 0;
    sppass = malloc((pSecurityParms->CSPPasswordLength + 1));
    if (!sppass)
    {
      log_errorf("%s is unable to allocate memory for a password", NAME);
      if (spuser)
      {
        free(spuser);
      }
      return;
    }
    strncpy(sppass, pSecurityParms->CSPPasswordPtr, pSecurityParms->CSPPasswordLength);
    sppass[pSecurityParms->CSPPasswordLength] = 0;
    log_debugf("%s with CSP user set. user=%s", __func__, spuser);
    bool authenticated = htpass_authenticate_user(HTPASSWD_FILE, spuser, sppass);

    if (authenticated)
    {
      *pCompCode = MQCC_OK;
      *pReason = MQRC_NONE;
      *pContinuation = MQZCI_CONTINUE;
      memcpy(pIdentityContext->UserIdentifier, spuser, sizeof(pIdentityContext->UserIdentifier));
      log_debugf("Authenticated user=%s", pIdentityContext->UserIdentifier);
    }
    else
    {
      log_debugf(
          "User authentication failed user=%s effuser=%s applname=%s cspuser=%s cc=%d reason=%d",
          trim(pIdentityContext->UserIdentifier),
          trim(pApplicationContext->EffectiveUserID),
          trim(pApplicationContext->ApplName),
          trim(spuser),
          *pCompCode,
          *pReason);
    }
    if (spuser)
    {
      free(spuser);
    }
    if (sppass)
    {
      free(sppass);
    }
  }
  else
  {
    // Password not supplied, so just check that the user ID is valid
    spuser = malloc(sizeof(PMQCHAR12) + 1);
    if (!spuser)
    {
      log_errorf("%s is unable to allocate memory to check a user", NAME);
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
      bool valid_user = htpass_valid_user(HTPASSWD_FILE, spuser);
      if (valid_user)
      {
        // An OK completion code means MQ will accept this user is authenticated
        *pCompCode = MQCC_OK;
        *pReason = MQRC_NONE;
        *pContinuation = MQZCI_CONTINUE;
        memcpy(pIdentityContext->UserIdentifier, spuser, sizeof(pIdentityContext->UserIdentifier));
      }
      else
      {
        log_debugf(
            "User authentication failed user=%s effuser=%s applname=%s cspuser=%s cc=%d reason=%d",
            trim(pIdentityContext->UserIdentifier),
            trim(pApplicationContext->EffectiveUserID),
            trim(pApplicationContext->ApplName),
            trim(spuser),
            *pCompCode,
            *pReason);
      }
      if (spuser)
      {
        free(spuser);
      }
    }
  }
  return;
}

/**
 * Called during MQDISC, as the inverse of the call to authenticate.
 */
static void MQENTRY mqhtpass_free_user(
    PMQCHAR pQMgrName,
    PMQZFP pFreeParms,
    PMQBYTE pComponentData,
    PMQLONG pContinuation,

    PMQLONG pCompCode,
    PMQLONG pReason)
{
  log_debugf("mqhtpass_freeuser()");
  *pCompCode = MQCC_WARNING;
  *pReason = MQRC_NONE;
  *pContinuation = MQZCI_CONTINUE;
}

/**
 * Called when the authorization service is terminated.
 */
static void MQENTRY mqhtpass_terminate(
    MQHCONFIG hc,
    MQLONG Options,
    PMQCHAR pQMgrName,
    PMQBYTE pComponentData,
    PMQLONG pCompCode,
    PMQLONG pReason)
{
  log_infof("Terminating %s", NAME);
  if (Options == MQZTO_PRIMARY)
  {
    log_close();
  }
  *pCompCode = MQCC_OK;
  *pReason = MQRC_NONE;
}

/**
 * Remove trailing spaces from a string.
 */
static char *trim(char *s)
{
  int i;
  for (i = strlen(s) - 1; i >= 0; i--)
  {
    if (s[i] == ' ')
      s[i] = 0;
    else
      break;
  }
  return s;
}
