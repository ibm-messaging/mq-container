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

set -e

BINPATH="/usr/local/bin/"
CV_YQ_VERSION=3.3.2
echo "Installing yq..."
curl -LO "https://github.com/mikefarah/yq/releases/download/$CV_YQ_VERSION/yq_linux_amd64"
chmod +x yq_linux_amd64
sudo mv yq_linux_amd64 ${BINPATH}/yq

usage="
Usage: create-image-manifest.sh -f image-manifest.yaml
Where:
-f - The file name to use
"

GREEN="\033[32m"
RED="\033[31m"
BLUE="\033[34m"
PURPLE="\033[35m"
AQUA="\033[36m"

END="\033[0m"

UNDERLINE="\033[4m"
BOLD="\033[1m"
ITALIC="\033[3m"
TITLE="\n"${BLUE}${BOLD}${UNDERLINE}
STEPTITLE=${BLUERIGHTARROW}" "${BOLD}${ITALIC}
SUBSTEPTITLE=${MINIARROW}${MINIARROW}${MINIARROW}" "${ITALIC}
RIGHTARROW="\xE2\x96\xB6"
MINIARROW="\xE2\x96\xBB"
BLUERIGHTARROW=${BLUE}${RIGHTARROW}${END}
GREENRIGHTARROW=${GREEN}${RIGHTARROW}${END}

ERROR=${RED}

TICK="\xE2\x9C\x94"
CROSS="\xE2\x9C\x97"
GREENTICK=${GREEN}${TICK}${END}
REDCROSS=${RED}${CROSS}${END}


SPACER="\n\n"

MQ_VERSION_TAG=
REGISTRY_USER=
REGISTRY_CREDENTIAL=
REGISTRY_HOSTNAME=
REGISTRY_NAMESPACE=


MQ_IMAGE_DEVSERVER_AMD64_DIGEST=
MQ_IMAGE_DEVSERVER_S390X_DIGEST=
MQ_IMAGE_DEVSERVER_PPC64LE_DIGEST=
MANIFEST_SHA_DEV=

MQ_IMAGE_ADVANCEDSERVER_AMD64_DIGEST=
MQ_IMAGE_ADVANCEDSERVER_S390X_DIGEST=
MQ_IMAGE_ADVANCEDSERVER_PPC64LE_DIGEST=
MANIFEST_SHA_ADV=

while getopts f:o:t:u:p:r:n:a:m:s: flag
do
    case "${flag}" in
        f) IMAGE_MANIFEST_FILE=${OPTARG};;
        o) MQ_VERSION_TAG=${OPTARG};;
        t) MQ_IMAGE_DEVSERVER_AMD64_DIGEST=${OPTARG};;
        u) MQ_IMAGE_DEVSERVER_S390X_DIGEST=${OPTARG};;
        p) MQ_IMAGE_DEVSERVER_PPC64LE_DIGEST=${OPTARG};;
        r) MANIFEST_SHA_DEV=${OPTARG};;
        n) MQ_IMAGE_ADVANCEDSERVER_AMD64_DIGEST=${OPTARG};;
        a) MQ_IMAGE_ADVANCEDSERVER_S390X_DIGEST=${OPTARG};;
        m) MQ_IMAGE_ADVANCEDSERVER_PPC64LE_DIGEST=${OPTARG};;
        s) MANIFEST_SHA_ADV=${OPTARG};;
    esac
done

MQ_TAG_REMOVED_DOT=$(echo "$MQ_VERSION_TAG" | awk -F'[.-]' '{print $1 "_" $2 "_" $3 "_" $4 "_" $5}')
MQ_VERSION=$(echo "$MQ_VERSION_TAG" | awk -F'[.-]' '{print $1 "." $2 "." $3 "." $4 "-" $5}')

PRODUCTION_TAG="${MQ_VERSION}-${APAR_NUMBER}-${FIX_NUMBER}"

if [[ -z $IMAGE_MANIFEST_FILE ]] ; then
    printf "${REDCROSS} ${ERROR}You must specify a filename${END}\n"
    printf "${ERROR}$usage${END}\n"
    exit 1
fi

DIR_PATH=$(dirname "$IMAGE_MANIFEST_FILE")
# Create the directory if it does not exist
if [ ! -d "$DIR_PATH" ]; then
  echo "Directory does not exist. Creating directory: $DIR_PATH"
  mkdir -p "$DIR_PATH"

  # Check if the directory creation succeeded
  if [ $? -ne 0 ]; then
    echo "Failed to create directory: $DIR_PATH"
    exit 1
  fi
fi

rm -f $IMAGE_MANIFEST_FILE
touch   $IMAGE_MANIFEST_FILE


