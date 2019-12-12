# IBM MQ Internet Pass-Thru on Docker

IBM® MQ Internet Pass-Thru (MQIPT) is an optional component of IBM MQ. MQIPT runs as a stand-alone service that can receive and forward IBM MQ message flows, either between two IBM MQ queue managers or between an IBM MQ client and an IBM MQ queue manager.
MQIPT enables this connection when the client and server are not on the same physical network.

This repository contains all of the resources you will need to create a Docker image containing MQIPT for use in your infrastructure.

## How to build this image

1. Download MQIPT from [IBM Fix Central for IBM MQ](https://ibm.biz/mq91ipt).
2. Ensure the MQIPT downloaded tar file is available in this directory.
3. If the tar file is not called `9.1.4.0-IBM-MQIPT-LinuxX64.tar.gz` you will need to either:
    * Rename the file.
    * Supply the new name as a `--build-arg IPTFILE=<new name>` when executing the docker build command in the next step.
    * Alter the Dockerfile `IPTFILE` ARG to specify the new file name.
4. Run the following command in this directory to build the Docker image. `docker build -t mqipt .`

Once the Docker build has completed you will have a new Docker image called `mqipt:latest` available which contains MQIPT.

## How to run this image

Before you run the MQIPT docker image you should understand how MQIPT operates. Please ensure you have read the MQIPT documentation in the [IBM MQ Knowledge Center](https://www.ibm.com/support/knowledgecenter/en/SSFKSJ_9.1.0/com.ibm.mq.pro.doc/ipt0000_.htm).

1. Create a MQIPT home directory that can be [mounted to a docker container](https://docs.docker.com/storage/). The MQIPT home directory contains configuration files and log files produced when MQIPT runs.
2. Create your [MQIPT configuration file](https://www.ibm.com/support/knowledgecenter/en/SSFKSJ_9.1.0/com.ibm.mq.ref.con.doc/ipt2540_.htm) in the MQIPT home directory. This file **must** be called `mqipt.conf`. A sample configuration file is supplied with MQIPT in `samples/mqiptSample.conf`.
3. Run the following command to start a container with your built MQIPT image:
    `docker run -d --volume <mqiptHome>:/var/mqipt -p <hostPort>:<containerPort> mqipt`
   where `mqiptHome` is the MQIPT home directory you created in step 1, and `containerPort` is a port to be exposed that MQIPT is listening on, such as a route port.
   
   If you want the container ports to be accessible outside of the host you must expose the required ports, this will map the container port to a port on the host, meaning you can connect to that port on the host and access MQIPT. You will need to provide multiple `-p` parameters to expose all of the ports required by your MQIPT configuration. **Note:** these must be available otherwise the Docker container will fail to start.
   
   See [Docker Run reference](https://docs.docker.com/engine/reference/run/#expose-incoming-ports) for more information on how to expose container ports.

## Further information

For further information on MQIPT please view the MQIPT documentation in the [IBM MQ Knowledge Center](https://www.ibm.com/support/knowledgecenter/en/SSFKSJ_9.1.0/com.ibm.mq.pro.doc/ipt0000_.htm).

## License

The Dockerfiles and associated code and scripts are provided as-is and licensed under the [Apache License 2.0](http://www.apache.org/licenses/LICENSE-2.0.html).

## Copyright

© Copyright IBM Corporation 2018,2019
