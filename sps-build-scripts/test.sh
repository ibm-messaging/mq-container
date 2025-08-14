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
#Adding SKIP_UNIT_TEST parameter which can be set in the environment to skip running the unit tests

if [ ! "$SKIP_UNIT_TEST" ] ; then
  if [ -z "$BUILD_INTERNAL_LEVEL" ] ; then
    if [ "$LTS" != true ] ; then
      echo 'Testing Developer image... start:test-devserver'
      make test-devserver
      echo 'end:test-devserver'
    fi
    if [ "$BUILD_ALL" = true ] || [ "$LTS" = true ] ; then
      if [[ "$ARCH" = "amd64" || "$ARCH" = "s390x" || "$ARCH" = "ppc64le" ]] ; then
          echo 'Testing Production image...'
          make test-advancedserver
      fi
    fi
  else
    if [[ "$BUILD_INTERNAL_LEVEL" == *".DE"* ]]; then
      echo 'Testing Developer image... start:test-devserver'
      make test-devserver
      echo -en 'end:test-devserver'
    else
      echo 'Testing Production image...start:test-advancedserver'
      make test-advancedserver
      echo -en 'end:test-advancedserver'
    fi
  fi
else
  echo "Skipping unit tests as SKIP_UNIT_TEST is set"
fi
echo 'Running gosec scan...'
if [ "$ARCH" = "amd64" ] ; then
    make gosec
else
    echo "Gosec not available on ppc64le/s390x...skipping gosec scan"
fi
echo -en 'end:gosec-scan\\r'
