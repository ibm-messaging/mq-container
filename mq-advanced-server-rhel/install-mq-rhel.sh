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

# Install one or more MQ components into a buildah container

set -ex

function usage {
  echo "Usage: $0 MQContainer MountLocation ARCHIVENAME PACKAGES"
  exit 20
}

if [ "$#" -ne 4 ]; then
  echo "ERROR: Invalid number of parameters"
  usage
fi

readonly ctr_mq=$1
readonly mnt_mq=$2
readonly archive=$3
readonly mq_packages=$4
readonly dir_extract=/tmp/extract
readonly mqm_uid=888
readonly mqm_gid=888

if [ ! -d ${dir_extract}/MQServer ]; then
  mkdir -p ${dir_extract}
  echo Extracting $archive
  tar -zxf $archive -C ${dir_extract}
  echo Extracting finished
fi

# Accept the MQ license
buildah run --volume ${dir_extract}:/mnt/mq-download $ctr_mq -- /mnt/mq-download/MQServer/mqlicense.sh -text_only -accept

buildah run --volume ${dir_extract}:/mnt/mq-download $ctr_mq -- bash -c "cd /mnt/mq-download/MQServer && rpm -ivh $mq_packages"

rm -rf ${dir_extract}/MQServer

# Remove 32-bit libraries from 64-bit container
find $mnt_mq/opt/mqm $mnt_mq/var/mqm -type f -exec file {} \; | awk -F: '/ELF 32-bit/{print $1}' | xargs --no-run-if-empty rm -f

# Remove tar.gz files unpacked by RPM postinst scripts
find $mnt_mq/opt/mqm -name '*.tar.gz' -delete

# Recommended: Set the default MQ installation (makes the MQ commands available on the PATH)
buildah run $ctr_mq -- /opt/mqm/bin/setmqinst -p /opt/mqm -i

mkdir -p $mnt_mq/run/runmqserver
chown ${mqm_uid}:${mqm_gid} $mnt_mq/run/runmqserver

# Remove the directory structure under /var/mqm which was created by the installer
rm -rf $mnt_mq/var/mqm

# Create the mount point for volumes, ensuring MQ has permissions to all directories
mkdir -p $mnt_mq/mnt/mqm
install --directory --mode 0775 --owner ${mqm_uid} --group root $mnt_mq/mnt
install --directory --mode 0775 --owner ${mqm_uid} --group root $mnt_mq/mnt/mqm
install --directory --mode 0775 --owner ${mqm_uid} --group root $mnt_mq/mnt/mqm/data

# Create the directory for MQ configuration files
mkdir -p /etc/mqm
install --directory --mode 0775 --owner ${mqm_uid} --group root $mnt_mq/etc/mqm

# Create a symlink for /var/mqm -> /mnt/mqm/data
buildah run $ctr_mq -- ln -s /mnt/mqm/data /var/mqm

# Optional: Set these values for the IBM Cloud Vulnerability Report
sed -i 's/PASS_MAX_DAYS\t99999/PASS_MAX_DAYS\t90/' $mnt_mq/etc/login.defs
sed -i 's/PASS_MIN_DAYS\t0/PASS_MIN_DAYS\t1/' $mnt_mq/etc/login.defs
sed -i 's/password\t\[success=1 default=ignore\]\tpam_unix\.so obscure sha512/password\t[success=1 default=ignore]\tpam_unix.so obscure sha512 minlen=8/' $mnt_mq/etc/pam.d/password-auth

buildah run $ctr_mq -- cp -rs /opt/mqm/licenses/ /
