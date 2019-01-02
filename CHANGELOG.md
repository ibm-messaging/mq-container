# Change log

## 9.1.1.0 (2018-11-30)

* Updated to MQ version 9.1.1.0
* Created seperate RedHat Makefile for building images on RedHat machines with buildah
* Enabled REST messaging capability for app user.
* Added support for container suplimentary groups
* Removed IBM MQ version 9.0.5 details.
* Added additional Diagnostics ([#203](https://github.com/ibm-messaging/mq-container/pull/203))
* Implementted GOSec to perform code scans for security vulnerabilities. (([#227](https://github.com/ibm-messaging/mq-container/pull/227)))
* Removed Queue manager create option from the MQ Console.
* Fixes for the following issues:
    * Check explicitly for `/mnt/mqm` ([#175](https://github.com/ibm-messaging/mq-container/pull/175))
    * Force string output in chkmqhealthy ([#174](https://github.com/ibm-messaging/mq-container/pull/174))
    * Use -aG not -G when adding a group for a user
    * Security fixes for libsystemd0 systemd systemd-sysv & libudev1

## 9.1.0.0 (2018-07-23)

* Updated to MQ version 9.1.0.0
* Added Docker 1.12 tests
* Added MQ SDK Docker image sample
* Added MQ Golang SDK Docker image sample
* Added Prometheus metric gathering implementation
* Added MQ Internet Pass-Thru (MS81) Docker image sample
* Added POWER & z/Linux image builds
* `devjmstest` image now built with Maven instead of gradle
* Added FAT manifests for Docker Hub/Docker Store
* Added Red Hat Enterprise Linux image build
* Added basic versioning debug information into golang programs
* Removed 9.0.4

## 9.0.5.0 (2018-03-13)

* Updated to MQ version 9.0.5.0
* Container's stdout can now be set to JSON format (set LOG_FORMAT=json)
* MQ error logs (in JSON or plain text) are now mirrored on stdout for the container.
* `chkmqready` now waits until MQSC scripts in `/etc/mqm` have been applied
* `chkmqready` and `chkmqhealthy` now run as the "mqm" user
* Added ability to optionally use an alternative base image
* Various build and test improvements
* Removed 9.0.3

## 9.0.4 (2017-11-06)

* Updated to MQ version 9.0.4.0
* Updated to Go version 9
* Removed packages `curl`, `ca-certificates`, and their dependencies, which were only used at build time
* Improved logging
* Helm charts now work on Kubernetes V1.6
* Production Helm chart now includes a default image repository and tag
* Updated to use multi-stage Docker build, so that Go code is built inside a container

## 9.0.3 (2017-10-17)

* Initial version