DATE_STAMP=`date --utc '+%Y-%m-%dT%H:%M:%S.%3N%Z' 2>&1`  || EXIT_CODE=$?
if [ "${EXIT_CODE}" != "0" ]; then
    DATE_STAMP=`date -u '+%Y-%m-%dT%H:%M:%S%Z'`
fi

echo "Generating build manifest process started"

yq write -i $IMAGE_MANIFEST_FILE metadata.createdAt $DATE_STAMP
yq write -i $IMAGE_MANIFEST_FILE metadata.commitId $TRAVIS_COMMIT
yq write -i $IMAGE_MANIFEST_FILE metadata.travisBuildId $TRAVIS_BUILD_ID
yq write -i $IMAGE_MANIFEST_FILE metadata.travisBuildUrl $TRAVIS_BUILD_WEB_URL
yq write -i $IMAGE_MANIFEST_FILE metadata.stage dev_ifix
yq write -i $IMAGE_MANIFEST_FILE images.operands.mq.${MQ_TAG_REMOVED_DOT}.ibmMQAdvancedServer.name ibm-mqadvanced-server
yq write -i $IMAGE_MANIFEST_FILE images.operands.mq.${MQ_TAG_REMOVED_DOT}.ibmMQAdvancedServer.productionName ibm-mqadvanced-server
yq write -i $IMAGE_MANIFEST_FILE images.operands.mq.${MQ_TAG_REMOVED_DOT}.ibmMQAdvancedServer.productionTag $PRODUCTION_TAG
yq write -i $IMAGE_MANIFEST_FILE images.operands.mq.${MQ_TAG_REMOVED_DOT}.ibmMQAdvancedServer.tag $MQ_VERSION_TAG
yq write -i $IMAGE_MANIFEST_FILE images.operands.mq.${MQ_TAG_REMOVED_DOT}.ibmMQAdvancedServer.digests.amd64 $MQ_IMAGE_ADVANCEDSERVER_AMD64_DIGEST
yq write -i $IMAGE_MANIFEST_FILE images.operands.mq.${MQ_TAG_REMOVED_DOT}.ibmMQAdvancedServer.digests.s390x $MQ_IMAGE_ADVANCEDSERVER_S390X_DIGEST
yq write -i $IMAGE_MANIFEST_FILE images.operands.mq.${MQ_TAG_REMOVED_DOT}.ibmMQAdvancedServer.digests.ppc64le $MQ_IMAGE_ADVANCEDSERVER_PPC64LE_DIGEST
yq write -i $IMAGE_MANIFEST_FILE images.operands.mq.${MQ_TAG_REMOVED_DOT}.ibmMQAdvancedServer.digests.fatManifest $MANIFEST_SHA_ADV
if [ "$PROMOTE_DEVELOPER_IMAGE_IFIX" = true ]; then
    yq write -i $IMAGE_MANIFEST_FILE images.operands.mq.${MQ_TAG_REMOVED_DOT}.ibmMQAdvancedServerDev.name  ibm-mqadvanced-server-dev
    yq write -i $IMAGE_MANIFEST_FILE images.operands.mq.${MQ_TAG_REMOVED_DOT}.ibmMQAdvancedServerDev.productionName  mq
    yq write -i $IMAGE_MANIFEST_FILE images.operands.mq.${MQ_TAG_REMOVED_DOT}.ibmMQAdvancedServerDev.productionTag $PRODUCTION_TAG
    yq write -i $IMAGE_MANIFEST_FILE images.operands.mq.${MQ_TAG_REMOVED_DOT}.ibmMQAdvancedServerDev.tag $MQ_VERSION_TAG
    yq write -i $IMAGE_MANIFEST_FILE images.operands.mq.${MQ_TAG_REMOVED_DOT}.ibmMQAdvancedServerDev.digests.amd64 $MQ_IMAGE_DEVSERVER_AMD64_DIGEST
    yq write -i $IMAGE_MANIFEST_FILE images.operands.mq.${MQ_TAG_REMOVED_DOT}.ibmMQAdvancedServerDev.digests.s390x $MQ_IMAGE_DEVSERVER_S390X_DIGEST
    yq write -i $IMAGE_MANIFEST_FILE images.operands.mq.${MQ_TAG_REMOVED_DOT}.ibmMQAdvancedServerDev.digests.ppc64le $MQ_IMAGE_DEVSERVER_PPC64LE_DIGEST
    yq write -i $IMAGE_MANIFEST_FILE images.operands.mq.${MQ_TAG_REMOVED_DOT}.ibmMQAdvancedServerDev.digests.fatManifest $MANIFEST_SHA_DEV
fi
echo "Generating build manifest process completed"