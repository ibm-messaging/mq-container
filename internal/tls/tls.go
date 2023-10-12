/*
© Copyright IBM Corporation 2019, 2023

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

// keystoreDirDefault is the location for the default CMS Keystore & PKCS#12 Truststore
const keystoreDirDefault = "/run/runmqserver/tls/"

// keystoreDirHA is the location for the HA CMS Keystore
const keystoreDirHA = "/run/runmqserver/ha/tls/"

// keyDirDefault is the location of the default keys to import
const keyDirDefault = "/etc/mqm/pki/keys"

// keyDirHA is the location of the HA keys to import
const keyDirHA = "/etc/mqm/ha/pki/keys"

// trustDirDefault is the location of the trust certificates to import
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

func configureTLSKeystores(keystoreDir, keyDir, trustDir string, p12TruststoreRequired bool, nativeTLSHA bool) (string, KeyStoreData, KeyStoreData, error) {
	var keyLabel string
	// Create the CMS Keystore & PKCS#12 Truststore (if required)
	tlsStore, err := generateAllKeystores(keystoreDir, p12TruststoreRequired, nativeTLSHA)
	if err != nil {
		return "", tlsStore.Keystore, tlsStore.Truststore, err
	}

	if tlsStore.Keystore.Keystore != nil {
		// Process all keys - add them to the CMS KeyStore
		keyLabel, err = processKeys(&tlsStore, keystoreDir, keyDir)
		if err != nil {
			return "", tlsStore.Keystore, tlsStore.Truststore, err
		}
	}

	// Process all trust certificates - add them to the CMS KeyStore & PKCS#12 Truststore (if required)
	err = processTrustCertificates(&tlsStore, trustDir)
	if err != nil {
		return "", tlsStore.Keystore, tlsStore.Truststore, err
	}

	return keyLabel, tlsStore.Keystore, tlsStore.Truststore, err
}

// ConfigureDefaultTLSKeystores configures the CMS Keystore & PKCS#12 Truststore
func ConfigureDefaultTLSKeystores() (string, KeyStoreData, KeyStoreData, error) {
	return configureTLSKeystores(keystoreDirDefault, keyDirDefault, trustDirDefault, true, false)
}

// ConfigureHATLSKeystore configures the CMS Keystore & PKCS#12 Truststore
func ConfigureHATLSKeystore() (string, KeyStoreData, KeyStoreData, error) {
	// *.crt files mounted to the HA TLS dir keyDirHA will be processed as trusted in the CMS keystore
	return configureTLSKeystores(keystoreDirHA, keyDirHA, keyDirHA, false, true)
}

// ConfigureTLS configures TLS for the queue manager
func ConfigureTLS(keyLabel string, cmsKeystore KeyStoreData, devMode bool, log *logger.Logger) error {

	const mqsc string = "/etc/mqm/15-tls.mqsc"
	const mqscTemplate string = mqsc + ".tpl"
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

	err := mqtemplate.ProcessTemplateFile(mqscTemplate, mqsc, map[string]string{
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

// generateAllKeystores creates the CMS Keystore & PKCS#12 Truststore (if required)
func generateAllKeystores(keystoreDir string, p12TruststoreRequired bool, nativeTLSHA bool) (TLSStore, error) {

	var cmsKeystore, p12Truststore KeyStoreData

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

	// Search the default keys directory for any keys/certs.
	keysDirectory := keyDirDefault
	// Change to default native HA TLS directory if we are configuring nativeHA
	if nativeTLSHA {
		keysDirectory = keyDirHA
	}
	// Create the CMS Keystore if we have been provided keys and certificates
	if haveKeysAndCerts(keysDirectory) || haveKeysAndCerts(trustDirDefault) {
		cmsKeystore.Keystore = keystore.NewCMSKeyStore(filepath.Join(keystoreDir, cmsKeystoreName), cmsKeystore.Password)
		err = cmsKeystore.Keystore.Create()
		if err != nil {
			return TLSStore{cmsKeystore, p12Truststore}, fmt.Errorf("Failed to create CMS Keystore: %v", err)
		}
	}

	// Create the PKCS#12 Truststore (if required)
	if p12TruststoreRequired {
		p12Truststore.Keystore = keystore.NewPKCS12KeyStore(filepath.Join(keystoreDir, p12TruststoreName), p12Truststore.Password)
		err = p12Truststore.Keystore.Create()
		if err != nil {
			return TLSStore{cmsKeystore, p12Truststore}, fmt.Errorf("Failed to create PKCS#12 Truststore: %v", err)
		}
	}

	return TLSStore{cmsKeystore, p12Truststore}, nil
}

// processKeys processes all keys - adding them to the CMS KeyStore
func processKeys(tlsStore *TLSStore, keystoreDir string, keyDir string) (string, error) {

	// Key label - will be set to the label of the first set of keys
	keyLabel := ""

	// Process all keys
	keyList, err := os.ReadDir(keyDir)
	if err == nil && len(keyList) > 0 {
		// Process each set of keys - each set should contain files: *.key & *.crt
		for _, keySet := range keyList {
			keys, _ := os.ReadDir(filepath.Join(keyDir, keySet.Name()))

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

			// Validate certificates for duplicate Subject DNs
			if len(caCertificate) > 0 {
				errCertValid := validateCertificates(publicCertificate, caCertificate)
				if errCertValid != nil {
					return "", errCertValid
				}
			}
			// Create a new PKCS#12 Keystore - containing private key, public certificate & optional CA certificate
			file, err := pkcs.Encode(rand.Reader, privateKey, publicCertificate, caCertificate, tlsStore.Keystore.Password)
			if err != nil {
				return "", fmt.Errorf("Failed to encode PKCS#12 Keystore %s: %v", keySet.Name()+".p12", err)
			}
			// #nosec G306 - this gives permissions to owner/s group only.
			err = os.WriteFile(filepath.Join(keystoreDir, keySet.Name()+".p12"), file, 0644)
			if err != nil {
				return "", fmt.Errorf("Failed to write PKCS#12 Keystore %s: %v", filepath.Join(keystoreDir, keySet.Name()+".p12"), err)
			}

			// Import the new PKCS#12 Keystore into the CMS Keystore
			err = tlsStore.Keystore.Keystore.Import(filepath.Join(keystoreDir, keySet.Name()+".p12"), tlsStore.Keystore.Password)
			if err != nil {
				return "", fmt.Errorf("Failed to import keys from %s into CMS Keystore: %v", filepath.Join(keystoreDir, keySet.Name()+".p12"), err)
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
			keys, _ := os.ReadDir(filepath.Join(trustDir, trustSet.Name()))

			for _, key := range keys {
				if strings.HasSuffix(key.Name(), ".crt") {
					// #nosec G304 - filename variable is derived from contents of 'trustDir' which is a defined constant
					file, err := os.ReadFile(filepath.Join(trustDir, trustSet.Name(), key.Name()))
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

						if tlsStore.Truststore.Keystore != nil {
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
func processPrivateKey(keyDir string, keySetName string, keys []os.DirEntry) (interface{}, string, error) {

	var privateKey interface{}
	keyPrefix := ""

	for _, key := range keys {

		if strings.HasSuffix(key.Name(), ".key") {
			// #nosec G304 - filename variable is derived from contents of 'keyDir' which is a defined constant
			file, err := os.ReadFile(filepath.Join(keyDir, keySetName, key.Name()))
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
func processCertificates(keyDir string, keySetName, keyPrefix string, keys []os.DirEntry, cmsKeystore, p12Truststore *KeyStoreData) (*x509.Certificate, []*x509.Certificate, error) {

	var publicCertificate *x509.Certificate
	var caCertificate []*x509.Certificate

	for _, key := range keys {

		if strings.HasPrefix(key.Name(), keyPrefix) && strings.HasSuffix(key.Name(), ".crt") {
			// #nosec G304 - filename variable is derived from contents of 'keyDir' which is a defined constant
			file, err := os.ReadFile(filepath.Join(keyDir, keySetName, key.Name()))
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
			file, err := os.ReadFile(filepath.Join(keyDir, keySetName, key.Name()))
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

				if p12Truststore.Keystore != nil {
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
		// #nosec G404 - this is only for internal keystore and using math/rand pose no harm.
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
			keys, _ := os.ReadDir(filepath.Join(keyDir, fileInfo.Name()))
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
