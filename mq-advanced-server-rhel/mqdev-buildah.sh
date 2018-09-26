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
readonly basetag=$1
readonly ctr_mq=$(buildah from $basetag)
readonly mnt_mq=$(buildah mount $ctr_mq)
readonly tag=$2
readonly version=$3


useradd --root $mnt_mq --gid mqm admin
groupadd --root $mnt_mq --system mqclient
useradd --root $mnt_mq --gid mqclient app

buildah run $ctr_mq -- id admin
buildah run $ctr_mq -- sh -c "echo admin:passw0rd | chpasswd"

mkdir -p $mnt_mq/run/runmqdevserver
chown 888:888 $mnt_mq/run/runmqdevserver

# Copy runmqdevserver program
install --mode 0750 --owner 888 --group 888 ./build/runmqdevserver ${mnt_mq}/usr/local/bin/

# Copy template files
cp incubating/mqadvanced-server-dev/*.tpl ${mnt_mq}/etc/mqm/

# Copy web XML files for default developer configuration
cp -R incubating/mqadvanced-server-dev/web/ ${mnt_mq}/etc/mqm/web

###############################################################################
# Final Buildah commands
###############################################################################

buildah config \
  --port 1414/tcp \
  --port 9157/tcp \
  --port 9443/tcp \
  --os linux \
  --label architecture=x86_64 \
  --label io.openshift.tags="mq messaging developer" \
  --label io.k8s.display-name="IBM MQ Advanced Server Developer Edition" \
  --label io.k8s.description="IBM MQ is messaging middleware that simplifies and accelerates the integration of diverse applications and business data across multiple platforms.  It uses message queues to facilitate the exchanges of information and offers a single messaging solution for cloud, mobile, Internet of Things (IoT) and on-premises environments." \
  --label name="${tag%:*}" \
  --label vendor="IBM" \
  --label version="$version" \
  --label release="1" \
  --label run="docker run -d -e LICENSE=accept --name ibm-mq-dev ${tag%:*}" \
  --label summary="IBM MQ Advanced Server Developer Edition" \
  --label description="IBM MQ is messaging middleware that simplifies and accelerates the integration of diverse applications and business data across multiple platforms.  It uses message queues to facilitate the exchanges of information and offers a single messaging solution for cloud, mobile, Internet of Things (IoT) and on-premises environments." \
  --env AMQ_ADDITIONAL_JSON_LOG=1 \
  --env LANG=en_US.UTF-8 \
  --env LOG_FORMAT=basic \
  --env MQ_ADMIN_PASSWORD=passw0rd \
  --env MQ_DEV=true \
  --entrypoint runmqdevserver \
  --user root \
  $ctr_mq
buildah unmount $ctr_mq
buildah commit $ctr_mq $tag

buildah rm $ctr_mq
