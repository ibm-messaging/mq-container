#!/bin/bash

# Build and test the Go code
go build ./cmd/runmqserver/
go build ./cmd/chkmqready/
go build ./cmd/chkmqhealthy/
go test -v ./cmd/runmqserver/
go test -v ./cmd/chkmqready/
go test -v ./cmd/chkmqhealthy/
go test -v ./internal/...
go vet ./cmd/... ./internal/...
