![IBM MQ logo](https://developer.ibm.com/messaging/wp-content/uploads/sites/18/2017/07/IBM-MQ-Square-200.png)

# IBM MQ

IBM® MQ is messaging middleware that simplifies and accelerates the integration of diverse applications and business data across multiple platforms. It uses message queues to facilitate the exchanges of information and offers a single messaging solution for cloud, mobile, Internet of Things (IoT) and on-premises environments.

# Introduction

This chart deploys a single IBM MQ Advanced server (queue manager) into an IBM Cloud private or other Kubernetes environment.

## Prerequisites

- Kubernetes 1.7 or greater, with beta APIs enabled
- If persistence is enabled (see [configuration](#configuration)), then you either need to create a PersistentVolume, or specify a Storage Class if classes are defined in your cluster.

## Installing the Chart

To install the chart with the release name `foo`:

```sh
helm install --name foo stable/ibm-mqadvanced-server-prod --set license=accept
```

This command accepts the [IBM MQ Advanced license](LICENSE) and deploys an MQ Advanced server on the Kubernetes cluster. The [configuration](#configuration) section lists the parameters that can be configured during installation.

> **Tip**: See all the resources deployed by the chart using `kubectl get all -l release=foo`

## Uninstalling the Chart

To uninstall/delete the `foo` release:

```sh
helm delete foo
```

The command removes all the Kubernetes components associated with the chart, except any Persistent Volume Claims (PVCs).  This is the default behavior of Kubernetes, and ensures that valuable data is not deleted.  In order to delete the Queue Manager's data, you can delete the PVC using the following command:

```sh
kubectl delete pvc -l release=foo
```

## Configuration
The following table lists the configurable parameters of the `ibm-mqadvanced-server-prod` chart and their default values.

| Parameter                       | Description                                                     | Default                                    |
| ------------------------------- | --------------------------------------------------------------- | ------------------------------------------ |
| `license`                       | Set to `accept` to accept the terms of the IBM license          | `"not accepted"`                           |
| `image.repository`              | Image full name including repository                            | `nil`                                      |
| `image.tag`                     | Image tag                                                       | `nil`                                      |
| `image.pullPolicy`              | Image pull policy                                               | `IfNotPresent`                             |
| `image.pullSecret`              | Image pull secret, if you are using a private Docker registry   | `nil`                                      |
| `persistence.enabled`           | Use persistent volumes for all defined volumes                  | `true`                                     |
| `persistence.useDynamicProvisioning` | Use dynamic provisioning (storage classes) for all volumes | `true`                                     |
| `dataPVC.name`                  | Suffix for the PVC name                                         | `"data"`                                   |
| `dataPVC.storageClassName`      | Storage class of volume for main MQ data (under `/var/mqm`)     | `""`                                       |
| `dataPVC.size`                  | Size of volume for main MQ data (under `/var/mqm`)              | `2Gi`                                      |
| `service.name`                  | Name of the Kubernetes service to create                        | `"qmgr"`                                   |
| `service.type`                  | Kubernetes service type exposing ports, e.g. `NodePort`         | `ClusterIP`                                |
| `resources.limits.cpu`          | Kubernetes CPU limit for the Queue Manager container            | `1`                                        |
| `resources.limits.memory`       | Kubernetes memory limit for the Queue Manager container         | `1Gi`                                      |
| `resources.requests.cpu`        | Kubernetes CPU request for the Queue Manager container          | `1`                                        |
| `resources.requests.memory`     | Kubernetes memory request for the Queue Manager container       | `1Gi`                                      |
| `queueManager.name`             | MQ Queue Manager name                                           | Helm release name                          |
| `nameOverride`                  | Set to partially override the resource names used in this chart | `nil`                                      |
| `livenessProbe.initialDelaySeconds` | The initial delay before starting the liveness probe. Useful for slower systems that take longer to start the Queue Manager. | 60 |
| `livenessProbe.periodSeconds` | How often to run the probe | 10 |
| `livenessProbe.timeoutSeconds` | Number of seconds after which the probe times out | 5 |
| `livenessProbe.failureThreshold` | Minimum consecutive failures for the probe to be considered failed after having succeeded | 1 |
| `readinessProbe.initialDelaySeconds` | The initial delay before starting the readiness probe | 10 |
| `readinessProbe.periodSeconds` | How often to run the probe | 5 |
| `readinessProbe.timeoutSeconds` | Number of seconds after which the probe times out | 3 |
| `readinessProbe.failureThreshold` | Minimum consecutive failures for the probe to be considered failed after having succeeded | 1 |

Specify each parameter using the `--set key=value[,key=value]` argument to `helm install`.

Alternatively, a YAML file that specifies the values for the parameters can be provided while installing the chart.

> **Tip**: You can use the default [values.yaml](values.yaml)

## Persistence

The chart mounts a [Persistent Volume](http://kubernetes.io/docs/user-guide/persistent-volumes/).


# Configuring MQ objects
You have two major options for configuring the MQ queue manager itself:

1. Use existing tools, such as `runmqsc`, MQ Explorer or the MQ Command Server to configure your queue manager directly.
2. Create a new image layer with your configuration baked-in

## Configuring MQ using existing tools
You will need to create any administrative entry point to your Queue Manager. This can be completed by either manually running kubectl commands to execute `runmqsc` and configure your entry point or creating a new image which automatically does this. At a minimum you will need to:

* Create a user with MQ administrative permissions (is a member of the `mqm` group) which you can use to log into your Queue Manager.
* Enable `ADOPTCTX` so we use the user for authorization as well as authentication when connecting a MQ Application.
* Refresh the security configuration so the new `ADOPTCTX` value becomes active
* Create a channel to use as our entrypoint.
* Create a channel authentication rule to allow access for administrative users to connect through the channel.

For the above minimum you, should execute the following commands through a shell prompt on the pod.  If you choose to do this then you should replace `mquser` with your own username:

```sh
useradd --gid mqm mquser
passwd mquser
runmqsc <QM Name>
```

Then in the `runmqsc` program, you could execute the following MQSC commands:

```
DEFINE CHANNEL('EXAMPLE.ENTRYPOINT') CHLTYPE(SVRCONN)
ALTER AUTHINFO(SYSTEM.DEFAULT.AUTHINFO.IDPWOS) AUTHTYPE(IDPWOS) ADOPTCTX(YES)
REFRESH SECURITY(*) TYPE(CONNAUTH)
SET CHLAUTH('EXAMPLE.ENTRYPOINT') TYPE(BLOCKUSER) USERLIST('nobody')
```

At this point you could now connect a MQ Explorer or other remote MQ administrative client using the channel `EXAMPLE.ENTRYPOINT` and user `mquser`.

> **Tip**: If you are using a client that has a compatibility mode option for user authentication to connect to your IBM MQ Queue Manager. Make sure you have compatibility mode turned off.

## Configuring MQ using a new image layer
You can create a new container image layer, on top of the IBM MQ Advanced base image.  You can add MQSC files to define MQ objects such as queues and topics, and place these files into `/etc/mqm` in your image.  When the MQ pod starts, it will run any MQSC files found in this directory (in sorted order).

### Example Dockerfile and MQSC script for creating a new image
In this example you will create a Dockerfile that creates two users:
* `admin` - Administrator user which is a member of the `mqm` group
* `app` - Client application user which is a member of the `mqclient` group. (You will also create this group)

You will also create a MQSC Script file called `config.mqsc` that will be run automatically when your container starts. This script will do the following:
* Create default local queues for my applications
* Create channels for use by the `admin` and `app` users
* Configure security to allow use of the channels by remote applications
* Create authority records to allow members of the `mqclient` group to access the Queue Manager and the default local queues.

First create a file called `config.mqsc`. This the MQSC file that will be run when an MQ container starts. It should contain the following:

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

Next create a `Dockerfile` that expands on the MQ Advanced Server image to create the users and groups. It should contain the following, replacing `<IMAGE NAME>` with the MQ image you want to use as a base:

```dockerfile
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

Finally, build and push the image to your registry.

You can then use the new image when you deploy MQ into your cluster. You will find that once you have run the image you will be able to see your new default objects and users.

# Copyright

© Copyright IBM Corporation 2017
