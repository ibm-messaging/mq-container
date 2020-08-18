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

echo 'Testing Developer image...' && echo -en 'travis_fold:start:test-devserver\\r'
make test-devserver
echo -en 'travis_fold:end:test-devserver\\r'
if [ "$BUILD_ALL" = true ] ; then
    if [[ "$ARCH" = "amd64" || "$ARCH" = "s390x" ]] ; then
        echo 'Testing Production image...' && echo -en 'travis_fold:start:test-advancedserver\\r'
        make test-advancedserver
        echo -en 'travis_fold:end:test-advancedserver\\r'
    fi
fi
echo 'Running gosec scan...' && echo -en 'travis_fold:start:gosec-scan\\r'
if [[ "$ARCH" = "amd64" || "$ARCH" = "s390x" ]] ; then
    make gosec
else
    echo "Gosec not available on ppc64le/s390x...skipping gosec scan"
fi
echo -en 'travis_fold:end:gosec-scan\\r'
