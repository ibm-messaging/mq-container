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

#include <stdbool.h>
#include <stdio.h>
#include <stdlib.h>
#include "log.h"
#include "simpleauth.h"
#include "simpleauth_test.h"
#include <stdlib.h>
#include <string.h>
// Headers for multi-threaded tests
#include <pthread.h>

// Start a test and log the function name
#define test_start() printf("=== RUN: %s\n", __func__)

// Indicate test has passed
#define test_pass() printf("--- PASS: %s\n", __func__)

// Indicate test has failed
void test_fail(const char *test_name)
{
  printf("--- FAIL: %s\n", test_name);
  exit(1);
}

// ----------------------------------------------------------------------------
// Simple test to read secret
// ----------------------------------------------------------------------------

void test_read_secret_ok()
{
  test_start();
  char *pwd = read_secret("./src/mqAdminPassword");
  char *password = "fred:$2y$05$3Fp9";
  if (0 != strcmp(pwd, password))
  {
    printf("%s: pwd: '%s'; password: '%s'\n", __func__, pwd, password);
    test_fail(__func__);
  }
  
  test_pass();    
}

// ----------------------------------------------------------------------------
// Simple tests for authentication
// ----------------------------------------------------------------------------

void test_simpleauth_valid_user_app_valid()
{
  test_start();
  bool validUser = simpleauth_valid_user(APP_USER_NAME);
  printf("%s: app - %d\n", __func__, validUser);

  if (!validUser)
    test_fail(__func__);
  test_pass();
}

void test_simpleauth_valid_user_admin_valid()
{
  test_start();
  bool validUser = simpleauth_valid_user(ADMIN_USER_NAME);
  printf("%s: admin - %d\n", __func__, validUser);
  if (!validUser)
    test_fail(__func__);
  test_pass();
}

void test_simpleauth_valid_user_george_invalid()
{
  test_start();
  bool validUser = simpleauth_valid_user("george");
  printf("%s: george - %d\n", __func__, validUser);
  if (validUser)
    test_fail(__func__);
  test_pass();
}

void test_simpleauth_authenticate_user_fred_unknown()
{
  test_start();
  test_set_app_password_env("passw0rd-fred-env");
  int rc = simpleauth_authenticate_user("fred", "passw0rd-fred-env");
  printf("%s: fred - %d\n", __func__, rc);
  if (rc != SIMPLEAUTH_INVALID_USER)
    test_fail(__func__);
  test_pass();
}

void test_simpleauth_authenticate_user_app_ok()
{
  test_start();
  test_set_app_password_env("passw0rd-app-env");
  int rc = simpleauth_authenticate_user("app", "passw0rd-app-env");
  printf("%s: app - %d\n", __func__, rc);
  if (rc != SIMPLEAUTH_VALID)
    test_fail(__func__);
  test_pass();
}

void test_simpleauth_authenticate_user_admin_ok()
{
  test_start();
  test_set_admin_password_env("passw0rd-admin-env");
  int rc = simpleauth_authenticate_user("admin", "passw0rd-admin-env");
  printf("%s: admin - %d\n", __func__, rc);
  if (rc != SIMPLEAUTH_VALID)
    test_fail(__func__);
  test_pass();
}

void test_simpleauth_authenticate_user_admin_invalidpasswords()
{
  test_start();
  test_set_admin_password_env("password-admin-env");
  const char *bad_passwords[] = {
      "",
      "passw0rd-admin-env",
      "Password-admin-env",
      "pass",
      "password",
      "password-app",
      "password-app-env",
      "password-admin-env-123"};
  size_t bad_pass_len = sizeof(bad_passwords) / sizeof(bad_passwords[0]);

  for (int i = 0; i < bad_pass_len; i++)
  {
    int rc = simpleauth_authenticate_user("admin", bad_passwords[i]);
    printf("%s: admin/%s - %d\n", __func__, bad_passwords[i], rc);
    if (rc != SIMPLEAUTH_INVALID_PASSWORD)
      test_fail(__func__);
    test_pass();
  } 
}

