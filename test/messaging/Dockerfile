# © Copyright IBM Corporation 2018, 2026
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

ARG BUILD_DIR=/var/workspace

###############################################################################
# JRE container, used by both the builder and runtime containers
###############################################################################
FROM registry.access.redhat.com/ubi10/ubi as jre
RUN dnf install \
    --assumeyes \
    --releasever 10 \
    --disableplugin subscription-manager \
    --disablerepo=* \
    --enablerepo=ubi-* \
    --setopt install_weak_deps=false \
    --nodocs \
    java-25-openjdk-headless

###############################################################################
# Builder container
###############################################################################
FROM jre AS builder
ARG BUILD_DIR
WORKDIR ${BUILD_DIR}
RUN dnf install \
    --assumeyes \
    --releasever 10 \
    --disableplugin subscription-manager \
    --disablerepo=* \
    --enablerepo=ubi-* \
    --setopt install_weak_deps=false \
    --nodocs \
    java-25-openjdk-devel
COPY downloads/com.ibm.mq.allclient.jar .
COPY downloads/jms.jar .
COPY downloads/org.json.jar .
# Copy source
COPY test/messaging/src ./src
# Create output directory for compiled classes
RUN mkdir -p ${BUILD_DIR}/classes
# Compile Java files
RUN javac -cp "${BUILD_DIR}/com.ibm.mq.allclient.jar:${BUILD_DIR}/jms.jar:${BUILD_DIR}/org.json.jar" \
    -d ${BUILD_DIR}/classes \
    $(find src/main/java -name "*.java")
# Create JAR file
RUN jar cf ${BUILD_DIR}/test-messaging.jar -C ${BUILD_DIR}/classes .

###############################################################################
# Application runtime
###############################################################################
FROM jre
ARG BUILD_DIR
COPY --chown=1001:root --from=builder ${BUILD_DIR}/test-messaging.jar /opt/app/
COPY --chown=1001:root --from=builder ${BUILD_DIR}/com.ibm.mq.allclient.jar /opt/app/
COPY --chown=1001:root --from=builder ${BUILD_DIR}/jms.jar /opt/app/
COPY --chown=1001:root --from=builder ${BUILD_DIR}/org.json.jar /opt/app/
USER 1001
ENTRYPOINT ["java", "-classpath", "/opt/app/*", "com.ibm.mqcontainer.test.Main"]
