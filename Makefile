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

# Set variable if running on a Red Hat Enterprise Linux host
ifneq ($(wildcard /etc/redhat-release),)
REDHAT_RELEASE = $(shell cat /etc/redhat-release)
ifeq "$(findstring Red Hat,$(REDHAT_RELEASE))" "Red Hat"
    RHEL_HOST = "true"
endif
endif

###############################################################################
# Build targets
###############################################################################

# Targets default to a RHEL image on a RHEL host, or an Ubuntu image everywhere else

.PHONY: build-devserver
ifdef RHEL_HOST
build-devserver: build-devserver-rhel
else
build-devserver: build-devserver-ubuntu
endif

.PHONY: build-advancedserver
ifdef RHEL_HOST
build-advancedserver: build-advancedserver-rhel
else
build-advancedserver: build-advancedserver-ubuntu
endif


.PHONY: test-devserver
ifdef RHEL_HOST
test-devserver: test-devserver-rhel
else
test-devserver: test-devserver-ubuntu
endif

.PHONY: test-advancedserver
ifdef RHEL_HOST
test-advancedserver: test-advancedserver-rhel
else
test-advancedserver: test-advancedserver-ubuntu
endif

.PHONY: build-devjmstest
ifdef RHEL_HOST
build-devjmstest: build-devjmstest-rhel
else
build-devjmstest: build-devjmstest-ubuntu
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

.PHONY: unknownos
unknownos:
	$(info $(SPACER)$(shell printf "ERROR: Unknown OS ("$(BASE_OS)") please run specific make targets"$(END)))
	exit 1

include formatting.mk
