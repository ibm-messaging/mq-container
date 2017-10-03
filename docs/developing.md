# Developing

## Prerequisites
You need to ensure you have the following tools installed:

* [Docker](https://www.docker.com/)
* [Go](https://golang.org/)
* [Glide](https://glide.sh/)
* [dep](https://github.com/golang/dep) (official Go dependency management tool)
* make

For running the Kubernetes tests, a Kubernetes environment is needed, for example [Minikube](https://github.com/kubernetes/minikube) or [IBM Cloud Private](https://www.ibm.com/cloud-computing/products/ibm-cloud-private/).

## Running the tests
There are three main sets of tests:

1. Unit tests
2. Docker tests, which test a complete Docker image, using the Docker API
3. Kubernetes tests, which test the Helm charts (and the Docker image) via [Helm](https://helm.sh)

### Running the tests
The unit and Docker tests can be run locally.  For example:

```bash
make test-devserver
```

### Running the Kubernetes tests

For the Kubernetes tests, you need to have built the Docker image, and pushed it to the registry used by your Kubernetes cluster.  Most of the configuration used by the tests is picked up from your `kubectl` configuration, but you will typically need to specify the image details.  For example:

```bash
DOCKER_REPO_DEVSERVER=mycluster.icp:8500/default/mq-devserver make test-kubernetes-devserver
```
