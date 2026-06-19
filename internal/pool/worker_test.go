package pool

import (
	"context"
	"testing"
	"time"

	"urlwatch/internal/domain"
)

// mockChecker allows us to inject custom behavior for testing the pool
type mockChecker struct {
	checkFunc func(ctx context.Context, url string) domain.CheckResult
}

func (m *mockChecker) Check(ctx context.Context, url string) domain.CheckResult {
	if m.checkFunc != nil {
		return m.checkFunc(ctx, url)
	}
	return domain.CheckResult{URL: url, OK: true, StatusCode: 200, LatencyMs: 10}
}

func TestProcessBatch(t *testing.T) {
	t.Run("Successful Fan-out/Fan-in", func(t *testing.T) {
		checker := &mockChecker{
			checkFunc: func(ctx context.Context, url string) domain.CheckResult {
				if url == "https://bad.invalid" {
					return domain.CheckResult{URL: url, OK: false, Error: "dns error"}
				}
				return domain.CheckResult{URL: url, OK: true, StatusCode: 200}
			},
		}

		urls := []string{"https://go.dev", "https://bad.invalid", "https://google.com"}
		ctx := context.Background()

		// Run with concurrency 2 to force the pool to multiplex the 3 URLs
		batch := ProcessBatch(ctx, checker, "b_123", urls, 2)

		if batch.Summary.Total != 3 {
			t.Errorf("Expected 3 total, got %d", batch.Summary.Total)
		}
		if batch.Summary.Up != 2 {
			t.Errorf("Expected 2 up, got %d", batch.Summary.Up)
		}
		if batch.Summary.Down != 1 {
			t.Errorf("Expected 1 down, got %d", batch.Summary.Down)
		}
		if len(batch.Results) != 3 {
			t.Errorf("Expected 3 results, got %d", len(batch.Results))
		}
	})

	t.Run("Context Timeout triggers graceful exit", func(t *testing.T) {
		checker := &mockChecker{
			checkFunc: func(ctx context.Context, url string) domain.CheckResult {
				// Simulate a very slow HTTP call
				select {
				case <-time.After(100 * time.Millisecond):
					return domain.CheckResult{URL: url, OK: true}
				case <-ctx.Done():
					return domain.CheckResult{URL: url, OK: false, Error: "context cancelled"}
				}
			},
		}

		urls := []string{"1.com", "2.com", "3.com", "4.com", "5.com"}

		// Set a timeout of 20ms, meaning no request (which takes 100ms) will finish naturally
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
		defer cancel()

		batch := ProcessBatch(ctx, checker, "b_timeout", urls, 2)

		// The engine should exit quickly without deadlocking
		if batch.Summary.DurationMs > 80 {
			t.Errorf("Expected pool to exit early due to context, took %d ms", batch.Summary.DurationMs)
		}
	})
}
