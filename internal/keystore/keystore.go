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

// Package keystore contains code to create and update keystores
package keystore

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/ibm-messaging/mq-container/internal/command"
)

// KeyStore describes information about a keystore file
type KeyStore struct {
	Filename     string
	Password     string
	keyStoreType string
	command      string
}

// NewJKSKeyStore creates a new Java Key Store, managed by the runmqckm command
func NewJKSKeyStore(filename, password string) *KeyStore {
	return &KeyStore{
		Filename:     filename,
		Password:     password,
		keyStoreType: "jks",
		command:      "/opt/mqm/bin/runmqckm",
	}
}

// NewCMSKeyStore creates a new MQ CMS Key Store, managed by the runmqakm command
func NewCMSKeyStore(filename, password string) *KeyStore {
	return &KeyStore{
		Filename:     filename,
		Password:     password,
		keyStoreType: "cms",
		command:      "/opt/mqm/bin/runmqakm",
	}
}

// NewPKCS12KeyStore creates a new PKCS12 Key Store, managed by the runmqakm command
func NewPKCS12KeyStore(filename, password string) *KeyStore {
	return &KeyStore{
		Filename:     filename,
		Password:     password,
		keyStoreType: "p12",
		command:      "/opt/mqm/bin/runmqakm",
	}
}

// Create a key store, if it doesn't already exist
func (ks *KeyStore) Create() error {
	_, err := os.Stat(ks.Filename)
	if err == nil {
		// Keystore already exists so we should refresh it by deleting it.
		extension := filepath.Ext(ks.Filename)
		if ks.keyStoreType == "cms" {
			// Only delete these when we are refreshing the kdb keystore
			stashFile := ks.Filename[0:len(ks.Filename)-len(extension)] + ".sth"
			rdbFile := ks.Filename[0:len(ks.Filename)-len(extension)] + ".rdb"
			crlFile := ks.Filename[0:len(ks.Filename)-len(extension)] + ".crl"
			err = os.Remove(stashFile)
			if err != nil {
				return err
			}
			err = os.Remove(rdbFile)
			if err != nil {
				return err
			}
			err = os.Remove(crlFile)
			if err != nil {
				return err
			}
		}
		err = os.Remove(ks.Filename)
		if err != nil {
			return err
		}
	} else if !os.IsNotExist(err) {
		// If the keystore exists but cannot be accessed then return the error
		return err
	}

	// Create the keystore now we're sure it doesn't exist
	out, _, err := command.Run(ks.command, "-keydb", "-create", "-type", ks.keyStoreType, "-db", ks.Filename, "-pw", ks.Password, "-stash")
	if err != nil {
		return fmt.Errorf("error running \"%v -keydb -create\": %v %s", ks.command, err, out)
	}

	mqmUID, mqmGID, err := command.LookupMQM()
	if err != nil {
		return err
	}
	err = os.Chown(ks.Filename, mqmUID, mqmGID)
	if err != nil {
		return err
	}
	return nil
}

// CreateStash creates a key stash, if it doesn't already exist
func (ks *KeyStore) CreateStash() error {
	extension := filepath.Ext(ks.Filename)
	stashFile := ks.Filename[0:len(ks.Filename)-len(extension)] + ".sth"
	_, err := os.Stat(stashFile)
	if err != nil {
		if os.IsNotExist(err) {
			out, _, err := command.Run(ks.command, "-keydb", "-stashpw", "-type", ks.keyStoreType, "-db", ks.Filename, "-pw", ks.Password)
			if err != nil {
				return fmt.Errorf("error running \"%v -keydb -stashpw\": %v %s", ks.command, err, out)
			}
		}
		return err
	}
	mqmUID, mqmGID, err := command.LookupMQM()
	if err != nil {
		return err
	}
	err = os.Chown(stashFile, mqmUID, mqmGID)
	if err != nil {
		return err
	}
	return nil
}

// Import imports a certificate file in the keystore
func (ks *KeyStore) Import(inputFile, password string) error {
	out, _, err := command.Run(ks.command, "-cert", "-import", "-file", inputFile, "-pw", password, "-target", ks.Filename, "-target_pw", ks.Password, "-target_type", ks.keyStoreType)
	if err != nil {
		return fmt.Errorf("error running \"%v -cert -import\": %v %s", ks.command, err, out)
	}
	return nil
}

