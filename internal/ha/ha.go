/*
© Copyright IBM Corporation 2020, 2024

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

	"github.com/ibm-messaging/mq-container/internal/fips"
	"github.com/ibm-messaging/mq-container/internal/mqtemplate"
	"github.com/ibm-messaging/mq-container/internal/tls"
	"github.com/ibm-messaging/mq-container/pkg/logger"
)

// ConfigureNativeHA configures native high availability
func ConfigureNativeHA(log *logger.Logger) error {
	fileLink := "/run/native-ha.ini"
	templateFile := "/etc/mqm/native-ha.ini.tpl"
	fipsAvailable := fips.IsFIPSEnabled()
	return loadConfigAndGenerate(templateFile, fileLink, fipsAvailable, log)
}

func loadConfigAndGenerate(templatePath string, outputPath string, fipsAvailable bool, log *logger.Logger) error {
	cfg, err := loadConfigFromEnv(log)
	if err != nil {
		return err
	}

	err = cfg.updateTLS()
	if err != nil {
		return err
	}

	return cfg.generate(templatePath, outputPath, log)
}

func loadConfigFromEnv(log *logger.Logger) (*haConfig, error) {
	cfg := &haConfig{
		Name: os.Getenv("HOSTNAME"),
		Instances: [3]haInstance{
			{
				Name:               os.Getenv("MQ_NATIVE_HA_INSTANCE_0_NAME"),
				ReplicationAddress: os.Getenv("MQ_NATIVE_HA_INSTANCE_0_REPLICATION_ADDRESS"),
			},
			{
				Name:               os.Getenv("MQ_NATIVE_HA_INSTANCE_1_NAME"),
				ReplicationAddress: os.Getenv("MQ_NATIVE_HA_INSTANCE_1_REPLICATION_ADDRESS"),
			},
			{
				Name:               os.Getenv("MQ_NATIVE_HA_INSTANCE_2_NAME"),
				ReplicationAddress: os.Getenv("MQ_NATIVE_HA_INSTANCE_2_REPLICATION_ADDRESS"),
			},
		},
		tlsEnabled:    os.Getenv("MQ_NATIVE_HA_TLS") == "true",
		cipherSpec:    os.Getenv("MQ_NATIVE_HA_CIPHERSPEC"),
		keyRepository: os.Getenv("MQ_NATIVE_HA_KEY_REPOSITORY"),
	}

	return cfg, nil
}

type haConfig struct {
	Name      string
	Instances [3]haInstance

	tlsEnabled       bool
	cipherSpec       string
	certificateLabel string
	keyRepository    string
	fipsAvailable    bool
}

func (h haConfig) CertificateLabel() string {
	if !h.tlsEnabled {
		return ""
	}
	return h.certificateLabel
}

func (h haConfig) CipherSpec() string {
	if !h.tlsEnabled {
		return ""
	}
	return h.cipherSpec
}

func (h haConfig) SSLFipsRequired() string {
	if !h.tlsEnabled {
		return ""
	}
	if h.fipsAvailable {
		return "Yes"
	}
	return "No"
}

func (h *haConfig) updateTLS() error {
	if !h.tlsEnabled {
		return nil
	}

	keyLabel, _, _, err := tls.ConfigureHATLSKeystore()
	if err != nil {
		return err
	}
	h.certificateLabel = keyLabel

	h.fipsAvailable = fips.IsFIPSEnabled()

	return nil
}

func (h haConfig) generate(templatePath string, outputPath string, log *logger.Logger) error {
	return mqtemplate.ProcessTemplateFile(templatePath, outputPath, h, log)
}

func (h haConfig) KeyRepository() string {
	if h.keyRepository != "" {
		return h.keyRepository
	}
	return "/run/runmqserver/ha/tls/key"
}

type haInstance struct {
	Name               string
	ReplicationAddress string
}
