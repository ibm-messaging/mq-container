#!/bin/bash
# -*- mode: sh -*-
# © Copyright IBM Corporation 2015, 2020
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

# Fail on any non-zero return code
set -ex

test -f /usr/bin/rpm && RPM=true || RPM=false
test -f /usr/bin/apt-get && UBUNTU=true || UBUNTU=false

# Download and extract the MQ unzippable server
DIR_TMP=/tmp/mq
mkdir -p ${DIR_TMP}
cd ${DIR_TMP}
curl -LO $MQ_URL

INSTALLATION_DIR=/opt/mqm
tar -C ${INSTALLATION_DIR} -xzf ./*.tar.gz
ls -la ${INSTALLATION_DIR}
rm -rf ${DIR_TMP}

# Accept the MQ license
${INSTALLATION_DIR}/bin/mqlicense -accept

# Remove 32-bit libraries from 64-bit container
# The "file" utility isn't installed by default in UBI, so only try this if it's installed
which file && find ${INSTALLATION_DIR} /var/mqm -type f -exec file {} \; | awk -F: '/ELF 32-bit/{print $1}' | xargs --no-run-if-empty rm -f

# Optional: Update the command prompt with the MQ version
$UBUNTU && echo "mq:$(dspmqver -b -f 2)" > /etc/debian_chroot

# Create the mount point for volumes, ensuring MQ has permissions to all directories
install --directory --mode 0775 --owner 1001 --group root /mnt
install --directory --mode 0775 --owner 1001 --group root /mnt/mqm
install --directory --mode 0775 --owner 1001 --group root /mnt/mqm/data
install --directory --mode 0775 --owner 1001 --group root /mnt/mqm-log
install --directory --mode 0775 --owner 1001 --group root /mnt/mqm-log/log
install --directory --mode 0775 --owner 1001 --group root /mnt/mqm-data
install --directory --mode 0775 --owner 1001 --group root /mnt/mqm-data/qmgrs

# Create the directory for MQ configuration files
install --directory --mode 0775 --owner 1001 --group root /etc/mqm

# Create the directory for MQ runtime files
install --directory --mode 2775 --owner 1001 --group root /run/mqm

# Create a symlink for /var/mqm -> /mnt/mqm/data
ln -s /mnt/mqm/data /var/mqm

# Optional: Ensure any passwords expire in a timely manner
sed -i 's/PASS_MAX_DAYS\t99999/PASS_MAX_DAYS\t90/' /etc/login.defs
sed -i 's/PASS_MIN_DAYS\t0/PASS_MIN_DAYS\t1/' /etc/login.defs
sed -i 's/PASS_MIN_LEN\t5/PASS_MIN_LEN\t8/' /etc/login.defs
$RPM && sed -i 's/# minlen/minlen/' /etc/security/pwquality.conf

$UBUNTU && PAM_FILE=/etc/pam.d/common-password
$RPM && PAM_FILE=/etc/pam.d/password-auth
sed -i 's/password\t\[success=1 default=ignore\]\tpam_unix\.so obscure sha512/password\t[success=1 default=ignore]\tpam_unix.so obscure sha512 minlen=8/' $PAM_FILE

# List all the installed packages, for the build log
$RPM && rpm -q --all || true
$UBUNTU && dpkg --list || true

#Update the license file to include UBI 8 instead of UBI 7
sed -i 's/v7.0/v8.0/g' /opt/mqm/licenses/non_ibm_license.txt

# Copy MQ Licenses into the correct location
mkdir -p /licenses
cp /opt/mqm/licenses/*.txt /licenses/
