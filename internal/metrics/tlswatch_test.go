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
	"context"
	"crypto/rsa"
	"crypto/x509"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	metricstest "github.com/ibm-messaging/mq-container/internal/metrics/test"
	"github.com/ibm-messaging/mq-container/pkg/logger"
)

func TestWatchDirectory(t *testing.T) {
	caCert, srvCerts, srvKeys, err := metricstest.GenerateTestKeys(2)
	if err != nil {
		t.Fatalf("Failed to generate test keys: %v", err)
	}

	updates := []string{}
	cbLock := sync.RWMutex{}
	callback := func(source string) error {
		cbLock.Lock()
		updates = append(updates, source)
		cbLock.Unlock()
		return nil
	}

	assertCalls := func(t *testing.T, expect []string, shutdown context.CancelFunc) {
		cbLock.RLock()
		defer cbLock.RUnlock()

		if len(updates) != len(expect) {
			shutdown()
			t.Fatalf("Watch calls do not match expectation:\n\tExpect:\t%v\n\tGot:\t%v\n", expect, updates)
		}
		for idx, exp := range expect {
			if updates[idx] != exp {
				shutdown()
				t.Fatalf("Watch calls do not match expectation:\n\tExpect:\t%v\n\tGot:\t%v\n", expect, updates)
			}
		}
	}

	clearCalls := func() {
		cbLock.Lock()
		updates = updates[0:0]
		cbLock.Unlock()
	}

	log, err := logger.NewLogger(os.Stdout, false, false, "test")
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	t.Run("Filesystem event trigger", func(t *testing.T) {
		certDir, err := os.MkdirTemp(os.TempDir(), "testCertMonitor_*")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(certDir)
		cm := newCertificateMonitor(context.Background(), certDir, log)
		err = cm.watch(callback)
		if err != nil {
			t.Fatalf("Failed to start watch: %v", err)
		}
		defer cm.shutdownFn()
		t.Run("Initial cert rollout", func(t *testing.T) {
			metricstest.WriteCertsToDir(caCert, srvCerts[0], srvKeys[0], certDir, false)

			// Event should only trigger after debounce time.
			// ... Check before (debounce/2) - expect no event
			time.Sleep(cm.debounceTime / 2)
			assertCalls(t, []string{}, cm.shutdownFn)
			// ... Check after (debounce/2 + debounce) - should have event now
			time.Sleep(cm.debounceTime)
			assertCalls(t, []string{sourceFilesystemEvent}, cm.shutdownFn)
			clearCalls()
		})

		t.Run("Certificate renewal", func(t *testing.T) {
			metricstest.WriteCertsToDir(caCert, srvCerts[1], srvKeys[1], certDir, false)

			// Recheck either side of debounce
			time.Sleep(cm.debounceTime / 2)
			assertCalls(t, []string{}, cm.shutdownFn)
			time.Sleep(cm.debounceTime)
			assertCalls(t, []string{sourceFilesystemEvent}, cm.shutdownFn)
			clearCalls()
		})
	})

	// Test polling fallback
	t.Run("Filesystem polling fallback", func(t *testing.T) {
		var validTiming bool
		shouldRetry := true
		for testDebounceTime := 1 * time.Millisecond; shouldRetry && testDebounceTime <= 10*time.Second && !validTiming; testDebounceTime = 2 * testDebounceTime {
			t.Run(fmt.Sprintf("%s debounce", testDebounceTime.String()), func(t *testing.T) {
				shouldRetry = false
				clearCalls()
				start := time.Now()
				assertTiming := func(maxAllowed time.Duration, shutdown context.CancelFunc) {
					endTime := time.Now()
					elapsedTime := endTime.Sub(start)
					if elapsedTime > maxAllowed {
						shutdown()
						time.Sleep(testDebounceTime)
						shouldRetry = true
						t.Skipf("Test has slept for too long to ensure validity; should sleep for less than %s but slept for %s", maxAllowed.String(), elapsedTime.String())
					}
				}

				t.Log("Create test directory")
				certDir, err := os.MkdirTemp(os.TempDir(), "testCertMonitor_*")
				if err != nil {
					t.Fatalf("Failed to create temp dir: %v", err)
				}
				testPollInterval := testDebounceTime / 10
				t.Logf("Create monitor (poll %s; debounce %s)", testPollInterval, testDebounceTime)
				cm := newCertificateMonitor(context.Background(), certDir, log)
				cm.debounceTime = testDebounceTime
				cm.pollInterval = testPollInterval
				t.Log("Write initial certs")
				metricstest.WriteCertsToDir(caCert, srvCerts[0], srvKeys[0], certDir, false)
				t.Log("Start watch")
				err = cm.watch(callback)
				if err != nil {
					t.Fatalf("Failed to start watch: %v", err)
				}

				// Expect only two triggers to occur (despite quick interval) in 2.5 * debounceTime
				t.Log("Wait for half debounce")
				time.Sleep(testDebounceTime / 2)
				assertTiming(testDebounceTime, cm.shutdownFn)
				assertCalls(t, []string{}, cm.shutdownFn)

				t.Log("Wait until after first debounce")
				time.Sleep(testDebounceTime)
				assertTiming(2*testDebounceTime, cm.shutdownFn)
				assertCalls(t, []string{sourcePoll}, cm.shutdownFn)

				// Do not expect a fsevent - the poll will already have triggered and be in debounce time
				metricstest.WriteCertsToDir(caCert, srvCerts[1], srvKeys[1], certDir, false)
				t.Log("Wait until after second debounce")
				time.Sleep(testDebounceTime)
				assertTiming(3*testDebounceTime, cm.shutdownFn)
				validTiming = true
				assertCalls(t, []string{sourcePoll, sourcePoll}, cm.shutdownFn)

				cm.shutdownFn()
				clearCalls()
			})
		}
		if !validTiming {
			t.Fatalf("Could not find valid debounce time for test to pass")
		}
	})
}

