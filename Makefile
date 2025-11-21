# © Copyright IBM Corporation 2017, 2025
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

include config.env
include source-branch.env
ifdef PIPELINE_RUN_ID
include pipeline.env
endif

# arch_uname is the platform architecture according to the uname program.  Can be differ by OS, e.g. `arm64` on macOS, but `aarch64` on Linux.
arch_uname := $(shell uname -m)
# arch_go is the platform architecture in Go-style (e.g. amd64, ppc64le, s390x or arm64).
arch_go := $(if $(findstring x86_64,$(arch_uname)),amd64,$(if $(findstring aarch64,$(arch_uname)),arm64,$(arch_uname)))
# ARCH is the platform architecture in Go-style (e.g. amd64, ppc64le, s390x or arm64).
# Override this to build an image for a different architecture.  Note that RUN instructions will not be able to succeed without the help of emulation provided by packages like qemu-user-static.
ARCH ?= $(arch_go)
# RELEASE shows what release of the container code has been built
RELEASE ?=
# MQ_ARCHIVE_REPOSITORY is a remote repository from which to pull the MQ_ARCHIVE (if required)
MQ_ARCHIVE_REPOSITORY ?=
# MQ_ARCHIVE_REPOSITORY_DEV is a remote repository from which to pull the MQ_ARCHIVE_DEV (if required)
MQ_ARCHIVE_REPOSITORY_DEV ?=
# MQ_ARCHIVE_REPOSITORY_USER is the user for the remote repository (if required)
MQ_ARCHIVE_REPOSITORY_USER ?=
# MQ_ARCHIVE_REPOSITORY_CREDENTIAL is the password/API key for the remote repository (if required)
MQ_ARCHIVE_REPOSITORY_CREDENTIAL ?=
# MQ_ARCHIVE is the name of the file, under the downloads directory, from which MQ Advanced can
# be installed. Does not apply to MQ Advanced for Developers
MQ_ARCHIVE ?= IBM_MQ_$(MQ_VERSION_VRM)_$(MQ_ARCHIVE_TYPE)_$(MQ_ARCHIVE_ARCH)_NOINST.tar.gz
# MQ_ARCHIVE_DEV is the name of the file, under the downloads directory, from which MQ Advanced
# for Developers can be installed
MQ_ARCHIVE_DEV ?= $(MQ_VERSION)-IBM-MQ-Advanced-for-Developers-Non-Install-$(MQ_ARCHIVE_DEV_TYPE)$(MQ_ARCHIVE_DEV_ARCH).tar.gz
# MQ_SDK_ARCHIVE specifies the archive to use for building the golang programs.  Defaults vary on developer or advanced.
MQ_SDK_ARCHIVE ?= $(MQ_ARCHIVE_DEV_$(MQ_VERSION))
# Options to `go test` for the Container tests
TEST_OPTS_CONTAINER ?=
# Timeout for the  tests
TEST_TIMEOUT_CONTAINER ?= 60m
# MQ_IMAGE_ADVANCEDSERVER is the name of the built MQ Advanced image
MQ_IMAGE_ADVANCEDSERVER ?=ibm-mqadvanced-server
# MQ_IMAGE_DEVSERVER is the name of the built MQ Advanced for Developers image
MQ_IMAGE_DEVSERVER ?=ibm-mqadvanced-server-dev
# MQ_MANIFEST_TAG is the tag to use for fat-manifest
MQ_MANIFEST_TAG ?= $(MQ_VERSION)$(RELEASE_TAG)$(LTS_TAG)$(MQ_MANIFEST_TAG_SUFFIX)
# MQ_TAG is the tag of the built MQ Advanced image & MQ Advanced for Developers image
MQ_TAG ?= $(MQ_MANIFEST_TAG)-$(ARCH)
# COMMAND is the container command to run.  "podman" or "docker"
COMMAND ?=$(shell type -p podman 2>&1 >/dev/null && echo podman || echo docker)
# MQ_DELIVERY_REGISTRY_HOSTNAME is a remote registry to push the MQ Image to (if required)
MQ_DELIVERY_REGISTRY_HOSTNAME ?=
# MQ_DELIVERY_REGISTRY_NAMESPACE is the namespace/path on the delivery registry (if required)
MQ_DELIVERY_REGISTRY_NAMESPACE ?=
# MQ_DELIVERY_REGISTRY_USER is the user for the remote registry (if required)
MQ_DELIVERY_REGISTRY_USER ?=
# MQ_DELIVERY_REGISTRY_CREDENTIAL is the password/API key for the remote registry (if required)
MQ_DELIVERY_REGISTRY_CREDENTIAL ?=
# LTS is a boolean value to enable/disable LTS container build
LTS ?= false
# VOLUME_MOUNT_OPTIONS is used when bind-mounting files from the "downloads" directory into the container.  By default, SELinux labels are automatically re-written, but this doesn't work on some filesystems with extended attributes (xattrs).  You can turn off the label re-writing by setting this variable to be blank.
VOLUME_MOUNT_OPTIONS ?= :Z

