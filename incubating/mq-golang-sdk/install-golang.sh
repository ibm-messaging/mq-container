#!/bin/bash
# -*- mode: sh -*-
# © Copyright IBM Corporation 2018
#
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

# Fail on any non-zero return code
set -ex

test -f /usr/bin/yum && RHEL=true || RHEL=false
test -f /usr/bin/apt-get && UBUNTU=true || UBUNTU=false

if ($UBUNTU); then
  export DEBIAN_FRONTEND=noninteractive
  # Use a reduced set of apt repositories.
  # This ensures no unsupported code gets installed, and makes the build faster
  source /etc/os-release
  # Figure out the correct apt URL based on the CPU architecture
  CPU_ARCH=$(uname -p)
  if [ ${CPU_ARCH} == "x86_64" ]; then
     APT_URL="http://archive.ubuntu.com/ubuntu/"
  else
     APT_URL="http://ports.ubuntu.com/ubuntu-ports/"
  fi
  # Use a reduced set of apt repositories.
  # This ensures no unsupported code gets installed, and makes the build faster
  echo "deb ${APT_URL} ${UBUNTU_CODENAME} main restricted" > /etc/apt/sources.list
  echo "deb ${APT_URL} ${UBUNTU_CODENAME}-updates main restricted" >> /etc/apt/sources.list
  echo "deb ${APT_URL} ${UBUNTU_CODENAME}-backports main restricted universe" >> /etc/apt/sources.list;
  echo "deb ${APT_URL} ${UBUNTU_CODENAME}-security main restricted" >> /etc/apt/sources.list
  
  apt-get update 
  apt-get install -y --no-install-recommends \
    golang-${GO_VERSION} \
    git \
    ca-certificates \
    curl \ 
    tar
fi

if ($RHEL); then
  # Install additional packages required by MQ, this install process and the runtime scripts
  yum -y install \
    git \
    curl \
    tar \
    gcc
fi

cd /tmp
curl -LO https://storage.googleapis.com/golang/go${GO_VERSION}.linux-amd64.tar.gz
tar -C /usr/local -xzf go${GO_VERSION}.linux-amd64.tar.gz

# Remove any orphaned packages
$UBUNTU && apt-get autoremove -y

# Clean up cached files
$UBUNTU && rm -rf /var/lib/apt/lists/*
$RHEL && yum -y clean all
$RHEL && rm -rf /var/cache/yum/*

# Make the GOLANG directories 
mkdir -p $GOPATH/src $GOPATH/bin
chmod -R 777 $GOPATH