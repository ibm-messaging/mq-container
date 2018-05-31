# © Copyright IBM Corporation 2018
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

###############################################################################
# Conditional variables - you can override the values of these variables from
# the command line
###############################################################################
# BASE_IMAGE is the MQ SDK base image to use
BASE_IMAGE ?= mq-sdk:9.0.5.0-x86_64-ubuntu-16.04
# GO_IMAGE is the GOLANG image to use for building samples
GO_IMAGE ?= golang:1.10
# DOCKER is the Docker command to run
DOCKER ?= docker
# BUILD_IMAGE is the name of the image that will be produced while building packages
BUILD_IMAGE ?= mq-golang-build:9.0.5.0-x86_64-ubuntu-16.04
# SAMPLE_BUILD_IMAGE is the name of the image that will be produced while building samples
SAMPLE_BUILD_IMAGE ?= mq-sample-build:9.0.5.0-x86_64-ubuntu-16.04

###############################################################################
# Other variables
###############################################################################

ifneq (,$(findstring Microsoft,$(shell uname -r)))
	PLATFORM=WINDOWS
else
	PLATFORM=UNIX
endif

###############################################################################
# Build targets
###############################################################################

# Build all packages when on unix
.PHONY: all
ifeq ("$(PLATFORM)", "WINDOWS")
all: unsupported-message
else 
all: build-packages-unix build-samples-unix
endif

.PHONY: clean
clean:
	$(DOCKER) rmi -f $(BUILD_IMAGE)
	$(DOCKER) rmi -f $(SAMPLE_BUILD_IMAGE)

.PHONY: build-packages-unix
build-packages-unix:
	$(info $(SPACER)$(shell printf $(TITLE)"Building packages in build container"$(END)))
	$(call docker-build,$(BUILD_IMAGE),Dockerfile-build-packages,$(BASE_IMAGE))

.PHONY: build-samples-unix
build-samples-unix: build-packages-unix
	$(info $(SPACER)$(shell printf $(TITLE)"Building samples in build container"$(END)))
	$(call docker-build,$(SAMPLE_BUILD_IMAGE),Dockerfile-build-samples,$(BUILD_IMAGE))

.PHONY: unsupported-message
unsupported-message:
	$(info $(SPACER)$(shell printf $(TITLE)"This makefile can only be ran on UNIX platforms"$(END)))

define docker-build
	# Build the image first to compile the package/samples
	$(DOCKER) build -t $1                   \
	   -f $2                                \
	   --build-arg BASE_IMAGE=$3            \
	   .
endef

include formatting.mk
