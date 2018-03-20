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

###############################################################################
# Application build environment (Gradle)
###############################################################################
FROM gradle as builder
ENV GRADLE_OPTS="-Dorg.gradle.daemon=false"
# Change where Gradle stores its cache, so that it's not under a volume
# (and therefore gets cached by Docker)
ENV GRADLE_USER_HOME=/home/gradle/gradle
RUN mkdir -p $GRADLE_USER_HOME
COPY --chown=gradle build.gradle /app/
WORKDIR /app
# Download dependencies separately, so Docker caches them
RUN gradle download
# Copy source
COPY --chown=gradle src /app/src
# Run the main build
RUN gradle install

###############################################################################
# Application runtime (JRE only, no build environment)
###############################################################################
FROM ibmjava:sfj
COPY --from=builder /app/lib/*.jar /opt/app/
ENTRYPOINT ["java", "-classpath", "/opt/app/*", "org.junit.platform.console.ConsoleLauncher", "-p", "com.ibm.mqcontainer.test", "--details", "verbose"]
