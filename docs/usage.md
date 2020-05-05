# Usage

In order to use the image, it is necessary to accept the terms of the IBM MQ license.  This is achieved by specifying the environment variable `LICENSE` equal to `accept` when running the image.  You can also view the license terms by setting this variable to `view`. Failure to set the variable will result in the termination of the container with a usage statement.  You can view the license in a different language by also setting the `LANG` environment variable.

> **Note**: You can use `podman` instead of `docker` in any of the examples on this page.

## Running with the default configuration
You can run a queue manager with the default configuration and a listener on port 1414 using the following command.  For example, the following command creates and starts a queue manager called `QM1`, and maps port 1414 on the host to the MQ listener on port 1414 inside the container, as well as port 9443 on the host to the web console on port 9443 inside the container:

```sh
docker run \
  --env LICENSE=accept \
  --env MQ_QMGR_NAME=QM1 \
  --publish 1414:1414 \
  --publish 9443:9443 \
  --detach \
  ibmcom/mq
```

## Running with the default configuration and a volume
The above example will not persist any configuration data or messages across container runs.  In order to do this, you need to use a [volume](https://docs.docker.com/storage/volumes/).  For example, you can create a volume with the following command:

```sh
docker volume create qm1data
```

You can then run a queue manager using this volume as follows:

```sh
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

## Running with the default configuration and Prometheus metrics enabled
You can run a queue manager with [Prometheus](https://prometheus.io) metrics enabled.  The following command will generate Prometheus metrics for your queue manager on `/metrics` port `9157`:

```sh
docker run \
  --env LICENSE=accept \
  --env MQ_QMGR_NAME=QM1 \
  --env MQ_ENABLE_METRICS=true \
  --publish 1414:1414 \
  --publish 9443:9443 \
  --publish 9157:9157 \
  --detach \
  ibmcom/mq
```

## Customizing the queue manager configuration

You can customize the configuration in several ways:

1. For getting started, you can use the [default developer configuration](developer-config.md), which is available out-of-the-box for the MQ Advanced for Developers image
2. By creating your own image and adding your own MQSC file into the `/etc/mqm` directory on the image.  This file will be run when your queue manager is created.
3. By using [remote MQ administration](https://www.ibm.com/support/knowledgecenter/SSFKSJ_9.1.0/com.ibm.mq.adm.doc/q021090_.htm), via an MQ command server, the MQ HTTP APIs, or using a tool such as the MQ web console or MQ Explorer.

Note that a listener is always created on port 1414 inside the container.  This port can be mapped to any port on the Docker host.

The following is an *example* `Dockerfile` for creating your own pre-configured image, which adds a custom MQ configuration file, and an administrative user `alice`.  Note that it is not normally recommended to include passwords in this way:

```dockerfile
FROM ibmcom/mq:9.1.4.0-r1
USER root
RUN useradd alice -G mqm && \
    echo alice:passw0rd | chpasswd
USER mqm
COPY 20-config.mqsc /etc/mqm/
```

The `USER` instructions are necessary to ensure that the `useradd` and `chpasswd` commands are run as the root user.

Here is an example corresponding `20-config.mqsc` script, which creates two local queues:

```mqsc
DEFINE QLOCAL(MY.QUEUE.1) REPLACE
DEFINE QLOCAL(MY.QUEUE.2) REPLACE
```

The file `20-config.mqsc` should be saved into the same directory as the `Dockerfile`.

## Running MQ commands
It is recommended that you configure MQ in your own custom image.  However, you may need to run MQ commands directly inside the process space of the container.  To run a command against a running queue manager, you can use `docker exec`, for example:

```sh
docker exec \
  --tty \
  --interactive \
  ${CONTAINER_ID} \
  dspmq
```

Using this technique, you can have full control over all aspects of the MQ installation.  Note that if you use this technique to make changes to the filesystem, then those changes would be lost if you re-created your container unless you make those changes in volumes.

## Supplying TLS certificates

If you wish to supply TLS Certificates that the queue manager and MQ Console should use for TLS operations then you must supply a PKCS#1 or unencrypted PKCS#8 PEM files for both the certificates and private keys in the following directories:

 * `/etc/mqm/pki/keys/<Label>` - for certificates with public and private keys
 * `/etc/mqm/pki/trust/<index>` - for certificates with only the public key

For example, if you have an identity certificate you wish to add with the label `mykey` and 2 certificates you wish to add as trusted then you would need to add the files into the following locations where files ending in `.key` contain private keys and `.crt` contain certificates:

 - `/etc/mqm/pki/keys/mykey/tls.key`
 - `/etc/mqm/pki/keys/mykey/tls.crt`
 - `/etc/mqm/pki/keys/mykey/ca.crt`
 - `/etc/mqm/pki/trust/0/tls.crt`
 - `/etc/mqm/pki/trust/1/tls.crt`

This can be achieved by either mounting the directories or files into the container when you run it or by baking the files into the correct location in the image. 

If you supply multiple identity certificates then the first label alphabetically will be chosen as the certificate to be used by the MQ Console and the default certificate for the queue manager. If you wish to use a different certificate on the queue manager then you can change the certificate to use at runtime by executing the MQSC command `ALTER QMGR CERTLABL('<newlabel>')`
