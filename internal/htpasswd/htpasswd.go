/*
© Copyright IBM Corporation 2020, 2021

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

//This is a developer only configuration and not recommended for production usage.

package htpasswd

import (
	"fmt"
	"io/ioutil"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

type mapHtPasswd map[string]string

func encryptPassword(password string) (string, error) {
	passwordBytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(passwordBytes), nil
}

// SetPassword sets encrypted password for the user into htpasswd file
func SetPassword(user string, password string, isTest bool) error {

	if len(strings.TrimSpace(user)) == 0 || len(strings.TrimSpace(password)) == 0 {
		return fmt.Errorf("UserId or Password are empty")
	}

	passwords := mapHtPasswd(map[string]string{})

	// Read the password file
	err := passwords.ReadHtPasswordFile(isTest)
	if err != nil {
		return err
	}

	pwd, err := encryptPassword(password)
	if err != nil {
		return err
	}
	// Set the new password
	passwords[user] = pwd

	// Update the password file
	return passwords.updateHtPasswordFile(isTest)
}

// GetBytes return the Bytes representation of the htpassword file
func (htpfile mapHtPasswd) GetBytes() (passwordBytes []byte) {
	passwordBytes = []byte{}
	for name, hash := range htpfile {
		passwordBytes = append(passwordBytes, []byte(name+":"+hash+"\n")...)
	}
	return passwordBytes
}

// ReadHtPasswordFile parses the htpasswd file
func (htpfile mapHtPasswd) ReadHtPasswordFile(isTest bool) error {

	file := "/etc/mqm/mq.htpasswd"
	if isTest {
		file = "my.htpasswd"
	}

	pwdsBytes, err := ioutil.ReadFile(file)
	if err != nil {
		return err
	}

	lines := strings.Split(string(pwdsBytes), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if len(line) == 0 {
			continue
		}
		parts := strings.Split(line, ":")
		if len(parts) != 2 {
			continue
		}
		for i, part := range parts {
			parts[i] = strings.TrimSpace(part)
		}
		htpfile[parts[0]] = parts[1]
	}
	return nil
}

func (htpfile mapHtPasswd) updateHtPasswordFile(isTest bool) error {

	file := "/etc/mqm/mq.htpasswd"
	if isTest {
		file = "my.htpasswd"
	}
	return ioutil.WriteFile(file, htpfile.GetBytes(), 0660)
}
