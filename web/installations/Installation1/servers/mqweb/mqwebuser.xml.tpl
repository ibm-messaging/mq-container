<?xml version="1.0" encoding="UTF-8"?>
<server>
    <featureManager>
        <feature>openidConnectClient-1.0</feature>
        <feature>ssl-1.0</feature>
    </featureManager>
    <enterpriseApplication id="com.ibm.mq.console">
        <application-bnd>
            <security-role name="MQWebAdmin">
                <group name="MQWebUI" realm="defaultRealm"/>
                {{- range $index, $element := .AdminUser}}
                <user name="admin{{$index}}" access-id="{{.}}"/>
                {{- end}}
            </security-role>
        </application-bnd>
    </enterpriseApplication>
    <enterpriseApplication id="com.ibm.mq.rest">
        <application-bnd>
            <security-role name="MQWebAdmin">
                <group name="MQWebUI" realm="defaultRealm"/>
            </security-role>
            <security-role name="MQWebUser">
                <group name="MQWebMessaging" realm="defaultRealm"/>
            </security-role>
        </application-bnd>
    </enterpriseApplication>
    <openidConnectClient id="mqclient"
        clientId="${env.MQ_OIDC_CLIENT_ID}"
        clientSecret="${env.MQ_OIDC_CLIENT_SECRET}"
        uniqueUserIdentifier="${env.MQ_OIDC_UNIQUE_USER_IDENTIFIER}"
        authorizationEndpointUrl="${env.MQ_OIDC_AUTHORIZATION_ENDPOINT}"
        tokenEndpointUrl="${env.MQ_OIDC_TOKEN_ENDPOINT}"
        scope="openid profile email"
        inboundPropagation="supported"
        jwkEndpointUrl="${env.MQ_OIDC_JWK_ENDPOINT}"
        signatureAlgorithm="RS256"
        issuerIdentifier="${env.MQ_OIDC_ISSUER_IDENTIFIER}">
    </openidConnectClient>
    <variable name="httpHost" value="*"/>
    <variable name="managementMode" value="externallyprovisioned"/>
    <httpDispatcher enableWelcomePage="false" appOrContextRootMissingMessage='&lt;script&gt;document.location.href="/ibmmq/console";&lt;/script&gt;' />
    <include location="tls.xml"/>
</server>
