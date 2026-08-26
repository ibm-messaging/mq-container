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
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ibm-messaging/mq-container/internal/ready"
)

const accessReviewPath = "/apis/authorization.k8s.io/v1/selfsubjectaccessreviews"

// newAccessReviewServer starts a mock Kubernetes API server that handles POST
// requests to the SelfSubjectAccessReview endpoint. It decodes the full
// resourceAttributes from the request body and grants access only when all
// five fields (verb, resource, group, namespace, name) match an entry in
// allowed. Any other route or method returns 400 Bad Request.
// The caller must call Close() on the returned server when done.
func newAccessReviewServer(t *testing.T, allowed []requiredPermission) *httptest.Server {
	t.Helper()

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Reject anything that is not a POST to accessReviewPath.
		if r.Method != http.MethodPost || r.URL.Path != accessReviewPath {
			http.Error(w, "unexpected request", http.StatusBadRequest)
			return
		}

		var payload selfSubjectAccessReviewPayload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, "cannot decode body", http.StatusBadRequest)
			return
		}

		attrs := payload.Spec.ResourceAttributes

		// Check whether the requested permission matches any allowed entry.
		grant := false
		for _, p := range allowed {
			if attrs.Verb == p.Verb &&
				attrs.Resource == p.Resource &&
				attrs.Group == p.APIGroup &&
				attrs.Namespace == p.Namespace &&
				attrs.Name == p.Name {
				grant = true
				break
			}
		}

		var result selfSubjectAccessReviewStatus
		result.Status.Allowed = grant
		if !grant {
			result.Status.Reason = "no RBAC policy grants the requested permission"
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(result)
	}))
}

// newBadServer starts a server that always returns 500 Internal Server Error,
// simulating an API server that is unhealthy or unreachable.
func newBadServer(t *testing.T) (*httptest.Server, *int) {
	t.Helper()
	requestCount := new(int)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		*requestCount++
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}))
	return srv, requestCount
}

// configForServer builds an inClusterConfig for a mocked server
func configForServer(srv *httptest.Server) *inClusterConfig {
	return &inClusterConfig{
		host:      srv.URL,
		token:     "test-token",
		namespace: "default",
		podName:   "test-pod",
		client:    srv.Client(),
	}
}

func TestSelfSubjectAccessReview(t *testing.T) {
	// Permissions that will be granted if requested
	knownGoodPerms := []requiredPermission{
		{Verb: "patch", Resource: "pods", APIGroup: "", Namespace: "default", Name: "test-pod"},
		{Verb: "get", Resource: "pods", APIGroup: "", Namespace: "default", Name: "test-pod"},
	}

	tests := []struct {
		name string
		// which permissions to request
		permissions []requiredPermission
		// useBadServer replaces the mock API server with one that always errors.
		useBadServer bool
		wantErr      bool
		// wantErrContains is a substring that must appear in the error message.
		wantErrContains string
	}{
		{
			// Single valid permission
			name: "patch pods - single allowed permission",
			permissions: []requiredPermission{
				{Verb: "patch", Resource: "pods", APIGroup: "", Namespace: "default", Name: "test-pod"},
			},
			wantErr: false,
		},
		{
			// Two valid permissions
			name: "patch pods + get pods - two allowed permissions",
			permissions: []requiredPermission{
				{Verb: "patch", Resource: "pods", APIGroup: "", Namespace: "default", Name: "test-pod"},
				{Verb: "get", Resource: "pods", APIGroup: "", Namespace: "default", Name: "test-pod"},
			},
			wantErr: false,
		},
		{
			// Request an invalid permission
			name: "invalid resource - server denies unknown resource",
			permissions: []requiredPermission{
				{Verb: "get", Resource: "nonexistentresource", APIGroup: "fake.k8s.io", Namespace: "default", Name: "test-pod"},
			},
			wantErr:         true,
			wantErrContains: "missing required permissions",
		},
		{
			// Request no permissions
			name:        "no permissions - success",
			permissions: []requiredPermission{},
			wantErr:     false,
		},
		{
			// Valid permission but the mock server returns 500.
			name: "get pods - bad server returns non-201 status",
			permissions: []requiredPermission{
				{Verb: "get", Resource: "pods", APIGroup: "", Namespace: "default", Name: "test-pod"},
			},
			useBadServer: true,
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var srv *httptest.Server
			if tt.useBadServer {
				srv, _ = newBadServer(t)
			} else {
				srv = newAccessReviewServer(t, knownGoodPerms)
			}
			defer srv.Close()

			client := &Client{}

			err := client.selfSubjectAccessReview(t.Context(), configForServer(srv), tt.permissions)

			if tt.wantErr {
				if err == nil {
					t.Errorf("selfSubjectAccessReview() expected an error, got nil")
					return
				}
				if tt.wantErrContains != "" && !strings.Contains(err.Error(), tt.wantErrContains) {
					t.Errorf("selfSubjectAccessReview() error = %q, want it to contain %q", err.Error(), tt.wantErrContains)
				}
				// When no specific substring is expected (bad server), confirm it is
				// not a missing-permission denial.
				_, isMissingPerm := errors.AsType[missingPermissionsError](err)
				if tt.wantErrContains == "" && isMissingPerm {
					t.Errorf("selfSubjectAccessReview() got a missingPermissionsError, want a hard server error: %v", err)
				}
				return
			}

			if err != nil {
				t.Errorf("selfSubjectAccessReview() unexpected error: %v", err)
			}
		})
	}
}

