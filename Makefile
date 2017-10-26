# © Copyright IBM Corporation 2017
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

BUILD_SERVER_CONTAINER=build-server
# Set architecture for Go code.  Don't set GOOS globally, so that tests can be run locally
export GOARCH ?= amd64
DOCKER_TAG_ARCH ?= x86_64
# By default, all Docker client commands are run inside a Docker container.
# This means that newer features of the client can be used, even with an older daemon.
DOCKER ?= docker run --tty --interactive --rm --volume /var/run/docker.sock:/var/run/docker.sock --volume "$(CURDIR)":/var/src --workdir /var/src docker:stable docker
DOCKER_TAG ?= latest-$(DOCKER_TAG_ARCH)
DOCKER_REPO_DEVSERVER ?= mq-devserver
DOCKER_REPO_ADVANCEDSERVER ?= mq-advancedserver
DOCKER_FULL_DEVSERVER = $(DOCKER_REPO_DEVSERVER):$(DOCKER_TAG)
DOCKER_FULL_ADVANCEDSERVER = $(DOCKER_REPO_ADVANCEDSERVER):$(DOCKER_TAG)
# Options to `go test` for the Docker tests
TEST_OPTS_DOCKER ?=
# Options to `go test` for the Kubernetes tests
TEST_OPTS_KUBERNETES ?=

.PHONY: default
default: build-devserver test

# Build all components (except incubating ones)
.PHONY: all
all: build-devserver build-advancedserver

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

downloads/mqadv_dev903_ubuntu_x86-64.tar.gz:
	$(info $(SPACER)$(shell printf $(TITLE)"Downloading IBM MQ Advanced for Developers"$(END)))
	mkdir -p downloads
	cd downloads; curl -LO https://public.dhe.ibm.com/ibmdl/export/pub/software/websphere/messaging/mqadv/mqadv_dev903_ubuntu_x86-64.tar.gz

.PHONY: downloads
downloads: downloads/mqadv_dev903_ubuntu_x86-64.tar.gz

.PHONY: deps
deps:
	glide install --strip-vendor
	cd test/docker && dep ensure -vendor-only
	cd test/kubernetes && dep ensure -vendor-only

build/runmqserver:
	mkdir -p build
	cd build; GOOS=linux go build ../cmd/runmqserver/

build/chkmqready:
	mkdir -p build
	cd build; GOOS=linux go build ../cmd/chkmqready/

build/chkmqhealthy:
	mkdir -p build
	cd build; GOOS=linux go build ../cmd/chkmqhealthy/

.PHONY: build
build: build/runmqserver build/chkmqready build/chkmqhealthy

.PHONY: build-cov
build-cov:
	mkdir -p build
	cd build; go test -c -covermode=count ../cmd/runmqserver

.PHONY: test-advancedserver
test-advancedserver: build
	cd pkg/name && go test
	cd test/docker && TEST_IMAGE=$(DOCKER_FULL_ADVANCEDSERVER) go test $(TEST_OPTS_DOCKER)

.PHONY: test-devserver
test-devserver: build
	$(info $(SPACER)$(shell printf $(TITLE)"Test $(DOCKER_FULL_DEVSERVER)"$(END)))
	cd pkg/name && go test
	cd test/docker && TEST_IMAGE=$(DOCKER_FULL_DEVSERVER) go test

.PHONY: test-kubernetes-devserver
test-kubernetes-devserver: build
	$(call test-kubernetes,$(DOCKER_REPO_DEVSERVER),$(DOCKER_TAG),"../../charts/ibm-mqadvanced-server-dev")

.PHONY: test-kubernetes-advancedserver
test-kubernetes-advancedserver: build
	$(call test-kubernetes,$(DOCKER_REPO_ADVANCEDSERVER),$(DOCKER_TAG),"../../charts/ibm-mqadvanced-server-prod")

define test-kubernetes
	$(info $(SPACER)$(shell printf $(TITLE)"Test $1:$2 on Kubernetes"$(END)))
	cd test/kubernetes && TEST_REPO=$1 TEST_TAG=$2 TEST_CHART=$3 go test $(TEST_OPTS_KUBERNETES)
