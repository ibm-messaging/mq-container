# Testing

## Prerequisites
You need to ensure you have the following tools installed:
* [Docker](https://www.docker.com/)
* [GNU make](https://www.gnu.org/software/make/)
* [Go](https://golang.org/) - only needed for running the tests
* [dep](https://github.com/golang/dep) (official Go dependency management tool) - needed to prepare for running the tests

## Running the tests
There are two main sets of tests:

1. Unit tests, which are run during a build
2. Docker tests, which test a complete Docker image, using the Docker API

### Running the Docker tests
The Docker tests can be run locally on a machine with Docker.  For example:

```
make test-devserver
make test-advancedserver
```

You can specify the image to use directly by using the `MQ_IMAGE_ADVANCEDSERVER` or `MQ_IMAGE_DEVSERVER` variables, for example:

```
MQ_IMAGE_ADVANCEDSERVER=ibm-mqadvanced-server:9.1.4.0-amd64 make test-advancedserver
```

You can pass parameters to `go test` with an environment variable.  For example, to run the "TestGoldenPath" test, run the following command::

```
TEST_OPTS_DOCKER="-run TestGoldenPath" make test-advancedserver
```

You can also use the same environment variables you specified when [building](./building), for example, the following will try and test an image called `ibm-mqadvanced-server:9.1.4.0-amd64`:

```
MQ_VERSION=9.1.4.0 make test-advancedserver
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
MQ_IMAGE=mycluster.icp:8500/default/mq-devserver make test-kubernetes-devserver
```
