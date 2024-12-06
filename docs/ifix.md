# IFix (interim-fix) for IBM MQ Software

An interim fix (or "ifix") is a patch or update that is released by IBM MQ support team to fix bugs, improve performance, or add new features to the MQ software. There may be scenarios when you need to apply these IBM MQ ifixes to a container that you've built yourself. This document describes how to apply an MQ software ifix during a container build.

The IBM MQ support have provided a generic set of steps in this page - https://www.ibm.com/support/pages/full-example-using-mqfixinstsh-install-mq-interim-fix-linux, for applying their ifixes on on-prem platforms like Linux. 

## Applying ifix via Dockerfile

There is a need to add commands for ifixes (patches or updates) of IBM MQ provided by support team to mq-container as well. This has to be done via the [Dockerfile](Dockerfile-Server) of the container. 

### Steps
To add commands for ifixes in the Dockerfile, you can follow these steps:

1. Identify the specific ifix that needs to be applied
2. Determine the installation instructions for the ifix provided by support
3. The exact location where the commands have to be added is marked by a comment in Dockerfile which can be identified by an eye-catcher string called **`IFIX`**
4. Build the container
5. After a successfuly build, `exec` into the running container and execute the command `dspmq` to know if the correct ifix version is reflected
6. If needed, test the container to ensure that the patch for which ifix has been provided is working as expected
7. It is important to note that the specific commands for adding ifixes may vary depending on the version of IBM MQ and the type of ifix being applied. Therefore, it is recommended to consult the IBM documentation for the specific version of IBM MQ and the type of ifix being applied.
