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
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	mathrand "math/rand"
	"os"
	"path"
	"sync"
	"testing"
	"time"

	"github.com/ibm-messaging/mq-container/pkg/logger"
)

func TestWatchDirectory(t *testing.T) {
	caCert, srvCerts, srvKeys, err := generateTestKeys(2)
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

	assertCalls := func(t *testing.T, expect []string) {
		cbLock.RLock()
		defer cbLock.RUnlock()

		if len(updates) != len(expect) {
			t.Fatalf("Watch calls do not match expectation:\n\tExpect:\t%v\n\tGot:\t%v\n", expect, updates)
		}
		for idx, exp := range expect {
			if updates[idx] != exp {
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
		cm := newCertificateMonitor(context.Background(), certDir, log)
		err = cm.watch(callback)
		if err != nil {
			t.Fatalf("Failed to start watch: %v", err)
		}
		defer cm.shutdownFn()
		t.Run("Initial cert rollout", func(t *testing.T) {
			writeCertsToDir(caCert, srvCerts[0], srvKeys[0], certDir, false)

			// Event should only trigger after debounce time.
			// ... Check before (debounce/2) - expect no event
			time.Sleep(debounceTime / 2)
			assertCalls(t, []string{})
			// ... Check after (debounce/2 + debounce) - should have event now
			time.Sleep(debounceTime)
			assertCalls(t, []string{sourceFilesystemEvent})
			clearCalls()
		})

		t.Run("Certificate renewal", func(t *testing.T) {
			writeCertsToDir(caCert, srvCerts[1], srvKeys[1], certDir, false)

			// Recheck either side of debounce
			time.Sleep(debounceTime / 2)
			assertCalls(t, []string{})
			time.Sleep(debounceTime)
			assertCalls(t, []string{sourceFilesystemEvent})
			clearCalls()
		})
	})

	// Test polling fallback
	t.Run("Filesystem event trigger", func(t *testing.T) {
		certDir, err := os.MkdirTemp(os.TempDir(), "testCertMonitor_*")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		cm := newCertificateMonitor(context.Background(), certDir, log)
		testPollInterval := 10 * time.Millisecond
		cm.pollInterval = testPollInterval
		writeCertsToDir(caCert, srvCerts[0], srvKeys[0], certDir, false)
		err = cm.watch(callback)
		if err != nil {
			t.Fatalf("Failed to start watch: %v", err)
		}
		defer cm.shutdownFn()

		// Expect only two triggers to occur (despite quick interval) in 2.5 * debounceTime
		time.Sleep(debounceTime / 2)
		assertCalls(t, []string{})
		time.Sleep(debounceTime)
		assertCalls(t, []string{sourcePoll})
		// Do not expect a fsevent - the poll will already have triggered and be in debounce time
		writeCertsToDir(caCert, srvCerts[1], srvKeys[1], certDir, false)
		time.Sleep(debounceTime)
		assertCalls(t, []string{sourcePoll, sourcePoll})
		clearCalls()
	})
}

func TestUpdateCert(t *testing.T) {
	caCert, srvCerts, srvKeys, err := generateTestKeys(2)
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
			writeCertsToDir(test.caCert, test.srvCert, test.srvKey, certDir, test.combinedCert)

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

func generateTestKeys(numServerKeys int) (*x509.Certificate, []*x509.Certificate, []*rsa.PrivateKey, error) {
	caKey, caCert, err := makeTestCA()
	if err != nil {
		return nil, nil, nil, err
	}

	srvCerts := make([]*x509.Certificate, 0, numServerKeys)
	srvKeys := make([]*rsa.PrivateKey, 0, numServerKeys)

	for i := 0; i < numServerKeys; i++ {
		srvKey, srvCert, err := makeTestCert(caCert, caKey, fmt.Sprintf("test-cert-%d", numServerKeys))
		if err != nil {
			return nil, nil, nil, err
		}
		srvCerts = append(srvCerts, srvCert)
		srvKeys = append(srvKeys, srvKey)
	}

	return caCert, srvCerts, srvKeys, nil
}

func writeCertsToDir(caCert, srvCert *x509.Certificate, srvKey *rsa.PrivateKey, certDir string, combineCert bool) error {
	var caCertPEM, srvCertPEM, srvKeyPEM []byte
	caCertFile := path.Join(certDir, "ca.crt")
	srvCertFile := path.Join(certDir, "tls.crt")
	srvKeyFile := path.Join(certDir, "tls.key")

	if caCert != nil {
		caCertPEM = pem.EncodeToMemory(&pem.Block{
			Type:  "CERTIFICATE",
			Bytes: caCert.Raw,
		})
	} else {
		_ = os.Remove(caCertFile)
	}

	if srvCert != nil {
		srvCertPEM = pem.EncodeToMemory(&pem.Block{
			Type:  "CERTIFICATE",
			Bytes: srvCert.Raw,
		})
	} else {
		_ = os.Remove(srvCertFile)
	}
	if srvKey != nil {
		srvKeyPEM = pem.EncodeToMemory(&pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: x509.MarshalPKCS1PrivateKey(srvKey),
		})
	} else {
		_ = os.Remove(srvKeyFile)
	}

	if combineCert {
		_ = os.Remove(caCertFile)
		combined := make([]byte, 0, len(caCertPEM)+len(srvCertPEM)+1)
		combined = append(combined, srvCertPEM...)
		if len(combined) > 0 && combined[len(combined)-1] != '\n' {
			combined = append(combined, '\n')
		}
		combined = append(combined, caCertPEM...)
		err := os.WriteFile(srvCertFile, combined, 0644)
		if err != nil {
			return err
		}
	} else {
		err := os.WriteFile(caCertFile, caCertPEM, 0644)
		if err != nil {
			return err
		}
		err = os.WriteFile(srvCertFile, srvCertPEM, 0644)
		if err != nil {
			return err
		}
	}
	err := os.WriteFile(srvKeyFile, srvKeyPEM, 0644)
	if err != nil {
		return err
	}
	return nil
}

func makeTestCA() (*rsa.PrivateKey, *x509.Certificate, error) {
	ca := &x509.Certificate{
		SerialNumber: big.NewInt(mathrand.Int63()), // Non-crypto rand acceptable for serial number
		Subject: pkix.Name{
			Organization:  []string{"IBM"},
			StreetAddress: []string{"1 New Orchard Road"},
			Locality:      []string{"Armonk"},
			Province:      []string{"New York"},
			PostalCode:    []string{"10504"},
			Country:       []string{"US"},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(10, 0, 0),
		IsCA:                  true,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
	}
	caPrivKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, err
	}
	ca.PublicKey = &caPrivKey.PublicKey
	caBytes, err := x509.CreateCertificate(rand.Reader, ca, ca, &caPrivKey.PublicKey, caPrivKey)
	if err != nil {
		return nil, nil, err
	}
	ca.Raw = caBytes
	return caPrivKey, ca, nil
}

func makeTestCert(caCert *x509.Certificate, caKey *rsa.PrivateKey, commonName string) (*rsa.PrivateKey, *x509.Certificate, error) {
	cert := &x509.Certificate{
		SerialNumber: big.NewInt(mathrand.Int63()), // Non-crypto rand acceptable for serial number
		Subject: pkix.Name{
			Organization:  []string{"IBM"},
			StreetAddress: []string{"1 New Orchard Road"},
			Locality:      []string{"Armonk"},
			Province:      []string{"New York"},
			PostalCode:    []string{"10504"},
			Country:       []string{"US"},
			CommonName:    commonName,
		},
		NotBefore:   time.Now(),
		NotAfter:    time.Now().AddDate(10, 0, 0),
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:    x509.KeyUsageDigitalSignature,
	}
	certPrivKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, err
	}
	cert.PublicKey = &certPrivKey.PublicKey
	certBytes, err := x509.CreateCertificate(rand.Reader, cert, caCert, &certPrivKey.PublicKey, caKey)
	if err != nil {
		return nil, nil, err
	}
	cert.Raw = certBytes
	return certPrivKey, cert, nil
}
