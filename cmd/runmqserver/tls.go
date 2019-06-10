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
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ibm-messaging/mq-container/internal/command"
	"github.com/ibm-messaging/mq-container/internal/mqtemplate"
	"github.com/ibm-messaging/mq-container/internal/tls"
)

// Location to store the keystores
const keyStoreDir = "/run/runmqserver/tls/"

// KeyDir is the location of the certificate keys to import
const keyDir = "/etc/mqm/pki/keys"

// TrustDir is the location of the Certifates to add
const trustDir = "/etc/mqm/pki/trust"

// configureWebTLS configures TLS for Web Console
func configureWebTLS(label string) error {
	// Return immediately if we have no certificate to use as identity
	if label == "" {
		return nil
	}

	webConfigDir := "/etc/mqm/web/installations/Installation1/servers/mqweb"
	tls := "tls.xml"

	tlsConfig := filepath.Join(webConfigDir, tls)
	newTLSConfig := filepath.Join(webConfigDir, tls+".tpl")
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

// configureTLSDev configures TLS for developer defaults
func configureTLSDev() error {
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

// configureTLS configures TLS for queue manager
func configureTLS(certLabel string, cmsKeystore tls.KeyStoreData, devmode bool) error {
	log.Debug("Configuring TLS")

	const mqsc string = "/etc/mqm/15-tls.mqsc"
	const mqscTemplate string = mqsc + ".tpl"

	err := mqtemplate.ProcessTemplateFile(mqscTemplate, mqsc, map[string]string{
		"SSLKeyR":          strings.TrimSuffix(cmsKeystore.Keystore.Filename, ".kdb"),
		"CertificateLabel": certLabel,
	}, log)
	if err != nil {
		return err
	}

	if devmode && certLabel != "" {
		err = configureTLSDev()
		if err != nil {
			return err
		}
	}

	return nil
}

// configureSSOTLS configures MQ Console TLS for Single Sign-On
func configureSSOTLS(p12TrustStore tls.KeyStoreData) (string, error) {
	// TODO find way to supply this
	// Override the webstore variables to hard coded defaults
	webKeyStoreName := tls.IntegrationDefaultLabel + ".p12"

	// Check keystore exists
	ks := filepath.Join(keyStoreDir, webKeyStoreName)
	_, err := os.Stat(ks)
	if err != nil {
		return "", fmt.Errorf("Failed to find existing keystore %s: %v", ks, err)
	}

	// Check truststore exists
	_, err = os.Stat(p12TrustStore.Keystore.Filename)
	if err != nil {
		return "", fmt.Errorf("Failed to find existing truststore %s: %v", p12TrustStore.Keystore.Filename, err)
	}

	return webKeyStoreName, nil
}
