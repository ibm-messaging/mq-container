#!/bin/bash
# -*- mode: sh -*-
# © Copyright IBM Corporation 2020
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
sudo apt-key adv --keyserver keyserver.ubuntu.com --recv-keys 7EA0A9C3F273FCD8
sudo add-apt-repository "deb [arch=$ARCH] https://download.docker.com/linux/ubuntu $(lsb_release -cs) stable"
sudo apt update
sudo apt -y install docker-ce pass

echo "default-cache-ttl 1200" > /home/travis/.gnupg/gpg-agent.conf
gpg-connect-agent reloadagent /bye

mkdir -p $GOPATH/src/github.com/docker
cd $GOPATH/src/github.com/docker
git clone https://github.com/docker/docker-credential-helpers
cd docker-credential-helpers
make pass 
cp bin/docker-credential-pass $GOPATH/bin/docker-credential-pass
mkdir -p /home/travis/.docker
echo '{ "credsStore": "pass" }' | tee /home/travis/.docker/config.json
gpg2 --batch --gen-key <<-EOF
%echo generating a standard key
Key-Type: DSA
Key-Length: 1024
Subkey-Type: ELG-E
Subkey-Length: 1024
Name-Real: Travis CI
Name-Email: travis@osism.io
Expire-Date: 0
Passphrase: $REGISTRY_PASS
%commit
%echo done
EOF
key=$(gpg2 --list-secret-keys | grep uid -B 1 | head -n 1 | sed 's/^ *//g')
pass init $key
pass insert docker-credential-helpers/docker-pass-initialized-check <<-EOF
pass is initialized
pass is initialized
EOF
gpg2 --passphrase $REGISTRY_PASS --pinentry-mode=loopback --output doc --decrypt ~/.password-store/docker-credential-helpers/docker-pass-initialized-check.gpg