func TestVerifyPermissions(t *testing.T) {
	// Expected set of permissions
	patchPodsPermission := requiredPermission{
		Verb:      "patch",
		Resource:  "pods",
		APIGroup:  "",
		Namespace: "default",
		Name:      "test-pod",
	}

	tests := []struct {
		name string
		// loadConfigErr causes loadConfig to return an error before any HTTP call.
		loadConfigErr error
		// useBadServer replaces the normal mock with one that always returns 500.
		useBadServer bool
		// serverPermissions controls which permissions the mock server will grant.
		serverPermissions []requiredPermission
		// wantErr is true whenever VerifyPermissions should return a non-nil error.
		wantErr bool
		// wantErrContains is a substring that must appear in the error message.
		wantErrContains string
	}{
		{
			// loadConfig returns an error before any HTTP call is made.
			name:            "loadConfig fails - returns config error",
			loadConfigErr:   fmt.Errorf("simulated config failure"),
			wantErr:         true,
			wantErrContains: "loading in-cluster config",
		},
		{
			// Server grants the patch-pods permission — VerifyPermissions should succeed.
			name:              "all permissions granted - success",
			serverPermissions: []requiredPermission{patchPodsPermission},
			wantErr:           false,
		},
		{
			// Server grants no permissions — expect "missing required permissions".
			name:              "permission denied - server denies patch pods",
			serverPermissions: []requiredPermission{},
			wantErr:           true,
			wantErrContains:   "missing required permissions",
		},
		{
			// Server returns 500.
			name:         "bad server - returns non-201 status",
			useBadServer: true,
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var srv *httptest.Server
			if tt.loadConfigErr == nil {
				// Only start a server when loadConfig itself is not the failure point.
				if tt.useBadServer {
					srv, _ = newBadServer(t)
				} else {
					srv = newAccessReviewServer(t, tt.serverPermissions)
				}
				defer srv.Close()
			}

			client := &Client{
				loadConfig: func() (*inClusterConfig, error) {
					if tt.loadConfigErr != nil {
						return nil, tt.loadConfigErr
					}
					return configForServer(srv), nil
				},
			}

			err := client.VerifyPermissions(t.Context())

			if tt.wantErr {
				if err == nil {
					t.Errorf("VerifyPermissions() expected an error, got nil")
					return
				}
				if tt.wantErrContains != "" && !strings.Contains(err.Error(), tt.wantErrContains) {
					t.Errorf("VerifyPermissions() error = %q, want it to contain %q", err.Error(), tt.wantErrContains)
				}
				// When no specific substring is expected (bad server), confirm it is
				// not missing-permission.
				_, isMissingPerm := errors.AsType[missingPermissionsError](err)
				if tt.wantErrContains == "" && isMissingPerm {
					t.Errorf("VerifyPermissions() got a missingPermissionsError, want a hard server error: %v", err)
				}
				return
			}

			if err != nil {
				t.Errorf("VerifyPermissions() unexpected error: %v", err)
			}
		})
	}
}

