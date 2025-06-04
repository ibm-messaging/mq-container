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
package main

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/ibm-messaging/mq-container/internal/command"
	containerruntime "github.com/ibm-messaging/mq-container/internal/containerruntime"
	"github.com/ibm-messaging/mq-container/internal/mqscredact"
	"github.com/ibm-messaging/mq-container/internal/mqversion"
	"github.com/ibm-messaging/mq-container/internal/pathutils"
	"github.com/ibm-messaging/mq-container/internal/ready"
)

// createDirStructure creates the default MQ directory structure under /var/mqm
func createDirStructure() error {
	// log file diagnostics before and after crtmqdir if DEBUG=true
	logDiagnostics()
	out, rc, err := command.Run("/opt/mqm/bin/crtmqdir", "-f", "-a")
	if err != nil {
		if rc == 10 {
			// Ignore warnings about 'mqwebuser.xml' being a symlink
			if !(strings.Join(strings.Fields(string(out)), " ") == "The filesystem object '/mnt/mqm/data/web/installations/Installation1/servers/mqweb/mqwebuser.xml' is a symbolic link.") {
				log.Printf("Warning creating directory structure: %v\n", string(out))
			}
		} else {
			log.Printf("Error creating directory structure: the 'crtmqdir' command returned with code: %v. Reason: %v\n", rc, string(out))
			return err
		}
	}
	log.Println("Created directory structure under /var/mqm")
	logDiagnostics()
	return nil
}

// createQueueManager creates a queue manager, if it doesn't already exist.
// It returns true if one was created (or a standby was created), or false if one already existed
func createQueueManager(name string, devMode bool) (bool, error) {
	log.Printf("Creating queue manager %v", name)

	mounts, err := containerruntime.GetMounts()
	if err != nil {
		log.Printf("Error getting mounts for queue manager")
		return false, err
	}

	dataDir := getQueueManagerDataDir(mounts, replaceCharsInQMName(name))

	// Run 'dspmqinf' to check if 'mqs.ini' configuration file exists
	// If command succeeds, the queue manager (or standby queue manager) has already been created
	_, _, err = command.Run("dspmqinf", name)
	if err == nil {
		log.Printf("Detected existing queue manager %v", name)
		// Check if MQ_QMGR_LOG_FILE_PAGES matches the value set in qm.ini
		lfp := os.Getenv("MQ_QMGR_LOG_FILE_PAGES")
		if lfp != "" {
			qmIniBytes, err := readQMIni(dataDir)
			if err != nil {
				log.Printf("Error reading qm.ini : %v", err)
				return false, err
			}
			if !validateLogFilePageSetting(qmIniBytes, lfp) {
				log.Println("Warning: the value of MQ_QMGR_LOG_FILE_PAGES does not match the value of 'LogFilePages' in the qm.ini. This setting cannot be altered after Queue Manager creation.")
			}
		}
		// Check if MQ_QMGR_PRIMARY_LOGFILES matches the value set in qm.ini
		plf := os.Getenv("MQ_QMGR_PRIMARY_LOGFILES")
		if plf != "" {
			qmIniBytes, err := readQMIni(dataDir)
			if err != nil {
				log.Printf("Error reading qm.ini : %v", err)
				return false, err
			}
			if !validatePrimaryLogFileSetting(qmIniBytes, plf) {
				log.Println("Warning: the value of MQ_QMGR_PRIMARY_LOGFILES does not match the value of 'LogPrimaryFiles' in the qm.ini. This setting cannot be altered after Queue Manager creation.")
			}
		}
		// Check if MQ_QMGR_SECONDARY_LOGFILES matches the value set in qm.ini
		slf := os.Getenv("MQ_QMGR_SECONDARY_LOGFILES")
		if slf != "" {
			qmIniBytes, err := readQMIni(dataDir)
			if err != nil {
				log.Printf("Error reading qm.ini : %v", err)
				return false, err
			}
			if !validateSecondaryLogFileSetting(qmIniBytes, slf) {
				log.Println("Warning: the value of MQ_QMGR_SECONDARY_LOGFILES does not match the value of 'LogSecondaryFiles' in the qm.ini. This setting cannot be altered after Queue Manager creation.")
			}
		}
		return false, nil
	}

	// Check if 'qm.ini' configuration file exists for the queue manager
	// TODO : handle possible race condition - use a file lock?
	_, err = os.Stat(pathutils.CleanPath(dataDir, "qm.ini"))
	if err != nil {
		// If 'qm.ini' is not found - run 'crtmqm' to create a new queue manager
		args := getCreateQueueManagerArgs(mounts, name, devMode)
		out, rc, err := command.Run("crtmqm", args...)
		if err != nil {
			log.Printf("Error creating queue manager: the 'crtmqm' command returned with code: %v. Reason: %v", rc, string(out))
			return false, err
		}
	} else {
		// If 'qm.ini' is found - run 'addmqinf' to create a standby queue manager with existing configuration
		args := getCreateStandbyQueueManagerArgs(name)
		out, rc, err := command.Run("addmqinf", args...)
		if err != nil {
			log.Printf("Error creating standby queue manager: the 'addmqinf' command returned with code: %v. Reason: %v",
				rc, string(out))
			return false, err
		}
		log.Println("Created standby queue manager")
		return true, nil
	}
	log.Println("Created queue manager")
	return true, nil
}

