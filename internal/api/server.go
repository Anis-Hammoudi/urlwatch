package api

import (
	"log/slog"
	"net/http"

	"urlwatch/internal/domain"
)

// NewServer configures the Go 1.22 ServeMux with our handlers and middleware.
func NewServer(logger *slog.Logger, store domain.Store, checker domain.Checker) http.Handler {
	mux := http.NewServeMux()
	handler := NewHandler(store, checker)

	// Note the Go 1.22 Method+Path syntax
	mux.HandleFunc("POST /v1/checks", handler.CreateBatch)
	mux.HandleFunc("GET /v1/checks/{id}", handler.GetBatch)
	mux.HandleFunc("GET /healthz", handler.Healthz)

	// Wrap the mux in our middleware stack (Recovery first, then Logging)
	var h http.Handler = mux
	h = LoggingMiddleware(logger, h)
	h = RecoveryMiddleware(logger, h)

	return h
}
