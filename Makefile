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
# Variables
###############################################################################
GO_PKG_DIRS = ./cmd ./internal ./test

BASE_OS = $(shell cat /etc/*-release | grep ID=)
ifeq "$(findstring ubuntu,$(BASE_OS))" "ubuntu"
	BASE_OS=UBUNTU
else ifeq "$(findstring rhel,$(BASE_OS))" "rhel"
	BASE_OS=RHEL
else
	BASE_OS=UNKNOWN
endif



###############################################################################
# Build targets
###############################################################################

# default to building UBUNTU as this was the default for the previous Makefile
.PHONY: build-devserver
ifeq ($(BASE_OS),UBUNTU)
build-devserver: build-devserver-ubuntu
else ifeq ($(BASE_OS),RHEL)
build-devserver: build-devserver-rhel
else
build-devserver: unknownos
endif

.PHONY: build-advancedserver
ifeq ($(BASE_OS),UBUNTU)
build-advancedserver: build-advancedserver-ubuntu
else ifeq ($(BASE_OS),RHEL)
build-advancedserver: build-advancedserver-rhel
else
build-advancedserver: unknownos
endif


.PHONY: test-devserver
ifeq ($(BASE_OS),UBUNTU)
test-devserver: test-devserver-ubuntu
else ifeq ($(BASE_OS),RHEL)
test-devserver: test-devserver-rhel
else
test-devserver: unknownos
endif

.PHONY: test-advancedserver
ifeq ($(BASE_OS),UBUNTU)
test-advancedserver: test-advancedserver-ubuntu
else ifeq ($(BASE_OS),RHEL)
test-advancedserver: test-advancedserver-rhel
else
test-advancedserver: unknownos
endif

.PHONY: build-devjmstest
ifeq ($(BASE_OS),UBUNTU)
build-devjmstest: build-devjmstest-ubuntu
else ifeq ($(BASE_OS),RHEL)
build-devjmstest: build-devjmstest-rhel
else
build-devjmstest: unknownos
endif

# UBUNTU building targets
.PHONY: build-devserver-ubuntu
build-devserver-ubuntu: 
	$(MAKE) -f Makefile-UBUNTU build-devserver

.PHONY: test-devserver-ubuntu
test-devserver-ubuntu: 
	$(MAKE) -f Makefile-UBUNTU test-devserver

.PHONY: build-devjmstest-ubuntu
	$(MAKE) -f Makefile-UBUNTU build-devjmstest

.PHONY: build-advancedserver-ubuntu
build-advancedserver-ubuntu: 
	$(MAKE) -f Makefile-UBUNTU build-advancedserver

.PHONY: test-advancedserver-ubuntu
test-advancedserver-ubuntu: 
	$(MAKE) -f Makefile-UBUNTU test-advancedserver

.PHONY: build-devjmstest-ubuntu
build-devjmstest-ubuntu:
	$(MAKE) -f Makefile-UBUNTU build-devjmstest

# RHEL building targets
.PHONY: build-devserver-rhel
build-devserver-rhel: 
	$(MAKE) -f Makefile-RHEL build-devserver

.PHONY: test-devserver-rhel
test-devserver-rhel: 
	$(MAKE) -f Makefile-RHEL test-devserver

.PHONY: build-advancedserver-rhel
build-advancedserver-rhel: 
	$(MAKE) -f Makefile-RHEL build-advancedserver

.PHONY: test-advancedserver-rhel
test-advancedserver-rhel: 
	$(MAKE) -f Makefile-RHEL test-advancedserver

.PHONY: build-devjmstest-rhel
build-devjmstest-rhel:
	$(MAKE) -f Makefile-RHEL build-devjmstest

# Common targets
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

.PHONY: unknownos
unknownos:
	$(info $(SPACER)$(shell printf "ERROR: Unknown OS ("$(BASE_OS)") please run specific make targets"$(END)))
	exit 1

include formatting.mk
