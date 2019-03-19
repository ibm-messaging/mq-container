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

package main

import (
	"strings"

	"github.com/ibm-messaging/mq-container/internal/command"
)

var (
	// ImageCreated is the date the image was built
	ImageCreated = "Not specified"
	// ImageRevision is the source control revision identifier
	ImageRevision = "Not specified"
	// ImageSource is the URL to get source code for building the image
	ImageSource = "Not specified"
	// ImageTag is the tag of the image
	ImageTag = "Not specified"
)

func logDateStamp() {
	log.Printf("Image created: %v", ImageCreated)
}

func logGitRepo() {
	log.Printf("Image revision: %v", ImageRevision)
}

func logGitCommit() {
	log.Printf("Image source: %v", ImageSource)
}

func logImageTag() {
	log.Printf("Image tag: %v", ImageTag)
}

func logMQVersion() {
	mqVersion, _, err := command.Run("dspmqver", "-b", "-f", "2")
	if err != nil {
		log.Printf("Error Getting MQ version: %v", strings.TrimSuffix(string(mqVersion), "\n"))
	}

	mqBuild, _, err := command.Run("dspmqver", "-b", "-f", "4")
	if err != nil {
		log.Printf("Error Getting MQ build: %v", strings.TrimSuffix(string(mqBuild), "\n"))
	}
	mqLicense, _, err := command.Run("dspmqver", "-b", "-f", "8192")
	if err != nil {
		log.Printf("Error Getting MQ license: %v", strings.TrimSuffix(string(mqLicense), "\n"))
	}

	log.Printf("MQ version: %v", strings.TrimSuffix(mqVersion, "\n"))
	log.Printf("MQ level: %v", strings.TrimSuffix(mqBuild, "\n"))
	log.Printf("MQ license: %v", strings.TrimSuffix(mqLicense, "\n"))
}

func logVersionInfo() {
	logDateStamp()
	logGitRepo()
	logGitCommit()
	logImageTag()
	logMQVersion()
}
