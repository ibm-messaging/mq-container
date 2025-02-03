/*
  Copyright (c) IBM Corporation 2024

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

/*
This module defines the interface for functions that will implement
OpenTelemetry context propagation.
*/
package ibmmq

import "context"

type OtelDisc func(*MQQueueManager)
type OtelPutTraceBefore func(OtelOpts, *MQQueueManager, *MQMD, *MQPMO, []byte)
type OtelPutTraceAfter func(OtelOpts, *MQQueueManager, *MQPMO)
type OtelGetTraceBefore func(OtelOpts, *MQQueueManager, *MQObject, *MQGMO, bool)
type OtelGetTraceAfter func(OtelOpts, *MQObject, *MQGMO, *MQMD, []byte, bool) int
type OtelOpen func(*MQObject, *MQOD, int32)
type OtelClose func(*MQObject)

type MQOtelFuncs struct {
	Disc           OtelDisc
	Open           OtelOpen
	Close          OtelClose
	PutTraceBefore OtelPutTraceBefore
	PutTraceAfter  OtelPutTraceAfter
	GetTraceBefore OtelGetTraceBefore
	GetTraceAfter  OtelGetTraceAfter
}

type OtelOpts struct {
	Context context.Context
	// During an MQGET, forcibly remove an RFH2 regardless of MQGMO and PROPCTL options
	// This might be needed if the only properties in an RFH2 are the propagated traceparent/state
	RemoveRFH2 bool
}

var otelFuncs MQOtelFuncs

func SetOtelFuncs(f MQOtelFuncs) {
	otelFuncs = f
}
