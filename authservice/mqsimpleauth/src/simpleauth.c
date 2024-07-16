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

#include <errno.h>
#include <stdbool.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include "log.h"
#include "simpleauth.h"
#include <linux/limits.h>

const char *_mq_app_secret_file = MQ_APP_SECRET_FILE_DEFAULT;
const char *_mq_admin_secret_file = MQ_ADMIN_SECRET_FILE_DEFAULT;

// Check if the user is valid
int simpleauth_authenticate_user(const char *const user, const char *const password)
{
  int result = -1;

  if (simpleauth_valid_user(user))
  {
    char *pwd = get_secret_for_user(user);
    if (pwd != NULL)
    {
      if (strcmp(pwd, password) == 0)
      {
        log_debugf("Correct password supplied. user=%s", user);
        result = SIMPLEAUTH_VALID;
      }
      else
      {
        log_debugf("Incorrect password supplied. user=%s", user);
        result = SIMPLEAUTH_INVALID_PASSWORD;
      }
      memset(pwd, 0, strlen(pwd));
      free(pwd);
    }
    else
    {
      log_debugf("Failed to get secret for user '%s'", user);
      result = SIMPLEAUTH_INVALID_PASSWORD;
    }
  }
  else
  {
    log_debugf("User does not exist. user=%s", user);
    result = SIMPLEAUTH_INVALID_USER;
  }
  return result;
}

bool simpleauth_valid_user(const char *const user)
{
  bool valid = false;
  if ((strcmp(user, APP_USER_NAME) == 0 || strcmp(user, ADMIN_USER_NAME) == 0))
  {
    valid = true;
  }
  return valid;
}

/**
 * get_secret_for_user will return a char* containing the credential for the given user
 * the credential is read from the filesystem if the relevant file exists and an environment
 * variable if not
 *
 * The caller is responsible for clearing then freeing memory
 */
char *get_secret_for_user(const char *const user)
{
  if (0 == strcmp(user, APP_USER_NAME))
  {
    char *secret = read_secret(_mq_app_secret_file);
    if (secret != NULL)
    {
      return secret;
    }
    else
    {
      const char *pwdFromEnv = getenv("MQ_APP_PASSWORD");
      if (pwdFromEnv != NULL)
      {
        log_infof("Environment variable MQ_APP_PASSWORD is deprecated, use secrets to set the passwords");
      }
      return strdup(pwdFromEnv);
    }
  }
  else if (0 == strcmp(user, ADMIN_USER_NAME))
  {
    char *secret = read_secret(_mq_admin_secret_file);
    if (secret != NULL)
    {
      return secret;
    }
    else
    {
      const char *pwdFromEnv = getenv("MQ_ADMIN_PASSWORD");
      if (pwdFromEnv != NULL)
      {
        log_infof("Environment variable MQ_ADMIN_PASSWORD is deprecated, use secrets to set the passwords");
      }
      return strdup(pwdFromEnv);
    }
  }
  else
  {
    return NULL;
  }
}

/**
 * read_secret will return a char* containing the credential read from the filesystem for the given user
 *
 * The caller is responsible for clearing then freeing memory
 */
char *read_secret(const char *const secret)
{
  FILE *fp = fopen(secret, "r");
  if (fp)
  {
    const int line_size = MAX_PASSWORD_LENGTH + 1;
    char *pwd = malloc(line_size);
    char *result;
    result = fgets(pwd, line_size, fp);
    fclose(fp);
    if (result == NULL)
    {
      memset(pwd, 0, line_size);
      free(pwd);
      return NULL;
    }
    result[strcspn(result, "\r\n")] = 0;
    return result;
  }
  else
  {
    return NULL;
  }
}
