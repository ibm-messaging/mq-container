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
  char *pwd = readSecret("./src/mqAdminPassword");
  char *password = "passw0rd";
  if (0 == strcmp(pwd, password))
    test_pass();
  else
    test_fail(__func__);
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
  setenv("MQ_APP_PASSWORD", "passw0rd", 1);
  int rc = simpleauth_authenticate_user("fred", "passw0rd");
  printf("%s: fred - %d\n", __func__, rc);
  if (rc != SIMPLEAUTH_INVALID_USER)
    test_fail(__func__);
  test_pass();
}

void test_simpleauth_authenticate_user_app_ok()
{
  test_start();
  setenv("MQ_APP_PASSWORD", "passw0rd", 1);
  int rc = simpleauth_authenticate_user("app", "passw0rd");
  printf("%s: app - %d\n", __func__, rc);
  if (rc != SIMPLEAUTH_VALID)
    test_fail(__func__);
  test_pass();
}

void test_simpleauth_authenticate_user_admin_ok()
{
  test_start();
  setenv("MQ_ADMIN_PASSWORD", "passw0rd", 1);
  int rc = simpleauth_authenticate_user("admin", "passw0rd");
  printf("%s: admin - %d\n", __func__, rc);
  if (rc != SIMPLEAUTH_VALID)
    test_fail(__func__);
  test_pass();
}

void test_simpleauth_authenticate_user_app_invalidpwd()
{
  test_start();
  char *password[] = {"passw0r", "pass", "passw0rd1", "NULL", "","password123"};
  setenv("MQ_APP_PASSWORD", "passw0rd", 1);
  
  for(int i=0; i< (sizeof(password)/sizeof(password[0])); ++i)
  {
    int rc = simpleauth_authenticate_user("app", password[i]);
    printf("%s: Validating app user with password set to %s and rc is %d\n", __func__,password[i], rc);
    if (rc != SIMPLEAUTH_INVALID_PASSWORD)
       test_fail(__func__);   
  }
  test_pass();  
}

void test_simpleauth_authenticate_user_admin_invalidpwd()
{
  test_start();
  char *password[] = {"passw0r", "pass", "passw0rd1", "NULL", "","password123"};
  setenv("MQ_ADMIN_PASSWORD", "passw0rd", 1);
  
  for(int i=0; i< (sizeof(password)/sizeof(password[0])); ++i)
  {
    int rc = simpleauth_authenticate_user("admin", password[i]);
    printf("%s: validating admin user with password set to %s and rc is %d\n", __func__,password[i], rc);
    if (rc != SIMPLEAUTH_INVALID_PASSWORD)
       test_fail(__func__);   
  }
  test_pass();  
}

void test_simpleauth_authenticate_user_admin_with_null_pwd()
{
  test_start();
  setenv("MQ_ADMIN_PASSWORD", "", 1);
  int rc = simpleauth_authenticate_user("admin", "passw0rd");
  printf("%s: admin - %d\n", __func__, rc);
  if (rc == SIMPLEAUTH_VALID)
    test_fail(__func__);
  test_pass();
}

void test_simpleauth_authenticate_user_admin_invalidpassword()
{
  test_start();
  setenv("MQ_ADMIN_PASSWORD", "password", 1);
  int rc = simpleauth_authenticate_user("admin", "passw0rd");
  printf("%s: admin - %d\n", __func__, rc);
  if (rc != SIMPLEAUTH_INVALID_PASSWORD)
    test_fail(__func__);
  test_pass();
}

void test_simpleauth_authenticate_user_admin_invalishortdpassword()
{
  test_start();
  setenv("MQ_ADMIN_PASSWORD", "password", 1);
  int rc = simpleauth_authenticate_user("admin", "pass");
  printf("%s: admin - %d\n", __func__, rc);
  if (rc != SIMPLEAUTH_INVALID_PASSWORD)
    test_fail(__func__);
  test_pass();
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
  setenv("MQ_ADMIN_PASSWORD", "passw0rd", 1);
  setenv("MQ_APP_PASSWORD", "passw0rd", 1);
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

int main()
{
  // Turn on debugging for the tests
  setenv("DEBUG", "true", true);
  log_init("simpleauth_test.log");
  test_read_secret_ok();
  test_simpleauth_authenticate_user_admin_invalidpwd();
  test_simpleauth_authenticate_user_app_invalidpwd();
  test_simpleauth_valid_user_app_valid();
  test_simpleauth_valid_user_admin_valid();
  test_simpleauth_valid_user_george_invalid();
  test_simpleauth_authenticate_user_fred_unknown();
  test_simpleauth_authenticate_user_app_ok();
  test_simpleauth_authenticate_user_admin_with_null_pwd();
  test_simpleauth_authenticate_user_admin_ok();
  test_simpleauth_authenticate_user_admin_invalidpassword();
  test_simpleauth_authenticate_user_admin_invalishortdpassword();
 
  log_close();

  // Call multi-threaded test last, because it re-initializes the log to use a file
  test_simpleauth_authenticate_user_multithreaded("simpleauth_test_multithreaded.log");
}