###############################################################################
# Other variables
###############################################################################
# Lock Docker API version for compatibility with Podman and with the Docker version in Travis' Ubuntu Bionic
DOCKER_API_VERSION=1.40
GO_PKG_DIRS = ./cmd ./internal ./test
MQ_ARCHIVE_TYPE=LINUX
MQ_ARCHIVE_DEV_TYPE=Linux
# BUILD_SERVER_CONTAINER is the name of the web server container used at build time
BUILD_SERVER_CONTAINER=build-server
# BUILD_SERVER_NETWORK is the name of the network to use for the web server container used at build time
BUILD_SERVER_NETWORK=build
# BASE_IMAGE_TAG is a normalized version of BASE_IMAGE, suitable for use in a Docker tag
BASE_IMAGE_TAG=$(lastword $(subst /, ,$(subst :,-,$(BASE_IMAGE))))
#BASE_IMAGE_TAG=$(subst /,-,$(subst :,-,$(BASE_IMAGE)))
MQ_IMAGE_DEVSERVER_BASE=mqadvanced-server-dev-base
# Docker image name to use for JMS tests
DEV_JMS_IMAGE=mq-dev-jms-test
# Variables for versioning
IMAGE_REVISION=$(shell git rev-parse HEAD)
IMAGE_SOURCE=$(shell git config --get remote.origin.url)
EMPTY:=
SPACE:= $(EMPTY) $(EMPTY)
# MQ_VERSION_VRM is MQ_VERSION with only the Version, Release and Modifier fields (no Fix field).  e.g. 9.2.0 instead of 9.2.0.0
MQ_VERSION_VRM=$(subst $(SPACE),.,$(wordlist 1,3,$(subst .,$(SPACE),$(MQ_VERSION))))

#sps : Set the pipeline parameters
include Makefile.pipeline.mk

# Make sure we don't use VOLUME_MOUNT_OPTIONS for Podman on macOS
# Push options used while pushing the manifest are set
# Compression flags while pushing images and manifest are set
ifeq "$(COMMAND)" "podman"
	NUM_CPU ?= $(or $(shell podman info --format "{{.Host.CPUs}}"),2)
	PUSH_OPTIONS="$(IMAGE_FORMAT) --rm"
	COMPRESSION_FLAGS := --compression-format=gzip --force-compression=true
	ifeq "$(shell uname -s)" "Darwin"
		VOLUME_MOUNT_OPTIONS:=
	endif
else ifeq "$(COMMAND)" "docker"
	NUM_CPU ?= $(or $(shell docker info --format "{{ .NCPU }}"),2)
	PUSH_OPTIONS="--purge"
else
	NUM_CPU ?= 2
endif

ifneq (,$(findstring Microsoft,$(shell uname -r)))
	DOWNLOADS_DIR=$(patsubst /mnt/c%,C:%,$(realpath ./downloads/))
else ifneq (,$(findstring Windows,$(shell echo ${OS})))
	DOWNLOADS_DIR=$(shell pwd)/downloads/
else
	DOWNLOADS_DIR=$(realpath ./downloads/)
endif

# Try to figure out which archive to use from the architecture
ifeq "$(ARCH)" "amd64"
	MQ_ARCHIVE_ARCH:=X86-64
	MQ_ARCHIVE_DEV_ARCH:=X64
else ifeq "$(ARCH)" "ppc64le"
	MQ_ARCHIVE_ARCH:=PPC64LE
	MQ_ARCHIVE_DEV_ARCH:=PPC64LE
else ifeq "$(ARCH)" "s390x"
	MQ_ARCHIVE_ARCH:=S390X
	MQ_ARCHIVE_DEV_ARCH:=S390X
else ifeq "$(ARCH)" "arm64"
	MQ_ARCHIVE_ARCH:=ARM64
	MQ_ARCHIVE_DEV_ARCH:=ARM64
endif


###############################################################################
# Build targets
###############################################################################
.PHONY: default
default: build-devserver

# Build all components (except incubating ones)
.PHONY: all
all: build-devserver build-advancedserver

.PHONY: test-all
test-all: build-devjmstest test-devserver test-advancedserver

.PHONY: devserver
devserver: build-devserver build-devjmstest test-devserver

.PHONY: advancedserver
advancedserver: build-advancedserver test-advancedserver

# Build incubating components
.PHONY: incubating
incubating: build-explorer

downloads/$(MQ_ARCHIVE_DEV):
	$(info $(SPACER)$(shell printf $(TITLE)"Downloading IBM MQ Advanced for Developers "$(MQ_VERSION)$(END)))
	mkdir -p downloads
ifneq "$(BUILD_RSYNC_SERVER)" "$(EMPTY)"
	rsync -rv -e "ssh -o BatchMode=yes -q -o StrictHostKeyChecking=no" --include="*/" --include="*.tar.gz" --exclude="*" $(BUILD_RSYNC_USER)@$(BUILD_RSYNC_SERVER):"$(BUILD_RSYNC_PATH)" downloads/$(MQ_ARCHIVE_DEV)
else
ifneq "$(MQ_ARCHIVE_REPOSITORY_DEV)" "$(EMPTY)"
	curl --fail --user $(MQ_ARCHIVE_REPOSITORY_USER):$(MQ_ARCHIVE_REPOSITORY_CREDENTIAL) --request GET "$(MQ_ARCHIVE_REPOSITORY_DEV)" --output downloads/$(MQ_ARCHIVE_DEV)
else
	./download-basemq.sh -r $(MQ_ARCHIVE_DEV) -v $(MQ_VERSION_VRM)
endif
endif

downloads/$(MQ_ARCHIVE):
	$(info $(SPACER)$(shell printf $(TITLE)"Downloading IBM MQ Advanced "$(MQ_VERSION)$(END)))
	mkdir -p downloads
ifneq "$(BUILD_RSYNC_SERVER)" "$(EMPTY)"
	rsync -rv -e "ssh -o BatchMode=yes -q -o StrictHostKeyChecking=no" --include="*/" --include="*.tar.gz" --exclude="*" $(BUILD_RSYNC_USER)@$(BUILD_RSYNC_SERVER):"$(BUILD_RSYNC_PATH)" downloads/$(MQ_ARCHIVE)
else
ifneq "$(MQ_ARCHIVE_REPOSITORY)" "$(EMPTY)"
	curl --fail --user $(MQ_ARCHIVE_REPOSITORY_USER):$(MQ_ARCHIVE_REPOSITORY_CREDENTIAL) --request GET "$(MQ_ARCHIVE_REPOSITORY)" --output downloads/$(MQ_ARCHIVE)
endif
endif

