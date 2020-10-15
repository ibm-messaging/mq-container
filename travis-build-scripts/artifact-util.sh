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

usage="
Usage: artifact-util.sh -c my-registry.com/artifacts/my-project/builds/123 -u me@org.com -p top-secret -f tagcache -l ./.tagcache --upload \"

Where:
-c - Full artifact destination hostname and path
-u - The username to access repository
-p - The password or api-key to access repository
-f - Name of the file in repository
-l - The path and name to the file whose contents is to be pushed or retrieved into
Then one action of either
--check - Check if the file exists
--upload - Upload the contents of a file [-l must be specified]
--get - Get a file and write to a local file [-l must be specified]
--delete - Delet the remote file from repository
"

GREEN="\033[32m"
RED="\033[31m"

END="\033[0m"


RIGHTARROW="\xE2\x96\xB6"
BLUERIGHTARROW=${BLUE}${RIGHTARROW}${END}
GREENRIGHTARROW=${GREEN}${RIGHTARROW}${END}

ERROR=${RED}

TICK="\xE2\x9C\x94"
CROSS="\xE2\x9C\x97"
GREENTICK=${GREEN}${TICK}${END}
REDCROSS=${RED}${CROSS}${END}


SPACER="\n\n"

USER=
CREDENTIAL=
FILE_NAME=
BUILD_ID=
REGISTRY_HOSTNAME=
FILE_LOCATION=

CHECK=false
UPLOAD=false
GET=false
DELETE=false
DELETE_NAMESPACE=false
num_commands_selected=0
while getopts "f:u:p:c:l:-:" flag
do
    case "${flag}" in
        f) FILE_NAME=${OPTARG};;
        u) USER=${OPTARG};;
        p) CREDENTIAL=${OPTARG};;
        c) CACHE_PATH=${OPTARG};;
        l) FILE_LOCATION=${OPTARG};;
        -)
            case "${OPTARG}" in
                check)
                    CHECK=true
                    num_commands_selected=$((num_commands_selected+1))
                    ;;
                upload)
                    UPLOAD=true
                    num_commands_selected=$((num_commands_selected+1))
                    ;;
                get)
                    GET=true
                    num_commands_selected=$((num_commands_selected+1))
                    ;;
                delete) 
                    DELETE=true
                    num_commands_selected=$((num_commands_selected+1))
                    ;;
                delete-namespace)
                    DELETE_NAMESPACE=true
                    num_commands_selected=$((num_commands_selected+1))
                    ;;
                *)
                    if [ "$OPTERR" = 1 ] && [ "${optspec:0:1}" != ":" ]; then
                        echo "Unknown option --${OPTARG}" >&2
                    fi
                    ;;
            esac;;
    esac
done

if [[ $num_commands_selected == 0 || $num_commands_selected -gt 1 ]]; then
    printf "${REDCROSS} ${ERROR}Too many actions specified. Should be one of ${END}--check${ERROR},${END} --get${ERROR},${END} --upload${ERROR} or${END} --delete${ERROR}!${END}\n"
    printf $SPACER
    printf "${ERROR}$usage${END}\n"
    exit 1
fi

if [ "$DELETE_NAMESPACE" != "true" ]; then
    if [[ -z $CACHE_PATH|| -z $USER || -z $CREDENTIAL || -z $FILE_NAME ]] ; then 
    printf "${REDCROSS} ${ERROR}Missing parameter!${END}\n"
    printf "Cache Path:"$CACHE_PATH"\n"
    printf "File name:"$FILE_NAME"\n"
    printf "User":$USER"\n"
    printf $SPACER
    printf "${ERROR}$usage${END}\n"
    exit 1
    fi
fi

REMOTE_PATH="https://${CACHE_PATH}/$TRAVIS_BUILD_ID"

if [ "$CHECK" == "true" ]; then
    printf "${GREENRIGHTARROW} Checking to see if file ${FILE_NAME} exists in repository ${REMOTE_PATH}\n"
    FILE_FOUND=`curl -u ${USER}:${CREDENTIAL} -X GET  "${REMOTE_PATH}/${FILE_NAME}" -o /dev/null -w "%{http_code}" -s`
    if [ "$FILE_FOUND" != "200" ]; then
        printf "${REDCROSS} File ${FILE_NAME} was not found\n"
        exit 1
    else
        printf "${GREENTICK} File ${FILE_NAME} was found\n"
    fi
