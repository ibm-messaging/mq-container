# © Copyright IBM Corporation 2017, 2019
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
# MQ_VERSION is the fully qualified MQ version number to build
MQ_VERSION ?= 9.1.3.0
# RELEASE shows what release of the container code has been built
# MQ_ARCHIVE is the name of the file, under the downloads directory, from which MQ Advanced can
# be installed. The default value is derived from MQ_VERSION, BASE_IMAGE and architecture
# Does not apply to MQ Advanced for Developers.
MQ_ARCHIVE ?= IBM_MQ_$(MQ_VERSION_VRM)_$(MQ_ARCHIVE_TYPE)_$(MQ_ARCHIVE_ARCH).tar.gz
# MQ_ARCHIVE_DEV is the name of the file, under the downloads directory, from which MQ Advanced
# for Developers can be installed
MQ_ARCHIVE_DEV ?= $(MQ_ARCHIVE_DEV_$(MQ_VERSION))
# MQ_SDK_ARCHIVE specifies the archive to use for building the golang programs.  Defaults vary on developer or advanced.
MQ_SDK_ARCHIVE ?= $(MQ_ARCHIVE_DEV_$(MQ_VERSION))
# Options to `go test` for the Docker tests
TEST_OPTS_DOCKER ?=
# MQ_IMAGE_ADVANCEDSERVER is the name of the built MQ Advanced image
MQ_IMAGE_ADVANCEDSERVER ?=mqadvanced-server
# MQ_IMAGE_DEVSERVER is the name of the built MQ Advanced for Developers image
MQ_IMAGE_DEVSERVER ?=mqadvanced-server-dev
# MQ_TAG is the tag of the built MQ Advanced image & MQ Advanced for Developers image
MQ_TAG ?=$(MQ_VERSION)-$(ARCH)
# MQ_PACKAGES specifies the MQ packages (.deb or .rpm) to install.  Defaults vary on base image.
MQ_PACKAGES ?=MQSeriesRuntime-*.rpm MQSeriesServer-*.rpm MQSeriesJava*.rpm MQSeriesJRE*.rpm MQSeriesGSKit*.rpm MQSeriesMsg*.rpm MQSeriesSamples*.rpm MQSeriesWeb*.rpm MQSeriesAMS-*.rpm
# MQM_UID is the UID to use for the "mqm" user
MQM_UID ?= 888
# COMMAND is the container command to run.  "podman" or "docker"
COMMAND ?=$(shell type -p podman 2>&1 >/dev/null && echo podman || echo docker)
# REGISTRY_USER is the username used to login to the Red Hat registry
REGISTRY_USER ?=
# REGISTRY_PASS is the password used to login to the Red Hat registry
REGISTRY_PASS ?=

###############################################################################
# Other variables
###############################################################################
GO_PKG_DIRS = ./cmd ./internal ./test
MQ_ARCHIVE_TYPE=LINUX
MQ_ARCHIVE_DEV_PLATFORM=linux
# ARCH is the platform architecture (e.g. amd64, ppc64le or s390x)
ARCH=$(if $(findstring x86_64,$(shell uname -m)),amd64,$(shell uname -m))
# BUILD_SERVER_CONTAINER is the name of the web server container used at build time
BUILD_SERVER_CONTAINER=build-server
# NUM_CPU is the number of CPUs available to Docker.  Used to control how many
# test run in parallel
NUM_CPU = $(or $(shell docker info --format "{{ .NCPU }}"),2)
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
# MQ_VERSION_VRM is MQ_VERSION with only the Version, Release and Modifier fields (no Fix field).  e.g. 9.1.3 instead of 9.1.3.0
MQ_VERSION_VRM=$(subst $(SPACE),.,$(wordlist 1,3,$(subst .,$(SPACE),$(MQ_VERSION))))

ifneq (,$(findstring Microsoft,$(shell uname -r)))
	DOWNLOADS_DIR=$(patsubst /mnt/c%,C:%,$(realpath ./downloads/))
else ifneq (,$(findstring Windows,$(shell echo ${OS})))
	DOWNLOADS_DIR=$(shell pwd)/downloads/
else
	DOWNLOADS_DIR=$(realpath ./downloads/)
endif

# Try to figure out which archive to use from the architecture
ifeq "$(ARCH)" "amd64"
	MQ_ARCHIVE_ARCH=X86-64
	MQ_DEV_ARCH=x86-64
else ifeq "$(ARCH)" "ppc64le"
	MQ_ARCHIVE_ARCH=LE_POWER
	MQ_DEV_ARCH=ppcle
else ifeq "$(ARCH)" "s390x"
	MQ_ARCHIVE_ARCH=SYSTEM_Z
	MQ_DEV_ARCH=s390x