void test_simpleauth_authenticate_user_admin_secret_file_valid()
{
  test_start();
  test_set_admin_password_file("password-admin-file");
  int rc = simpleauth_authenticate_user("admin", "password-admin-file");
  printf("%s: admin - %d\n", __func__, rc);
  if (rc != SIMPLEAUTH_VALID)
    test_fail(__func__);
  test_pass();
}

void test_simpleauth_authenticate_user_admin_secret_file_long()
{
  test_start();

  const int test_password_length = MAX_PASSWORD_LENGTH + 7;
  char test_password[test_password_length];
  char truncated_password[MAX_PASSWORD_LENGTH + 1];
  for (int i = 0; i < test_password_length; i++)
  {
    test_password[i] = '0' + ((i + 1) % 10);
    if (i < MAX_PASSWORD_LENGTH)
    {
      truncated_password[i] = test_password[i];
    }
  }
  test_password[test_password_length] = 0;
  truncated_password[MAX_PASSWORD_LENGTH] = 0;
  test_set_admin_password_file(test_password);

  int rc = simpleauth_authenticate_user("admin", test_password);
  if (rc != SIMPLEAUTH_INVALID_PASSWORD)
  {
    printf("%s: admin/'%s' - %d\n", __func__, test_password, rc);
    test_fail(__func__);
  }

  rc = simpleauth_authenticate_user("admin", truncated_password);
  if (rc != SIMPLEAUTH_VALID)
  {
    printf("%s: admin/'%s' - %d\n", __func__, truncated_password, rc);
    test_fail(__func__);
  }

  test_pass();
}

void test_simpleauth_authenticate_user_admin_secret_file_invalid()
{
  test_start();

  test_set_admin_password_file("password-admin-file");
  const char *bad_passwords[] = {
      "",
      "passw0rd-admin-file",
      "Password-admin-file",
      "pass",
      "password",
      "password-app-file",
      "password-admin-file-123"};
  size_t bad_pass_len = sizeof(bad_passwords) / sizeof(bad_passwords[0]);

  for (int i = 0; i < bad_pass_len; i++)
  {
    int rc = simpleauth_authenticate_user("admin", bad_passwords[i]);
    printf("%s: admin/%s - %d\n", __func__, bad_passwords[i], rc);
    if (rc != SIMPLEAUTH_INVALID_PASSWORD)
      test_fail(__func__);
    test_pass();
  }
}

void test_simpleauth_authenticate_user_app_secret_file_valid()
{
  test_start();
  test_set_app_password_file("password-app-file");
  int rc = simpleauth_authenticate_user("app", "password-app-file");
  printf("%s: admin - %d\n", __func__, rc);
  if (rc != SIMPLEAUTH_VALID)
    test_fail(__func__);
  test_pass();
}

void test_simpleauth_authenticate_user_app_secret_file_invalid()
{
  test_start();

  test_set_app_password_file("password-app-file");
  const char *bad_passwords[] = {
      "",
      "passw0rd-app-file",
      "Password-app-file",
      "pass",
      "password",
      "password-admin-file",
      "password-app-file-123"};
  size_t bad_pass_len = sizeof(bad_passwords) / sizeof(bad_passwords[0]);

  for (int i = 0; i < bad_pass_len; i++)
  {
    int rc = simpleauth_authenticate_user("app", bad_passwords[i]);
    printf("%s: app/%s - %d\n", __func__, bad_passwords[i], rc);
    if (rc != SIMPLEAUTH_INVALID_PASSWORD)
      test_fail(__func__);
    test_pass();
  }
}

// ----------------------------------------------------------------------------
// Multi-threaded test
// ----------------------------------------------------------------------------

#define NUM_THREADS 5
// Number of tests to perform per thread.  Higher numbers are more likely to trigger timing issue.
#define NUM_TESTS_PER_THREAD 1000
// Maximum number of JSON errors to report (log can get flooded)
#define MAX_JSON_ERRORS 10

// Authenticate multiple users, multiple times
void *authenticate_many_times(void *p)
{
  test_set_admin_password_env("passw0rd");
  test_set_app_password_env("passw0rd");
  for (int i = 0; i < NUM_TESTS_PER_THREAD; i++)
  {
    int rc = simpleauth_authenticate_user("admin", "passw0rd");
    if (rc != SIMPLEAUTH_VALID)
      test_fail(__func__);
    rc = simpleauth_authenticate_user("app", "passw0rd");
    if (rc != SIMPLEAUTH_VALID)
      test_fail(__func__);
  }
  pthread_exit(NULL);
}

