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
package main

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"testing/synctest"

	"github.com/ibm-messaging/mq-container/internal/ready"
)

// fakeKubeClient satisfies the kubeClient interface for unit tests.
// It records every status passed to PatchPodMetadataIfNeeded so assertions
// can inspect what the server actually sent.
type fakeKubeClient struct {
	verifyErr      error            // returned by VerifyPermissions (when verifyFn is nil)
	verifyFn       func() error     // when non-nil, called instead of returning verifyErr
	annotationsErr error            // returned by ValidateStatusAnnotations (when annotationsFn is nil)
	annotationsFn  func() error     // when non-nil, called instead of returning annotationsErr
	patchErr       error            // returned by PatchPodMetadataIfNeeded
	patchFn        func()           // when non-nil, called after each PatchPodMetadataIfNeeded
	patched        []ready.QMStatus // records each status received
}

// VerifyPermissions delegates to verifyFn when set, otherwise returns verifyErr.
func (f *fakeKubeClient) VerifyPermissions(_ context.Context) error {
	if f.verifyFn != nil {
		return f.verifyFn()
	}
	return f.verifyErr
}

// ValidateStatusAnnotations delegates to annotationsFn when set, otherwise returns annotationsErr.
func (f *fakeKubeClient) ValidateStatusAnnotations() error {
	if f.annotationsFn != nil {
		return f.annotationsFn()
	}
	return f.annotationsErr
}

func (f *fakeKubeClient) PatchPodMetadataIfNeeded(_ context.Context, s ready.QMStatus) error {
	f.patched = append(f.patched, s)
	if f.patchFn != nil {
		f.patchFn()
	}
	return f.patchErr
}

// newTestServer creates a server with a fakeKubeClient already injected.
func newTestServer(t *testing.T) (*server, *fakeKubeClient) {
	t.Helper()
	testLog := newTestLogger(t)
	s := NewServer("/tmp/test-"+t.Name()+".sock", testLog)
	fake := &fakeKubeClient{}
	s.kubeClient = fake
	return s, fake
}

type testLogger struct {
	t *testing.T
}

func newTestLogger(t *testing.T) *testLogger {
	return &testLogger{
		t: t,
	}
}

func (tl testLogger) Println(args ...interface{}) {
	tl.t.Log(args...)
}

func (tl testLogger) Printf(format string, args ...interface{}) {
	tl.t.Logf(format, args...)
}

// post is a convenience wrapper that issues a single POST to handleQMStatus.
func post(s *server, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/qmStatus", strings.NewReader(body))
	rec := httptest.NewRecorder()
	s.handleQMStatus(rec, req)
	return rec
}

// TestHandleQMStatus covers all four cases for the handleQMStatus handler.
func TestHandleQMStatus(t *testing.T) {
	activeJSON := `{"status":1}`  // ready.StatusActiveQM
	standbyJSON := `{"status":2}` // ready.StatusStandbyQM
	unknownJSON := `{"status":0}` // ready.StatusUnknown

	wantActive := ready.StatusActiveQM
	wantStandby := ready.StatusStandbyQM

	tests := []struct {
		name string
		// setup drives the server and returns the recorder for the last request.
		setup          func(s *server) *httptest.ResponseRecorder
		wantStatusCode int
		// wantChanStatus, when non-nil, is the status expected to remain in statusCh.
		wantChanStatus *ready.QMStatus
	}{
		{
			name: "valid status — channel empty",
			setup: func(s *server) *httptest.ResponseRecorder {
				return post(s, activeJSON)
			},
			wantStatusCode: http.StatusAccepted,
			wantChanStatus: &wantActive,
		},
		{
			name: "valid status — channel full, newer status wins",
			setup: func(s *server) *httptest.ResponseRecorder {
				post(s, activeJSON)         // fills the channel; this status must be dropped
				return post(s, standbyJSON) // newer status replaces it
			},
			wantStatusCode: http.StatusAccepted,
			wantChanStatus: &wantStandby,
		},
		{
			name: "malformed JSON body",
			setup: func(s *server) *httptest.ResponseRecorder {
				return post(s, `not-json`)
			},
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name: "StatusUnknown rejected",
			setup: func(s *server) *httptest.ResponseRecorder {
				return post(s, unknownJSON)
			},
			wantStatusCode: http.StatusBadRequest,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s, _ := newTestServer(t)
			rec := tc.setup(s)

			if rec.Code != tc.wantStatusCode {
				t.Errorf("status code: got %d, want %d", rec.Code, tc.wantStatusCode)
			}

			if tc.wantChanStatus != nil {
				select {
				case got := <-s.statusCh:
					if got != *tc.wantChanStatus {
						t.Errorf("channel status: got %d, want %d", got, *tc.wantChanStatus)
					}
				default:
					t.Error("expected a status in the channel, but it was empty")
				}
			}
		})
	}
}