// podPatchPath returns the API path for patching a specific pod.
func podPatchPath(namespace, podName string) string {
	return fmt.Sprintf("/api/v1/namespaces/%s/pods/%s", namespace, podName)
}

// newPodPatchServer starts a test HTTP server that counts all incoming requests
// and records the last decoded patch body. requestCount is incremented on every
// request. Valid PATCHes are decoded into lastPatch and return 200; any other
// method, path, or Content-Type returns 400.
func newPodPatchServer(t *testing.T, namespace, podName string) (*httptest.Server, *int, *podPatch) {
	t.Helper()
	path := podPatchPath(namespace, podName)
	requestCount := new(int)
	lastPatch := new(podPatch)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		*requestCount++

		if r.Method != http.MethodPatch || r.URL.Path != path {
			http.Error(w, "unexpected request", http.StatusBadRequest)
			return
		}

		contentType := r.Header.Get("Content-Type")
		if contentType != "application/strategic-merge-patch+json" {
			http.Error(w, "unexpected Content-Type: "+contentType, http.StatusBadRequest)
			return
		}

		if err := json.NewDecoder(r.Body).Decode(lastPatch); err != nil {
			http.Error(w, "decoding body: "+err.Error(), http.StatusBadRequest)
			return
		}

		w.WriteHeader(http.StatusOK)
	}))
	return srv, requestCount, lastPatch
}

// downwardAPILines formats key/value pairs into the Kubernetes downward API format
func downwardAPILines(pairs map[string]string) []byte {
	var sb strings.Builder
	for k, v := range pairs {
		// Wrap value in double quotes to match the downward API format.
		fmt.Fprintf(&sb, "%s=%q\n", k, v)
	}
	return []byte(sb.String())
}

// metaReaderWith builds a podMetaReader which returns local data, or an error if the provided data is nil
func metaReaderWith(annotationsData, labelsData []byte) *podMetaReader {
	fileOrError := func(data []byte) func() ([]byte, error) {
		if data == nil {
			return func() ([]byte, error) { return nil, fmt.Errorf("simulated file read error") }
		}
		return func() ([]byte, error) { return data, nil }
	}
	return &podMetaReader{
		readAnnotationsFile: fileOrError(annotationsData),
		readLabelsFile:      fileOrError(labelsData),
	}
}

