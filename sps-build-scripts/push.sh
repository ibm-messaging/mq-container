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

if [[ "$PIPELINE_NAMESPACE" == *pr* ]]; then
    echo "Not pushing as we are a pull request"
    exit 0
fi


function push_developer {
    echo 'Pushing Developer image...'
    make push-devserver
}

function push_production {
    echo 'Pushing Production image...'
    make push-advancedserver
}

# call relevant push function
if [ ! -z $1 ]; then
    case "$1" in
        developer) push_developer
            ;;
        production) push_production
            ;;
        *) echo "ERROR: Type ( developer | production ) must be passed to push.sh"
        exit 1
        ;;
    esac
else
    echo "ERROR: Type ( developer | production ) must be passed to push.sh"
    exit 1
fi
