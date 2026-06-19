package api

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"urlwatch/internal/domain"
	"urlwatch/internal/pool"
)

type Handler struct {
	store   domain.Store
	checker domain.Checker
}

func NewHandler(store domain.Store, checker domain.Checker) *Handler {
	return &Handler{store: store, checker: checker}
}

// writeJSON is a helper to centralize JSON responses
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

// writeError generates the exact error payload requested by the exam
func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, ErrorResponse{
		Error: ErrorDetail{Code: code, Message: message},
	})
}

// CreateBatch handles POST /v1/checks
func (h *Handler) CreateBatch(w http.ResponseWriter, r *http.Request) {
	var req CreateBatchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "corps JSON malformé")
		return
	}

	if err := req.Validate(); err != nil {
		writeError(w, http.StatusBadRequest, "validation_error", err.Error())
		return
	}

	// Generate a small random ID (e.g., b_4f3c1a)
	b := make([]byte, 3)
	rand.Read(b)
	batchID := "b_" + hex.EncodeToString(b)

	// Create a context with the requested timeout
	ctx, cancel := context.WithTimeout(r.Context(), time.Duration(req.Options.TimeoutMs)*time.Millisecond)
	defer cancel()

	// Execute the concurrent worker pool
	batch := pool.ProcessBatch(ctx, h.checker, batchID, req.URLs, req.Options.Concurrency)

	// Save to store (using background context because we still want to save it even if the request context timed out)
	if err := h.store.Save(context.Background(), batch); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "impossible de sauvegarder le lot")
		return
	}

	w.Header().Set("X-Batch-ID", batchID)
	writeJSON(w, http.StatusCreated, batch)
}

// GetBatch handles GET /v1/checks/{id}
func (h *Handler) GetBatch(w http.ResponseWriter, r *http.Request) {
	// In Go 1.22 ServeMux, PathValue extracts wildcards like {id}
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "id manquant")
		return
	}

	batch, err := h.store.Get(r.Context(), id)
	if err != nil {
		if errors.Is(err, domain.ErrBatchNotFound) {
			writeError(w, http.StatusNotFound, "batch_not_found", "aucun lot avec l'id "+id)
			return
		}
		writeError(w, http.StatusInternalServerError, "internal_error", "erreur lors de la lecture")
		return
	}

	w.Header().Set("X-Batch-ID", id)
	writeJSON(w, http.StatusOK, batch)
}

// Healthz handles GET /healthz
func (h *Handler) Healthz(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok\n"))
}
