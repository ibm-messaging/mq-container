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
	"fmt"
	"os"

	"github.com/ibm-messaging/mq-container/internal/fips"
	"github.com/ibm-messaging/mq-container/internal/mqtemplate"
	"github.com/ibm-messaging/mq-container/internal/tls"
	"github.com/ibm-messaging/mq-container/pkg/logger"
)

// ConfigureNativeHA configures native high availability
func ConfigureNativeHA(log *logger.Logger) error {
	if os.Getenv("MQ_NATIVE_HA") != "true" {
		return nil
	}
	fipsAvailable := fips.IsFIPSEnabled()

	haCertLabel, haGroupCertLabel, _, _, err := tls.ConfigureHATLSKeystore(log)
	if err != nil {
		return fmt.Errorf("error loading tls keys: %w", err)
	}

	configFiles := map[string]string{
		"/run/10-native-ha-instance.ini": "/etc/mqm/10-native-ha-instance.ini.tpl",
	}
	if haCertLabel != "" || haGroupCertLabel != "" {
		configFiles["/run/10-native-ha-keystore.ini"] = "/etc/mqm/10-native-ha-keystore.ini.tpl"
	}
	if envConfigPresent() {
		log.Println("Configuring Native HA using values provided in environment variables")
		configFiles["/run/10-native-ha.ini"] = "/etc/mqm/10-native-ha.ini.tpl"
	}
	return loadConfigAndGenerate(configFiles, fipsAvailable, haCertLabel, haGroupCertLabel, log)
}

func loadConfigAndGenerate(templateConfigs map[string]string, fipsAvailable bool, haCertLabel, haGroupCertLabel string, log *logger.Logger) error {
	cfg, err := loadConfigFromEnv(log)
	if err != nil {
		return err
	}
	err = cfg.updateTLS(fipsAvailable, haCertLabel, haGroupCertLabel)
	if err != nil {
		return err
	}
	for outputPath, templateFile := range templateConfigs {
		err := cfg.generate(templateFile, outputPath, log)
		if err != nil {
			return err
		}
	}
	return nil
}

func envConfigPresent() bool {
	checkVars := []string{
		"MQ_NATIVE_HA_INSTANCE_0_NAME",
		"MQ_NATIVE_HA_INSTANCE_0_REPLICATION_ADDRESS",
		"MQ_NATIVE_HA_INSTANCE_1_NAME",
		"MQ_NATIVE_HA_INSTANCE_1_REPLICATION_ADDRESS",
		"MQ_NATIVE_HA_INSTANCE_2_NAME",
		"MQ_NATIVE_HA_INSTANCE_2_REPLICATION_ADDRESS",
		"MQ_NATIVE_HA_TLS",
		"MQ_NATIVE_HA_CIPHERSPEC",
	}
	for _, checkVar := range checkVars {
		if os.Getenv(checkVar) != "" {
			return true
		}
	}
	return false
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
		Group: haGroupConfig{
			Local: haLocalGroupConfig{
				Address: os.Getenv("MQ_NATIVE_HA_GROUP_LOCAL_ADDRESS"),
				Name:    os.Getenv("MQ_NATIVE_HA_GROUP_LOCAL_NAME"),
				Role:    os.Getenv("MQ_NATIVE_HA_GROUP_ROLE"),
			},
			Recovery: haRecoveryGroupConfig{
				Address: os.Getenv("MQ_NATIVE_HA_GROUP_REPLICATION_ADDRESS"),
				Name:    os.Getenv("MQ_NATIVE_HA_GROUP_RECOVERY_NAME"),
				Enabled: os.Getenv("MQ_NATIVE_HA_GROUP_RECOVERY_ENABLED") != "false",
			},
			CipherSpec: os.Getenv("MQ_NATIVE_HA_GROUP_CIPHERSPEC"),
		},
		CipherSpec:    os.Getenv("MQ_NATIVE_HA_CIPHERSPEC"),
		keyRepository: os.Getenv("MQ_NATIVE_HA_KEY_REPOSITORY"),
	}

	if cfg.Group.Recovery.Name == "" {
		cfg.Group.Recovery.Enabled = false
	}

	return cfg, nil
}

type haConfig struct {
	Name      string
	Instances [3]haInstance
	Group     haGroupConfig

	haTLSEnabled     bool
	CipherSpec       string
	CertificateLabel string
	keyRepository    string
	fipsAvailable    bool
}

func (h haConfig) ShouldConfigureTLS() bool {
	if h.haTLSEnabled {
		return true
	}
	if h.Group.Local.Name != "" {
		return true
	}
	return false
}

func (h haConfig) SSLFipsRequired() string {
	return yesNo(h.fipsAvailable).String()
}

func (h *haConfig) updateTLS(fipsAvailable bool, haCertLabel, haGroupCertLabel string) error {
	if haCertLabel != "" {
		h.CertificateLabel = haCertLabel
		h.haTLSEnabled = true
	}
	if haGroupCertLabel != "" {
		h.Group.CertificateLabel = haGroupCertLabel
		h.haTLSEnabled = true
	}

	h.fipsAvailable = fipsAvailable

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

type haGroupConfig struct {
	Local            haLocalGroupConfig
	Recovery         haRecoveryGroupConfig
	CipherSpec       string
	CertificateLabel string
}

type haLocalGroupConfig struct {
	Name    string
	Role    string
	Address string
}
type haRecoveryGroupConfig struct {
	Name    string
	Enabled yesNo
	Address string
}

type haInstance struct {
	Name               string
	ReplicationAddress string
}

type yesNo bool

func (yn yesNo) String() string {
	if yn {
		return "Yes"
	}
	return "No"
}
