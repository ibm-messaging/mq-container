#!/bin/bash
# -*- mode: sh -*-
# © Copyright IBM Corporation 2015, 2019
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

# Install Docker and dep, required by build (assumes Ubuntu host, as used by Travis build)

set -ex

sudo curl -Lo /usr/local/bin/dep https://github.com/golang/dep/releases/download/v0.5.1/dep-linux-$ARCH
sudo chmod +x /usr/local/bin/dep

go install golang.org/x/lint/golint@latest
curl -sfL https://raw.githubusercontent.com/securego/gosec/master/install.sh | sh -s -- -b $GOPATH/bin 2.0.0 || echo "Gosec not installed. Platform may not be supported."
