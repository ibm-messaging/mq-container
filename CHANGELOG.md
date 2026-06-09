# Change log

## 10.0.0.0 (2026-06)

* Updated to MQ version 10.0.0.0
* Fixed upgrade issue [DT472893](https://www.ibm.com/mysupport/s/defect/aCIgJ000000EjDN/dt472893) by refreshing the WebSphere Liberty 'ltpa.keys' file. To prevent your existing 'ltpa.keys' file being deleted (on upgrade to this MQ version) you must set environment variable `AMQ_ENABLE_DT472893_REGENERATE_LTPA_KEYS=false`
* Added option to include version 3.8.0 (2026.1.3) of the IBM MQ tracing user exit. See [IBM MQ tracing](https://www.ibm.com/docs/en/SSE1JP5_current/src/pages/ecosystem/ibmmq/mq_tracing.html) in the IBM Instana documentation.
* Uses `ubi-micro` image instead of `ubi-minimal`.  This requires additional build stages.  Removes the `microdnf` and `rpm` commands, along with a number of other unnecessary programs.  This helps make the image smaller, and reduce the number of security vulnerabilities.
* Use Go cryptography provider for FIPS, instead of OpenSSL.  This applies only to the metrics HTTPS server.
* Updated checks to ensure FIPS is only enabled on supported platforms (amd64, ppc64le).
   * **Note** FIPS support is no longer available for s390x. If you require FIPS support on s390x use a previous version of IBM MQ.
* Uses `C.utf8` as the default locale instead of `C`.  This will cause _new_ queue managers to use CCSID 1208 (UTF-8), instead of 819 (ISO 8859-1).
* Adds language packs for other languages supported by MQ.  For example, setting the environment variable `LANG=de_DE.utf8` will cause the container to use the German language.
* Generate initial key for Liberty AES encryption. An initial key can also be passed in using a Secret.
* Added probe logging to capture Kubernetes liveness and startup probe execution events, with summary reporting on container shutdown.
* Updated MQ Advanced for Developers image to prioritize secrets over env vars for **app** and **admin** user credentials.
* Updated the minimum TLS version for the queue manager metrics server to 1.3.
* Added MQ_METRICS_REQUIRE_QUANTUM_SAFE environment variable.  If set to true, this configures the metrics server to only accept quantum safe TLS connections.

## 9.4.5.0-r2 (2026-03)
* New environment variable: MQ_ENABLE_SOFT_FILE_LIMIT_INCREASE
  * Setting the value to `true` raises the soft limit for the number of open files up to the hard limit prior to starting MQ.

## 9.4.5.0 (2026-02)

* Updated to MQ version 9.4.5.0
* New environment variable: MQ_ENABLE_CLEAN_TMP_ON_START
  * Setting the value to `true` will delete the contents of `/tmp` on container startup

## 9.4.4.0 (2025-10)

* Updated to MQ version 9.4.4.0
* Logs now consistently routed to stdout/stderr; logs to stdout will conform to the selected log format (MQ_LOGGING_CONSOLE_FORMAT), logs to stderr are unformatted errors
* Updated to GSKit9, which now requires the `libicu` library; this is included in the MQ container image but must be added when building your own.

## 9.4.3.0 (2025-06)

* Updated to MQ version 9.4.3.0
* Refactored the file permission logs to match the container logging format
* Allow enabling audit logging of access to Prometheus metrics endpoint (enable audit logging by setting `MQ_LOGGING_METRICS_AUDIT_ENABLED=true`)

## 9.4.2.0 (2025-02)

* Updated to MQ version 9.4.2.0
* Queue Manager metrics can be served via HTTPS by populating `/etc/mqm/metrics/pki/keys` with TLS keys.
* Queue Manager web console can be restarted using the 'strmqweb' command.

## 9.4.1.0 (2024-10)

* Updated to MQ version 9.4.1.0
* Fix to diable FIPS mode for `runmqakm` key store generation, when FIPS is not enabled
* Fix APAR IT46430
* Changed build Dockerfile to reduce file duplication across image layers
* Changed shutdown flow to continue reaping orphan processes during queue manager shutdown
* Allow Native HA configuration to be externally provided rather than generated from template.
  * Deprecate use of environment variable configuration of Native HA (except `MQ_NATIVE_HA=true` which is still required).
  * Clarify behaviour of now deprecated environment variable configuration in IBM documentation
* Clarified new minimum versions of Docker and Podman; new version required due to the move to UBI 9

## 9.4.0.0 (2024-06)

* Updated to MQ version 9.4.0.0
* Based on [Red Hat Universal Base Image 9.4-949.1716471857](https://catalog.redhat.com/software/containers/ubi9/ubi-minimal/615bd9b4075b022acc111bf5?image=664f4c2d9cbb931e839f138b&architecture=amd64).
   * **Note** UBI 9 has pending FIPS 140-3 certification.  UBI 9 is not supported on the POWER 8 architecture.
* Added new optional value "mqsc" for the environment variable MQ_LOGGING_CONSOLE_SOURCE. This will reflect the contents of autocfgmqsc.LOG.
* Environment variables **MQ_ADMIN_PASSWORD** and **MQ_APP_PASSWORD** are deprecated for the MQ Advanced for Developers image. Secrets should be used to set the passwords for **app** and **admin** users.
* MQ Advanced for Developers image will no longer use mq.htpasswd file for the MQ Authorization Service. Secrets will be used to authorize the **app** and **admin** users.
* `chkmqstarted` command updated for Native-HA deployments to additionally check if the queue manager instance is in-sync with one or more replicas.
* New Model Queue `DEV.APP.MODEL.QUEUE` defined for MQ Advanced for Developers image with `BROWSE, DISPLAY, GET, INQUIRE, PUT` Authority for user `app`.

## 9.3.5.0 (2024-02)

* Updated to MQ version 9.3.5.0

### Security Fixes
* Fixed a security issue, where unencrypted credentials in mqwebuser.xml would be copied to /var/mqm.  A symbolic link is now used instead.
* golang.org/x/crypto library has been upgraded to remediate CVE-2023-48795 vulnerability.
* More secure sha512 algorithm will be used instead of sha256 to create self signed Certificate in the Web keystore.
* The MQ container generates a PKCS#12 key store for use with the MQ web server.This keystore is generated using a legacy SHA-1 encryption,container code has been updated to use Pkcs12.Modern.Encode function which uses SHA-2 encryption.
* Vulnerability has been reported on PathTraversal method usages which now have been fixed.

## 9.3.4.0 (2023-12)

* Updated to MQ version 9.3.4.0
* Fixed the signal handler so that it correctly processes control signals if a termination signal is received before startup is complete
* The default value for the environment variable MQ_LOGGING_CONSOLE_SOURCE is now "qmgr,web" instead of "qmgr".
* Removed MQ Explorer and MQ SDK samples from the "incubating" folder.  IBM MQ Explorer was removed from the IBM MQ install package in 9.3.0.

## 9.3.3.2-r1 (2023-10)

* Updated to MQ version 9.3.3.0

## 9.3.3.1-r1 (2023-08)

* Updated to MQ version 9.3.3.0

## 9.3.3.0-r2 (2023-07)

* Updated to MQ version 9.3.3.0

## 9.3.3.0 (2023-06)

* Updated to MQ version 9.3.3.0

## 9.3.2.0 (2023-02)

* Updated to MQ version 9.3.2.0
* Queue manager certificates with the same Subject Distinguished Name (DN) as the issuer (CA) certificate are not supported. A certificate must have a unique Subject Distinguished Name.
* New logging environment variables: MQ_LOGGING_CONSOLE_SOURCE, MQ_LOGGING_CONSOLE_FORMAT, MQ_LOGGING_CONSOLE_EXCLUDE_ID.  The LOG_FORMAT variable is deprecated.
* New environment variable: MQ_QMGR_LOG_FILE_PAGES

## 9.3.1.0-r2 (2022-11)

* Queue manager attribute SSLKEYR is now set to blank instead of '/run/runmqserver/tls/key' if key and certificate are not supplied.

## 9.3.1.0 (2022-10)

* Updated to MQ version 9.3.1.0

## 9.3.0.0 (2022-06)

* Updated to MQ version 9.3.0.0
* Use `registry.access.redhat.com` instead of `registry.redhat.io`, so that you don't need to login with a Red Hat account.
* Updated default developer config to use TLS cipher `ANY_TLS12_OR_HIGHER` instead of `ANY_TLS12`
* Added default `jvm.options` file fix issue with missing preferences file causing an error in the web server log.
* Updated to allow building image from Podman on macOS (requires Podman 4.1)
* Container builds are now faster
* Updated signal handling to use a buffer, as recommended by the Go 1.17 vetting tool

## 9.2.5.0 (2022-03)

* Updated to MQ version 9.2.5.0

## 9.2.4.0 (2021-11)

* Updated to MQ version 9.2.4.0

## 9.2.3.0 (2021-07-22)

* Updated to MQ version 9.2.3.0

## 9.2.2.0 (2021-03-26)

* Updated to MQ version 9.2.2.0

## 9.2.1.0 (2020-02-18)

* Updated to MQ version 9.2.1.0


## 9.2.0.1-LTS (2020-12-04)

* Added support for MQ Long Term Support (production licensed only) in the mq-container

## 9.2.0.0 (2020-07-23)

* Updated to [MQ version 9.2.0.0](https://www.ibm.com/support/knowledgecenter/SSFKSJ_9.2.0/com.ibm.mq.pro.doc/q113110_.htm)
* Use `-ic` arguments with `crtmqm` to process MQSC files in `/etc/mqm`. Replaces previous use of "runmqsc" commands

## 9.1.5.0 (2020-04-02)

* Updated to MQ version 9.1.5.0
* Can now run as a random user, instead of the "mqm" user, which has now been removed. This adds compatability for the [Red Hat OpenShift restricted SCC](https://docs.openshift.com/container-platform/4.3/authentication/managing-security-context-constraints.html#security-context-constraints-about_configuring-internal-oauth). The default image UID is `1001`.

## 9.1.4.0 (2019-12-06)

* Updated to MQ version 9.1.4.0
* Updated to use UBI8 as base image
* Added required security settings to self signed certificates to align with macOS Catalina requirements

## 9.1.3.0 (2019-07-19)

* Updated to MQ version 9.1.3.0
* Allow generation of TLS certificate with given hostname
* Fixes for the following issues:
  * `MQ_EPHEMERAL_PREFIX` UNIX sockets fix
  * Fix Makefile for Windows
  * Use -a option on crtmqdir
  * Remove check for certificate environment variable

## 9.1.2.0-UBI (2019-06-21)

**Breaking changes**:
* UID of the mqm user is now 888.  You need to run the container with an entrypoint of `runmqserver -i` under the root user to update any existing files.
* MQSC files supplied will be verified before being run. Files containing invalid MQSC will cause the container to fail to start

**Other changes**:
* Security fixes
* Web console added to production image
* Container built on RedHat host

## 9.1.2.0 (2019-03-21)

* Updated to MQ version 9.1.2.0
* Now runs using the "mqm" user instead of root.  See new [security doc](https://github.com/ibm-messaging/mq-container/blob/master/docs/security.md)
* New [IGNSTATE](https://www.ibm.com/support/knowledgecenter/en/SSFKSJ_9.1.0/com.ibm.mq.pro.doc/q132310_.htm#q132310___ignstateparm) parameter used in default developer config
* Termination log moved from `/dev/termination-log` to `/run/termination-log`, to make permissions easier to handle
* Fixes for the following issues:
    * Brackets no longer appear in termination log
    * Test timeouts weren't being used correctly
    * Building on subscribed and unsubscribed hosts ([#273](https://github.com/ibm-messaging/mq-container/pull/273))
    * Gosec failures ([#286](https://github.com/ibm-messaging/mq-container/pull/286))
    * Security fix for perl-base ([#253](https://github.com/ibm-messaging/mq-container/pull/253))

## 9.1.1.0 (2018-11-30)

* Updated to MQ version 9.1.1.0
* Created seperate RedHat Makefile for building images on RedHat machines with buildah
* Enabled REST messaging capability for app user.
* Added support for container supplementary groups
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
