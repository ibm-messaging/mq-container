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
	// Set a default app password for the web server, if one isn't already set
	_, set := os.LookupEnv("MQ_APP_PASSWORD")
	if !set {
		// Take all current environment variables, and add the app password
		cmd.Env = append(os.Environ(), "MQ_APP_PASSWORD=passw0rd")
	} else {
		cmd.Env = os.Environ()
	}

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

func configureSSO(p12TrustStore tls.KeyStoreData, webKeystore string) (string, error) {
	requiredEnvVars := []string{}
	_, set := os.LookupEnv("MQ_ZEN_INTERNAL_ENDPOINT")
	if !set {
		// Ensure all required environment variables are set for SSO
		requiredEnvVars = []string{
			"MQ_OIDC_CLIENT_ID",
			"MQ_OIDC_CLIENT_SECRET",
			"MQ_OIDC_UNIQUE_USER_IDENTIFIER",
			"MQ_OIDC_AUTHORIZATION_ENDPOINT",
			"MQ_OIDC_TOKEN_ENDPOINT",
			"MQ_OIDC_JWK_ENDPOINT",
			"MQ_OIDC_ISSUER_IDENTIFIER",
		}
	} else {
		// Ensure all required environment variables are set for Zen SSO
		requiredEnvVars = []string{
			"MQ_ZEN_UNIQUE_USER_IDENTIFIER",
			"MQ_ZEN_INTERNAL_ENDPOINT",
			"MQ_ZEN_ISSUER_IDENTIFIER",
			"MQ_ZEN_AUDIENCES",
			"MQ_ZEN_CONTEXT_NAME",
			"MQ_ZEN_BASE_URI",
			"MQ_ZEN_CONTEXT_NAMESPACE",
			"IAM_URL",
		}
	}
	for _, envVar := range requiredEnvVars {
		if len(os.Getenv(envVar)) == 0 {
			return "", fmt.Errorf("%v must be set when MQ_BETA_ENABLE_SSO=true", envVar)
		}
	}

	// Check mqweb directory exists
	const mqwebDir string = "/etc/mqm/web/installations/Installation1/servers/mqweb"
	_, err := os.Stat(mqwebDir)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}

	// Process SSO template for generating file mqwebuser.xml
	adminUsers := strings.Split(os.Getenv("MQ_WEB_ADMIN_USERS"), "\n")
	err = mqtemplate.ProcessTemplateFile(mqwebDir+"/mqwebuser.xml.tpl", mqwebDir+"/mqwebuser.xml", map[string][]string{"AdminUser": adminUsers}, log)
	if err != nil {
		return "", err
	}

	// Configure SSO TLS
	return tls.ConfigureWebKeystore(p12TrustStore, webKeystore)
}

func configureWebServer(keyLabel string, p12Truststore tls.KeyStoreData) (string, error) {
	var webKeystore string

	// Configure TLS for Web Console first if we have a certificate to use
	err := tls.ConfigureWebTLS(keyLabel)
	if err != nil {
		return "", err
	}
	if keyLabel != "" {
		webKeystore = keyLabel + ".p12"
	}

	// Configure Single-Sign-On for the web server (if enabled)
	enableSSO := os.Getenv("MQ_BETA_ENABLE_SSO")
	if enableSSO == "true" || enableSSO == "1" {
		webKeystore, err = configureSSO(p12Truststore, webKeystore)
		if err != nil {
			return "", err
		}
	} else if keyLabel == "" && os.Getenv("MQ_GENERATE_CERTIFICATE_HOSTNAME") != "" {
		webKeystore, err = tls.ConfigureWebKeystore(p12Truststore, webKeystore)
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
			err := copy.CopyFile(from, to)
			if err != nil {
				log.Error(err)
				return err
			}
		}
		return nil
	})

	return webKeystore, err
}

// Configure FIPS mode for MQ Web Server
func configureFIPSWebServer(p12TrustStore tls.KeyStoreData) error {
	var errOut error
	// Need to update jvm.options file of MQ Web Server. We don't update the jvm.options file
	// in /var/mqm/web/installations/Installation1/servers/mqweb directory. Instead we update
	// the one in /var/mqm/web/installations/Installation1/servers/mqweb/configDropins/defaults.
	// During runtime MQ Web Server merges the data from two files.
	mqwebJvmOptsDir := "/var/mqm/web/installations/Installation1/servers/mqweb/configDropins/defaults"
	_, errOut = os.Stat(mqwebJvmOptsDir)
	if errOut == nil {
		// Update the jvm.options file using the data from template file. Tell the MQ Web Server
		// use a FIPS provider by setting "-Dcom.ibm.jsse2.usefipsprovider=true" and then tell it
		// use a specific FIPS provider by setting "Dcom.ibm.jsse2.usefipsProviderName=IBMJCEPlusFIPS".
		errOut = mqtemplate.ProcessTemplateFile(mqwebJvmOptsDir+"/jvm.options.tpl",
			mqwebJvmOptsDir+"/jvm.options", map[string]string{
				"FipsProvider":     "true",
				"FipsProviderName": "IBMJCEPlusFIPS",
			}, log)
	}
	return errOut
}
