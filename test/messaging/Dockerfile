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
# Builder container, with Maven and JDK
###############################################################################
FROM jre AS builder
RUN dnf install \
    --assumeyes \
    --releasever 10 \
    --disableplugin subscription-manager \
    --disablerepo=* \
    --enablerepo=ubi-* \
    --setopt install_weak_deps=false \
    --nodocs \
    maven-openjdk25
WORKDIR /var/workspace
COPY pom.xml ./
# Download dependencies separately, so they are cached in their own layer
RUN mvn dependency:go-offline install
# Copy source
COPY src ./src
# Run the main build
RUN mvn --offline install

###############################################################################
# Application runtime (JRE only, no build environment)
###############################################################################
FROM jre
COPY --chown=1001:root --from=builder /var/workspace/target/*.jar /opt/app/
COPY --chown=1001:root --from=builder /var/workspace/target/lib/*.jar /opt/app/
USER 1001
ENTRYPOINT ["java", "-classpath", "/opt/app/*", "org.junit.platform.console.ConsoleLauncher", "execute", "--fail-if-no-tests", "-p", "com.ibm.mqcontainer.test", "--details", "verbose"]
