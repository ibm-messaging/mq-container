#!/bin/bash
# -*- mode: sh -*-
# © Copyright IBM Corporation 2015, 2018
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

# If MQ_PACKAGES isn't specifically set, then choose a valid set of defaults
if [ -z $MQ_PACKAGES ]; then
  $UBUNTU && MQ_PACKAGES="ibmmq-server ibmmq-java ibmmq-jre ibmmq-gskit ibmmq-msg-.* ibmmq-samples ibmmq-ams"
  $RHEL && MQ_PACKAGES="MQSeriesRuntime-*.rpm MQSeriesServer-*.rpm MQSeriesJava*.rpm MQSeriesJRE*.rpm MQSeriesGSKit*.rpm MQSeriesMsg*.rpm MQSeriesSamples*.rpm MQSeriesAMS-*.rpm"
fi

if ($UBUNTU); then
  export DEBIAN_FRONTEND=noninteractive
  # Use a reduced set of apt repositories.
  # This ensures no unsupported code gets installed, and makes the build faster
  source /etc/os-release
  echo "deb http://archive.ubuntu.com/ubuntu/ ${UBUNTU_CODENAME} main restricted" > /etc/apt/sources.list
  echo "deb http://archive.ubuntu.com/ubuntu/ ${UBUNTU_CODENAME}-updates main restricted" >> /etc/apt/sources.list
  echo "deb http://archive.ubuntu.com/ubuntu/ ${UBUNTU_CODENAME}-security main restricted" >> /etc/apt/sources.list
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

# Install additional packages required by MQ, this install process and the runtime scripts
$RHEL && yum -y install \
  bash \
  bc \
  ca-certificates \
  coreutils \
  curl \
  file \
  findutils \
  gawk \
  glibc-common \
  grep \
  passwd \
  procps-ng \
  sed \
  tar \
  util-linux

# Download and extract the MQ installation files
DIR_EXTRACT=/tmp/mq
mkdir -p ${DIR_EXTRACT} 
cd ${DIR_EXTRACT}
curl -LO $MQ_URL
tar -zxvf ./*.tar.gz

# Remove packages only needed by this script
$UBUNTU && apt-get purge -y \
  ca-certificates \
  curl

# Note: ca-certificates and curl are installed by default in RHEL

# Remove any orphaned packages
$UBUNTU && apt-get autoremove -y

# Recommended: Create the mqm user ID with a fixed UID and group, so that the file permissions work between different images
$UBUNTU && groupadd --system --gid 999 mqm
$UBUNTU && useradd --system --uid 999 --gid mqm mqm
$RHEL && groupadd --system --gid 888 mqm
$RHEL && useradd --system --uid 888 --gid mqm mqm
usermod -G mqm root

# Find directory containing .deb files
$UBUNTU && DIR_DEB=$(find ${DIR_EXTRACT} -name "*.deb" -printf "%h\n" | sort -u | head -1)
$RHEL && DIR_RPM=$(find ${DIR_EXTRACT} -name "*.rpm" -printf "%h\n" | sort -u | head -1)
# Find location of mqlicense.sh
MQLICENSE=$(find ${DIR_EXTRACT} -name "mqlicense.sh")

# Accept the MQ license
${MQLICENSE} -text_only -accept
$UBUNTU && echo "deb [trusted=yes] file:${DIR_DEB} ./" > /etc/apt/sources.list.d/IBM_MQ.list

# Install MQ using the DEB packages
$UBUNTU && apt-get update
$UBUNTU && apt-get install -y $MQ_PACKAGES

$RHEL && cd $DIR_RPM && rpm -ivh $MQ_PACKAGES

# Remove 32-bit libraries from 64-bit container
find /opt/mqm /var/mqm -type f -exec file {} \; | awk -F: '/ELF 32-bit/{print $1}' | xargs --no-run-if-empty rm -f

# Remove tar.gz files unpacked by RPM postinst scripts
find /opt/mqm -name '*.tar.gz' -delete

# Recommended: Set the default MQ installation (makes the MQ commands available on the PATH)
/opt/mqm/bin/setmqinst -p /opt/mqm -i

# Clean up all the downloaded files
$UBUNTU && rm -f /etc/apt/sources.list.d/IBM_MQ.list
rm -rf ${DIR_EXTRACT}

# Apply any bug fixes not included in base Ubuntu or MQ image.
# Don't upgrade everything based on Docker best practices https://docs.docker.com/engine/userguide/eng-image/dockerfile_best-practices/#run
$UBUNTU && apt-get upgrade -y libsystemd0 systemd systemd-sysv libudev1
# End of bug fixes

# Clean up cached files
$UBUNTU && rm -rf /var/lib/apt/lists/*
$RHEL && yum -y clean all
$RHEL && rm -rf /var/cache/yum/*

# Optional: Update the command prompt with the MQ version
$UBUNTU && echo "mq:$(dspmqver -b -f 2)" > /etc/debian_chroot

# Remove the directory structure under /var/mqm which was created by the installer
rm -rf /var/mqm

# Create the mount point for volumes
mkdir -p /mnt/mqm

# Create the directory for MQ configuration files
mkdir -p /etc/mqm

# Create a symlink for /var/mqm -> /mnt/mqm/data
ln -s /mnt/mqm/data /var/mqm

# Optional: Set these values for the Bluemix Vulnerability Report
sed -i 's/PASS_MAX_DAYS\t99999/PASS_MAX_DAYS\t90/' /etc/login.defs
sed -i 's/PASS_MIN_DAYS\t0/PASS_MIN_DAYS\t1/' /etc/login.defs

$UBUNTU && PAM_FILE=/etc/pam.d/common-password
$RHEL && PAM_FILE=/etc/pam.d/password-auth
sed -i 's/password\t\[success=1 default=ignore\]\tpam_unix\.so obscure sha512/password\t[success=1 default=ignore]\tpam_unix.so obscure sha512 minlen=8/' $PAM_FILE
