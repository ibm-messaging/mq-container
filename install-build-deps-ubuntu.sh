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

curl https://glide.sh/get | sh
sudo curl -Lo /usr/local/bin/dep https://github.com/golang/dep/releases/download/v0.5.0/dep-linux-amd64
sudo chmod +x /usr/local/bin/dep

go get -u golang.org/x/lint/golint

if [[ "${COMMAND}" == "podman" ]]; then
    sudo apt-get update -qq
    sudo apt-get install -qq -y software-properties-common uidmap
    sudo add-apt-repository -y ppa:projectatomic/ppa
    sudo apt-get update -qq
    sudo apt-get -qq -y install podman
    CONTAINER_STORAGE=/var/run/containers/storage
    sudo mkdir -p ${CONTAINER_STORAGE}
    sudo chown root:$(id -g) ${CONTAINER_STORAGE}
    sudo chmod 0770 ${CONTAINER_STORAGE}
    mkdir -p $HOME/config/containers
    cat << EOF > $HOME/.config/containers/storage.conf
[storage]
driver = "overlay"
runroot = "${CONTAINER_STORAGE}"
graphroot = "${CONTAINER_STORAGE}"
EOF
fi