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

ARG BASE_IMAGE=mq-golang-build:9.0.5.0-x86_64-ubuntu-16.04

FROM $BASE_IMAGE

RUN mkdir -p "$GOPATH/src/github.com/ibm-messaging/mq-golang/samples"
WORKDIR $GOPATH/src/github.com/ibm-messaging/mq-golang/samples

COPY ./samples/clientconn clientconn
COPY ./samples/mqitest mqitest

RUN go install ./clientconn  \
  && go install ./mqitest
