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

#include <errno.h>
#include <stdbool.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include "log.h"
#include <linux/limits.h>
#include <apr_general.h>
#include <apr_errno.h>
#include <apr_md5.h>

bool htpass_valid_file(char *filename)
{
  bool valid = true;
  FILE *fp;
  char *huser;

  fp = fopen(filename, "r");
  if (fp == NULL)
  {
    log_errorf("Error %d opening htpasswd file '%s'", errno, filename);
  }
  if (fp)
  {
    const size_t line_size = 1024;
    char *line = malloc(line_size);
    while (fgets(line, line_size, fp) != NULL)
    {
      char *saveptr;
      // Need to use strtok_r to be safe for multiple threads
      huser = strtok_r(line, ":", &saveptr);
      if (strlen(huser) >= 12)
      {
        log_errorf("Invalid htpasswd file for use with IBM MQ.  User '%s' is longer than twelve characters", huser);
        valid = false;
        break;
      }
      else {

      }
    }
    fclose(fp);
    if (line)
    {
      free(line);
    }
  }
  return valid;
}

char *find_hash(char *filename, char *user)
{
  bool found = false;
  FILE *fp;
  char *huser;
  char *hash;

  fp = fopen(filename, "r");
  if (fp == NULL)
  {
    log_errorf("Error %d opening htpasswd file '%s'", errno, filename);
  }
  if (fp)
  {
    const size_t line_size = 1024;
    char *line = malloc(line_size);
    while (fgets(line, line_size, fp) != NULL)
    {
      char *saveptr;
      // Need to use strtok_r to be safe for multiple threads
      huser = strtok_r(line, ":", &saveptr);
      if (huser && (strcmp(user, huser) == 0))
      {
        // Make a duplicate of the string, because we'll be keeping it
        hash = strdup(strtok_r(NULL, " \r\n\t", &saveptr));
        found = true;
        break;
      }
    }
    fclose(fp);
    if (line)
    {
      free(line);
    }
  }
  if (!found)
  {
    hash = NULL;
  }
  return hash;
}

bool htpass_authenticate_user(char *filename, char *user, char *password)
{
  char *hash = find_hash(filename, user);
  bool result = false;
  if (hash == NULL)
  {
    log_debugf("User does not exist. user=%s", user);
  }
  else
  {
    // Use the Apache Portable Runtime utilities to validate the password against the hash.
    // Supports multiple hashing algorithms, but we should only be using bcrypt
    apr_status_t status = apr_password_validate(password, hash);
    // status is usually either APR_SUCCESS or APR_EMISMATCH
    if (status == APR_SUCCESS)
    {
      result = true;
      log_debugf("Correct password supplied. user=%s", user);
    }
    else
    {
      log_debugf("Incorrect password supplied. user=%s", user);
    }
  }
  return result;
}

bool htpass_valid_user(char *filename, char *user)
{
  char *hash = find_hash(filename, user);
  bool valid = false;
  if (hash != NULL)
  {
    valid = true;
  }
  return valid;
}
