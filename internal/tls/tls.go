/*
© Copyright IBM Corporation 2019

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
package tls

import (
	"bufio"
	"fmt"
	"io/ioutil"
	pwr "math/rand"
	"os"
	"path/filepath"
	"strings"
	"time"

	"crypto/rand"
	"crypto/sha512"
	"crypto/x509"
	"encoding/pem"

	pkcs "software.sslmate.com/src/go-pkcs12"

	"github.com/ibm-messaging/mq-container/internal/keystore"
	"github.com/ibm-messaging/mq-container/internal/mqtemplate"
	"github.com/ibm-messaging/mq-container/pkg/logger"
)

// cmsKeystoreName is the name of the CMS Keystore
const cmsKeystoreName = "key.kdb"

// p12TruststoreName is the name of the PKCS#12 Truststore
const p12TruststoreName = "trust.p12"

// keystoreDir is the location for the default CMS Keystore & PKCS#12 Truststore
const keystoreDirDefault = "/run/runmqserver/tls/"

// keystoreDirHA is the location for the HA CMS Keystore
const keystoreDirHA = "/run/runmqserver/ha/tls/"

// keyDir is the location of the keys to import
const keyDirDefault = "/etc/mqm/pki/keys"

// keyDir is the location of the HA keys to import
const keyDirHA = "/etc/mqm/ha/pki/keys"

// trustDir is the location of the trust certificates to import
const trustDirDefault = "/etc/mqm/pki/trust"

type KeyStoreData struct {
	Keystore          *keystore.KeyStore
	Password          string
	TrustedCerts      []*pem.Block
	KnownFingerPrints []string
	KeyLabels         []string
}

type P12KeyFiles struct {
	Keystores []string
	Password  string
}

type TLSStore struct {
	Keystore   KeyStoreData
	Truststore KeyStoreData
}

// ConfigureDefaultTLSKeystores configures the CMS Keystore & PKCS#12 Truststore
func ConfigureDefaultTLSKeystores() (string, KeyStoreData, KeyStoreData, error) {

	// Create the CMS Keystore & PKCS#12 Truststore
	tlsStore, err := generateAllDefaultKeystores()
	if err != nil {
		return "", tlsStore.Keystore, tlsStore.Truststore, err
	}

	// Process all keys - add them to the CMS KeyStore
	keyLabel, err := processKeys(&tlsStore, keystoreDirDefault, keyDirDefault)
	if err != nil {
		return "", tlsStore.Keystore, tlsStore.Truststore, err
	}

	// Process all trust certificates - add them to the CMS KeyStore & PKCS#12 Truststore
	err = processTrustCertificates(&tlsStore, trustDirDefault)
	if err != nil {
		return "", tlsStore.Keystore, tlsStore.Truststore, err
	}

	return keyLabel, tlsStore.Keystore, tlsStore.Truststore, err
}

// ConfigureHATLSKeystore configures the CMS Keystore & PKCS#12 Truststore
func ConfigureHATLSKeystore() (string, KeyStoreData, error) {

	// Create the CMS Keystore & PKCS#12 Truststore
	tlsStore, err := generateHAKeystore()
	if err != nil {
		return "", tlsStore.Keystore, err
	}

	// Process all keys - add them to the CMS KeyStore
	keyLabel, err := processKeys(&tlsStore, keystoreDirHA, keyDirHA)
	if err != nil {
		return "", tlsStore.Keystore, err
	}

	return keyLabel, tlsStore.Keystore, err
}

// ConfigureTLS configures TLS for the queue manager
func ConfigureTLS(keyLabel string, cmsKeystore KeyStoreData, devMode bool, log *logger.Logger) error {

	const mqsc string = "/etc/mqm/15-tls.mqsc"
	const mqscTemplate string = mqsc + ".tpl"

	err := mqtemplate.ProcessTemplateFile(mqscTemplate, mqsc, map[string]string{
		"SSLKeyR":          strings.TrimSuffix(cmsKeystore.Keystore.Filename, ".kdb"),
		"CertificateLabel": keyLabel,
	}, log)
	if err != nil {
		return err
	}

	if devMode && keyLabel != "" {
		err = configureTLSDev(log)
		if err != nil {
			return err
		}
	}

	return nil
}

// configureTLSDev configures TLS for the developer defaults
func configureTLSDev(log *logger.Logger) error {

	const mqsc string = "/etc/mqm/20-dev-tls.mqsc"
	const mqscTemplate string = mqsc + ".tpl"

	if os.Getenv("MQ_DEV") == "true" {
		err := mqtemplate.ProcessTemplateFile(mqscTemplate, mqsc, map[string]string{}, log)
		if err != nil {
			return err
		}
	} else {
		_, err := os.Stat(mqsc)
		if !os.IsNotExist(err) {
			err = os.Remove(mqsc)
			if err != nil {
				return fmt.Errorf("Failed to remove file %s: %v", mqsc, err)
			}
		}
	}

	return nil
}

// generateAllDefaultKeystores creates the CMS Keystore & PKCS#12 Truststore
func generateAllDefaultKeystores() (TLSStore, error) {

	var cmsKeystore, p12Truststore KeyStoreData

	// Generate a pasword for use with both the CMS Keystore & PKCS#12 Truststore
	pw := generateRandomPassword()
	cmsKeystore.Password = pw
	p12Truststore.Password = pw

	// Create the Keystore directory - if it does not already exist
	// #nosec G301 - write group permissions are required
	err := os.MkdirAll(keystoreDirDefault, 0770)
	if err != nil {
		return TLSStore{cmsKeystore, p12Truststore}, fmt.Errorf("Failed to create Keystore directory: %v", err)
	}

	// Create the CMS Keystore
	cmsKeystore.Keystore = keystore.NewCMSKeyStore(filepath.Join(keystoreDirDefault, cmsKeystoreName), cmsKeystore.Password)
	err = cmsKeystore.Keystore.Create()
	if err != nil {
		return TLSStore{cmsKeystore, p12Truststore}, fmt.Errorf("Failed to create CMS Keystore: %v", err)
	}

	// Create the PKCS#12 Truststore
	p12Truststore.Keystore = keystore.NewPKCS12KeyStore(filepath.Join(keystoreDirDefault, p12TruststoreName), p12Truststore.Password)
	err = p12Truststore.Keystore.Create()
	if err != nil {
		return TLSStore{cmsKeystore, p12Truststore}, fmt.Errorf("Failed to create PKCS#12 Truststore: %v", err)
	}
	return TLSStore{cmsKeystore, p12Truststore}, nil
}

// generateHAKeystore creates the CMS Keystore for Native HA replication
func generateHAKeystore() (TLSStore, error) {
	var cmsKeystore KeyStoreData

	// Generate a pasword for use with the CMS Keystore
	pw := generateRandomPassword()
	cmsKeystore.Password = pw

	// Create the Keystore directory - if it does not already exist
	// #nosec G301 - write group permissions are required
	err := os.MkdirAll(keystoreDirHA, 0770)
	if err != nil {
		return TLSStore{Keystore: cmsKeystore}, fmt.Errorf("Failed to create HA Keystore directory: %v", err)
	}

	// Create the CMS Keystore
	cmsKeystore.Keystore = keystore.NewCMSKeyStore(filepath.Join(keystoreDirHA, cmsKeystoreName), cmsKeystore.Password)
	err = cmsKeystore.Keystore.Create()
	if err != nil {
		return TLSStore{Keystore: cmsKeystore}, fmt.Errorf("Failed to create CMS Keystore: %v", err)
	}

	return TLSStore{Keystore: cmsKeystore}, nil
}

// processKeys processes all keys - adding them to the CMS KeyStore
func processKeys(tlsStore *TLSStore, keystoreDir string, keyDir string) (string, error) {

	// Key label - will be set to the label of the first set of keys
	keyLabel := ""

	// Process all keys
	keyList, err := ioutil.ReadDir(keyDir)
	if err == nil && len(keyList) > 0 {

		// Process each set of keys - each set should contain files: *.key & *.crt
		for _, keySet := range keyList {
			keys, _ := ioutil.ReadDir(filepath.Join(keyDir, keySet.Name()))

			// Ensure the label of the set of keys does not match the name of the PKCS#12 Truststore
			if keySet.Name() == p12TruststoreName[0:len(p12TruststoreName)-len(filepath.Ext(p12TruststoreName))] {
				return "", fmt.Errorf("Key label cannot be set to the Truststore name: %v", keySet.Name())
			}

			// Process private key (*.key)
			privateKey, keyPrefix, err := processPrivateKey(keyDir, keySet.Name(), keys)
			if err != nil {
				return "", err
			}

			// If private key does not exist - skip this set of keys
			if privateKey == nil {
				continue
			}

			// Process certificates (*.crt) - public certificate & optional CA certificate
			publicCertificate, caCertificate, err := processCertificates(keyDir, keySet.Name(), keyPrefix, keys, &tlsStore.Keystore, &tlsStore.Truststore)
			if err != nil {
				return "", err
			}

			// Create a new PKCS#12 Keystore - containing private key, public certificate & optional CA certificate
			file, err := pkcs.Encode(rand.Reader, privateKey, publicCertificate, caCertificate, tlsStore.Keystore.Password)
			if err != nil {
				return "", fmt.Errorf("Failed to encode PKCS#12 Keystore %s: %v", keySet.Name()+".p12", err)
			}
			err = ioutil.WriteFile(filepath.Join(keystoreDir, keySet.Name()+".p12"), file, 0644)
			if err != nil {
				return "", fmt.Errorf("Failed to write PKCS#12 Keystore %s: %v", filepath.Join(keystoreDir, keySet.Name()+".p12"), err)
			}

			// Import the new PKCS#12 Keystore into the CMS Keystore
			err = tlsStore.Keystore.Keystore.Import(filepath.Join(keystoreDir, keySet.Name()+".p12"), tlsStore.Keystore.Password)
			if err != nil {
				return "", fmt.Errorf("Failed tp import keys from %s into CMS Keystore: %v", filepath.Join(keystoreDir, keySet.Name()+".p12"), err)
			}

			// Relabel the certificate in the CMS Keystore
			err = relabelCertificate(keySet.Name(), &tlsStore.Keystore)
			if err != nil {
				return "", err
			}

			// Set key label - for first set of keys only
			if keyLabel == "" {
				keyLabel = keySet.Name()
			}
		}
	}

	return keyLabel, nil
}

// processTrustCertificates processes all trust certificates - adding them to the CMS KeyStore & PKCS#12 Truststore
func processTrustCertificates(tlsStore *TLSStore, trustDir string) error {

	// Process all trust certiifcates
	trustList, err := ioutil.ReadDir(trustDir)
	if err == nil && len(trustList) > 0 {

		// Process each set of keys
		for _, trustSet := range trustList {
			keys, _ := ioutil.ReadDir(filepath.Join(trustDir, trustSet.Name()))

			for _, key := range keys {
				if strings.HasSuffix(key.Name(), ".crt") {
					// #nosec G304 - filename variable is derived from contents of 'trustDir' which is a defined constant
					file, err := ioutil.ReadFile(filepath.Join(trustDir, trustSet.Name(), key.Name()))
					if err != nil {
						return fmt.Errorf("Failed to read file %s: %v", filepath.Join(trustDir, trustSet.Name(), key.Name()), err)
					}

					for string(file) != "" {
						var block *pem.Block
						block, file = pem.Decode(file)
						if block == nil {
							break
						}

						// Add to known certificates for the CMS Keystore
						err = addToKnownCertificates(block, &tlsStore.Keystore, true)
						if err != nil {
							return fmt.Errorf("Failed to add to know certificates for CMS Keystore")
						}

						// Add to known certificates for the PKCS#12 Truststore
						err = addToKnownCertificates(block, &tlsStore.Truststore, true)
						if err != nil {
							return fmt.Errorf("Failed to add to know certificates for PKCS#12 Truststore")
						}
					}
				}
			}
		}
	}

	// Add all trust certificates to PKCS#12 Truststore
	if len(tlsStore.Truststore.TrustedCerts) > 0 {
		err = addCertificatesToTruststore(&tlsStore.Truststore)
		if err != nil {
			return err
		}
	}

	// Add all trust certificates to CMS Keystore
	if len(tlsStore.Keystore.TrustedCerts) > 0 {
		err = addCertificatesToCMSKeystore(&tlsStore.Keystore)
		if err != nil {
			return err
		}
	}

	return nil
}

// processPrivateKey processes the private key (*.key) from a set of keys
func processPrivateKey(keyDir string, keySetName string, keys []os.FileInfo) (interface{}, string, error) {

	var privateKey interface{}
	keyPrefix := ""

	for _, key := range keys {

		if strings.HasSuffix(key.Name(), ".key") {
			// #nosec G304 - filename variable is derived from contents of 'keyDir' which is a defined constant
			file, err := ioutil.ReadFile(filepath.Join(keyDir, keySetName, key.Name()))
			if err != nil {
				return nil, "", fmt.Errorf("Failed to read private key %s: %v", filepath.Join(keyDir, keySetName, key.Name()), err)
			}
			block, _ := pem.Decode(file)
			if block == nil {
				return nil, "", fmt.Errorf("Failed to decode private key %s: pem.Decode returned nil", filepath.Join(keyDir, keySetName, key.Name()))
			}

			// Check if the private key is PKCS1
			privateKey, err = x509.ParsePKCS1PrivateKey(block.Bytes)
			if err != nil {
				// Check if the private key is PKCS8
				privateKey, err = x509.ParsePKCS8PrivateKey(block.Bytes)
				if err != nil {
					return nil, "", fmt.Errorf("Failed to parse private key %s: %v", filepath.Join(keyDir, keySetName, key.Name()), err)
				}
			}
			keyPrefix = key.Name()[0 : len(key.Name())-len(filepath.Ext(key.Name()))]
		}
	}

	return privateKey, keyPrefix, nil
}

// processCertificates processes the certificates (*.crt) from a set of keys
func processCertificates(keyDir string, keySetName, keyPrefix string, keys []os.FileInfo, cmsKeystore, p12Truststore *KeyStoreData) (*x509.Certificate, []*x509.Certificate, error) {

	var publicCertificate *x509.Certificate
	var caCertificate []*x509.Certificate

	for _, key := range keys {

		if strings.HasPrefix(key.Name(), keyPrefix) && strings.HasSuffix(key.Name(), ".crt") {
			// #nosec G304 - filename variable is derived from contents of 'keyDir' which is a defined constant
			file, err := ioutil.ReadFile(filepath.Join(keyDir, keySetName, key.Name()))
			if err != nil {
				return nil, nil, fmt.Errorf("Failed to read public certificate %s: %v", filepath.Join(keyDir, keySetName, key.Name()), err)
			}
			block, _ := pem.Decode(file)
			if block == nil {
				return nil, nil, fmt.Errorf("Failed to decode public certificate %s: pem.Decode returned nil", filepath.Join(keyDir, keySetName, key.Name()))
			}
			publicCertificate, err = x509.ParseCertificate(block.Bytes)
			if err != nil {
				return nil, nil, fmt.Errorf("Failed to parse public certificate %s: %v", filepath.Join(keyDir, keySetName, key.Name()), err)
			}

			// Add to known certificates for the CMS Keystore
			err = addToKnownCertificates(block, cmsKeystore, false)
			if err != nil {
				return nil, nil, fmt.Errorf("Failed to add to know certificates for CMS Keystore")
			}

		} else if strings.HasSuffix(key.Name(), ".crt") {
			// #nosec G304 - filename variable is derived from contents of 'keyDir' which is a defined constant
			file, err := ioutil.ReadFile(filepath.Join(keyDir, keySetName, key.Name()))
			if err != nil {
				return nil, nil, fmt.Errorf("Failed to read CA certificate %s: %v", filepath.Join(keyDir, keySetName, key.Name()), err)
			}

			for string(file) != "" {
				var block *pem.Block
				block, file = pem.Decode(file)
				if block == nil {
					break
				}

				// Add to known certificates for the CMS Keystore
				err = addToKnownCertificates(block, cmsKeystore, false)
				if err != nil {
					return nil, nil, fmt.Errorf("Failed to add to know certificates for CMS Keystore")
				}

				if p12Truststore != nil {
					// Add to known certificates for the PKCS#12 Truststore
					err = addToKnownCertificates(block, p12Truststore, true)
					if err != nil {
						return nil, nil, fmt.Errorf("Failed to add to know certificates for PKCS#12 Truststore")
					}
				}

				certificate, err := x509.ParseCertificate(block.Bytes)
				if err != nil {
					return nil, nil, fmt.Errorf("Failed to parse CA certificate %s: %v", filepath.Join(keyDir, keySetName, key.Name()), err)
				}
				caCertificate = append(caCertificate, certificate)
			}
		}
	}

	return publicCertificate, caCertificate, nil
}

// relabelCertificate sets a new label for a certificate in the CMS Keystore
func relabelCertificate(newLabel string, cmsKeystore *KeyStoreData) error {

	allLabels, err := cmsKeystore.Keystore.GetCertificateLabels()
	if err != nil {
		return fmt.Errorf("Failed to get list of all certificate labels from CMS Keystore: %v", err)
	}
	relabelled := false
	for _, label := range allLabels {
		found := false
		for _, keyLabel := range cmsKeystore.KeyLabels {
			if strings.Trim(label, "\"") == keyLabel {
				found = true
				break
			}
		}
		if !found {
			err = cmsKeystore.Keystore.RenameCertificate(strings.Trim(label, "\""), newLabel)
			if err != nil {
				return err
			}
			relabelled = true
			cmsKeystore.KeyLabels = append(cmsKeystore.KeyLabels, newLabel)
			break
		}
	}

	if !relabelled {
		return fmt.Errorf("Failed to relabel certificate for %s in CMS keystore", newLabel)
	}

	return nil
}

// addCertificatesToTruststore adds trust certificates to the PKCS#12 Truststore
func addCertificatesToTruststore(p12Truststore *KeyStoreData) error {

	temporaryPemFile := filepath.Join("/tmp", "trust.pem")
	_, err := os.Stat(temporaryPemFile)
	if err == nil {
		err = os.Remove(temporaryPemFile)
		if err != nil {
			return fmt.Errorf("Failed to remove file %v: %v", temporaryPemFile, err)
		}
	}

	err = writeCertificatesToFile(temporaryPemFile, p12Truststore.TrustedCerts)
	if err != nil {
		return err
	}

	err = p12Truststore.Keystore.AddNoLabel(temporaryPemFile)
	if err != nil {
		return fmt.Errorf("Failed to add certificates to PKCS#12 Truststore: %v", err)
	}

	// Relabel all certiifcates
	allCertificates, err := p12Truststore.Keystore.ListAllCertificates()
	if err != nil || len(allCertificates) <= 0 {
		return fmt.Errorf("Failed to get any certificates from PKCS#12 Truststore: %v", err)
	}

	for i, certificate := range allCertificates {
		certificate = strings.Trim(certificate, "\"")
		certificate = strings.TrimSpace(certificate)
		newLabel := fmt.Sprintf("Trust%d", i)

		err = p12Truststore.Keystore.RenameCertificate(certificate, newLabel)
		if err != nil || len(allCertificates) <= 0 {
			return fmt.Errorf("Failed to rename certificate %s to %s in PKCS#12 Truststore: %v", certificate, newLabel, err)
		}
	}

	return nil
}

// addCertificatesToCMSKeystore adds trust certificates to the CMS keystore
func addCertificatesToCMSKeystore(cmsKeystore *KeyStoreData) error {

	temporaryPemFile := filepath.Join("/tmp", "cmsTrust.pem")
	_, err := os.Stat(temporaryPemFile)
	if err == nil {
		err = os.Remove(temporaryPemFile)
		if err != nil {
			return fmt.Errorf("Failed to remove file %v: %v", temporaryPemFile, err)
		}
	}

	err = writeCertificatesToFile(temporaryPemFile, cmsKeystore.TrustedCerts)
	if err != nil {
		return err
	}

	err = cmsKeystore.Keystore.AddNoLabel(temporaryPemFile)
	if err != nil {
		return fmt.Errorf("Failed to add certificates to CMS keystore: %v", err)
	}

	return nil
}

// generateRandomPassword generates a random 12 character password from the characters a-z, A-Z, 0-9
func generateRandomPassword() string {
	pwr.Seed(time.Now().Unix())
	validChars := "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"
	validcharArray := []byte(validChars)
	password := ""
	for i := 0; i < 12; i++ {
		password = password + string(validcharArray[pwr.Intn(len(validcharArray))])
	}

	return password
}

// addToKnownCertificates adds to the list of known certificates for a Keystore
func addToKnownCertificates(block *pem.Block, keyData *KeyStoreData, addToKeystore bool) error {
	sha512str, err := getCertificateFingerprint(block)
	if err != nil {
		return err
	}
	known := false
	for _, fingerprint := range keyData.KnownFingerPrints {
		if fingerprint == sha512str {
			known = true
			break
		}
	}

	if !known {
		if addToKeystore {
			keyData.TrustedCerts = append(keyData.TrustedCerts, block)
		}
		keyData.KnownFingerPrints = append(keyData.KnownFingerPrints, sha512str)
	}

	return nil
}

// getCertificateFingerprint returns a fingerprint for a certificate
func getCertificateFingerprint(block *pem.Block) (string, error) {
	certificate, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return "", fmt.Errorf("Failed to parse x509 certificate: %v", err)
	}
	sha512Sum := sha512.Sum512(certificate.Raw)
	sha512str := string(sha512Sum[:])

	return sha512str, nil
}

// writeCertificatesToFile writes a list of certificates to a file
func writeCertificatesToFile(file string, certificates []*pem.Block) error {
	f, err := os.Create(file)
	if err != nil {
		return fmt.Errorf("Failed to create file %s: %v", file, err)
	}
	defer f.Close()

	w := bufio.NewWriter(f)

	for i, c := range certificates {
		err := pem.Encode(w, c)
		if err != nil {
			return fmt.Errorf("Failed to encode certificate number %d: %v", i, err)
		}
		err = w.Flush()
		if err != nil {
			return fmt.Errorf("Failed to write certificate to file %s: %v", file, err)
		}
	}
	return nil
}
