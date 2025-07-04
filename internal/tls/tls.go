/*
© Copyright IBM Corporation 2019, 2024

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
	"crypto"
	"fmt"
	pwr "math/rand"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"crypto/sha512"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"

	pkcs "software.sslmate.com/src/go-pkcs12"

	"github.com/ibm-messaging/mq-container/internal/keystore"
	"github.com/ibm-messaging/mq-container/internal/mqtemplate"
	"github.com/ibm-messaging/mq-container/internal/pathutils"
	"github.com/ibm-messaging/mq-container/pkg/logger"
)

// cmsKeystoreName is the name of the CMS Keystore
const cmsKeystoreName = "key.kdb"

// p12TruststoreName is the name of the PKCS#12 Truststore
const p12TruststoreName = "trust.p12"

// keystoreDirDefault is the location for the default CMS Keystore & PKCS#12 Truststore
const keystoreDirDefault = "/run/runmqserver/tls/"

// keystoreDirHA is the location for the HA CMS Keystore
const keystoreDirHA = "/run/runmqserver/ha/tls/"

// keyDirDefault is the location of the default keys to import
const keyDirDefault = "/etc/mqm/pki/keys"

// keyDirHA is the location of the HA keys to import
const keyDirHA = "/etc/mqm/ha/pki/keys"

// keyDirGroupHA is the location of the GroupHA keys to import
const keyDirGroupHA = "/etc/mqm/groupha/pki/keys"

// trustDirDefault is the location of the trust certificates to import
const trustDirDefault = "/etc/mqm/pki/trust"

// trustDirGroupDefault is the location of the GroupHA trust certificates to import
const trustDirGroupHA = "/etc/mqm/groupha/pki/trust"

type KeyStoreData struct {
	Keystore          *keystore.KeyStore
	Password          string
	TrustedCerts      []*pem.Block
	KnownFingerPrints []string
	KeyLabels         []string
	keyLabelLookup    map[comparablePrivateKey]privateKeyInfo
}

type privateKeyInfo struct {
	keySetName string
	directory  string
	filename   string
}

type P12KeyFiles struct {
	Keystores []string
	Password  string
}

type TLSStore struct {
	Keystore   KeyStoreData
	Truststore KeyStoreData
}

func configureTLSKeystores(keystoreDir string, keyDirs, trustDirs []string, p12TruststoreRequired bool, nativeTLSHA bool, log *logger.Logger) ([]string, KeyStoreData, KeyStoreData, error) {
	var keyLabel string

	cmsKeystoreRequired := false
	allDirs := append([]string{}, keyDirs...)
	allDirs = append(allDirs, trustDirs...)
	for _, dir := range allDirs {
		if haveKeysAndCerts(dir) {
			cmsKeystoreRequired = true
			break
		}
	}
	// Create the CMS Keystore & PKCS#12 Truststore (if required)
	tlsStore, err := generateAllKeystores(keystoreDir, cmsKeystoreRequired, p12TruststoreRequired, nativeTLSHA)
	if err != nil {
		return nil, tlsStore.Keystore, tlsStore.Truststore, err
	}

	keyLabels := make([]string, len(keyDirs))
	if tlsStore.Keystore.Keystore != nil {
		for idx, keyDir := range keyDirs {
			// Process all keys - add them to the CMS KeyStore
			keyLabel, err = processKeys(&tlsStore, keystoreDir, keyDir, log)
			if err != nil {
				return nil, tlsStore.Keystore, tlsStore.Truststore, err
			}
			keyLabels[idx] = keyLabel
		}
	}

	for _, trustDir := range trustDirs {
		// Process all trust certificates - add them to the CMS KeyStore & PKCS#12 Truststore (if required)
		err = processTrustCertificates(&tlsStore, trustDir)
		if err != nil {
			return nil, tlsStore.Keystore, tlsStore.Truststore, err
		}
	}

	return keyLabels, tlsStore.Keystore, tlsStore.Truststore, err
}

// ConfigureDefaultTLSKeystores configures the CMS Keystore & PKCS#12 Truststore
func ConfigureDefaultTLSKeystores(log *logger.Logger) (string, KeyStoreData, KeyStoreData, error) {
	certLabels, keyStore, trustStore, err := configureTLSKeystores(keystoreDirDefault, []string{keyDirDefault}, []string{trustDirDefault}, true, false, log)
	if err != nil {
		return "", keyStore, trustStore, err
	}
	certLabel := ""
	if len(certLabels) > 0 {
		certLabel = certLabels[0]
	}
	return certLabel, keyStore, trustStore, err
}

// ConfigureHATLSKeystore configures the CMS Keystore & PKCS#12 Truststore
func ConfigureHATLSKeystore(log *logger.Logger) (string, string, KeyStoreData, KeyStoreData, error) {
	// *.crt files mounted to the HA TLS dir keyDirHA will be processed as trusted in the CMS keystore
	keyDirs := []string{keyDirHA, keyDirGroupHA}
	trustDirs := []string{trustDirGroupHA}
	haCertLabels, haKeystore, haTruststore, err := configureTLSKeystores(keystoreDirHA, keyDirs, trustDirs, false, true, log)
	if err != nil {
		return "", "", haKeystore, haTruststore, err
	}
	if len(haCertLabels) != len(keyDirs) {
		return "", "", haKeystore, haTruststore, fmt.Errorf("incorrect number of certificate labels returned (expected %d, got %d)", len(keyDirs), len(haCertLabels))
	}

	return haCertLabels[0], haCertLabels[1], haKeystore, haTruststore, err
}

// ConfigureTLS configures TLS for the queue manager
func ConfigureTLS(keyLabel string, cmsKeystore KeyStoreData, devMode bool, log *logger.Logger) error {

	const mqscLink string = "/run/15-tls.mqsc"
	const mqscTemplate string = "/etc/mqm/15-tls.mqsc.tpl"
	sslKeyRing := ""
	var fipsEnabled = "NO"

	// Don't set SSLKEYR if no keys or crts are not supplied
	// Key label will be blank if no private keys were added during processing keys and certs.
	if cmsKeystore.Keystore != nil && len(keyLabel) > 0 {
		certList, _ := cmsKeystore.Keystore.ListAllCertificates()
		if len(certList) > 0 {
			sslKeyRing = strings.TrimSuffix(cmsKeystore.Keystore.Filename, ".kdb")
		}

		if cmsKeystore.Keystore.IsFIPSEnabled() {
			fipsEnabled = "YES"
		}
	}
	err := mqtemplate.ProcessTemplateFile(mqscTemplate, mqscLink, map[string]string{
		"SSLKeyR":          sslKeyRing,
		"CertificateLabel": keyLabel,
		"SSLFips":          fipsEnabled,
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

	const mqscLink string = "/run/20-dev-tls.mqsc"
	const mqscTemplate string = "/etc/mqm/20-dev-tls.mqsc.tpl"

	if os.Getenv("MQ_DEV") == "true" {
		err := mqtemplate.ProcessTemplateFile(mqscTemplate, mqscLink, map[string]string{}, log)
		if err != nil {
			return err
		}
	}

	return nil
}

// generateAllKeystores creates the CMS Keystore & PKCS#12 Truststore (if required)
func generateAllKeystores(keystoreDir string, createCMSKeystore bool, p12TruststoreRequired bool, nativeTLSHA bool) (TLSStore, error) {

	var cmsKeystore, p12Truststore KeyStoreData

	cmsKeystore.keyLabelLookup = map[comparablePrivateKey]privateKeyInfo{}
	p12Truststore.keyLabelLookup = map[comparablePrivateKey]privateKeyInfo{}

	// Generate a pasword for use with both the CMS Keystore & PKCS#12 Truststore
	pw := generateRandomPassword()
	cmsKeystore.Password = pw
	p12Truststore.Password = pw

	// Create the Keystore directory - if it does not already exist
	// #nosec G301 - write group permissions are required
	err := os.MkdirAll(keystoreDir, 0770)
	if err != nil {
		return TLSStore{cmsKeystore, p12Truststore}, fmt.Errorf("Failed to create Keystore directory: %v", err)
	}

	// Create the CMS Keystore if we have been provided keys and certificates
	if createCMSKeystore {
		cmsKeystore.Keystore = keystore.NewCMSKeyStore(pathutils.CleanPath(keystoreDir, cmsKeystoreName), cmsKeystore.Password)
		err = cmsKeystore.Keystore.Create()
		if err != nil {
			return TLSStore{cmsKeystore, p12Truststore}, fmt.Errorf("Failed to create CMS Keystore: %v", err)
		}
	}

	// Create the PKCS#12 Truststore (if required)
	if p12TruststoreRequired {
		p12Truststore.Keystore = keystore.NewPKCS12KeyStore(pathutils.CleanPath(keystoreDir, p12TruststoreName), p12Truststore.Password)
		err = p12Truststore.Keystore.Create()
		if err != nil {
			return TLSStore{cmsKeystore, p12Truststore}, fmt.Errorf("Failed to create PKCS#12 Truststore: %v", err)
		}
	}

	return TLSStore{cmsKeystore, p12Truststore}, nil
}

// processKeys processes all keys - adding them to the CMS KeyStore
func processKeys(tlsStore *TLSStore, keystoreDir string, keyDir string, log *logger.Logger) (string, error) {

	// Key label - will be set to the label of the first set of keys
	keyLabel := ""

	// Process all keys
	keyList, err := os.ReadDir(keyDir)
	if err == nil && len(keyList) > 0 {
		// Process each set of keys - each set should contain files: *.key & *.crt
		for _, keySet := range keyList {
			keys, _ := os.ReadDir(pathutils.CleanPath(keyDir, keySet.Name()))

			// Ensure the label of the set of keys does not match the name of the PKCS#12 Truststore
			if keySet.Name() == p12TruststoreName[0:len(p12TruststoreName)-len(filepath.Ext(p12TruststoreName))] {
				return "", fmt.Errorf("key label cannot be set to the Truststore name: %v", keySet.Name())
			}

			// Process private key (*.key)
			privateKey, keyPrefix, previousLabel, previousDirectory, err := processPrivateKey(keyDir, keySet.Name(), keys, &tlsStore.Keystore)
			if err != nil {
				return "", err
			}

			// If private key does not exist - skip this set of keys
			if privateKey == nil {
				continue
			}

			// Process certificates (*.crt) - public certificate & optional CA certificate
			publicCertificate, caCertificate, newPublicCerts, err := processCertificates(keyDir, keySet.Name(), keyPrefix, keys, &tlsStore.Keystore, &tlsStore.Truststore)
			if err != nil {
				return "", err
			}

			// Skip key if it has already been loaded
			if previousLabel != "" && !newPublicCerts {
				if keyLabel == "" {
					keyLabel = previousLabel
				}
				keySetDir := path.Join(keyDir, keySet.Name())
				log.Printf("No new keys found while processing '%s' (duplicate of '%s'); skip loading directory", keySetDir, previousDirectory)
				continue
			}

			// Return an error if corresponding public certificate was not found. Both private key and
			// it's corresponding public certificate are required.
			if publicCertificate == nil {
				return "", fmt.Errorf("Failed to find public certificate in directory %s", keyDir)
			}

			// Validate certificates for duplicate Subject DNs
			if len(caCertificate) > 0 {
				errCertValid := validateCertificates(publicCertificate, caCertificate)
				if errCertValid != nil {
					return "", errCertValid
				}
			}
			// Create a new PKCS#12 Keystore - containing private key, public certificate & optional CA certificate
			file, err := pkcs.Modern.Encode(privateKey, publicCertificate, caCertificate, tlsStore.Keystore.Password)
			if err != nil {
				return "", fmt.Errorf("Failed to encode PKCS#12 Keystore %s: %v", keySet.Name()+".p12", err)
			}
			keystorePath := pathutils.CleanPath(keystoreDir, keySet.Name()+".p12")
			// #nosec G306 - this gives permissions to owner/s group only.
			err = os.WriteFile(keystorePath, file, 0644)
			if err != nil {
				return "", fmt.Errorf("Failed to write PKCS#12 Keystore %s: %v", keystorePath, err)
			}

			// Import the new PKCS#12 Keystore into the CMS Keystore
			err = tlsStore.Keystore.Keystore.Import(keystorePath, tlsStore.Keystore.Password)
			if err != nil {
				return "", fmt.Errorf("Failed to import keys from %s into CMS Keystore: %v", keystorePath, err)
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

// processTrustCertificates processes all trust certificates - adding them to the CMS KeyStore & PKCS#12 Truststore (if required)
func processTrustCertificates(tlsStore *TLSStore, trustDir string) error {

	// Process all trust certiifcates
	trustList, err := os.ReadDir(trustDir)
	if err == nil && len(trustList) > 0 {

		// Process each set of keys
		for _, trustSet := range trustList {
			keys, _ := os.ReadDir(pathutils.CleanPath(trustDir, trustSet.Name()))

			for _, key := range keys {
				if strings.HasSuffix(key.Name(), ".crt") {
					trustSetPath := pathutils.CleanPath(trustDir, trustSet.Name(), key.Name())
					// #nosec G304 - filename variable is derived from contents of 'trustDir' which is a defined constant
					file, err := os.ReadFile(trustSetPath)
					if err != nil {
						return fmt.Errorf("Failed to read file %s: %v", trustSetPath, err)
					}

					for string(file) != "" {
						var block *pem.Block
						block, file = pem.Decode(file)
						if block == nil {
							break
						}

						// Add to known certificates for the CMS Keystore
						_, err = addToKnownCertificates(block, &tlsStore.Keystore, true)
						if err != nil {
							return fmt.Errorf("Failed to add to know certificates for CMS Keystore")
						}

						if tlsStore.Truststore.Keystore != nil {
							// Add to known certificates for the PKCS#12 Truststore
							_, err = addToKnownCertificates(block, &tlsStore.Truststore, true)
							if err != nil {
								return fmt.Errorf("Failed to add to know certificates for PKCS#12 Truststore")
							}
						}
					}
				}
			}
		}
	}

	// Add all trust certificates to PKCS#12 Truststore (if required)
	if tlsStore.Truststore.Keystore != nil && len(tlsStore.Truststore.TrustedCerts) > 0 {
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
func processPrivateKey(keyDir string, keySetName string, keys []os.DirEntry, keystore *KeyStoreData) (interface{}, string, string, string, error) {

	var privateKey crypto.PrivateKey
	keyPrefix := ""

	pkInfo := privateKeyInfo{}
	for _, key := range keys {

		privateKeyPath := pathutils.CleanPath(keyDir, keySetName, key.Name())
		if strings.HasSuffix(key.Name(), ".key") {
			// #nosec G304 - filename variable is derived from contents of 'keyDir' which is a defined constant
			file, err := os.ReadFile(privateKeyPath)
			if err != nil {
				return nil, "", "", "", fmt.Errorf("Failed to read private key %s: %v", privateKeyPath, err)
			}
			block, _ := pem.Decode(file)
			if block == nil {
				return nil, "", "", "", fmt.Errorf("Failed to decode private key %s: pem.Decode returned nil", privateKeyPath)
			}

			// Check if the private key is PKCS1
			privateKey, err = x509.ParsePKCS1PrivateKey(block.Bytes)
			if err != nil {
				// Check if the private key is PKCS8
				privateKey, err = x509.ParsePKCS8PrivateKey(block.Bytes)
				if err != nil {
					return nil, "", "", "", fmt.Errorf("Failed to parse private key %s: %v", privateKeyPath, err)
				}
			}
			keyPrefix = key.Name()[0 : len(key.Name())-len(filepath.Ext(key.Name()))]
			keySetDir := path.Join(keyDir, keySetName)
			pkInfo = privateKeyInfo{
				keySetName: keySetName,
				directory:  keySetDir,
				filename:   key.Name(),
			}
		}
	}

	for k, previous := range keystore.keyLabelLookup {
		if k.Equal(privateKey) {
			return privateKey, keyPrefix, previous.keySetName, previous.directory, nil
		}
	}

	comparableKey, ok := privateKey.(comparablePrivateKey)
	if !ok {
		return nil, "", "", "", fmt.Errorf("failed to cast private key to comparable type")
	}
	keystore.keyLabelLookup[comparableKey] = pkInfo

	return privateKey, keyPrefix, "", "", nil
}

// processCertificates processes the certificates (*.crt) from a set of keys
func processCertificates(keyDir string, keySetName, keyPrefix string, keys []os.DirEntry, cmsKeystore, p12Truststore *KeyStoreData) (*x509.Certificate, []*x509.Certificate, bool, error) {

	var publicCertificate *x509.Certificate
	var caCertificate []*x509.Certificate

	newPublicCount := 0
	for _, key := range keys {

		keystorePath := pathutils.CleanPath(keyDir, keySetName, key.Name())
		if strings.HasPrefix(key.Name(), keyPrefix) && strings.HasSuffix(key.Name(), ".crt") {
			// #nosec G304 - filename variable is derived from contents of 'keyDir' which is a defined constant
			file, err := os.ReadFile(keystorePath)
			if err != nil {
				return nil, nil, false, fmt.Errorf("Failed to read public certificate %s: %v", keystorePath, err)
			}
			block, file := pem.Decode(file)
			if block == nil {
				return nil, nil, false, fmt.Errorf("Failed to decode public certificate %s: pem.Decode returned nil", keystorePath)
			}
			publicCertificate, err = x509.ParseCertificate(block.Bytes)
			if err != nil {
				return nil, nil, false, fmt.Errorf("Failed to parse public certificate %s: %v", keystorePath, err)
			}

			// Add to known certificates for the CMS Keystore
			newCert, err := addToKnownCertificates(block, cmsKeystore, false)
			if err != nil {
				return nil, nil, false, fmt.Errorf("Failed to add to known certificates for CMS Keystore")
			}
			if newCert {
				newPublicCount++
			}

			// Pick up any other intermediate certificates
			for string(file) != "" {
				var block *pem.Block
				block, file = pem.Decode(file)
				if block == nil {
					break
				}

				// Add to known certificates for the CMS Keystore
				newIntermediate, err := addToKnownCertificates(block, cmsKeystore, false)
				if err != nil {
					return nil, nil, false, fmt.Errorf("Failed to add to known certificates for CMS Keystore")
				}
				if newIntermediate {
					newPublicCount++
				}

				if p12Truststore.Keystore != nil {
					// Add to known certificates for the PKCS#12 Truststore
					_, err = addToKnownCertificates(block, p12Truststore, true)
					if err != nil {
						return nil, nil, false, fmt.Errorf("Failed to add to known certificates for PKCS#12 Truststore")
					}
				}

				certificate, err := x509.ParseCertificate(block.Bytes)
				if err != nil {
					return nil, nil, false, fmt.Errorf("Failed to parse CA certificate %s: %v", keystorePath, err)
				}
				caCertificate = append(caCertificate, certificate)
			}

		} else if strings.HasSuffix(key.Name(), ".crt") {
			// #nosec G304 - filename variable is derived from contents of 'keyDir' which is a defined constant
			file, err := os.ReadFile(keystorePath)
			if err != nil {
				return nil, nil, false, fmt.Errorf("Failed to read CA certificate %s: %v", keystorePath, err)
			}

			for string(file) != "" {
				var block *pem.Block
				block, file = pem.Decode(file)
				if block == nil {
					break
				}

				// Add to known certificates for the CMS Keystore
				newIntermediate, err := addToKnownCertificates(block, cmsKeystore, false)
				if err != nil {
					return nil, nil, false, fmt.Errorf("Failed to add to known certificates for CMS Keystore")
				}
				if newIntermediate {
					newPublicCount++
				}

				if p12Truststore.Keystore != nil {
					// Add to known certificates for the PKCS#12 Truststore
					_, err = addToKnownCertificates(block, p12Truststore, true)
					if err != nil {
						return nil, nil, false, fmt.Errorf("Failed to add to known certificates for PKCS#12 Truststore")
					}
				}

				certificate, err := x509.ParseCertificate(block.Bytes)
				if err != nil {
					return nil, nil, false, fmt.Errorf("Failed to parse CA certificate %s: %v", keystorePath, err)
				}
				caCertificate = append(caCertificate, certificate)
			}
		}
	}

	return publicCertificate, caCertificate, newPublicCount > 0, nil
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

	temporaryPemFile := pathutils.CleanPath("/tmp", "trust.pem")
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

	temporaryPemFile := pathutils.CleanPath("/tmp", "cmsTrust.pem")
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
		// #nosec G404 - this is only for internal keystore and using math/rand pose no harm.
		password = password + string(validcharArray[pwr.Intn(len(validcharArray))])
	}

	return password
}

// addToKnownCertificates adds to the list of known certificates for a Keystore
func addToKnownCertificates(block *pem.Block, keyData *KeyStoreData, addToKeystore bool) (bool, error) {
	sha512str, err := getCertificateFingerprint(block)
	if err != nil {
		return false, err
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

	return !known, nil
}

// getCertificateFingerprint returns a fingerprint for a certificate
func getCertificateFingerprint(block *pem.Block) (string, error) {
	certificate, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return "", fmt.Errorf("Failed to parse x509 certificate: %v", err)
	}
	sha512Sum := sha512.Sum512(certificate.Raw)
	sha512str := hex.EncodeToString(sha512Sum[:])

	return sha512str, nil
}

// writeCertificatesToFile writes a list of certificates to a file
func writeCertificatesToFile(file string, certificates []*pem.Block) error {

	// #nosec G304 - this is a temporary pem file to write certs.
	f, err := os.Create(file)
	if err != nil {
		return fmt.Errorf("Failed to create file %s: %v", file, err)
	}
	// #nosec G307 - local to this function, pose no harm.
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

// Search the specified directory for .key and .crt files.
// Return true if at least one .key or .crt file is found else false
func haveKeysAndCerts(keyDir string) bool {
	fileList, err := os.ReadDir(keyDir)
	if err == nil && len(fileList) > 0 {
		for _, fileInfo := range fileList {
			// Keys and certs will be supplied in an user defined subdirectory.
			// Do a listing of the subdirectory and then search for .key and .cert files
			keys, _ := os.ReadDir(pathutils.CleanPath(keyDir, fileInfo.Name()))
			for _, key := range keys {
				if strings.HasSuffix(key.Name(), ".key") || strings.HasSuffix(key.Name(), ".crt") {
					// We found at least one key/crt file.
					return true
				}
			}
		}
	}
	return false
}

// Iterate through the certificates to ensure there are no two certificates with same Subject DN.
// GSKit does not allow two certificates with same Subject DN/Friendly Names
func validateCertificates(personalCert *x509.Certificate, caCertificates []*x509.Certificate) error {
	// Check if we have been asked to override certificate validation by setting
	// MQ_ENABLE_CERT_VALIDATION to false
	enableValidation, enableValidationSet := os.LookupEnv("MQ_ENABLE_CERT_VALIDATION")
	if !enableValidationSet || (enableValidationSet && !strings.EqualFold(strings.Trim(enableValidation, ""), "false")) {
		for _, caCert := range caCertificates {
			if strings.EqualFold(personalCert.Subject.String(), caCert.Subject.String()) {
				return fmt.Errorf("Error: The Subject DN of the Issuer Certificate and the Queue Manager are same")
			}
		}
	}
	return nil
}

// All go private keys implement this interface (https://pkg.go.dev/crypto#PrivateKey)
type comparablePrivateKey interface {
	Equal(x crypto.PrivateKey) bool
}
