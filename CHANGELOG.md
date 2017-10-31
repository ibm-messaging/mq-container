# Change log

## 9.0.4 (2017-11-01)
* Updated to MQ version 9.0.4.0
* Updated to Go version 9
* Removed packages `curl`, `ca-certificates`, and their dependencies, which were only used at build time
* Improved logging
* Helm charts now work on Kubernetes V1.6
* Production Helm chart now includes a default image repository and tag
* Updated to use multi-stage Docker build, so that Go code is built inside a container

## 9.0.3 (2017-10-17)
* Initial version