# IBM MQ container


[![Build Status](https://travis-ci.org/ibm-messaging/mq-container.svg?branch=master)](https://travis-ci.org/ibm-messaging/mq-container)

**Note**: The `master` branch may be in an *unstable or even broken state* during development.
To get a stable version, please use the correct [branch](https://github.com/ibm-messaging/mq-container/branches) for your MQ version, instead of the `master` branch.

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
- **MQ_QMGR_LOG_FILE_PAGES** - Set this to control the value for LogFilePages passed to the "crtmqm" command.  Cannot be changed after queue manager creation.
- **MQ_LOGGING_CONSOLE_SOURCE** - Specifies a comma-separated list of sources for logs which are mirrored to the container's stdout. The valid values are "qmgr", "web" and "mqsc". Defaults to "qmgr,web". 
- **MQ_LOGGING_CONSOLE_FORMAT** - Changes the format of the logs which are printed on the container's stdout.  Set to "json" to use JSON format (JSON object per line); set to "basic" to use a simple human-readable format.  Defaults to "basic".
- **MQ_LOGGING_CONSOLE_EXCLUDE_ID** - Excludes log messages with the specified ID.  The log messages still appear in the log file on disk, but are excluded from the container's stdout.  Defaults to "AMQ5041I,AMQ5052I,AMQ5051I,AMQ5037I,AMQ5975I".
- **MQ_ENABLE_METRICS** - Set this to `true` to generate Prometheus metrics for your Queue Manager.

See the [default developer configuration docs](docs/developer-config.md) for the extra environment variables supported by the MQ Advanced for Developers image.

### Kubernetes

If you want to use IBM MQ on [Kubernetes](https://kubernetes.io), you can find an example [Helm](https://helm.sh/) chart here: [IBM MQ Sample Helm Chart](https://github.com/ibm-messaging/mq-helm).  This can be used to run the container on a Kubernetes cluster, such as the [IBM Cloud Kubernetes Service](https://www.ibm.com/cloud/container-service).

## Issues and contributions

For issues relating specifically to the container image or Helm chart, please use the [GitHub issue tracker](https://github.com/ibm-messaging/mq-container/issues). Pull requests are not currently accepted.

## License

The Dockerfiles and associated code and scripts are licensed under the [Apache License 2.0](http://www.apache.org/licenses/LICENSE-2.0.html).
Licenses for the products installed within the images are as follows:

- [IBM MQ Advanced for Developers](http://www14.software.ibm.com/cgi-bin/weblap/lap.pl?la_formnum=Z125-3301-14&li_formnum=L-HYGL-6STWD6) (International License Agreement for Non-Warranted Programs). This license may be viewed from an image using the `LICENSE=view` environment variable as described above or by following the link above.
- [IBM MQ Advanced](http://www14.software.ibm.com/cgi-bin/weblap/lap.pl?la_formnum=Z125-3301-14&li_formnum=L-NUUP-23NH8Y) (International Program License Agreement). This license may be viewed from an image using the `LICENSE=view` environment variable as described above or by following the link above.

Note: The IBM MQ Advanced for Developers license does not permit further distribution and the terms restrict usage to a developer machine.


## Copyright

© Copyright IBM Corporation 2015, 2024
