#!/bin/bash
# -*- mode: sh -*-
# © Copyright IBM Corporation 2018, 2019
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

function usage {
  echo "Usage: $0 REDIST-ARCHIVE-NAME TAG"
  exit 20
}

if [ "$#" -ne 2 ]; then
  echo "ERROR: Invalid number of parameters"
  usage
fi

readonly mq_redist_archive=downloads/$1
readonly tag=$2
# Use Red Hat's Go toolset image as the base
readonly ctr_mq=$(buildah from devtools/go-toolset-7-rhel7)
if [ -z "$ctr_mq" ]
then
  echo "ERROR: ctr_mq is empty. Check above output for errors"
  exit 50
fi

readonly mnt_mq_go=$(buildah mount $ctr_mq)
if [ -z "$mnt_mq_go" ]
then
  echo "ERROR: mnt_mq_go is empty. Check above output for errors"
  exit 50
fi

# Install the MQ redistributable client (including header files) into the Go builder image
mkdir -p ${mnt_mq_go}/opt/mqm
tar -xzf ${mq_redist_archive} -C ${mnt_mq_go}/opt/mqm

# Clean up Yum files
rm -rf ${mnt_mq_go}/etc/yum.repos.d/*

buildah unmount ${ctr_mq}
# Set environment variables for MQ/Go compilation
buildah config \
  --os linux \
  --env CGO_CFLAGS="-I/opt/mqm/inc/" \
  --env CGO_LDFLAGS_ALLOW="-Wl,-rpath.*" \
  ${ctr_mq}
buildah commit ${ctr_mq} ${tag}

buildah rm ${ctr_mq}
