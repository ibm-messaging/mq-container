#!/bin/bash

# © Copyright IBM Corporation 2020
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

k=0
dd=$(date)

# Output 200 lines of dummy output to force the logs to flush if they have broken, happens every 8 mins
while true; do
    echo -e "travis_fold:start:build_heartbeat.$k\033[33;1mDumping heartbeat logs to keep build alive - $dd\033[0m"
    for i in {1..200}; do 
        echo "Keepalive $i"
    done
    echo -e "\ntravis_fold:end:build_heartbeat.$k\r"
    sleep 480
    k=$((k+1))
    dd=$(date)
done
