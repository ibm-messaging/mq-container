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

// Package mqini provides information about queue managers
package mqinimerge

import (
	"bufio"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/ibm-messaging/mq-container/pkg/mqini"
)

var qmgrDir string

var stanzasQMINI []string

var stanzasMQATINI []string

// AddStanzas reads supplied MQ INI configuration files and updates the stanzas
// in the queue manager's INI configuration files.
func AddStanzas(qmname string) error {
	// Find the qmgr directory.
	qm, err := mqini.GetQueueManager(qmname)
	if err != nil {
		return err
	}
	qmgrDir = mqini.GetDataDirectory(qm)
	// Find the users ini configuration file
	files, err := getIniFileList()
	if err != nil {
		return err
	}
	if len(files) > 1 {
		msg := fmt.Sprintf("[ %v ]", files)
		return errors.New("Only a single INI file can be provided. Following INI files were found:" + msg)
	}
	if len(files) == 0 {
		// No INI file update required.
		return nil
	}

	iniFileBytes, err := ioutil.ReadFile(files[0])
	if err != nil {
		return err
	}
	userconfig := string(iniFileBytes)
	if len(userconfig) == 0 {
		return nil
	}

	// Prepare a list of all supported stanzas
	PopulateAllAvailableStanzas()

	// Update the qmgr ini file with user config.
	qmConfig, atConfig, err := PrepareConfigStanzasToWrite(userconfig)
	if err != nil {
		return err
	}
	err = writeConfigStanzas(qmConfig, atConfig)
	if err != nil {
		return err
	}

	return nil
}

// PopulateAllAvailableStanzas initializes the INI stanzas prescribed by MQ specification.
func PopulateAllAvailableStanzas() {
	stanzasQMINI = []string{"ExitPath",
		"Log",
		"Service",
		"ServiceComponent",
		"Channels",
		"TCP",
		"ApiExitLocal",
		"AccessMode",
		"RestrictedMode",
		"XAResourceManager",
		"DefaultBindType",
		"SSL",
		"DiagnosticMessages",
		"Filesystem",
		"Security",
		"TuningParameters",
		"ExitPropertiesLocal",
		"LU62",
		"NETBIOS"}

	stanzasMQATINI = []string{"AllActivityTrace", "ApplicationTrace"}
}

