/*
© Copyright IBM Corporation 2017, 2023

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
package containerruntimelogger

import (
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/ibm-messaging/mq-container/internal/containerruntime"
	"github.com/ibm-messaging/mq-container/internal/user"
	"github.com/ibm-messaging/mq-container/pkg/logger"
)

// LogContainerDetails logs details about the container runtime
func LogContainerDetails(log *logger.Logger) error {
	if runtime.GOOS != "linux" {
		return fmt.Errorf("Unsupported platform: %v", runtime.GOOS)
	}
	log.Printf("CPU architecture: %v", runtime.GOARCH)
	kv, err := containerruntime.GetKernelVersion()
	if err == nil {
		log.Printf("Linux kernel version: %v", kv)
	}

	bi, err := containerruntime.GetBaseImage()
	if err == nil {
		log.Printf("Base image: %v", bi)
	}
	u, err := user.GetUser()
	if err != nil {
		log.Printf("Error: %v\nUser:\n  uid: %v\n  gid: %v\n  supGid: %v", err, u.UID, u.PrimaryGID, u.SupplementalGID)
	}
	if err == nil {
		if len(u.SupplementalGID) == 0 {
			log.Printf("Running as user ID %v with primary group %v", u.UID, u.PrimaryGID)
		} else {
			log.Printf("Running as user ID %v with primary group %v, and supplementary groups %v", u.UID, u.PrimaryGID, strings.Trim(strings.Join(strings.Fields(fmt.Sprint(u.SupplementalGID)), ","), "[]"))
		}
	}
	caps, err := containerruntime.GetCapabilities(1)
	capLogged := false
	if err == nil {
		for k, v := range caps {
			if len(v) > 0 {
				log.Printf("Capabilities (%s set): %v", strings.ToLower(k), strings.Join(v, ","))
				capLogged = true
			}
		}
		if !capLogged {
			log.Print("Capabilities: none")
		}
	} else {
		log.Errorf("Error getting capabilities: %v", err)
	}
	sc := containerruntime.GetSeccomp()
	log.Printf("seccomp enforcing mode: %v", sc)

	log.Printf("Process security attributes: %v", containerruntime.GetSecurityAttributes())
	m, err := containerruntime.GetMounts()
	if err == nil {
		if len(m) == 0 {
			log.Print("No volume detected. Persistent messages may be lost")
		} else {
			for mountPoint, fsType := range m {
				log.Printf("Detected '%v' volume mounted to %v", fsType, mountPoint)
				if !containerruntime.SupportedFilesystem(fsType) {
					return fmt.Errorf("%v uses unsupported filesystem type: %v", mountPoint, fsType)
				}
			}
		}
	}
	// For a multi-instance queue manager - check all required mounts exist & validate filesystem type
	if os.Getenv("MQ_MULTI_INSTANCE") == "true" {
		log.Println("Multi-instance queue manager: enabled")
		reqMounts := []string{"/mnt/mqm", "/mnt/mqm-log", "/mnt/mqm-data"}
		for _, mountPoint := range reqMounts {
			if fsType, ok := m[mountPoint]; ok {
				if !containerruntime.ValidMultiInstanceFilesystem(fsType) {
					return fmt.Errorf("%v uses filesystem type '%v' which is invalid for a multi-instance queue manager", mountPoint, fsType)
				}
			} else {
				return fmt.Errorf("Missing required mount '%v' for a multi-instance queue manager", mountPoint)
			}
		}
	}

	if os.Getenv("MQ_LOGGING_CONSOLE_FORMAT") == "" && os.Getenv("LOG_FORMAT") != "" {
		log.Println("Environment variable LOG_FORMAT is deprecated. Use MQ_LOGGING_CONSOLE_FORMAT instead.")
	}

	return nil
}
