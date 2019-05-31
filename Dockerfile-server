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

###############################################################################
# Build stage to build Go code
###############################################################################
FROM registry.access.redhat.com/devtools/go-toolset-7-rhel7 as builder
# FROM docker.io/centos/go-toolset-7-centos7 as builder
# The URL to download the MQ installer from in tar.gz format
# This assumes an archive containing the MQ RPM install packages
ARG MQ_URL="https://public.dhe.ibm.com/ibmdl/export/pub/software/websphere/messaging/mqadv/mqadv_dev912_linux_x86-64.tar.gz"
ARG IMAGE_REVISION="Not specified"
ARG IMAGE_SOURCE="Not specified"
ARG IMAGE_TAG="Not specified"
ARG MQM_UID=888
USER 0
COPY install-mq.sh /usr/local/bin/
RUN chmod a+x /usr/local/bin/install-mq.sh \
  && sleep 1 \
  && MQ_PACKAGES="MQSeriesRuntime-*.rpm MQSeriesSDK-*.rpm MQSeriesSamples*.rpm" install-mq.sh $MQM_UID
WORKDIR /opt/app-root/src/go/src/github.com/ibm-messaging/mq-container/
COPY cmd/ ./cmd
COPY internal/ ./internal
COPY vendor/ ./vendor
ENV PATH="${PATH}:/opt/rh/go-toolset-7/root/usr/bin" \
    CGO_CFLAGS="-I/opt/mqm/inc/" \
    CGO_LDFLAGS_ALLOW="-Wl,-rpath.*"
RUN go build -ldflags "-X \"main.ImageCreated=$(date --iso-8601=seconds)\" -X \"main.ImageRevision=$IMAGE_REVISION\" -X \"main.ImageSource=$IMAGE_SOURCE\" -X \"main.ImageTag=$IMAGE_TAG\"" ./cmd/runmqserver/
RUN go build ./cmd/chkmqready/
RUN go build ./cmd/chkmqhealthy/
RUN go build ./cmd/runmqdevserver/
RUN go test -v ./cmd/runmqdevserver/...
RUN go test -v ./cmd/runmqserver/
RUN go test -v ./cmd/chkmqready/
RUN go test -v ./cmd/chkmqhealthy/
RUN go test -v ./internal/...
RUN go vet ./cmd/... ./internal/...

###############################################################################
# Main build stage, to build MQ image
###############################################################################
FROM registry.access.redhat.com/ubi7/ubi-minimal AS mq-server
# The MQ packages to install - see install-mq.sh for default value
ARG MQ_URL="https://public.dhe.ibm.com/ibmdl/export/pub/software/websphere/messaging/mqadv/mqadv_dev912_linux_x86-64.tar.gz"
ARG MQ_PACKAGES="MQSeriesRuntime-*.rpm MQSeriesServer-*.rpm MQSeriesJava*.rpm MQSeriesJRE*.rpm MQSeriesGSKit*.rpm MQSeriesMsg*.rpm MQSeriesSamples*.rpm MQSeriesWeb*.rpm MQSeriesAMS-*.rpm"
#ARG MQ_PACKAGES="ibmmq-server ibmmq-java ibmmq-jre ibmmq-gskit ibmmq-msg-.* ibmmq-samples ibmmq-web ibmmq-ams"
ARG MQM_UID=888
LABEL summary="IBM MQ Advanced Server"
LABEL description="Simplify, accelerate and facilitate the reliable exchange of data with a security-rich messaging solution — trusted by the world’s most successful enterprises"
LABEL vendor="IBM"
LABEL distribution-scope="private"
LABEL authoritative-source-url="https://www.ibm.com/software/passportadvantage/"
LABEL url="https://www.ibm.com/products/mq/advanced"
LABEL io.openshift.tags="mq messaging"
LABEL io.k8s.display-name="IBM MQ Advanced Server"
LABEL io.k8s.description="Simplify, accelerate and facilitate the reliable exchange of data with a security-rich messaging solution — trusted by the world’s most successful enterprises"
COPY install-mq.sh /usr/local/bin/
COPY install-mq-server-prereqs.sh /usr/local/bin/
# Install MQ.  To avoid a "text file busy" error here, we sleep before installing.
RUN env && chmod u+x /usr/local/bin/install-*.sh \
  && sleep 1 \
  && install-mq-server-prereqs.sh $MQM_UID \
  && install-mq.sh $MQM_UID
# Create a directory for runtime data from runmqserver
RUN mkdir -p /run/runmqserver \
  && chown mqm:mqm /run/runmqserver
COPY --from=builder /opt/app-root/src/go/src/github.com/ibm-messaging/mq-container/runmqserver /usr/local/bin/
COPY --from=builder /opt/app-root/src/go/src/github.com/ibm-messaging/mq-container/chkmq* /usr/local/bin/
COPY NOTICES.txt /opt/mqm/licenses/notices-container.txt
# Copy web XML files
COPY web /etc/mqm/web
COPY etc/mqm/*.tpl /etc/mqm/
RUN chmod ug+x /usr/local/bin/runmqserver \
  && chown mqm:mqm /usr/local/bin/*mq* \
  && chmod ug+xs /usr/local/bin/chkmq* \
  && chown -R mqm:mqm /etc/mqm/* \
  && install --directory --mode 0775 --owner mqm --group root /run/runmqserver \
  && touch /run/termination-log \
  && chown mqm:root /run/termination-log \
  && chmod 0660 /run/termination-log
# Always use port 1414 for MQ & 9157 for the metrics
EXPOSE 1414 9157 9443
ENV LANG=en_US.UTF-8 AMQ_DIAGNOSTIC_MSG_SEVERITY=1 AMQ_ADDITIONAL_JSON_LOG=1 LOG_FORMAT=basic
USER $MQM_UID
ENTRYPOINT ["runmqserver"]

###############################################################################
# Add default developer config
###############################################################################
FROM mq-server AS mq-dev-server
ARG MQM_UID=888
# Enable MQ developer default configuration
ENV MQ_DEV=true
# Default administrator password
ENV MQ_ADMIN_PASSWORD=passw0rd
LABEL summary="IBM MQ Advanced for Developers Server"
LABEL description="Simplify, accelerate and facilitate the reliable exchange of data with a security-rich messaging solution — trusted by the world’s most successful enterprises"
LABEL vendor="IBM"
LABEL distribution-scope="private"
LABEL authoritative-source-url="https://www.ibm.com/software/passportadvantage/"
LABEL url="https://www.ibm.com/products/mq/advanced"
LABEL io.openshift.tags="mq messaging"
LABEL io.k8s.display-name="IBM MQ Advanced for Developers Server"
LABEL io.k8s.description="Simplify, accelerate and facilitate the reliable exchange of data with a security-rich messaging solution — trusted by the world’s most successful enterprises"
USER 0
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
COPY --from=builder /opt/app-root/src/go/src/github.com/ibm-messaging/mq-container/runmqdevserver /usr/local/bin/
# Copy template files
COPY incubating/mqadvanced-server-dev/*.tpl /etc/mqm/
# Copy web XML files for default developer configuration
COPY incubating/mqadvanced-server-dev/web /etc/mqm/web
RUN chown -R mqm:mqm /etc/mqm/* \
  && chmod +x /usr/local/bin/runmq* \
  && install --directory --mode 0775 --owner mqm --group root /run/runmqdevserver
ENV MQ_ENABLE_EMBEDDED_WEB_SERVER=1
USER $MQM_UID
ENTRYPOINT ["runmqdevserver"]
