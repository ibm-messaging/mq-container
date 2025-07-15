#!/bin/bash

# © Copyright IBM Corporation 2019, 2024
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
GREEN="\033[32m"

END="\033[0m"

RIGHTARROW="\xE2\x96\xB6"
GREENRIGHTARROW=${GREEN}${RIGHTARROW}${END}

echo "MQ_SNAPSHOT_NAME=$MQ_SNAPSHOT_NAME"
echo "APAR_NUMBER=$APAR_NUMBER"

echo "started manifest sync operation with ifix stage branch"
printf ${GREENRIGHTARROW}" Installing pipeline-util\n"

mkdir -p $GOPATH/src/github.ibm.com/mq-cloudpak

git clone git@github.ibm.com:mq-cloudpak/pipeline-util.git "$GOPATH/src/github.ibm.com/mq-cloudpak/pipeline-util"

cd "$GOPATH/src/github.ibm.com/mq-cloudpak/pipeline-util"

make install

echo 'Sync with linked stage branch for ifix ...' && echo -en 'start:sync-latest\\r'
pipeline-util stages sync --stage=dev-ifix --mq-snapshot-name=${MQ_SNAPSHOT_NAME} --apar-number=${APAR_NUMBER} --sync-branch-name=${BRANCH} --sync-repository-name=mq-container --promotion-type=IFIX --sps-token=${MQCON2_IAM_KEY}
echo -en 'end:sync-latest\\r'