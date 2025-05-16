package metrics

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestAuditingHandler(t *testing.T) {
	tests := []struct {
		name        string
		writeStatus bool
		statusCode  int
	}{
		{"goodpath", true, http.StatusOK},
		{"badrequest", true, http.StatusBadRequest},
		{"noresponse", false, 0},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			logger := &auditTestLogger{}
			testAuditWrapper := newAuditingHandlerFuncWrapper(test.name, logger)
			testBaseFunc := func(w http.ResponseWriter, req *http.Request) {
				if test.writeStatus {
					w.WriteHeader(test.statusCode)
				}
			}
			handler := testAuditWrapper(testBaseFunc)
			recorder := &httptest.ResponseRecorder{}
			testRequest := httptest.NewRequest(http.MethodGet, "http://localhost/metrics", nil)
			testRequestRemote := testRequest.RemoteAddr

			beforeEvent := time.Now().UTC()
			handler.ServeHTTP(recorder, testRequest)
			afterEvent := time.Now().UTC()

			if test.writeStatus && recorder.Code != test.statusCode {
				t.Fatalf("Unexpected status code sent (expected %d, got %d)", test.statusCode, recorder.Code)
			}

			if len(logger.logs) != 1 {
				t.Fatalf("Incorrect number of audit events produced (expect 1, got %d)", len(logger.logs))
			}
			event := logger.logs[0]

			t.Logf("Audit event: %s", event)

			decoded := auditEvent{}
			err := json.Unmarshal([]byte(event), &decoded)
			if err != nil {
				t.Fatalf("Failed to unmarshal audit event: %s", err.Error())
			}

			if decoded.QueuemanagerName != test.name {
				t.Fatalf("Incorrect queuemanager name recorded in audit event (expected '%s', got '%s')", test.name, decoded.QueuemanagerName)
			}

			if decoded.RemoteAddr != testRequestRemote {
				t.Fatalf("Incorrect remote address recorded in audit event (expected '%s', got '%s')", testRequestRemote, decoded.RemoteAddr)
			}

			if test.writeStatus && decoded.StatusCode != test.statusCode {
				t.Fatalf("Unexpected status code recorded in audit event (expected %d, got %d)", test.statusCode, decoded.StatusCode)
			} else if !test.writeStatus && decoded.StatusCode != 0 {
				t.Fatalf("Unexpected status code recorded in audit event (expected 0, got %d)", decoded.StatusCode)
			}

			ts, err := time.Parse(time.RFC3339, decoded.Timestamp)
			if err != nil {
				t.Fatalf("Failed to parse audit event timestamp: %s", err.Error())
			}
			if ts.Before(beforeEvent.Truncate(time.Second)) || ts.After(afterEvent) {
				t.Fatalf("Audit timestamp outside expected range (expected between '%s' and '%s', got '%s')", beforeEvent.Format(time.RFC3339), afterEvent.Format(time.RFC3339), decoded.Timestamp)
			}
		})
	}
}

type auditTestLogger struct {
	logs []string
}

func (a *auditTestLogger) Append(messageLine string, deduplicateLine bool) {
	a.logs = append(a.logs, messageLine)
}
