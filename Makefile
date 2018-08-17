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
# Build targets
###############################################################################

# default to building UBUNTU as this was the default for the previous Makefile
.PHONY: build-devserver
build-devserver: build-devserver-ubuntu

.PHONY: build-advancedserver
build-advancedserver: build-advancedserver-ubuntu

.PHONY: test-devserver
test-devserver: test-devserver-ubuntu

.PHONY: test-advancedserver
test-advancedserver: test-advancedserver-ubuntu


# UBUNTU building targets
.PHONY: build-devserver-ubuntu
build-devserver-ubuntu: 
	$(MAKE) -f Makefile-UBUNTU build-devserver

.PHONY: test-devserver-ubuntu
test-devserver-ubuntu: 
	$(MAKE) -f Makefile-UBUNTU test-devserver

.PHONY: build-devjmstest
build-devjmstest:
	$(MAKE) -f Makefile-UBUNTU build-devjmstest

.PHONY: build-devjmstest-ubuntu
	$(MAKE) -f Makefile-UBUNTU build-devjmstest

.PHONY: build-advancedserver-ubuntu
build-advancedserver-ubuntu: 
	$(MAKE) -f Makefile-UBUNTU build-advancedserver

.PHONY: test-advancedserver-ubuntu
test-advancedserver-ubuntu: 
	$(MAKE) -f Makefile-UBUNTU test-advancedserver

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
