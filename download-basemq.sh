#!/bin/bash

# © Copyright IBM Corporation 2024
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

while getopts r:v: flag
do
    case "${flag}" in
        r) MQ_ARCHIVE_DEV=${OPTARG};;
        v) MQ_VERSION_VRM=${OPTARG};;
    esac
done

if [[ -z $MQ_ARCHIVE_DEV || -z $MQ_VERSION_VRM ]] ; then
  printf "${ERROR}MQ driver download script parameters missing!${END}\n"
  exit 1
fi

BASE_MQ_LOCATION="https://public.dhe.ibm.com/ibmdl/export/pub/software/websphere/messaging/mqadv"

FILE_FOUND=$(curl -I $BASE_MQ_LOCATION/$MQ_ARCHIVE_DEV -w "%{http_code}" -s -o /dev/null)

if [ "$FILE_FOUND" -eq 200 ]; then
   curl --fail --location $BASE_MQ_LOCATION/$MQ_ARCHIVE_DEV  --output downloads/"$MQ_ARCHIVE_DEV"

elif [ "$FILE_FOUND" -eq 404 ]; then
   curl -s --list-only --location $BASE_MQ_LOCATION | sed 's/href=/\nhref=/g' |grep href=\" |sed 's/.*href="//g;s/".*//g'  > downloads/base-mq-file-list.txt
   echo "$MQ_ARCHIVE_DEV is not available at $BASE_MQ_LOCATION" && echo "================================================="
   grep "$MQ_VERSION_VRM" downloads/base-mq-file-list.txt| grep "IBM-MQ-" && echo "=================================================" && echo "$MQ_VERSION_VRM images available in the download site are listed above"
   echo "Choose any of the available version and run build command for example,'MQ_VERSION=9.4.0.0 make build-devserver'"
   rm -f downloads/base-mq-file-list.txt
   exit 1
else
    echo "Unexpected error when downloading MQ driver from $BASE_MQ_LOCATION"
    exit 1
fi
