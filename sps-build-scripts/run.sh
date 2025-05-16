#!/bin/bash

# © Copyright IBM Corporation 2019, 2025
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

set -e

if [[ -z $PIPELINE_RUN_ID ]] ; then
  if [ "$(uname -m)" = "x86_64" ] ; then export ARCH="amd64" ; else export ARCH=$(uname -m) ; fi
fi

if [ "$PUSH_MANIFEST_ONLY" = true ] ; then
   if [ "$BASE_MQ_LOCKED" = true ] ; then
     printf '\nNot pushing manifest to Artifactory because the stream is locked.\n'
     exit 0
   fi
   echo 'Retrieving remote tagcache'
   ./sps-build-scripts/artifact-util.sh -c ${CACHE_PATH} -u ${REPOSITORY_USER} -p ${REPOSITORY_CREDENTIAL} -f cache/${TAGCACHE_FILE} -l ${SPS_BUILD_DIR}/.tagcache --get
   make push-manifest
   exit 0
fi
if [ "$BUILD_MANIFEST" = true ] ; then
  echo 'Retrieving remote tagcache for building manifest file'
  ./sps-build-scripts/artifact-util.sh -c ${CACHE_PATH} -u ${REPOSITORY_USER} -p ${REPOSITORY_CREDENTIAL} -f cache/${TAGCACHE_FILE} -l ${SPS_BUILD_DIR}/.tagcache --get
  echo 'Preparing build manifest'
  make build-manifest
  make commit-build-manifest
  exit 0
fi

echo 'Downgrading Docker (if necessary)...'
eval "$DOCKER_DOWNGRADE"

## Build images
./sps-build-scripts/build.sh

##sps: Test images
if [[ -z $PIPELINE_RUN_ID ]] ; then
  ./sps-build-scripts/test.sh
else
  if [[ "$ARCH" == "amd64" ]]; then
    ./sps-build-scripts/test.sh
  fi
fi

## Push images
if [ -z "$BUILD_INTERNAL_LEVEL" ] ; then
  if [ "$BUILD_ALL" = true ] ; then
    if [ "$BASE_MQ_LOCKED" = true ] ; then
      printf '\nNot pushing or writing images to Artifactory because the stream is locked.\n'
      exit 0
    fi
   ./sps-build-scripts/push.sh developer
    ./sps-build-scripts/push.sh production
  fi
else
  if [[ "$BUILD_INTERNAL_LEVEL" == *".DE"* ]]; then
    ./sps-build-scripts/push.sh developer
  else
    ./sps-build-scripts/push.sh production
  fi
fi

if [ "$LTS" = true ] ; then
    printf '\nIn CD stream but building LTS image. Do not push LTS image to artifactory\n'
fi