// readQMIni reads the qm.ini file and returns it as a byte array
// This function is specific to comply with the nosec.
func readQMIni(dataDir string) ([]byte, error) {
	qmgrDir := pathutils.CleanPath(dataDir, "qm.ini")
	// #nosec G304 - qmgrDir filepath is derived from dspmqinf
	iniFileBytes, err := os.ReadFile(qmgrDir)
	if err != nil {
		return nil, err
	}
	return iniFileBytes, err
}

// validateLogFilePageSetting validates if the specified logFilePage number is equal to the existing value in the qm.ini
func validateLogFilePageSetting(iniFileBytes []byte, logFilePages string) bool {
	lfpString := "LogFilePages=" + logFilePages
	qminiConfigStr := string(iniFileBytes)
	return strings.Contains(qminiConfigStr, lfpString)
}

// validatePrimaryLogFileSetting validates if the specified primaryLogFiles number is equal to the existing value in the qm.ini
func validatePrimaryLogFileSetting(iniFileBytes []byte, primaryLogFiles string) bool {
	plfString := "LogPrimaryFiles=" + primaryLogFiles
	qminiConfigStr := string(iniFileBytes)
	return strings.Contains(qminiConfigStr, plfString)
}

// validateSecondaryLogFileSetting validates if the specified secondaryLogFiles number is equal to the existing value in the qm.ini
func validateSecondaryLogFileSetting(iniFileBytes []byte, secondaryLogFiles string) bool {
	slfString := "LogSecondaryFiles=" + secondaryLogFiles
	qminiConfigStr := string(iniFileBytes)
	return strings.Contains(qminiConfigStr, slfString)
}

func updateCommandLevel() error {
	level, ok := os.LookupEnv("MQ_CMDLEVEL")
	if ok && level != "" {
		log.Printf("Setting CMDLEVEL to %v", level)
		out, rc, err := command.Run("strmqm", "-e", "CMDLEVEL="+level)
		if err != nil {
			log.Printf("Error setting CMDLEVEL for queue manager: the 'strmqm' command returned with code: %v. Reason: %v",
				rc, string(out))
			return err
		}
	}
	return nil
}

func startQueueManager(name string) error {
	log.Println("Starting queue manager")
	out, rc, err := command.Run("strmqm", "-x", name)
	if err != nil {
		// 30=standby queue manager started, which is fine
		// 94=native HA replica started, which is fine
		if rc == 30 {
			log.Printf("Started standby queue manager")
			return nil
		} else if rc == 94 {
			log.Printf("Started replica queue manager")
			return nil
		}
		log.Printf("Error starting queue manager: the 'strmqm' command returned with code: %v. Reason: %v", rc, string(out))
		return err
	}
	log.Println("Started queue manager")
	return nil
}

