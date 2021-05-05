# Security

## Container runtime

### User

The MQ server image is run using with UID 1001, though this can be any UID, with a fixed GID of 0 (root).

### Capabilities

The MQ Advanced image requires no Linux capabilities, so you can drop any capabilities which are added by default.  For example, in Docker you could do the following:

```sh
docker run \
  --cap-drop=ALL \
  --env LICENSE=accept \
  --env MQ_QMGR_NAME=QM1 \
  --detach \
  ibm-mqadvanced-server:9.2.3.0-amd64
```
