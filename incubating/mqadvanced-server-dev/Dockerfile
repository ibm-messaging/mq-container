# © Copyright IBM Corporation 2015, 2019
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

ARG BASE_IMAGE=mqadvanced-server-dev-base:9.1.1.0-x86_64-ubuntu-16.04
ARG BUILDER_IMAGE=mq-golang-sdk:9.1.1.0-x86_64-ubuntu-16.04

###############################################################################
# Build stage to build Go code
###############################################################################
FROM $BUILDER_IMAGE as builder
ARG IMAGE_REVISION="Not specified"
ARG IMAGE_SOURCE="Not specified"
ARG IMAGE_TAG="Not specified"
WORKDIR /go/src/github.com/ibm-messaging/mq-container/
COPY cmd/ ./cmd
COPY internal/ ./internal
COPY vendor/ ./vendor
# Re-build runmqserver, with code tagged with 'mqdev' enabled
RUN go build -ldflags "-X \"main.ImageCreated=$(date --iso-8601=seconds)\" -X \"main.ImageRevision=$IMAGE_REVISION\" -X \"main.ImageSource=$IMAGE_SOURCE\" -X \"main.ImageTag=$IMAGE_TAG\"" --tags 'mqdev'  ./cmd/runmqserver
RUN go build ./cmd/runmqdevserver/
# Run all unit tests
RUN go test -v ./cmd/runmqdevserver/...

###############################################################################
# Main build stage
###############################################################################
FROM $BASE_IMAGE

# Enable MQ developer default configuration
ENV MQ_DEV=true

# Default administrator password
ENV MQ_ADMIN_PASSWORD=passw0rd

ARG MQM_UID=999

USER root

COPY incubating/mqadvanced-server-dev/install-extra-packages.sh /usr/local/bin/

RUN chmod u+x /usr/local/bin/install-extra-packages.sh \
  && sleep 1 \
  && install-extra-packages.sh

# WARNING: This is what allows the mqm user to change the password of any other user
# It's used by runmqdevserver to change the admin/app passwords.
RUN echo "mqm    ALL = NOPASSWD: /usr/sbin/chpasswd" > /etc/sudoers.d/mq-dev-config

## Add admin and app users, and set a default password for admin
RUN useradd admin -G mqm \
  && groupadd mqclient \
  && useradd app -G mqclient \
  && echo admin:$MQ_ADMIN_PASSWORD | chpasswd

# Create a directory for runtime data from runmqserver
RUN mkdir -p /run/runmqdevserver \
  && chown mqm:mqm /run/runmqdevserver

COPY --from=builder /go/src/github.com/ibm-messaging/mq-container/runmqserver /usr/local/bin/
COPY --from=builder /go/src/github.com/ibm-messaging/mq-container/runmqdevserver /usr/local/bin/

# Copy template files
COPY incubating/mqadvanced-server-dev/*.tpl /etc/mqm/
# Copy web XML files for default developer configuration
COPY incubating/mqadvanced-server-dev/web /etc/mqm/web

RUN chown -R mqm:mqm /etc/mqm/* \
  && chmod +x /usr/local/bin/runmq* \
  && install --directory --mode 0775 --owner mqm --group root /run/runmqdevserver

EXPOSE 9443

USER $MQM_UID

ENTRYPOINT ["runmqdevserver"]
