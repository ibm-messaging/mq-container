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
readonly imagename=$1

# Initialize yum for use with the scratch container
rpm --root $mnt_mq --initdb

yumdownloader --destdir=/tmp redhat-release-server
rpm --root $mnt_mq -ihv /tmp/redhat-release-server*.rpm || true

yum --installroot $mnt_mq install -y \
    java-1.7.0-openjdk-devel \
    java \
    which \
    wget

buildah run $ctr_mq -- sh -c "cd /tmp && wget http://mirror.olnevhost.net/pub/apache/maven/binaries/apache-maven-3.2.2-bin.tar.gz"
tar xvf $mnt_mq/tmp/apache-maven-3.2.2-bin.tar.gz -C $mnt_mq/tmp/

mkdir -p $mnt_mq/usr/src/mymaven
cp pom.xml $mnt_mq/usr/src/mymaven/
cp -R src $mnt_mq/usr/src/mymaven/src

buildah run $ctr_mq -- sh -c "cd /usr/src/mymaven && export M2_HOME=/tmp/apache-maven-3.2.2 && export M2=\$M2_HOME/bin && export PATH=\$M2:\$PATH && mvn --version && mvn dependency:go-offline install && mvn --offline install"

mkdir -p $mnt_mq/opt/app

cp $mnt_mq/usr/src/mymaven/target/*.jar $mnt_mq/opt/app/
cp $mnt_mq/usr/src/mymaven/target/lib/*.jar $mnt_mq/opt/app/

###############################################################################
# Post install tidy up
###############################################################################

rm -rf $mnt_mq/tmp/*
rm -rf $mnt_mq/usr/src/mymaven

# We can't uninstall tar or gzip because 
yum --installroot $mnt_mq remove -y \
    wget 

###############################################################################
# Contain image finalization
###############################################################################

buildah config \
  --os linux \
  --label architecture=x86_64 \
  --label name="${imagename%:*}" \
  --entrypoint '["java", "-classpath", "/opt/app/*", "org.junit.platform.console.ConsoleLauncher", "-p", "com.ibm.mqcontainer.test", "--details", "verbose"]' \
  $ctr_mq
buildah unmount $ctr_mq
buildah commit $ctr_mq $imagename

buildah rm $ctr_mq