void check_log_file_valid(char *filename)
{
  int errors = 0;
  printf("--- Checking log file is valid\n");
  // Check that the JSON log file isn't corrupted
  FILE *log = fopen(filename, "r");
  if (log == NULL)
  {
    test_fail(__func__);
  }
  const size_t line_size = 1024;
  char *line = malloc(line_size);
  while (fgets(line, line_size, log) != NULL)
  {
    if ((line[0] != '{') && (errors < MAX_JSON_ERRORS))
    {
      printf("*** Invalid JSON detected: %s\n", line);
      errors++;
    }
  }
  if (line)
  {
    free(line);
  }
  fclose(log);
}

// Test authenticate_user with multiple threads, each doing many authentications
void test_simpleauth_authenticate_user_multithreaded(char *logfile)
{
  pthread_t threads[NUM_THREADS];
  int rc;
  test_start();
  // Re-initialize the log to use a file for the multi-threaded test
  log_init(logfile);
  for (int i = 0; i < NUM_THREADS; i++)
  {
    printf("Creating thread %d\n", i);
    rc = pthread_create(&threads[i], NULL, authenticate_many_times, NULL);
    if (rc)
    {
      printf("Error: Unable to create thread, %d\n", rc);
      test_fail(__func__);
    }
  }
  // Wait for all the threads to complete
  for (int i = 0; i < NUM_THREADS; i++)
  {
    pthread_join(threads[i], NULL);
  }
  check_log_file_valid(logfile);
  test_pass();
}

// ----------------------------------------------------------------------------
// Test utility functions
// ----------------------------------------------------------------------------
int write_secret(const char *const secretFile, const char *const value)
{
  FILE *fp = fopen(secretFile, "w");
  if (fp)
  {
    int rc;
    rc = fprintf(fp, "%s\n", value);
    fclose(fp);
    return rc;
  }
  else
  {
    return 1;
  }
}

void test_set_admin_password_env(const char *const password)
{
  setenv("MQ_ADMIN_PASSWORD", password, 1);
  _mq_admin_secret_file = MQ_ADMIN_SECRET_FILE_DEFAULT;
}

void test_set_app_password_env(const char *const password)
{
  setenv("MQ_APP_PASSWORD", password, 1);
  _mq_app_secret_file = MQ_APP_SECRET_FILE_DEFAULT;
}

void test_set_admin_password_file(const char *const password)
{
  write_secret(MQ_ADMIN_SECRET_FILE_TEST, password);
  _mq_admin_secret_file = MQ_ADMIN_SECRET_FILE_TEST;
  unsetenv("MQ_ADMIN_PASSWORD");
}

void test_set_app_password_file(const char *const password)
{
  write_secret(MQ_APP_SECRET_FILE_TEST, password);
  _mq_app_secret_file = MQ_APP_SECRET_FILE_TEST;
  unsetenv("MQ_APP_PASSWORD");
}

// ----------------------------------------------------------------------------

int main()
{
  // Turn on debugging for the tests
  setenv("DEBUG", "true", true);
  log_init("simpleauth_test.log");
 
  test_read_secret_ok();
  test_simpleauth_valid_user_app_valid();
  test_simpleauth_valid_user_admin_valid();
  test_simpleauth_valid_user_george_invalid();
  test_simpleauth_authenticate_user_fred_unknown();
  test_simpleauth_authenticate_user_app_ok();
  test_simpleauth_authenticate_user_admin_ok();
  test_simpleauth_authenticate_user_admin_invalidpasswords();
  test_simpleauth_authenticate_user_admin_secret_file_valid();
  test_simpleauth_authenticate_user_admin_secret_file_long();
  test_simpleauth_authenticate_user_admin_secret_file_invalid();
  test_simpleauth_authenticate_user_app_secret_file_valid();
  test_simpleauth_authenticate_user_app_secret_file_invalid();
 
  log_close();

  // Call multi-threaded test last, because it re-initializes the log to use a file
  test_simpleauth_authenticate_user_multithreaded("simpleauth_test_multithreaded.log");
}