func TestPatchPodMetadataIfNeeded(t *testing.T) {
	const (
		testNamespace = "default"
		testPodName   = "test-pod"
	)

	annotationsForStatus := func(status ready.QMStatus) []byte {
		switch status {
		case ready.StatusActiveQM, ready.StatusRecoveryLeaderQM:
			return downwardAPILines(map[string]string{
				annotationWhenActive: `{"productChargedContainers":"All","productID":"1234"}`,
			})
		case ready.StatusStandbyQM:
			return downwardAPILines(map[string]string{
				annotationWhenStandby: `{"productChargedContainers":"None","productID":"1234"}`,
			})
		case ready.StatusReplicaQM:
			return downwardAPILines(map[string]string{
				annotationWhenReplica: `{"productChargedContainers":"None","productID":"1234"}`,
			})
		default:
			return downwardAPILines(map[string]string{})
		}
	}

	// labelsStale returns a labels file whose mq.ibm.com/status value differs
	// from wantLabel, ensuring matchesDesired returns false (patch proceeds).
	labelsStale := func() []byte {
		return downwardAPILines(map[string]string{labelMQStatus: "stale-value"})
	}

	tests := []struct {
		name              string
		status            ready.QMStatus
		podMetadataReader *podMetaReader
		// loadConfigErr causes loadConfig to fail
		loadConfigErr error
		// useBadServer replaces the normal mock with one that always returns 500.
		useBadServer    bool
		wantErr         bool
		wantErrContains string
		// wantNoPatch validates that no patch request is made
		wantNoPatch bool
		// wantLabel is the expected mq.ibm.com/status label value in the patch body.
		// Only checked when a patch is expected and no error is expected.
		wantLabel string
		// wantAnnotations is the expected annotations map in the patch body.
		// A nil value asserts that no annotations are sent.
		// Only checked when a patch is expected and no error is expected.
		wantAnnotations map[string]string
	}{
		{
			// StatusUnknown has no label mapping, so the function returns an error
			// immediately.
			name:            "StatusUnknown - no label mapping returns error",
			status:          ready.StatusUnknown,
			wantErr:         true,
			wantErrContains: "no label value for status",
		},
		{
			name:            "StatusActiveQM - patch sent with label 'active'",
			status:          ready.StatusActiveQM,
			wantLabel:       "active",
			wantAnnotations: map[string]string{"productChargedContainers": "All", "productID": "1234"},
		},
		{
			name:            "StatusStandbyQM - patch sent with label 'standby'",
			status:          ready.StatusStandbyQM,
			wantLabel:       "standby",
			wantAnnotations: map[string]string{"productChargedContainers": "None", "productID": "1234"},
		},
		{
			name:            "StatusReplicaQM - patch sent with label 'replica'",
			status:          ready.StatusReplicaQM,
			wantLabel:       "replica",
			wantAnnotations: map[string]string{"productChargedContainers": "None", "productID": "1234"},
		},
		{
			name:            "StatusRecoveryLeaderQM - patch sent with label 'active'",
			status:          ready.StatusRecoveryLeaderQM,
			wantLabel:       "active",
			wantAnnotations: map[string]string{"productChargedContainers": "All", "productID": "1234"},
		},
		{
			name:            "StatusStarting - patch sent with label 'starting', no license annotations",
			status:          ready.StatusStarting,
			wantLabel:       "starting",
			wantAnnotations: nil,
		},
		{
			name:   "already up-to-date - no patch sent",
			status: ready.StatusActiveQM,
			podMetadataReader: metaReaderWith(
				downwardAPILines(map[string]string{
					// Meta-annotation: tells resolveLicenseAnnotations what to apply.
					annotationWhenActive: `{"productChargedContainers":"All","productID":"1234"}`,
					// Active annotations already present on the live pod, so matchesDesired returns true.
					"productChargedContainers": "All",
					"productID":                "1234",
				}),
				downwardAPILines(map[string]string{labelMQStatus: "active"}),
			),
			wantNoPatch: true,
		},
		{
			name:            "label stale, annotations correct - patch sent",
			status:          ready.StatusActiveQM,
			wantLabel:       "active",
			wantAnnotations: map[string]string{"productChargedContainers": "All", "productID": "1234"},
			podMetadataReader: metaReaderWith(
				downwardAPILines(map[string]string{
					annotationWhenActive:       `{"productChargedContainers":"All","productID":"1234"}`,
					"productChargedContainers": "All",
					"productID":                "1234",
				}),
				downwardAPILines(map[string]string{labelMQStatus: "starting"}),
			),
		},
		{
			name:            "label correct, annotations stale - patch sent",
			status:          ready.StatusActiveQM,
			wantLabel:       "active",
			wantAnnotations: map[string]string{"productChargedContainers": "All", "productID": "1234"},
			podMetadataReader: metaReaderWith(
				downwardAPILines(map[string]string{
					annotationWhenActive:       `{"productChargedContainers":"All","productID":"1234"}`,
					"productChargedContainers": "All",
					"productID":                "wrong-value",
				}),
				downwardAPILines(map[string]string{labelMQStatus: "active"}),
			),
		},
		{
			name:   "annotations file is unreadable",
			status: ready.StatusActiveQM,
			podMetadataReader: metaReaderWith(
				nil,
				labelsStale(),
			),
			wantErr:         true,
			wantErrContains: "resolving license annotations",
		},
		{
			// loadConfig fails inside patch()
			name:            "loadConfig fails",
			status:          ready.StatusActiveQM,
			loadConfigErr:   fmt.Errorf("simulated config failure"),
			wantErr:         true,
			wantErrContains: "loading in-cluster config",
		},
		{
			// The Kubernetes API server returns 500.
			name:         "bad server",
			status:       ready.StatusActiveQM,
			useBadServer: true,
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// If meta is not explicitly set in the test default to stale
			meta := tt.podMetadataReader
			if meta == nil {
				meta = metaReaderWith(
					annotationsForStatus(tt.status),
					labelsStale(),
				)
			}

			var srv *httptest.Server
			var requestCount *int
			var lastPatch *podPatch
			switch {
			case tt.useBadServer:
				srv, requestCount = newBadServer(t)
				defer srv.Close()
			case tt.loadConfigErr == nil:
				srv, requestCount, lastPatch = newPodPatchServer(t, testNamespace, testPodName)
				defer srv.Close()
			}

			client := &Client{
				loadConfig: func() (*inClusterConfig, error) {
					if tt.loadConfigErr != nil {
						return nil, tt.loadConfigErr
					}
					cfg := configForServer(srv)
					cfg.namespace = testNamespace
					cfg.podName = testPodName
					return cfg, nil
				},
				meta: meta,
			}

			err := client.PatchPodMetadataIfNeeded(t.Context(), tt.status)

			if tt.wantErr {
				if err == nil {
					t.Errorf("PatchPodMetadataIfNeeded() expected an error, got nil")
					return
				}
				if tt.wantErrContains != "" && !strings.Contains(err.Error(), tt.wantErrContains) {
					t.Errorf("PatchPodMetadataIfNeeded() error = %q, want it to contain %q",
						err.Error(), tt.wantErrContains)
				}
				return
			}

			if err != nil {
				t.Errorf("PatchPodMetadataIfNeeded() unexpected error: %v", err)
			}

			if tt.wantNoPatch && *requestCount != 0 {
				t.Errorf("PatchPodMetadataIfNeeded() made %d request(s), want none", *requestCount)
			}
			if !tt.wantNoPatch && *requestCount != 1 {
				t.Errorf("PatchPodMetadataIfNeeded() made %d request(s), want exactly 1", *requestCount)
			}
			if !tt.wantNoPatch && lastPatch != nil {
				if got := lastPatch.Metadata.Labels[labelMQStatus]; got != tt.wantLabel {
					t.Errorf("patch label %q = %q, want %q", labelMQStatus, got, tt.wantLabel)
				}
				if tt.wantAnnotations == nil && lastPatch.Metadata.Annotations != nil {
					t.Errorf("patch annotations = %v, want none", lastPatch.Metadata.Annotations)
				}
				for k, want := range tt.wantAnnotations {
					if got := lastPatch.Metadata.Annotations[k]; got != want {
						t.Errorf("patch annotation %q = %q, want %q", k, got, want)
					}
				}
			}
		})
	}
}

