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

#ifndef _LOG_H
#define _LOG_H

/**
 * Initialize the log to use the given file name.
 */
int log_init(char *);

/**
 * Initialize the log with an existing file handle.
 */
void log_init_file(FILE *);

/**
 * Write a message to the log file, based on a printf format string.
 */
void log_printf(const char*, int, const char*, const char*, ...);

void log_close();

/**
 * Variadic macro to write an informational message to the log file, based on a printf format string.
 */
#define log_infof(format,...) log_printf(__FILE__, __LINE__, "INFO", format, ##__VA_ARGS__)

/**
 * Variadic macro to write an error message to the log file, based on a printf format string.
 */
#define log_errorf(format,...) log_printf(__FILE__, __LINE__, "ERROR", format, ##__VA_ARGS__)

/**
 * Variadic macro to write a debug message to the log file, based on a printf format string.
 */
#define log_debugf(format,...) log_printf(__FILE__, __LINE__, "DEBUG", format, ##__VA_ARGS__)


#endif