/*
© Copyright IBM Corporation 2025

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

package metrics

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
)

type auditEvent struct {
	Timestamp        string `json:"timestamp"`
	Event            string `json:"event"`
	Pod              string `json:"pod"`
	RemoteAddr       string `json:"remote_addr"`
	Endpoint         string `json:"endpoint"`
	Result           string `json:"result"`
	StatusCode       int    `json:"status_code"`
	QueuemanagerName string `json:"queuemanager_name"`
}

// passthroughHandlerFuncWrapper does not modify the base handler
func passthroughHandlerFuncWrapper(base http.HandlerFunc) http.HandlerFunc {
	return base
}

// newAuditingHandlerFuncWrapper generates a handlerFuncWrapper which allows creation of handlers that log audit entries for every request
func newAuditingHandlerFuncWrapper(qmName string, logger logHandler) handlerFuncWrapper {
	return func(base http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, req *http.Request) {
			podName, _ := os.Hostname()
			event := auditEvent{
				Timestamp:        time.Now().UTC().Format(time.RFC3339),
				Event:            "metrics",
				Pod:              podName,
				Endpoint:         req.URL.RequestURI(),
				QueuemanagerName: qmName,
				RemoteAddr:       req.RemoteAddr,
			}

			capWriter := newStatusCapturingResponseWriter(w)
			base(capWriter, req)
			statusCode := capWriter.statusCode
			event.StatusCode = statusCode
			event.Result = http.StatusText(statusCode)

			eventBytes, err := json.Marshal(event)
			if err != nil {
				logger.Append(fmt.Sprintf("Error writing audit log; next event may contain incomplete data: %s", err.Error()), false)
				fmt.Printf("Error constructing audit log event: %s\n", err.Error())
			}
			logger.Append(string(eventBytes), false)
		}
	}
}

// wrappedHandler implements http.Handler using a stored http.HandlerFunc for the ServeHTTP method
type wrappedHandler struct {
	handlerFunc http.HandlerFunc
}

func (wh wrappedHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	wh.handlerFunc(w, r)
}

// wrapHandler creates a new http.Handler with the function passed as wrapper around the base handler's ServeHTTP method, allowing augmentation of an existing http.Handler's ServeHTTP behaviour
func wrapHandler(base http.Handler, wrapperFunc handlerFuncWrapper) wrappedHandler {
	return wrappedHandler{
		handlerFunc: wrapperFunc(base.ServeHTTP),
	}
}

type handlerFuncWrapper func(base http.HandlerFunc) http.HandlerFunc

// statusCapturingResponseWriter captures the status code sent to the client
type statusCapturingResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func newStatusCapturingResponseWriter(base http.ResponseWriter) *statusCapturingResponseWriter {
	return &statusCapturingResponseWriter{
		ResponseWriter: base,
	}
}

func (c *statusCapturingResponseWriter) WriteHeader(statusCode int) {
	c.statusCode = statusCode
	c.ResponseWriter.WriteHeader(statusCode)
}

type logHandler interface {
	Append(messageLine string, deduplicateLine bool)
}
