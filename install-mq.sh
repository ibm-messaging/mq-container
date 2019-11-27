#!/bin/bash
# -*- mode: sh -*-
# © Copyright IBM Corporation 2015, 2019
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

mqm_uid=${1:-888}

test -f /usr/bin/yum && YUM=true || YUM=false
test -f /usr/bin/microdnf && MICRODNF=true || MICRODNF=false
test -f /usr/bin/rpm && RPM=true || RPM=false
test -f /usr/bin/apt-get && UBUNTU=true || UBUNTU=false

# Download and extract the MQ installation files
DIR_EXTRACT=/tmp/mq
mkdir -p ${DIR_EXTRACT}
cd ${DIR_EXTRACT}
curl -LO $MQ_URL
tar -zxf ./*.tar.gz

# Recommended: Create the mqm user ID with a fixed UID and group, so that the file permissions work between different images
groupadd --system --gid ${mqm_uid} mqm
useradd --system --uid ${mqm_uid} --gid mqm --groups 0 mqm

# Find directory containing .deb files
$UBUNTU && DIR_DEB=$(find ${DIR_EXTRACT} -name "*.deb" -printf "%h\n" | sort -u | head -1)
$RPM && DIR_RPM=$(find ${DIR_EXTRACT} -name "*.rpm" -printf "%h\n" | sort -u | head -1)
# Find location of mqlicense.sh
MQLICENSE=$(find ${DIR_EXTRACT} -name "mqlicense.sh")

# Accept the MQ license
${MQLICENSE} -text_only -accept
$UBUNTU && echo "deb [trusted=yes] file:${DIR_DEB} ./" > /etc/apt/sources.list.d/IBM_MQ.list

# Install MQ using the DEB packages
$UBUNTU && apt-get update
$UBUNTU && apt-get install -y $MQ_PACKAGES

$RPM && cd $DIR_RPM && rpm -ivh $MQ_PACKAGES

# Remove 32-bit libraries from 64-bit container
# The "file" utility isn't installed by default in UBI, so only try this if it's installed
which file && find /opt/mqm /var/mqm -type f -exec file {} \; | awk -F: '/ELF 32-bit/{print $1}' | xargs --no-run-if-empty rm -f

# Remove tar.gz files unpacked by RPM postinst scripts
find /opt/mqm -name '*.tar.gz' -delete

# Recommended: Set the default MQ installation (makes the MQ commands available on the PATH)
/opt/mqm/bin/setmqinst -p /opt/mqm -i

# Clean up all the downloaded files
$UBUNTU && rm -f /etc/apt/sources.list.d/IBM_MQ.list
rm -rf ${DIR_EXTRACT}

# Optional: Update the command prompt with the MQ version
$UBUNTU && echo "mq:$(dspmqver -b -f 2)" > /etc/debian_chroot

# Remove the directory structure under /var/mqm which was created by the installer
rm -rf /var/mqm

# Create the mount point for volumes, ensuring MQ has permissions to all directories
install --directory --mode 0775 --owner mqm --group root /mnt
install --directory --mode 0775 --owner mqm --group root /mnt/mqm
install --directory --mode 0775 --owner mqm --group root /mnt/mqm/data
install --directory --mode 0775 --owner mqm --group root /mnt/mqm-log
install --directory --mode 0775 --owner mqm --group root /mnt/mqm-log/log
install --directory --mode 0775 --owner mqm --group root /mnt/mqm-data
install --directory --mode 0775 --owner mqm --group root /mnt/mqm-data/qmgrs

# Create the directory for MQ configuration files
install --directory --mode 0775 --owner mqm --group root /etc/mqm

# Create the directory for MQ runtime files
install --directory --mode 0775 --owner mqm --group root /run/mqm

# Create a symlink for /var/mqm -> /mnt/mqm/data
ln -s /mnt/mqm/data /var/mqm

# Optional: Ensure any passwords expire in a timely manner
sed -i 's/PASS_MAX_DAYS\t99999/PASS_MAX_DAYS\t90/' /etc/login.defs
sed -i 's/PASS_MIN_DAYS\t0/PASS_MIN_DAYS\t1/' /etc/login.defs
sed -i 's/PASS_MIN_LEN\t5/PASS_MIN_LEN\t8/' /etc/login.defs
sed -i 's/# minlen = 9/minlen = 8/' /etc/security/pwquality.conf

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

