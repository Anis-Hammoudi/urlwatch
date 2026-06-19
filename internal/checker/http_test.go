package checker

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestHTTPChecker_Check(t *testing.T) {
	// Setup a fake local server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/ok":
			w.WriteHeader(http.StatusOK)
		case "/error":
			w.WriteHeader(http.StatusInternalServerError)
		case "/slow":
			time.Sleep(100 * time.Millisecond) // Simulate a slow response
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer server.Close()

	checker := NewHTTPChecker(server.Client())

	tests := []struct {
		name       string
		url        string
		timeout    time.Duration
		wantOK     bool
		wantStatus int
		wantErr    bool
	}{
		{
			name:       "Successful 200 OK",
			url:        server.URL + "/ok",
			timeout:    1 * time.Second,
			wantOK:     true,
			wantStatus: 200,
			wantErr:    false,
		},
		{
			name:       "Server returns 500",
			url:        server.URL + "/error",
			timeout:    1 * time.Second,
			wantOK:     false,
			wantStatus: 500,
			wantErr:    false, // No network error, just a bad HTTP status
		},
		{
			name:       "Context Timeout",
			url:        server.URL + "/slow",
			timeout:    10 * time.Millisecond, // Timeout triggers before the 100ms server sleep
			wantOK:     false,
			wantStatus: 0,
			wantErr:    true, // Expecting a "context deadline exceeded" error
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), tt.timeout)
			defer cancel()

			result := checker.Check(ctx, tt.url)

			if result.OK != tt.wantOK {
				t.Errorf("expected OK %v, got %v", tt.wantOK, result.OK)
			}
			if result.StatusCode != tt.wantStatus {
				t.Errorf("expected Status %d, got %d", tt.wantStatus, result.StatusCode)
			}
			if (result.Error != "") != tt.wantErr {
				t.Errorf("expected Error existence %v, got '%v'", tt.wantErr, result.Error)
			}
			if result.LatencyMs < 0 {
				t.Errorf("expected LatencyMs to be >= 0, got %d", result.LatencyMs)
			}
		})
	}
}
