# IBM MQ container

[![Build Status](https://travis-ci.org/ibm-messaging/mq-container.svg?branch=master)](https://travis-ci.org/ibm-messaging/mq-container)

**Note**: The `master` branch may be in an *unstable or even broken state* during development.
To get a stable version, please use the correct [branch](https://github.com/ibm-messaging/mq-container/branches) for your MQ version, instead of the `master` branch.

<img src="https://raw.githubusercontent.com/IBM/charts/master/logo/ibm-mq-icon.svg?sanitize=true" width="100" alt="IBM MQ logo" />

## Overview

Run [IBM® MQ](http://www-03.ibm.com/software/products/en/ibm-mq) in a container.

You can build an image containing either IBM MQ Advanced, or IBM MQ Advanced for Developers.  The developer image includes a [default developer configuration](docs/developer-config.md), to make it easier to get started.  There is also an [incubating](incubating) folder for additional images for other MQ components, which you might find useful.

## Build

After extracting the code from this repository, you can follow the [build documentation](docs/building.md) to build an image.

## Usage

See the [usage documentation](docs/usage.md) for details on how to run a container.

Note that in order to use the image, it is necessary to accept the terms of the [IBM MQ license](#license).

### Environment variables supported by this image

- **LICENSE** - Set this to `accept` to agree to the MQ Advanced for Developers license. If you wish to see the license you can set this to `view`.
- **LANG** - Set this to the language you would like the license to be printed in.
- **MQ_QMGR_NAME** - Set this to the name you want your Queue Manager to be created with.
- **LOG_FORMAT** - Set this to change the format of the logs which are printed on the container's stdout.  Set to "json" to use JSON format (JSON object per line); set to "basic" to use a simple human-readable format.  Defaults to "basic".
- **MQ_ENABLE_METRICS** - Set this to `true` to generate Prometheus metrics for your Queue Manager.

See the [default developer configuration docs](docs/developer-config.md) for the extra environment variables supported by the MQ Advanced for Developers image.

### Kubernetes

If you want to use IBM MQ in [Kubernetes](https://kubernetes.io), you can find an example [Helm](https://helm.sh/) chart here: [IBM charts](https://github.com/IBM/charts).  This can be used to run the container on a cluster, such as [IBM Cloud Private](https://www.ibm.com/cloud-computing/products/ibm-cloud-private/) or the [IBM Cloud Kubernetes Service](https://www.ibm.com/cloud/container-service).

## Issues and contributions

For issues relating specifically to the container image or Helm chart, please use the [GitHub issue tracker](https://github.com/ibm-messaging/mq-container/issues). If you do submit a Pull Request related to this Docker image, please indicate in the Pull Request that you accept and agree to be bound by the terms of the [IBM Contributor License Agreement](CLA.md).

## License

The Dockerfiles and associated code and scripts are licensed under the [Apache License 2.0](http://www.apache.org/licenses/LICENSE-2.0.html).
Licenses for the products installed within the images are as follows:

- [IBM MQ Advanced for Developers](http://www14.software.ibm.com/cgi-bin/weblap/lap.pl?la_formnum=Z125-3301-14&li_formnum=L-APIG-AVCJ4S) (International License Agreement for Non-Warranted Programs). This license may be viewed from an image using the `LICENSE=view` environment variable as described above or by following the link above.
- [IBM MQ Advanced](http://www14.software.ibm.com/cgi-bin/weblap/lap.pl?la_formnum=Z125-3301-14&li_formnum=L-APIG-AV6GV5) (International Program License Agreement). This license may be viewed from an image using the `LICENSE=view` environment variable as described above or by following the link above.
- License information for Ubuntu packages may be found in `/usr/share/doc/${package}/copyright`

Note: The IBM MQ Advanced for Developers license does not permit further distribution and the terms restrict usage to a developer machine.

## Copyright

© Copyright IBM Corporation 2015, 2018
