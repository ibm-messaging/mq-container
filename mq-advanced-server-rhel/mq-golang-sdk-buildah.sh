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

# Build a RHEL image for building Go programs which use MQ

set -ex
readonly mq_archive=downloads/$1
readonly tag=$2
# Use plain RHEL 7 container
# Note: Red Hat's devtools/go-toolset-7-rhel7 image doesn't allow use of 'root'
# user required for installing the MQ SDK
readonly ctr_mq=$(buildah from rhel7)
readonly mnt_mq=$(buildah mount $ctr_mq)

# Add mqm user
groupadd --root $mnt_mq --system --gid 888 mqm
useradd --root $mnt_mq --system --uid 888 --gid mqm mqm
usermod --root $mnt_mq -aG root mqm
usermod --root $mnt_mq -aG mqm root

# Enable Yum repository for "optional" RPMs, which is needed for "golang"
buildah run ${ctr_mq} -- yum-config-manager --enable rhel-7-server-optional-rpms
# Install Go compiler
buildah run ${ctr_mq} -- yum install -y golang git gcc

# Install the MQ SDK into the Go builder image
./mq-advanced-server-rhel/install-mq-rhel.sh ${ctr_mq} "${mnt_mq}" "${mq_archive}" "MQSeriesRuntime-*.rpm MQSeriesSDK-*.rpm MQSeriesSamples*.rpm"
# Clean up Yum files
buildah run ${ctr_mq} -- yum clean all --releasever 7
rm -rf ${mnt_mq}/var/cache/yum/*
buildah unmount ${ctr_mq}
# Set environment variables for MQ/Go compilation
buildah config \
  --os linux \
  --env CGO_CFLAGS="-I/opt/mqm/inc/" \
  --env CGO_LDFLAGS_ALLOW="-Wl,-rpath.*" \
  ${ctr_mq}
buildah commit ${ctr_mq} ${tag}

buildah rm ${ctr_mq}
