# IBM MQ Internet Pass-Thru in a container

IBM® MQ Internet Pass-Thru (MQIPT) is an optional component of IBM MQ. MQIPT runs as a stand-alone service that can receive and forward IBM MQ message flows, either between two IBM MQ queue managers, or between an IBM MQ client and an IBM MQ queue manager.
MQIPT enables this connection when the client and server are not on the same physical network.

This repository contains all the resources that you will need to create a container image that contains MQIPT.

## How to build this image

1. Download MQIPT for Linux x86_64 from [Fix Central](https://ibm.biz/mq93ipt). The name of the download file is similar to `9.3.x.x-IBM-MQIPT-LinuxX64.tar.gz`.
2. Ensure the MQIPT downloaded tar file is available in this directory.
3. Run the following command in this directory to build the container image:
   `docker build --build-arg MQIPT_ARCHIVE=<tar_file_name> -t mqipt .`

Once the build has completed you will have a new container image called `mqipt:latest` which contains MQIPT.

## How to run this image

Before you run the MQIPT container image you should understand how MQIPT operates. You can read about MQIPT in the [IBM MQ documentation](https://www.ibm.com/docs/en/ibm-mq/9.3?topic=overview-mq-internet-pass-thru).

1. Create a MQIPT home directory that can be [mounted to a container](https://docs.docker.com/storage/). The MQIPT home directory contains configuration files and log files that are produced when MQIPT runs.
2. Create your [MQIPT configuration file](https://www.ibm.com/docs/en/ibm-mq/9.3?topic=reference-mq-internet-pass-thru-configuration) in the MQIPT home directory. This file **must** be called `mqipt.conf`. A sample configuration file is supplied with MQIPT in `samples/mqiptSample.conf`.
3. Run the following command to start a container with your built MQIPT image:
    `docker run -d --volume <mqiptHome>:/var/mqipt -p <hostPort>:<containerPort> mqipt`
   where `mqiptHome` is the MQIPT home directory you created in step 1, and `containerPort` is a port to be exposed that MQIPT is listening on, such as a route port.

  If you want the container ports to be accessible outside of the host you must expose the required ports. This maps the container port to a port on the host, so that you can connect to the port on the host and access MQIPT. You might need to provide more than one `-p` parameters to expose all the ports that are required by your MQIPT configuration. **Note:** These ports must be available on the host. If the ports are not available, the Docker container will not start.

  For more information about how to expose container ports, see [Docker Run reference](https://docs.docker.com/engine/reference/run/#expose-incoming-ports).

## Further information

For more information about MQIPT, see MQIPT documentation in the [IBM MQ documentation](https://www.ibm.com/docs/en/ibm-mq/9.3?topic=overview-mq-internet-pass-thru).

## License

The Dockerfile and associated code and scripts are provided as-is and licensed under the [Apache License 2.0](http://www.apache.org/licenses/LICENSE-2.0.html).

## Copyright

© Copyright IBM Corporation 2018, 2023