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
	"strings"

	"github.com/ibm-messaging/mq-container/internal/command"
)

var (
	Buildstamp     = "No date value provided"
	Buildgitcommit = "No commit value provided"
	Buildgitrepo   = "No git repo value provided"
)

func PrintDateStamp() {
	log.Printf("Build Date Stamp: %v", Buildstamp)
}

func PrintGitCommit() {
	log.Printf("Build Git Commit Hash: %v", Buildgitcommit)
}

func PrintGitRepo() {
	log.Printf("Build Git repository: %v", Buildgitrepo)
}

func PrintMQVersion() {
	mqVersion, _, err := command.Run("dspmqver", "-b", "-f", "2")
	if err != nil {
		log.Printf("Error Getting MQ Version: %v", strings.TrimSuffix(string(mqVersion), "\n"))
	}

	mqBuild, _, err := command.Run("dspmqver", "-b", "-f", "4")
	if err != nil {
		log.Printf("Error Getting MQ Build: %v", strings.TrimSuffix(string(mqBuild), "\n"))
	}
	mqLicense, _, err := command.Run("dspmqver", "-b", "-f", "8192")
	if err != nil {
		log.Printf("Error Getting MQ License: %v", strings.TrimSuffix(string(mqLicense), "\n"))
	}
	mqCmdLevel, _, err := command.Run("dspmqver", "-b", "-f", "1024")
	if err != nil {
		log.Printf("Error Getting MQ Cmdlevel: %v", strings.TrimSuffix(string(mqCmdLevel), "\n"))
	}
	mqOS, _, err := command.Run("dspmqver", "-b", "-f", "64")
	if err != nil {
		log.Printf("Error Getting MQ OS: %v", strings.TrimSuffix(string(mqOS), "\n"))
	}
	mqMode, _, err := command.Run("dspmqver", "-b", "-f", "32")
	if err != nil {
		log.Printf("Error Getting MQ Mode: %v", strings.TrimSuffix(string(mqMode), "\n"))
	}
	mqPlatform, _, err := command.Run("dspmqver", "-b", "-f", "16")
	if err != nil {
		log.Printf("Error Getting MQ Platform: %v", strings.TrimSuffix(string(mqPlatform), "\n"))
	}

	log.Printf("MQ Version: %v", strings.TrimSuffix(mqVersion, "\n"))
	log.Printf("MQ Build: %v", strings.TrimSuffix(mqBuild, "\n"))
	log.Printf("MQ License: %v", strings.TrimSuffix(mqLicense, "\n"))
	log.Printf("MQ Cmdlevel: %v", strings.TrimSuffix(mqCmdLevel, "\n"))
	log.Printf("MQ OS: %v", strings.TrimSuffix(mqOS, "\n"))
	log.Printf("MQ Mode: %v", strings.TrimSuffix(mqMode, "\n"))
	log.Printf("MQ Platform: %v", strings.TrimSuffix(mqPlatform, "\n"))
}

func PrintVersionInfo() {
	PrintDateStamp()
	PrintGitRepo()
	PrintGitCommit()
	PrintMQVersion()
}
