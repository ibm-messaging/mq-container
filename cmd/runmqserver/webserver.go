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
	"os/exec"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/ibm-messaging/mq-container/internal/command"
	"github.com/ibm-messaging/mq-container/internal/copy"
	"github.com/ibm-messaging/mq-container/internal/mqtemplate"
	"github.com/ibm-messaging/mq-container/internal/tls"
)

func startWebServer(keystore, keystorepw string) error {
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
	if keystore != "" {
		cmd.Env = append(cmd.Env, "AMQ_WEBKEYSTORE="+keystore)
		cmd.Env = append(cmd.Env, "AMQ_WEBKEYSTOREPW="+keystorepw)
	}

	uid, gid, err := command.LookupMQM()
	if err != nil {
		return err
	}
	u, err := user.Current()
	if err != nil {
		return err
	}
	currentUID, err := strconv.Atoi(u.Uid)
	if err != nil {
		return fmt.Errorf("Error converting UID to string: %v", err)
	}
	// Add credentials to run as 'mqm', only if we aren't already 'mqm'
	if currentUID != uid {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
		cmd.SysProcAttr.Credential = &syscall.Credential{Uid: uint32(uid), Gid: uint32(gid)}
	}
	out, rc, err := command.RunCmd(cmd)
	if err != nil {
		log.Printf("Error %v starting web server: %v", rc, string(out))
		return err
	}
	log.Println("Started web server")
	return nil
}

func configureSSO(p12TrustStore tls.KeyStoreData) (string, error) {
	// Ensure all required environment variables are set for SSO
	requiredEnvVars := []string{
		"MQ_WEB_ADMIN_USERS",
		"MQ_OIDC_CLIENT_ID",
		"MQ_OIDC_CLIENT_SECRET",
		"MQ_OIDC_UNIQUE_USER_IDENTIFIER",
		"MQ_OIDC_AUTHORIZATION_ENDPOINT",
		"MQ_OIDC_TOKEN_ENDPOINT",
		"MQ_OIDC_JWK_ENDPOINT",
		"MQ_OIDC_ISSUER_IDENTIFIER",
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
	return configureSSOTLS(p12TrustStore)
}

func configureWebServer(keyLabel string, p12Trust tls.KeyStoreData) (string, error) {
	var keystore string
	// Configure TLS for Web Console first if we have a certificate to use
	err := configureWebTLS(keyLabel)
	if err != nil {
		return keystore, err
	}
	if keyLabel != "" {
		keystore = keyLabel + ".p12"
	}

	// Configure Single-Sign-On for the web server (if enabled)
	enableSSO := os.Getenv("MQ_BETA_ENABLE_SSO")
	if enableSSO == "true" || enableSSO == "1" {
		keystore, err = configureSSO(p12Trust)
		if err != nil {
			return keystore, err
		}
	}
	_, err = os.Stat("/opt/mqm/bin/strmqweb")
	if err != nil {
		if os.IsNotExist(err) {
			return keystore, nil
		}
		return keystore, err
	}
	const webConfigDir string = "/etc/mqm/web"
	_, err = os.Stat(webConfigDir)
	if err != nil {
		if os.IsNotExist(err) {
			return keystore, nil
		}
		return keystore, err
	}
	uid, gid, err := command.LookupMQM()
	if err != nil {
		return keystore, err
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
		err = os.Chown(to, uid, gid)
		if err != nil {
			return err
		}
		return nil
	})
	return keystore, err
}
