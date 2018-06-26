#!/bin/bash
# -*- mode: sh -*-
# © Copyright IBM Corporation 2018
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

# Build a RHEL image, using the buildah tool

set -x
set -e

MQ_ARCHIVE=downloads/mqadv_dev905_linux_x86-64.tar.gz
MQ_PACKAGES="MQSeriesRuntime-*.rpm MQSeriesServer-*.rpm MQSeriesJava*.rpm MQSeriesJRE*.rpm MQSeriesGSKit*.rpm MQSeriesMsg*.rpm MQSeriesSamples*.rpm MQSeriesAMS-*.rpm"

# Use a "scratch" container, so the resulting image has minimal files
# Resulting image won't have yum, for example
ctr=$(buildah from scratch)
scratchmnt=$(buildah mount $ctr)

# Initialize yum for use with the scratch container
rpm --root $scratchmnt --initdb
yum install yum-utils
yumdownloader --destdir=/tmp redhat-release-server
rpm --root $scratchmnt -ihv /tmp/redhat-release-server*.rpm

# Install the packages required by MQ
yum install -y --installroot=$scratchmnt \
  bash \
  bc \
  coreutils \
  file \
  findutils \
  gawk \
  glibc-common \
  grep \
  passwd \
  procps-ng \
  sed \
  tar \
  util-linux

# Clean up cached files
yum clean all --installroot=$scratchmnt
rm -rf $scratchmnt/var/cache/yum/*

groupadd --root $scratchmnt --system --gid 888 mqm
useradd --root $scratchmnt --system --uid 888 --gid mqm mqm
usermod --root $scratchmnt -G root mqm

DIR_EXTRACT=$scratchmnt/tmp/extract
mkdir -p $scratchmnt/tmp/extract
tar -zxvf ${MQ_ARCHIVE} -C ${DIR_EXTRACT}
DIR_RPM=$(find ${DIR_EXTRACT} -name "*.rpm" -printf "%h\n" | sort -u | head -1)
DIR_RPM=${DIR_RPM#$scratchmnt}
#DIR_RPM=$(buildah run $ctr -- find ${DIR_EXTRACT} -name "*.rpm" -printf "%h\n" | sort -u | head -1)
# Find location of mqlicense.sh
#MQLICENSE=$(buildah run $ctr -- find ${DIR_EXTRACT} -name "mqlicense.sh")
MQLICENSE=$(find ${DIR_EXTRACT} -name "mqlicense.sh")
MQLICENSE=${MQLICENSE#$scratchmnt}

# Accept the MQ license
buildah run $ctr -- ${MQLICENSE} -text_only -accept

buildah run $ctr -- bash -c "cd $DIR_RPM && rpm -ivh $MQ_PACKAGES"
rm -rf ${DIR_EXTRACT}

# Remove 32-bit libraries from 64-bit container
find $scratchmnt/opt/mqm $scratchmnt/var/mqm -type f -exec file {} \; | awk -F: '/ELF 32-bit/{print $1}' | xargs --no-run-if-empty rm -f

# Remove tar.gz files unpacked by RPM postinst scripts
find $scratchmnt/opt/mqm -name '*.tar.gz' -delete

# Recommended: Set the default MQ installation (makes the MQ commands available on the PATH)
buildah run $ctr -- /opt/mqm/bin/setmqinst -p /opt/mqm -i

# Remove the directory structure under /var/mqm which was created by the installer
rm -rf $scratchmnt/var/mqm

# Create the mount point for volumes
mkdir -p $scratchmnt/mnt/mqm

# Create the directory for MQ configuration files
mkdir -p $scratchmnt/etc/mqm

# Create a symlink for /var/mqm -> /mnt/mqm/data
buildah run $ctr ln -s /mnt/mqm/data /var/mqm

# Optional: Set these values for the Bluemix Vulnerability Report
sed -i 's/PASS_MAX_DAYS\t99999/PASS_MAX_DAYS\t90/' $scratchmnt/etc/login.defs
sed -i 's/PASS_MIN_DAYS\t0/PASS_MIN_DAYS\t1/' $scratchmnt/etc/login.defs
sed -i 's/password\t\[success=1 default=ignore\]\tpam_unix\.so obscure sha512/password\t[success=1 default=ignore]\tpam_unix.so obscure sha512 minlen=8/' $scratchmnt/etc/pam.d/password-auth

# Build and test the Go code
go build ./cmd/runmqserver/
go build ./cmd/chkmqready/
go build ./cmd/chkmqhealthy/
go test -v ./cmd/runmqserver/
go test -v ./cmd/chkmqready/
go test -v ./cmd/chkmqhealthy/
go test -v ./internal/...
go vet ./cmd/... ./internal/...
# Install the Go binaries into the image
cp runmqserver $scratchmnt/usr/local/bin/
cp chkmq* $scratchmnt/usr/local/bin/
cp NOTICES.txt $scratchmnt/opt/mqm/licenses/notices-container.txt
chmod ug+x $scratchmnt/usr/local/bin/runmqserver
chown mqm:mqm $scratchmnt/usr/local/bin/*mq*
chmod ug+xs $scratchmnt/usr/local/bin/chkmq*

buildah config \
  --port 1414/tcp \
  --port 9157/tcp \
  --os linux \
  --label architecture=x86_64 \
  --label io.openshift.tags="mq messaging" \
  --label io.k8s.display-name="IBM MQ Advanced Server" \
  --label io.k8s.description="IBM MQ is messaging middleware that simplifies and accelerates the integration of diverse applications and business data across multiple platforms.  It uses message queues to facilitate the exchanges of information and offers a single messaging solution for cloud, mobile, Internet of Things (IoT) and on-premises environments." \
  --label name="mqadvanced-server" \
  --label vendor="IBM" \
  --label version="9.0.5.0" \
  --env AMQ_ADDITIONAL_JSON_LOG=1 \
  --env LANG=en_US.UTF-8 \
  --env LOG_FORMAT=basic \
  --entrypoint runmqserver \
  --user 888 \
  $ctr
buildah unmount $ctr
buildah commit $ctr mymq

# TODO: Leaves the working container lying around.  Good for dev.