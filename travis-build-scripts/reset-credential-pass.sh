#!/bin/bash

# © Copyright IBM Corporation 2025
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
if [ -f "doc" ] ; then
    rm -f doc
fi
gpg2 --passphrase $REGISTRY_PASS --pinentry-mode=loopback --output doc --decrypt ~/.password-store/docker-credential-helpers/docker-pass-initialized-check.gpg
