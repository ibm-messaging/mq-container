/*
© Copyright IBM Corporation 2018

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
	"os"
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

// Create a key store, if it doesn't already exist
func (ks *KeyStore) Create() error {
	_, err := os.Stat(ks.Filename)
	if err != nil {
		if os.IsNotExist(err) {
			_, _, err := command.Run(ks.command, "-keydb", "-create", "-type", ks.keyStoreType, "-db", ks.Filename, "-pw", ks.Password, "-stash")
			if err != nil {
				return fmt.Errorf("error running \"%v -keydb -create\": %v", ks.command, err)
			}
		}
	}
	// TODO: Lookup value for MQM user here?
	err = os.Chown(ks.Filename, 999, 999)
	if err != nil {
		log.Error(err)
		return err
	}
	return nil
}

// CreateStash creates a key stash, if it doesn't already exist
func (ks *KeyStore) CreateStash() error {
	extension := filepath.Ext(ks.Filename)
	stashFile := ks.Filename[0:len(ks.Filename)-len(extension)] + ".sth"
	log.Debugf("TLS stash file: %v", stashFile)
	_, err := os.Stat(stashFile)
	if err != nil {
		if os.IsNotExist(err) {
			_, _, err := command.Run(ks.command, "-keydb", "-stashpw", "-type", ks.keyStoreType, "-db", ks.Filename, "-pw", ks.Password)
			if err != nil {
				return fmt.Errorf("error running \"%v -keydb -stashpw\": %v", ks.command, err)
			}
		}
		return err
	}
	// TODO: Lookup value for MQM user here?
	err = os.Chown(stashFile, 999, 999)
	if err != nil {
		log.Error(err)
		return err
	}
	return nil
}

// Import imports a certificate file in the keystore
func (ks *KeyStore) Import(inputFile, password string) error {
	_, _, err := command.Run(ks.command, "-cert", "-import", "-file", inputFile, "-pw", password, "-target", ks.Filename, "-target_pw", ks.Password, "-target_type", ks.keyStoreType)
	if err != nil {
		return fmt.Errorf("error running \"%v -cert -import\": %v", ks.command, err)
	}
	return nil
}

// GetCertificateLabels returns the labels of all certificates in the key store
func (ks *KeyStore) GetCertificateLabels() ([]string, error) {
	out, _, err := command.Run(ks.command, "-cert", "-list", "-type", ks.keyStoreType, "-db", ks.Filename, "-pw", ks.Password)
	if err != nil {
		return nil, fmt.Errorf("error running \"%v -cert -list\": %v", ks.command, err)
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
	_, _, err := command.Run(ks.command, "-cert", "-rename", "-db", ks.Filename, "-pw", ks.Password, "-label", from, "-new_label", to)
	if err != nil {
		return fmt.Errorf("error running \"%v -cert -rename\": %v", ks.command, err)
	}
	return nil
}