// getIniFileList checks for the user supplied INI file in `/etc/mqm` directory.
func getIniFileList() ([]string, error) {
	fileList := []string{}
	err := filepath.Walk("/etc/mqm", func(path string, f os.FileInfo, err error) error {
		if strings.HasSuffix(path, ".ini") {
			fileList = append(fileList, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return fileList, nil
}

// PrepareConfigStanzasToWrite Reads through the user supplied INI config file and prepares list of
// updates to be written into corresponding mq ini files (qm.ini and/or mqat.ini files)
func PrepareConfigStanzasToWrite(userconfig string) (string, string, error) {
	var qminiConfigStr string
	var mqatiniConfigStr string

	//read the initial version.
	// #nosec G304 - qmgrDir filepath is derived from dspmqinf
	iniFileBytes, err := ioutil.ReadFile(filepath.Join(qmgrDir, "qm.ini"))
	if err != nil {
		return "", "", err
	}
	qminiConfigStr = string(iniFileBytes)

	// #nosec G304 - qmgrDir filepath is derived from dspmqinf
	iniFileBytes, err = ioutil.ReadFile(filepath.Join(qmgrDir, "mqat.ini"))
	if err != nil {
		return "", "", err
	}
	mqatiniConfigStr = string(iniFileBytes)

	stanzaListMerge := make(map[string]strings.Builder)
	stanzaListAppend := make(map[string]strings.Builder)
	var sbAppend strings.Builder
	var sbMerger strings.Builder

	scanner := bufio.NewScanner(strings.NewReader(userconfig))
	scanner.Split(bufio.ScanLines)
	consumetoAppend := false
	consumeToMerge := false
	var stanza string

	// Read through the user file and prepare what we want.
	for scanner.Scan() {
		if strings.Contains(scanner.Text(), ":") {
			consumetoAppend = false
			consumeToMerge = false
			stanza = scanner.Text()
			// Check if this stanza exists in the qm.ini/mqat.ini files
			if strings.Contains(qminiConfigStr, stanza) ||
				(strings.Contains(mqatiniConfigStr, stanza) && !(strings.Contains(stanza, "ApplicationTrace"))) {
				consumeToMerge = true
				sbMerger = strings.Builder{}

				stanzaListMerge[stanza] = sbMerger
			} else {
				consumetoAppend = true
				sbAppend = strings.Builder{}
				stanzaListAppend[stanza] = sbAppend
			}
		} else {
			if consumetoAppend {
				sb := stanzaListAppend[stanza]
				_, err := sb.WriteString(scanner.Text() + "\n")
				if err != nil {
					return "", "", err
				}
				stanzaListAppend[stanza] = sb
			}
			if consumeToMerge {
				sb := stanzaListMerge[stanza]
				_, err := sb.WriteString(scanner.Text() + "\n")
				if err != nil {
					return "", "", err
				}
				stanzaListMerge[stanza] = sb
			}
		}
	}

	// Merge if stanza exits
	if len(stanzaListMerge) > 0 {
		for key := range stanzaListMerge {
			toWrite, filename := ValidateStanzaToWrite(key)
			if toWrite {
				attrList := stanzaListMerge[key]
				switch filename {
				case "qm.ini":
					qminiConfigStr = prepareStanzasToMerge(key, attrList, qminiConfigStr)
				case "mqat.ini":
					mqatiniConfigStr = prepareStanzasToMerge(key, attrList, mqatiniConfigStr)
				default:
				}
			}
		}
	}

	// Append new stanzas.
	if len(stanzaListAppend) > 0 {
		for key := range stanzaListAppend {
			attrList := stanzaListAppend[key]
			if strings.Contains(strings.Join(stanzasMQATINI, ", "), strings.TrimSuffix(strings.TrimSpace(key), ":")) {
				mqatiniConfigStr = prepareStanzasToAppend(key, attrList, mqatiniConfigStr)
			} else {
				qminiConfigStr = prepareStanzasToAppend(key, attrList, qminiConfigStr)
			}
		}
	}

	return qminiConfigStr, mqatiniConfigStr, nil
}

// ValidateStanzaToWrite validates stanza to be written and the file it belongs to.
func ValidateStanzaToWrite(stanza string) (bool, string) {
	stanza = strings.TrimSpace(stanza)
	if strings.Contains(stanza, ":") {
		stanza = stanza[:len(stanza)-1]
	}

	if strings.Contains(strings.Join(stanzasQMINI, ", "), stanza) {
		return true, "qm.ini"
	} else if strings.Contains(strings.Join(stanzasMQATINI, ", "), stanza) {
		return true, "mqat.ini"
	} else {
		return false, ""
	}
}

// prepareStanzasToAppend Prepares list of stanzas that are to be appended into qm ini files(qm.ini/mqat.ini)
func prepareStanzasToAppend(key string, attrList strings.Builder, iniConfig string) string {
	newVal := key + "\n" + attrList.String()
	iniConfig = iniConfig + newVal
	return iniConfig
}

// prepareStanzasToMerge Prepares list of stanzas that are to be updated into qm ini files(qm.ini/mqat.ini)
// These stanzas are already present in mq ini files and their values have to be updated with user supplied ini.
func prepareStanzasToMerge(key string, attrList strings.Builder, iniConfig string) string {
	lineScanner := bufio.NewScanner(strings.NewReader(attrList.String()))
	lineScanner.Split(bufio.ScanLines)
	for lineScanner.Scan() {
		attrLine := lineScanner.Text()
		keyvalue := strings.Split(attrLine, "=")
		// This line present in qm.ini, update value.
		if strings.Contains(iniConfig, keyvalue[0]) {
			re := regexp.MustCompile(keyvalue[0] + "=.*")
			iniConfig = re.ReplaceAllString(iniConfig, attrLine)
		} else {
			// This line not present in qm.ini file, add it.
			re := regexp.MustCompile(key)
			newVal := key + "\n" + attrLine
			iniConfig = re.ReplaceAllString(iniConfig, newVal)
		}
	}
	return iniConfig
}

// writeConfigStanzas writes the INI file updates into corresponding mq ini files.
func writeConfigStanzas(qmConfig string, atConfig string) error {
	err := ioutil.WriteFile(filepath.Join(qmgrDir, "qm.ini"), []byte(qmConfig), 0644)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(filepath.Join(qmgrDir, "mqat.ini"), []byte(atConfig), 0644)
	if err != nil {
		return err
	}
	return nil
}
