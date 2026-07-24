/*
© Copyright IBM Corporation 2026

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

package fake

import (
	"net"
	"net/http"
	"net/http/httptest"
)

// NewMetadataServer creates an HTTP server listening on a Unix socket that
// handles only the given endpoint, returning the provided status code.
// Any other path returns 404, so a mis-routed call will fail the test.
func NewMetadataServer(socketPath, endpoint string, statusCode int) (*httptest.Server, error) {
	mux := http.NewServeMux()
	mux.HandleFunc(endpoint, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(statusCode)
	})

	srv := httptest.NewUnstartedServer(mux)
	unixListener, err := net.Listen("unix", socketPath)
	if err != nil {
		return nil, err
	}
	srv.Listener = unixListener
	return srv, nil
}
