package api

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"urlwatch/internal/domain"
	"urlwatch/internal/store"
)

// mockChecker for isolated testing
type mockChecker struct{}

func (m *mockChecker) Check(ctx context.Context, url string) domain.CheckResult {
	return domain.CheckResult{URL: url, OK: true, StatusCode: 200, LatencyMs: 10}
}

func setupTestServer() (http.Handler, *store.MemoryStore) {
	memStore := store.NewMemoryStore()
	checker := &mockChecker{}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil)) // Discard logs during tests
	return NewServer(logger, memStore, checker), memStore
}

func TestAPI_GetBatch_NotFound(t *testing.T) {
	srv, _ := setupTestServer()

	req := httptest.NewRequest("GET", "/v1/checks/non_existent", nil)
	rr := httptest.NewRecorder()

	srv.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rr.Code)
	}

	var errResp ErrorResponse
	json.NewDecoder(rr.Body).Decode(&errResp)
	if errResp.Error.Code != "batch_not_found" {
		t.Errorf("expected batch_not_found code, got %s", errResp.Error.Code)
	}
}

func TestAPI_CreateBatch_Valid(t *testing.T) {
	srv, memStore := setupTestServer()

	body := []byte(`{
		"urls": ["https://go.dev"],
		"options": {"concurrency": 2, "timeout_ms": 1000}
	}`)

	req := httptest.NewRequest("POST", "/v1/checks", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	srv.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created, got %d. Body: %s", rr.Code, rr.Body.String())
	}

	var batch domain.Batch
	json.NewDecoder(rr.Body).Decode(&batch)

	if len(batch.Results) != 1 {
		t.Errorf("expected 1 result, got %d", len(batch.Results))
	}

	// Verify it actually saved to the database
	_, err := memStore.Get(context.Background(), batch.ID)
	if err != nil {
		t.Errorf("expected batch to be saved in store, got error: %v", err)
	}
}

func TestAPI_CreateBatch_ValidationError(t *testing.T) {
	srv, _ := setupTestServer()

	// Invalid URL
	body := []byte(`{"urls": ["not-a-url"]}`)
	req := httptest.NewRequest("POST", "/v1/checks", bytes.NewBuffer(body))
	rr := httptest.NewRecorder()

	srv.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 Bad Request, got %d", rr.Code)
	}
}
