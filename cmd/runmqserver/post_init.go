/*
© Copyright IBM Corporation 2018, 2022

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
	"os"

	"github.com/ibm-messaging/mq-container/internal/fips"
	"github.com/ibm-messaging/mq-container/internal/tls"
)

// postInit is run after /var/mqm is set up
func postInit(name, keyLabel string, p12Truststore tls.KeyStoreData) error {
	enableWebServer := os.Getenv("MQ_ENABLE_EMBEDDED_WEB_SERVER")
	if enableWebServer == "true" || enableWebServer == "1" {
		// Configure the web server (if enabled)
		webKeystore, err := configureWebServer(keyLabel, p12Truststore)
		if err != nil {
			return err
		}
		// If trust-store is empty, set reference to point to the keystore
		webTruststoreRef := "MQWebTrustStore"
		if len(p12Truststore.TrustedCerts) == 0 {
			webTruststoreRef = "MQWebKeyStore"
		}

		// Enable FIPS for MQ Web Server if asked for.
		if fips.IsFIPSEnabled() {
			err = configureFIPSWebServer(p12Truststore)
			if err != nil {
				return err
			}
		}

		// Start the web server, in the background (if installed)
		// WARNING: No error handling or health checking available for the web server
		go func() {
			err = startWebServer(webKeystore, p12Truststore.Password, webTruststoreRef)
			if err != nil {
				log.Printf("Error starting web server: %v", err)
			}
		}()
	}
	return nil
}
