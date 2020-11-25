/*
© Copyright IBM Corporation 2020

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

// Package ha contains code for high availability
package ha

import (
	"os"

	"github.com/ibm-messaging/mq-container/internal/mqtemplate"
	"github.com/ibm-messaging/mq-container/pkg/logger"
)

// ConfigureNativeHA configures native high availability
func ConfigureNativeHA(log *logger.Logger) error {

	file := "/etc/mqm/native-ha.ini"
	templateFile := file + ".tpl"

	templateMap := map[string]string{}
	templateMap["Name"] = os.Getenv("HOSTNAME")
	templateMap["NativeHAInstance0_Name"] = os.Getenv("MQ_NATIVE_HA_INSTANCE_0_NAME")
	templateMap["NativeHAInstance1_Name"] = os.Getenv("MQ_NATIVE_HA_INSTANCE_1_NAME")
	templateMap["NativeHAInstance2_Name"] = os.Getenv("MQ_NATIVE_HA_INSTANCE_2_NAME")
	templateMap["NativeHAInstance0_ReplicationAddress"] = os.Getenv("MQ_NATIVE_HA_INSTANCE_0_REPLICATION_ADDRESS")
	templateMap["NativeHAInstance1_ReplicationAddress"] = os.Getenv("MQ_NATIVE_HA_INSTANCE_1_REPLICATION_ADDRESS")
	templateMap["NativeHAInstance2_ReplicationAddress"] = os.Getenv("MQ_NATIVE_HA_INSTANCE_2_REPLICATION_ADDRESS")

	if os.Getenv("MQ_NATIVE_HA_TLS") == "true" {
		templateMap["CertificateLabel"] = os.Getenv("MQ_NATIVE_HA_TLS_CERTLABEL")

		keyRepository, ok := os.LookupEnv("MQ_NATIVE_HA_KEY_REPOSITORY")
		if !ok {
			keyRepository = "/run/runmqserver/tls/key"
		}
		templateMap["KeyRepository"] = keyRepository

		cipherSpec, ok := os.LookupEnv("MQ_NATIVE_HA_CIPHERSPEC")
		if ok {
			templateMap["CipherSpec"] = cipherSpec
		}
	}

	err := mqtemplate.ProcessTemplateFile(templateFile, file, templateMap, log)
	if err != nil {
		return err
	}

	return nil
}
