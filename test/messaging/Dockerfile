# © Copyright IBM Corporation 2018, 2022
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
# Application build environment (Maven)
###############################################################################
FROM registry.access.redhat.com/ubi8/openjdk-8 as builder
COPY pom.xml ./
#WORKDIR /usr/src/mymaven
# Download dependencies separately, so Docker caches them
RUN mvn dependency:go-offline install
# Copy source
COPY src ./src
# Run the main build
RUN mvn --offline install
# Print a list of all the files (useful for debugging)
RUN find ./

###############################################################################
# Application runtime (JRE only, no build environment)
###############################################################################
# OpenJDK is not technically supported with the MQ client, but is good enough for these tests
FROM registry.access.redhat.com/ubi8/openjdk-8-runtime
COPY --from=builder /home/jboss/target/*.jar /opt/app/
COPY --from=builder /home/jboss/target/lib/*.jar /opt/app/
USER 1001
ENTRYPOINT ["java", "-classpath", "/opt/app/*", "org.junit.platform.console.ConsoleLauncher", "--fail-if-no-tests", "-p", "com.ibm.mqcontainer.test", "--details", "verbose"]
