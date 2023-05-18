module github.com/ibm-messaging/mq-container/test/docker

go 1.19

require (
	// Note: This is not actually Docker v17.12!
	// Go modules require the use of semver, but Docker does not use semver and has not
	// [opted-in to use Go modules](https://github.com/golang/go/wiki/Modules#can-a-module-consume-a-package-that-has-not-opted-in-to-modules)
	// This means that when you `go get` Docker, you need to do so based on a commit,
	// e.g. `go get -v github.com/docker/docker@420b1d36250f9cfdc561f086f25a213ecb669b6f`,
	// which uses the commit for [Docker v19.03.15](https://github.com/moby/moby/releases/tag/v19.03.15)
	// Go will then find the latest tag with a semver-compatible tag.  In Docker's case,
	// v17.12.0 is valid semver, but v18.09 and v19.03 are not.
	// Also note: Docker v20.10 is valid semver, but the v20.10 client API requires use of Docker API
	// version 1.41 on the server, which is currently too new for the version of Docker in Travis (Ubuntu Bionic)
	github.com/docker/docker v17.12.0-ce-rc1.0.20210128214336-420b1d36250f+incompatible
	github.com/docker/go-connections v0.4.0
)

require (
	github.com/Azure/go-ansiterm v0.0.0-20230124172434-306776ec8161 // indirect
	github.com/Microsoft/go-winio v0.5.1 // indirect
	github.com/containerd/containerd v1.6.6 // indirect
	github.com/docker/distribution v2.8.1+incompatible // indirect
	github.com/docker/go-units v0.4.0 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/gorilla/mux v1.8.0 // indirect
	github.com/morikuni/aec v1.0.0 // indirect
	github.com/opencontainers/go-digest v1.0.0 // indirect
	github.com/opencontainers/image-spec v1.0.3-0.20211202183452-c5a74bcca799 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/sirupsen/logrus v1.8.1 // indirect
	golang.org/x/net v0.0.0-20211216030914-fe4d6282115f // indirect
	golang.org/x/sys v0.0.0-20220422013727-9388b58f7150 // indirect
	google.golang.org/genproto v0.0.0-20211208223120-3a66f561d7aa // indirect
	google.golang.org/grpc v1.46.0 // indirect
	google.golang.org/protobuf v1.27.1 // indirect
	gotest.tools v2.2.0+incompatible // indirect
)
