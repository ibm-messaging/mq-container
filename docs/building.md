# Building a container image

## Prerequisites

You need to have the following tools installed:

* [Docker](https://www.docker.com/) V17.06.1 or later, or [Podman](https://podman.io) V1.0 or later
* [GNU make](https://www.gnu.org/software/make/)

If you are working in the Windows Subsystem for Linux, follow [this guide by Microsoft to set up Docker](https://blogs.msdn.microsoft.com/commandline/2017/12/08/cross-post-wsl-interoperability-with-docker/) first.

You will also need a [Red Hat Account](https://access.redhat.com) to be able to access the Red Hat Registry. 

## Building a production image

From MQ 9.2.X, the MQ container adds support for MQ Long Term Support (LTS) **production licensed** releases.

### MQ Continuous Delivery (CD)

Note: To build the latest Continuous Delivery (CD) version, follow the latest build [instructions](/../master/docs/building.md#building-a-production-image).

### MQ Long Term Support (LTS)

Note: 9.2.0.X is no longer the latest LTS release; MQ 9.3 is the latest MQ version with MQ Long Term Support (LTS). To build MQ 9.3, follow the building [instructions](/../master/docs/building.md#building-a-production-image) for MQ 9.3.

However, if you wish to build the previous 9.2.0.X MQ LTS, follow the procedure below for `amd64` and `s390x` architectures.

1. Create a `downloads` directory in the root of this repository
2. Download MQ from [IBM Passport Advantage](https://www.ibm.com/software/passportadvantage/). Identify the correct 'Long Term Support for containers' eImage part number for your architecture from the appropriate 9.2.0.X LTS tab at https://www.ibm.com/support/pages/downloading-ibm-mq-92.
3. Ensure the `tar.gz` file is in the `downloads` directory
4. Run `LTS=true make build-advancedserver`

If you have an MQ archive file with a different file name, you can specify a particular file (which must be in the `downloads` directory).  You should also specify the MQ version, so that the resulting image is tagged correctly, for example:

```bash
MQ_ARCHIVE=mq-1.2.3.4.tar.gz MQ_VERSION=1.2.3.4 LTS=true make build-advancedserver
```

## Building a developer image

Login to the Red Hat Registry: `docker login registry.redhat.io` using your Customer Portal credentials.
Run `make build-devserver`, which will download the latest version of MQ Advanced for Developers from IBM developerWorks.  This is currently only available on the `amd64` architecture.

You can use the environment variable `MQ_ARCHIVE_DEV` to specify an alternative local file to install from (which must be in the `downloads` directory).

## Installed components

This image includes the core MQ server, Java, language packs, GSKit, and web server.  This is configured in the `Generate MQ package in INSTALLATION_DIR` section [here](../install-mq.sh), with the configured options being picked up at build time.
