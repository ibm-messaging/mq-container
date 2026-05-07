# Using a different base image

The `Dockerfile-server` has an argument for `BASE_VARIANT`.  The only maintained base
variant is `ubi`.  If you want to use a different base image, then you can add
additional targets into the `Dockerfile-server` file, which can create new variants.  This technique is useful to allow multiple different base variants to co-exist.

Note that the following are examples only, and are not fully maintained and tested
with the latest versions of base images or package versions.

## Ubuntu

The following example creates an `ubuntu` base variant, with both a `builder-ubuntu` image and a new base image for the queue manager container.

```dockerfile
# Base image to use when BASE_VARIANT=ubuntu
ARG UBUNTU_BASE_IMAGE=public.ecr.aws/ubuntu/ubuntu
# Which version of Ubuntu to use when BASE_VARIANT=ubuntu
# This is not as precisely pinned as the UBI variant.
ARG UBUNTU_BASE_TAG=24.04
# Image for the Go and C builds, as well as the MQ and UBI packaging build stages to use
ARG UBUNTU_BUILDER_IMAGE=public.ecr.aws/ubuntu/ubuntu
# Newer version of Ubuntu needed here, to get a more recent Go compiler
ARG UBUNTU_BUILDER_TAG=25.10

FROM $UBUNTU_BUILDER_IMAGE:$UBUNTU_BUILDER_TAG AS builder-ubuntu
ARG GO_WORKDIR
WORKDIR ${GO_WORKDIR}
ENV GOPATH=${GO_WORKDIR}
RUN apt-get update -y \
  && apt-get install -y --no-install-recommends \
    build-essential \
    ca-certificates \
    gcc \
    golang-1.25 \
    make \
  && rm -rf /var/lib/apt/lists/*
ENV PATH=$PATH:/usr/lib/go-1.25/bin/

FROM $UBUNTU_BASE_IMAGE:$UBUNTU_BASE_TAG AS base-plus-packages-ubuntu
ARG BASE_DIR
ARG PACKAGES_TO_INSTALL="bash bc ca-certificates coreutils curl debianutils findutils gawk grep language-pack-en libc-bin mount openssl procps sed tar util-linux libicu-dev"
RUN apt-get update -y \
  && apt-get install \
    -y \
    --no-install-recommends \
    ${PACKAGES_TO_INSTALL} \
  && rm -rf /var/lib/apt/lists/*
```

## SLES

The following example creates a `sles` base variant, with both a `builder-sles` image and a new base image for the queue manager container.

```dockerfile
# Base image to use when BASE_VARIANT=sles
ARG SLES_BASE_IMAGE=registry.suse.com/suse/sle15
# Which version of SLES to use when BASE_VARIANT=sles
# This is not as precisely pinned as the UBI variant.
# TODO: SLES 15.4 is not compatible with Go EXEs produced by the latest UBI Go compiler (glibc version)
ARG SLES_BASE_TAG=15.7
# Image for the Go and C builds, as well as the MQ and UBI packaging build stages to use
ARG SLES_BUILDER_IMAGE=registry.suse.com/bci/golang
ARG SLES_BUILDER_TAG=1.25

FROM $SLES_BUILDER_IMAGE:$SLES_BUILDER_TAG AS builder-sles
ARG GO_WORKDIR
ENV GOPATH=${GO_WORKDIR}
WORKDIR ${GO_WORKDIR}

FROM $SLES_BASE_IMAGE:$SLES_BASE_TAG AS base-plus-packages-sles
ARG BASE_DIR
ARG PACKAGES_TO_INSTALL="bash bc ca-certificates findutils gawk glibc grep openssl procps sed tar util-linux which libicu"
RUN zypper install \
    --no-confirm \
    --no-recommends \
    ${PACKAGES_TO_INSTALL} \
  && zypper clean -a \
  && rm -rf /var/cache/zypp/*
```