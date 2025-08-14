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

RED="\033[31m"
END="\033[0m"
ERROR=${RED}

while getopts r:n:i: flag; do
    case "${flag}" in
        r) AMD64_DIGEST=${OPTARG} ;;
        n) S390X_DIGEST=${OPTARG} ;;
        i) PPC64LE_DIGEST=${OPTARG} ;;
        *)
            echo "Unknown option: -${flag}" >&2
            ;;
    esac
done


if [[ -z "$AMD64_DIGEST" || -z "$S390X_DIGEST" || -z "$PPC64LE_DIGEST" ]]; then
  printf "%sEnsure all three architecture images are successfully obtained and build should be completed successfully if not restart the affected architecture builds to ensure all images are built successfully before pushing the manifest.%s\n" \
    "$ERROR" "$END"

  exit 1
fi
