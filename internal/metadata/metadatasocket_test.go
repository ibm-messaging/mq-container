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
	"net/http"
	"os"
	"testing"

	"github.com/ibm-messaging/mq-container/internal/metadata/api"
	"github.com/ibm-messaging/mq-container/internal/metadata/fake"
	"github.com/ibm-messaging/mq-container/internal/ready"
)

func TestSendQMStatus(t *testing.T) {
	tests := []struct {
		name               string
		queueManagerStatus ready.QMStatus
		// Return code to be returned if a valid status is sent
		qmStatusEndpointGoodResponse int
		wantErr                      bool
	}{
		{
			name:                         "active-202",
			queueManagerStatus:           ready.StatusActiveQM,
			qmStatusEndpointGoodResponse: http.StatusAccepted,
			wantErr:                      false,
		},
		{
			name:                         "active-500",
			queueManagerStatus:           ready.StatusActiveQM,
			qmStatusEndpointGoodResponse: http.StatusInternalServerError,
			wantErr:                      true,
		},
		{
			name:                         "active-300",
			queueManagerStatus:           ready.StatusActiveQM,
			qmStatusEndpointGoodResponse: 300,
			wantErr:                      true,
		},
		{
			// 1XX codes are informational, and a 200 is also returned.
			name:                         "active-199",
			queueManagerStatus:           ready.StatusActiveQM,
			qmStatusEndpointGoodResponse: 199,
			wantErr:                      false,
		},
		{
			name:                         "active-299",
			queueManagerStatus:           ready.StatusActiveQM,
			qmStatusEndpointGoodResponse: 299,
			wantErr:                      false,
		},
		{
			name:                         "active-200",
			queueManagerStatus:           ready.StatusActiveQM,
			qmStatusEndpointGoodResponse: 200,
			wantErr:                      false,
		},
		{
			name:                         "recovery-202",
			queueManagerStatus:           ready.StatusRecoveryQM,
			qmStatusEndpointGoodResponse: http.StatusAccepted,
			wantErr:                      false,
		},
		{
			name:                         "replica-202",
			queueManagerStatus:           ready.StatusReplicaQM,
			qmStatusEndpointGoodResponse: http.StatusAccepted,
			wantErr:                      false,
		},
		{
			name:                         "standby-202",
			queueManagerStatus:           ready.StatusStandbyQM,
			qmStatusEndpointGoodResponse: http.StatusAccepted,
			wantErr:                      false,
		},
		{
			name:                         "unknown-400",
			queueManagerStatus:           ready.StatusUnknown,
			qmStatusEndpointGoodResponse: http.StatusBadRequest,
			wantErr:                      true,
		},
		{
			name:                         "recovery-500",
			queueManagerStatus:           ready.StatusRecoveryQM,
			qmStatusEndpointGoodResponse: http.StatusInternalServerError,
			wantErr:                      true,
		},
		{
			name:                         "replica-500",
			queueManagerStatus:           ready.StatusReplicaQM,
			qmStatusEndpointGoodResponse: http.StatusInternalServerError,
			wantErr:                      true,
		},
		{
			name:                         "standby-500",
			queueManagerStatus:           ready.StatusStandbyQM,
			qmStatusEndpointGoodResponse: http.StatusInternalServerError,
			wantErr:                      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a temporary directory to write the socket into
			tmpDir := t.TempDir()
			socketPath := tmpDir + "/test.sock"

			server, err := fake.NewMetadataServer(socketPath, api.EndpointQMStatus, tt.qmStatusEndpointGoodResponse)
			if err != nil {
				t.Fatal("fake.NewMetadataServer() failed to create a server")
			}
			server.Start()
			defer server.Close()

			// Override socket path
			err = os.Setenv(socketPathOverrideEnvVar, socketPath)
			if err != nil {
				t.Fatal("could not set socketPath override")
			}
			unsetSocketPathOverride := func() {
				_ = os.Unsetenv(socketPathOverrideEnvVar)
			}
			defer unsetSocketPathOverride()

			gotErr := SendQMStatus(t.Context(), tt.queueManagerStatus)
			if gotErr != nil {
				if !tt.wantErr {
					t.Errorf("SendQMStatus() failed: %v", gotErr)
				}
				return
			}
			if tt.wantErr {
				t.Fatal("SendQMStatus() succeeded unexpectedly")
			}
		})
	}
}

func TestCheckReadyStatus(t *testing.T) {
	tests := []struct {
		name                      string
		readyEndpointGoodResponse int
		wantErr                   bool
	}{
		{
			name:                      "accept",
			readyEndpointGoodResponse: http.StatusAccepted,
			wantErr:                   false,
		},
		{
			name:                      "badreq",
			readyEndpointGoodResponse: http.StatusBadRequest,
			wantErr:                   true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			// Create a temporary directory to write the socket into
			tmpDir := t.TempDir()
			socketPath := tmpDir + "/test.sock"

			server, err := fake.NewMetadataServer(socketPath, api.EndpointReady, tt.readyEndpointGoodResponse)
			if err != nil {
				t.Fatal("fake.NewMetadataServer() failed to create a server")
			}

			server.Start()
			defer server.Close()

			// Override socket path
			err = os.Setenv(socketPathOverrideEnvVar, socketPath)
			if err != nil {
				t.Fatal("could not set socketPath override")
			}
			unsetSocketPathOverride := func() {
				_ = os.Unsetenv(socketPathOverrideEnvVar)
			}
			defer unsetSocketPathOverride()

			gotErr := CheckReadyStatus(t.Context())
			if gotErr != nil {
				if !tt.wantErr {
					t.Errorf("CheckReadyStatus() failed: %v", gotErr)
				}
				return
			}
			if tt.wantErr {
				t.Fatal("CheckReadyStatus() succeeded unexpectedly")
			}
		})
	}
}

func TestCheckLiveStatus(t *testing.T) {
	tests := []struct {
		name                     string
		liveEndpointGoodResponse int
		wantErr                  bool
	}{
		{
			name:                     "accept",
			liveEndpointGoodResponse: http.StatusAccepted,
			wantErr:                  false,
		},
		{
			name:                     "badreq",
			liveEndpointGoodResponse: http.StatusBadRequest,
			wantErr:                  true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			// Create a temporary directory to write the socket into
			tmpDir := t.TempDir()
			socketPath := tmpDir + "/test.sock"

			server, err := fake.NewMetadataServer(socketPath, api.EndpointLive, tt.liveEndpointGoodResponse)
			if err != nil {
				t.Fatal("fake.NewMetadataServer() failed to create a server")
			}

			server.Start()
			defer server.Close()

			// Override socket path
			err = os.Setenv(socketPathOverrideEnvVar, socketPath)
			if err != nil {
				t.Fatal("could not set socketPath override")
			}
			unsetSocketPathOverride := func() {
				_ = os.Unsetenv(socketPathOverrideEnvVar)
			}
			defer unsetSocketPathOverride()

			gotErr := CheckLiveStatus(t.Context())
			if gotErr != nil {
				if !tt.wantErr {
					t.Errorf("CheckLiveStatus() failed: %v", gotErr)
				}
				return
			}
			if tt.wantErr {
				t.Fatal("CheckLiveStatus() succeeded unexpectedly")
			}
		})
	}
}