endif
# Archive names for IBM MQ Advanced for Developers
MQ_ARCHIVE_DEV_9.1.0.0=mqadv_dev910_$(MQ_ARCHIVE_DEV_PLATFORM)_$(MQ_DEV_ARCH).tar.gz
MQ_ARCHIVE_DEV_9.1.1.0=mqadv_dev911_$(MQ_ARCHIVE_DEV_PLATFORM)_$(MQ_DEV_ARCH).tar.gz
MQ_ARCHIVE_DEV_9.1.2.0=mqadv_dev912_$(MQ_ARCHIVE_DEV_PLATFORM)_$(MQ_DEV_ARCH).tar.gz
MQ_ARCHIVE_DEV_9.1.3.0=mqadv_dev913_$(MQ_ARCHIVE_DEV_PLATFORM)_$(MQ_DEV_ARCH).tar.gz

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

# Build incubating components
.PHONY: incubating
incubating: build-explorer

downloads/$(MQ_ARCHIVE_DEV):
	$(info $(SPACER)$(shell printf $(TITLE)"Downloading IBM MQ Advanced for Developers "$(MQ_VERSION)$(END)))
	mkdir -p downloads
	cd downloads; curl -LO https://public.dhe.ibm.com/ibmdl/export/pub/software/websphere/messaging/mqadv/$(MQ_ARCHIVE_DEV)

downloads/$(MQ_SDK_ARCHIVE):
	$(info $(SPACER)$(shell printf $(TITLE)"Downloading IBM MQ Advanced for Developers "$(MQ_VERSION)$(END)))
	mkdir -p downloads
	cd downloads; curl -LO https://public.dhe.ibm.com/ibmdl/export/pub/software/websphere/messaging/mqadv/$(MQ_SDK_ARCHIVE)

.PHONY: downloads
downloads: downloads/$(MQ_ARCHIVE_DEV) downloads/$(MQ_SDK_ARCHIVE)

# Vendor Go dependencies for the Docker tests
test/docker/vendor:
	cd test/docker && dep ensure -vendor-only

# Shortcut to just run the unit tests
.PHONY: test-unit
test-unit:
	docker build --target builder --file Dockerfile-server .

.PHONY: test-advancedserver
test-advancedserver: test/docker/vendor
	$(info $(SPACER)$(shell printf $(TITLE)"Test $(MQ_IMAGE_ADVANCEDSERVER):$(MQ_TAG) on $(shell docker --version)"$(END)))
	docker inspect $(MQ_IMAGE_ADVANCEDSERVER):$(MQ_TAG)
	cd test/docker && TEST_IMAGE=$(MQ_IMAGE_ADVANCEDSERVER):$(MQ_TAG) EXPECTED_LICENSE=Production go test -parallel $(NUM_CPU) $(TEST_OPTS_DOCKER)

.PHONY: build-devjmstest
build-devjmstest:
	$(info $(SPACER)$(shell printf $(TITLE)"Build JMS tests for developer config"$(END)))
	cd test/messaging && docker build --tag $(DEV_JMS_IMAGE) .

.PHONY: test-devserver
test-devserver: test/docker/vendor
	$(info $(SPACER)$(shell printf $(TITLE)"Test $(MQ_IMAGE_DEVSERVER):$(MQ_TAG) on $(shell docker --version)"$(END)))
	docker inspect $(MQ_IMAGE_DEVSERVER):$(MQ_TAG)
	cd test/docker && TEST_IMAGE=$(MQ_IMAGE_DEVSERVER):$(MQ_TAG) EXPECTED_LICENSE=Developer DEV_JMS_IMAGE=$(DEV_JMS_IMAGE) IBMJRE=true go test -parallel $(NUM_CPU) -tags mqdev $(TEST_OPTS_DOCKER) 

.PHONY: coverage
coverage:
	mkdir coverage

.PHONY: test-advancedserver-cover
test-advancedserver-cover: test/docker/vendor coverage
	$(info $(SPACER)$(shell printf $(TITLE)"Test $(MQ_IMAGE_ADVANCEDSERVER):$(MQ_TAG) with code coverage on $(shell docker --version)"$(END)))
	rm -f ./coverage/unit*.cov
	# Run unit tests with coverage, for each package under 'internal'
	go list -f '{{.Name}}' ./internal/... | xargs -I {} go test -cover -covermode count -coverprofile ./coverage/unit-{}.cov ./internal/{}