endef

define docker-build-mq
	# Create a temporary network to use for the build
	$(DOCKER) network create build
	# Start a web server to host the MQ downloadable (tar.gz) file
	$(DOCKER) run \
	  --rm \
	  --name $(BUILD_SERVER_CONTAINER) \
	  --network build \
	  --network-alias build \
	  --volume "$(realpath ./downloads/)":/usr/share/nginx/html:ro \
	  --detach \
	  nginx:alpine
	# Build the new image
	$(DOCKER) build \
	  --tag $1 \
	  --file $2 \
	  --network build \
	  --build-arg MQ_URL=http://build:80/$3 \
	  --build-arg IBM_PRODUCT_ID=$4 \
	  --build-arg IBM_PRODUCT_NAME=$5 \
	  --build-arg IBM_PRODUCT_VERSION=$6 \
	  .
	# Stop the web server (will also remove the container)
	$(DOCKER) kill $(BUILD_SERVER_CONTAINER)
	# Delete the temporary network
	$(DOCKER) network rm build
endef

.PHONY: build-advancedserver
build-advancedserver: build downloads/CNJR7ML.tar.gz
	$(info $(SPACER)$(shell printf $(TITLE)"Build $(DOCKER_FULL_ADVANCEDSERVER)"$(END)))
	$(call docker-build-mq,$(DOCKER_FULL_ADVANCEDSERVER),Dockerfile-server,CNJR7ML.tar.gz,"4486e8c4cc9146fd9b3ce1f14a2dfc5b","IBM MQ Advanced","9.0.3")
	$(DOCKER) tag $(DOCKER_FULL_ADVANCEDSERVER) $(DOCKER_REPO_ADVANCEDSERVER):9.0.3-$(DOCKER_TAG_ARCH)

.PHONY: build-devserver
build-devserver: build downloads/mqadv_dev903_ubuntu_x86-64.tar.gz
	$(info $(shell printf $(TITLE)"Build $(DOCKER_FULL_DEVSERVER)"$(END)))
	$(call docker-build-mq,$(DOCKER_FULL_DEVSERVER),Dockerfile-server,mqadv_dev903_ubuntu_x86-64.tar.gz,"98102d16795c4263ad9ca075190a2d4d","IBM MQ Advanced for Developers (Non-Warranted)","9.0.3")
	$(DOCKER) tag $(DOCKER_FULL_DEVSERVER) $(DOCKER_REPO_DEVSERVER):9.0.3-$(DOCKER_TAG_ARCH)

# .PHONY: build-server
# build-server: build downloads/CNJR7ML.tar.gz
# 	$(call docker-build-mq,mq-server:latest-$(DOCKER_TAG_ARCH),Dockerfile-server,"79afd716d55b4f149a87bec52c9dc1aa","IBM MQ","9.0.3")
# 	$(DOCKER) tag mq-server:latest-$(DOCKER_TAG_ARCH) mq-server:9.0.3-$(DOCKER_TAG_ARCH)

.PHONY: build-advancedserver-cover
build-advancedserver-cover: build-advanced-server build-cov
	$(DOCKER) build -t mq-advancedserver:cover -f Dockerfile-server.cover .

# .PHONY: build-web
# build-web: build downloads/CNJR7ML.tar.gz
# 	$(call docker-build-mq,mq-web:latest-$(DOCKER_TAG_ARCH),Dockerfile-mq-web)

.PHONY: build-explorer
build-explorer: build downloads/mqadv_dev903_ubuntu_x86-64.tar.gz
	$(call docker-build-mq,mq-explorer:latest-$(DOCKER_TAG_ARCH),incubating/mq-explorer/Dockerfile-mq-explorer,mqadv_dev903_ubuntu_x86-64.tar.gz,"98102d16795c4263ad9ca075190a2d4d","IBM MQ Advanced for Developers (Non-Warranted)","9.0.3")

include formatting.mk
