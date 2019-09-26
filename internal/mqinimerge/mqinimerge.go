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

// Package mqinimerge merges user-supplied INI files into qm.ini and mqat.ini
package mqinimerge

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
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

	//read user supplied config file.
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
	qminiConfiglist, qmatConfiglist, err := PrepareConfigStanzasToWrite(userconfig)
	if err != nil {
		return err
	}
	err = writeConfigStanzas(qminiConfiglist, qmatConfiglist)
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
func PrepareConfigStanzasToWrite(userconfig string) ([]string, []string, error) {
	var qminiConfigStr string
	var mqatiniConfigStr string

	//read the initial version.
	// #nosec G304 - qmgrDir filepath is derived from dspmqinf
	iniFileBytes, err := ioutil.ReadFile(filepath.Join(qmgrDir, "qm.ini"))
	if err != nil {
		return nil, nil, err
	}
	qminiConfigStr = string(iniFileBytes)
	qminiConfiglist := strings.Split(qminiConfigStr, "\n")

	// #nosec G304 - qmgrDir filepath is derived from dspmqinf
	iniFileBytes, err = ioutil.ReadFile(filepath.Join(qmgrDir, "mqat.ini"))
	if err != nil {
		return nil, nil, err
	}
	mqatiniConfigStr = string(iniFileBytes)
	qmatConfiglist := strings.Split(mqatiniConfigStr, "\n")

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
		//if this is comment or an empty line, ignore it.
		if strings.HasPrefix(scanner.Text(), "#") || len(strings.TrimSpace(scanner.Text())) == 0 {
			continue
		}
		//thumb rule - all stanzas have ":".
		if strings.Contains(scanner.Text(), ":") {
			stanza = strings.TrimSpace(scanner.Text())
			consumetoAppend = false
			consumeToMerge = false

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
					return nil, nil, err
				}
				stanzaListAppend[stanza] = sb
			}
			if consumeToMerge {
				sb := stanzaListMerge[stanza]
				_, err := sb.WriteString(scanner.Text() + "\n")
				if err != nil {
					return nil, nil, err
				}
				stanzaListMerge[stanza] = sb
			}
		}
	}

	// do merge.
	if len(stanzaListMerge) > 0 {
		for key := range stanzaListMerge {
			toWrite, filename := ValidateStanzaToWrite(key)
			if toWrite {
				attrList := stanzaListMerge[key]
				switch filename {
				case "qm.ini":
					qminiConfiglist, err = prepareStanzasToMerge(key, attrList, qminiConfiglist)
					if err != nil {
						return nil, nil, err
					}
				case "mqat.ini":
					qmatConfiglist, err = prepareStanzasToMerge(key, attrList, qmatConfiglist)
					if err != nil {
						return nil, nil, err
					}
				default:
				}
			}
		}
	}

	// do append.
	if len(stanzaListAppend) > 0 {
		for key := range stanzaListAppend {
			attrList := stanzaListAppend[key]
			if strings.Contains(strings.Join(stanzasMQATINI, ", "), strings.TrimSuffix(strings.TrimSpace(key), ":")) {
				qmatConfiglist = prepareStanzasToAppend(key, attrList, qmatConfiglist)
			} else {
				qminiConfiglist = prepareStanzasToAppend(key, attrList, qminiConfiglist)
			}
		}
	}

	return qminiConfiglist, qmatConfiglist, nil
}

// ValidateStanzaToWrite validates stanza to be written and the file it belongs to.
func ValidateStanzaToWrite(stanza string) (bool, string) {
	stanza = strings.TrimSuffix(strings.TrimSpace(stanza), ":")
	if strings.Contains(strings.Join(stanzasQMINI, ", "), stanza) {
		return true, "qm.ini"
	} else if strings.Contains(strings.Join(stanzasMQATINI, ", "), stanza) {
		return true, "mqat.ini"
	} else {
		return false, ""
	}
}

// prepareStanzasToAppend Prepares list of stanzas that are to be appended into qm ini files(qm.ini/mqat.ini)
func prepareStanzasToAppend(key string, attrList strings.Builder, iniConfigList []string) []string {
	newVal := key + "\n" + attrList.String()
	list := strings.Split(newVal, "\n")
	iniConfigList = append(iniConfigList, list...)
	return iniConfigList
}

// prepareStanzasToMerge Prepares list of stanzas that are to be updated into qm ini files(qm.ini/mqat.ini)
// These stanzas are already present in mq ini files and their values have to be updated with user supplied ini.
func prepareStanzasToMerge(key string, attrList strings.Builder, iniConfigList []string) ([]string, error) {

	pos := -1
	//find the index of current stanza in qm's ini file.
	for i := 0; i < len(iniConfigList); i++ {
		if strings.Contains(iniConfigList[i], key) {
			pos = i
			break
		}
	}

	var appList strings.Builder
	lineScanner := bufio.NewScanner(strings.NewReader(attrList.String()))
	lineScanner.Split(bufio.ScanLines)

	//Now go through the array and merge the values.
	for lineScanner.Scan() {
		attrLine := lineScanner.Text()
		keyvalue := strings.Split(attrLine, "=")
		merged := false
		for i := pos + 1; i < len(iniConfigList); i++ {
			if strings.HasPrefix(iniConfigList[i], "#") {
				continue
			}
			if strings.Contains(iniConfigList[i], ":") {
				break
			}
			if strings.Contains(iniConfigList[i], keyvalue[0]) {
				iniConfigList[i] = attrLine
				merged = true
				break
			}
		}
		//If this is not merged, then its a new parameter in existing stanza.
		if !merged && len(strings.TrimSpace(attrLine)) > 0 {
			_, err := appList.WriteString(attrLine)
			if err != nil {
				return nil, err
			}
			merged = false
		}

		if len(appList.String()) > 0 {
			temp := make([]string, pos+1)
			for i := 0; i < pos+1; i++ {
				temp[i] = iniConfigList[i]
			}
			list := strings.Split(appList.String(), "\n")
			temp = append(temp, list...)
			temp1 := iniConfigList[pos+1:]
			iniConfigList = append(temp, temp1...)
		}

	}
	return iniConfigList, nil
}

// writeFileIfChanged writes the specified data to the specified file path
// (just like ioutil.WriteFile), but first checks if this is needed
func writeFileIfChanged(path string, data []byte, perm os.FileMode) error {
	// #nosec G304 - internal utility using file name derived from dspmqinf
	current, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	// Only write the new file if the it's different from the current file
	if !bytes.Equal(current, data) {
		err = ioutil.WriteFile(path, data, perm)
		if err != nil {
			return err
		}
	}
	return nil
}

// writeConfigStanzas writes the INI file updates into corresponding MQ INI files.
func writeConfigStanzas(qmConfig []string, atConfig []string) error {
	err := writeFileIfChanged(filepath.Join(qmgrDir, "qm.ini"), []byte(strings.Join(qmConfig, "\n")), 0644)
	if err != nil {
		return err
	}

	err = writeFileIfChanged(filepath.Join(qmgrDir, "mqat.ini"), []byte(strings.Join(atConfig, "\n")), 0644)
	if err != nil {
		return err
	}
	return nil
}
