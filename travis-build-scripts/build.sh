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

echo 'Building Developer JMS test image...' && echo -en 'travis_fold:start:build-devjmstest\\r'
make build-devjmstest
echo -en 'travis_fold:end:build-devjmstest\\r'
echo 'Building Developer image...' && echo -en 'travis_fold:start:build-devserver\\r'
make build-devserver
echo -en 'travis_fold:end:build-devserver\\r'
if [ "$BUILD_ALL" = true ] ; then
    if [ "$ARCH" = "amd64" ] ; then
        echo 'Building Production image...' && echo -en 'travis_fold:start:build-advancedserver\\r'
        make build-advancedserver
        echo -en 'travis_fold:end:build-advancedserver\\r'
    fi
fi
