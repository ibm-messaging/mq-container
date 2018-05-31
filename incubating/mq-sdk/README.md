# IBM MQ Software Developer Kit (SDK)

This image contains the MQ SDK and the `build-essential` package, which includes GNU C and C++ compilers plus other essential tools.

## Usage

For example, you could compile the `amqsput0.c` sample program by running the following command in an SDK container:

```sh
gcc -o /tmp/amqsput0 /opt/mqm/samp/amqsput0.c -I /opt/mqm/inc -L /opt/mqm/lib64 -lmqm
```

Compiler and linker output is placed on the container's filesystem, so using multi-stage Docker builds is useful to build a final container.