.PHONY: downloads
downloads: downloads/$(MQ_ARCHIVE_DEV) downloads/$(MQ_SDK_ARCHIVE)

.PHONY: cache-mq-tag
cache-mq-tag:
	@printf "MQ_MANIFEST_TAG=$(MQ_MANIFEST_TAG)\n" | tee $(PATH_TO_MQ_TAG_CACHE)

###############################################################################
# Test targets
###############################################################################

# Vendor Go dependencies for the Container tests
test/container/vendor:
	cd test/container && go mod vendor

# Shortcut to just run the unit tests
.PHONY: test-unit
test-unit:
	$(COMMAND) build $(NETWORK) --target builder --file Dockerfile-server .

define inspect-image
	@$(COMMAND) inspect --format ">>> IMAGE UNDER TEST\n    RepoTags:     {{.RepoTags}}\n    RepoDigests:  {{.RepoDigests}}\n    Created:      {{.Created}}\n    Architecture: {{.Architecture}}" $1:$2
endef

#sps modify the go path in test-advanced server from /usr/local/go/bin/go to go
.PHONY: test-advancedserver
test-advancedserver:
	$(info $(SPACER)$(shell printf $(TITLE)"Test $(MQ_IMAGE_ADVANCEDSERVER):$(MQ_TAG) on $(shell $(COMMAND) --version)"$(END)))
	$(call inspect-image,$(MQ_IMAGE_ADVANCEDSERVER),$(MQ_TAG))
	cd test/container && TEST_IMAGE=$(MQ_IMAGE_ADVANCEDSERVER):$(MQ_TAG) EXPECTED_LICENSE=Production DOCKER_API_VERSION=$(DOCKER_API_VERSION) COMMAND=$(COMMAND) go test -parallel $(NUM_CPU) -timeout $(TEST_TIMEOUT_CONTAINER) $(TEST_OPTS_CONTAINER)

.PHONY: build-devjmstest
build-devjmstest:
	$(info $(SPACER)$(shell printf $(TITLE)"Build JMS tests for developer config"$(END)))
	cd test/messaging && $(COMMAND) build $(NETWORK) --tag $(DEV_JMS_IMAGE) .

.PHONY: test-devserver
test-devserver:
	$(info $(SPACER)$(shell printf $(TITLE)"Test $(MQ_IMAGE_DEVSERVER):$(MQ_TAG) on $(shell $(COMMAND) --version)"$(END)))
	$(call inspect-image,$(MQ_IMAGE_DEVSERVER),$(MQ_TAG))
	cd test/container && TEST_IMAGE=$(MQ_IMAGE_DEVSERVER):$(MQ_TAG) EXPECTED_LICENSE=Developer DEV_JMS_IMAGE=$(DEV_JMS_IMAGE) IBMJRE=false DOCKER_API_VERSION=$(DOCKER_API_VERSION) COMMAND=$(COMMAND) go test -parallel $(NUM_CPU) -timeout $(TEST_TIMEOUT_CONTAINER) -tags mqdev $(TEST_OPTS_CONTAINER)

.PHONY: coverage
coverage:
	mkdir coverage

.PHONY:
test-advancedserver-cover: coverage
	$(info $(SPACER)$(shell printf $(TITLE)"Test $(MQ_IMAGE_ADVANCEDSERVER):$(MQ_TAG) with code coverage on $(shell $(COMMAND) --version)"$(END)))
	rm -f ./coverage/unit*.cov
	# Run unit tests with coverage, for each package under 'internal'
	go list -f '{{.Name}}' ./internal/... | xargs -I {} go test -cover -covermode count -coverprofile ./coverage/unit-{}.cov ./internal/{}
