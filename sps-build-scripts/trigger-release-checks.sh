#!/bin/bash

# © Copyright IBM Corporation 2024, 2025
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

if [[ -n "$PIPELINE_RUN_ID" ]]; then
  if [[ -z "$GOPATH" ]]; then
    echo "GOPATH not set to clone release checks."
    exit 1
  fi

  EVENT_SOURCE="$(get_env APP_REPO_NAME)"

  if [[ -z "$EVENT_SOURCE" ]]; then
    echo "EVENT_SOURCE is not set. Release checks cannot be triggered. Exiting...."
    exit 1
  fi

  echo "Triggering release checks for event source: $EVENT_SOURCE"

  REPO="$GOPATH/src/github.ibm.com/mq-cloudpak/release-checks"
  mkdir -p "$REPO"

  echo "Cloning release-checks repo..."
  git clone git@github.ibm.com:mq-cloudpak/release-checks.git "$REPO" && cd "$REPO"

  go run scripts/sps_tekton.go "$EVENT_SOURCE"
fi