func stopQueueManager(name string) error {
	log.Println("Stopping queue manager")
	qmGracePeriod := os.Getenv("MQ_GRACE_PERIOD")
	status, err := ready.Status(context.Background(), name)
	if err != nil {
		log.Printf("Error getting status for queue manager %v. The 'dspmq' command returned reason: %v",
			name, err.Error())

		return err
	}
	isStandby := status.StandbyQM()
	args := []string{"-w", "-r", "-tp", qmGracePeriod, name}
	if os.Getenv("MQ_MULTI_INSTANCE") == "true" {
		if isStandby {
			args = []string{"-x", name}
		} else {
			args = []string{"-s", "-w", "-tp", qmGracePeriod, name}
		}
	}
	out, rc, err := command.Run("endmqm", args...)
	if err != nil {
		log.Printf("Error stopping queue manager: the 'endmqm' command returned with code: %v.  Reason: %v", rc, string(out))
		return err
	}
	if isStandby {
		log.Printf("Stopped standby queue manager")
	} else {
		log.Println("Stopped queue manager")
	}
	return nil
}

func startMQTrace() error {
	log.Println("Starting MQ trace")
	out, rc, err := command.Run("strmqtrc")
	if err != nil {
		log.Printf("Error starting MQ trace: the 'strmqtrc' command returned with code: %v.  Reason: %v", rc, string(out))
		return err
	}
	log.Println("Started MQ trace")
	return nil
}

func endMQTrace() error {
	log.Println("Ending MQ Trace")
	out, rc, err := command.Run("endmqtrc")
	if err != nil {
		log.Printf("Error ending MQ trace: the 'endmqtrc' command returned with code: %v.  Reason: %v", rc, string(out))
		return err
	}
	log.Println("Ended MQ trace")
	return nil
}

func formatMQSCOutput(out string) string {
	// redact sensitive information
	out, _ = mqscredact.Redact(out)

	// add tab characters to make it more readable as part of the log
	return strings.Replace(string(out), "\n", "\n\t", -1)
}

func isStandbyQueueManager(name string) (bool, error) {
	out, rc, err := command.Run("dspmq", "-n", "-m", name)
	if err != nil {
		log.Printf("Error while getting status for queue manager %v: the 'dspmq' command returned with code: %v.  Reason: %v",
			name, rc, string(out))
		return false, err
	}
	if strings.Contains(string(out), "(RUNNING AS STANDBY)") {
		return true, nil
	}
	return false, nil
}

func getQueueManagerDataDir(mounts map[string]string, name string) string {
	dataDir := pathutils.CleanPath("/var/mqm/qmgrs", name)
	if _, ok := mounts["/mnt/mqm-data"]; ok {
		dataDir = pathutils.CleanPath("/mnt/mqm-data/qmgrs", name)
	}
	return dataDir
}

