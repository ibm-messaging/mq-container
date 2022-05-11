# Building a container image

## Prerequisites

You need to have the following tools installed:

* [Docker](https://www.docker.com/) 17.06.1 or later, or [Podman](https://podman.io) 1.0 or later (Podman 4.1 on macOS).  If using Podman on macOS, the you need to be in "rootful" mode to allow the use of a network during builds.  Run `podman machine init --rootful`.
* [GNU make](https://www.gnu.org/software/make/)

If you are working in the Windows Subsystem for Linux, follow [this guide by Microsoft to set up Docker](https://blogs.msdn.microsoft.com/commandline/2017/12/08/cross-post-wsl-interoperability-with-docker/) first.

## Building a production image

From MQ 9.2.X, the MQ container adds support for MQ Long Term Support (LTS) **production licensed** releases.

### MQ Continuous Delivery (CD)

This procedure works for building the MQ Continuous Delivery release, on `amd64`, `ppc64le` and `s390x` architectures.

1. Create a `downloads` directory in the root of this repository
2. Download MQ from [IBM Passport Advantage](https://www.ibm.com/software/passportadvantage/) or [IBM Fix Central](https://www.ibm.com/support/fixcentral), and place the downloaded file (for example, `IBM_MQ_9.3.0_LINUX_X86-64_NOINST.tar.gz`) in the `downloads` directory
3. Run `make build-advancedserver`

> **Warning**: Note that from MQ 9.2.X CD, the MQ container build uses a 'No-Install' MQ Package, available under `IBM MQ V9.2.x Continuous Delivery Release components eAssembly, part no. CJ7CNML`

If you have an MQ archive file with a different file name, you can specify a particular file (which must be in the `downloads` directory).  You should also specify the MQ version, so that the resulting image is tagged correctly, for example:

```bash
MQ_ARCHIVE=mq-1.2.3.4.tar.gz MQ_VERSION=1.2.3.4 make build-advancedserver
```

### MQ Long Term Support (LTS)

This procedure works for building the MQ Long Term Support release, on `amd64`, `ppc64le` and `s390x` architectures.

1. Create a `downloads` directory in the root of this repository
2. Download MQ from [IBM Passport Advantage](https://www.ibm.com/software/passportadvantage/) or [IBM Fix Central](https://www.ibm.com/support/fixcentral), and place the downloaded file (for example, `9.2.0.4-IBM-MQ-Advanced-Non-Install-LinuxX86.tar.gz`) in the `downloads` directory
3. Run `LTS=true make build-advancedserver`

> **Warning**: Note that from MQ 9.2 LTS, the MQ container build uses a 'No-Install' MQ Package, available under `IBM MQ V9.2 Long Term Support Release components eAssembly, part no. CXXXXXX`

If you have an MQ archive file with a different file name, you can specify a particular file (which must be in the `downloads` directory).  You should also specify the MQ version, so that the resulting image is tagged correctly, for example:

```bash
MQ_ARCHIVE=mq-1.2.3.4.tar.gz MQ_VERSION=1.2.3.4 LTS=true make build-advancedserver
```

## Building a developer image

Run `make build-devserver`, which will download the latest version of MQ Advanced for Developers from IBM developerWorks.  This is currently only available on the `amd64` architecture.

You can use the environment variable `MQ_ARCHIVE_DEV` to specify an alternative local file to install from (which must be in the `downloads` directory).

## Installed components

This image includes the core MQ server, Java, language packs, GSKit, and web server.  This is configured in the `Generate MQ package in INSTALLATION_DIR` section [here](../install-mq.sh), with the configured options being picked up at build time.
