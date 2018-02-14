# Building a Docker image 

## Prerequisites
You need to ensure you have the following tools installed:
* [Docker](https://www.docker.com/) V17.06.1 or later
* [GNU make](https://www.gnu.org/software/make/)

## Building a production image
This procedure works for building the MQ Continuous Delivery release, on `x86_64`, `ppc64le` and `s390x` architectures.

1. Download MQ from IBM Passport Advantage, and place the downloaded file (for example, `IBM_MQ_9.0.4.0_UBUNTU_X86-64.tar.gz` for MQ V9.0.4 for Ubuntu on x86_64 architecture) in the `downloads` directory
2. Run `make build-advancedserver`

> **Warning**: Note that MQ offers two different sets of packaging on Linux: one is called "MQ for Linux" and contains RPM files for installing on Red Hat Enterprise Linux and SUSE Linux Enterprise Server.  The other package is called "MQ for Ubuntu", and contains DEB files for installing on Ubuntu.

You can build a different version of MQ by setting the `MQ_VERSION` environment variable, for example:

```bash
MQ_VERSION=9.0.4.0 make build-advancedserver
```

If you have an MQ archive file with a different file name, you can specify a particular file (which must be in the `downloads` directory).  You should also specify the MQ version, so that the resulting image is tagged correctly, for example:

```bash
MQ_ARCHIVE=mq-1.2.3.4.tar.gz MQ_VERSION=1.2.3.4 make build-advancedserver
```

## Building a developer image
Run `make build-devserver`, which will download the latest version of MQ Advanced for Developers from IBM developerWorks.  This is currently only available on the `x86_64` architecture.

## Building on a different base image
By default, the MQ images use Ubuntu as the base layer.  You can build using a Red Hat Enterprise Linux compatible base layer by setting the `BASE_IMAGE` environment variable.  For example:

```
BASE_IMAGE=centos:7 make build-advancedserver
```

The `make` tool will try and locate the right archive file under the `downloads` directory, based on your platform architecture and your `MQ_VERSION` environment variable, for example `IBM_MQ_9.0.4.0_LINUX_X86_64.tar.gz` for MQ V9.0.4.0 on x86_64.  You can also set the `MQ_ARCHIVE` environment variable to set the specific file name.

Note that if you are using Red Hat Enterprise Linux, you will need to create your own base image layer, with your subscription enabled, as described [here](https://access.redhat.com/documentation/en-us/red_hat_enterprise_linux_atomic_host/7/html/getting_started_with_containers/get_started_with_docker_formatted_container_images).  The MQ image build needs to install some additional packages, and a subscription is required to access the Red Hat repositories.