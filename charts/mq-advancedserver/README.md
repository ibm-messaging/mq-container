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
| ----------------------------    | ---------------------------------------------   | ---------------------------------------------------------- |
| `license`                    | Set to `accept` to accept the terms of the IBM license  | `not accepted`                                                      |
| `image.name`                    | Image name                                      | `nil`                                                      |
| `image.tag`                     | Image tag                                       | `nil`                                                      |
| `image.pullPolicy`              | Image pull policy                               | `IfNotPresent`                                             |
| `image.pullSecret`              | Image pull secret, if you are using a private Docker registry | `nil`                                        |
| `data.persistence.enabled`      | Use a PersistentVolume to persist MQ data (under `/var/mqm`)  | `true`                                       |
| `data.persistence.storageClass` | Storage class of backing Persistent Volume                    | `nil`                                        |
| `data.persistence.size`         | Size of data volume                             | `2Gi`                                                      |
| `service.name`                  | Name of the Kubernetes service to create        | `qmgr`                                                     |
| `service.type`                  | Kubernetes service type exposing ports, e.g. `NodePort`       | `ClusterIP`                                  |
| `queueManager.name`                  | MQ Queue Manager name       | Helm release name                                  |

Specify each parameter using the `--set key=value[,key=value]` argument to `helm install`.

Alternatively, a YAML file that specifies the values for the parameters can be provided while installing the chart.

> **Tip**: You can use the default [values.yaml](values.yaml)

## Persistence

The chart mounts a [Persistent Volume](http://kubernetes.io/docs/user-guide/persistent-volumes/).


# Configuring MQ objects
You have two major options for configuring the MQ queue manager itself:

1. Use existing tools, such as `runmqsc`, MQ Explorer or the MQ Command Server to configure your queue manager directly.
2. Create a new image with your configuration baked-in 

## Configuring MQ objects with a new image
You can create a new container image layer, on top of the IBM MQ Advanced base image.  You can add MQSC files to define MQ objects such as queues and topics, and place these files into `/etc/mqm` in your image.  When the MQ container starts, it will run any MQSC files found in this directory (in sorted order).

# Copyright

© Copyright IBM Corporation 2017