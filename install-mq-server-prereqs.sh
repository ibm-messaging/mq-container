#!/bin/bash
# -*- mode: sh -*-
# © Copyright IBM Corporation 2015, 2020
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

test -f /usr/bin/yum && YUM=true || YUM=false
test -f /usr/bin/microdnf && MICRODNF=true || MICRODNF=false
test -f /usr/bin/rpm && RPM=true || RPM=false
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
  echo "deb ${APT_URL} ${UBUNTU_CODENAME}-security main restricted" >> /etc/apt/sources.list
  # Install additional packages required by MQ, this install process and the runtime scripts
  apt-get update
  apt-get install -y --no-install-recommends \
    bash \
    bc \
    ca-certificates \
    coreutils \
    curl \
    debianutils \
    file \
    findutils \
    gawk \
    grep \
    libc-bin \
    mount \
    passwd \
    procps \
    sed \
    tar \
    util-linux 
fi

if ($RPM); then
  EXTRA_RPMS="bash bc ca-certificates file findutils gawk glibc-common grep ncurses-compat-libs passwd procps-ng sed shadow-utils tar util-linux which"
  # Install additional packages required by MQ, this install process and the runtime scripts
  $YUM && yum -y install --setopt install_weak_deps=false ${EXTRA_RPMS}
  $MICRODNF && microdnf --disableplugin=subscription-manager install ${EXTRA_RPMS}
fi

# Apply any bug fixes not included in base Ubuntu or MQ image.
# Don't upgrade everything based on Docker best practices https://docs.docker.com/engine/userguide/eng-image/dockerfile_best-practices/#run
$UBUNTU && apt-get install -y libapparmor1 libsystemd0 systemd systemd-sysv libudev1 perl-base --only-upgrade
# End of bug fixes

# Clean up cached files
$UBUNTU && rm -rf /var/lib/apt/lists/*
$YUM && yum -y clean all
$YUM && rm -rf /var/cache/yum/*
$MICRODNF && microdnf --disableplugin=subscription-manager clean all
