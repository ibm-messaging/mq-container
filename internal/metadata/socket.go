/*
© Copyright IBM Corporation 2026

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package metadata

import (
	"net/url"
	"os"
)

const (
	defaultSocketPath        = "/dev/shm/metadata/server.sock"
	socketPathOverrideEnvVar = "MQ_METADATA_SOCKET_PATH"
)

func SocketPath() string {
	sockPath := os.Getenv(socketPathOverrideEnvVar)
	if sockPath == "" {
		return defaultSocketPath
	}
	return sockPath
}

// getBaseAddress returns an address to use when talking to the metadata-manager socket.
// As 'DialContext' is set on the HTTP client 'unix.sock' is not used, but a valid address is required.
func getBaseAddress() (*url.URL, error) {
	return url.Parse("http://unix.sock")
}
