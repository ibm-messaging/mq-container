# © Copyright IBM Corporation 2017, 2018
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
# BASE_IMAGE is the base image to use for MQ, for example "ubuntu" or "rhel"
BASE_IMAGE ?= ubuntu:16.04
# MQ_VERSION is the fully qualified MQ version number to build
MQ_VERSION ?= 9.0.5.0
# MQ_ARCHIVE is the name of the file, under the downloads directory, from which MQ Advanced can
# be installed. The default value is derived from MQ_VERSION, BASE_IMAGE and architecture
# Does not apply to MQ Advanced for Developers.
MQ_ARCHIVE ?= IBM_MQ_$(MQ_VERSION)_$(MQ_ARCHIVE_TYPE)_$(MQ_ARCHIVE_ARCH).tar.gz
# MQ_ARCHIVE_DEV is the name of the file, under the downloads directory, from which MQ Advanced
# for Developers can be installed
MQ_ARCHIVE_DEV ?= $(MQ_ARCHIVE_DEV_$(MQ_VERSION))
# Options to `go test` for the Docker tests
TEST_OPTS_DOCKER ?=
# MQ_IMAGE_ADVANCEDSERVER is the name and tag of the built MQ Advanced image
MQ_IMAGE_ADVANCEDSERVER ?=mqadvanced-server:$(MQ_VERSION)-$(ARCH)-$(BASE_IMAGE_TAG)
# MQ_IMAGE_DEVSERVER is the name and tag of the built MQ Advanced for Developers image
MQ_IMAGE_DEVSERVER ?=mqadvanced-server-dev:$(MQ_VERSION)-$(ARCH)-$(BASE_IMAGE_TAG)
# MQ_IMAGE_SDK is the name and tag of the built MQ Advanced for Developers SDK image
MQ_IMAGE_SDK ?=mq-sdk:$(MQ_VERSION)-$(ARCH)-$(BASE_IMAGE_TAG)
# MQ_IMAGE_GOLANG_SDK is the name and tag of the built MQ Advanced for Developers SDK image, plus Go tools
MQ_IMAGE_GOLANG_SDK ?=mq-golang-sdk:$(MQ_VERSION)-$(ARCH)-$(BASE_IMAGE_TAG)
# DOCKER is the Docker command to run
DOCKER ?= docker
# MQ_PACKAGES specifies the MQ packages (.deb or .rpm) to install.  Defaults vary on base image.
MQ_PACKAGES ?=

###############################################################################
# Other variables
###############################################################################
# ARCH is the platform architecture (e.g. x86_64, ppc64le or s390x)
ARCH = $(shell uname -m)
# BUILD_SERVER_CONTAINER is the name of the web server container used at build time
BUILD_SERVER_CONTAINER=build-server
# NUM_CPU is the number of CPUs available to Docker.  Used to control how many
# test run in parallel
NUM_CPU = $(or $(shell docker info --format "{{ .NCPU }}"),2)
# BASE_IMAGE_TAG is a normalized version of BASE_IMAGE, suitable for use in a Docker tag
BASE_IMAGE_TAG=$(subst /,-,$(subst :,-,$(BASE_IMAGE)))
MQ_IMAGE_DEVSERVER_BASE=mqadvanced-server-dev-base:$(MQ_VERSION)-$(ARCH)-$(BASE_IMAGE_TAG)
# Docker image name to use for JMS tests
DEV_JMS_IMAGE=mq-dev-jms-test


ifneq (,$(findstring Microsoft,$(shell uname -r)))
	DOWNLOADS_DIR=$(patsubst /mnt/c%,C:%,$(realpath ./downloads/))
else
	DOWNLOADS_DIR=$(realpath ./downloads/)
endif

# Try to figure out which archive to use from the BASE_IMAGE
ifeq "$(findstring ubuntu,$(BASE_IMAGE))" "ubuntu"
	MQ_ARCHIVE_TYPE=UBUNTU
	MQ_ARCHIVE_DEV_PLATFORM=ubuntu
else
	MQ_ARCHIVE_TYPE=LINUX
	MQ_ARCHIVE_DEV_PLATFORM=linux
