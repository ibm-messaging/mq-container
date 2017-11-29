# Developing

## Prerequisites
You need to ensure you have the following tools installed:

* [Docker](https://www.docker.com/)
* [Go](https://golang.org/) - only needed for running the tests
* [Glide](https://glide.sh/)
* [dep](https://github.com/golang/dep) (official Go dependency management tool)
* make
* [Helm](https://helm.sh) - only needed for running the Kubernetes tests

For running the Kubernetes tests, a Kubernetes environment is needed, for example [Minikube](https://github.com/kubernetes/minikube) or [IBM Cloud Private](https://www.ibm.com/cloud-computing/products/ibm-cloud-private/).

## Building a production image
This procedure works for building the MQ Continuous Delivery release, on `x86_64`, `ppc64le` and `s390x` architectures.

1. Download MQ from IBM Passport Advantage, and place the downloaded file (for example, `CNLE4ML.tar.gz` for MQ V9.0.4 on x86_64 architecture) in the `downloads` directory
2. Run `make build-advancedserver`

You can build a different version of MQ by setting the `MQ_VERSION` environment variable, for example:

```bash
MQ_VERSION=9.0.3.0 make build-advancedserver
```

## Running the tests
There are three main sets of tests:

1. Unit tests, which are run during a build
2. Docker tests, which test a complete Docker image, using the Docker API
3. Kubernetes tests, which test the Helm charts (and the Docker image) via [Helm](https://helm.sh)

### Running the Docker tests
The Docker tests can be run locally.  Before you run them for the first time, you need to download the test dependencies:

```
make deps
```

You can then run the tests, for example:

```
make test-devserver
```

or:

```
make test-advancedserver
```

You can pass parameters to `go test` with an environment variable.  For example, to run the "TestGoldenPath" test, run the following command::

```
TEST_OPTS_DOCKER="-run TestGoldenPath" make test-advancedserver
```

### Running the Docker tests with code coverage
You can produce code coverage results from the Docker tests by running the following:

```
make build-advancedserver-cover
make test-advancedserver-cover
```

In order to generate code coverage metrics from the Docker tests, the build step creates a new Docker image with an instrumented version of the code.  Each test is then run individually, producing a coverage report each under `test/docker/coverage/`.  These individual reports are then combined.  The combined report is written to the `coverage` directory. 


### Running the Kubernetes tests

For the Kubernetes tests, you need to have built the Docker image, and pushed it to the registry used by your Kubernetes cluster.  Most of the configuration used by the tests is picked up from your `kubectl` configuration, but you will typically need to specify the image details.  For example:

```bash
DOCKER_REPO_DEVSERVER=mycluster.icp:8500/default/mq-devserver make test-kubernetes-devserver
```
