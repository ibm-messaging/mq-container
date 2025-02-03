package ibmmq

/*
  Copyright (c) IBM Corporation 2023

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
	"os"
	"strings"
	"time"
)

/*
 * A very simple tracing module that prints to stderr
 */

var tracing = false

// This is a public function so it could be called dynamically by
// an application at runtime
func SetTrace(b bool) {
	tracing = b
}

func logTrace(format string, v ...interface{}) {
	if tracing {
		d := time.Now().Format("2006-01-02T15:04:05.000")
		fmt.Fprintf(os.Stderr, "[ibmmq] %s : ", d)
		fmt.Fprintf(os.Stderr, format, v...)
		if !strings.HasSuffix(format, "\n") {
			fmt.Fprintf(os.Stderr, "\n")
		}
	}
}

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

func traceExit(f string) {
	traceExitF(f, 0, "Error: nil")
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
