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

#ifndef _HTPASS_H
#define _HTPASS_H

#define HTPASS_VALID 0
#define HTPASS_INVALID_USER 1
#define HTPASS_INVALID_PASSWORD 2

/**
 * Validate an HTPasswd file for use with IBM MQ.
 * 
 * @param filename the HTPasswd file
 */
_Bool htpass_valid_file(char *filename);

/**
 * Authenticate a user, based on the supplied file name.
 * 
 * @param filename the HTPasswd file
 * @param user the user name to authenticate
 * @param password the password of the user
 * @return HTPASS_VALID, HTPASS_INVALID_USER or HTPASS_INVALID_PASSWORD
 */
int htpass_authenticate_user(char *filename, char *user, char *password);

/**
 * Validate that a user exists in the password file.
 * 
 * @param filename the HTPasswd file
 * @param user the user name to validate
 */
_Bool htpass_valid_user(char *filename, char *user);

#endif