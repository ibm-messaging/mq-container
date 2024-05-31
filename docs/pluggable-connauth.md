
### Queue Manager Connection Authentication using secrets

Prior to IBM MQ v9.4.0.0, passwords could be supplied through MQ_ADMIN_PASSWORD and MQ_APP_PASSWORD environment variables. From IBM MQ v9.4.0.0, supplying passwords through environment variables is deprecated and not recommended. IBM MQ v9.4.0.0 provides a new authentication mode to allow developers using mq-container developer image to authenticate users. In this new authentication mode passwords for `app` and `admin` users are supplied through secrets securely mounted into the file system. Secret with name `mqAppPassword` and `mqAdminPassword` can now be used to supply password for users `app` and `admin`. This is in addition to the existing methods of users authentication, described in [User authentication and authorization for IBM MQ in containers] (https://www.ibm.com/docs/en/ibm-mq/latest?topic=containers-user-authentication-authorization-mq-in)

**Please note:**
1. This new feature is enabled only when environment variable `MQ_CONNAUTH_USE_HTP=true` is set while starting the MQ Container.
2. When enabled, the `AuthType` value of the ConnectionAuthentication (`CONNAUTH`) is ignored and secrets are used. However,
the MQ authority records created using (`setmqaut` or `AUTHREC`) will be in effect while using the secrets.
3. Channel Authentication records (`CHLAUTH`) will be in effect while using the secrets.
4. This is developer only feature and not recommended for use in Production.

### Using Secrets

1. `mqAppPassword` and `mqAdminPassword` secrets passed to the container are mounted under /run/secrets directory. These secrets are used for authentication of `app` or `admin` users. It must be noted that `app` and `admin` user do not have any default password.
2. The `app` user is authorized to access `DEV.*` objects of the queue manager.

#### Next Steps:

Use an administrative tool or your application to connect to queue manager using the passwords that are set as secrets for user `app` and `admin`.

**Please note**: When an authentication request is made with a userid other than `app` or `admin`, then the authentication process is delegated to queue manager to handle. This will then use `IDPWOS` or `LDAP` modes for further processing.

#### Troubleshooting

A log file named `mqsimpleauth.log` is generated under `/var/mqm/errors` directory path of the container.  This file will contain all the failed connection authentication requests.  Additional information is logged to this file if the environment variable `DEBUG` is set to `true`.

**Please note**: This log file will be wiped when the queue manager is next started.