#	ls -1 ./cmd | xargs -I {} go test -cover -covermode count -coverprofile ./coverage/unit-{}.cov ./cmd/{}/...
	echo 'mode: count' > ./coverage/unit.cov
	tail -q -n +2 ./coverage/unit-*.cov >> ./coverage/unit.cov
	go tool cover -html=./coverage/unit.cov -o ./coverage/unit.html

	rm -f ./test/docker/coverage/*.cov
	rm -f ./coverage/docker.*
	mkdir -p ./test/docker/coverage/
	cd test/docker && TEST_IMAGE=$(MQ_IMAGE_ADVANCEDSERVER):$(MQ_TAG)-cover TEST_COVER=true go test $(TEST_OPTS_DOCKER)
	echo 'mode: count' > ./coverage/docker.cov
	tail -q -n +2 ./test/docker/coverage/*.cov >> ./coverage/docker.cov
	go tool cover -html=./coverage/docker.cov -o ./coverage/docker.html

	echo 'mode: count' > ./coverage/combined.cov
	tail -q -n +2 ./coverage/unit.cov ./coverage/docker.cov  >> ./coverage/combined.cov
	go tool cover -html=./coverage/combined.cov -o ./coverage/combined.html

# Build an MQ image.  The commands used are slightly different between Docker and Podman
define build-mq
	$(if $(findstring docker,$(COMMAND)), @docker network create build,)
	$(if $(findstring docker,$(COMMAND)), @docker run --rm --name $(BUILD_SERVER_CONTAINER) --network build --network-alias build --volume $(DOWNLOADS_DIR):/usr/share/nginx/html:ro --detach docker.io/nginx:alpine,)
	$(eval EXTRA_ARGS=$(if $(findstring docker,$(COMMAND)), --network build --build-arg MQ_URL=http://build:80/$4, --volume $(DOWNLOADS_DIR):/var/downloads --build-arg MQ_URL=file:///var/downloads/$4))
	# Build the new image
	$(COMMAND) build \
	  --tag $1:$2 \
	  --file $3 \
		$(EXTRA_ARGS) \
	  --build-arg MQ_PACKAGES="$(MQ_PACKAGES)" \
	  --build-arg IMAGE_REVISION="$(IMAGE_REVISION)" \
	  --build-arg IMAGE_SOURCE="$(IMAGE_SOURCE)" \
	  --build-arg IMAGE_TAG="$1:$2" \
	  --build-arg MQM_UID=$(MQM_UID) \
	  --label version=$(MQ_VERSION) \
	  --label name=$1 \
	  --label build-date=$(shell date +%Y-%m-%dT%H:%M:%S%z) \
	  --label architecture="$(ARCH)" \
	  --label run="docker run -d -e LICENSE=accept $1:$2" \
	  --label vcs-ref=$(IMAGE_REVISION) \
	  --label vcs-type=git \
	  --label vcs-url=$(IMAGE_SOURCE) \
	  $(EXTRA_LABELS) \
	  --target $5 \
	  .
	$(if $(findstring docker,$(COMMAND)), @docker kill $(BUILD_SERVER_CONTAINER))
	$(if $(findstring docker,$(COMMAND)), @docker network rm build)
endef

DOCKER_SERVER_VERSION=$(shell docker version --format "{{ .Server.Version }}")
DOCKER_CLIENT_VERSION=$(shell docker version --format "{{ .Client.Version }}")
PODMAN_VERSION=$(shell podman version --format "{{ .Version }}")
.PHONY: command-version
command-version:
# If we're using Docker, then check it's recent enough to support multi-stage builds
ifneq (,$(findstring docker,$(COMMAND)))
	@test "$(word 1,$(subst ., ,$(DOCKER_CLIENT_VERSION)))" -ge "17" || ("$(word 1,$(subst ., ,$(DOCKER_CLIENT_VERSION)))" -eq "17" && "$(word 2,$(subst ., ,$(DOCKER_CLIENT_VERSION)))" -ge "05") || (echo "Error: Docker client 17.05 or greater is required" && exit 1)
	@test "$(word 1,$(subst ., ,$(DOCKER_SERVER_VERSION)))" -ge "17" || ("$(word 1,$(subst ., ,$(DOCKER_SERVER_VERSION)))" -eq "17" && "$(word 2,$(subst ., ,$(DOCKER_CLIENT_VERSION)))" -ge "05") || (echo "Error: Docker server 17.05 or greater is required" && exit 1)
endif
ifneq (,$(findstring podman,$(COMMAND)))
	@test "$(word 1,$(subst ., ,$(PODMAN_VERSION)))" -ge "1" || (echo "Error: Podman version 1.0 or greater is required" && exit 1)
endif

.PHONY: build-advancedserver-host
build-advancedserver-host: build-advancedserver

.PHONY: build-advancedserver
build-advancedserver: registry-login log-build-env downloads/$(MQ_ARCHIVE) command-version
	$(info $(SPACER)$(shell printf $(TITLE)"Build $(MQ_IMAGE_ADVANCEDSERVER):$(MQ_TAG)"$(END)))
	$(call build-mq,$(MQ_IMAGE_ADVANCEDSERVER),$(MQ_TAG),Dockerfile-server,$(MQ_ARCHIVE),mq-server)

.PHONY: build-devserver-host
build-devserver-host: build-devserver

.PHONY: build-devserver
build-devserver: registry-login log-build-env downloads/$(MQ_ARCHIVE_DEV) command-version 
	$(info $(shell printf $(TITLE)"Build $(MQ_IMAGE_DEVSERVER):$(MQ_TAG)"$(END)))
	$(call build-mq,$(MQ_IMAGE_DEVSERVER),$(MQ_TAG),Dockerfile-server,$(MQ_ARCHIVE_DEV),mq-dev-server)

.PHONY: build-advancedserver-cover
build-advancedserver-cover: registry-login command-version
	$(COMMAND) build --build-arg BASE_IMAGE=$(MQ_IMAGE_ADVANCEDSERVER):$(MQ_TAG) -t $(MQ_IMAGE_ADVANCEDSERVER):$(MQ_TAG)-cover -f Dockerfile-server.cover .

.PHONY: build-explorer
build-explorer: registry-login downloads/$(MQ_ARCHIVE_DEV)
	$(call build-mq,mq-explorer,latest-$(ARCH),incubating/mq-explorer/Dockerfile,$(MQ_ARCHIVE_DEV),mq-explorer)

.PHONY: build-sdk
build-sdk: registry-login downloads/$(MQ_ARCHIVE_DEV)
	$(info $(shell printf $(TITLE)"Build $(MQ_IMAGE_SDK)"$(END)))
	$(call build-mq,mq-sdk,$(MQ_TAG),incubating/mq-sdk/Dockerfile,$(MQ_SDK_ARCHIVE),mq-sdk)

.PHONY: registry-login
registry-login:
ifneq ($(REGISTRY_USER),)
	$(COMMAND) login -u $(REGISTRY_USER) -p $(REGISTRY_PASS) registry.redhat.io
endif

.PHONY: log-build-env
log-build-vars:
	$(info $(SPACER)$(shell printf $(TITLE)"Build environment"$(END)))
	@echo ARCH=$(ARCH)
	@echo MQ_VERSION=$(MQ_VERSION)
	@echo MQ_ARCHIVE=$(MQ_ARCHIVE)
	@echo MQ_IMAGE_DEVSERVER=$(MQ_IMAGE_DEVSERVER)
	@echo MQ_IMAGE_ADVANCEDSERVER=$(MQ_IMAGE_ADVANCEDSERVER)
	@echo COMMAND=$(COMMAND)
	@echo MQM_UID=$(MQM_UID)
	@echo REGISTRY_USER=$(REGISTRY_USER)

.PHONY: log-build-env
log-build-env: log-build-vars
	$(info $(SPACER)$(shell printf $(TITLE)"Build environment - $(COMMAND) info"$(END)))
	@echo Command version: $(shell $(COMMAND) --version)
	$(COMMAND) info

include formatting.mk

.PHONY: clean
clean:
	rm -rf ./coverage
	rm -rf ./build
	rm -rf ./deps

.PHONY: deps
deps:
	glide install --strip-vendor

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
gosec: $(info $(SPACER)$(shell printf "Running gosec test"$(END)))
	@gosec -fmt=json -out=gosec_results.json cmd/... internal/... 2> /dev/null ;\
	cat "gosec_results.json" ;\
	cat gosec_results.json | grep HIGH | grep severity > /dev/null ;\
	if [ $$? -eq 0 ]; then \
		printf "\nFAILURE: gosec found files containing HIGH severity issues - see results.json\n" ;\
		exit 1 ;\
	else \
		printf "\ngosec found no HIGH severity issues\n" ;\
	fi ;\
	cat gosec_results.json | grep MEDIUM | grep severity > /dev/null ;\
	if [ $$? -eq 0 ]; then \
		printf "\nFAILURE: gosec found files containing MEDIUM severity issues - see results.json\n" ;\
		exit 1 ;\
	else \
		printf "\ngosec found no MEDIUM severity issues\n" ;\
	fi ;\
	cat gosec_results.json | grep LOW | grep severity > /dev/null;\
	if [ $$? -eq 0 ]; then \
		printf "\nFAILURE: gosec found files containing LOW severity issues - see results.json\n" ;\
		exit 1;\
	else \
		printf "\ngosec found no LOW severity issues\n" ;\
fi ;\

include formatting.mk
