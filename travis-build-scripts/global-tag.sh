#!/bin/bash

# © Copyright IBM Corporation 2020
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

echo 'Cacheing MQ tag...' && echo -en 'travis_fold:start:build-cache-mq-tag\\r'
make cache-mq-tag
echo -en 'travis_fold:end:cache-mq-tag\\r'
echo 'Caching tagcache for future stages' && echo -en 'travis_fold:start:tag-cache\\r'
./travis-build-scripts/artifact-util.sh -c ${CACHE_PATH} -u ${REPOSITORY_USER} -p ${REPOSITORY_CREDENTIAL} -f cache/tagcache -l ./.tagcache --upload
echo -en 'travis_fold:end:tag-cache\\r' 
