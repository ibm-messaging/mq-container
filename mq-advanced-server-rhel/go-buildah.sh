#!/bin/bash
# -*- mode: sh -*-
# © Copyright IBM Corporation 2018
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

readonly tag=$1

if [[ $* < 1 ]]; then
  printf "Usage: $0 TAG\n" >&2
  printf "Where:\n" >&2
  printf "  TAG\tTag for image containing MQ SDK and Go compiler\n" >&2
  exit 1
fi

# Copy the go-build.sh script into the Go builder image
install --mode 0755 go-build.sh $mnt/usr/local/bin/
# Run the Go build script inside the Go container, mounting the source
# directory in
podman run \
  --volume ${PWD}/../..:/go/src/github.com/ibm-messaging/mq-container/ \
  --env GOPATH=/go \
  ${tag} \
  bash -c "cd /go/src/github.com/ibm-messaging/mq-container/ && ./incubating/mq-advanced-server-rhel/go-build.sh"