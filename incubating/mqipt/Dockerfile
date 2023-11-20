# © Copyright IBM Corporation 2018, 2023
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

FROM registry.access.redhat.com/ubi8/ubi-minimal
ARG MQIPT_ARCHIVE=./IBM-MQIPT-LinuxX64.tar
RUN microdnf --disableplugin=subscription-manager install bash grep procps-ng sed which
ADD $MQIPT_ARCHIVE /opt
COPY startMQIPT.sh /usr/local/bin
ENV MQIPT_PATH=/opt/mqipt
RUN chown -R 1001:0 $MQIPT_PATH \
  && chown -R 1001:0 /usr/local/bin/startMQIPT.sh \
  && chmod -R 550 /usr/local/bin/startMQIPT.sh
VOLUME /var/mqipt
USER 1001
ENTRYPOINT ["startMQIPT.sh"]