// TestHandleReady covers the two cases for the handleReady handler.
func TestHandleReady(t *testing.T) {
	tests := []struct {
		name           string
		setup          func(s *server)
		wantStatusCode int
	}{
		{
			name:           "not ready — zero value",
			setup:          nil, // apiserverReady is false after NewServer
			wantStatusCode: http.StatusServiceUnavailable,
		},
		{
			name: "ready",
			setup: func(s *server) {
				s.apiserverReady = true
			},
			wantStatusCode: http.StatusOK,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s, _ := newTestServer(t)

			if tc.setup != nil {
				tc.setup(s)
			}

			req := httptest.NewRequest(http.MethodGet, "/ready", nil)
			rec := httptest.NewRecorder()
			s.handleReady(rec, req)

			if rec.Code != tc.wantStatusCode {
				t.Errorf("status code: got %d, want %d", rec.Code, tc.wantStatusCode)
			}
		})
	}
}

// TestHandleLive covers the two cases for the handleLive handler.
// apiserverOK is initialised to true by NewServer, so the "OK" case needs no
// setup and the "not OK" case writes false directly — mirroring TestHandleReady.
func TestHandleLive(t *testing.T) {
	tests := []struct {
		name           string
		setup          func(s *server)
		wantStatusCode int
	}{
		{
			// NewServer sets apiserverOK = true, so no setup is needed.
			name:           "OK — default true",
			setup:          nil,
			wantStatusCode: http.StatusOK,
		},
		{
			name: "not OK",
			setup: func(s *server) {
				s.apiserverOK = false
			},
			wantStatusCode: http.StatusServiceUnavailable,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s, _ := newTestServer(t)

			if tc.setup != nil {
				tc.setup(s)
			}

			req := httptest.NewRequest(http.MethodGet, "/live", nil)
			rec := httptest.NewRecorder()
			s.handleLive(rec, req)

			if rec.Code != tc.wantStatusCode {
				t.Errorf("status code: got %d, want %d", rec.Code, tc.wantStatusCode)
			}
		})
	}
}

// TestRecordPatchResult covers all four cases for recordPatchResult.
func TestRecordPatchResult(t *testing.T) {
	tests := []struct {
		name            string
		calls           []bool // sequence of success (true) / failure (false) values
		wantFails       int
		wantApiserverOK bool
	}{
		{
			name:            "single success",
			calls:           []bool{true},
			wantFails:       0,
			wantApiserverOK: true,
		},
		{
			name:            "single failure",
			calls:           []bool{false},
			wantFails:       1,
			wantApiserverOK: true,
		},
		{
			// maxConsecutiveFails is 3; two failures must not trip the flag.
			name:            "failures below threshold",
			calls:           []bool{false, false},
			wantFails:       2,
			wantApiserverOK: true,
		},
		{
			// Three consecutive failures must trip the liveness flag.
			name:            "failures reach threshold",
			calls:           []bool{false, false, false},
			wantFails:       3,
			wantApiserverOK: false,
		},
		{
			// A success after tripping the flag must reset both fields.
			name:            "success resets after failures",
			calls:           []bool{false, false, false, true},
			wantFails:       0,
			wantApiserverOK: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s, _ := newTestServer(t)

			for _, success := range tc.calls {
				s.recordPatchResult(success)
			}

			gotFails := s.consecutiveFails
			gotOK := s.apiserverOK

			if gotFails != tc.wantFails {
				t.Errorf("consecutiveFails: got %d, want %d", gotFails, tc.wantFails)
			}
			if gotOK != tc.wantApiserverOK {
				t.Errorf("apiserverOK: got %v, want %v", gotOK, tc.wantApiserverOK)
			}
		})
	}
}

