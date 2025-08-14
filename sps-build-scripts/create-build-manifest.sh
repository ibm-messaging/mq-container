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



MQ_VERSION_TAG=

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
        f) IMAGE_MANIFEST_FILE=${OPTARG} ;;
        o) MQ_VERSION_TAG=${OPTARG} ;;
        t) MQ_IMAGE_DEVSERVER_AMD64_DIGEST=${OPTARG} ;;
        u) MQ_IMAGE_DEVSERVER_S390X_DIGEST=${OPTARG} ;;
        p) MQ_IMAGE_DEVSERVER_PPC64LE_DIGEST=${OPTARG} ;;
        r) MANIFEST_SHA_DEV=${OPTARG} ;;
        n) MQ_IMAGE_ADVANCEDSERVER_AMD64_DIGEST=${OPTARG} ;;
        a) MQ_IMAGE_ADVANCEDSERVER_S390X_DIGEST=${OPTARG} ;;
        m) MQ_IMAGE_ADVANCEDSERVER_PPC64LE_DIGEST=${OPTARG} ;;
        s) MANIFEST_SHA_ADV=${OPTARG} ;;
        \?) echo "Error: Invalid option -$OPTARG" >&2; exit 1 ;;
    esac
done


MQ_TAG_REMOVED_DOT=$(echo "$MQ_VERSION_TAG" | awk -F'[.-]' '{print $1 "_" $2 "_" $3 "_" $4 "_" $5}')
MQ_VERSION=$(echo "$MQ_VERSION_TAG" | awk -F'[.-]' '{print $1 "." $2 "." $3 "." $4 "-" $5}')

PRODUCTION_TAG="${MQ_VERSION}-${APAR_NUMBER}-${FIX_NUMBER}"

if [[ -z $IMAGE_MANIFEST_FILE ]] ; then
    echo "Error: IMAGE_MANIFEST_FILE is not set."
    exit 1
fi

DIR_PATH=$(dirname "$IMAGE_MANIFEST_FILE")
# Create the directory if it does not exist
if [[ ! -d "$DIR_PATH" ]]; then
  echo "Directory does not exist. Creating directory: $DIR_PATH"
  mkdir -p "$DIR_PATH"

  # Check if the directory creation succeeded
  if [[ $? -ne 0 ]]; then
    echo "Error: Failed to create directory: $DIR_PATH"
    exit 1
  fi
fi

rm -f "$IMAGE_MANIFEST_FILE"
touch   "$IMAGE_MANIFEST_FILE"


DATE_STAMP=$(date --utc '+%Y-%m-%dT%H:%M:%S.%3N%Z' 2>&1) || EXIT_CODE=$?
if [ "${EXIT_CODE}" != "0" ]; then
    DATE_STAMP=$(date -u '+%Y-%m-%dT%H:%M:%S%Z')
fi


echo "Generating build manifest process started"

yq write -i "$IMAGE_MANIFEST_FILE" metadata.createdAt "$DATE_STAMP"
yq write -i "$IMAGE_MANIFEST_FILE" metadata.commitId "$COMMIT_SHA"
yq write -i "$IMAGE_MANIFEST_FILE" metadata.buildId "$PIPELINE_RUN_ID"
yq write -i "$IMAGE_MANIFEST_FILE" metadata.buildUrl "$PIPELINE_RUN_URL"
yq write -i "$IMAGE_MANIFEST_FILE" metadata.stage dev_ifix
yq write -i "$IMAGE_MANIFEST_FILE" "images.operands.mq.${MQ_TAG_REMOVED_DOT}.ibmMQAdvancedServer.name" ibm-mqadvanced-server
yq write -i "$IMAGE_MANIFEST_FILE" "images.operands.mq.${MQ_TAG_REMOVED_DOT}.ibmMQAdvancedServer.productionName" ibm-mqadvanced-server
yq write -i "$IMAGE_MANIFEST_FILE" "images.operands.mq.${MQ_TAG_REMOVED_DOT}.ibmMQAdvancedServer.productionTag" "$PRODUCTION_TAG"
yq write -i "$IMAGE_MANIFEST_FILE" "images.operands.mq.${MQ_TAG_REMOVED_DOT}.ibmMQAdvancedServer.tag" "$MQ_VERSION_TAG"
yq write -i "$IMAGE_MANIFEST_FILE" "images.operands.mq.${MQ_TAG_REMOVED_DOT}.ibmMQAdvancedServer.digests.amd64" "$MQ_IMAGE_ADVANCEDSERVER_AMD64_DIGEST"
yq write -i "$IMAGE_MANIFEST_FILE" "images.operands.mq.${MQ_TAG_REMOVED_DOT}.ibmMQAdvancedServer.digests.s390x" "$MQ_IMAGE_ADVANCEDSERVER_S390X_DIGEST"
yq write -i "$IMAGE_MANIFEST_FILE" "images.operands.mq.${MQ_TAG_REMOVED_DOT}.ibmMQAdvancedServer.digests.ppc64le" "$MQ_IMAGE_ADVANCEDSERVER_PPC64LE_DIGEST"
yq write -i "$IMAGE_MANIFEST_FILE" "images.operands.mq.${MQ_TAG_REMOVED_DOT}.ibmMQAdvancedServer.digests.fatManifest" "$MANIFEST_SHA_ADV"

if [ "$PROMOTE_DEVELOPER_IMAGE_IFIX" = true ]; then
    yq write -i "$IMAGE_MANIFEST_FILE" "images.operands.mq.${MQ_TAG_REMOVED_DOT}.ibmMQAdvancedServerDev.name" ibm-mqadvanced-server-dev
    yq write -i "$IMAGE_MANIFEST_FILE" "images.operands.mq.${MQ_TAG_REMOVED_DOT}.ibmMQAdvancedServerDev.productionName" mq
    yq write -i "$IMAGE_MANIFEST_FILE" "images.operands.mq.${MQ_TAG_REMOVED_DOT}.ibmMQAdvancedServerDev.productionTag" "$PRODUCTION_TAG"
    yq write -i "$IMAGE_MANIFEST_FILE" "images.operands.mq.${MQ_TAG_REMOVED_DOT}.ibmMQAdvancedServerDev.tag" "$MQ_VERSION_TAG"
    yq write -i "$IMAGE_MANIFEST_FILE" "images.operands.mq.${MQ_TAG_REMOVED_DOT}.ibmMQAdvancedServerDev.digests.amd64" "$MQ_IMAGE_DEVSERVER_AMD64_DIGEST"
    yq write -i "$IMAGE_MANIFEST_FILE" "images.operands.mq.${MQ_TAG_REMOVED_DOT}.ibmMQAdvancedServerDev.digests.s390x" "$MQ_IMAGE_DEVSERVER_S390X_DIGEST"
    yq write -i "$IMAGE_MANIFEST_FILE" "images.operands.mq.${MQ_TAG_REMOVED_DOT}.ibmMQAdvancedServerDev.digests.ppc64le" "$MQ_IMAGE_DEVSERVER_PPC64LE_DIGEST"
    yq write -i "$IMAGE_MANIFEST_FILE" "images.operands.mq.${MQ_TAG_REMOVED_DOT}.ibmMQAdvancedServerDev.digests.fatManifest" "$MANIFEST_SHA_DEV"
fi

echo "Generating build manifest process completed"