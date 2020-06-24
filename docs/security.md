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
  ibm-mqadvanced-server:9.2.0.0-amd64
```

The MQ Advanced for Developers image does require the "chown", "setuid", "setgid" and "audit_write" capabilities (plus "dac_override" if you're using an image based on Red Hat Enterprise Linux).  This is because it uses the "sudo" command to change passwords inside the container.  For example, in Docker, you could do the following:

```sh
docker run \
  --cap-drop=ALL \
  --cap-add=CHOWN \
  --cap-add=SETUID \
  --cap-add=SETGID \
  --cap-add=AUDIT_WRITE \
  --env LICENSE=accept \
  --env MQ_QMGR_NAME=QM1 \
  --detach \
  ibm-mqadvanced-server-dev:9.2.0.0-amd64
```
