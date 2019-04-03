#!/bin/bash
# -*- mode: sh -*-
# © Copyright IBM Corporation 2018, 2019
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

# Copy in licenses from installed packages

set -e 

rm -f /licenses/installed_package_notices

for p in $(rpm -qa | sort)
do 
  rpm -qi $p >> /licenses/installed_package_notices
  printf "\n" >> /licenses/installed_package_notices
done

chmod 0444 /licenses/installed_package_notices
