/*
© Copyright IBM Corporation 2022

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
#include <string.h>
#include "log.h"

// Headers for multi-threaded tests
#include <pthread.h>

// Start a test and log the function name
#define test_start() printf("=== RUN: %s\n", __func__)

// Indicate test has passed
#define test_pass() printf("--- PASS: %s\n", __func__)

// The length of strings used in the tests
#define STR_LEN 5

// Indicate test has failed
void test_fail(const char *test_name)
{
  printf("--- FAIL: %s\n", test_name);
  exit(1);
}

// Print a fixed-width string in hexadecimal
void print_hex(char fw_string[STR_LEN])
{
  printf("[");
  for (int i=0; i<STR_LEN; i++)
  {
    printf("%02x", fw_string[i]);
    if (i < STR_LEN-1)
      printf(",");
  }
  printf("]");
}

// ----------------------------------------------------------------------------
// Tests for string manipulation
// ----------------------------------------------------------------------------

void test_trimmed_len(const char *test_name, char fw_string[STR_LEN], int expected_len)
{
  printf("=== RUN: %s\n", test_name);
  int len;
  // Create a copy of the fixed-width string
  char fw_string2[STR_LEN];
  memcpy(fw_string2, fw_string, STR_LEN * sizeof(char));
  // Call the function under test
  len = trimmed_len(fw_string, STR_LEN);
  // Check the result is correct
  if (len != expected_len)
  {
    printf("%s: Expected result to be %d; got %d\n", __func__, expected_len, len);
    test_fail(test_name);
  }
  // Check that the original string has not been changed
  for (int i=0; i<STR_LEN; i++)
  {
    if (fw_string[i] != fw_string2[i])
    {
      printf("%c-%c\n", fw_string[i], fw_string2[i]);
      printf("%s: Expected string to be identical to input hex ", __func__);
      print_hex(fw_string2);
      printf("; got hex ");
      print_hex(fw_string);
      printf("\n");
      test_fail(test_name);
    }
  }
  printf("--- PASS: %s\n", test_name);
}

void test_trimmed_len_normal()
{
  char fw_string[STR_LEN] = {'a','b','c',' ',' '};
  test_trimmed_len(__func__, fw_string, 3);
}

void test_trimmed_len_full()
{
  char fw_string[STR_LEN] = {'a','b','c','d','e'};
  test_trimmed_len(__func__, fw_string, 5);
}

void test_trimmed_len_empty()
{
  char fw_string[STR_LEN] = {' ',' ',' ',' ',' '};
  test_trimmed_len(__func__, fw_string, 0);
}

// ----------------------------------------------------------------------------

int main()
{
  // Turn on debugging for the tests
  setenv("DEBUG", "true", true);
  log_init("log_test.log");
  test_trimmed_len_normal();
  test_trimmed_len_full();
  test_trimmed_len_empty();
  log_close();
}