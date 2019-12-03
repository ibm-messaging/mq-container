#!/bin/bash

# © Copyright IBM Corporation 2019
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

# staging or production
TYPE=""
MANIFEST_FILE=manifest-9.1.4.yaml

# set type of release
if [ ! -z $1 ]; then
    case "$1" in
        staging) TYPE=$1
            ;;
        production) TYPE=$1
            ;;
        *) echo "echo Release type ( staging | production ) must passed to release.sh"
        exit 1
        ;;
    esac
else 
    echo "Release type ( staging | production ) must passed to release.sh"
    exit 1
fi

if [ TYPE = "staging" ]; then 
    # push developer image to pre-release registry
    ./build-scripts/push-dev.sh amd64
    ./build-scripts/push-dev.sh ppc64le
    ./build-scripts/push-dev.sh s390x

    # staging registry
    export MQ_DELIVERY_REGISTRY_HOSTNAME=$MQ_STAGING_REGISTRY
    export MQ_DELIVERY_REGISTRY_USER=$MQ_STAGING_REGISTRY_USER
    export MQ_DELIVERY_REGISTRY_CREDENTIAL=$MQ_STAGING_REGISTRY_CREDENTIAL

    # push production image to staging registy
    ./build-scripts/push-prod.sh amd64
    ./build-scripts/push-prod.sh ppc64le
    ./build-scripts/push-prod.sh s390x

elif [ TYPE = "production" ]; then
    # pull developer image from pre-release registry
    make pull-devserver
    # pull production image from staging
    export MQ_DELIVERY_REGISTRY_HOSTNAME=$MQ_STAGING_REGISTRY
    export MQ_DELIVERY_REGISTRY_USER=$MQ_STAGING_REGISTRY_USER
    export MQ_DELIVERY_REGISTRY_CREDENTIAL=$MQ_STAGING_REGISTRY_CREDENTIAL
    make pull-advancedserver

    # release developer images with fat manifests
    # dockerhub
    export MQ_DELIVERY_REGISTRY_HOSTNAME=ibmcom
    export MQ_DELIVERY_REGISTRY_NAMESPACE=mq
    export MQ_DELIVERY_REGISTRY_USER=$MQ_DOCKERHUB_REGISTRY_USER
    export MQ_DELIVERY_REGISTRY_CREDENTIAL=$MQ_DOCKERHUB_REGISTRY_CREDENTIAL
    # UNCOMMENT WHEN FINISHED TESTING
    # ./build-scripts/push-dev.sh amd64
    # ./build-scripts/push-dev.sh ppc64le
    # ./build-scripts/push-dev.sh s390x

    docker login --username $MQ_DOCKERHUB_REGISTRY_USER --password $MQ_DOCKERHUB_REGISTRY_CREDENTIAL
    # ./manifest-tool-linux-amd64 push from-spec manifests/dockerhub/$MANIFEST_FILE
    # ./manifest-tool-linux-amd64 push from-spec manifests/dockerhub/manifest-latest.yaml

    # dockerstore
    export MQ_DELIVERY_REGISTRY_HOSTNAME=ibmcorp
    export MQ_DELIVERY_REGISTRY_NAMESPACE=""
    # ./build-scripts/push-dev.sh amd64
    # ./build-scripts/push-dev.sh ppc64le
    # ./build-scripts/push-dev.sh s390x

    # docker login --username $MQ_DOCKERHUB_REGISTRY_USER --password $MQ_DOCKERHUB_REGISTRY_CREDENTIAL
    # ./manifest-tool-linux-amd64 push from-spec manifests/dockerstore/$MANIFEST_FILE

    # release production image
    export MQ_DELIVERY_REGISTRY_HOSTNAME=$MQ_PRODUCTION_REGISTRY
    export MQ_DELIVERY_REGISTRY_USER=$MQ_PRODUCTION_REGISTRY_USER
    export MQ_DELIVERY_REGISTRY_CREDENTIAL=$MQ_PRODUCTION_REGISTRY_CREDENTIAL
    # ./build-scripts/push-prod.sh amd64
    # ./build-scripts/push-prod.sh ppc64le
    # ./build-scripts/push-prod.sh s390x
fi
