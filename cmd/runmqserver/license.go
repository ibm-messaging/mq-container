/*
© Copyright IBM Corporation 2017

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
	"errors"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
)

// resolveLicenseFile returns the file name of the MQ license file, taking into
// account the language set by the LANG environment variable
func resolveLicenseFile() string {
	lang, ok := os.LookupEnv("LANG")
	if !ok {
		return "English.txt"
	}
	switch {
	case strings.HasPrefix(lang, "zh_TW"):
		return "Chinese_TW.txt"
	case strings.HasPrefix(lang, "zh"):
		return "Chinese.txt"
	case strings.HasPrefix(lang, "cs"):
		return "Czech.txt"
	case strings.HasPrefix(lang, "fr"):
		return "French.txt"
	case strings.HasPrefix(lang, "de"):
		return "German.txt"
	case strings.HasPrefix(lang, "el"):
		return "Greek.txt"
	case strings.HasPrefix(lang, "id"):
		return "Indonesian.txt"
	case strings.HasPrefix(lang, "it"):
		return "Italian.txt"
	case strings.HasPrefix(lang, "ja"):
		return "Japanese.txt"
	case strings.HasPrefix(lang, "ko"):
		return "Korean.txt"
	case strings.HasPrefix(lang, "lt"):
		return "Lithuanian.txt"
	case strings.HasPrefix(lang, "pl"):
		return "Polish.txt"
	case strings.HasPrefix(lang, "pt"):
		return "Portugese.txt"
	case strings.HasPrefix(lang, "ru"):
		return "Russian.txt"
	case strings.HasPrefix(lang, "sl"):
		return "Slovenian.txt"
	case strings.HasPrefix(lang, "es"):
		return "Spanish.txt"
	case strings.HasPrefix(lang, "tr"):
		return "Turkish.txt"
	}
	return "English.txt"
}

func checkLicense() (bool, error) {
	lic, ok := os.LookupEnv("LICENSE")
	switch {
	case ok && lic == "accept":
		return true, nil
	case ok && lic == "view":
		file := filepath.Join("/opt/mqm/licenses", resolveLicenseFile())
		buf, err := ioutil.ReadFile(file)
		if err != nil {
			log.Println(err)
			return false, err
		}
		log.Println(string(buf))
		return false, nil
	}
	log.Println("Error: Set environment variable LICENSE=accept to indicate acceptance of license terms and conditions.")
	log.Println("License agreements and information can be viewed by setting the environment variable LICENSE=view.  You can also set the LANG environment variable to view the license in a different language.")
	return false, errors.New("Set environment variable LICENSE=accept to indicate acceptance of license terms and conditions")
}
