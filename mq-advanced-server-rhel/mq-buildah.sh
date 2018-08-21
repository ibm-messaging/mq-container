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

# Build a RHEL image, using the buildah tool
# Usage
# mq-buildah.sh ARCHIVEFILE PACKAGES

set -x
set -e

###############################################################################
# Setup MQ server working container
###############################################################################

# Use a "scratch" container, so the resulting image has minimal files
# Resulting image won't have yum, for example
readonly ctr_mq=$(buildah from scratch)
readonly mnt_mq=$(buildah mount $ctr_mq)
readonly archive=downloads/$1
readonly packages=$2
readonly tag=$3
readonly version=$4
readonly mqdev=$5

# Initialize yum for use with the scratch container
rpm --root $mnt_mq --initdb

yumdownloader --destdir=/tmp redhat-release-server
rpm --root $mnt_mq -ihv /tmp/redhat-release-server*.rpm || true

###############################################################################
# Install MQ server
###############################################################################

# Install the packages required by MQ
yum install -y --installroot=${mnt_mq} --setopt install_weak_deps=false --setopt=tsflags=nodocs --setopt=override_install_langs=en_US.utf8 \
  bash \
  bc \
  coreutils \
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

# Clean up cached files
yum clean all --installroot=${mnt_mq}
rm -rf ${mnt_mq}/var/cache/yum/*

# Install MQ server packages into the MQ builder image
./mq-advanced-server-rhel/install-mq-rhel.sh ${ctr_mq} "${mnt_mq}" "${archive}" "${packages}"

# Create the directory for MQ configuration files
mkdir -p ${mnt_mq}/etc/mqm
chown 888:888 ${mnt_mq}/etc/mqm

# Install the Go binaries into the image
install --mode 0750 --owner 888 --group 888 ./build/runmqserver ${mnt_mq}/usr/local/bin/
install --mode 6750 --owner 888 --group 888 ./build/chk* ${mnt_mq}/usr/local/bin/
install --mode 0750 --owner 888 --group 888 ./NOTICES.txt ${mnt_mq}/opt/mqm/licenses/notices-container.txt

###############################################################################
# Final Buildah commands
###############################################################################

if [ "$mqdev" = "TRUE" ]; then
  OSTAG="mq messaging developer"
  DISNAME="IBM MQ Advanced Server Developer Edition"
else
  OSTAG="mq messaging"
  DISNAME="IBM MQ Advanced Server"
fi



buildah config \
  --port 1414/tcp \
  --port 9157/tcp \
  --os linux \
  --label architecture=x86_64 \
  --label io.openshift.tags="$OSTAG" \
  --label io.k8s.display-name="$DISNAME" \
  --label io.k8s.description="IBM MQ is messaging middleware that simplifies and accelerates the integration of diverse applications and business data across multiple platforms.  It uses message queues to facilitate the exchanges of information and offers a single messaging solution for cloud, mobile, Internet of Things (IoT) and on-premises environments." \
  --label name="${tag%:*}" \
  --label vendor="IBM" \
  --label version="$version" \
  --env AMQ_ADDITIONAL_JSON_LOG=1 \
  --env LANG=en_US.UTF-8 \
  --env LOG_FORMAT=basic \
  --entrypoint runmqserver \
  $ctr_mq
buildah unmount $ctr_mq
buildah commit $ctr_mq $tag
