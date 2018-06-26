This is a work-in-progress for a Docker image based on Red Hat Enterprise Linux (RHEL).

The current MQ container build requires Docker V17.05 or greater (required features include multi-stage Docker build, and "ARG"s in the "FROM" statement).  Red Hat Enterprise Linux V7.5 includes Docker up to version V1.13.

In order to build images with Red Hat Enterprise Linux, license registration is required.  The license of the host server can be used, as long as you either use Red Hat's patched version of Docker (which is an old version), or if you use alternative container management tools such as [`buildah`](https://github.com/projectatomic/buildah/) and `podman` (from [`libpod`](https://github.com/projectatomic/libpod)).

This directory contains scripts for building with `buildah`.  The build itself isn't containerized, so more software than usual is needed on the RHEL host, so an Ansible playbook is also provided to help set up the host.