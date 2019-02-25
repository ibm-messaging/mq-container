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

FROM ubuntu:16.04

# The URL to download the MQ installer from in tar.gz format
ARG MQ_URL=https://public.dhe.ibm.com/ibmdl/export/pub/software/websphere/messaging/mqadv/mqadv_dev911_ubuntu_x86-64.tar.gz

# The MQ packages to install
ARG MQ_PACKAGES="ibmmq-explorer"

ARG MQM_UID=999

RUN export DEBIAN_FRONTEND=noninteractive \
  && apt-get update \
  && apt-get install -y \
    libgtk2.0-0 \
    libxtst6

ADD install-mq.sh /usr/local/bin/
RUN chmod u+x /usr/local/bin/install-mq.sh $MQM_UID \
  && install-mq.sh

ENV LANG=en_US.UTF-8

# Run as mqm (999)
USER 999

ENTRYPOINT ["MQExplorer"]