endif
# Try to figure out which archive to use from the architecture
ifeq "$(ARCH)" "x86_64"
	MQ_ARCHIVE_ARCH=X86-64
else ifeq "$(ARCH)" "ppc64le"
	MQ_ARCHIVE_ARCH=LE_POWER
else ifeq "$(ARCH)" "s390x"
	MQ_ARCHIVE_ARCH=SYSTEM_Z
endif
# Archive names for IBM MQ Advanced for Developers
MQ_ARCHIVE_DEV_9.0.4.0=mqadv_dev904_$(MQ_ARCHIVE_DEV_PLATFORM)_x86-64.tar.gz
MQ_ARCHIVE_DEV_9.0.5.0=mqadv_dev905_$(MQ_ARCHIVE_DEV_PLATFORM)_x86-64.tar.gz

###############################################################################
# Build targets
###############################################################################
.PHONY: vars
vars:
#ifeq "$(findstring ubuntu,$(BASE_IMAGE))","ubuntu"
	@echo $(MQ_ARCHIVE_ARCH)
	@echo $(MQ_ARCHIVE_TYPE)
	@echo $(MQ_ARCHIVE)

.PHONY: default
default: build-devserver test

# Build all components (except incubating ones)
.PHONY: all
all: build-devserver build-advancedserver

.PHONY: test-all
test-all: test-devserver test-advancedserver

.PHONY: precommit
precommit: fmt lint all test-all

.PHONY: devserver
devserver: build-devserver test-devserver

# Build incubating components
.PHONY: incubating
incubating: build-explorer

.PHONY: clean
clean:
	rm -rf ./coverage
	rm -rf ./build
	rm -rf ./deps

downloads/$(MQ_ARCHIVE_DEV):
	$(info $(SPACER)$(shell printf $(TITLE)"Downloading IBM MQ Advanced for Developers"$(END)))
	mkdir -p downloads
	cd downloads; curl -LO https://public.dhe.ibm.com/ibmdl/export/pub/software/websphere/messaging/mqadv/$(MQ_ARCHIVE_DEV)

.PHONY: downloads
downloads: downloads/$(MQ_ARCHIVE_DEV)

.PHONY: deps
deps:
	glide install --strip-vendor

# Vendor Go dependencies for the Docker tests
test/docker/vendor:
	cd test/docker && dep ensure -vendor-only

.PHONY: build-cov
build-cov:
	mkdir -p build
	cd build; go test -c -covermode=count ../cmd/runmqserver

# Shortcut to just run the unit tests
.PHONY: test-unit
test-unit:
	docker build --target builder --file Dockerfile-server .

.PHONY: test-advancedserver
test-advancedserver: test/docker/vendor
	$(info $(SPACER)$(shell printf $(TITLE)"Test $(MQ_IMAGE_ADVANCEDSERVER) on $(shell docker --version)"$(END)))
	cd test/docker && TEST_IMAGE=$(MQ_IMAGE_ADVANCEDSERVER) EXPECTED_LICENSE=Production go test -parallel $(NUM_CPU) $(TEST_OPTS_DOCKER)

.PHONY: build-devjmstest
build-devjmstest:
	$(info $(SPACER)$(shell printf $(TITLE)"Build JMS tests for developer config"$(END)))
	cd test/messaging && docker build --tag $(DEV_JMS_IMAGE) .

.PHONY: test-devserver
test-devserver: test/docker/vendor
	$(info $(SPACER)$(shell printf $(TITLE)"Test $(MQ_IMAGE_DEVSERVER) on $(shell docker --version)"$(END)))
	cd test/docker && TEST_IMAGE=$(MQ_IMAGE_DEVSERVER) EXPECTED_LICENSE=Developer DEV_JMS_IMAGE=$(DEV_JMS_IMAGE) go test -parallel $(NUM_CPU) -tags mqdev $(TEST_OPTS_DOCKER)

coverage:
	mkdir coverage