func TestUpdateCert(t *testing.T) {
	caCert, srvCerts, srvKeys, err := metricstest.GenerateTestKeys(2)
	if err != nil {
		t.Fatalf("Failed to generate test keys: %v", err)
	}
	_ = caCert
	tests := []struct {
		name         string
		combinedCert bool
		caCert       *x509.Certificate
		srvCert      *x509.Certificate
		srvKey       *rsa.PrivateKey
		expectError  bool
	}{
		{
			name:         "Single certificate (separate CA)",
			combinedCert: false,
			caCert:       caCert,
			srvCert:      srvCerts[0],
			srvKey:       srvKeys[0],
			expectError:  false,
		},
		{
			name:         "Single certificate (combined CA)",
			combinedCert: true,
			caCert:       caCert,
			srvCert:      srvCerts[0],
			srvKey:       srvKeys[0],
			expectError:  false,
		},
		{
			name:         "No CA",
			combinedCert: false,
			caCert:       nil,
			srvCert:      srvCerts[0],
			srvKey:       srvKeys[0],
			expectError:  false,
		},
		{
			name:         "Mismatched key pair",
			combinedCert: false,
			caCert:       caCert,
			srvCert:      srvCerts[0],
			srvKey:       srvKeys[1],
			expectError:  true,
		},
		{
			name:         "Missing cert",
			combinedCert: false,
			caCert:       caCert,
			srvCert:      nil,
			srvKey:       srvKeys[0],
			expectError:  true,
		},
		{
			name:         "Missing cert (but with CA in tls.crt)",
			combinedCert: true,
			caCert:       caCert,
			srvCert:      nil,
			srvKey:       srvKeys[0],
			expectError:  true,
		},
		{
			name:         "Missing key",
			combinedCert: false,
			caCert:       caCert,
			srvCert:      srvCerts[0],
			srvKey:       nil,
			expectError:  true,
		},
	}

	certDir, err := os.MkdirTemp(os.TempDir(), "testCertMonitor_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() {
		if err := os.RemoveAll(certDir); err != nil {
			t.Logf("Failed to remove certificate test directory: %v", err)
		}
	}()

	log, err := logger.NewLogger(os.Stdout, false, false, "test")
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	cm := newCertificateMonitor(context.Background(), certDir, log)
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			metricstest.WriteCertsToDir(test.caCert, test.srvCert, test.srvKey, certDir, test.combinedCert)

			err := cm.updateCert(fmt.Sprintf("test(%s)", test.name))
			if test.expectError && (err == nil) {
				t.Fatalf("Expected an error but got nil")
			}
			if !test.expectError && (err != nil) {
				t.Fatalf("Got unexpected error: %v", err)
			}
		})
	}
}
