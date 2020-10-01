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

#include <stdio.h>
#include <stdlib.h>
#include "log.h"
#include "htpass.h"

void test_fail(const char *test_name) {
  printf("Failed test %s\n", test_name);
  exit(1);
}

void test_htpass_authenticate_user_fred_valid()
{
  int ok = htpass_authenticate_user("./src/htpass_test.htpasswd", "fred", "passw0rd");
  printf("%s: fred - %d\n", __func__, ok);
  if (!ok) test_fail(__func__);
}

void test_htpass_authenticate_user_fred_invalid1()
{
  int ok = htpass_authenticate_user("./src/htpass_test.htpasswd", "fred", "passw0rd ");
  printf("%s: fred - %d\n", __func__, ok);
  if (ok) test_fail(__func__);
}

void test_htpass_authenticate_user_fred_invalid2()
{
  int ok = htpass_authenticate_user("./src/htpass_test.htpasswd", "fred", "");
  printf("%s: fred - %d\n", __func__, ok);
  if (ok) test_fail(__func__);
}

void test_htpass_authenticate_user_fred_invalid3()
{
  int ok = htpass_authenticate_user("./src/htpass_test.htpasswd", "fred", "clearlywrong");
  printf("%s: fred - %d\n", __func__, ok);
  if (ok) test_fail(__func__);
}

void test_htpass_authenticate_user_barney_valid()
{
  int ok = htpass_authenticate_user("./src/htpass_test.htpasswd", "barney", "s3cret");
  printf("%s: barney - %d\n", __func__, ok);
  if (!ok) test_fail(__func__);
}

int main()
{
  log_init_file(stdout);
  printf("TESTING BEGINS\n");
  test_htpass_authenticate_user_fred_valid();
  test_htpass_authenticate_user_fred_invalid1();
  test_htpass_authenticate_user_fred_invalid2();
  test_htpass_authenticate_user_fred_invalid3();
  test_htpass_authenticate_user_barney_valid();
}