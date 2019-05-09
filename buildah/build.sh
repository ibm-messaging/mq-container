#!/bin/bash
# -*- mode: sh -*-
# © Copyright IBM Corporation 2019
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

# Build a container image from a Dockerfile using Buildah
# If the Docker socket is available, the image will be pushed to Docker once built.

readonly IMAGE=$1:$2
shift
shift
readonly SRC="/src"
readonly OCI_DIR="/var/oci"

echo "****************************************"
echo " Inside the Buildah container"
echo "****************************************"
set -ex
# Build using the supplied options.  Always pass the source directory in, and 
# use it as the build context
buildah build-using-dockerfile --tag ${IMAGE} --volume /src:/src "$@" /src

if [ -e ${OCI_DIR} ]; then
  buildah push ${IMAGE} oci-archive:${OCI_DIR}/${IMAGE}
fi

if [ -e /var/run/docker.sock ]; then
  buildah push ${IMAGE} docker-daemon:${IMAGE}
fi