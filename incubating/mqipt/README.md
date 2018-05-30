# IBM MQ Internet Pass-Thru (SupportPac MS81) on Docker

IBM® MQ Internet Pass-Thru (MQIPT) is an extension to the base IBM MQ product. MQIPT runs as a stand-alone service that can receive and forward IBM MQ message flows, either between two IBM MQ queue managers or between an IBM MQ client and an IBM MQ queue manager.
MQIPT enables this connection when the client and server are not on the same physical network.

This repository contains all of the resources you will need to create a Docker image containing MQIPT for use in your infrastructure.

## How to build this image

1. First download MQIPT from the [IBM MQ SupportPacs website](http://www-01.ibm.com/support/docview.wss?rs=171&uid=swg27007198#3). MQIPT is a `Category 3 - Product Extensions SupportPacs` with the identifier `MS81`.
2. Ensure the MQIPT downloaded tar file is available in this directory.
3. If the tar file is not called `ms81_2.1.0.4_amd64_linux_2.tar` you will need to either:
    * Rename the file.
    * Supply the new name as a `--build-arg IPTFILE=<new name>` when executing the docker build command in the next step.
    * Alter the Dockerfile `IPTFILE` ARG to specify the new file name.
4. Run the following command in this directory to build the Docker image. `docker build -t mqipt .`

Once the Docker build has completed you will have a new Docker image called `mqipt:latest` available which contains MQIPT.

## How to run this image

Before you run the MQIPT docker image you should understand how MQIPT operates, please ensure you have read the [MQIPT knowledgecenter](https://www.ibm.com/support/knowledgecenter/en/SSFKSJ_9.0.0/com.ibm.mq.ipt.doc/ipt0000_.htm) and any documentation supplied with the MQIPT installation tar.

First you need to create your [MQIPT configuration file](https://www.ibm.com/support/knowledgecenter/en/SSFKSJ_9.0.0/com.ibm.mq.ipt.doc/ipt2540_.htm) and place this file in a directory that can be [mounted to a docker container](https://docs.docker.com/storage/). This file **must** be called `mqipt.conf`; there is a sample file available in the MQIPT installation tar.

Run the following command to start a container with your built MQIPT image:

`docker run -d --volume <path to config>:/var/mqipt -p 1414:1414 mqipt`

If you want the container ports to be accessible outside of the host you must expose the required ports, this will map the container port to a port on the host, meaning you can connect to that port on the host and access MQIPT. You will need to provide multiple `-p` parameters to expose all of the ports required by your MQIPT configuration. **Note:** these must be available otherwise the docker container will fail to start.

See [Docker Run reference](https://docs.docker.com/engine/reference/run/#expose-incoming-ports) for more information on how to expose container ports.

## Further information

For further information on MQIPT please view the [MQIPT knowledgecenter](https://www.ibm.com/support/knowledgecenter/en/SSFKSJ_9.0.0/com.ibm.mq.ipt.doc/ipt0000_.htm)

## License

The Dockerfiles and associated code and scripts are provided as-is and licensed under the [Apache License 2.0](http://www.apache.org/licenses/LICENSE-2.0.html).

## Copyright

© Copyright IBM Corporation 2018
