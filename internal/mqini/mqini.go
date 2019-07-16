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
package mqini

import (
	"fmt"
	"bufio"
	"path/filepath"
	"strings"
	"io/ioutil"
	"os"

	"github.com/ibm-messaging/mq-container/internal/command"
)

// QueueManager describe high-level configuration information for a queue manager
type QueueManager struct {
	Name             string
	Prefix           string
	Directory        string
	DataPath         string
	InstallationName string
}

// getQueueManagerFromStanza parses a queue manager stanza
func getQueueManagerFromStanza(stanza string) (*QueueManager, error) {
	scanner := bufio.NewScanner(strings.NewReader(stanza))
	qm := QueueManager{}
	for scanner.Scan() {
		l := scanner.Text()
		l = strings.TrimSpace(l)
		t := strings.Split(l, "=")
		switch t[0] {
		case "Name":
			qm.Name = t[1]
		case "Prefix":
			qm.Prefix = t[1]
		case "Directory":
			qm.Directory = t[1]
		case "DataPath":
			qm.DataPath = t[1]
		case "InstallationName":
			qm.InstallationName = t[1]
		}
	}
	return &qm, scanner.Err()
}

// GetQueueManager returns queue manager configuration information
func GetQueueManager(name string) (*QueueManager, error) {
	// dspmqinf essentially returns a subset of mqs.ini, but it's simpler to parse
	out, _, err := command.Run("dspmqinf", "-o", "stanza", name)
	if err != nil {
		return nil, err
	}
	return getQueueManagerFromStanza(out)
}

// GetErrorLogDirectory returns the directory holding the error logs for the
// specified queue manager
func GetErrorLogDirectory(qm *QueueManager) string {
	if qm.DataPath != "" {
		return filepath.Join(qm.DataPath, "errors")
	}
	return filepath.Join(qm.Prefix, "qmgrs", qm.Directory, "errors")
}

//Update qm.ini file with the user supplied stanzas.
func AddStanzas(qmname string) error {
	inifilepath, _ := GetQueueManagerIniFile(qmname)
	
	apiexitStanzaL := make(map[string]string)
	apiexitStanzaQM := make(map[string]string)
	var userconfig string

	//Find the ini file
	files, err := ioutil.ReadDir("/etc/MQOpenTracing/")
    if err != nil {
        return err
	}
	//If we are given ini file, read it.
    for _, infile := range files {
		if strings.HasSuffix(infile.Name(), ".ini") {
			iniFileBytes, err := ioutil.ReadFile(filepath.Join("/etc/MQOpenTracing/",infile.Name()))
			if err != nil {
				return err
			}
			userconfig = string(iniFileBytes)
		}
	}

	//No ini file supplied, so nothing to do.
	if len(userconfig) == 0 {
		return nil
	}
	
	//mat := "ApiExitLocal: \n  Sequence=100 \n  Function=EntryPoint \n  Module=/opt/MQOpenTracing/MQOpenTracingExit.so \n  Name=MQOpenTracingExit \n"
	scanner := bufio.NewScanner(strings.NewReader(userconfig))
	scanner.Split(bufio.ScanLines)
	for scanner.Scan() {
		if (strings.Contains(scanner.Text(), "ApiExitLocal:")) {
			for scanner.Scan() {
				line := scanner.Text()
				if strings.Contains(line,":"){
					break;
				}
				keyvalue := strings.Split(line,"=")
				apiexitStanzaL[keyvalue[0]]=keyvalue[1]
			}
		}
	}

	//Next read from the qm.ini file.
	iniFileBytes,  err := ioutil.ReadFile(inifilepath)
	if err != nil {
		return err
	}
	curFile := string(iniFileBytes)
	if strings.Contains(curFile, "ApiExitLocal:") {
		scanner = bufio.NewScanner(strings.NewReader(curFile))
		scanner.Split(bufio.ScanLines)

		for scanner.Scan() {
			if (strings.Contains(scanner.Text(), "ApiExitLocal:")) {
				for scanner.Scan() {
					line := scanner.Text()
					if strings.Contains(line,":"){
						break;
					}
					keyvalue := strings.Split(line,"=")
					apiexitStanzaQM[keyvalue[0]]=keyvalue[1]
				}
			}
		}

		//Prepare the text to write.
		for key, _ := range apiexitStanzaL {
			_, ok := apiexitStanzaQM[key]
			old := fmt.Sprintf("%s=%s", key, apiexitStanzaQM[key])
			if ok {
				apiexitStanzaQM[key]=apiexitStanzaL[key]
			}
		
			new := fmt.Sprintf("%s=%s", key, apiexitStanzaQM[key])
			curFile = strings.Replace(curFile,old, new, 5)
		}

		//Rewrite qm.ini file
		err := ioutil.WriteFile(inifilepath, []byte(curFile), 0644)
		if err != nil {
			return err
		}
	} else {
		f, err := os.OpenFile(inifilepath, os.O_APPEND|os.O_WRONLY, 0600)
		if err != nil {
			return err
		}
		defer f.Close()
		if _, err = f.WriteString(userconfig); err != nil {
			return err
		}
	}
	return nil
}

//Gets the ini file location
func GetQueueManagerIniFile(qmname string) (string, error) {
	qm, err := GetQueueManager(qmname)
	if err != nil {
		return "", err
	}
	qmINIFile := filepath.Join(qm.Prefix, "qmgrs", qm.Directory, "qm.ini")
	return qmINIFile, nil
}
