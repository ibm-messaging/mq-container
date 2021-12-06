#!/bin/bash -ex
# -*- mode: sh -*-
# © Copyright IBM Corporation 2018, 2021
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
PKCS=server.p12
PASSWORD=passw0rd

# Create a private key and certificate in PEM format, for the server to use
openssl req \
       -newkey rsa:2048 -nodes -keyout ${KEY} \
       -subj "/CN=localhost" \
       -addext "subjectAltName = DNS:localhost" \
       -x509 -days 3650 -out ${CERT}

# Add the key and certificate to a PKCS #12 key store, for the server to use
openssl pkcs12 \
       -inkey ${KEY} \
       -in ${CERT} \
       -export -out ${PKCS} \
       -password pass:${PASSWORD}

# Add the certificate to a trust store in JKS format, for Java clients to use when connecting
keytool -import \
	-alias server-cert \
	-file ${CERT} \
	-keystore client-trust.jks \
	-storepass ${PASSWORD} \
	-noprompt