func TestLoadInClusterConfig(t *testing.T) {
	// validCA generates a minimal self-signed PEM certificate usable as a CA.
	validCA := func(t *testing.T) []byte {
		t.Helper()
		priv, err := rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			t.Fatalf("generating RSA key: %v", err)
		}
		certTemplate := &x509.Certificate{
			SerialNumber: big.NewInt(1),
			Subject:      pkix.Name{CommonName: "test-ca"},
			IsCA:         true,
			KeyUsage:     x509.KeyUsageCertSign,
		}
		der, err := x509.CreateCertificate(rand.Reader, certTemplate, certTemplate, &priv.PublicKey, priv)
		if err != nil {
			t.Fatalf("creating certificate: %v", err)
		}
		return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	}

	// writeFile writes data to a temporary file and returns its path.
	writeFile := func(t *testing.T, dir, name string, data []byte) string {
		t.Helper()
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, data, 0600); err != nil {
			t.Fatalf("writing file %s: %v", path, err)
		}
		return path
	}

	// setEnv sets an environment variable for the duration of the test,
	// restoring the original value (or unsetting it) when the test ends.
	setEnv := func(t *testing.T, key, value string) {
		t.Helper()
		original, existed := os.LookupEnv(key)
		if err := os.Setenv(key, value); err != nil {
			t.Fatalf("setting env %s: %v", key, err)
		}
		t.Cleanup(func() {
			if existed {
				_ = os.Setenv(key, original)
			} else {
				_ = os.Unsetenv(key)
			}
		})
	}

	tests := []struct {
		name string
		// setup configures env vars and files before loadInClusterConfig is called.
		setup           func(t *testing.T)
		wantErr         bool
		wantErrContains string
		// wantCfg, when non-nil, describes the fields expected in the returned config.
		wantCfg *inClusterConfig
	}{
		{

			name: "missing KUBERNETES_SERVICE_HOST and KUBERNETES_SERVICE_PORT",
			setup: func(t *testing.T) {
				_ = os.Unsetenv("KUBERNETES_SERVICE_HOST")
				_ = os.Unsetenv("KUBERNETES_SERVICE_PORT")
			},
			wantErr:         true,
			wantErrContains: "KUBERNETES_SERVICE_HOST or KUBERNETES_SERVICE_PORT not set",
		},
		{
			name: "missing KUBERNETES_SERVICE_PORT",
			setup: func(t *testing.T) {
				setEnv(t, "KUBERNETES_SERVICE_HOST", "10.0.0.1")
				_ = os.Unsetenv("KUBERNETES_SERVICE_PORT")
			},
			wantErr:         true,
			wantErrContains: "KUBERNETES_SERVICE_HOST or KUBERNETES_SERVICE_PORT not set",
		},
		{
			name: "missing KUBERNETES_SERVICE_HOST",
			setup: func(t *testing.T) {
				setEnv(t, "KUBERNETES_SERVICE_PORT", "2302")
				_ = os.Unsetenv("KUBERNETES_SERVICE_HOST")
			},
			wantErr:         true,
			wantErrContains: "KUBERNETES_SERVICE_HOST or KUBERNETES_SERVICE_PORT not set",
		},
		{
			name: "token file missing",
			setup: func(t *testing.T) {
				setEnv(t, "KUBERNETES_SERVICE_HOST", "10.0.0.1")
				setEnv(t, "KUBERNETES_SERVICE_PORT", "443")
				dir := t.TempDir()
				cleanupFunc := overrideServiceAccountDir(dir)
				t.Cleanup(cleanupFunc)
			},
			wantErr:         true,
			wantErrContains: "reading service account token",
		},
		{
			name: "namespace file missing",
			setup: func(t *testing.T) {
				setEnv(t, "KUBERNETES_SERVICE_HOST", "10.0.0.1")
				setEnv(t, "KUBERNETES_SERVICE_PORT", "443")
				dir := t.TempDir()
				writeFile(t, dir, "token", []byte("mygreattoken"))
				cleanupFunc := overrideServiceAccountDir(dir)
				t.Cleanup(cleanupFunc)
			},
			wantErr:         true,
			wantErrContains: "reading namespace",
		},
		{
			// Token and namespace exist, but CA file does not.
			name: "CA file missing",
			setup: func(t *testing.T) {
				setEnv(t, "KUBERNETES_SERVICE_HOST", "10.0.0.1")
				setEnv(t, "KUBERNETES_SERVICE_PORT", "443")
				dir := t.TempDir()
				writeFile(t, dir, "token", []byte("mygreattoken"))
				writeFile(t, dir, "namespace", []byte("default"))
				cleanupFunc := overrideServiceAccountDir(dir)
				t.Cleanup(cleanupFunc)
			},
			wantErr:         true,
			wantErrContains: "reading CA certificate",
		},
		{
			name: "CA file invalid PEM",
			setup: func(t *testing.T) {
				setEnv(t, "KUBERNETES_SERVICE_HOST", "10.0.0.1")
				setEnv(t, "KUBERNETES_SERVICE_PORT", "443")
				dir := t.TempDir()
				writeFile(t, dir, "token", []byte("mygreattoken"))
				writeFile(t, dir, "namespace", []byte("default"))
				writeFile(t, dir, "ca.crt", []byte("not a valid PEM certificate"))
				cleanupFunc := overrideServiceAccountDir(dir)
				t.Cleanup(cleanupFunc)
			},
			wantErr:         true,
			wantErrContains: "failed to parse CA certificate",
		},
		{
			// All files present and valid, but POD_NAME is not set.
			name: "POD_NAME missing",
			setup: func(t *testing.T) {
				setEnv(t, "KUBERNETES_SERVICE_HOST", "10.0.0.1")
				setEnv(t, "KUBERNETES_SERVICE_PORT", "443")
				_ = os.Unsetenv("POD_NAME")
				dir := t.TempDir()
				writeFile(t, dir, "token", []byte("mygreattoken"))
				writeFile(t, dir, "namespace", []byte("default"))
				writeFile(t, dir, "ca.crt", validCA(t))
				cleanupFunc := overrideServiceAccountDir(dir)
				t.Cleanup(cleanupFunc)
			},
			wantErr:         true,
			wantErrContains: "POD_NAME env var not set",
		},
		{
			name: "valid configuration",
			setup: func(t *testing.T) {
				setEnv(t, "KUBERNETES_SERVICE_HOST", "10.0.0.1")
				setEnv(t, "KUBERNETES_SERVICE_PORT", "443")
				setEnv(t, "POD_NAME", "mygreatpod")
				dir := t.TempDir()
				writeFile(t, dir, "token", []byte("mygreattoken"))
				writeFile(t, dir, "namespace", []byte("mygreatns"))
				writeFile(t, dir, "ca.crt", validCA(t))
				cleanupFunc := overrideServiceAccountDir(dir)
				t.Cleanup(cleanupFunc)
			},
			wantErr: false,
			wantCfg: &inClusterConfig{
				host:      "https://10.0.0.1:443",
				token:     "mygreattoken",
				namespace: "mygreatns",
				podName:   "mygreatpod",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup(t)

			cfg, err := loadInClusterConfig()

			if tt.wantErr {
				if err == nil {
					t.Errorf("loadInClusterConfig() expected an error, got nil")
					return
				}
				if tt.wantErrContains != "" && !strings.Contains(err.Error(), tt.wantErrContains) {
					t.Errorf("loadInClusterConfig() error = %q, want it to contain %q", err.Error(), tt.wantErrContains)
				}
				return
			}

			if err != nil {
				t.Errorf("loadInClusterConfig() unexpected error: %v", err)
				return
			}
			// Check if the config is valid
			if cfg.host != tt.wantCfg.host {
				t.Errorf("loadInClusterConfig() host = %q, want %q", cfg.host, tt.wantCfg.host)
			}
			if cfg.token != tt.wantCfg.token {
				t.Errorf("loadInClusterConfig() token = %q, want %q", cfg.token, tt.wantCfg.token)
			}
			if cfg.namespace != tt.wantCfg.namespace {
				t.Errorf("loadInClusterConfig() namespace = %q, want %q", cfg.namespace, tt.wantCfg.namespace)
			}
			if cfg.podName != tt.wantCfg.podName {
				t.Errorf("loadInClusterConfig() podName = %q, want %q", cfg.podName, tt.wantCfg.podName)
			}
			if cfg.client == nil {
				t.Errorf("loadInClusterConfig() client is nil, want a configured http.Client")
			}
		})
	}
}