.PHONY: test-advancedserver-cover
test-advancedserver-cover: test/docker/vendor coverage
	$(info $(SPACER)$(shell printf $(TITLE)"Test $(MQ_IMAGE_ADVANCEDSERVER) with code coverage on $(shell docker --version)"$(END)))
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
	cd test/docker && TEST_IMAGE=$(MQ_IMAGE_ADVANCEDSERVER)-cover TEST_COVER=true go test $(TEST_OPTS_DOCKER)
	echo 'mode: count' > ./coverage/docker.cov
	tail -q -n +2 ./test/docker/coverage/*.cov >> ./coverage/docker.cov
	go tool cover -html=./coverage/docker.cov -o ./coverage/docker.html

	echo 'mode: count' > ./coverage/combined.cov
	tail -q -n +2 ./coverage/unit.cov ./coverage/docker.cov  >> ./coverage/combined.cov
	go tool cover -html=./coverage/combined.cov -o ./coverage/combined.html

define docker-build-mq
	# Create a temporary network to use for the build
	$(DOCKER) network create build
	# Start a web server to host the MQ downloadable (tar.gz) file
	$(DOCKER) run \
	  --rm \
	  --name $(BUILD_SERVER_CONTAINER) \
	  --network build \
	  --network-alias build \
	  --volume $(DOWNLOADS_DIR):/usr/share/nginx/html:ro \
	  --detach \
	  nginx:alpine
	# Build the new image (use --pull to make sure we have the latest base image)
	$(DOCKER) build \
	  --tag $1 \
	  --file $2 \
	  --network build \
	  --build-arg MQ_URL=http://build:80/$3 \
	  --build-arg BASE_IMAGE=$(BASE_IMAGE) \
	  --build-arg BUILDER_IMAGE=$(MQ_IMAGE_GOLANG_SDK) \
	  --label IBM_PRODUCT_ID=$4 \
	  --label IBM_PRODUCT_NAME=$5 \
	  --label IBM_PRODUCT_VERSION=$6 \
	  --build-arg MQ_PACKAGES="$(MQ_PACKAGES)" \
	  . ; $(DOCKER) kill $(BUILD_SERVER_CONTAINER) && $(DOCKER) network rm build
endef

DOCKER_SERVER_VERSION=$(shell docker version --format "{{ .Server.Version }}")
DOCKER_CLIENT_VERSION=$(shell docker version --format "{{ .Client.Version }}")
.PHONY: docker-version
docker-version:
	@test "$(word 1,$(subst ., ,$(DOCKER_CLIENT_VERSION)))" -ge "17" || ("$(word 1,$(subst ., ,$(DOCKER_CLIENT_VERSION)))" -eq "17" && "$(word 2,$(subst ., ,$(DOCKER_CLIENT_VERSION)))" -ge "05") || (echo "Error: Docker client 17.05 or greater is required" && exit 1)
	@test "$(word 1,$(subst ., ,$(DOCKER_SERVER_VERSION)))" -ge "17" || ("$(word 1,$(subst ., ,$(DOCKER_SERVER_VERSION)))" -eq "17" && "$(word 2,$(subst ., ,$(DOCKER_CLIENT_VERSION)))" -ge "05") || (echo "Error: Docker server 17.05 or greater is required" && exit 1)

.PHONY: build-advancedserver
build-advancedserver: downloads/$(MQ_ARCHIVE) docker-version build-golang-sdk
	$(info $(SPACER)$(shell printf $(TITLE)"Build $(MQ_IMAGE_ADVANCEDSERVER)"$(END)))
	$(call docker-build-mq,$(MQ_IMAGE_ADVANCEDSERVER),Dockerfile-server,$(MQ_ARCHIVE),"4486e8c4cc9146fd9b3ce1f14a2dfc5b","IBM MQ Advanced",$(MQ_VERSION))

.PHONY: build-devserver
# Target-specific variable to add web server into devserver image
ifeq "$(findstring ubuntu,$(BASE_IMAGE))" "ubuntu"
build-devserver: MQ_PACKAGES=ibmmq-server ibmmq-java ibmmq-jre ibmmq-gskit ibmmq-msg-.* ibmmq-samples ibmmq-ams ibmmq-web
else 
build-devserver: MQ_PACKAGES=MQSeriesRuntime-*.rpm MQSeriesServer-*.rpm MQSeriesJava*.rpm MQSeriesJRE*.rpm MQSeriesGSKit*.rpm MQSeriesMsg*.rpm MQSeriesSamples*.rpm MQSeriesAMS-*.rpm MQSeriesWeb-*.rpm
endif
build-devserver: downloads/$(MQ_ARCHIVE_DEV) docker-version build-golang-sdk
	$(info $(shell printf $(TITLE)"Build $(MQ_IMAGE_DEVSERVER_BASE)"$(END)))
	$(call docker-build-mq,$(MQ_IMAGE_DEVSERVER_BASE),Dockerfile-server,$(MQ_ARCHIVE_DEV),"98102d16795c4263ad9ca075190a2d4d","IBM MQ Advanced for Developers (Non-Warranted)",$(MQ_VERSION))
	$(DOCKER) build --tag $(MQ_IMAGE_DEVSERVER) --build-arg BASE_IMAGE=$(MQ_IMAGE_DEVSERVER_BASE) --build-arg BUILDER_IMAGE=$(MQ_IMAGE_GOLANG_SDK) --file incubating/mqadvanced-server-dev/Dockerfile .

.PHONY: build-advancedserver-cover
build-advancedserver-cover: docker-version
	$(DOCKER) build --build-arg BASE_IMAGE=$(MQ_IMAGE_ADVANCEDSERVER) -t $(MQ_IMAGE_ADVANCEDSERVER)-cover -f Dockerfile-server.cover .

.PHONY: build-explorer docker-pull
build-explorer: downloads/$(MQ_ARCHIVE_DEV)
	$(call docker-build-mq,mq-explorer:latest-$(ARCH),incubating/mq-explorer/Dockerfile-mq-explorer,$(MQ_ARCHIVE_DEV),"98102d16795c4263ad9ca075190a2d4d","IBM MQ Advanced for Developers (Non-Warranted)",$(MQ_VERSION))

ifeq "$(findstring ubuntu,$(BASE_IMAGE))" "ubuntu"
build-sdk: MQ_PACKAGES=ibmmq-sdk ibmmq-samples build-essential
else 
build-sdk: MQ_PACKAGES=MQSeriesRuntime-*.rpm MQSeriesSDK-*.rpm MQSeriesSamples*.rpm
endif
build-sdk: downloads/$(MQ_ARCHIVE_DEV) docker-version docker-pull
	$(call docker-build-mq,$(MQ_IMAGE_SDK),incubating/mq-sdk/Dockerfile,$(MQ_ARCHIVE_DEV),"98102d16795c4263ad9ca075190a2d4d","IBM MQ Advanced for Developers SDK (Non-Warranted)",$(MQ_VERSION))

build-golang-sdk: downloads/$(MQ_ARCHIVE_DEV) docker-version build-sdk
	$(DOCKER) build --build-arg BASE_IMAGE=$(MQ_IMAGE_SDK) -t $(MQ_IMAGE_GOLANG_SDK) -f incubating/mq-golang-sdk/Dockerfile .
#	$(call docker-build-mq,$(MQ_IMAGE_GOLANG_SDK),incubating/mq-golang-sdk/Dockerfile,$(MQ_ARCHIVE),"98102d16795c4263ad9ca075190a2d4d","IBM MQ Advanced for Developers SDK (Non-Warranted)",$(MQ_VERSION))

docker-pull:
	$(DOCKER) pull $(BASE_IMAGE)

GO_PKG_DIRS = ./cmd ./internal ./test

fmt: $(addsuffix /$(wildcard *.go), $(GO_PKG_DIRS))
	go fmt $(addsuffix /..., $(GO_PKG_DIRS))

lint: $(addsuffix /$(wildcard *.go), $(GO_PKG_DIRS))
	@# This expression is necessary because /... includes the vendor directory in golint
	@# As of 11/04/2018 there is an open issue to fix it: https://github.com/golang/lint/issues/320
	golint -set_exit_status $(sort $(dir $(wildcard $(addsuffix /*/*.go, $(GO_PKG_DIRS)))))

include formatting.mk
