#!/bin/bash
# -*- mode: sh -*-
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

KEY=server.key
CERT=server.crt
PKCS=server.p12
PASSWORD=passw0rd

# Clean up old ones
rm $KEY
rm $CERT
rm $PKCS
rm client-trust.jks
rm singlequotecert/*
rm testcert1/*
rm testcert2/*
rm testcertca1/*
rm clientcert/*
rm clientcert/certonly/*

# Create a private key and certificate in PEM format, for the server to use
openssl req \
       -newkey rsa:2048 -nodes -keyout ${KEY} \
       -subj "/CN=localhost" \
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

cp $KEY testcert1/
cp $CERT testcert1/

# make new TLS test certificates - testcert2
openssl req \
       -newkey rsa:4098 -nodes -keyout testcert2/${KEY} \
       -subj "/CN=localhost" \
       -x509 -days 3650 -out testcert2/${CERT}

# make new TLS test certificates - singlequotecert
openssl req \
       -newkey rsa:4098 -nodes -keyout singlequotecert/${KEY} \
       -subj "/CN=localhost,O=Xa'ou" \
       -x509 -days 3650 -out singlequotecert/${CERT}

# make new TLS test certificates - testcertca1
openssl genrsa -out rootCA.key 4096
openssl req \
       -x509 -new -nodes -key rootCA.key -sha256 -days 3650 \
       -subj "/CN=ROOTCA,O=MQ Test" \
       -out rootCA.crt 

cp rootCA.crt testcertca1/ca.crt

openssl genrsa -out testcertca1/${KEY} 2048
openssl req -new -sha256 -key testcertca1/${KEY} \
       -subj "/CN=localhost" \
       -out cert.csr

openssl x509 -req -in cert.csr -CA rootCA.crt -CAkey rootCA.key \
       -CAcreateserial -out testcertca1/${CERT} -days 3650 -sha256

# clean up TLS Root ca stuff
rm rootCA.crt
rm rootCA.key
rm cert.csr
rm rootCA.srl

# Create certificates for a client test
openssl req \
       -newkey rsa:4098 -nodes -keyout clientcert/${KEY} \
       -subj "/CN=clientcert" \
       -x509 -days 3650 -out clientcert/${CERT}

cp clientcert/${CERT} clientcert/certonly/${CERT}
