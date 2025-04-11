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

GO_VERSION="1.23.6"
sudo rm -rf /usr/local/go
DOWNLOAD_URL="https://go.dev/dl/go${GO_VERSION}.linux-${ARCH}.tar.gz"
curl -fLo go.tar.gz "${DOWNLOAD_URL}"
sudo tar -C /usr/local -xzf go.tar.gz
export PATH=/usr/local/go/bin:$PATH
go version
