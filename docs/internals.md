# Internals

This page documents internal code details and design decisions.

The resulting Docker image contains the following:

* Base linux distribution - this provides standard Linux libraries (such as "glibc") and utilities (such as "ls" and "grep") required by MQ
* MQ installation (under `/opt/mqm`)
* Three additional programs, to enable running in a containerized environment:
   - `runmqserver` - The main process, which creates and runs a queue manager
   - `runmqdevserver` - The main process for MQ Advanced for Developers
   - `chkmqhealthy` - Checks the health of the queue manager.  This can be used by (say) a Kubernetes liveness probe.
   - `chkmqready` - Checks if the queue manager is ready for work.  This can be used by (say) a Kubernetes readiness probe.
   - `chkmqstarted` - Checks if the queue manager has successfully started.  This can be used by (say) a Kubernetes startup probe.

## runmqserver
The `runmqserver` command has the following responsibilities:

* Checks license acceptance
* Sets up `/var/mqm`
    - MQ data directory needs to be set up at container creation time.  This is done using the `crtmqdir` utility, which was introduced in MQ V9.0.3
    - It assumes that a storage volume for data is mounted under `/mnt/mqm`.  It creates a sub-directory for the MQ data, so `/var/mqm` is a symlink which resolves to `/mnt/mqm/data`.  The reason for this is that it's not always possible to change the ownership of an NFS mount point directly (`/var/mqm` needs to be owned by "mqm"), but you can change the ownership of a sub-directory.
* Acts like a daemon
    - Handles UNIX signals, like SIGTERM
    - Works as PID 1, so is responsible for [reaping zombie processes](https://blog.phusion.nl/2015/01/20/docker-and-the-pid-1-zombie-reaping-problem/)
* Creating and starting a queue manager
* Configuring the queue manager, by running any MQSC scripts found under `/etc/mqm`
* Starts the MQ web server (if enabled)
* Starting Prometheus metrics generation for the queue manager (if enabled)
* Indicates to the `chkmqready` command that configuration is complete, and that normal readiness checking can happen.  This is done by writing a file into `/run/runmqserver`

In addition, for MQ Advanced for Developers only, the web server is started.

## runmqdevserver
The `runmqdevserver` command is added to the MQ Advanced for Developers image only.  It does the following, before invoking `runmqserver`:

1. Sets passwords based on supplied environment variables
2. Generates MQSC files to put in `/etc/mqm`, based on a template, which is updated with values based on supplied environment variables.
3. If requested, it creates TLS key stores under `/run/runmqserver`, and configures MQ and the web server to use them

## Prometheus metrics
[Prometheus](https://prometheus.io) metrics are generated for the queue manager as follows:

1. A connection is established with the queue manager
2. Metrics are discovered by subscribing to topics that provide meta-data on metric classes, types and elements
3. Subscriptions are then created for each topic that provides this metric data
4. Metrics are initialised using Prometheus names mapped from their element descriptions
5. The metrics are then registered with the Prometheus registry as Prometheus Gauges
6. Publications are processed on a periodic basis to retrieve the metric data
7. A web server is setup to listen for requests from Prometheus on `/metrics` port `9157`.
    - From 9.4.2.0 onwards, if TLS keys are provided in `/etc/mqm/metrics/pki/keys`, an HTTPS server will be started on this port. Files in this directory should be PEM encoded certificates:
        - `tls.crt` server public certificate
        - `tls.key` server private key
        - `ca.crt` CA public certificate (optional)
    - If no TLS keys are provided (or earlier versions of MQ are used), an HTTP server will be started
8. Prometheus requests are handled by updating the Prometheus Gauges with the latest metric data
9. These updated Prometheus Gauges are then collected by the Prometheus registry
