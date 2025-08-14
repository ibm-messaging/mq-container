# © Copyright IBM Corporation 2025
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
# http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

#sps : Check if the environment is SPS or not
ifneq (,$(PIPELINE_RUN_ID))
	PIPELINE_BRANCH=$(BRANCH)
	BUILD_DIRECTORY=$(SPS_BUILD_DIR)
	BUILD_SCRIPTS_PATH=sps-build-scripts
	COMMAND=podman
	IMAGE_FORMAT=--format docker
	ifeq "$(ARCH)" "ppc64le"
	    NUM_CPU=2
	endif
ifeq "$(PIPELINE_NAMESPACE)" "pr"
	PIPELINE_PULL_REQUEST=true
endif
ifeq "$(PIPELINE_NAMESPACE)" "ci"
	PIPELINE_PULL_REQUEST=false
endif
else
# sps: If TRAVIS has a value then the build is running on travis
	PIPELINE_BRANCH=$(TRAVIS_BRANCH)
	PIPELINE_PULL_REQUEST=$(TRAVIS_PULL_REQUEST)
	BUILD_DIRECTORY=$(TRAVIS_BUILD_DIR)
	BUILD_SCRIPTS_PATH=travis-build-scripts
endif

# If this is a fake master build, push images to alternative location (pipeline wont consider these images GA candidates)
ifeq ($(shell [ "$(TRAVIS)" = "true" ] && [ -n "$(MAIN_BRANCH)" ] && [ -n "$(SOURCE_BRANCH)" ] && [ "$(MAIN_BRANCH)" != "$(SOURCE_BRANCH)" ] && echo "true"), true)
	MQ_DELIVERY_REGISTRY_NAMESPACE="master-fake"
endif

# LTS_TAG is the tag modifier for an LTS container build
LTS_TAG=
ifeq "$(LTS)" "true"
ifneq "$(LTS_TAG_OVERRIDE)" "$(EMPTY)"
	LTS_TAG=$(LTS_TAG_OVERRIDE)
else
	LTS_TAG=-lts
endif
	MQ_ARCHIVE:=$(MQ_VERSION)-IBM-MQ-Advanced-Non-Install-Linux$(MQ_ARCHIVE_ARCH).tar.gz
	MQ_DELIVERY_REGISTRY_NAMESPACE:=$(MQ_DELIVERY_REGISTRY_NAMESPACE)$(LTS_TAG)
endif

ifneq "$(MQ_DELIVERY_REGISTRY_NAMESPACE)" "$(EMPTY)"
	MQ_DELIVERY_REGISTRY_FULL_PATH=$(MQ_DELIVERY_REGISTRY_HOSTNAME)/$(MQ_DELIVERY_REGISTRY_NAMESPACE)
else
	MQ_DELIVERY_REGISTRY_FULL_PATH=$(MQ_DELIVERY_REGISTRY_HOSTNAME)
endif

#sps : check if the build is Travis or SPS 
ifeq ($(shell ( [ ! -z $(TRAVIS) ] || [ ! -z $(PIPELINE_RUN_ID) ] ) && echo "$(PIPELINE_BRANCH)" | grep -q '^ifix-' && echo true), true)
	MQ_DELIVERY_REGISTRY_FULL_PATH=$(MQ_DELIVERY_REGISTRY_HOSTNAME)/$(MQ_DELIVERY_REGISTRY_NAMESPACE_IFIX)
	MQ_DELIVERY_REGISTRY_NAMESPACE=$(MQ_DELIVERY_REGISTRY_NAMESPACE_IFIX)
endif

# image tagging

ifneq "$(RELEASE)" "$(EMPTY)"
	EXTRA_LABELS_RELEASE=--label "release=$(RELEASE)"
	RELEASE_TAG=-$(RELEASE)
endif

ifneq "$(MQ_ARCHIVE_LEVEL)" "$(EMPTY)"
	EXTRA_LABELS_LEVEL=--label "mq-build=$(MQ_ARCHIVE_LEVEL)"
endif

EXTRA_LABELS=$(EXTRA_LABELS_RELEASE) $(EXTRA_LABELS_LEVEL)

ifeq "$(TIMESTAMPFLAT)" "$(EMPTY)"
	TIMESTAMPFLAT=$(shell date "+%Y%m%d%H%M%S")
endif

ifeq "$(GIT_COMMIT)" "$(EMPTY)"
	GIT_COMMIT=$(shell git rev-parse --short HEAD)
endif

