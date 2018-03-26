# Usage

In order to use the image, it is necessary to accept the terms of the IBM MQ license.  This is achieved by specifying the environment variable `LICENSE` equal to `accept` when running the image.  You can also view the license terms by setting this variable to `view`. Failure to set the variable will result in the termination of the container with a usage statement.  You can view the license in a different language by also setting the `LANG` environment variable.

## Running with the default configuration
You can run a queue manager with the default configuration and a listener on port 1414 using the following command.  For example, the following command creates and starts a queue manager called `QM1`, and maps port 1414 on the host to the MQ listener on port 1414 inside the container, as well as port 9443 on the host to the web console on port 9443 inside the container:

```
docker run \
  --env LICENSE=accept \
  --env MQ_QMGR_NAME=QM1 \
  --publish 1414:1414 \
  --publish 9443:9443 \
  --detach \
  ibmcom/mq
```

## Running with the default configuration and a volume
The above example will not persist any configuration data or messages across container runs.  In order to do this, you need to use a [volume](https://docs.docker.com/engine/admin/volumes/volumes/).  For example, you can create a volume with the following command:

```
docker volume create qm1data
```

You can then run a queue manager using this volume as follows:

```
docker run \
  --env LICENSE=accept \
  --env MQ_QMGR_NAME=QM1 \
  --publish 1414:1414 \
  --publish 9443:9443 \
  --detach \
  --volume qm1data:/mnt/mqm \
  ibmcom/mq
```

The Docker image always uses `/mnt/mqm` for MQ data, which is correctly linked for you under `/var/mqm` at runtime.  This is to handle problems with file permissions on some platforms.

## Customizing the queue manager configuration

You can customize the configuration in several ways:

1. For getting started, you can use the [default developer configuration](developer-config.md), which is available out-of-the-box for the MQ Advanced for Developers image
2. By creating your own image and adding your own MQSC file into the `/etc/mqm` directory on the image.  This file will be run when your queue manager is created.
3. By using [remote MQ administration](http://www-01.ibm.com/support/knowledgecenter/SSFKSJ_9.0.0/com.ibm.mq.adm.doc/q021090_.htm), via an MQ command server, the MQ HTTP APIs, or using a tool such as the MQ web console or MQ Explorer.

Note that a listener is always created on port 1414 inside the container.  This port can be mapped to any port on the Docker host.

The following is an *example* `Dockerfile` for creating your own pre-configured image, which adds a custom `config.mqsc` and an administrative user `alice`.  Note that it is not normally recommended to include passwords in this way:

```dockerfile
FROM ibmcom/mq
RUN useradd alice -G mqm && \
    echo alice:passw0rd | chpasswd
COPY 20-config.mqsc /etc/mqm/
```

Here is an example corresponding `20-config.mqsc` script from the [mqdev blog](https://www.ibm.com/developerworks/community/blogs/messaging/entry/getting_going_without_turning_off_mq_security?lang=en), which allows users with passwords to connect on the `PASSWORD.SVRCONN` channel:

```
DEFINE CHANNEL(PASSWORD.SVRCONN) CHLTYPE(SVRCONN) REPLACE
SET CHLAUTH(PASSWORD.SVRCONN) TYPE(BLOCKUSER) USERLIST('nobody') DESCR('Allow privileged users on this channel')
SET CHLAUTH('*') TYPE(ADDRESSMAP) ADDRESS('*') USERSRC(NOACCESS) DESCR('BackStop rule')
SET CHLAUTH(PASSWORD.SVRCONN) TYPE(ADDRESSMAP) ADDRESS('*') USERSRC(CHANNEL) CHCKCLNT(REQUIRED)
ALTER AUTHINFO(SYSTEM.DEFAULT.AUTHINFO.IDPWOS) AUTHTYPE(IDPWOS) ADOPTCTX(YES)
REFRESH SECURITY TYPE(CONNAUTH)
```

## Running MQ commands
It is recommended that you configure MQ in your own custom image.  However, you may need to run MQ commands directly inside the process space of the container.  To run a command against a running queue manager, you can use `docker exec`, for example:

```
docker exec \
  --tty \
  --interactive \
  ${CONTAINER_ID} \
  dspmq
```

Using this technique, you can have full control over all aspects of the MQ installation.  Note that if you use this technique to make changes to the filesystem, then those changes would be lost if you re-created your container unless you make those changes in volumes.