#	ls -1 ./cmd | xargs -I {} go test -cover -covermode count -coverprofile ./coverage/unit-{}.cov ./cmd/{}/...
	echo 'mode: count' > ./coverage/unit.cov
	tail -q -n +2 ./coverage/unit-*.cov >> ./coverage/unit.cov
	go tool cover -html=./coverage/unit.cov -o ./coverage/unit.html

	rm -f ./test/container/coverage/*.cov
	rm -f ./coverage/container.*
	mkdir -p ./test/container/coverage/
	cd test/container && TEST_IMAGE=$(MQ_IMAGE_ADVANCEDSERVER):$(MQ_TAG)-cover TEST_COVER=true DOCKER_API_VERSION=$(DOCKER_API_VERSION) go test $(TEST_OPTS_CONTAINER)
	echo 'mode: count' > ./coverage/container.cov
	tail -q -n +2 ./test/container/coverage/*.cov >> ./coverage/container.cov
	go tool cover -html=./coverage/container.cov -o ./coverage/container.html

	echo 'mode: count' > ./coverage/combined.cov
	tail -q -n +2 ./coverage/unit.cov ./coverage/container.cov  >> ./coverage/combined.cov
	go tool cover -html=./coverage/combined.cov -o ./coverage/combined.html

###############################################################################
# Build functions
###############################################################################

# Command to build the image
# Args: imageName, imageTag, dockerfile, extraArgs, dockerfileTarget
# If the ARCH variable has been changed from the default value (arch_go variable), then the `--platform` parameter is added
# Args: imageName, imageTag, dockerfile, mqArchive, dockerfileTarget
define build-mq
	rm -f .dockerignore && echo ".git\ndownloads\n!downloads/$4" > .dockerignore
	$(COMMAND) build $(NETWORK) \
	  --tag $1:$2 \
	  --file $3 \
	  --build-arg IMAGE_REVISION="$(IMAGE_REVISION)" \
	  --build-arg IMAGE_SOURCE="$(IMAGE_SOURCE)" \
	  --build-arg IMAGE_TAG="$1:$2" \
	  --build-arg MQ_ARCHIVE="downloads/$4" \
	  --label version=$(MQ_VERSION) \
	  --label name=$1 \
	  --label build-date=$(shell date +%Y-%m-%dT%H:%M:%S%z) \
	  --label architecture="$(ARCH)" \
	  --label run="podman run -d -e LICENSE=accept $1:$2" \
	  --label vcs-ref=$(IMAGE_REVISION) \
	  --label vcs-type=git \
	  --label vcs-url=$(IMAGE_SOURCE) \
	  $(if $(findstring $(arch_go),$(ARCH)),,--platform=linux/$(ARCH)) \
	  $(IMAGE_FORMAT) \
	  $(EXTRA_LABELS) \
	  --target $5 \
	  .
endef

###############################################################################
# Build targets
###############################################################################
.PHONY: build-advancedserver-host
build-advancedserver-host: build-advancedserver

.PHONY: build-advancedserver
build-advancedserver: log-build-env downloads/$(MQ_ARCHIVE) command-version
	$(info $(SPACER)$(shell printf $(TITLE)"Build $(MQ_IMAGE_ADVANCEDSERVER):$(MQ_TAG)"$(END)))
	$(call build-mq,$(MQ_IMAGE_ADVANCEDSERVER),$(MQ_TAG),Dockerfile-server,$(MQ_ARCHIVE),mq-server)

.PHONY: build-devserver-host
build-devserver-host: build-devserver

.PHONY: build-devserver
build-devserver: log-build-env downloads/$(MQ_ARCHIVE_DEV) command-version
	$(info $(shell printf $(TITLE)"Build $(MQ_IMAGE_DEVSERVER):$(MQ_TAG)"$(END)))
	$(call build-mq,$(MQ_IMAGE_DEVSERVER),$(MQ_TAG),Dockerfile-server,$(MQ_ARCHIVE_DEV),mq-dev-server)

.PHONY: build-advancedserver-cover
build-advancedserver-cover: command-version
	$(COMMAND) build $(NETWORK) --build-arg BASE_IMAGE=$(MQ_IMAGE_ADVANCEDSERVER):$(MQ_TAG) -t $(MQ_IMAGE_ADVANCEDSERVER):$(MQ_TAG)-cover -f Dockerfile-server.cover .

.PHONY: build-explorer
build-explorer: downloads/$(MQ_ARCHIVE_DEV)
	$(call build-mq,mq-explorer,latest-$(ARCH),incubating/mq-explorer/Dockerfile,$(MQ_ARCHIVE_DEV),mq-explorer)

.PHONY: build-sdk
build-sdk: downloads/$(MQ_ARCHIVE_DEV)
	$(info $(shell printf $(TITLE)"Build $(MQ_IMAGE_SDK)"$(END)))
	$(call build-mq,mq-sdk,$(MQ_TAG),incubating/mq-sdk/Dockerfile,$(MQ_SDK_ARCHIVE),mq-sdk)

###############################################################################
# Logging targets
###############################################################################
.PHONY: log-build-env
log-build-vars:
	$(info $(SPACER)$(shell printf $(TITLE)"Build environment"$(END)))
	@echo arch_uname=$(arch_uname)
	@echo arch_go=$(arch_go)
	@echo "ARCH=$(ARCH) (origin:$(origin ARCH))"
	@echo MQ_VERSION="$(MQ_VERSION) (origin:$(origin MQ_VERSION))"
	@echo MQ_ARCHIVE="$(MQ_ARCHIVE) (origin:$(origin MQ_ARCHIVE))"
	@echo MQ_ARCHIVE_DEV_ARCH=$(MQ_ARCHIVE_DEV_ARCH)
	@echo MQ_ARCHIVE_DEV=$(MQ_ARCHIVE_DEV)
	@echo MQ_IMAGE_DEVSERVER=$(MQ_IMAGE_DEVSERVER)
	@echo MQ_IMAGE_ADVANCEDSERVER=$(MQ_IMAGE_ADVANCEDSERVER)
	@echo COMMAND=$(COMMAND)

.PHONY: log-build-env
log-build-env: log-build-vars
	$(info $(SPACER)$(shell printf $(TITLE)"Build environment - $(COMMAND) info"$(END)))
	@echo Command version: $(shell $(COMMAND) --version)
	$(COMMAND) info

include Makefile.formatting.mk

###############################################################################
# Push/pull targets
###############################################################################
.PHONY: pull-mq-archive
pull-mq-archive:
	curl --fail --user $(MQ_ARCHIVE_REPOSITORY_USER):$(MQ_ARCHIVE_REPOSITORY_CREDENTIAL) --request GET "$(MQ_ARCHIVE_REPOSITORY)" --output downloads/$(MQ_ARCHIVE)

.PHONY: pull-mq-archive-dev
pull-mq-archive-dev:
	curl --fail --user $(MQ_ARCHIVE_REPOSITORY_USER):$(MQ_ARCHIVE_REPOSITORY_CREDENTIAL) --request GET "$(MQ_ARCHIVE_REPOSITORY_DEV)" --output downloads/$(MQ_ARCHIVE_DEV)

.PHONY: push-advancedserver
push-advancedserver:
	@if [ $(MQ_DELIVERY_REGISTRY_NAMESPACE) = "master-fake" ]; then\
		echo "Detected fake master build. Note that the push destination is set to the fake master namespace: $(MQ_DELIVERY_REGISTRY_FULL_PATH)";\
	fi
	$(info $(SPACER)$(shell printf $(TITLE)"Push production image to $(MQ_DELIVERY_REGISTRY_FULL_PATH)"$(END)))
	$(COMMAND) login $(MQ_DELIVERY_REGISTRY_HOSTNAME) -u $(MQ_DELIVERY_REGISTRY_USER) -p $(MQ_DELIVERY_REGISTRY_CREDENTIAL)
	$(COMMAND) tag $(MQ_IMAGE_ADVANCEDSERVER)\:$(MQ_TAG) $(MQ_DELIVERY_REGISTRY_FULL_PATH)/$(MQ_IMAGE_FULL_RELEASE_NAME)
	$(COMMAND) push $(COMPRESSION_FLAGS) $(MQ_DELIVERY_REGISTRY_FULL_PATH)/$(MQ_IMAGE_FULL_RELEASE_NAME)

.PHONY: push-devserver
push-devserver:
	@if [ $(MQ_DELIVERY_REGISTRY_NAMESPACE) = "master-fake" ]; then\
		echo "Detected fake master build. Note that the push destination is set to the fake master namespace: $(MQ_DELIVERY_REGISTRY_FULL_PATH)";\
	fi
	$(info $(SPACER)$(shell printf $(TITLE)"Push developer image to $(MQ_DELIVERY_REGISTRY_FULL_PATH)"$(END)))
	$(COMMAND) login $(MQ_DELIVERY_REGISTRY_HOSTNAME) -u $(MQ_DELIVERY_REGISTRY_USER) -p $(MQ_DELIVERY_REGISTRY_CREDENTIAL)
	$(COMMAND) tag $(MQ_IMAGE_DEVSERVER)\:$(MQ_TAG) $(MQ_DELIVERY_REGISTRY_FULL_PATH)/$(MQ_IMAGE_DEV_FULL_RELEASE_NAME)
	$(COMMAND) push $(COMPRESSION_FLAGS) $(MQ_DELIVERY_REGISTRY_FULL_PATH)/$(MQ_IMAGE_DEV_FULL_RELEASE_NAME)

.PHONY: pull-advancedserver
pull-advancedserver:
	$(info $(SPACER)$(shell printf $(TITLE)"Pull production image from $(MQ_DELIVERY_REGISTRY_FULL_PATH)"$(END)))
	$(COMMAND) login $(MQ_DELIVERY_REGISTRY_HOSTNAME) -u $(MQ_DELIVERY_REGISTRY_USER) -p $(MQ_DELIVERY_REGISTRY_CREDENTIAL)
	$(COMMAND) pull $(MQ_DELIVERY_REGISTRY_FULL_PATH)/$(MQ_IMAGE_FULL_RELEASE_NAME)
	$(COMMAND) tag $(MQ_DELIVERY_REGISTRY_FULL_PATH)/$(MQ_IMAGE_FULL_RELEASE_NAME) $(MQ_IMAGE_ADVANCEDSERVER)\:$(MQ_TAG)

.PHONY: pull-devserver
pull-devserver:
	$(info $(SPACER)$(shell printf $(TITLE)"Pull developer image from $(MQ_DELIVERY_REGISTRY_FULL_PATH)"$(END)))
	$(COMMAND) login $(MQ_DELIVERY_REGISTRY_HOSTNAME) -u $(MQ_DELIVERY_REGISTRY_USER) -p $(MQ_DELIVERY_REGISTRY_CREDENTIAL)
	$(COMMAND) pull $(MQ_DELIVERY_REGISTRY_FULL_PATH)/$(MQ_IMAGE_DEV_FULL_RELEASE_NAME)
	$(COMMAND) tag $(MQ_DELIVERY_REGISTRY_FULL_PATH)/$(MQ_IMAGE_DEV_FULL_RELEASE_NAME) $(MQ_IMAGE_DEVSERVER)\:$(MQ_TAG)

.PHONY: push-manifest
push-manifest: build-skopeo-container
	$(info $(SPACER)$(shell printf $(TITLE)"** Determining the image digests **"$(END)))
ifneq "$(LTS)" "true"
	$(eval MQ_IMAGE_DEVSERVER_AMD64_DIGEST=$(shell $(COMMAND) run $(NETWORK) skopeo:latest --override-os linux inspect --creds $(MQ_ARCHIVE_REPOSITORY_USER):$(MQ_ARCHIVE_REPOSITORY_CREDENTIAL) docker://$(MQ_IMAGE_DEVSERVER_AMD64) | jq -r .Digest))
	$(eval MQ_IMAGE_DEVSERVER_S390X_DIGEST=$(shell $(COMMAND) run $(NETWORK) skopeo:latest --override-os linux inspect --creds $(MQ_ARCHIVE_REPOSITORY_USER):$(MQ_ARCHIVE_REPOSITORY_CREDENTIAL) docker://$(MQ_IMAGE_DEVSERVER_S390X) | jq -r .Digest))
	$(eval MQ_IMAGE_DEVSERVER_PPC64LE_DIGEST=$(shell $(COMMAND) run $(NETWORK) skopeo:latest --override-os linux inspect --creds $(MQ_ARCHIVE_REPOSITORY_USER):$(MQ_ARCHIVE_REPOSITORY_CREDENTIAL) docker://$(MQ_IMAGE_DEVSERVER_PPC64LE) | jq -r .Digest))

	$(shell ./$(BUILD_SCRIPTS_PATH)/check-digest.sh -r "$(MQ_IMAGE_DEVSERVER_AMD64_DIGEST)" -n "$(MQ_IMAGE_DEVSERVER_S390X_DIGEST)" -i "$(MQ_IMAGE_DEVSERVER_PPC64LE_DIGEST)")

	$(info $(shell printf "** Determined the built $(MQ_IMAGE_DEVSERVER_AMD64) has a digest of $(MQ_IMAGE_DEVSERVER_AMD64_DIGEST)**"$(END)))
	$(info $(shell printf "** Determined the built $(MQ_IMAGE_DEVSERVER_S390X) has a digest of $(MQ_IMAGE_DEVSERVER_S390X_DIGEST)**"$(END)))
	$(info $(shell printf "** Determined the built $(MQ_IMAGE_DEVSERVER_PPC64LE) has a digest of $(MQ_IMAGE_DEVSERVER_PPC64LE_DIGEST)**"$(END)))
endif
	$(eval MQ_IMAGE_ADVANCEDSERVER_AMD64_DIGEST=$(shell $(COMMAND) run $(NETWORK) skopeo:latest --override-os linux inspect --creds $(MQ_ARCHIVE_REPOSITORY_USER):$(MQ_ARCHIVE_REPOSITORY_CREDENTIAL) docker://$(MQ_IMAGE_ADVANCEDSERVER_AMD64) | jq -r .Digest))
	$(eval MQ_IMAGE_ADVANCEDSERVER_S390X_DIGEST=$(shell $(COMMAND) run $(NETWORK) skopeo:latest --override-os linux inspect --creds $(MQ_ARCHIVE_REPOSITORY_USER):$(MQ_ARCHIVE_REPOSITORY_CREDENTIAL) docker://$(MQ_IMAGE_ADVANCEDSERVER_S390X) | jq -r .Digest))
	$(eval MQ_IMAGE_ADVANCEDSERVER_PPC64LE_DIGEST=$(shell $(COMMAND) run $(NETWORK) skopeo:latest --override-os linux inspect --creds $(MQ_ARCHIVE_REPOSITORY_USER):$(MQ_ARCHIVE_REPOSITORY_CREDENTIAL) docker://$(MQ_IMAGE_ADVANCEDSERVER_PPC64LE) | jq -r .Digest))

	$(shell ./$(BUILD_SCRIPTS_PATH)/check-digest.sh -r "$(MQ_IMAGE_ADVANCEDSERVER_AMD64_DIGEST)" -n "$(MQ_IMAGE_ADVANCEDSERVER_S390X_DIGEST)" -i "$(MQ_IMAGE_ADVANCEDSERVER_PPC64LE_DIGEST)")

	$(info $(shell printf "** Determined the built $(MQ_IMAGE_ADVANCEDSERVER_AMD64) has a digest of $(MQ_IMAGE_ADVANCEDSERVER_AMD64_DIGEST)**"$(END)))
	$(info $(shell printf "** Determined the built $(MQ_IMAGE_ADVANCEDSERVER_S390X) has a digest of $(MQ_IMAGE_ADVANCEDSERVER_S390X_DIGEST)**"$(END)))
	$(info $(shell printf "** Determined the built $(MQ_IMAGE_ADVANCEDSERVER_PPC64LE) has a digest of $(MQ_IMAGE_ADVANCEDSERVER_PPC64LE_DIGEST)**"$(END)))
ifneq "$(LTS)" "true"
	$(info $(shell printf "** Calling script to create fat-manifest for $(MQ_IMAGE_DEVSERVER_MANIFEST)**"$(END)))
	COMMAND=$(COMMAND) PUSH_OPTIONS=$(PUSH_OPTIONS) COMPRESSION_FLAGS="$(COMPRESSION_FLAGS)" ./$(BUILD_SCRIPTS_PATH)/create-manifest-list.sh -r $(MQ_DELIVERY_REGISTRY_HOSTNAME) -n $(MQ_DELIVERY_REGISTRY_NAMESPACE) -i $(MQ_IMAGE_DEVSERVER) -t $(MQ_MANIFEST_TAG) -u $(MQ_ARCHIVE_REPOSITORY_USER) -p $(MQ_ARCHIVE_REPOSITORY_CREDENTIAL)  -d "$(MQ_IMAGE_DEVSERVER_AMD64_DIGEST) $(MQ_IMAGE_DEVSERVER_S390X_DIGEST) $(MQ_IMAGE_DEVSERVER_PPC64LE_DIGEST)" $(END)
endif
	$(info $(shell printf "** Calling script to create fat-manifest for $(MQ_IMAGE_ADVANCEDSERVER_MANIFEST)**"$(END)))
	COMMAND=$(COMMAND) PUSH_OPTIONS=$(PUSH_OPTIONS) COMPRESSION_FLAGS="$(COMPRESSION_FLAGS)" ./$(BUILD_SCRIPTS_PATH)/create-manifest-list.sh -r $(MQ_DELIVERY_REGISTRY_HOSTNAME) -n $(MQ_DELIVERY_REGISTRY_NAMESPACE) -i $(MQ_IMAGE_ADVANCEDSERVER) -t $(MQ_MANIFEST_TAG) -u $(MQ_ARCHIVE_REPOSITORY_USER) -p $(MQ_ARCHIVE_REPOSITORY_CREDENTIAL) -d "$(MQ_IMAGE_ADVANCEDSERVER_AMD64_DIGEST) $(MQ_IMAGE_ADVANCEDSERVER_S390X_DIGEST) $(MQ_IMAGE_ADVANCEDSERVER_PPC64LE_DIGEST)" $(END)

.PHONY: build-manifest
build-manifest:	build-skopeo-container
	$(eval MQ_IMAGE_DEVSERVER_AMD64_DIGEST=$(shell $(COMMAND) run $(NETWORK) skopeo:latest --override-os linux inspect --creds $(MQ_ARCHIVE_REPOSITORY_USER):$(MQ_ARCHIVE_REPOSITORY_CREDENTIAL) docker://$(MQ_IMAGE_DEVSERVER_AMD64) | jq -r .Digest))
	$(eval MQ_IMAGE_DEVSERVER_S390X_DIGEST=$(shell $(COMMAND) run $(NETWORK) skopeo:latest --override-os linux inspect --creds $(MQ_ARCHIVE_REPOSITORY_USER):$(MQ_ARCHIVE_REPOSITORY_CREDENTIAL) docker://$(MQ_IMAGE_DEVSERVER_S390X) | jq -r .Digest))
	$(eval MQ_IMAGE_DEVSERVER_PPC64LE_DIGEST=$(shell $(COMMAND) run $(NETWORK) skopeo:latest --override-os linux inspect --creds $(MQ_ARCHIVE_REPOSITORY_USER):$(MQ_ARCHIVE_REPOSITORY_CREDENTIAL) docker://$(MQ_IMAGE_DEVSERVER_PPC64LE) | jq -r .Digest))
	$(info $(shell printf "** Determined the built $(MQ_IMAGE_DEVSERVER_AMD64) has a digest of $(MQ_IMAGE_DEVSERVER_AMD64_DIGEST)**"$(END)))
	$(info $(shell printf "** Determined the built $(MQ_IMAGE_DEVSERVER_S390X) has a digest of $(MQ_IMAGE_DEVSERVER_S390X_DIGEST)**"$(END)))
	$(info $(shell printf "** Determined the built $(MQ_IMAGE_DEVSERVER_PPC64LE) has a digest of $(MQ_IMAGE_DEVSERVER_PPC64LE_DIGEST)**"$(END)))

	$(eval MQ_IMAGE_ADVANCEDSERVER_AMD64_DIGEST=$(shell $(COMMAND) run $(NETWORK) skopeo:latest --override-os linux inspect --creds $(MQ_ARCHIVE_REPOSITORY_USER):$(MQ_ARCHIVE_REPOSITORY_CREDENTIAL) docker://$(MQ_IMAGE_ADVANCEDSERVER_AMD64) | jq -r .Digest))
	$(eval MQ_IMAGE_ADVANCEDSERVER_S390X_DIGEST=$(shell $(COMMAND) run $(NETWORK) skopeo:latest --override-os linux inspect --creds $(MQ_ARCHIVE_REPOSITORY_USER):$(MQ_ARCHIVE_REPOSITORY_CREDENTIAL) docker://$(MQ_IMAGE_ADVANCEDSERVER_S390X) | jq -r .Digest))
	$(eval MQ_IMAGE_ADVANCEDSERVER_PPC64LE_DIGEST=$(shell $(COMMAND) run $(NETWORK) skopeo:latest --override-os linux inspect --creds $(MQ_ARCHIVE_REPOSITORY_USER):$(MQ_ARCHIVE_REPOSITORY_CREDENTIAL) docker://$(MQ_IMAGE_ADVANCEDSERVER_PPC64LE) | jq -r .Digest))
	$(info $(shell printf "** Determined the built $(MQ_IMAGE_ADVANCEDSERVER_AMD64) has a digest of $(MQ_IMAGE_ADVANCEDSERVER_AMD64_DIGEST)**"$(END)))
	$(info $(shell printf "** Determined the built $(MQ_IMAGE_ADVANCEDSERVER_S390X) has a digest of $(MQ_IMAGE_ADVANCEDSERVER_S390X_DIGEST)**"$(END)))
	$(info $(shell printf "** Determined the built $(MQ_IMAGE_ADVANCEDSERVER_PPC64LE) has a digest of $(MQ_IMAGE_ADVANCEDSERVER_PPC64LE_DIGEST)**"$(END)))

	$(eval MQ_IMAGE_DEVSERVER_MANIFEST_DIGEST=$(shell $(COMMAND) run $(NETWORK) skopeo:latest --override-os linux inspect --creds $(MQ_ARCHIVE_REPOSITORY_USER):$(MQ_ARCHIVE_REPOSITORY_CREDENTIAL) docker://$(MQ_IMAGE_DEVSERVER_MANIFEST_IFIX) | jq -r .Digest))
	$(eval MQ_IMAGE_ADVANCESERVER_MANIFEST_DIGEST=$(shell $(COMMAND) run $(NETWORK) skopeo:latest --override-os linux inspect --creds $(MQ_ARCHIVE_REPOSITORY_USER):$(MQ_ARCHIVE_REPOSITORY_CREDENTIAL) docker://$(MQ_IMAGE_ADVANCESERVER_MANIFEST_IFIX) | jq -r .Digest))
	$(info $(shell printf "** Determined the built has a advanceserver digest for ifix of $(MQ_IMAGE_ADVANCESERVER_MANIFEST_DIGEST)**"$(END)))
	$(info $(shell printf "** Determined the built has a devserver digest for ifix  of $(MQ_IMAGE_DEVSERVER_MANIFEST_DIGEST)**"$(END)))
	@./$(BUILD_SCRIPTS_PATH)/create-build-manifest.sh -f $(BUILD_MANIFEST_FILE) -o $(MQ_MANIFEST_TAG) -t $(MQ_IMAGE_DEVSERVER_AMD64_DIGEST)  -u ${MQ_IMAGE_DEVSERVER_S390X_DIGEST} -p ${MQ_IMAGE_DEVSERVER_PPC64LE_DIGEST} -r ${MQ_IMAGE_DEVSERVER_MANIFEST_DIGEST} -n $(MQ_IMAGE_ADVANCEDSERVER_AMD64_DIGEST) -a ${MQ_IMAGE_ADVANCEDSERVER_S390X_DIGEST} -m ${MQ_IMAGE_ADVANCEDSERVER_PPC64LE_DIGEST} -s ${MQ_IMAGE_ADVANCESERVER_MANIFEST_DIGEST}

.PHONY: build-skopeo-container
build-skopeo-container:
	$(COMMAND) images | grep -q "skopeo"; if [ $$? != 0 ]; then $(COMMAND) build $(NETWORK) -t skopeo:latest ./docker-builds/skopeo/; fi

###############################################################################
# Other targets
###############################################################################

.PHONY: clean
clean:
	rm -rf ./coverage
	rm -rf ./build
	rm -rf ./deps

.PHONY: go-install
go-install:
	ARCH=$(ARCH) ./$(BUILD_SCRIPTS_PATH)/go-install.sh

.PHONY: install-build-deps
install-build-deps:
	ARCH=$(ARCH) ./install-build-deps.sh

.PHONY: install-credential-helper
install-credential-helper:
ifeq ($(ARCH),amd64)
	ARCH=$(ARCH) ./$(BUILD_SCRIPTS_PATH)/install-credential-helper.sh
endif

.PHONY: build-cov
build-cov:
	mkdir -p build
	cd build; go test -c -covermode=count ../cmd/runmqserver

.PHONY: precommit
precommit: fmt lint

.PHONY: fmt
fmt: $(addsuffix /$(wildcard *.go), $(GO_PKG_DIRS))
	go fmt $(addsuffix /..., $(GO_PKG_DIRS))

.PHONY: lint
lint: $(addsuffix /$(wildcard *.go), $(GO_PKG_DIRS))
	@# This expression is necessary because /... includes the vendor directory in golint
	@# As of 11/04/2018 there is an open issue to fix it: https://github.com/golang/lint/issues/320
	golint -set_exit_status $(sort $(dir $(wildcard $(addsuffix /*/*.go, $(GO_PKG_DIRS)))))

