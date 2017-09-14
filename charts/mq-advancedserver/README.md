<p align="center"><img src="https://developer.ibm.com/messaging/wp-content/uploads/sites/18/2017/07/IBM-MQ-Square-200.png" width="150"></p>

# IBM MQ

IBM® MQ is messaging middleware that simplifies and accelerates the integration of diverse applications and business data across multiple platforms. It uses message queues to facilitate the exchanges of information and offers a single messaging solution for cloud, mobile, Internet of Things (IoT) and on-premises environments.

# Introduction

This chart deploys a single IBM MQ Advanced server (queue manager) into an IBM Cloud private or other Kubernetes environment.

## Prerequisites

- Kubernetes 1.5 or greater, with beta APIs enabled
- If persistence is enabled (see [configuration](#configuration)), then you either need to create a PersistentVolume, or specify a Storage Class if classes are defined in your cluster.

## Installing the Chart

To install the chart with the release name `foo`:

```bash
helm install --name foo stable/mq-advancedserver --set license=accept
```

This command accepts the [IBM MQ Advanced license](LICENSE) and deploys an MQ Advanced server on the Kubernetes cluster. The [configuration](#configuration) section lists the parameters that can be configured during installation.

> **Tip**: See all the resources deployed by the chart using `kubectl get all -l release=foo`

## Uninstalling the Chart

To uninstall/delete the `foo` release:

```bash
helm delete foo
```

The command removes all the Kubernetes components associated with the chart, except any Persistent Volume Claims (PVCs).  This is the default behavior of Kubernetes, and ensures that valuable data is not deleted.  In order to delete the Queue Manager's data, you can delete the PVC using the following command:

```bash
kubectl delete pvc -l release=foo
```

## Configuration
The following table lists the configurable parameters of the `mq-advancedserver` chart and their default values.

| Parameter                       | Description                                     | Default                                                    |
| ------------------------------- | ----------------------------------------------- | ---------------------------------------------------------- |
| `license`                       | Set to `accept` to accept the terms of the IBM license  | `not accepted`                                     |
| `image.repository`              | Image full name including repository            | `nil`                                                      |
| `image.tag`                     | Image tag                                       | `nil`                                                      |
| `image.pullPolicy`              | Image pull policy                               | `IfNotPresent`                                             |
| `image.pullSecret`              | Image pull secret, if you are using a private Docker registry | `nil`                                        |
| `data.persistence.enabled`      | Use a PersistentVolume to persist MQ data (under `/var/mqm`)  | `true`                                       |
| `data.persistence.storageClass` | Storage class of backing Persistent Volume      | `nil`                                                      |
| `data.persistence.size`         | Size of data volume                             | `2Gi`                                                      |
| `service.name`                  | Name of the Kubernetes service to create        | `qmgr`                                                     |
| `service.type`                  | Kubernetes service type exposing ports, e.g. `NodePort`       | `ClusterIP`                                  |
| `resources.limits.cpu`          | Kubernetes CPU limit for the Queue Manager container | `1`                                                   |
| `resources.limits.memory`       | Kubernetes memory limit for the Queue Manager container | `1Gi`                                              |
| `resources.requests.cpu`        | Kubernetes CPU request for the Queue Manager container | `1`                                                 |
| `resources.requests.memory`     | Kubernetes memory request for the Queue Manager container | `1Gi`                                            |
| `queueManager.name`             | MQ Queue Manager name       | Helm release name                                                              |
| `nameOverride`                  | Set to partially override the resource names used in this chart | `nil`                                      |
| `livenessDelay`                 | Raises the time out before Kubernetes checks for Queue Manager's health. Useful for slower systems that take longer to start the Queue Manager. | 60 |

Specify each parameter using the `--set key=value[,key=value]` argument to `helm install`.

Alternatively, a YAML file that specifies the values for the parameters can be provided while installing the chart.

> **Tip**: You can use the default [values.yaml](values.yaml)

## Persistence

The chart mounts a [Persistent Volume](http://kubernetes.io/docs/user-guide/persistent-volumes/).


# Configuring MQ objects
You have two major options for configuring the MQ queue manager itself:

1. Use existing tools, such as `runmqsc`, MQ Explorer or the MQ Command Server to configure your queue manager directly.
2. Create a new image with your configuration baked-in

If you decide to opt for option 1 you will need to create any administrative entry point to your Queue Manager. This can be completed by either manually running kubectl commands to execute `runmqsc` and configure your entry point or creating a new image which automatically does this. At a minimum you will need to:

* Create a user with MQ administrative permissions (is a member of the `mqm` group) which you can use to log into your Queue Manager.
* Enable `ADOPTCTX` so we use the user for authorization as well as authentication when connecting a MQ Application.
* Refresh the security configuration so the new `ADOPTCTX` value becomes active
* Create a channel to use as our entrypoint.
* Create a channel authentication rule to allow access for administrative users to connect through the channel.

For the above minimum you should execute the following commands through a shell prompt on the pod, if you choose to do this then you should replace `mquser` with your own username:

```
useradd --gid mqm mquser
passwd mquser
runmqsc <QM Name>
```

Then in the runmqsc program i would execute the following MQSC commands:

```
DEFINE CHANNEL('EXAMPLE.ENTRYPOINT') CHLTYPE(SVRCONN)
ALTER AUTHINFO(SYSTEM.DEFAULT.AUTHINFO.IDPWOS) AUTHTYPE(IDPWOS) ADOPTCTX(YES)
REFRESH SECURITY(*) TYPE(CONNAUTH)
SET CHLAUTH('EXAMPLE.ENTRYPOINT') TYPE(BLOCKUSER) USERLIST('nobody')
```

At this point you can now connect a MQ Explorer or other remote MQ administrative client using the channel `EXAMPLE.ENTRYPOINT` and user `mquser`.

> **Tip**: If you are using a client that has a compatibility mode option for user authentication to connect to your IBM MQ Queue Manager. Make sure you have compatibility mode turned off.

## Configuring MQ objects with a new image
You can create a new container image layer, on top of the IBM MQ Advanced base image.  You can add MQSC files to define MQ objects such as queues and topics, and place these files into `/etc/mqm` in your image.  When the MQ pod starts, it will run any MQSC files found in this directory (in sorted order).

## Example Dockerfile and MQSC script for creating a new image
In this example you will create a Dockerfile that creates two users:
* `admin` - Administrator users which is a member of the `mqm` group
* `app` - Client application user which is a member of the `mqclient` group. (You will also create this group)

You will also create a MQSC Script file called `config.mqsc` that will be copied to `/etc/mqm` on my pod so it is ran automatically at startup. This script will do multiple things including:
* Creating default Local Queues for my applications
* Creating channels for my Admin and App users
* Configuring security to allow use of the channels by remote applications
* Creating authority records to allow members of the `mqclient` group to access the Queue Manager and the default Local Queues.

First create a file called `config.MQSC`. This the MQSC file that will be ran at startup. It should contain the following:

```
* Create Local Queues that my application(s) can use.
DEFINE QLOCAL('EXAMPLE.QUEUE.1') REPLACE
DEFINE QLOCAL('EXAMPLE.QUEUE.2') REPLACE

* Create a Dead Letter Queue for undeliverable messages and set the Queue Manager to use it.
DEFINE QLOCAL('EXAMPLE.DEAD.LETTER.QUEUE') REPLACE
ALTER QMGR DEADQ('EXAMPLE.DEAD.LETTER.QUEUE')

* Set ADOPTCTX to YES so we use the same userid passed for authentication as the one for authorization and refresh the security configuration
ALTER AUTHINFO(SYSTEM.DEFAULT.AUTHINFO.IDPWOS) AUTHTYPE(IDPWOS) ADOPTCTX(YES)
REFRESH SECURITY(*) TYPE(CONNAUTH)

* Create a entry channel for the Admin user and Application user
DEFINE CHANNEL('EXAMP.ADMIN.SVRCONN') CHLTYPE(SVRCONN) REPLACE
DEFINE CHANNEL('EXAMP.APP.SVRCONN') CHLTYPE(SVRCONN) MCAUSER('app') REPLACE

* Set Channel authentication rules to only allow access through the two channels we created and only allow admin users to connect through EXAMPLE.ADMIN.SVRCONN
SET CHLAUTH('*') TYPE(ADDRESSMAP) ADDRESS('*') USERSRC(NOACCESS) DESCR('Back-stop rule - Blocks everyone') ACTION(REPLACE)
SET CHLAUTH('EXAMP.APP.SVRCONN') TYPE(ADDRESSMAP) ADDRESS('*') USERSRC(CHANNEL) DESCR('Allows connection via APP channel') ACTION(REPLACE)
SET CHLAUTH('EXAMP.ADMIN.SVRCONN') TYPE(ADDRESSMAP) ADDRESS('*') USERSRC(CHANNEL) DESCR('Allows connection via ADMIN channel') ACTION(REPLACE)
SET CHLAUTH('EXAMP.ADMIN.SVRCONN') TYPE(BLOCKUSER) USERLIST('nobody') DESCR('Allows admins on ADMIN channel') ACTION(REPLACE)

* Set Authority records to allow the members of the mqclient group to connect to the Queue Manager and access the Local Queues which start with "EXAMPLE."
SET AUTHREC OBJTYPE(QMGR) GROUP('mqclient') AUTHADD(CONNECT,INQ)
SET AUTHREC PROFILE('EXAMPLE.**') OBJTYPE(QUEUE) GROUP('mqclient') AUTHADD(INQ,PUT,GET,BROWSE)
```

Next create a `Dockerfile` that expands on the MQ Advanced server image to create the users and groups. It should contain the following, replacing <IMAGE NAME> with the mqadvanced image you want to base this new image off:

```
FROM <IMAGE NAME>

# Add the admin user as a member of the mqm group and set their password
RUN useradd admin -G mqm \
    && echo admin:passw0rd | chpasswd \
# Create the mqclient group
    && groupadd mqclient \
# Create the app user as a member of the mqclient group and set their password
    && useradd app -G mqclient \
    && echo app:passw0rd | chpasswd

# Copy the configuration script to /etc/mqm where it will be picked up automatically
COPY config.mqsc /etc/mqm/
```

Finally build and push the image to your registry.

You can then use the new image when you deploy MQ into your cluster. You will find that once you have run the image you will be able to see your new default objects and users.

# Copyright

© Copyright IBM Corporation 2017
