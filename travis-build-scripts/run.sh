#!/bin/bash

# © Copyright IBM Corporation 2019, 2020
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

if [ "$(uname -m)" = "x86_64" ] ; then export ARCH="amd64" ; else export ARCH=$(uname -m) ; fi

if [ "$PUSH_MANIFEST_ONLY" = true ] ; then
  if [ "$BASE_MQ_LOCKED" = true ] ; then
    printf '\nNot pushing manifest to Artifactory because the stream is locked.\n'
    exit 0
  fi
  echo 'Retrieving remote tagcache' && echo -en 'travis_fold:start:retrieve-tag-cache\\r'
  ./travis-build-scripts/artifact-util.sh -c ${CACHE_PATH} -u ${REPOSITORY_USER} -p ${REPOSITORY_CREDENTIAL} -f cache/${TAGCACHE_FILE} -l ./.tagcache --get
  echo -en 'travis_fold:end:retrieve-tag-cache\\r'
  make push-manifest
  exit 0
fi
if [ "$BUILD_MANIFEST" = true ] ; then
  echo 'Retrieving remote tagcache for building manifest file' && echo -en 'travis_fold:start:retrieve-tag-cache\\r'
  ./travis-build-scripts/artifact-util.sh -c ${CACHE_PATH} -u ${REPOSITORY_USER} -p ${REPOSITORY_CREDENTIAL} -f cache/${TAGCACHE_FILE} -l ./.tagcache --get
  echo 'Preparing build manifest'
  make build-manifest
  make commit-build-manifest
  exit 0
fi

echo 'Downgrading Docker (if necessary)...' && echo -en 'travis_fold:start:docker-downgrade\\r'
eval "$DOCKER_DOWNGRADE"
echo -en 'travis_fold:end:docker-downgrade\\r'


## Build images
./travis-build-scripts/build.sh

## Test images
./travis-build-scripts/test.sh

## Push images
if [ -z "$BUILD_INTERNAL_LEVEL" ] ; then
  if [ "$BUILD_ALL" = true ] ; then
    if [ "$BASE_MQ_LOCKED" = true ] ; then
      printf '\nNot pushing or writing images to Artifactory because the stream is locked.\n'
      exit 0
    fi
    ./travis-build-scripts/promote.sh developer
    ./travis-build-scripts/promote.sh production
  fi
else
  if [[ "$BUILD_INTERNAL_LEVEL" == *".DE"* ]]; then
    ./travis-build-scripts/promote.sh developer
  else
    ./travis-build-scripts/promote.sh production
  fi
fi

if [ "$LTS" = true ] ; then
    printf '\nIn CD stream but building LTS image. Do not push LTS image to artifactory\n'
fi
