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
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"

	"github.com/ibm-messaging/mq-container/pkg/logger"
)

const (
	defaultCertificatePollInverval = 15 * time.Minute
	defaultDebounceTime            = 1 * time.Second
	sourceStartup                  = "startup"
	sourceFilesystemEvent          = "fsevent"
	sourcePoll                     = "poll"
)

type certificateMonitor struct {
	certDir    string
	cert       *tls.Certificate
	certSerial string

	log *logger.Logger

	ctx          context.Context
	shutdownFn   context.CancelFunc
	certLock     sync.RWMutex
	pollInterval time.Duration
	debounceTime time.Duration
}

func loadAndWatchCertificates(ctx context.Context, certDir string, log *logger.Logger) (*certificateMonitor, error) {
	cm := newCertificateMonitor(ctx, certDir, log)

	err := cm.updateCert(sourceStartup)
	if err != nil {
		cm.shutdownFn()
		return nil, err
	}

	err = cm.watch(cm.updateCert)
	if err != nil {
		cm.shutdownFn()
		return nil, err
	}

	return cm, nil
}

func newCertificateMonitor(ctx context.Context, certDir string, log *logger.Logger) *certificateMonitor {
	watchCtx, stopWatch := context.WithCancel(ctx)
	return &certificateMonitor{
		ctx:          watchCtx,
		shutdownFn:   stopWatch,
		certDir:      certDir,
		log:          log,
		pollInterval: defaultCertificatePollInverval,
		debounceTime: defaultDebounceTime,
	}
}

// latestCert will return the last successfully loaded certificate
func (cm *certificateMonitor) latestCert() *tls.Certificate {
	cm.certLock.RLock()
	defer cm.certLock.RUnlock()
	return cm.cert
}

// watch will trigger a callback whenever certificates update on disk.
// Detection of changes can be triggered by a filesystem event or a slower poll (to catch missed events).
// All detections are "debounced" to prevent filesystem changes causing rapid reload of potentially inconsistent certificates as a rotation occurs.
func (cm *certificateMonitor) watch(callback updateFn) error {
	ticker := time.NewTicker(cm.pollInterval)
	fsWatcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("failed to set up fsnotify: %w", err)
	}
	err = fsWatcher.Add(cm.certDir)
	if err != nil {
		return fmt.Errorf("failed to watch certificate directory: %w", err)
	}

	triggerUpdateQueue := make(chan string)
	triggerUpdate := func(source string) {
		// Do not block pushing event - ignores events triggered during previous ongoing trigger
		select {
		case triggerUpdateQueue <- source:
		default:
		}
	}

	go func() {
		for {
			select {
			case <-cm.ctx.Done():
				return
			case source := <-triggerUpdateQueue:
				time.Sleep(cm.debounceTime)
				err := callback(source)
				if err != nil {
					cm.log.Errorf("error loading updated certificate pair for metrics server: %v", err)
					continue
				}
			}
		}
	}()

	go func() {
		// Trigger updates on filesystem event or poll timer
		defer ticker.Stop()
		for {
			select {
			case <-fsWatcher.Events:
				triggerUpdate(sourceFilesystemEvent)
			case <-ticker.C:
				triggerUpdate(sourcePoll)
			case <-cm.ctx.Done():
				_ = fsWatcher.Close()
				return
			}
		}
	}()
	return nil
}

// updateCert loads the latest certificates from disk.
// If the load fails for any reason, the previously loaded certificates will not be overwritten.
func (cm *certificateMonitor) updateCert(updateTrigger string) error {
	keyFile := cm.certDir + "/tls.key"
	certFiles := []string{
		cm.certDir + "/tls.crt",
		cm.certDir + "/ca.crt",
	}

	// Load certificate and append CA certificate if it exists
	certPEM := []byte{}
	for _, certFile := range certFiles {
		cert, err := os.ReadFile(filepath.Clean(certFile))
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return fmt.Errorf("failed to read cert file (%s): %w", certFile, err)
		}
		if len(certPEM) > 0 && certPEM[len(certPEM)-1] != '\n' {
			cert = append(cert, '\n')
		}
		certPEM = append(certPEM, cert...)
	}

	// Load key
	keyPEM, err := os.ReadFile(filepath.Clean(keyFile))
	if err != nil {
		return fmt.Errorf("failed to read key file (%s): %w", keyFile, err)
	}

	newCertPair, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		cm.log.Errorf("error loading key pair: %v", err)
		return err
	}
	leaf, err := x509.ParseCertificate(newCertPair.Certificate[0])
	if err != nil {
		cm.log.Errorf("error reading certificate serial: %v", err)
		return err
	}
	serial := leaf.SerialNumber.Text(16)

	// Switch to the newly loaded certificate for future requests
	updated := false
	cm.certLock.Lock()
	if serial != cm.certSerial {
		cm.certSerial = serial
		updated = true
	}
	cm.cert = &newCertPair
	cm.certLock.Unlock()
	if updated {
		log.Printf("HTTPS metrics TLS certificate reload triggered by %s, Loaded new certificate (serial: %s)", updateTrigger, serial)
	}
	return nil
}

func (cm *certificateMonitor) stop() {
	if cm != nil && cm.shutdownFn != nil {
		cm.shutdownFn()
	}
}

type updateFn func(source string) error
