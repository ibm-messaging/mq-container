/*
© Copyright IBM Corporation 2018, 2023

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

	"github.com/ibm-messaging/mq-container/internal/mqtemplate"
)

func updateMQSC(appPasswordRequired bool) error {

	var checkClient string
	if appPasswordRequired {
		checkClient = "REQUIRED"
	} else {
		checkClient = "ASQMGR"
	}

	const mqscLink string = "/run/10-dev.mqsc"
	const mqscTemplate string = "/etc/mqm/10-dev.mqsc.tpl"

	if os.Getenv("MQ_DEV") == "true" {
		// Re-configure channel if app password not set
		err := mqtemplate.ProcessTemplateFile(mqscTemplate, mqscLink, map[string]string{"ChckClnt": checkClient}, log)
		if err != nil {
			return err
		}
	}

	return nil
}
