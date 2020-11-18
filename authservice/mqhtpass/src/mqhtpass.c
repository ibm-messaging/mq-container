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

/****************************************************************************/
/* Declare the internal functions that implement the interface              */
/****************************************************************************/
MQZ_INIT_AUTHORITY MQStart;
static MQZ_AUTHENTICATE_USER mqhtpass_authenticate_user;
static MQZ_FREE_USER mqhtpass_free_user;
static MQZ_TERM_AUTHORITY mqhtpass_term_auth;

#define LOG_FILE "/var/mqm/errors/mqhtpass.log"
#define HTPASSWD_FILE "/etc/mqm/mq.htpasswd"

static char *trim(char *s);

/**
 * Initialization and entrypoint for the dynamically loaded
 * authorization installable service. It registers the addresses of the
 * other functions which are to be called by the queue manager.
 *
 * This function is called whenever the module is loaded.  The Options
 * field will show whether it's a PRIMARY (i.e. during qmgr startup) or
 * SECONDARY (any other time - normally during the start of an agent
 * process which is not necessarily the same as during MQCONN, especially
 * when running multi-threaded agents) initialization, but there's
 * nothing different that we'd want to do here based on that flag.
 *
 * Because of when the init function is called, there is no need to
 * worry about multi-threaded stuff in this particular function.
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
  
  log_rc = log_init(LOG_FILE);
  if (log_rc != 0)
  {
    CC = MQCC_FAILED;
    Reason = MQRC_INITIALIZATION_FAILED;
  }

  log_infof("MQStart options=%s qmgr=%s", ((Options == MQZIO_SECONDARY) ? "Secondary" : "Primary"), trim(QMgrName));
  /************************************************************************/
  /* Initialize the entry point vectors.  This is performed for both      */
  /* global and process initialisation, i.e whatever the value of the     */
  /* Options field.                                                       */
  /************************************************************************/
  if (CC == MQCC_OK)
    hc->MQZEP_Call(hc, MQZID_INIT_AUTHORITY, (PMQFUNC)MQStart, &CC, &Reason);

  if (CC == MQCC_OK)
    hc->MQZEP_Call(hc, MQZID_TERM_AUTHORITY, (PMQFUNC)mqhtpass_term_auth, &CC, &Reason);

  if (CC == MQCC_OK)
    hc->MQZEP_Call(hc, MQZID_AUTHENTICATE_USER, (PMQFUNC)mqhtpass_authenticate_user, &CC, &Reason);

  if (CC == MQCC_OK)
    hc->MQZEP_Call(hc, MQZID_FREE_USER, (PMQFUNC)mqhtpass_free_user, &CC, &Reason);

  *Version = MQZAS_VERSION_5;
  *pCompCode = CC;
  *pReason = Reason;
  return;
}

/**
 * Called during the connection of any application. This allows the OAM
 * to change the userid associated with the connection, regardless of the
 * operating system user ID. One reason you might want to do that is to
 * deal with non-standard user IDs, which perhaps are longer than 12
 * characters. The CorrelationPtr can be assigned in this function to
 * point to some OAM-managed storage, and is available as part of the
 * MQZED structure for all subsequent functions. Note that there is only
 * one CorrelPtr stored for the user's hconn, so if two OAMs are chained
 * and both want to manage storage for the connection, there would be
 * difficulties as there is no reverse call that would allow the second
 * to reset the first's pointer (or vice versa). I'd suggest instead
 * using something like thread-specific storage as each thread is tied
 * to the hconn.
 *
 * When a clntconn/svrconn channel connects to the queue manager, the
 * authentication is supposed to take two stages. First as the
 * channel program connects, and then as the MCAUSER is set. You will
 * see this as "initial" and "change" context values in the parameters.
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
      log_errorf("Unable to allocate memory");
      return;
    }
    strncpy(spuser, pSecurityParms->CSPUserIdPtr, pSecurityParms->CSPUserIdLength);
    spuser[pSecurityParms->CSPUserIdLength] = 0;
    sppass = malloc(pSecurityParms->CSPPasswordLength + 1);
    if (!sppass)
    {
      log_errorf("Unable to allocate memory");
      if (spuser)
        free(spuser);
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
          "Failed to authenticate user=%s effuser=%s applname=%s cspuser=%s cc=%d reason=%d",
          pIdentityContext->UserIdentifier,
          pApplicationContext->EffectiveUserID,
          pApplicationContext->ApplName,
          spuser,
          *pCompCode,
          *pReason);
    }
    if (spuser)
      free(spuser);
    if (sppass)
      free(sppass);
  }
  else
  {
    // Password not supplied, so just check that the user ID is valid
    spuser = malloc(sizeof(PMQCHAR12) + 1);
    if (!sppass)
    {
      log_errorf("Unable to allocate memory");
      return;
    }
    strncpy(spuser, pApplicationContext->EffectiveUserID, strlen(pApplicationContext->EffectiveUserID));
    spuser[sizeof(PMQCHAR12)] = 0;
    log_debugf("%s without CSP user set.  effectiveuid=%s", __func__, spuser);
    bool valid_user = htpass_valid_user(HTPASSWD_FILE, spuser);
    if (valid_user)
    {
      *pCompCode = MQCC_OK;
      *pReason = MQRC_NONE;
      *pContinuation = MQZCI_CONTINUE;
      memcpy(pIdentityContext->UserIdentifier, spuser, sizeof(pIdentityContext->UserIdentifier));
    }
    else
    {
      log_debugf(
          "Invalid user=%s effuser=%s applname=%s cspuser=%s cc=%d reason=%d",
          pIdentityContext->UserIdentifier,
          pApplicationContext->EffectiveUserID,
          pApplicationContext->ApplName,
          spuser,
          *pCompCode,
          *pReason);
    }
    if (spuser)
      free(spuser);
  }
  return;
}

/**
 * Called during MQDISC, as the inverse of the Authenticate. If the authorization
 * service has allocated private storage to hold additional information about
 * the user, then this is the time to free it. No more calls will be made
 * to the authorization service for this connection instance of this user.
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
 * Called during MQDISC, as the inverse of the Authenticate. If the OAM
 * has allocated private storage to hold additional information about
 * the user, then this is the time to free it. No more calls will be made
 * to the authorization service for this connection instance of this user.
 */
static void MQENTRY mqhtpass_term_auth(
    MQHCONFIG hc,
    MQLONG Options,
    PMQCHAR pQMgrName,
    PMQBYTE pComponentData,
    PMQLONG pCompCode,
    PMQLONG pReason)
{
  log_debugf("mqhtpass_term_auth()");
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
