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

ARG BASE_IMAGE=mq-sdk:9.1.1.0-x86_64-ubuntu-16.04

FROM $BASE_IMAGE

COPY incubating/mq-golang-sdk/install-golang.sh /usr/local/bin

ENV GO_VERSION=1.10.7

ENV PATH="${PATH}:/usr/lib/go-${GO_VERSION}/bin:/go/bin:/usr/local/go/bin" \
    CGO_CFLAGS="-I/opt/mqm/inc/"                                           \
    CGO_LDFLAGS_ALLOW="-Wl,-rpath.*"                                       \
    GOPATH="/go"

# Install the Go compiler and Git
RUN chmod +x /usr/local/bin/install-golang.sh \
    && sleep 1 \
    && install-golang.sh 

WORKDIR $GOPATH