// overrideServiceAccountDir sets package variables to temporary locations and returns a restoration function
func overrideServiceAccountDir(dir string) func() {
	oldToken, oldNS, oldCA := tokenFile, namespaceFile, caFile
	tokenFile = filepath.Join(dir, "token")
	namespaceFile = filepath.Join(dir, "namespace")
	caFile = filepath.Join(dir, "ca.crt")
	return func() {
		tokenFile = oldToken
		namespaceFile = oldNS
		caFile = oldCA
	}
}

func TestValidateStatusAnnotations(t *testing.T) {
	tests := []struct {
		name            string
		annotationsData []byte
		wantErr         bool
		wantErrContains string
	}{
		{
			name:            "feature not configured - no when-active annotation",
			annotationsData: downwardAPILines(map[string]string{}),
		},
		{
			name: "no peer annotations to cross-check - only when-active present",
			annotationsData: downwardAPILines(map[string]string{
				annotationWhenActive: `{"productChargedContainers":"All","productID":"1234"}`,
			}),
		},
		{
			name: "identical key sets in when-active and when-replica",
			annotationsData: downwardAPILines(map[string]string{
				annotationWhenActive:  `{"a":"active-val","b":"active-val2"}`,
				annotationWhenReplica: `{"a":"replica-val","b":"replica-val2"}`,
			}),
		},
		{
			name: "identical key sets in when-active and when-standby",
			annotationsData: downwardAPILines(map[string]string{
				annotationWhenActive:  `{"a":"active-val","b":"active-val2"}`,
				annotationWhenStandby: `{"a":"standby-val","b":"standby-val2"}`,
			}),
		},
		{
			name: "both standby and replica annotations defined – error returned",
			annotationsData: downwardAPILines(map[string]string{
				annotationWhenActive:  `{"a":"1","b":"2"}`,
				annotationWhenReplica: `{"a":"x","b":"y"}`,
				annotationWhenStandby: `{"a":"p","b":"q"}`,
			}),
			wantErr:         true,
			wantErrContains: "both standby and replica annotations defined",
		},
		{
			name: "when-replica is missing a key that when-active has",
			annotationsData: downwardAPILines(map[string]string{
				annotationWhenActive:  `{"a":"1","b":"2"}`,
				annotationWhenReplica: `{"a":"x"}`,
			}),
			wantErr:         true,
			wantErrContains: "missing b",
		},
		{
			name: "when-replica has an extra key that when-active does not",
			annotationsData: downwardAPILines(map[string]string{
				annotationWhenActive:  `{"a":"1"}`,
				annotationWhenReplica: `{"a":"x","b":"y"}`,
			}),
			wantErr:         true,
			wantErrContains: "missing b",
		},
		{
			name: "when-standby is missing a key that when-active has",
			annotationsData: downwardAPILines(map[string]string{
				annotationWhenActive:  `{"a":"1","b":"2"}`,
				annotationWhenStandby: `{"a":"p"}`,
			}),
			wantErr:         true,
			wantErrContains: "missing b",
		},
		{
			name: "when-replica has an extra key even though when-standby matches",
			annotationsData: downwardAPILines(map[string]string{
				annotationWhenActive:  `{"a":"1"}`,
				annotationWhenReplica: `{"a":"x","b":"y"}`,
				annotationWhenStandby: `{"a":"p"}`,
			}),
			wantErr:         true,
			wantErrContains: "missing b",
		},
		{
			name: "when-active value is not valid JSON",
			annotationsData: downwardAPILines(map[string]string{
				annotationWhenActive:  `not-json`,
				annotationWhenReplica: `{"a":"x"}`,
			}),
			wantErr:         true,
			wantErrContains: "failed to parse annotation",
		},
		{
			name: "when-replica value is not valid JSON",
			annotationsData: downwardAPILines(map[string]string{
				annotationWhenActive:  `{"a":"1"}`,
				annotationWhenReplica: `not-json`,
			}),
			wantErr:         true,
			wantErrContains: "failed to parse annotation",
		},
		{
			name:            "annotations file is unreadable",
			annotationsData: nil,
			wantErr:         true,
			wantErrContains: "simulated file read error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Client{
				meta: metaReaderWith(tt.annotationsData, []byte{}),
			}

			err := c.ValidateStatusAnnotations()

			if tt.wantErr {
				if err == nil {
					t.Fatalf("ValidateStatusAnnotations() expected an error, got nil")
				}
				if tt.wantErrContains != "" && !strings.Contains(err.Error(), tt.wantErrContains) {
					t.Errorf("ValidateStatusAnnotations() error = %q, want it to contain %q",
						err.Error(), tt.wantErrContains)
				}
				return
			}
			if err != nil {
				t.Errorf("ValidateStatusAnnotations() unexpected error: %v", err)
			}
		})
	}
}
