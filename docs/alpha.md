Alpha/experimental features
===========================
Use of the following environment variables is unsupported, and the features are subject to change without notice:

* **MQ_ALPHA_JSON_LOGS** - Set this to `true` or `1` to log in JSON format to stdout from the container.
* **MQ_ALPHA_MIRROR_ERROR_LOGS** - Set this to `true` or `1` to mirror the MQ error logs to the container's stdout

## Viewing the logs on the command line
The JSON-formatted logs are useful for forwarding to a log analysis system, but not very friendly when using `docker logs` or `kubectl logs`.  You can format them using a command line utility like `jq`, for example:

```
docker run -e LICENSE=accept --rm -i -e MQ_QMGR_NAME=qm1 -e MQ_ALPHA_JSON_LOGS=1 mqadvanced-server:9.0.4.0-x86_64-ubuntu-16.04 2>&1 | jq -r '.ibm_datetime + " " + .message'
```
