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
package kubeapi

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCheckPermission(t *testing.T) {
	// patchPods is the single permission the access-review server will grant.
	patchPods := requiredPermission{
		Verb:      "patch",
		Resource:  "pods",
		APIGroup:  "",
		Namespace: "default",
		Name:      "test-pod",
	}

	// deniedPerm is the permission the server will refuse.
	deniedPerm := requiredPermission{
		Verb:      "delete",
		Resource:  "secrets",
		APIGroup:  "fake.k8s.io",
		Namespace: "default",
		Name:      "test-pod",
	}

	tests := []struct {
		name string
		// perm is the permission passed to checkPermission.
		perm requiredPermission
		// serverFn overrides the test server when a custom response is needed.
		// When nil the standard access-review server is used.
		serverFn        func(t *testing.T) *httptest.Server
		wantErr         bool
		wantMissingPerm bool
		wantErrContains string
	}{
		{
			name:    "patch pods - permission granted",
			perm:    patchPods,
			wantErr: false,
		},
		{
			name:            "unknown resource - permission denied",
			perm:            deniedPerm,
			wantErr:         true,
			wantMissingPerm: true,
		},
		{
			name: "bad server - non-201 status",
			perm: patchPods,
			serverFn: func(t *testing.T) *httptest.Server {
				srv, _ := newBadServer(t)
				return srv
			},
			wantErr:         true,
			wantMissingPerm: false,
			wantErrContains: "unexpected status",
		},
		{
			name: "malformed response body",
			perm: patchPods,
			serverFn: func(t *testing.T) *httptest.Server {
				t.Helper()
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusCreated)
					_, _ = w.Write([]byte("not valid json{{"))
				}))
			},
			wantErr:         true,
			wantMissingPerm: false,
			wantErrContains: "decoding SelfSubjectAccessReview response",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var srv *httptest.Server
			if tt.serverFn != nil {
				srv = tt.serverFn(t)
			} else {
				srv = newAccessReviewServer(t, []requiredPermission{patchPods})
			}
			defer srv.Close()

			cfg := configForServer(srv)
			url := cfg.host + accessReviewPath

			err := checkPermission(t.Context(), cfg, url, tt.perm)

			if tt.wantErr {
				if err == nil {
					t.Errorf("checkPermission() expected an error, got nil")
					return
				}

				// Verify the error type
				permErr, isMissingPerm := errors.AsType[missingPermissionsError](err)
				if tt.wantMissingPerm && !isMissingPerm {
					t.Errorf("checkPermission() expected missingPermissionsError, got: %v", err)
				}
				if !tt.wantMissingPerm && isMissingPerm {
					t.Errorf("checkPermission() got missingPermissionsError, want a hard error: %v", err)
				}

				if tt.wantMissingPerm && isMissingPerm && permErr.permission != tt.perm {
					t.Errorf("checkPermission() missingPermissionsError.permission = %+v, want %+v",
						permErr.permission, tt.perm)
				}

				if tt.wantErrContains != "" && !strings.Contains(err.Error(), tt.wantErrContains) {
					t.Errorf("checkPermission() error = %q, want it to contain %q", err.Error(), tt.wantErrContains)
				}
				return
			}

			if err != nil {
				t.Errorf("checkPermission() unexpected error: %v", err)
			}
		})
	}
}
