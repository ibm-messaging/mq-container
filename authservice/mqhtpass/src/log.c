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
#include <stdarg.h>
#include <time.h>
#include <sys/time.h>
#include <unistd.h>

#define LOG_LEVEL_INFO 0
#define LOG_LEVEL_ERROR 1
#define LOG_LEVEL_DEBUG 2


FILE *fp = NULL;
int pid;

int log_init(char *filename)
{
  int result = 0;
  pid = getpid();
  if (!fp)
  {
    fp = fopen(filename, "a");
    if (fp)
    {
      setbuf(fp, NULL);
    }
    else
    {
      result = 1;
    }
  }
  return result;
}

void log_init_file(FILE *f)
{
  fp = f;
}

void log_close()
{
  if (fp)
  {
    fclose(fp);
    fp = NULL;
  }
}

void log_printf(const char *source_file, int source_line, const char *level, const char *format, ...)
{
  if (fp)
  {
    char buff[70];
    struct tm *utc;
    time_t t;

    struct timeval now;
    gettimeofday(&now, NULL);
    t = now.tv_sec;
    t = time(NULL);
    utc = gmtime(&t);

    fprintf(fp, "{");
    fprintf(fp, "\"loglevel\":\"%s\"", level);
    // Print ISO-8601 time and date
    if (strftime(buff, sizeof buff, "%FT%T", utc))
    {
      fprintf(fp, ", \"ibm_datetime\":\"%s.%3ld", buff, now.tv_usec);
    }
    fprintf(fp, ", \"ibm_processId\":\"%d\"", pid);
    fprintf(fp, ", \"module\":\"%s:%d\"", source_file, source_line);
    fprintf(fp, ", \"message\":\"");

    // Print log message, using varargs
    va_list args;
    va_start(args, format);
    vfprintf(fp, format, args);
    va_end(args);
    fprintf(fp, "\"}\n");
  }
}

/**
 * Writes a message to the log file, using the specified type, based on a printf format string.
 */
// void log_printf(const char *level, const char *format, va_list args)
// {
//   // FindSize();
//   if (fp)
//   {
//     char buff[70];
//     struct tm *utc;
//     time_t t;

//     struct timeval now;
//     gettimeofday(&now, NULL);
//     t = now.tv_sec;

//     // Print ISO-8601 time and date
//     t = time(NULL);
//     utc = gmtime(&t);
//     fprintf(fp, "{");
//     fprintf(fp, "\"loglevel\":\"%s\"", level);
//     if (strftime(buff, sizeof buff, "%FT%T", utc))
//     {
//       fprintf(fp, ", \"ibm_datetime\":\"%s.%3ld", buff, now.tv_usec);
//     }
//     fprintf(fp, ", \"ibm_processId\": \"%d\"", pid);
//     fprintf(fp, ", \"message\":\"");

//     // Print log message, using varargs
//     // va_list args;
//     // va_start(args, format);
//     vfprintf(fp, format, args);
//     // va_end(args);
//     fprintf(fp, "\"}\n");
//   }
// }

// void log_errorf(const char *format, ...)
// {
//     va_list args;
//     va_start(args, format);
//     log_printf("ERROR", format, args);
//     va_end(args);
// }

// void log_infof(const char *format, ...)
// {
//     va_list args;
//     va_start(args, format);
//     log_printf("INFO", format, args);
//     va_end(args);
// }

// void log_debugf(const char *format, ...)
// {
//     va_list args;
//     va_start(args, format);
//     log_printf("DEBUG", format, args);
//     va_end(args);
// }

// void log_debugf2(const char *source_file, const char *source_line, const char *format, ...)
// {
//     va_list args;
//     va_start(args, format);
//     log_printf(source_line, format, args);
//     va_end(args);
// }

