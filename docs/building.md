# Building a container image

## Prerequisites

You need to have the following tools installed:

* [Docker](https://www.docker.com/) 17.06.1 or later, or [Podman](https://podman.io) 1.0 or later (Podman 4.1 on macOS).
* [GNU make](https://www.gnu.org/software/make/)

If you are working in the Windows Subsystem for Linux, follow [this guide by Microsoft to set up Docker](https://blogs.msdn.microsoft.com/commandline/2017/12/08/cross-post-wsl-interoperability-with-docker/) first.

## Building Images

To build an IBM MQ image, navigate to the appropriate section:

- [Building a production image](#building-a-production-image)
- [Building a developer image](#building-a-developer-image)

## Building a production image

From MQ 9.2.X, the MQ container adds support for MQ Long Term Support (LTS) **production licensed** releases.

### MQ Continuous Delivery (CD)

The procedure below is for building the 9.4.1 release, on `amd64`, `ppc64le` and `s390x` architectures.

1. Create a `downloads` directory in the root of this repository
2. Download MQ from [IBM Passport Advantage](https://www.ibm.com/software/passportadvantage/). Identify the correct 'Continuous Delivery release' eImage part number for your architecture from the CD tab at https://www.ibm.com/support/pages/downloading-ibm-mq-94-cd
3. Ensure the `tar.gz` file is in the `downloads` directory
4. Run `make build-advancedserver`

If you have an MQ archive file with a different file name, you can specify a particular file (which must be in the `downloads` directory).  You should also specify the MQ version, so that the resulting image is tagged correctly, for example:

```bash
MQ_ARCHIVE=mq-1.2.3.4.tar.gz MQ_VERSION=1.2.3.4 make build-advancedserver
```

### MQ Long Term Support (LTS)

The procedure below is for building the 9.3.0 release, on `amd64`, `ppc64le` and `s390x` architectures.

1. Create a `downloads` directory in the root of this repository
2. Download MQ from [IBM Passport Advantage](https://www.ibm.com/software/passportadvantage/). Identify the correct 'Long Term Support Release for Containers' eImage part number for your architecture from the 9.3.0 LTS tab at https://www.ibm.com/support/pages/downloading-ibm-mq-930
3. Ensure the `tar.gz` file is in the `downloads` directory
4. Run `make build-advancedserver`

If you have an MQ archive file with a different file name, you can specify a particular file (which must be in the `downloads` directory).  You should also specify the MQ version, so that the resulting image is tagged correctly, for example:

```bash
MQ_ARCHIVE=mq-1.2.3.4.tar.gz MQ_VERSION=1.2.3.4 make build-advancedserver
```

## Building a developer image

Run `make build-devserver`, which will download the latest version of MQ Advanced for Developers.  This is available on the `amd64` and `arm64` (Apple Silicon) architectures.

You can use the environment variable `MQ_ARCHIVE_DEV` to specify an alternative local file to install from (which must be in the `downloads` directory).

## Installed components

This image includes the core MQ server, Java, language packs, GSKit, and web server.  This is configured in the `mq-redux` build stage in `Dockerfile-server`.
