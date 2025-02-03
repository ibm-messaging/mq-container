package mqmetric

/*
  Copyright (c) IBM Corporation 2016, 2020

  Licensed under the Apache License, Version 2.0 (the "License");
  you may not use this file except in compliance with the License.
  You may obtain a copy of the License at

  http://www.apache.org/licenses/LICENSE-2.0

  Unless required by applicable law or agreed to in writing, software
  distributed under the License is distributed on an "AS IS" BASIS,
  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
  See the License for the specific language governing permissions and
  limitations under the License.

   Contributors:
     Mark Taylor - Initial Contribution
*/

import (
	"fmt"
)

// Setup for the 7 levels of logging that logrus allows, even if we don't
// intend to use all of them for now.
type Logger struct {
	Trace func(string, ...interface{})
	Debug func(string, ...interface{})
	Info  func(string, ...interface{})
	Warn  func(string, ...interface{})
	Error func(string, ...interface{})
	Fatal func(string, ...interface{})
	Panic func(string, ...interface{})
}

var logger *Logger = nil

func SetLogger(l *Logger) {
	logger = l
}

func logTrace(format string, v ...interface{}) {
	if logger != nil && logger.Trace != nil {
		logger.Trace(format, v...)
	}
}
func logDebug(format string, v ...interface{}) {
	if logger != nil && logger.Debug != nil {
		logger.Debug(format, v...)
	}
}
func logInfo(format string, v ...interface{}) {
	if logger != nil && logger.Info != nil {
		logger.Info(format, v...)
	}
}
func logWarn(format string, v ...interface{}) {
	if logger != nil && logger.Warn != nil {
		logger.Warn(format, v...)
	}
}

// Errors should be reported always. Also use this for what you might
// think of as warnings.
func logError(format string, v ...interface{}) {
	if logger != nil && logger.Error != nil {
		logger.Error(format, v...)
	} else {
		fmt.Printf(format, v...)
	}
}

// Panic and Fatal are not going to be used for now but
// are in here for completeness
/*
func logFatal(format string, v ...interface{}) {
	if logger != nil && logger.Fatal != nil {
		logger.Fatal(format, v...)
	}
}
func logPanic(format string, v ...interface{}) {
	if logger != nil && logger.Panic != nil {
		logger.Panic(format, v...)
	}
}
*/

// Some interfaces to enable tracing. In its simplest form, tracing the
// entry point just needs the function name. There are often several exit
// points from functions when we short-circuit via early parameter tests,
// so we make them unique with a mandatory returnPoint value.
// More sophisticated tracing of input and output values can be done with the
// EntryF and ExitF functions that take the usual formatting strings.
func traceEntry(f string) {
	traceEntryF(f, "")
}
func traceEntryF(f string, format string, v ...interface{}) {
	if format != "" {
		fs := make([]interface{}, 1)
		fs[0] = f
		logTrace("> [%s] : "+format, append(fs, v...)...)
	} else {
		logTrace("> [%s]", f)
	}
}

func traceExit(f string, returnPoint int) {
	traceExitF(f, returnPoint, "")
}
func traceExitErr(f string, returnPoint int, err error) {
	if err == nil {
		traceExitF(f, returnPoint, "Error: nil")
	} else {
		traceExitF(f, returnPoint, "Error: %v", err)
	}
}

func traceExitF(f string, returnPoint int, format string, v ...interface{}) {
	if format != "" {
		fs := make([]interface{}, 2)
		fs[0] = f
		fs[1] = returnPoint
		if len(v) > 0 {
			fs = append(fs, v...)
		}
		logTrace("< [%s] rp: %d "+format, fs...)
	} else {
		logTrace("< [%s] rp: %d", f, returnPoint)
	}
}
