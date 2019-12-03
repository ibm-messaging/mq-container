<?xml version="1.0" encoding="UTF-8"?>
<server>
    <keyStore id="MQWebKeyStore" location="/run/runmqserver/tls/${env.AMQ_WEBKEYSTORE}" type="PKCS12" password="${env.AMQ_WEBKEYSTOREPW}"/>
    <keyStore id="MQWebTrustStore" location="/run/runmqserver/tls/trust.p12" type="PKCS12" password="${env.AMQ_WEBKEYSTOREPW}"/>
    <ssl id="thisSSLConfig" clientAuthenticationSupported="true" keyStoreRef="MQWebKeyStore" trustStoreRef="${env.AMQ_WEBTRUSTSTOREREF}" sslProtocol="TLSv1.2"/>
    <sslDefault sslRef="thisSSLConfig"/>
</server>
