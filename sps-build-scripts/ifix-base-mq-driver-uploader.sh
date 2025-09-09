#!/bin/bash

# © Copyright IBM Corporation 2024, 2025
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

GREEN="\033[32m"
RED="\033[31m"

END="\033[0m"


RIGHTARROW="\xE2\x96\xB6"

GREENRIGHTARROW=${GREEN}${RIGHTARROW}${END}

TICK="\xE2\x9C\x94"
CROSS="\xE2\x9C\x97"
GREENTICK=${GREEN}${TICK}${END}
REDCROSS=${RED}${CROSS}${END}

printf "${GREENRIGHTARROW} Checking to see if mq build folder ${MQ_SNAPSHOT_NAME} exists in repository ${IFIX_BASE_MQ_DRIVER_ARCHIVE_REPOSITORY}\n"

REMOTE_PATH="${IFIX_BASE_MQ_DRIVER_ARCHIVE_REPOSITORY}/${MQ_SNAPSHOT_NAME}"
FILE_FOUND=$(curl -u "${REPOSITORY_USER}:${REPOSITORY_CREDENTIAL}" -L -X GET "${REMOTE_PATH}" -o /dev/null -w "%{http_code}" -s)

if [ "$FILE_FOUND" -eq 200 ]; then
    printf "${GREENTICK} Build Folder ${MQ_SNAPSHOT_NAME} was found in path ${REMOTE_PATH} \n"
elif [ "$FILE_FOUND" -eq 404 ]; then
    printf "${REDCROSS} Folder ${MQ_SNAPSHOT_NAME} was not found\n"
    mkdir -p tmp
    git clone git@github.ibm.com:mq-cloudpak/pipeline-scripts.git
    cd pipeline-scripts/store-base-ifix-driver
    chmod +x store-base-ifix-driver.sh
    ./store-base-ifix-driver.sh
else
    echo "Unexpected HTTP status code: $FILE_FOUND"
    exit 1
fi

echo "Check and upload base MQ driver process completed."