fi

if [ "$UPLOAD" == "true" ]; then
    printf "${GREENRIGHTARROW} Attempting to upload the file ${FILE_NAME} to repository ${REMOTE_PATH}\n"
    if [[ -z $FILE_LOCATION ]]; then
        printf "${REDCROSS} Location for ${FILE_NAME} was not supplied please do so\n"
        printf $SPACER
        printf "${ERROR}$usage${END}\n"
        exit 1
    fi
    if [ ! -f "$FILE_LOCATION" ]; then
        printf "${REDCROSS} Location supplied ${FILE_LOCATION } for file ${FILE_NAME} did not resolve to a file with contents to upload\n"
        printf $SPACER
        printf "${ERROR}$usage${END}\n"
        exit 1
    fi
    curl -u ${USER}:${CREDENTIAL} -X PUT "$REMOTE_PATH/${FILE_NAME}" -T ${FILE_LOCATION}
fi

if [ "$GET" == "true" ]; then
    printf "${GREENRIGHTARROW} Attempting to download file ${FILE_NAME} from repository ${REMOTE_PATH} to ${FILE_LOCATION}\n"
    if [[ -z $FILE_LOCATION ]]; then
        printf "${REDCROSS} Location for ${FILE_NAME} was not supplied please do so\n"
        printf $SPACER
        printf "${ERROR}$usage${END}\n"
        exit 1
    fi
    curl -u ${USER}:${CREDENTIAL} "$REMOTE_PATH/${FILE_NAME}" -o ${FILE_LOCATION} -s
    if [ $? != 0 ]; then
        printf "${REDCROSS} Failed download\n"
    else
        printf "${GREENTICK} File ${FILE_NAME} was downloaded to ${FILE_LOCATION}\n"
    fi
fi

if [ "$DELETE" == "true" ]; then
    printf "${GREENRIGHTARROW} Checking to see if file ${FILE_NAME} exists in repository ${REMOTE_PATH} before delete\n"
    FILE_FOUND=`curl -u ${USER}:${CREDENTIAL} -X GET  "${REMOTE_PATH}/${FILE_NAME}" -o /dev/null -w "%{http_code}" -s`
    if [ "$FILE_FOUND" != "200" ]; then
        printf "${REDCROSS} File ${FILE_NAME} was not found to delete\n"
        exit 1
    else
        printf "${GREENTICK} File ${FILE_NAME} was found\n"
        printf "${GREENRIGHTARROW} Attempting the delete of ${REMOTE_PATH}/${FILE_NAME}"
        curl -u ${USER}:${CREDENTIAL} -X DELETE  "${REMOTE_PATH}/${FILE_NAME}" -s
        if [ $? != 0 ]; then
            printf "${REDCROSS} Failed delete\n"
        else
            printf "${GREENTICK} File ${FILE_NAME} was deleted from "${REMOTE_PATH}"\n"
        fi
    fi
fi

if [ "$DELETE_NAMESPACE" == "true" ]; then
    printf "${GREENRIGHTARROW} Checking to see if repository ${REMOTE_PATH} exists before delete\n"
    DIR_FOUND=`curl -u ${USER}:${CREDENTIAL} -X GET  "${REMOTE_PATH}" -o /dev/null -w "%{http_code}" -s`
    if [ "$DIR_FOUND" != "200" ]; then
        printf "${REDCROSS} Namespace ${REMOTE_PATH} was not found to delete\n"
        exit 1
    else
        printf "${GREENTICK} Namespace ${REMOTE_PATH} was found\n"
        printf "${GREENRIGHTARROW} Attempting the delete of ${REMOTE_PATH}"
        curl -u ${USER}:${CREDENTIAL} -X DELETE  "${REMOTE_PATH}" -s
        if [ $? != 0 ]; then
            printf "${REDCROSS} Failed delete\n"
        else
            printf "${GREENTICK} Namespace ${REMOTE_PATH} deleted \n"
        fi
    fi
fi
exit 0
