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

# Builds and tests the golang programs used by the MQ image.

set -e

cd $GOPATH/src/github.com/ibm-messaging/mq-container/

# Build and test the Go code
mkdir -p build
cd build

rm -f chkmqready chkmqhealthy runmqserver runmqdevserver

if [ "$MQDEV" = "TRUE" ]; then 
    # Build and test the Go code
    go build -ldflags "-X \"main.ImageCreated=$(date --iso-8601=seconds)\" -X \"main.ImageRevision=$IMAGE_REVISION\" -X \"main.ImageSource=$IMAGE_SOURCE\"" --tags 'mqdev' ../cmd/runmqserver/
    go build ../cmd/runmqdevserver/
else 
    go build -ldflags "-X \"main.ImageCreated=$(date --iso-8601=seconds)\" -X \"main.ImageRevision=$IMAGE_REVISION\" -X \"main.ImageSource=$IMAGE_SOURCE\"" ../cmd/runmqserver/
fi

go build ../cmd/chkmqready/
go build ../cmd/chkmqhealthy/
go test -v ../cmd/runmqserver/
go test -v ../cmd/chkmqready/
go test -v ../cmd/chkmqhealthy/
if [ "$MQDEV" = "TRUE" ]; then 
    go test -v ../cmd/runmqdevserver
fi
go test -v ../internal/...
go vet ../cmd/... ../internal/...
