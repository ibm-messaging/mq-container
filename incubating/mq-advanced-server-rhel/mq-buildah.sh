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

set -x
set -e

MQ_ARCHIVE=downloads/mqadv_dev910_linux_x86-64.tar.gz

###############################################################################
# Setup MQ server working container
###############################################################################

# Use a "scratch" container, so the resulting image has minimal files
# Resulting image won't have yum, for example
readonly ctr_mq=$(buildah from scratch)
readonly mnt_mq=$(buildah mount $ctr_mq)

# Initialize yum for use with the scratch container
rpm --root $mnt_mq --initdb

# TODO: eek
yum install yum-utils
yumdownloader --destdir=/tmp redhat-release-server
rpm --root $mnt_mq -ihv /tmp/redhat-release-server*.rpm || true

###############################################################################
# Install MQ server
###############################################################################

# Install the packages required by MQ
yum install -y --installroot=${mnt_mq} --releasever 7 --setopt install_weak_deps=false --setopt=tsflags=nodocs --setopt=override_install_langs=en_US.utf8 \
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
yum clean all --installroot=${mnt_mq} --releasever 7
rm -rf ${mnt_mq}/var/cache/yum/*

# Install MQ server packages into the MQ builder image
./install-mq-rhel.sh ${ctr_mq} "${mnt_mq}" "${MQ_ARCHIVE}" "MQSeriesRuntime-*.rpm MQSeriesServer-*.rpm MQSeriesJava*.rpm MQSeriesJRE*.rpm MQSeriesGSKit*.rpm MQSeriesMsg*.rpm MQSeriesSamples*.rpm MQSeriesAMS-*.rpm"

# Remove the directory structure under /var/mqm which was created by the installer
rm -rf ${mnt_mq}/var/mqm

# Create the mount point for volumes
mkdir -p ${mnt_mq}/mnt/mqm

# Create the directory for MQ configuration files
mkdir -p ${mnt_mq}/etc/mqm

# Create a symlink for /var/mqm -> /mnt/mqm/data
buildah run $ctr ln -s /mnt/mqm/data /var/mqm

# Install the Go binaries into the image
install --mode 0750 --owner 888 --group 888 ../../runmqserver ${mnt_mq}/usr/local/bin/
install --mode 6750 --owner 888 --group 888 ../../chk* ${mnt_mq}/usr/local/bin/
install --mode 0750 --owner 888 --group 888 ../../NOTICES.txt ${mnt_mq}/opt/mqm/licenses/notices-container.txt
# cp runmqserver $mnt_mq/usr/local/bin/
# cp chkmq* $mnt_mq/usr/local/bin/
# cp NOTICES.txt $mnt_mq/opt/mqm/licenses/notices-container.txt
# chmod ug+x $mnt_mq/usr/local/bin/runmqserver
# chown mqm:mqm $mnt_mq/usr/local/bin/*mq*
# chmod ug+xs $mnt_mq/usr/local/bin/chkmq*

###############################################################################
# Final Buildah commands
###############################################################################

buildah config \
  --port 1414/tcp \
  --port 9157/tcp \
  --os linux \
  --label architecture=x86_64 \
  --label io.openshift.tags="mq messaging" \
  --label io.k8s.display-name="IBM MQ Advanced Server" \
  --label io.k8s.description="IBM MQ is messaging middleware that simplifies and accelerates the integration of diverse applications and business data across multiple platforms.  It uses message queues to facilitate the exchanges of information and offers a single messaging solution for cloud, mobile, Internet of Things (IoT) and on-premises environments." \
  --label name="mqadvanced-server" \
  --label vendor="IBM" \
  --label version="9.1.0.0" \
  --env AMQ_ADDITIONAL_JSON_LOG=1 \
  --env LANG=en_US.UTF-8 \
  --env LOG_FORMAT=basic \
  --entrypoint runmqserver \
  --user 888 \
  $ctr_mq
buildah unmount $ctr_mq
buildah commit $ctr_mq mymq

# TODO: Leaves the working container lying around.  Good for dev.