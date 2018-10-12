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
	"os"
)

func updateMQSC(appPasswordRequired bool) error {
	var checkClient string
	if appPasswordRequired {
		checkClient = "REQUIRED"
	} else {
		checkClient = "ASQMGR"
	}
	const mqsc string = "/etc/mqm/10-dev.mqsc"
	if os.Getenv("MQ_DEV") == "true" {
		const mqscTemplate string = mqsc + ".tpl"
		// Re-configure channel if app password not set
		err := processTemplateFile(mqsc+".tpl", mqsc, map[string]string{"ChckClnt": checkClient})
		if err != nil {
			return err
		}
	} else {
		_, err := os.Stat(mqsc)
		if !os.IsNotExist(err) {
			err = os.Remove(mqsc)
			if err != nil {
				log.Errorf("Error removing file %s: %v", mqsc, err)
				return err
			}
		}
	}
	return nil
}
