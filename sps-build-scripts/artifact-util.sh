#!/bin/bash

# © Copyright IBM Corporation 2020, 2025
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

FILE_LOCATION=
PROPERTY_NAME=

CHECK=false
UPLOAD=false
GET=false
GET_PROPERTY=false
DELETE=false
DELETE_NAMESPACE=false
num_commands_selected=0
while getopts "f:u:p:c:l:n:-:" flag
do
    case "${flag}" in
        f) FILE_NAME=${OPTARG};;
        u) USER=${OPTARG};;
        p) CREDENTIAL=${OPTARG};;
        c) CACHE_PATH=${OPTARG};;
        l) FILE_LOCATION=${OPTARG};;
        n) PROPERTY_NAME=${OPTARG};;
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
                get-property)
                    GET_PROPERTY=true
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

#sps modify from build directory to pipeline run id
REMOTE_PATH="https://${CACHE_PATH}/${PIPELINE_RUN_ID}"

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
        printf "${REDCROSS} Location -- ${PWD} -- ${PATH_TO_MQ_TAG_CACHE} supplied ${FILE_LOCATION} for file ${FILE_NAME} did not resolve to a file with contents to upload\n"
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

if [ "$GET_PROPERTY" == "true" ]; then
    if [[ -z $PROPERTY_NAME ]]; then
        printf "${REDCROSS} Property name to retrieve from '${FILE_NAME}' was not supplied please do so\n"
        printf $SPACER
        printf "${ERROR}$usage${END}\n"
        exit 1
    fi
    if [[ -z $FILE_LOCATION ]]; then
        printf "${REDCROSS} File location to store property value in was not supplied please do so\n"
        printf $SPACER
        printf "${ERROR}$usage${END}\n"
        exit 1
    fi
    printf "${GREENRIGHTARROW} Attempting to retrieve ${PROPERTY_NAME} of ${FILE_NAME} from repository ${REMOTE_PATH} and store it in ${FILE_LOCATION}\n"

    query_url="${FILE_NAME}"
    query_url="${query_url/\/artifactory\//\/artifactory\/api\/storage\//}?properties=${PROPERTY_NAME}"
    request_result="$(curl -s -u ${USER}:${CREDENTIAL} "${query_url}")"
    if [ $? != 0 ]; then
        printf "Unable to retrieve properties from ${query_url}"
        exit 1
    else
        printf "${GREENTICK} Properties retrieved from ${query_url}"
    fi

    jq -r '.properties.snapshot|first' <<<"$request_result" > ${FILE_LOCATION}

    if [ $? != 0 ]; then
        printf "Unable to write snapshot property to ${FILE_LOCATION}"
        exit 1
    else
        printf "${GREENTICK} Property written to ${FILE_LOCATION}"
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
