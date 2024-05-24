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

// Check if the user is valid
int simpleauth_authenticate_user(char *user, char *password)
{
  int result = -1;

  if (simpleauth_valid_user(user))
  {
    char *pwd = getSecretForUser(user);
    if(pwd != NULL)
    {
      int pwdCheck = strncmp(pwd, password, strlen(password));
      if (pwdCheck == 0)
      {
        log_debugf("Correct password supplied. user=%s", user);
        result = SIMPLEAUTH_VALID;
      }
      else
      {
        log_debugf("Incorrect password supplied. user=%s", user);
        result = SIMPLEAUTH_INVALID_PASSWORD;
      }
      pwd = NULL;
    }
    else
    {
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

bool simpleauth_valid_user(char *user)
{
  bool valid = false;
  if ((strcmp(user, APP_USER_NAME)==0 || strcmp(user, ADMIN_USER_NAME)==0))
  {
    valid = true;
  }
  return valid;
}

char *getSecretForUser(char *user)
{
  if (0 == strcmp(user, APP_USER_NAME))
  {
    char *secret = readSecret(MQ_APP_SECRET_FILE);
    if (secret != NULL)
    {
      return secret;
    }
    else
    {
      char* pwdFromEnv = getenv("MQ_APP_PASSWORD");
      if (pwdFromEnv != NULL)
      {
        log_infof("Environment variable MQ_APP_PASSWORD is deprecated, use secrets to set the passwords");
      }
      return pwdFromEnv;
    }
  } else if (0 == strcmp(user, ADMIN_USER_NAME))
  {
      char *secret = readSecret(MQ_ADMIN_SECRET_FILE);
      if (secret != NULL)
      {
        return secret;
      }
      else
      {
        char* pwdFromEnv =  getenv("MQ_ADMIN_PASSWORD");
        if (pwdFromEnv != NULL)
        {
          log_infof("Environment variable MQ_ADMIN_PASSWORD is deprecated, use secrets to set the passwords");
        }
        return pwdFromEnv;
      }
  }
  else
  {
    return NULL;
  }
}

char *readSecret(char* secret)
{
  FILE *fp = fopen(secret, "r");
  const size_t line_size = 1024;
  if (fp)
  {
    char *pwd = malloc(line_size);
    fgets(pwd, line_size, fp);
    fclose(fp);
    return pwd;
  }
  else
  {
    return NULL;
  }
}
