/*
© Copyright IBM Corporation 2018, 2023

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
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/ibm-messaging/mq-container/internal/copy"
	"github.com/ibm-messaging/mq-container/internal/mqtemplate"
	"github.com/ibm-messaging/mq-container/internal/tls"
)

func startWebServer(webKeystore, webkeystorePW, webTruststoreRef string) error {
	_, err := os.Stat("/opt/mqm/bin/strmqweb")
	if err != nil && os.IsNotExist(err) {
		log.Debug("Skipping web server, because it's not installed")
		return nil
	}
	log.Println("Starting web server")
	// #nosec G204 - command is fixed, no injection vector
	cmd := exec.Command("strmqweb")

	// Pass all the environment to MQ Web Server JVM
	cmd.Env = os.Environ()

	// TLS enabled
	if webKeystore != "" {
		cmd.Env = append(cmd.Env, "AMQ_WEBKEYSTORE="+webKeystore)
		cmd.Env = append(cmd.Env, "AMQ_WEBKEYSTOREPW="+webkeystorePW)
		cmd.Env = append(cmd.Env, "AMQ_WEBTRUSTSTOREREF="+webTruststoreRef)
	}
	out, err := cmd.CombinedOutput()
	rc := cmd.ProcessState.ExitCode()
	if err != nil {
		log.Printf("Error %v starting web server: %v", rc, string(out))
		return err
	}
	log.Println("Started web server")
	return nil
}

func configureWebServer(keyLabel string, p12Truststore tls.KeyStoreData) (string, error) {

	webKeystore := ""

	// Copy server.xml file to ensure that we have the latest expected contents - this file is only populated on QM creation
	err := copy.CopyFile("/opt/mqm/samp/web/server.xml", "/var/mqm/web/installations/Installation1/servers/mqweb/server.xml")
	if err != nil {
		log.Error(err)
		return "", err
	}

	// Configure TLS for the Web Console
	err = tls.ConfigureWebTLS(keyLabel, log)
	if err != nil {
		return "", err
	}

	// Configure the Web Keystore
	if keyLabel != "" || os.Getenv("MQ_GENERATE_CERTIFICATE_HOSTNAME") != "" {
		webKeystore, err = tls.ConfigureWebKeystore(p12Truststore, keyLabel)
		if err != nil {
			return "", err
		}
	}

	_, err = os.Stat("/opt/mqm/bin/strmqweb")
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	const webConfigDir string = "/etc/mqm/web"
	_, err = os.Stat(webConfigDir)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}

	const prefix string = "/etc/mqm/web"
	err = filepath.Walk(prefix, func(from string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		to := fmt.Sprintf("/var/mqm/web%v", from[len(prefix):])
		exists := true
		_, err = os.Stat(to)
		if err != nil {
			if os.IsNotExist(err) {
				exists = false
			} else {
				return err
			}
		}
		if info.IsDir() {
			if !exists {
				// #nosec G301 - write group permissions are required
				err := os.MkdirAll(to, 0770)
				if err != nil {
					return err
				}
			}
		} else {
			if exists {
				err := os.Remove(to)
				if err != nil {
					return err
				}
			}

			// Use a symlink for file 'mqwebuser.xml'
			if strings.HasSuffix(from, "/mqwebuser.xml") {
				err = os.Symlink(from, to)
				if err != nil {
					log.Error(err)
					return err
				}

			} else {
				err := copy.CopyFile(from, to)
				if err != nil {
					log.Error(err)
					return err
				}
			}
		}
		return nil
	})

	return webKeystore, err
}

// Configure FIPS mode for MQ Web Server
func configureFIPSWebServer(p12TrustStore tls.KeyStoreData) error {

	// Need to update jvm.options file of MQ Web Server. We don't update the jvm.options file
	// in /etc/mqm/web/installations/Installation1/servers/mqweb directory. Instead we update
	// the one in /etc/mqm/web/installations/Installation1/servers/mqweb/configDropins/defaults.
	// During runtime MQ Web Server merges the data from two files.
	const jvmOptsLink string = "/run/jvm.options"
	const jvmOptsTemplate string = "/etc/mqm/web/installations/Installation1/servers/mqweb/configDropins/defaults/jvm.options.tpl"

	// Update the jvm.options file using the data from template file. Tell the MQ Web Server
	// use a FIPS provider by setting "-Dcom.ibm.jsse2.usefipsprovider=true" and then tell it
	// use a specific FIPS provider by setting "Dcom.ibm.jsse2.usefipsProviderName=IBMJCEPlusFIPS".
	err := mqtemplate.ProcessTemplateFile(jvmOptsTemplate, jvmOptsLink, map[string]string{
		"FipsProvider":     "true",
		"FipsProviderName": "IBMJCEPlusFIPS",
	}, log)

	return err
}
