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

package metricstest

import (
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
	"time"
)

func GenerateTestKeys(numServerKeys int, serverCommonNames ...string) (*x509.Certificate, []*x509.Certificate, []*rsa.PrivateKey, error) {
	caKey, caCert, err := makeTestCA()
	if err != nil {
		return nil, nil, nil, err
	}

	srvCerts := make([]*x509.Certificate, 0, numServerKeys)
	srvKeys := make([]*rsa.PrivateKey, 0, numServerKeys)

	if len(serverCommonNames) == 0 {
		serverCommonNames = []string{fmt.Sprintf("test-cert-%d", numServerKeys)}
	}

	for i := 0; i < numServerKeys; i++ {
		srvKey, srvCert, err := makeTestCert(caCert, caKey, serverCommonNames...)
		if err != nil {
			return nil, nil, nil, err
		}
		srvCerts = append(srvCerts, srvCert)
		srvKeys = append(srvKeys, srvKey)
	}

	return caCert, srvCerts, srvKeys, nil
}

func MakeCACertPool(caCert *x509.Certificate) *x509.CertPool {
	caPool := x509.NewCertPool()
	caPool.AppendCertsFromPEM(pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: caCert.Raw,
	}))
	return caPool
}

func WriteCertsToDir(caCert, srvCert *x509.Certificate, srvKey *rsa.PrivateKey, certDir string, combineCert bool) error {
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
		err := os.WriteFile(srvCertFile, combined, 0644) // #nosec G306 -- Test-only Certificate needs to be readable by multiple users
		if err != nil {
			return err
		}
	} else {
		err := os.WriteFile(caCertFile, caCertPEM, 0644) // #nosec G306 -- Test-only Certificate needs to be readable by multiple users
		if err != nil {
			return err
		}
		err = os.WriteFile(srvCertFile, srvCertPEM, 0644) // #nosec G306 -- Test-only Certificate needs to be readable by multiple users
		if err != nil {
			return err
		}
	}
	err := os.WriteFile(srvKeyFile, srvKeyPEM, 0644) // #nosec G306 -- Test-only Certificate needs to be readable by multiple users
	if err != nil {
		return err
	}
	return nil
}

func makeTestCA() (*rsa.PrivateKey, *x509.Certificate, error) {
	ca := &x509.Certificate{
		// #nosec G404 -- Non crypto rand acceptable for serial number
		SerialNumber: big.NewInt(mathrand.Int63()),
		Subject: pkix.Name{
			Organization:  []string{"IBM"},
			StreetAddress: []string{"1 New Orchard Road"},
			Locality:      []string{"Armonk"},
			Province:      []string{"New York"},
			PostalCode:    []string{"10504"},
			Country:       []string{"US"},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(0, 0, 1),
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

func makeTestCert(caCert *x509.Certificate, caKey *rsa.PrivateKey, serverCommonNames ...string) (*rsa.PrivateKey, *x509.Certificate, error) {
	cert := &x509.Certificate{
		// #nosec G404 -- Noncrypto rand acceptable for serial number
		SerialNumber: big.NewInt(mathrand.Int63()),
		Subject: pkix.Name{
			Organization:  []string{"IBM"},
			StreetAddress: []string{"1 New Orchard Road"},
			Locality:      []string{"Armonk"},
			Province:      []string{"New York"},
			PostalCode:    []string{"10504"},
			Country:       []string{"US"},
			CommonName:    serverCommonNames[0],
		},
		DNSNames:    serverCommonNames,
		NotBefore:   time.Now(),
		NotAfter:    time.Now().AddDate(0, 0, 1),
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
