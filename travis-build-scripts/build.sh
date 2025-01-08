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

archive_level_cache_dir="$(mktemp -d)"

get_archive_level() {
  local level_path
  local archive_variable
  archive_variable="$1"
  MQ_ARCHIVE_LEVEL=""
  level_path="${archive_level_cache_dir}/${archive_variable}.level"

  if [[ ! -f "$level_path" ]]; then
    if [[ -z "${REPOSITORY_USER}" || -z "${REPOSITORY_CREDENTIAL}" ]]; then
      echo 'Skipping level lookup as repository credentials not set'
      return
    fi
    if [[ -z "${!archive_variable}" ]]; then
      echo "Skipping level lookup as '\$${archive_variable}' is not set"
      return
    fi
    ./travis-build-scripts/artifact-util.sh -f "${!archive_variable}" -u "${REPOSITORY_USER}" -p "${REPOSITORY_CREDENTIAL}" -l "$level_path" -n snapshot --get-property
  fi
  read -r MQ_ARCHIVE_LEVEL < "$level_path"
  export MQ_ARCHIVE_LEVEL
}

if [[ ("$TRAVIS_BRANCH" == "$MAIN_BRANCH" && "$TRAVIS_PULL_REQUEST" = "false") || "$TRAVIS_BRANCH" == ifix* ]]; then
  echo 'Retrieving global tagcache' && echo -en 'travis_fold:start:tag-cache-retrieve\\r'
  ./travis-build-scripts/artifact-util.sh -c ${CACHE_PATH} -u ${REPOSITORY_USER} -p ${REPOSITORY_CREDENTIAL} -f cache/${TAGCACHE_FILE} -l ./.tagcache --check
  ./travis-build-scripts/artifact-util.sh -c ${CACHE_PATH} -u ${REPOSITORY_USER} -p ${REPOSITORY_CREDENTIAL} -f cache/${TAGCACHE_FILE} -l ./.tagcache --get
  echo -en 'travis_fold:end:tag-cache-retrieve\\r'
fi

if [ -z "$BUILD_INTERNAL_LEVEL" ] ; then
  if [ "$LTS" != true ] ; then
    echo 'Building Developer JMS test image...' && echo -en 'travis_fold:start:build-devjmstest\\r'
    make build-devjmstest
    echo -en 'travis_fold:end:build-devjmstest\\r'
    echo 'Building Developer image...' && echo -en 'travis_fold:start:build-devserver\\r'
    get_archive_level MQ_ARCHIVE_REPOSITORY_DEV
    make build-devserver
    make push-devserver
    echo -en 'travis_fold:end:build-devserver\\r'
  fi
  if [ "$BUILD_ALL" = true ] || [ "$LTS" = true ] ; then
      if [[ "$ARCH" = "amd64" || "$ARCH" = "s390x" || "$ARCH" = "ppc64le" ]] ; then
          echo 'Building Production image...' && echo -en 'travis_fold:start:build-advancedserver\\r'
          get_archive_level MQ_ARCHIVE_REPOSITORY
          make build-advancedserver
          make push-advancedserver
          echo -en 'travis_fold:end:build-advancedserver\\r'
      fi
  fi
else
  echo 'Building Developer JMS test image...' && echo -en 'travis_fold:start:build-devjmstest\\r'
  make build-devjmstest
  echo -en 'travis_fold:end:build-devjmstest\\r'

  if [[ "$BUILD_INTERNAL_LEVEL" == *".DE"* ]]; then
    echo 'Building Developer image...' && echo -en 'travis_fold:start:build-devserver\\r'
    get_archive_level MQ_ARCHIVE_REPOSITORY_DEV
    make build-devserver
    make push-devserver
    echo -en 'travis_fold:end:build-devserver\\r'
  else
    echo 'Building Production image...' && echo -en 'travis_fold:start:build-advancedserver\\r'
    get_archive_level MQ_ARCHIVE_REPOSITORY
    make build-advancedserver
    make push-advancedserver
    echo -en 'travis_fold:end:build-advancedserver\\r'
  fi
fi