// CreateSelfSignedCertificate creates a self-signed certificate in the keystore
func (ks *KeyStore) CreateSelfSignedCertificate(label, dn, hostname string) error {
	out, _, err := command.Run(ks.command, "-cert", "-create", "-db", ks.Filename, "-pw", ks.Password, "-label", label, "-dn", dn, "-san_dnsname", hostname, "-size 2048 -sig_alg sha256 -eku serverAuth")
	if err != nil {
		return fmt.Errorf("error running \"%v -cert -create\": %v %s", ks.command, err, out)
	}
	return nil
}

// Add adds a CA certificate to the keystore
func (ks *KeyStore) Add(inputFile, label string) error {
	out, _, err := command.Run(ks.command, "-cert", "-add", "-db", ks.Filename, "-type", ks.keyStoreType, "-pw", ks.Password, "-file", inputFile, "-label", label)
	if err != nil {
		return fmt.Errorf("error running \"%v -cert -add\": %v %s", ks.command, err, out)
	}
	return nil
}

// Add adds a CA certificate to the keystore
func (ks *KeyStore) AddNoLabel(inputFile string) error {
	out, _, err := command.Run(ks.command, "-cert", "-add", "-db", ks.Filename, "-type", ks.keyStoreType, "-pw", ks.Password, "-file", inputFile)
	if err != nil {
		return fmt.Errorf("error running \"%v -cert -add\": %v %s", ks.command, err, out)
	}
	return nil
}

// GetCertificateLabels returns the labels of all certificates in the key store
func (ks *KeyStore) GetCertificateLabels() ([]string, error) {
	out, _, err := command.Run(ks.command, "-cert", "-list", "-type", ks.keyStoreType, "-db", ks.Filename, "-pw", ks.Password)
	if err != nil {
		return nil, fmt.Errorf("error running \"%v -cert -list\": %v %s", ks.command, err, out)
	}
	scanner := bufio.NewScanner(strings.NewReader(out))
	var labels []string
	for scanner.Scan() {
		s := scanner.Text()
		if strings.HasPrefix(s, "-") || strings.HasPrefix(s, "*-") {
			s := strings.TrimLeft(s, "-*")
			labels = append(labels, strings.TrimSpace(s))
		}
	}
	err = scanner.Err()
	if err != nil {
		return nil, err
	}
	return labels, nil
}

// RenameCertificate renames the specified certificate
func (ks *KeyStore) RenameCertificate(from, to string) error {
	if ks.command == "/opt/mqm/bin/runmqakm" {
		// runmqakm can't handle certs with ' in them so just use capicmd
		// Overriding gosec here as this function is in an internal package and only callable by our internal functions.
		// #nosec G204
		cmd := exec.Command("/opt/mqm/gskit8/bin/gsk8capicmd_64", "-cert", "-rename", "-db", ks.Filename, "-pw", ks.Password, "-label", from, "-new_label", to)
		cmd.Env = append(os.Environ(), "LD_LIBRARY_PATH=/opt/mqm/gskit8/lib64/:/opt/mqm/gskit8/lib")
		out, _, err := command.RunCmd(cmd)
		if err != nil {
			return fmt.Errorf("error running \"%v -cert -rename\": %v %s", "/opt/mqm/gskit8/bin/gsk8capicmd_64", err, out)
		}
	} else {
		out, _, err := command.Run(ks.command, "-cert", "-rename", "-db", ks.Filename, "-pw", ks.Password, "-label", from, "-new_label", to)
		if err != nil {
			return fmt.Errorf("error running \"%v -cert -rename\": %v %s", ks.command, err, out)
		}
	}

	return nil
}

// ListAllCertificates Lists all certificates in the keystore
func (ks *KeyStore) ListAllCertificates() ([]string, error) {
	out, _, err := command.Run(ks.command, "-cert", "-list", "-type", ks.keyStoreType, "-db", ks.Filename, "-pw", ks.Password)
	if err != nil {
		return nil, fmt.Errorf("error running \"%v -cert -list\": %v %s", ks.command, err, out)
	}
	scanner := bufio.NewScanner(strings.NewReader(out))
	var labels []string
	for scanner.Scan() {
		s := scanner.Text()
		if strings.HasPrefix(s, "-") || strings.HasPrefix(s, "*-") || strings.HasPrefix(s, "!") {
			s := strings.TrimLeft(s, "-*!")
			labels = append(labels, strings.TrimSpace(s))
		}
	}
	err = scanner.Err()
	if err != nil {
		return nil, err
	}
	return labels, nil
}
