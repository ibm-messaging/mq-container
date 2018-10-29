# Building a container image

## Prerequisites

### Prerequisites for building an Ubuntu image
If you want to build a container image with Ubuntu Linux as the base OS, then you need to have the following tools installed:

* [Docker](https://www.docker.com/) V17.06.1 or later
* [GNU make](https://www.gnu.org/software/make/)

If you are working in the Windows Subsystem for Linux, follow [this guide by Microsoft to set up Docker](https://blogs.msdn.microsoft.com/commandline/2017/12/08/cross-post-wsl-interoperability-with-docker/) first.

### Prerequisites for building a Red Hat Enterprise Linux image
If you want to build a container image with Red Hat Enterprise Linux as the base OS, then you need to use a host server with Red Hat Enterprise Linux.  You must also have the following tools installed:

* [`buildah`](https://buildah.io) (available in `rhel-7-server-extras`)
* [`podman`](https://podman.io) (available in `rhel-7-server-extras`)

In addition, you need the following commonly installed tools:

* `bash`
* `coreutils`
* `findutils`
* `make`
* `sed`
* `shadow-utils`
* `tar`

## Building a production image
This procedure works for building the MQ Continuous Delivery release, on `x86_64`, `ppc64le` and `s390x` architectures.

1. Create a `downloads` directory in the root of this repository
2. Download MQ from [IBM Passport Advantage](https://www.ibm.com/software/passportadvantage/) or [IBM Fix Central](https://www.ibm.com/support/fixcentral), and place the downloaded file (for example, `IBM_MQ_9.1_UBUNTU_X86-64.tar.gz` for MQ V9.1.0 for Ubuntu on x86_64 architecture) in the `downloads` directory
3. Run `make build-advancedserver`

> **Warning**: Note that MQ offers two different sets of packaging on Linux: one is called "MQ for Linux" and contains RPM files for installing on Red Hat Enterprise Linux and SUSE Linux Enterprise Server.  The other package is called "MQ for Ubuntu", and contains DEB files for installing on Ubuntu.

On a Red Hat Enterprise Linux host, the command `make build-advancedserver` will build a container image using Red Hat Enterprise Linux as the base.  On all other hosts, the base image will be Ubuntu.

You can build a different version of MQ by setting the `MQ_VERSION` environment variable, for example:

```bash
MQ_VERSION=9.0.5.0 make build-advancedserver
```

If you have an MQ archive file with a different file name, you can specify a particular file (which must be in the `downloads` directory).  You should also specify the MQ version, so that the resulting image is tagged correctly, for example:

```bash
MQ_ARCHIVE=mq-1.2.3.4.tar.gz MQ_VERSION=1.2.3.4 make build-advancedserver
```

## Building a developer image
Run `make build-devserver`, which will download the latest version of MQ Advanced for Developers from IBM developerWorks.  This is currently only available on the `x86_64` architecture.  On a Red Hat Enterprise Linux host, this command will build a container image using Red Hat Enterprise Linux as the base.  On all other hosts, the base image will be Ubuntu.

You can use the environment variable `MQ_ARCHIVE_DEV` to specify an alternative local file to install from (which must be in the `downloads` directory).

## Installed components

This image includes the core MQ server, Java, language packs, and GSKit.  This can be configured by setting the `MQ_PACKAGES` argument to `make`.  For the Ubuntu-based image, you can also directly set a [Docker build argument](https://docs.docker.com/engine/reference/commandline/build/#set-build-time-variables-build-arg).
