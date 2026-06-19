package domain

import (
	"context"
	"time"
)

// Summary holds the aggregated statistics for a batch of URL checks.
type Summary struct {
	Total      int `json:"total"`
	Up         int `json:"up"`
	Down       int `json:"down"`
	DurationMs int `json:"duration_ms"`
}

// CheckResult represents the outcome of a single URL check.
type CheckResult struct {
	URL string `json:"url"`
	// Using omitempty: if the request fails (e.g., DNS error), StatusCode is 0 and won't appear in JSON.
	StatusCode int  `json:"status_code,omitempty"`
	OK         bool `json:"ok"`
	LatencyMs  int  `json:"latency_ms"`
	// Using omitempty: if there is no error, it won't appear in the JSON.
	Error string `json:"error,omitempty"`
}

// Batch represents a completed or ongoing group of URL checks.
type Batch struct {
	ID        string        `json:"batch_id"`
	CreatedAt time.Time     `json:"created_at"`
	Summary   Summary       `json:"summary"`
	Results   []CheckResult `json:"results"`
}

// =====================================================================
// Core Interfaces
// =====================================================================

// Checker verifies a single URL.
// The default implementation makes a real HTTP call; a mock is used in tests.
type Checker interface {
	Check(ctx context.Context, url string) CheckResult
}

// Store persists and retrieves batches.
type Store interface {
	Save(ctx context.Context, b Batch) error
	Get(ctx context.Context, id string) (Batch, error)
}
