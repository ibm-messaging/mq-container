/*
© Copyright IBM Corporation 2021

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
#include "htpass.h"

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
// Simple tests for file validation
// ----------------------------------------------------------------------------

void test_htpass_valid_file_ok()
{
  test_start();
  int ok = htpass_valid_file("./src/htpass_test.htpasswd");
  if (!ok)
    test_fail(__func__);
  test_pass();
}

void test_htpass_valid_file_too_long()
{
  test_start();
  int ok = htpass_valid_file("./src/htpass_test_invalid.htpasswd");
  if (ok)
    test_fail(__func__);
  test_pass();
}

// ----------------------------------------------------------------------------
// Simple tests for authentication
// ----------------------------------------------------------------------------

void test_htpass_authenticate_user_fred_valid()
{
  test_start();
  int rc = htpass_authenticate_user("./src/htpass_test.htpasswd", "fred", "passw0rd");
  printf("%s: fred - %d\n", __func__, rc);
  if (rc != HTPASS_VALID)
    test_fail(__func__);
  test_pass();
}

void test_htpass_authenticate_user_fred_invalid1()
{
  test_start();
  int rc = htpass_authenticate_user("./src/htpass_test.htpasswd", "fred", "passw0rd ");
  printf("%s: fred - %d\n", __func__, rc);
  if (rc != HTPASS_INVALID_PASSWORD)
    test_fail(__func__);
  test_pass();
}

void test_htpass_authenticate_user_fred_invalid2()
{
  test_start();
  int rc = htpass_authenticate_user("./src/htpass_test.htpasswd", "fred", "");
  printf("%s: fred - %d\n", __func__, rc);
  if (rc != HTPASS_INVALID_PASSWORD)
    test_fail(__func__);
  test_pass();
}

void test_htpass_authenticate_user_fred_invalid3()
{
  test_start();
  int rc = htpass_authenticate_user("./src/htpass_test.htpasswd", "fred", "clearlywrong");
  printf("%s: fred - %d\n", __func__, rc);
  if (rc != HTPASS_INVALID_PASSWORD)
    test_fail(__func__);
  test_pass();
}

void test_htpass_authenticate_user_barney_valid()
{
  test_start();
  int rc = htpass_authenticate_user("./src/htpass_test.htpasswd", "barney", "s3cret");
  printf("%s: barney - %d\n", __func__, rc);
  if (rc != HTPASS_VALID)
    test_fail(__func__);
  test_pass();
}

void test_htpass_authenticate_user_unknown()
{
  test_start();
  int rc = htpass_authenticate_user("./src/htpass_test.htpasswd", "george", "s3cret");
  printf("%s: barney - %d\n", __func__, rc);
  if (rc != HTPASS_INVALID_USER)
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
  for (int i = 0; i < NUM_TESTS_PER_THREAD; i++)
  {
    int rc = htpass_authenticate_user("./src/htpass_test.htpasswd", "barney", "s3cret");
    if (rc != HTPASS_VALID)
      test_fail(__func__);
    rc = htpass_authenticate_user("./src/htpass_test.htpasswd", "fred", "passw0rd");
    if (rc != HTPASS_VALID)
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
void test_htpass_authenticate_user_multithreaded(char *logfile)
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
  log_init("htpass_test.log");
  test_htpass_valid_file_ok();
  test_htpass_valid_file_too_long();
  test_htpass_authenticate_user_fred_valid();
  test_htpass_authenticate_user_fred_invalid1();
  test_htpass_authenticate_user_fred_invalid2();
  test_htpass_authenticate_user_fred_invalid3();
  test_htpass_authenticate_user_barney_valid();
  test_htpass_authenticate_user_unknown();
  log_close();

  // Call multi-threaded test last, because it re-initializes the log to use a file
  test_htpass_authenticate_user_multithreaded("htpass_test_multithreaded.log");
}