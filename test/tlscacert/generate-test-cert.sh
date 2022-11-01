#!/bin/bash -ex
# -*- mode: sh -*-
# © Copyright IBM Corporation 2018, 2022
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

KEY=server.key
CERT=server.crt
CACERT=cacert.crt
CAPEM=rootcakey.pem

# Create a private key and certificate in PEM format, for the server to use
openssl req \
       -newkey rsa:2048 -nodes -keyout ${KEY} \
       -subj "/CN=localhost" \
       -addext "subjectAltName = DNS:localhost" \
       -x509 -days 3650 -out ${CERT}

# Generate the private key of the root CA
openssl genrsa -out ${CAPEM} 2048

#Generate the self-signed root CA certificate. Manual input is required when prompted
openssl req -x509 -sha256 -new -nodes -key ${CAPEM} -days 3650 -out ${CACERT}

