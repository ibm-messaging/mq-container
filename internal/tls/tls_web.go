/*
© Copyright IBM Corporation 2019, 2023, 2026

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
	"fmt"
	"os"

	"github.com/ibm-messaging/mq-container/internal/keystore"
	"github.com/ibm-messaging/mq-container/internal/mqtemplate"
	"github.com/ibm-messaging/mq-container/internal/pathutils"
	"github.com/ibm-messaging/mq-container/internal/securityutility"
	"github.com/ibm-messaging/mq-container/internal/sensitive"
	"github.com/ibm-messaging/mq-container/pkg/logger"
)

// webKeystoreDefault is the name of the default web server Keystore
const webKeystoreDefault = "default.p12"

// ConfigureWebTLS configures TLS for the web server
func ConfigureWebTLS(keyLabel, webKeystore string, p12Truststore KeyStoreData, log *logger.Logger) error {

	// Return immediately if we have no certificate to use as identity
	if keyLabel == "" && os.Getenv("MQ_GENERATE_CERTIFICATE_HOSTNAME") == "" {
		return nil
	}

	// If trust-store is empty, set reference to point to the keystore
	webTruststoreRef := "MQWebTrustStore"
	if len(p12Truststore.TrustedCerts) == 0 {
		webTruststoreRef = "MQWebKeyStore"
	}

	var initialKey *sensitive.Sensitive
	key, err := os.ReadFile("/run/secrets/initial.key")
	if err != nil {
		log.Printf("No initial key specified at /run/secrets/initial.key. Generating a random initial key")
	} else {
		initialKey = trimKey(key)
		if initialKey == nil {
			log.Printf("WARNING: An initial key was specified under /run/secrets/initial.key but did not contain any valid character. Generating a random initial key")
		}
	}
	// Generate a random key to use, if the user did not provide a key, or the provided key was invalid.
	if initialKey == nil {
		initialKey = generateRandomPassword()
	}

	err = securityutility.GenerateLibertyAESKeyFile(initialKey)
	if err != nil {
		return err
	}

	tlsConfigLink := "/run/tls.xml"
	tlsConfigTemplate := "/etc/mqm/web/installations/Installation1/servers/mqweb/tls.xml.tpl"
	encryptedPassword, err := securityutility.EncodeSecrets(p12Truststore.Password, true)
	if err != nil {
		log.Printf("Password encoding for Web Keystore failed with error %v", err)
		// We couldn't encode the passwords so using an empty string as password
		encryptedPassword = ""
	}
	// Password successfully encoded using securityUtility use the encoded password the template
	templateErr := mqtemplate.ProcessTemplateFile(tlsConfigTemplate, tlsConfigLink, map[string]string{"password": encryptedPassword, "webKeystore": webKeystore, "webTruststoreRef": webTruststoreRef}, log)
	if templateErr != nil {
		return templateErr
	}

	return nil
}

// trimKey will return the first contiguous set of valid characters from the provided AES initialization key. If the key contains no valid characters it returns nil.
func trimKey(key []byte) *sensitive.Sensitive {
	// Valid characters are anything that is not a new line, whitespace, or a null byte.
	isValidChar := func(b byte) bool {
		switch b {
		case '\n', ' ', 0:
			return false
		default:
			return true
		}
	}
	startingMarker := 0
	finalMarker := 0
	// Find the start of the valid key
	for startingMarker = 0; startingMarker < len(key); startingMarker++ {
		if isValidChar(key[startingMarker]) {
			break
		}
		// Zero out any initial invalid characters
		key[startingMarker] = 0
	}
	// If there are no valid characters in the key return nil.
	if startingMarker >= len(key) {
		return nil
	}

	// Find the rest of the valid characters in the key
	for finalMarker = startingMarker + 1; finalMarker < len(key); finalMarker++ {
		if !isValidChar(key[finalMarker]) {
			break
		}
	}
	// Zero bytes after the final marker
	for i := finalMarker; i < len(key); i++ {
		key[i] = 0
	}
	// If the final and starting markers have no gap between them, the length of the valid key is 0 and it is invalid.
	if (finalMarker - startingMarker) < 1 {
		return nil
	}
	// Return the valid key from between the two markers
	return sensitive.New(key[startingMarker:finalMarker])
}

// ConfigureWebKeyStore configures the Web Keystore
func ConfigureWebKeystore(p12Truststore KeyStoreData, keyLabel string) (string, error) {

	webKeystore := webKeystoreDefault
	if keyLabel != "" {
		webKeystore = keyLabel + ".p12"
	}
	webKeystoreFile := pathutils.CleanPath(keystoreDirDefault, webKeystore)

	// Check if a new self-signed certificate should be generated
	if keyLabel == "" {

		// Get hostname to use for self-signed certificate
		genHostName := os.Getenv("MQ_GENERATE_CERTIFICATE_HOSTNAME")

		// Create the Web Keystore
		newWebKeystore := keystore.NewPKCS12KeyStore(webKeystoreFile, p12Truststore.Password)
		err := newWebKeystore.Create()
		if err != nil {
			return "", fmt.Errorf("failed to create Web Keystore %s: %v", webKeystoreFile, err)
		}

		// Generate a new self-signed certificate in the Web Keystore
		err = newWebKeystore.CreateSelfSignedCertificate("default", fmt.Sprintf("CN=%s", genHostName), genHostName)
		if err != nil {
			return "", fmt.Errorf("failed to generate certificate in Web Keystore %s with DN of 'CN=%s': %v", webKeystoreFile, genHostName, err)
		}
	} else {
		// Check Web Keystore already exists
		_, err := os.Stat(webKeystoreFile)
		if err != nil {
			return "", fmt.Errorf("failed to find existing Web Keystore %s: %v", webKeystoreFile, err)
		}
	}

	// Check Web Truststore already exists
	_, err := os.Stat(p12Truststore.Keystore.Filename)
	if err != nil {
		return "", fmt.Errorf("failed to find existing Web Truststore %s: %v", p12Truststore.Keystore.Filename, err)
	}

	return webKeystore, nil
}
