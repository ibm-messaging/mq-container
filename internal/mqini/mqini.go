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
	"bufio"
	"path/filepath"
	"strings"
	"io/ioutil"
	"os"
	"regexp"
	"errors"

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

var isUnitTest bool

func SetUnitTestFlag(){
	isUnitTest = true
}

var qmconfigStr string

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

	qm, err := GetQueueManager(qmname)
	if err != nil {
		return err
	}

    //Find the ini file
    files := getIniFileList()
	
	//If we are given ini file, read it.
    for _, infile := range files {
        iniFileBytes, err := ioutil.ReadFile(infile)
		if err != nil {
			return err
		}
		userconfig := string(iniFileBytes)

	    //No ini file supplied, so nothing to do.
	    if len(userconfig) == 0 {
		    continue
	    } else {
			//find the corresponding qmgrs config file.
			inifilepath := GetIniFilePath(infile, qm)
			if err != nil {
				return err
			}
			iniFileBytes, err := ioutil.ReadFile(inifilepath)
			if err != nil {
				return err
			}
			//read the initial version.
			qmconfigStr = string(iniFileBytes)
			if err != nil {
				return err
			}
			//Update the qmgr ini file with user config.
            WriteToIniFile(userconfig, inifilepath)
        }
	}
	return nil
}

// read through /etc/ and check if user provided any .ini file 
// to update.
func getIniFileList() []string {

    fileList := []string{}
    filepath.Walk("/etc", func(path string, f os.FileInfo, err error) error {
        if strings.HasSuffix(path, ".ini") {
            fileList = append(fileList, path)
            return nil
		}
		return nil       
	})
	return nil
}

// Based on the ini file(qm.ini or mqs.ini or mqat.ini), return corresponding
// qmgr's ini file path
func GetIniFilePath(inifilename string, qm *QueueManager) string {
	var inipath string

    if strings.HasSuffix(inifilename, "qm.ini") {
		return filepath.Join(qm.Prefix, "qmgrs", qm.Directory, "qm.ini")
    } else if strings.HasSuffix(inifilename, "mqs.ini") {
		return filepath.Join(qm.Prefix,"/mqs.ini")
    } else if strings.HasSuffix(inifilename, "mqat.ini") {
		return filepath.Join(qm.Prefix,"/mqat.ini")
	} 
	return inipath
}

func SetQMConfigStr(config string) {
	qmconfigStr = config
}
func GetQMConfigStr() string {
	return qmconfigStr
}

func WriteToIniFile(userconfig string, inifilepath string) error {

	stanzaList := make(map[string]strings.Builder)
	var sbAppend strings.Builder
	var sbMerger strings.Builder

    //No ini file supplied, so nothing to do.
	if len(userconfig) == 0 {
		return errors.New("User config supplied was empty")
	}

    scanner := bufio.NewScanner(strings.NewReader(userconfig))
	scanner.Split(bufio.ScanLines)
    consumetoAppend := false
    consumeToMerge := false
	var stanza string

    //read through the user file and prepare what we want.
	for scanner.Scan() {
		if (strings.Contains(scanner.Text(), ":")) {
            consumetoAppend=false
			consumeToMerge=false			
			stanza = scanner.Text()
			
            //check if this stanza exists in the qm.ini
            if strings.Contains(qmconfigStr,stanza){
				consumeToMerge=true
				sbMerger = strings.Builder{}
				stanzaList[stanza]= sbMerger
			} else {
                sbAppend.WriteString(stanza+"\n")
                consumetoAppend=true
            }
        } else {
            if consumetoAppend {
                sbAppend.WriteString(scanner.Text()+"\n")
            }
            if consumeToMerge {
				sb := stanzaList[stanza]
				sb.WriteString(scanner.Text()+"\n")
				stanzaList[stanza]=sb
            }
        }
    }

	//merge if stanza exits.
	if len(stanzaList) > 0 {
		for key, _ := range stanzaList {
			attrList := stanzaList[key]
			lineScanner := bufio.NewScanner(strings.NewReader(attrList.String()))
			lineScanner.Split(bufio.ScanLines)
			for lineScanner.Scan() {
		                                                   		attrLine := lineScanner.Text()
				keyvalue := strings.Split(attrLine,"=")
				//this line present in qm.ini, update value.
				if strings.Contains(qmconfigStr, keyvalue[0]) {
					re := regexp.MustCompile(keyvalue[0]+"=.*")
					qmconfigStr = re.ReplaceAllString(qmconfigStr, attrLine)
				} else { //this line not present in qm.ini file, add it.
					re := regexp.MustCompile(key)
					newVal := key+"\n"+attrLine
					qmconfigStr = re.ReplaceAllString(qmconfigStr, newVal)
				}
			}
		}
	}

	//append if stanza doesn't exist.
	if len(sbAppend.String()) > 0 {
       qmconfigStr = qmconfigStr + sbAppend.String()
	}

	//If this is a unit-test call, we don't write, just return.
	if isUnitTest {
		return nil
	}

	//all done - now write the qm config.
	err := ioutil.WriteFile(inifilepath, []byte(qmconfigStr), 0644)
	if err != nil {
		return err
	}
	return nil
}
