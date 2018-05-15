# IBM MQ Software Developer Kit (SDK) with Go

This image contains the MQ SDK, Git, the Go compiler, and the `build-essential` package (which includes GNU C and C++ compilers plus other essential tools like `make`).

This image doesn't contain any Go code for MQ.  You can add a CGO wrapper for the MQ C client, for example [mq-golang](https://github.com/ibm-messaging/mq-golang), via your vendor directory, or directly using `go get`.