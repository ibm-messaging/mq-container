#!/bin/bash

# © Copyright IBM Corporation 2019, 2021
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

if [ "$TRAVIS_PULL_REQUEST" != "false" ]; then
    echo "Not Promoting as we are a pull request"
    exit 0
fi

if [ ! -z $2 ]; then
    export ARCH=$2
fi

function promote_developer {
    echo 'Promoting Developer image...' && echo -en 'travis_fold:start:promote-devserver\\r'
    make promote-devserver
    echo -en 'travis_fold:end:promote-devserver\\r'
}

function promote_production {
    echo 'Promoting Production image...' && echo -en 'travis_fold:start:promote-advancedserver\\r'
    make promote-advancedserver
    echo -en 'travis_fold:end:promote-advancedserver\\r'
}

# call relevant promote function
if [ ! -z $1 ]; then
    case "$1" in
        developer) promote_developer
            ;;
        production) promote_production
            ;;
        *) echo "ERROR: Type ( developer | production ) must be passed to promote.sh"
        exit 1
        ;;
    esac
else
    echo "ERROR: Type ( developer | production ) must be passed to promote.sh"
    exit 1
fi
