/*
© Copyright IBM Corporation 2021, 2022

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
 * Initialize the log to use the given file name, wiping any existing contents.
 */
int log_init_reset(char *filename);

/**
 * Initialize the log to use the given file name.
 */
int log_init(char *filename);

/**
 * Initialize the log with an existing file handle.
 */
void log_init_file(FILE *f);

/**
 * Write a message to the log file, based on a printf format string.
 * 
 * @param source_file the name of the source code file submitting this log message
 * @param source_line the line of code in the source file
 * @param level the log level, one of "DEBUG", "INFO" or "ERROR"
 * @param format the printf format string for the message
 */
void log_printf(const char *source_file, int source_line, const char *level, const char *format, ...);

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

/**
 * Return the length of the string when trimmed of trailing spaces.
 * IBM MQ uses fixed length strings, so this function can be used to print
 * a trimmed version of a string using the "%.*s" printf format string.
 * For example, `log_printf("%.*s", trimmed_len(fw_str, 48), fw_str)`
 */
int trimmed_len(char *s, int);

#endif