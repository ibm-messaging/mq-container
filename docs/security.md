# Security

## Container runtime

### User

The MQ server image is run using the "mqm" user.  On the Ubuntu-based image, this uses the UID and GID of 999.  On the Red Hat Enterprise Linux image, it uses the UID and GID of 888.

### Capabilities

The MQ Advanced image requires no Linux capabilities, so you can drop any capabilities which are added by default.  For example, in Docker you could do the following:

```sh
docker run \
  --cap-drop=ALL \
  --env LICENSE=accept \
  --env MQ_QMGR_NAME=QM1 \
  --detach \
  mqadvanced-server:9.1.1.0-x86_64-ubuntu-16.04
```

The MQ Advanced for Developers image does requires the "chown", "setuid", "setgid" and "audit_write" capabilities (plus "dac_override" if you're using an image based on Red Hat Enterprise Linux).  This is because it uses the "sudo" command to change passwords inside the container.  For example, in Docker, you could do the following:

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
  mqadvanced-server-dev:9.1.1.0-x86_64-ubuntu-16.04
```

### SELinux

The SELinux label "spc_t" (super-privileged container) is needed to run the MQ container on a host with SELinux enabled.  This is due to a current limitation in how MQ data is stored on volumes, which violates the usual policy applied when using the standard "container_t" label.