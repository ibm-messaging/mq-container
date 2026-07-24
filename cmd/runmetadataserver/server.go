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
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/ibm-messaging/mq-container/cmd/runmetadataserver/kubeapi"
	"github.com/ibm-messaging/mq-container/internal/metadata/api"
	"github.com/ibm-messaging/mq-container/internal/ready"
)

const (
	maxConsecutiveFails    = 3
	preflightRetryInterval = 5 * time.Second
)

type server struct {
	socketPath string
	apiMux     *http.ServeMux
	statusCh   chan ready.QMStatus
	kubeClient kubeClient
	log        serverLogger

	apiserverReady   bool // true once SelfSubjectReview has succeeded at startup
	apiserverOK      bool // false after maxConsecutiveFails consecutive PATCH failures
	consecutiveFails int

	apiserverHealthMu sync.RWMutex // guards apiserverReady, apiserverOK, consecutiveFails
	statusQueueMu     sync.Mutex   // guards the statusCh drain-and-replace in handleQMStatus
}

type kubeClient interface {
	PatchPodMetadataIfNeeded(ctx context.Context, status ready.QMStatus) error
	VerifyPermissions(ctx context.Context) error
	ValidateStatusAnnotations() error
}

func NewServer(socketPath string, log serverLogger) *server {
	s := &server{
		socketPath:  socketPath,
		statusCh:    make(chan ready.QMStatus, 1),
		apiserverOK: true, // optimistic until a PATCH fails
		kubeClient:  kubeapi.NewClient(),
		log:         log,
	}
	mux := http.NewServeMux()
	mux.HandleFunc("POST "+api.EndpointQMStatus, s.handleQMStatus)
	mux.HandleFunc("GET "+api.EndpointReady, s.handleReady)
	mux.HandleFunc("GET "+api.EndpointLive, s.handleLive)
	s.apiMux = mux
	return s
}

func (s *server) run(ctx context.Context) error {
	go s.runPreflight(ctx)
	go s.runStatusWorker(ctx)
	return s.serve(ctx)
}

func (s *server) serve(ctx context.Context) error {
	l, err := s.createSocket()
	if err != nil {
		return err
	}

	httpServer := &http.Server{
		Handler:           s.apiMux,
		BaseContext:       func(net.Listener) context.Context { return ctx },
		ReadHeaderTimeout: 5 * time.Second,
	}

	// #nosec G118 - context.Background() is intentional: the shutdown timeout
	// must outlive the cancelled ctx so that in-flight requests can drain.
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = httpServer.Shutdown(shutdownCtx)
	}()

	s.log.Println("Starting server...")
	if err := httpServer.Serve(l); !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

func (s *server) createSocket() (net.Listener, error) {
	if _, err := os.Stat(s.socketPath); err == nil {
		if err := os.Remove(s.socketPath); err != nil {
			return nil, err
		}
	}
	if err := os.MkdirAll(filepath.Dir(s.socketPath), 0700); err != nil {
		return nil, err
	}
	return net.Listen("unix", s.socketPath)
}

// handleQMStatus is called when the pod readiness probe is invoked in the main
// container. It returns an immediate acknowledgement; any calls to the
// Kubernetes apiserver are performed asynchronously so as not to interfere with
// readiness probe execution time.
func (s *server) handleQMStatus(w http.ResponseWriter, r *http.Request) {
	var msg api.SocketMessage
	if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if msg.Status <= ready.StatusUnknown || msg.Status >= ready.StatusMaxValue {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	// Non-blocking enqueue: drop any unconsumed status and replace with the
	// newer one. statusQueueMu ensures concurrent callers do not interleave.
	s.statusQueueMu.Lock()
	select {
	case <-s.statusCh:
	default:
	}
	s.statusCh <- msg.Status
	s.statusQueueMu.Unlock()
	w.WriteHeader(http.StatusAccepted)
}

// handleReady returns 200 once the initial SelfSubjectReview has confirmed
// apiserver access. It is used as the startup probe for the main QM container.
func (s *server) handleReady(w http.ResponseWriter, r *http.Request) {
	s.apiserverHealthMu.RLock()
	ready := s.apiserverReady
	s.apiserverHealthMu.RUnlock()
	if ready {
		w.WriteHeader(http.StatusOK)
		return
	}
	w.WriteHeader(http.StatusServiceUnavailable)
}

// handleLive returns 200 under normal operation. It returns 503 if recent
// calls to the Kubernetes apiserver have failed after retries.
func (s *server) handleLive(w http.ResponseWriter, r *http.Request) {
	s.apiserverHealthMu.RLock()
	ok := s.apiserverOK
	s.apiserverHealthMu.RUnlock()
	if ok {
		w.WriteHeader(http.StatusOK)
		return
	}
	w.WriteHeader(http.StatusServiceUnavailable)
}

// runStatusWorker drains the statusCh and applies a Kubernetes PATCH for each
// received QM status update. It updates apiserverOK based on consecutive failures.
func (s *server) runStatusWorker(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case status := <-s.statusCh:
			if err := s.kubeClient.PatchPodMetadataIfNeeded(ctx, status); err != nil {
				s.log.Printf("Failed to patch pod: %v", err)
				s.recordPatchResult(false)
			} else {
				s.recordPatchResult(true)
			}
		}
	}
}

// runPreflight calls SelfSubjectReview on startup to confirm apiserver access.
// It retries until it succeeds or the context is cancelled, then sets
// apiserverReady so that /ready starts returning 200.
func (s *server) runPreflight(ctx context.Context) {
	firstRun := true
	for {
		if !firstRun {
			select {
			case <-time.After(preflightRetryInterval):
			case <-ctx.Done():
				return
			}
		}
		firstRun = false

		if err := s.kubeClient.ValidateStatusAnnotations(); err != nil {
			s.log.Printf("Annotation validation failed (will retry): %v", err)
			continue
		}

		if err := s.kubeClient.VerifyPermissions(ctx); err != nil {
			s.log.Printf("Kubernetes permissions check failed (will retry): %v", err)
			continue
		}

		s.log.Println("Preflight checks completed successfully")
		s.apiserverHealthMu.Lock()
		s.apiserverReady = true
		s.apiserverHealthMu.Unlock()
		return
	}
}

// recordPatchResult updates apiserverOK based on the result of a PATCH call.
func (s *server) recordPatchResult(success bool) {
	s.apiserverHealthMu.Lock()
	defer s.apiserverHealthMu.Unlock()
	if success {
		s.consecutiveFails = 0
		s.apiserverOK = true
	} else {
		s.consecutiveFails++
		if s.consecutiveFails >= maxConsecutiveFails {
			s.apiserverOK = false
		}
	}
}