func getCreateQueueManagerArgs(mounts map[string]string, name string, devMode bool) []string {

	mqversionBase := "9.2.1.0"

	// use "UserExternal" only if we are 9.2.1.0 or above.
	oaVal := "user"
	mqVersionCheck, err := mqversion.Compare(mqversionBase)

	if err != nil {
		log.Printf("Error comparing MQ versions for oa,rc: %v", mqVersionCheck)
	}
	if mqVersionCheck >= 0 {
		oaVal = "UserExternal"
	}

	//build args
	args := []string{"-ii", "/etc/mqm/", "-ic", "/etc/mqm/", "-q", "-p", "1414"}

	if os.Getenv("MQ_NATIVE_HA") == "true" {
		args = append(args, "-lr", os.Getenv("HOSTNAME"))
	}
	if devMode {
		args = append(args, "-oa", oaVal)
	}
	if _, ok := mounts["/mnt/mqm-log"]; ok {
		args = append(args, "-ld", "/mnt/mqm-log/log")
	}
	if _, ok := mounts["/mnt/mqm-data"]; ok {
		args = append(args, "-md", "/mnt/mqm-data/qmgrs")
	}
	if os.Getenv("MQ_QMGR_LOG_FILE_PAGES") != "" {
		_, err = strconv.Atoi(os.Getenv("MQ_QMGR_LOG_FILE_PAGES"))
		if err != nil {
			log.Printf("Error processing MQ_QMGR_LOG_FILE_PAGES, the default value for LogFilePages will be used. Err: %v", err)
		} else {
			args = append(args, "-lf", os.Getenv("MQ_QMGR_LOG_FILE_PAGES"))
		}
	}
	if os.Getenv("MQ_QMGR_PRIMARY_LOGFILES") != "" {
		_, err = strconv.Atoi(os.Getenv("MQ_QMGR_PRIMARY_LOGFILES"))
		if err != nil {
			log.Printf("Error processing MQ_QMGR_PRIMARY_LOGFILES, the default value for LogPrimaryFiles will be used. Err: %v", err)
		} else {
			args = append(args, "-lp", os.Getenv("MQ_QMGR_PRIMARY_LOGFILES"))
		}
	}
	if os.Getenv("MQ_QMGR_SECONDARY_LOGFILES") != "" {
		_, err = strconv.Atoi(os.Getenv("MQ_QMGR_SECONDARY_LOGFILES"))
		if err != nil {
			log.Printf("Error processing MQ_QMGR_SECONDARY_LOGFILES, the default value for LogSecondaryFiles will be used. Err: %v", err)
		} else {
			args = append(args, "-ls", os.Getenv("MQ_QMGR_SECONDARY_LOGFILES"))
		}
	}
	args = append(args, name)
	return args
}

func getCreateStandbyQueueManagerArgs(name string) []string {
	args := []string{"-s", "QueueManager"}
	args = append(args, "-v", fmt.Sprintf("Name=%v", name))
	args = append(args, "-v", fmt.Sprintf("Directory=%v", name))
	args = append(args, "-v", "Prefix=/var/mqm")
	args = append(args, "-v", fmt.Sprintf("DataPath=/mnt/mqm-data/qmgrs/%v", name))
	return args
}

// updateQMini removes the original ServicecCmponent stanza so we can add a new one
func updateQMini(qmname string) error {

	val, set := os.LookupEnv("MQ_CONNAUTH_USE_HTP")
	if !set {
		//simpleauth mode not enabled.
		return nil
	}
	bval, err := strconv.ParseBool(strings.ToLower(val))
	if err != nil {
		return err
	}
	if bval == false {
		//simpleauth mode not enabled.
		return nil
	}

	log.Printf("Removing existing ServiceComponent configuration")

	mounts, err := containerruntime.GetMounts()
	if err != nil {
		log.Printf("Error getting mounts for queue manager")
		return err
	}
	dataDir := getQueueManagerDataDir(mounts, replaceCharsInQMName(qmname))
	qmgrDir := pathutils.CleanPath(dataDir, "qm.ini")
	//read the initial version.
	// #nosec G304 - qmgrDir filepath is derived from dspmqinf
	iniFileBytes, err := os.ReadFile(qmgrDir)
	if err != nil {
		return err
	}
	qminiConfigStr := string(iniFileBytes)
	if strings.Contains(qminiConfigStr, "ServiceComponent:") {
		var re = regexp.MustCompile(`(?m)^.*ServiceComponent.*$\s^.*Service.*$\s^.*Name.*$\s^.*Module.*$\s^.*ComponentDataSize.*$`)
		curFile := re.ReplaceAllString(qminiConfigStr, "")
		// #nosec G304 G306 - qmgrDir filepath is derived from dspmqinf and
		// its a read by owner/s group, and pose no harm.
		err := os.WriteFile(qmgrDir, []byte(curFile), 0660)
		if err != nil {
			return err
		}
	}
	return nil
}

// If queue manager name contains a '.', then the '.' will be replaced with '!'
// in the name of the data directory created by the queue manager. Similarly
// '/' will be replaced with '&'.
func replaceCharsInQMName(qmname string) string {
	replacedName := qmname
	if strings.Contains(replacedName, ".") {
		replacedName = strings.ReplaceAll(replacedName, ".", "!")
	}
	if strings.Contains(replacedName, "/") {
		replacedName = strings.ReplaceAll(replacedName, "/", "&")
	}
	return replacedName
}