.PHONY: gosec
gosec:
	$(info $(SPACER)$(shell printf "Running gosec test"$(END)))
	@gosecrc=0; gosec -fmt=json -out=gosec_results.json cmd/... internal/... 2> /dev/null || gosecrc=$$?; \
	cat gosec_results.json | jq '{"GolangErrors": (.["Golang errors"]|length>0),"Issues":(.Issues|length>0)}' | grep 'true' >/dev/null ;\
	if [ $$? -eq 0 ] || [ $$gosecrc -ne 0 ]; then \
		printf "FAILURE: Issues found running gosec - see gosec_results.json\n" ;\
		cat "gosec_results.json" ;\
		exit 1 ;\
	else \
		printf "gosec found no issues\n" ;\
		cat "gosec_results.json" ;\
	fi

.PHONY: update-release-information
update-release-information:
	sed -i.bak 's/ARG MQ_ARCHIVE=.*-LinuxX64.tar.gz"/ARG MQ_ARCHIVE="downloads\/$(MQ_VERSION)-IBM-MQ-Advanced-for-Developers-Non-Install-LinuxX64.tar.gz"/g' Dockerfile-server && rm Dockerfile-server.bak
	sed -i.bak 's/ibm-mqadvanced-server:.*-amd64/ibm-mqadvanced-server:$(MQ_VERSION)-amd64/g' docs/security.md
	sed -i.bak 's/ibm-mqadvanced-server-dev.*-amd64/ibm-mqadvanced-server-dev:$(MQ_VERSION)-amd64/g' docs/security.md && rm docs/security.md.bak
	sed -i.bak 's/MQ_IMAGE_ADVANCEDSERVER=ibm-mqadvanced-server:.*-amd64/MQ_IMAGE_ADVANCEDSERVER=ibm-mqadvanced-server:$(MQ_VERSION)-amd64/g' docs/testing.md && rm docs/testing.md.bak
	$(eval MQ_VERSION_VR=$(subst $(SPACE),.,$(wordlist 1,2,$(subst .,$(SPACE),$(MQ_VERSION_VRM)))))
	sed -i.bak 's/knowledgecenter\/SSFKSJ_.*\/com/knowledgecenter\/SSFKSJ_${MQ_VERSION_VR}.0\/com/g' docs/usage.md && rm docs/usage.md.bak
	$(eval MQ_VERSION_VRM_FLAT=$(shell echo '${MQ_VERSION_VRM}' | sed "s/\./_/g"))
	sed -i.bak 's/MQ_._._._ARCHIVE_REPOSITORY/MQ_${MQ_VERSION_VRM_FLAT}_ARCHIVE_REPOSITORY/g' .travis.yml && rm .travis.yml.bak

