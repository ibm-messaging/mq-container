# © Copyright IBM Corporation 2017, 2022
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

# This Makefile expects the following to be installed:
# - gcc
# - ldd
# - MQ SDK (mqm_r library, plus header files)

SRC_DIR = src
BUILD_DIR = ./build
ARCH ?= $(if $(findstring x86_64,$(shell uname -m)),amd64,$(if $(findstring aarch64,$(shell uname -m)),aarch64,$(shell uname -m)))

# Flags passed to the C compiler.  Need to use gnu11 to get POSIX functions needed for file locking.
CFLAGS.amd64 := -m64
CFLAGS.ppc64le := -m64
CFLAGS.s390x := -m64
# -m64 is not a valid compiler option on aarch64/arm64 (ARM)
CFLAGS.arm64 :=
CFLAGS += -std=gnu11 -fPIC -Wall ${CFLAGS.${ARCH}}

LIB_MQ = -L/opt/mqm/lib64 -lmqm_r

all: $(BUILD_DIR)/mqsimpleauth.so $(BUILD_DIR)/simpleauth_test $(BUILD_DIR)/log_test

$(BUILD_DIR)/log.o : $(SRC_DIR)/log.c $(SRC_DIR)/log.h
	mkdir -p ${dir $@}
	gcc $(CFLAGS) -c $(SRC_DIR)/log.c -o $@

$(BUILD_DIR)/log_test : $(BUILD_DIR)/log.o
	mkdir -p ${dir $@}
	gcc $(CFLAGS) $(SRC_DIR)/log_test.c $^ -o $@
# Run Logging tests, and print log if they fail
	$@ || (cat log_test*.log && exit 1)

$(BUILD_DIR)/simpleauth.o : $(SRC_DIR)/simpleauth.c $(SRC_DIR)/simpleauth.h
	mkdir -p ${dir $@}
	gcc $(CFLAGS) -c $(SRC_DIR)/simpleauth.c -o $@

$(BUILD_DIR)/simpleauth_test : $(BUILD_DIR)/simpleauth.o $(BUILD_DIR)/log.o
	mkdir -p ${dir $@}
	gcc $(CFLAGS) -lpthread $(SRC_DIR)/simpleauth_test.c $^ -o $@
# Run SimpleAuth tests, and print log if they fail
	$@ || (cat simpleauth_test*.log && exit 1)

$(BUILD_DIR)/mqsimpleauth.so : $(BUILD_DIR)/log.o $(BUILD_DIR)/simpleauth.o
	mkdir -p ${dir $@}
	gcc $(CFLAGS) -I/opt/mqm/inc -D_REENTRANT $(LIB_MQ) -Wl,-rpath,/opt/mqm/lib64 -Wl,-rpath,/usr/lib64 -shared $(SRC_DIR)/mqsimpleauth.c $^ -o $@
	ldd $@
