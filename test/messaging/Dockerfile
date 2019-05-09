# © Copyright IBM Corporation 2018, 2019
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
FROM docker.io/maven:3-ibmjava as builder
COPY pom.xml /usr/src/mymaven/
WORKDIR /usr/src/mymaven
# Download dependencies separately, so Docker caches them
RUN mvn dependency:go-offline install
# Copy source
COPY src /usr/src/mymaven/src
# Run the main build
RUN mvn --offline install
# Print a list of all the files (useful for debugging)
RUN find /usr/src/mymaven

###############################################################################
# Application runtime (JRE only, no build environment)
###############################################################################
FROM docker.io/ibmjava:8-jre
COPY --from=builder /usr/src/mymaven/target/*.jar /opt/app/
COPY --from=builder /usr/src/mymaven/target/lib/*.jar /opt/app/
ENTRYPOINT ["java", "-classpath", "/opt/app/*", "org.junit.platform.console.ConsoleLauncher", "-p", "com.ibm.mqcontainer.test", "--details", "verbose"]
