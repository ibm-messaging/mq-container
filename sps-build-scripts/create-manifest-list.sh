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

set -e

usage="
Usage: create-image-manifest.sh -r hyc-mq-container-team-docker-local.artifactory.swg-devops.com -n foo -i ibm-mqadvanced-server-dev -t test -d \"sha256:038ad492532b099c324b897ce9da31ae0be312a1d0063f6456f2e3143cc4f4b8 sha256:754f466cf2cfc5183ac705689ce6720f27fecd07c97970ba3ec48769acba067d\"

Where:
-r - The image registry hostname
-n - The image registry namespace
-i - The image name
-t - The desired top level manifest tag
-d - A space separated list of sha256 image digests to be included
"

RED="\033[31m"

END="\033[0m"

ERROR=${RED}

CROSS="\xE2\x9C\x97"
REDCROSS=${RED}${CROSS}${END}


while getopts r:n:i:t:d:h:u:p: flag; do
    case "${flag}" in
        r) REGISTRY="${OPTARG}" ;;
        n) NAMESPACE="${OPTARG}" ;;
        i) IMAGE="${OPTARG}" ;;
        t) TAG="${OPTARG}" ;;
        d) DIGESTS="${OPTARG}" ;;
        u) USER="${OPTARG}" ;;
        p) CREDENTIAL="${OPTARG}" ;;
        *) 
            echo "Unknown option: -${flag}" >&2
            ;;
    esac
done


if [[ -z $REGISTRY || -z $NAMESPACE || -z $IMAGE || -z $TAG || -z $DIGESTS ]] ; then 
  printf "${REDCROSS} ${ERROR}Missing parameter!${END}\n"
  printf "${ERROR}$usage${END}\n"
  exit 1
fi

echo "At create-manifest, COMMAND: ${COMMAND}"
if [ "$COMMAND" == "docker" ]; then
  # Docker CLI manifest commands require experimental features to be turned on
  export DOCKER_CLI_EXPERIMENTAL=enabled
fi

MANIFESTS=""
for digest in $DIGESTS ; do \
  MANIFESTS+=" $REGISTRY/$NAMESPACE/$IMAGE@$digest"
done

$COMMAND login $REGISTRY -u $USER -p $CREDENTIAL
$COMMAND manifest create $REGISTRY/$NAMESPACE/$IMAGE:$TAG $MANIFESTS > /dev/null
MANIFEST_DIGEST=$($COMMAND manifest push $PUSH_OPTIONS $COMPRESSION_FLAGS $REGISTRY/$NAMESPACE/$IMAGE:$TAG)

if [ "$COMMAND" = "podman" ]; then
    echo "Inspecting image with skopeo: docker://$REGISTRY/$NAMESPACE/$IMAGE:$TAG"
    MANIFEST_DIGEST=$(skopeo inspect docker://$REGISTRY/$NAMESPACE/$IMAGE:$TAG | jq -r '.Digest')
fi

if [ -z "$MANIFEST_DIGEST" ]; then
  echo "Warning: At create-manifest, Failed to retrieve manifest digest"
else
  echo "MANIFEST_DIGEST: $MANIFEST_DIGEST"
fi