COMMAND_SERVER_VERSION=$(shell $(COMMAND) version --format "{{ .Server.Version }}")
COMMAND_CLIENT_VERSION=$(shell $(COMMAND) version --format "{{ .Client.Version }}")
PODMAN_VERSION=$(shell podman version --format "{{ .Version }}")
.PHONY: command-version
command-version:
# If we're using Docker, then check it's recent enough to support multi-stage builds
ifneq (,$(findstring docker,$(COMMAND)))
	@test "$(word 1,$(subst ., ,$(COMMAND_CLIENT_VERSION)))" -ge "17" || ("$(word 1,$(subst ., ,$(COMMAND_CLIENT_VERSION)))" -eq "17" && "$(word 2,$(subst ., ,$(COMMAND_CLIENT_VERSION)))" -ge "05") || (echo "Error: Docker client 17.05 or greater is required" && exit 1)
	@test "$(word 1,$(subst ., ,$(COMMAND_SERVER_VERSION)))" -ge "17" || ("$(word 1,$(subst ., ,$(COMMAND_SERVER_VERSION)))" -eq "17" && "$(word 2,$(subst ., ,$(COMMAND_CLIENT_VERSION)))" -ge "05") || (echo "Error: Docker server 17.05 or greater is required" && exit 1)
endif
ifneq (,$(findstring podman,$(COMMAND)))
	@test "$(word 1,$(subst ., ,$(PODMAN_VERSION)))" -ge "1" || (echo "Error: Podman version 1.0 or greater is required" && exit 1)
endif

.PHONY: commit-build-manifest
commit-build-manifest:
	@echo "The value of CURRENT_BRANCH is: $(PIPELINE_BRANCH)"
	@echo "The value of BUILD_MANIFEST_FILE is: $(BUILD_MANIFEST_FILE)"
	echo "Checking git status..."
	git status
	echo "Staging changes..."
	git add $(BUILD_MANIFEST_FILE)
	echo "Committing changes..."
	git commit -m "[ci skip]: Commit the digests for ifix and the build-manifest back to the branch"
	echo "Pulling latest changes from remote..."
	git pull --rebase origin $(PIPELINE_BRANCH)
	echo "Pushing changes to remote..."
	git push origin HEAD:$(PIPELINE_BRANCH)