// TestRunPreflight covers the three cases for runPreflight.
//
// Each sub-test runs inside a synctest.Test bubble so that the
// time.After(preflightRetryInterval) inside runPreflight uses a fake clock
// and the tests complete instantly — no real sleeping required.
func TestRunPreflight(t *testing.T) {
	errPerm := errors.New("permissions denied")
	errAnnotations := errors.New("annotation keys do not match")

	tests := []struct {
		name string
		// setupFake configures the fake client before the goroutine is launched.
		setupFake func(fake *fakeKubeClient)
		// cancelCtx, when true, cancels the context after launching the goroutine
		// so that runPreflight exits via ctx.Done().
		cancelCtx bool
		wantReady bool
	}{
		{
			// VerifyPermissions returns nil immediately; runPreflight sets
			// apiserverReady and returns without ever sleeping.
			name: "immediate success",
			setupFake: func(fake *fakeKubeClient) {
				fake.verifyErr = nil
			},
			wantReady: true,
		},
		{
			// First call returns an error, second returns nil. runPreflight
			// sleeps for preflightRetryInterval between attempts; the synctest
			// fake clock advances instantly once all goroutines are blocked.
			name: "eventual success after one failure",
			setupFake: func(fake *fakeKubeClient) {
				badCalls := 2
				fake.verifyFn = func() error {
					if badCalls > 0 {
						badCalls--
						return errPerm
					}
					return nil
				}
			},
			wantReady: true,
		},
		{
			// VerifyPermissions always errors; the context is cancelled before
			// any retry succeeds. runPreflight exits via ctx.Done().
			name: "context cancelled before success",
			setupFake: func(fake *fakeKubeClient) {
				fake.verifyErr = errPerm
			},
			cancelCtx: true,
			wantReady: false,
		},
		{
			// ValidateStatusAnnotations always errors; the context is cancelled before
			// any retry succeeds. apiserverReady must remain false and
			// VerifyPermissions must never be called.
			name: "annotation error blocks ready",
			setupFake: func(fake *fakeKubeClient) {
				fake.annotationsErr = errAnnotations
				fake.verifyErr = nil // would succeed, but must not be reached
			},
			cancelCtx: true,
			wantReady: false,
		},
		{
			// ValidateStatusAnnotations fails on the first attempt then succeeds.
			// VerifyPermissions succeeds unconditionally. runPreflight should
			// set apiserverReady after one retry.
			name: "eventual success after annotation failure",
			setupFake: func(fake *fakeKubeClient) {
				badCalls := 2
				fake.annotationsFn = func() error {
					if badCalls > 0 {
						badCalls--
						return errAnnotations
					}
					return nil
				}
			},
			wantReady: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				s, fake := newTestServer(t)
				tc.setupFake(fake)

				// t.Context() inside a synctest bubble returns a context whose
				// Done() channel is bubble-associated, making the select inside
				// runPreflight durably blocking so fake time can advance.
				ctx, cancel := context.WithCancel(t.Context())
				defer cancel()

				// Use a wait group so that we can wait until runPreflight has completed for non error cases (It will exit).
				var wg sync.WaitGroup
				wg.Add(1)
				go func() {
					defer wg.Done()
					s.runPreflight(ctx)
				}()

				// Cancel context if the test requires it
				if tc.cancelCtx {
					cancel()
				}

				// Block until runPreflight has returned.
				wg.Wait()

				gotReady := s.apiserverReady

				if gotReady != tc.wantReady {
					t.Errorf("apiserverReady: got %v, want %v", gotReady, tc.wantReady)
				}
			})
		})
	}
}

func TestRunStatusWorker(t *testing.T) {
	errPatch := errors.New("patch failed")

	tests := []struct {
		name            string
		run             func(t *testing.T, s *server, fake *fakeKubeClient)
		wantPatched     int
		wantFails       int
		wantApiserverOK bool
	}{
		{
			name: "successful patch",
			run: func(t *testing.T, s *server, _ *fakeKubeClient) {
				s.statusCh <- ready.StatusActiveQM
			},
			wantPatched:     1,
			wantFails:       0,
			wantApiserverOK: true,
		},
		{
			name: "patch failure below threshold",
			run: func(t *testing.T, s *server, fake *fakeKubeClient) {
				fake.patchErr = errPatch
				s.statusCh <- ready.StatusActiveQM
				s.statusCh <- ready.StatusActiveQM
			},
			wantPatched:     2,
			wantFails:       2,
			wantApiserverOK: true,
		},
		{
			name: "patch failure reaches threshold",
			run: func(t *testing.T, s *server, fake *fakeKubeClient) {
				fake.patchErr = errPatch
				s.statusCh <- ready.StatusActiveQM
				s.statusCh <- ready.StatusActiveQM
				s.statusCh <- ready.StatusActiveQM
			},
			wantPatched:     3,
			wantFails:       3,
			wantApiserverOK: false,
		},
		{
			name:            "context cancellation exits cleanly",
			run:             func(t *testing.T, s *server, _ *fakeKubeClient) {},
			wantPatched:     0,
			wantFails:       0,
			wantApiserverOK: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s, fake := newTestServer(t)

			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()

			var processWG sync.WaitGroup
			processWG.Add(tc.wantPatched)
			fake.patchFn = func() { processWG.Done() }

			var serverWG sync.WaitGroup
			serverWG.Go(func() {
				s.runStatusWorker(ctx)
			})

			tc.run(t, s, fake)

			processWG.Wait()
			cancel()
			serverWG.Wait()

			if len(fake.patched) != tc.wantPatched {
				t.Errorf("patched count: got %d, want %d", len(fake.patched), tc.wantPatched)
			}
			if s.consecutiveFails != tc.wantFails {
				t.Errorf("consecutiveFails: got %d, want %d", s.consecutiveFails, tc.wantFails)
			}
			if s.apiserverOK != tc.wantApiserverOK {
				t.Errorf("apiserverOK: got %v, want %v", s.apiserverOK, tc.wantApiserverOK)
			}
		})
	}
}
