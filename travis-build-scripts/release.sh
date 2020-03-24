#!/bin/bash

# © Copyright IBM Corporation 2019, 2020
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
MANIFEST_FILE=manifest-9.1.5.yaml

# set type of release
if [ ! -z $1 ]; then
    case "$1" in
        staging) TYPE=$1
            ;;
        production) TYPE=$1
            ;;
        *) echo "ERROR: Release type ( staging | production ) must passed to release.sh"
        exit 1
        ;;
    esac
else 
    echo "ERROR: Release type ( staging | production ) must passed to release.sh"
    exit 1
fi

## Pull all images from default repository
## BUILD PRODUCTION ONLY UNTIL DEV AUTH CONFIG COMPLETE
# ARCH=amd64 make pull-devserver
# ARCH=ppc64le make pull-devserver
# ARCH=s390x make pull-devserver

ARCH=amd64 make pull-advancedserver
# ARCH=ppc64le make pull-advancedserver
# ARCH=s390x make pull-advancedserver


function set_staging_registry {
    export MQ_DELIVERY_REGISTRY_HOSTNAME=$MQ_STAGING_REGISTRY
    export MQ_DELIVERY_REGISTRY_NAMESPACE=""
    export MQ_DELIVERY_REGISTRY_USER=$MQ_STAGING_REGISTRY_USER
    export MQ_DELIVERY_REGISTRY_CREDENTIAL=$MQ_STAGING_REGISTRY_CREDENTIAL
}

function set_docker_hub {
    export MQ_DELIVERY_REGISTRY_HOSTNAME=ibmcom
    export MQ_DELIVERY_REGISTRY_NAMESPACE=""
    export MQ_DELIVERY_REGISTRY_USER=$MQ_DOCKERHUB_REGISTRY_USER
    export MQ_DELIVERY_REGISTRY_CREDENTIAL=$MQ_DOCKERHUB_REGISTRY_CREDENTIAL
}

function set_production_registry {
    export MQ_DELIVERY_REGISTRY_HOSTNAME=$MQ_PRODUCTION_REGISTRY
    export MQ_DELIVERY_REGISTRY_NAMESPACE=""
    export MQ_DELIVERY_REGISTRY_USER=$MQ_PRODUCTION_REGISTRY_USER
    export MQ_DELIVERY_REGISTRY_CREDENTIAL=$MQ_PRODUCTION_REGISTRY_CREDENTIAL
}

if [ "$TYPE" = "staging" ]; then 

    set_staging_registry

    # push production images to staging registy
    ./travis-build-scripts/push.sh production amd64
    # ./travis-build-scripts/push.sh production ppc64le
    # ./travis-build-scripts/push.sh production s390x

elif [ "$TYPE" = "production" ]; then

    # pull production images from staging
    set_staging_registry

    ARCH=amd64 make pull-advancedserver
    # ARCH=ppc64le make pull-advancedserver
    # ARCH=s390x make pull-advancedserver

    # release developer image with fat manifest
    set_docker_hub

    ## BUILD PRODUCTION ONLY UNTIL DEV AUTH CONFIG COMPLETE 
    # ARCH=amd64 make push-devserver-dockerhub
    # ARCH=ppc64le make push-devserver-dockerhub
    # ARCH=s390x make push-devserver-dockerhub

    # curl -LO https://github.com/estesp/manifest-tool/releases/download/v0.9.0/manifest-tool-linux-amd64
    # chmod a+x manifest-tool-linux-amd64

    # docker login --username $MQ_DOCKERHUB_REGISTRY_USER --password $MQ_DOCKERHUB_REGISTRY_CREDENTIAL
    # ./manifest-tool-linux-amd64 push from-spec manifests/dockerhub/$MANIFEST_FILE
    # ./manifest-tool-linux-amd64 push from-spec manifests/dockerhub/manifest-latest.yaml

    # release production image
    set_production_registry

    ./travis-build-scripts/push.sh production amd64
    # ./travis-build-scripts/push.sh production ppc64le
    # ./travis-build-scripts/push.sh production s390x
fi
