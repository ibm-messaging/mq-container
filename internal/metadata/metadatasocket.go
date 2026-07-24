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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"

	"github.com/ibm-messaging/mq-container/internal/metadata/api"
	"github.com/ibm-messaging/mq-container/internal/ready"
)

// CheckReadyStatus checks if the metadata sidecar is ready to receive requests
func CheckReadyStatus(ctx context.Context) error {
	err := sendHTTPOverSocket(ctx, SocketPath(), api.EndpointReady, http.MethodGet, nil)
	if err != nil {
		return err
	}
	return nil
}

// CheckReadyStatus checks if the metadata sidecar is ready to receive requests
func CheckLiveStatus(ctx context.Context) error {
	err := sendHTTPOverSocket(ctx, SocketPath(), api.EndpointLive, http.MethodGet, nil)
	if err != nil {
		return err
	}
	return nil
}

// SendQMStatus sends the current QueueManager status to the metadata sidecar
func SendQMStatus(ctx context.Context, queueManagerStatus ready.QMStatus) error {
	err := sendHTTPOverSocket(ctx, SocketPath(), api.EndpointQMStatus, http.MethodPost, &api.SocketMessage{Status: queueManagerStatus})
	if err != nil {
		return err
	}
	return nil
}

// sendHTTPOverSocket sends a SocketMessage to a given socket over HTTP
func sendHTTPOverSocket(ctx context.Context, socketPath, endpoint string, httpMethod string, socketMessage *api.SocketMessage) error {
	// Create HTTP client to communicate HTTP over the socket
	socketHttpClient := &http.Client{
		Timeout: 500 * time.Millisecond,
		Transport: &http.Transport{
			DialContext: func(dialCtx context.Context, _, _ string) (net.Conn, error) {
				d := net.Dialer{Timeout: 300 * time.Millisecond}
				return d.DialContext(dialCtx, "unix", socketPath)
			},
		},
	}

	var bodyReader io.Reader
	bodyReader = nil
	// Generate the request
	if socketMessage != nil {
		jsonBytes, err := json.Marshal(socketMessage)
		if err != nil {
			return err
		}
		bodyReader = bytes.NewBuffer(jsonBytes)
	}
	baseAddress, err := getBaseAddress()
	if err != nil {
		return err
	}
	request, err := http.NewRequestWithContext(ctx, httpMethod, baseAddress.JoinPath(endpoint).String(), bodyReader)
	if err != nil {
		return err
	}
	if socketMessage != nil {
		request.Header.Add("Content-Type", "application/json")
	}

	// Send the request
	response, err := socketHttpClient.Do(request)
	if err != nil {
		return err
	}
	closeBody := func() {
		_ = response.Body.Close()
	}
	defer closeBody()

	// Accept only 2XX responses
	if response.StatusCode >= 200 && response.StatusCode < 300 {
		return nil
	}
	return fmt.Errorf("response from sidecar could not be validated")
}
