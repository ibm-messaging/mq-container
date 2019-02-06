#!/bin/bash
# -*- mode: sh -*-
# © Copyright IBM Corporation 2018, 2019
#
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

# Run the Go build script inside the Go container, mounting the source
# directory in

function usage {
  echo "Usage: $0 TAG DevModeFlag"
  exit 20
}

if [ "$#" -ne 2 ]; then
  echo "ERROR: Invalid number of parameters"
  usage
fi

readonly tag=$1
readonly dev=$2

IMAGE_REVISION=${IMAGE_REVISION:="Not Applicable"}
IMAGE_SOURCE=${IMAGE_SOURCE:="Not Applicable"}

podman run \
  --volume ${PWD}:/opt/app-root/src/go/src/github.com/ibm-messaging/mq-container/ \
  --env IMAGE_REVISION="$IMAGE_REVISION" \
  --env IMAGE_SOURCE="$IMAGE_SOURCE" \
  --env MQDEV=${dev} \
  --user $(id -u) \
  --rm \
  --network podman \
  ${tag} \
  bash -c "cd /opt/app-root/src/go/src/github.com/ibm-messaging/mq-container/ && ./mq-advanced-server-rhel/go-build.sh"