#sps: Use the new variable PIPELINE_PULL_REQUEST
ifeq ($(shell ( [ ! -z $(TRAVIS) ] || [ ! -z $(PIPELINE_RUN_ID) ] ) && [ "$(PIPELINE_PULL_REQUEST)" = "false" ] && [ "$(PIPELINE_BRANCH)" = "$(MAIN_BRANCH)" ] && echo true), true)
	MQ_MANIFEST_TAG_SUFFIX=.$(TIMESTAMPFLAT).$(GIT_COMMIT)
endif

#sps: Use the new variable PIPELINE_PULL_REQUEST
ifeq ($(shell ( [ ! -z $(TRAVIS) ] || [ ! -z $(PIPELINE_RUN_ID) ] ) && [ "$(PIPELINE_PULL_REQUEST)" = "false" ] && echo "$(PIPELINE_BRANCH)" | grep -q '^ifix-' && echo true), true)
	MQ_MANIFEST_TAG_SUFFIX=-$(APAR_NUMBER)-$(FIX_NUMBER).$(TIMESTAMPFLAT).$(GIT_COMMIT)
endif

#sps: Update the TRAVIS_BUILD_DIR variable to use BUILD_DIRECTORY
PATH_TO_MQ_TAG_CACHE=$(BUILD_DIRECTORY)/.tagcache
ifneq ($(strip $(TRAVIS))$(strip $(PIPELINE_RUN_ID)),)
ifneq ("$(wildcard $(PATH_TO_MQ_TAG_CACHE))","")
include $(PATH_TO_MQ_TAG_CACHE)
endif
endif

MQ_AMD64_TAG=$(MQ_MANIFEST_TAG)-amd64
MQ_S390X_TAG?=$(MQ_MANIFEST_TAG)-s390x
MQ_PPC64LE_TAG?=$(MQ_MANIFEST_TAG)-ppc64le

# end image tagging

MQ_IMAGE_FULL_RELEASE_NAME=$(MQ_IMAGE_ADVANCEDSERVER):$(MQ_TAG)
MQ_IMAGE_DEV_FULL_RELEASE_NAME=$(MQ_IMAGE_DEVSERVER):$(MQ_TAG)

#setup variables for fat-manifests
MQ_IMAGE_DEVSERVER_MANIFEST=$(MQ_IMAGE_DEVSERVER):$(MQ_MANIFEST_TAG)
MQ_IMAGE_ADVANCEDSERVER_MANIFEST=$(MQ_IMAGE_ADVANCEDSERVER):$(MQ_MANIFEST_TAG)
MQ_IMAGE_DEVSERVER_AMD64=$(MQ_DELIVERY_REGISTRY_FULL_PATH)/$(MQ_IMAGE_DEVSERVER):$(MQ_AMD64_TAG)
MQ_IMAGE_DEVSERVER_S390X=$(MQ_DELIVERY_REGISTRY_FULL_PATH)/$(MQ_IMAGE_DEVSERVER):$(MQ_S390X_TAG)
MQ_IMAGE_DEVSERVER_PPC64LE=$(MQ_DELIVERY_REGISTRY_FULL_PATH)/$(MQ_IMAGE_DEVSERVER):$(MQ_PPC64LE_TAG)
MQ_IMAGE_ADVANCEDSERVER_AMD64=$(MQ_DELIVERY_REGISTRY_FULL_PATH)/$(MQ_IMAGE_ADVANCEDSERVER):$(MQ_AMD64_TAG)
MQ_IMAGE_ADVANCEDSERVER_S390X=$(MQ_DELIVERY_REGISTRY_FULL_PATH)/$(MQ_IMAGE_ADVANCEDSERVER):$(MQ_S390X_TAG)
MQ_IMAGE_ADVANCEDSERVER_PPC64LE=$(MQ_DELIVERY_REGISTRY_FULL_PATH)/$(MQ_IMAGE_ADVANCEDSERVER):$(MQ_PPC64LE_TAG)

MQ_IMAGE_DEVSERVER_MANIFEST_IFIX=$(MQ_DELIVERY_REGISTRY_FULL_PATH)/$(MQ_IMAGE_DEVSERVER):$(MQ_MANIFEST_TAG)
MQ_IMAGE_ADVANCESERVER_MANIFEST_IFIX=$(MQ_DELIVERY_REGISTRY_FULL_PATH)/$(MQ_IMAGE_ADVANCEDSERVER):$(MQ_MANIFEST_TAG)

PROJECT_DIR := $(shell pwd)
BUILD_MANIFEST_FILE := $(PROJECT_DIR)/latest-build-info/build-manifest.yaml