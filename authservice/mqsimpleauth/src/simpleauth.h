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

#ifndef _SIMPLEAUTH_H
#define _SIMPLEAUTH_H

#define SIMPLEAUTH_VALID 0
#define SIMPLEAUTH_INVALID_USER 1
#define SIMPLEAUTH_INVALID_PASSWORD 2
#define MQ_APP_SECRET_FILE "/run/secrets/mqAppPassword"
#define MQ_ADMIN_SECRET_FILE "/run/secrets/mqAdminPassword"
#define APP_USER_NAME "app"
#define ADMIN_USER_NAME "admin"

/**
 * Authenticate a user, based on the supplied file name.
 *
 * @param user the user name to authenticate
 * @param password the password of the user
 * @return SIMPLEAUTH_VALID, SIMPLEAUTH_INVALID_USER or SIMPLEAUTH_INVALID_PASSWORD
 */
int simpleauth_authenticate_user(char *user, char *password);

/**
 * Validate that a user exists in the password file.
 *
 * @param user the user name to validate
 */
bool simpleauth_valid_user(char *user);

/**
 * Get the secret of the UserId.
 *
 * @param user the user name to validate
 */
char *getSecretForUser(char *user);

/**
 * Get the secret of the UserId.
 *
 * @param secret path for the secret file
*/
char *readSecret(char* secret);

#endif