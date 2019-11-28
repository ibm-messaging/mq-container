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

FROM registry.redhat.io/rhel8/llvm-toolset:8.0.1-10  AS mq-sdk
#FROM docker.io/centos/devtoolset-7-toolchain-centos7 AS mq-sdk

# The URL to download the MQ installer from in tar.gz format
# This assumes an archive containing the MQ Debian (.deb) install packages
ARG MQ_URL

# The packages to install in install-mq.sh
ENV MQ_PACKAGES="MQSeriesRuntime-*.rpm MQSeriesSDK-*.rpm MQSeriesSamples*.rpm"

ENV MQM_UID=888

USER 0
COPY install-mq.sh /usr/local/bin/

# Install MQ.  To avoid a "text file busy" error here, we sleep before installing.
# Need to re-instate the `/var/mqm` directory after installation, to avoid MQ 
# errors with some commands (e.g. `dspmqver`)
RUN chmod u+x /usr/local/bin/install-mq.sh \
  && sleep 1 \
  && install-mq.sh $MQM_UID \
  && rm -rf /var/mqm \
  && /opt/mqm/bin/crtmqdir -f -s
USER 1001