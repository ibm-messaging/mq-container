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
    ./sps-build-scripts/artifact-util.sh -f "${!archive_variable}" -u "${REPOSITORY_USER}" -p "${REPOSITORY_CREDENTIAL}" -l "$level_path" -n snapshot --get-property
  fi
  read -r MQ_ARCHIVE_LEVEL < "$level_path"
  export MQ_ARCHIVE_LEVEL
}

#sps: modify the conditions to use the sps environment variables as we wouldn't have done a make to get the variables set in the Makefile?
if [[ ("$BRANCH" == "$MAIN_BRANCH" && "$PIPELINE_NAMESPACE" != *pr*) || "$BRANCH" == ifix* || "$FEATURE_BUILD_OVERRIDE" = true ]]; then
  echo 'Retrieving global tagcache'
  ./sps-build-scripts/artifact-util.sh -c ${CACHE_PATH} -u ${REPOSITORY_USER} -p ${REPOSITORY_CREDENTIAL} -f cache/${TAGCACHE_FILE} -l ${SPS_BUILD_DIR}/.tagcache --check
  ./sps-build-scripts/artifact-util.sh -c ${CACHE_PATH} -u ${REPOSITORY_USER} -p ${REPOSITORY_CREDENTIAL} -f cache/${TAGCACHE_FILE} -l ${SPS_BUILD_DIR}/.tagcache --get
fi


if [ -z "$BUILD_INTERNAL_LEVEL" ] ; then
  if [ "$LTS" != true ] ; then
    echo 'Building Developer JMS test image...'
    make build-devjmstest
    echo 'Building Developer image...'
    get_archive_level MQ_ARCHIVE_REPOSITORY_DEV
    make build-devserver
  fi
  if [ "$BUILD_ALL" = true ] || [ "$LTS" = true ] ; then
      if [[ "$ARCH" = "amd64" || "$ARCH" = "s390x" || "$ARCH" = "ppc64le" ]] ; then
          echo 'Building Production image...'
          get_archive_level MQ_ARCHIVE_REPOSITORY
          make build-advancedserver
      fi
  fi
else
  echo 'Building Developer JMS test image...'
  make build-devjmstest

  if [[ "$BUILD_INTERNAL_LEVEL" == *".DE"* ]]; then
    echo 'Building Developer image...'
    get_archive_level MQ_ARCHIVE_REPOSITORY_DEV
    make build-devserver
  else
    echo 'Building Production image...'
    get_archive_level MQ_ARCHIVE_REPOSITORY
    make build-advancedserver
  fi
fi
