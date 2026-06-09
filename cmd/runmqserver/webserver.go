/*
© Copyright IBM Corporation 2018, 2026

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

func startWebServer() error {
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

	out, err := cmd.CombinedOutput()
	rc := cmd.ProcessState.ExitCode()
	if err != nil {
		log.Printf("Error %v starting web server: %v", rc, string(out))
		return err
	}
	log.Println("Started web server")
	return nil
}

func configureWebServer(keyLabel string, p12Truststore tls.KeyStoreData) error {

	webKeystore := ""

	// Fix for breaking change introduced for WebSphere Liberty 'ltpa.keys' file
	if strings.ToLower(os.Getenv("AMQ_ENABLE_DT472893_REGENERATE_LTPA_KEYS")) == "true" {
		err := ltpaKeysFileFix()
		if err != nil {
			return err
		}
	}

	// Copy server.xml file to ensure that we have the latest expected contents - this file is only populated on QM creation
	err := copy.CopyFile("/opt/mqm/samp/web/server.xml", "/var/mqm/web/installations/Installation1/servers/mqweb/server.xml")
	if err != nil {
		log.Error(err)
		return err
	}

	// Configure the Web Keystore
	if keyLabel != "" || os.Getenv("MQ_GENERATE_CERTIFICATE_HOSTNAME") != "" {
		webKeystore, err = tls.ConfigureWebKeystore(p12Truststore, keyLabel)
		if err != nil {
			return err
		}
	}

	// Configure TLS for the Web Console
	err = tls.ConfigureWebTLS(keyLabel, webKeystore, p12Truststore, log)
	if err != nil {
		return err
	}

	_, err = os.Stat("/opt/mqm/bin/strmqweb")
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	const webConfigDir string = "/etc/mqm/web"
	_, err = os.Stat(webConfigDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
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

			// Use a symlink for file 'mqwebuser.xml', so that if it contains a secret, it doesn't get persisted to a volume
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

	return err
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

// ltpaKeysFileFix fixes the breaking change introduced for WebSphere Liberty 'ltpa.keys' file
// - Returns an error if unable to successfully complete the fix
func ltpaKeysFileFix() error {

	const ltpaKeysFile string = "/var/mqm/web/installations/Installation1/servers/mqweb/resources/security/ltpa.keys"
	const refreshedFile string = "/var/mqm/web/installations/Installation1/servers/mqweb/mq-DT472893-ltpa-keys.refreshed"

	// Check if 'ltpa.keys' file exists
	ltpaKeysFileExists, ltpaKeysFileErr := fileExists(ltpaKeysFile)
	if ltpaKeysFileExists {

		// The 'ltpa.keys' file exists
		// - Check if it has already been refreshed - indicated by presence of marker file
		refreshedFileExists, refreshedFileErr := fileExists(refreshedFile)
		if refreshedFileErr == nil && !refreshedFileExists {

			// The marker file does not exist - refresh the 'ltpa.keys' file
			// - Remove existing file - it will be regenerated by WebSphere Liberty using the new ltpaKeysPassword property
			err := os.Remove(ltpaKeysFile)
			if err != nil {
				log.Printf("Error removing file: '%v': %v", ltpaKeysFile, err)
				return err
			}
			log.Print("Removed existing file 'ltpa.keys' - it will be regenerated by WebServer")

			// Write marker file to indicate that we have refreshed the 'ltpa.keys' file
			writeDT472893MarkerFile(refreshedFile)

		} else if refreshedFileErr != nil {

			// Unable to determine if 'ltpa.keys' file has already been refreshed
			log.Printf("Error getting marker file: '%v': %v", refreshedFile, refreshedFileErr)
			return refreshedFileErr
		}

	} else if ltpaKeysFileErr == nil {

		// The 'ltpa.keys' file does not exist - it will be generated by WebSphere Liberty using the new ltpaKeysPassword property
		// - Write marker file to prevent a refresh on next container restart
		writeDT472893MarkerFile(refreshedFile)

	} else {

		// Unable to determine if 'ltpa.keys' file exists
		log.Printf("Error getting file: '%v': %v", ltpaKeysFile, ltpaKeysFileErr)
		return ltpaKeysFileErr
	}

	return nil
}

// writeDT472893MarkerFile writes the marker file for known issue DT472893
func writeDT472893MarkerFile(refreshedFile string) {

	// #nosec G306 - required permissions
	err := os.WriteFile(refreshedFile, []byte(""), 0660)
	if err == nil {
		log.Print("Created marker file 'mq-DT472893-ltpa-keys.refreshed'")
	} else {
		log.Printf("Error writing marker file: '%v' - 'ltpa.keys' file will be regenerated on next container restart: %v", refreshedFile, err)
	}
}

// fileExists returns true if the specified file exists, otherwise false
// - Returns an error if there was an issue determining existence of the file
func fileExists(fileName string) (bool, error) {
	_, err := os.Stat(fileName)
	if err != nil {
		if !os.IsNotExist(err) {
			return false, err
		}
		return false, nil
	}
	return true, nil
}
