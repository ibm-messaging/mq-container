# Building a container image

## Prerequisites

You need to have the following tools installed:

* [Docker](https://www.docker.com/) V17.06.1 or later
* [GNU make](https://www.gnu.org/software/make/)

If you are working in the Windows Subsystem for Linux, follow [this guide by Microsoft to set up Docker](https://blogs.msdn.microsoft.com/commandline/2017/12/08/cross-post-wsl-interoperability-with-docker/) first.

## Building a production image

This procedure works for building the MQ Continuous Delivery release, on `amd64`, `ppc64le` and `s390x` architectures.

1. Create a `downloads` directory in the root of this repository
2. Download MQ from [IBM Passport Advantage](https://www.ibm.com/software/passportadvantage/) or [IBM Fix Central](https://www.ibm.com/support/fixcentral), and place the downloaded file (for example, `IBM_MQ_9.1.3_LINUX_X86-64.tar.gz`) in the `downloads` directory
3. Run `make build-advancedserver`

> **Warning**: Note that MQ offers two different sets of packaging on Linux: one is called "MQ for Linux" and contains RPM files for installing on Red Hat Enterprise Linux and SUSE Linux Enterprise Server.  The MQ container build uses a Red Hat Universal Base Image, so you need the "MQ for Linux" RPM files.

You can build a different version of MQ by setting the `MQ_VERSION` environment variable, for example:

```bash
MQ_VERSION=9.1.0.0 make build-advancedserver
```

If you have an MQ archive file with a different file name, you can specify a particular file (which must be in the `downloads` directory).  You should also specify the MQ version, so that the resulting image is tagged correctly, for example:

```bash
MQ_ARCHIVE=mq-1.2.3.4.tar.gz MQ_VERSION=1.2.3.4 make build-advancedserver
```

## Building a developer image
Run `make build-devserver`, which will download the latest version of MQ Advanced for Developers from IBM developerWorks.  This is currently only available on the `amd64` architecture.

You can use the environment variable `MQ_ARCHIVE_DEV` to specify an alternative local file to install from (which must be in the `downloads` directory).

## Building from a Red Hat Enterprise Linux host
Red Hat Enterprise Linux (RHEL) offers a suite of container tools, including Buildah for building container images, and Podman for running containers.  Buildah can accept input described in a [Dockerfile](https://docs.docker.com/engine/reference/builder/).  This MQ sample uses a multi-stage build, which requires a recent version of Podman, which is not yet available in Red Hat Enterprise Linux V7.  Therefore, if you are on a RHEL host, then the `build-devserver` and `build-advancedserver` targets are run using a more recent version of Buildah from inside a container.

The containerized build process on a RHEL host will write an OCI compliant archive file to `/tmp/mq-buildah`.  If a version of Docker is installed on the host, it will also push the image into Docker's internal image registry.

## Installed components

This image includes the core MQ server, Java, language packs, GSKit, and web server.  This can be configured by setting the `MQ_PACKAGES` argument to `make`.
