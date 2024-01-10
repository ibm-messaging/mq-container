#!/bin/bash

# © Copyright IBM Corporation 2024
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

GREEN="\033[32m"
RED="\033[31m"

END="\033[0m"

RIGHTARROW="\xE2\x96\xB6"
GREENRIGHTARROW=${GREEN}${RIGHTARROW}${END}

TICK="\xE2\x9C\x94"
CROSS="\xE2\x9C\x97"
GREENTICK=${GREEN}${TICK}${END}
REDCROSS=${RED}${CROSS}${END}
printf "${GREENRIGHTARROW} Attempting to trigger new release-checks build\n"

repo_name=$(echo "${TRAVIS_REPO_SLUG}" | cut -d'/' -f2-)

request_body="{ \"request\": { \"message\": \"Trigger release checks build from ${repo_name}:${TRAVIS_BRANCH}\", \"branch\": \"main\", \"merge_mode\": \"deep_merge_append\", \"config\": { \"env\": { \"global\": [ \"EVENT_SOURCE=${repo_name}\" ]}}}}"

request_response="$(curl -X POST -H "Content-Type: application/json" -H "Travis-API-Version: 3" -H "Authorization: token ${TRAVIS_TOKEN}" -d "${request_body}" https://v3.travis.ibm.com/api/repo/mq-cloudpak%2Frelease-checks/requests -o /dev/null -w "%{http_code}" -s)"
if [ "$request_response" != "202" ]; then
    printf "${REDCROSS} ${RED}Could not create new request${END}\n"
    exit 1
else
    printf "${GREENTICK} Successfully created new request\n"
fi