# © Copyright IBM Corporation 2018,2020
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

FROM ibmcom/ibmjava:jre

ARG IPTFILE=9.1.5.0-IBM-MQIPT-LinuxX64.tar.gz

COPY $IPTFILE /opt/

RUN rm -rf /var/lib/apt/lists/* \
  && cd /opt/ \
  && tar xvzf ./$IPTFILE \
  && chmod -R a-w /opt/mqipt

ENV MQIPT_PATH=/opt/mqipt

COPY startMQIPT.sh /usr/local/bin

VOLUME /var/mqipt

ENTRYPOINT ["startMQIPT.sh"]