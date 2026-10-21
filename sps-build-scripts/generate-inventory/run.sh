#!/bin/bash

# © Copyright IBM Corporation 2026
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

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Generate inventory only on the mainline branch or ifix branches.
# Feature branches and PR builds must not push inventory entries to the
# shared mq-con-sps-promotion repository.
if [[ "${BRANCH}" != "${MAIN_BRANCH}" && "${BRANCH}" != ifix* ]]; then
    echo "Skipping generate-inventory: BRANCH=${BRANCH} is not the mainline branch (${MAIN_BRANCH}) and is not an ifix branch."
    exit 0
fi

# Generate inventory only when BASE_MQ_LOCKED=false in pipeline.env (stream not locked)
if [[ "${BASE_MQ_LOCKED}" != "false" ]]; then
    echo "Branch marked as locked in pipeline.env (BASE_MQ_LOCKED=${BASE_MQ_LOCKED}). Skipping generate-inventory."
    exit 0
fi

# Strip any trailing '/<prefix>' appended by Makefile to ensure a clean base namespace
DELIVERY_NAMESPACE="${IMAGE_REGISTRY_DELIVERY_NAMESPACE%/*}"
DELIVERY_REPO_PREFIX="${IMAGE_REGISTRY_DELIVERY_REPO_PREFIX:-ci}"

# CI_INVENTORY_BRANCH, INVENTORY_VERSION, INV_MGMT_DIR are all exported by
# inventory-set-env.sh (sourced in the pipeline before this script runs).
sps-promotion-util generate-inventory \
    --mapping              "${SCRIPT_DIR}/image-inventory-mapping.yaml" \
    --host-cred-mapping    "${INV_MGMT_DIR}/host-credential-mapping.yaml" \
    --inventory-dir        "${SPS_BUILD_DIR}/.build-inventory" \
    --inventory-version    "${INVENTORY_VERSION}" \
    --production-tag       "${PRODUCTION_TAG}" \
    --inventory-branch     "${CI_INVENTORY_BRANCH}" \
    --delivery-namespace   "${DELIVERY_NAMESPACE}" \
    --delivery-repo-prefix "${DELIVERY_REPO_PREFIX}"
