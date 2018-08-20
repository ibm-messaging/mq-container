#!/bin/bash
# © Copyright IBM Corporation 2018
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

set -x
set -e

###############################################################################
# Setup MQ JMS Test container
###############################################################################

# Use a "scratch" container, so the resulting image has minimal files
# Resulting image won't have yum, for example
readonly ctr_mq=$(buildah from scratch)
readonly mnt_mq=$(buildah mount $ctr_mq)

# Initialize yum for use with the scratch container
rpm --root $mnt_mq --initdb

yumdownloader --destdir=/tmp redhat-release-server
rpm --root $mnt_mq -ihv /tmp/redhat-release-server*.rpm || true

yum --installroot $mnt_mq install -y \
    java \
    wget \
    tar \
    gzip \
    which

buildah run $ctr_mq -- sh -c "cd /tmp && wget http://mirror.olnevhost.net/pub/apache/maven/binaries/apache-maven-3.2.2-bin.tar.gz"
buildah run $ctr_mq -- sh -c "cd /tmp && tar xvf apache-maven-3.2.2-bin.tar.gz"

mkdir -p $mnt_mq/usr/src/mymaven
cp pom.xml $mnt_mq/usr/src/mymaven/
cp -R src $mnt_mq/usr/src/mymaven/src

buildah run $ctr_mq -- sh -c "cd /usr/src/mymaven/src && export M2_HOME=/tmp/apache-maven-3.2.2 && export M2=\$M2_HOME/bin && export PATH=\$M2:\$PATH && env && mvn --version"