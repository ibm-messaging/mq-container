#!/bin/bash
# -*- mode: sh -*-
# © Copyright IBM Corporation 2015, 2023
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

# Accept the MQ license
/opt/mqm/bin/mqlicense -accept

# Optional: Update the command prompt with the MQ version
$UBUNTU && echo "mq:$(dspmqver -b -f 2)" > /etc/debian_chroot

# Create the mount point for volumes, ensuring MQ has permissions to all directories
install --directory --mode 2775 --owner 1001 --group root /mnt
install --directory --mode 2775 --owner 1001 --group root /mnt/mqm
install --directory --mode 2775 --owner 1001 --group root /mnt/mqm/data
install --directory --mode 2775 --owner 1001 --group root /mnt/mqm-log
install --directory --mode 2775 --owner 1001 --group root /mnt/mqm-log/log
install --directory --mode 2775 --owner 1001 --group root /mnt/mqm-data
install --directory --mode 2775 --owner 1001 --group root /mnt/mqm-data/qmgrs
install --directory --mode 2775 --owner 1001 --group root /run

# Create the directory for MQ configuration files
install --directory --mode 2775 --owner 1001 --group root /etc/mqm

# Create the directory for scratch volume 
install --directory --mode 2775 --owner 1001 --group root /run/scratch

# Create the directory for runmqserver files
install --directory --mode 2775 --owner 1001 --group root /run/scratch/runmqserver

# Create the directory for MQ runtime files
install --directory --mode 2775 --owner 1001 --group root /run/scratch/mqm

# Create a symlink for /var/mqm -> /mnt/mqm/data
ln -s /mnt/mqm/data /var/mqm

# Create a symlink for /run/runmqserver -> /run/scratch/runmqserver
ln -s /run/scratch/runmqserver /run/runmqserver

# Optional: Ensure any passwords expire in a timely manner
sed -i 's/PASS_MAX_DAYS\t99999/PASS_MAX_DAYS\t90/' /etc/login.defs
sed -i 's/PASS_MIN_DAYS\t0/PASS_MIN_DAYS\t1/' /etc/login.defs
sed -i 's/PASS_MIN_LEN\t5/PASS_MIN_LEN\t8/' /etc/login.defs
$RPM && sed -i 's/# minlen/minlen/' /etc/security/pwquality.conf

$UBUNTU && PAM_FILE=/etc/pam.d/common-password
$RPM && PAM_FILE=/etc/pam.d/password-auth
sed -i 's/password\t\[success=1 default=ignore\]\tpam_unix\.so obscure sha512/password\t[success=1 default=ignore]\tpam_unix.so obscure sha512 minlen=8/' $PAM_FILE

# List all the installed packages, for the build log
$RPM && (rpm -q --all | sort) || true
$UBUNTU && (dpkg --list | sort) || true

# Update the license file to include UBI 8 instead of UBI 7
sed -i 's/v7.0/v8.0/g' /opt/mqm/licenses/non_ibm_license.txt

# Copy MQ Licenses into the correct location
mkdir -p /licenses
cp /opt/mqm/licenses/*.txt /licenses/

# Update server.xml to include mqwebexternal.xml
sed -i 's|<include location="mqwebuser.xml"/>|<include location="mqwebexternal.xml"/>\n    <include location="mqwebuser.xml"/>|' /opt/mqm/samp/web/server.xml
