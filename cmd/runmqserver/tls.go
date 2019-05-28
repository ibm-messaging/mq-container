/*
© Copyright IBM Corporation 2018, 2019

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
	"bufio"
	"fmt"
	"io/ioutil"
	pwr "math/rand"
	"os"
	"path/filepath"
	"strings"
	"time"

	"crypto/rand"
	"crypto/sha1"
	"crypto/x509"
	"encoding/pem"

	"github.com/ibm-messaging/mq-container/internal/command"
	"github.com/ibm-messaging/mq-container/internal/keystore"
	"github.com/ibm-messaging/mq-container/internal/mqtemplate"
	pkcs "software.sslmate.com/src/go-pkcs12"
)

const keystoreDir = "/run/runmqserver/tls/"

// CIPDefaultLabel is the default certificate label used by Cloud Integration Platform
const CIPDefaultLabel = "default"

// P12TrustStoreName is the name of the PKCS#12 truststore used by the webconsole
const P12TrustStoreName = "trust.p12"

// CMSKeyStoreName is the name of the CMS Keystore used by the queue manager
const CMSKeyStoreName = "key.kdb"

// KeyStorePasswords The password of the keystores. Should never be printed!
var keyStorePasswords string

// KeyDir is the location of the certificate keys to import
const KeyDir = "/etc/mqm/pki/keys"

// TrustDir is the location of the Certifates to add
const TrustDir = "/etc/mqm/pki/trust"

// Used to track certificates and keys we've found to add
var p12TrustCerts []*pem.Block
var cmsTrustCerts []*pem.Block
var p12TrustKnownFingerPrints []string
var cmsTrustKnownFingerPrints []string
var cmsKeyLabels []string

// The keystore objects
var p12TrustStore *keystore.KeyStore
var cmsKeyStore *keystore.KeyStore

// Variables for storing the label of the certificate to use
var webkeyStoreName string
var qmKeyLabel string

// Tracks whether the keystores have been configured
var keystoresConfigured bool = false

func getCertFingerPrint(block *pem.Block) (string, error) {
	// Add to future truststore and known certs (if not already there)
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return "", fmt.Errorf("could not parse x509 certificate: %v", err)
	}
	sha1Sum := sha1.Sum(cert.Raw)
	sha1str := string(sha1Sum[:])

	return sha1str, nil
}

// Add to future pkcs#12 Truststore and known certs (if not already there)
func addCertToP12TrustStoreNoDups(block *pem.Block) error {
	sha1str, err := getCertFingerPrint(block)
	if err != nil {
		return err
	}
	known := false
	for _, t := range p12TrustKnownFingerPrints {
		if t == sha1str {
			known = true
			break
		}
	}

	if !known {
		p12TrustCerts = append(p12TrustCerts, block)
		p12TrustKnownFingerPrints = append(p12TrustKnownFingerPrints, sha1str)
	}
	return nil
}

// Add to CMS Keystore known certs (if not already there) and add to the
// CMS Keystore if "addToKeystore" is true.
func addCertToCMSKeystoreNoDups(block *pem.Block, addToKeystore bool) error {
	sha1str, err := getCertFingerPrint(block)
	if err != nil {
		return err
	}
	known := false
	for _, t := range cmsTrustKnownFingerPrints {
		if t == sha1str {
			known = true
			break
		}
	}

	if !known {
		// Sometimes we don't want to add to the CMS keystore trust here.
		// For example if it will be imported with the key later.
		if addToKeystore {
			cmsTrustCerts = append(cmsTrustCerts, block)
		}
		cmsTrustKnownFingerPrints = append(cmsTrustKnownFingerPrints, sha1str)
	}
	return nil
}

// Generates a random 12 character password from the characters a-z, A-Z, 0-9.
func generateRandomPassword() {
	pwr.Seed(time.Now().Unix())
	validChars := "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"
	validcharArray := []byte(validChars)
	password := ""
	for i := 0; i < 12; i++ {
		password = password + string(validcharArray[pwr.Intn(len(validcharArray))])
	}

	keyStorePasswords = password
}

// Creates the PKCS#12 Truststore and the CMS Keystore.
func generateAllStores() error {
	generateRandomPassword()

	// Create the keystore Directory (if it doesn't already exist)
	os.MkdirAll(keystoreDir, 0775)

	p12TrustStore = keystore.NewPKCS12KeyStore(filepath.Join(keystoreDir, P12TrustStoreName), keyStorePasswords)
	err := p12TrustStore.Create(log)
	if err != nil {
		return fmt.Errorf("Failed to create PKCS#12 TrustStore: %v", err)
	}

	cmsKeyStore = keystore.NewCMSKeyStore(filepath.Join(keystoreDir, CMSKeyStoreName), keyStorePasswords)
	err = cmsKeyStore.Create(log)
	if err != nil {
		return fmt.Errorf("Failed to create CMS KeyStore: %v", err)
	}

	return nil
}

// processKeys walks through the KeyDir directory and imports any keys it finds to individual PKCS#12 keystores
// and the CMS KeyStore. The label it uses is the name of the directory if finds the keys in.
func processKeys() error {
	keyList, err := ioutil.ReadDir(KeyDir)
	if err == nil && len(keyList) > 0 {
		// Found some keys, verify the contents
		for _, key := range keyList {
			keys, _ := ioutil.ReadDir(filepath.Join(KeyDir, key.Name()))
			keyLabel := key.Name()
			keyfilename := ""
			var keyfile interface{}
			var certFile *x509.Certificate
			var caFile []*x509.Certificate

			// find the keyfile name
			for _, a := range keys {
				if strings.HasSuffix(a.Name(), ".key") {
					keyFile, err := ioutil.ReadFile(filepath.Join(KeyDir, key.Name(), a.Name()))
					if err != nil {
						return fmt.Errorf("Could not read keyfile %s: %v", filepath.Join(KeyDir, key.Name(), a.Name()), err)
					}
					block, _ := pem.Decode(keyFile)
					if block == nil {
						return fmt.Errorf("Could not decode keyfile %s: pem.Decode returned nil", filepath.Join(KeyDir, key.Name(), a.Name()))
					}

					//Test whether it is PKCS1
					keyfile, err = x509.ParsePKCS1PrivateKey(block.Bytes)
					if err != nil {
						// Before we fail check whether it is PKCS8
						keyfile, err = x509.ParsePKCS8PrivateKey(block.Bytes)
						if err != nil {
							fmt.Printf("key %s ParsePKCS1/8PrivateKey ERR: %v\n", filepath.Join(KeyDir, key.Name(), a.Name()), err)
							return err
						}
						//It was PKCS8 afterall
					}
					keyfilename = a.Name()
				}
			}
			if keyfile == nil {
				break
			}

			// Find out what the keyfile was called without the extension
			prefix := keyfilename[0 : len(keyfilename)-len(filepath.Ext(keyfilename))]

			for _, a := range keys {
				if strings.HasSuffix(a.Name(), ".key") {
					continue
				}
				if strings.HasPrefix(a.Name(), prefix) && strings.HasSuffix(a.Name(), ".crt") {
					cert, err := ioutil.ReadFile(filepath.Join(KeyDir, key.Name(), a.Name()))
					if err != nil {
						return fmt.Errorf("Could not read file %s: %v", filepath.Join(KeyDir, key.Name(), a.Name()), err)
					}
					block, _ := pem.Decode(cert)
					if block == nil {
						return fmt.Errorf("Could not decode certificate %s: pem.Decode returned nil", filepath.Join(KeyDir, key.Name(), a.Name()))
					}
					certFile, err = x509.ParseCertificate(block.Bytes)
					if err != nil {
						return fmt.Errorf("Could not parse certificate %s: %v", filepath.Join(KeyDir, key.Name(), a.Name()), err)
					}
					// Add to the dup list for the CMS keystore but not the PKCS#12 Truststore
					addCertToCMSKeystoreNoDups(block, false)

				} else if strings.HasSuffix(a.Name(), ".crt") {
					remainder, err := ioutil.ReadFile(filepath.Join(KeyDir, key.Name(), a.Name()))
					if err != nil {
						return fmt.Errorf("Could not read file %s: %v", filepath.Join(KeyDir, key.Name(), a.Name()), err)
					}

					for string(remainder) != "" {
						var block *pem.Block
						block, remainder = pem.Decode(remainder)
						// If we can't decode the CA certificate then just exit.
						if block == nil {
							break
						}

						// Add to the dup list for the CMS keystore
						addCertToCMSKeystoreNoDups(block, false)

						// Add to the p12 truststore
						addCertToP12TrustStoreNoDups(block)

						caCert, err := x509.ParseCertificate(block.Bytes)
						if err != nil {
							return fmt.Errorf("Could not parse CA certificate %s: %v", filepath.Join(KeyDir, key.Name(), a.Name()), err)
						}

						caFile = append(caFile, caCert)
					}
				}
			}

			// Create p12 keystore
			file, err := pkcs.Encode(rand.Reader, keyfile, certFile, caFile, keyStorePasswords)
			if err != nil {
				return fmt.Errorf("Could not encode PKCS#12 Keystore %s: %v", keyLabel+".p12", err)
			}

			err = ioutil.WriteFile(filepath.Join(keystoreDir, keyLabel+".p12"), file, 0644)
			if err != nil {
				return fmt.Errorf("Could not write PKCS#12 Keystore %s: %v", filepath.Join(keystoreDir, keyLabel+".p12"), err)
			}

			// Add to the CMS keystore
			err = cmsKeyStore.Import(filepath.Join(keystoreDir, keyLabel+".p12"), keyStorePasswords)
			if err != nil {
				return fmt.Errorf("Could not import keys from %s into CMS Keystore: %v", filepath.Join(keystoreDir, keyLabel+".p12"), err)
			}

			// Relabel it
			allLabels, err := cmsKeyStore.GetCertificateLabels()
			if err != nil {
				return fmt.Errorf("Could not list keys in CMS Keystore: %v", err)
			}
			relabelled := false
			for _, cl := range allLabels {
				found := false
				for _, kl := range cmsKeyLabels {
					if strings.Trim(cl, "\"") == kl {
						found = true
						break
					}
				}
				if !found {
					// This is the one to rename
					err = cmsKeyStore.RenameCertificate(strings.Trim(cl, "\""), keyLabel)
					if err != nil {
						return err
					}
					relabelled = true
					cmsKeyLabels = append(cmsKeyLabels, keyLabel)
					break
				}
			}

			if !relabelled {
				return fmt.Errorf("Unable to find the added key for %s in CMS keystore", keyLabel)
			}

			// First keystore so mark as the one to use with web console.
			if webkeyStoreName == "" {
				webkeyStoreName = keyLabel + ".p12"
			}
			// First key added to CMS Keystore so mark it as the one to use with the queue manager.
			if qmKeyLabel == "" {
				qmKeyLabel = keyLabel
			}
		}
	}
	return nil
}

// processTrustCertificates walks through the TrustDir directory and adds any certificates it finds
// to the PKCS#12 Truststore and the CMS KeyStore as long as has not already been added.
func processTrustCertificates() error {
	certList, err := ioutil.ReadDir(TrustDir)
	if err == nil && len(certList) > 0 {
		// Found some keys, verify the contents
		for _, cert := range certList {
			certs, _ := ioutil.ReadDir(filepath.Join(TrustDir, cert.Name()))
			for _, a := range certs {
				if strings.HasSuffix(a.Name(), ".crt") {
					remainder, err := ioutil.ReadFile(filepath.Join(TrustDir, cert.Name(), a.Name()))
					if err != nil {
						return fmt.Errorf("Could not read file %s: %v", filepath.Join(TrustDir, cert.Name(), a.Name()), err)
					}

					for string(remainder) != "" {
						var block *pem.Block
						block, remainder = pem.Decode(remainder)
						if block == nil {
							break
						}

						// Add to the CMS keystore
						addCertToCMSKeystoreNoDups(block, true)

						// Add to the p12 truststore
						addCertToP12TrustStoreNoDups(block)
					}
				}
			}
		}
	}
	// We've potentially created two lists of certificates to import. Add them both to relevant Truststores
	if len(p12TrustCerts) > 0 {
		// Do P12 TrustStore first
		temporaryPemFile := filepath.Join("/tmp", "trust.pem")

		err := writeCertsToFile(temporaryPemFile, p12TrustCerts)
		if err != nil {
			return err
		}

		err = p12TrustStore.AddNoLabel(temporaryPemFile)
		if err != nil {
			return fmt.Errorf("Could not add certificates to PKCS#12 Truststore: %v", err)
		}
	}

	if len(cmsTrustCerts) > 0 {
		// Now the CMS Keystore
		temporaryPemFile := filepath.Join("/tmp", "cmsTrust.pem")

		err = writeCertsToFile(temporaryPemFile, cmsTrustCerts)
		if err != nil {
			return err
		}

		err = cmsKeyStore.AddNoLabel(temporaryPemFile)
		if err != nil {
			return fmt.Errorf("Could not add certificates to CMS keystore: %v", err)
		}
	}
	return nil
}

// Writes a given list of certificates to a file.
func writeCertsToFile(file string, certs []*pem.Block) error {
	f, err := os.Create(file)
	if err != nil {
		return fmt.Errorf("writeCertsToFile: Could not create file %s: %v", file, err)
	}
	defer f.Close()

	w := bufio.NewWriter(f)

	for i, c := range certs {
		err := pem.Encode(w, c)
		if err != nil {
			return fmt.Errorf("writeCertsToFile: Could not encode certificate number %d: %v", i, err)
		}
		w.Flush()
	}
	return nil
}

// ConfigureTLSKeystores sets up the  TLS Trust and Keystores for use
func ConfigureTLSKeystores() error {
	err := generateAllStores()
	if err != nil {
		return err
	}

	err = handleCIPGeneratedCerts()
	if err != nil {
		return err
	}

	err = expandOldTLSVarible()
	if err != nil {
		return err
	}

	err = processKeys()
	if err != nil {
		return err
	}

	err = processTrustCertificates()
	if err != nil {
		return err
	}

	// set that the keystores have been configured
	keystoresConfigured = true

	return nil
}

// ConfigureWebTLS configures TLS for Web Console
func ConfigureWebTLS() error {
	if !keystoresConfigured {
		err := ConfigureTLSKeystores()
		if err != nil {
			return err
		}
	}

	const webConfigDir string = "/etc/mqm/web/installations/Installation1/servers/mqweb"
	const tls string = "tls.xml"
	const tlsTemplate string = tls + ".tpl"

	tlsConfig := filepath.Join(webConfigDir, tls)
	newTLSConfig := filepath.Join(webConfigDir, tlsTemplate)
	err := os.Remove(tlsConfig)
	if err != nil {
		return fmt.Errorf("Could not delete file %s: %v", tlsConfig, err)
	}
	// we symlink here to prevent issues on restart
	err = os.Symlink(newTLSConfig, tlsConfig)
	if err != nil {
		return fmt.Errorf("Could not create symlink %s->%s: %v", newTLSConfig, tlsConfig, err)
	}
	mqmUID, mqmGID, err := command.LookupMQM()
	if err != nil {
		return fmt.Errorf("Could not find mqm user or group: %v", err)
	}
	err = os.Chown(tlsConfig, mqmUID, mqmGID)
	if err != nil {
		return fmt.Errorf("Could change ownership of %s to mqm: %v", tlsConfig, err)
	}

	return nil
}

// ConfigureTLSDev configures TLS for developer defaults
func ConfigureTLSDev() error {
	if !keystoresConfigured {
		err := ConfigureTLSKeystores()
		if err != nil {
			return err
		}
	}
	if qmKeyLabel == "" {
		// We haven't set a key to use so don't set the QM to use TLS
		return nil
	}
	const mqsc string = "/etc/mqm/20-dev-tls.mqsc"
	const mqscTemplate string = mqsc + ".tpl"
	const sslCipherSpec string = "TLS_RSA_WITH_AES_128_CBC_SHA256"

	if os.Getenv("MQ_DEV") == "true" {
		err := mqtemplate.ProcessTemplateFile(mqscTemplate, mqsc, map[string]string{
			"SSLCipherSpec": sslCipherSpec,
		}, log)
		if err != nil {
			return err
		}
	} else {
		_, err := os.Stat(mqsc)
		if !os.IsNotExist(err) {
			err = os.Remove(mqsc)
			if err != nil {
				log.Errorf("Error removing file %s: %v", mqsc, err)
				return err
			}
		}
	}

	return nil
}

// ConfigureTLS configures TLS for queue manager
func ConfigureTLS(devmode bool) error {
	if !keystoresConfigured {
		err := ConfigureTLSKeystores()
		if err != nil {
			return err
		}
	}
	log.Debug("Configuring TLS")
	keyFile := filepath.Join(keystoreDir, CMSKeyStoreName)

	const mqsc string = "/etc/mqm/15-tls.mqsc"
	const mqscTemplate string = mqsc + ".tpl"

	err := mqtemplate.ProcessTemplateFile(mqscTemplate, mqsc, map[string]string{
		"SSLKeyR":          strings.TrimSuffix(keyFile, ".kdb"),
		"CertificateLabel": qmKeyLabel,
	}, log)
	if err != nil {
		return err
	}

	if devmode {
		err = ConfigureTLSDev()
		if err != nil {
			return err
		}
	}

	return nil
}

// ConfigureSSOTLS configures TLS for the Cloud Integration Platform Single Sign-On
func ConfigureSSOTLS() error {
	if !keystoresConfigured {
		err := ConfigureTLSKeystores()
		if err != nil {
			return err
		}
	}

	// TODO find way to supply this
	// Override the webstore variables to hard coded defaults
	webkeyStoreName = CIPDefaultLabel + ".p12"

	// Check keystore exists
	ks := filepath.Join(keystoreDir, webkeyStoreName)
	_, err := os.Stat(ks)
	if err != nil {
		return fmt.Errorf("Failed to find existing keystore %s: %v", ks, err)
	}

	// Check truststore exists
	ts := filepath.Join(keystoreDir, P12TrustStoreName)
	_, err = os.Stat(ts)
	if err != nil {
		return fmt.Errorf("Failed to find existing truststore %s: %v", ts, err)
	}

	// Add OIDC cert to the truststore
	err = p12TrustStore.Add(os.Getenv("MQ_OIDC_CERTIFICATE"), "OIDC")
	if err != nil {
		return err
	}

	return nil
}

// This function supports the old mechanism of importing certificates supplied for
// Cloud Integration platform
func handleCIPGeneratedCerts() error {
	dir := "/mnt/tls"
	outputdir := filepath.Join(KeyDir, CIPDefaultLabel)
	keyfile := "tls.key"
	crtfile := "tls.crt"

	// check that the files exist, if not just quietly leave as there's nothing to import
	_, err := os.Stat(filepath.Join(dir, keyfile))
	if err != nil {
		return nil
	}

	_, err = os.Stat(filepath.Join(dir, crtfile))
	if err != nil {
		return nil
	}

	// Check the destination directory DOES not exist ahead of time
	_, err = os.Stat(outputdir)
	if err == nil {
		return fmt.Errorf("Found CIP certificates to import but a TLS secret called %s is already present", CIPDefaultLabel)
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("Failed to check that %s did not exist: %v", outputdir, err)
	}

	// We have certificate to import and no duplicates
	err = os.MkdirAll(outputdir, 0775)
	if err != nil {
		return fmt.Errorf("Could not create %s: %v", outputdir, err)
	}

	err = CopyFileMode(filepath.Join(dir, keyfile), filepath.Join(outputdir, keyfile), 0644)
	if err != nil {
		return fmt.Errorf("Could not copy %s: %v", keyfile, err)
	}

	err = CopyFileMode(filepath.Join(dir, crtfile), filepath.Join(outputdir, crtfile), 0644)
	if err != nil {
		return fmt.Errorf("Could not copy %s: %v", keyfile, err)
	}

	// With certificates copied into place the rest of the TLS handling code will import them into the correct place
	return nil
}

// This function supports the old mechanism of importing certificates supplied by the MQ_TLS_KEYSTORE envvar
func expandOldTLSVarible() error {
	// TODO: Change this or find a way to set it
	outputDirName := "acopiedcertificate"

	// Check whether the old variable is set. If not exit quietly
	keyfile := os.Getenv("MQ_TLS_KEYSTORE")
	if keyfile == "" {
		return nil
	}

	// There is a file to read and process
	keyfilepw := os.Getenv("MQ_TLS_PASSPHRASE")

	if !strings.HasSuffix(keyfile, ".p12") {
		return fmt.Errorf("MQ_TLS_KEYSTORE (%s) does not point to a PKCS#12 file ending with the suffix .p12", keyfile)
	}

	_, err := os.Stat(keyfile)
	if err != nil {
		return fmt.Errorf("File %s referenced by MQ_TLS_KEYSTORE does not exist", keyfile)
	}

	readkey, err := ioutil.ReadFile(keyfile)
	if err != nil {
		return fmt.Errorf("Failed to read %s: %v", keyfile, err)
	}

	// File has been checked and read, decode it.
	pk, cert, cas, err := pkcs.DecodeChain(readkey, keyfilepw)
	if err != nil {
		return fmt.Errorf("Failed to decode %s: %v", keyfile, err)
	}

	// Find a directory name that doesn't exist
	for {
		_, err := os.Stat(filepath.Join(KeyDir, outputDirName))
		if err == nil {
			outputDirName = outputDirName + "0"
		} else {
			break
		}
	}

	//Bceause they supplied this certificate using the old method we should use this for qm & webconsole
	webkeyStoreName = outputDirName + ".p12"
	qmKeyLabel = outputDirName

	// Write out the certificate for the private key
	if cert != nil {
		block := pem.Block{
			Type:    "CERTIFICATE",
			Headers: nil,
			Bytes:   cert.Raw,
		}
		err = addCertToCMSKeystoreNoDups(&block, false)
		if err != nil {
			return fmt.Errorf("expandOldTLSVarible: Failed to add cert to CMS Keystore duplicate list: %v", err)
		}
		err = addCertToP12TrustStoreNoDups(&block)
		if err != nil {
			return fmt.Errorf("expandOldTLSVarible: Failed to add cert to P12 Truststore duplicate list: %v", err)
		}
	}

	// now write out all the ca certificates
	if cas != nil || len(cas) > 0 {
		for i, c := range cas {
			block := pem.Block{
				Type:    "CERTIFICATE",
				Headers: nil,
				Bytes:   c.Raw,
			}

			// Add to the dup list for the CMS keystore
			err = addCertToCMSKeystoreNoDups(&block, false)
			if err != nil {
				return fmt.Errorf("expandOldTLSVarible: Failed to add CA cert %d to CMS Keystore duplicate list: %v", i, err)
			}

			// Add to the p12 truststore
			err = addCertToP12TrustStoreNoDups(&block)
			if err != nil {
				return fmt.Errorf("expandOldTLSVarible: Failed to add CA cert %d to P12 Truststore duplicate list: %v", i, err)
			}
		}
	}

	// Now we've handled the certificates copy the keystore into place
	destination := filepath.Join(keystoreDir, outputDirName+".p12")

	// Create p12 keystore
	file, err := pkcs.Encode(rand.Reader, pk, cert, cas, keyStorePasswords)
	if err != nil {
		return fmt.Errorf("Failed to re-encode p12 keystore: %v", err)
	}

	err = ioutil.WriteFile(destination, file, 0644)
	if err != nil {
		return fmt.Errorf("Failed to write p12 keystore: %v", err)
	}

	// Add to the CMS keystore
	err = cmsKeyStore.Import(destination, keyStorePasswords)
	if err != nil {
		return fmt.Errorf("Failed to import p12 keystore %s: %v", destination, err)
	}

	if pk != nil {
		// Relabel the key
		allLabels, err := cmsKeyStore.GetCertificateLabels()
		if err != nil {
			fmt.Printf("cms GetCertificateLabels: %v\n", err)
			return err
		}
		relabelled := false
		for _, cl := range allLabels {
			found := false
			for _, kl := range cmsKeyLabels {
				if strings.Trim(cl, "\"") == kl {
					found = true
					break
				}
			}
			if !found {
				// This is the one to rename
				err = cmsKeyStore.RenameCertificate(strings.Trim(cl, "\""), outputDirName)
				if err != nil {
					return err
				}
				relabelled = true
				cmsKeyLabels = append(cmsKeyLabels, outputDirName)
				break
			}
		}

		if !relabelled {
			return fmt.Errorf("Unable to find the added key in CMS keystore")
		}
	}

	return nil
}
