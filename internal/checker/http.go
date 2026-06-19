package checker

import (
	"context"
	"net/http"
	"time"

	"urlwatch/internal/domain"
)

// HTTPChecker implements domain.Checker using the standard net/http client.
type HTTPChecker struct {
	client *http.Client
}

// NewHTTPChecker creates a new HTTPChecker.
// We accept an *http.Client to allow custom transports or global connection pooling.
func NewHTTPChecker(client *http.Client) *HTTPChecker {
	if client == nil {
		client = http.DefaultClient
	}
	return &HTTPChecker{client: client}
}

// Check performs an HTTP GET request to the target URL.
func (c *HTTPChecker) Check(ctx context.Context, targetURL string) domain.CheckResult {
	start := time.Now()
	result := domain.CheckResult{
		URL: targetURL,
	}

	// 1. Create request bound to the provided context (handles timeouts/cancellations)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, targetURL, nil)
	if err != nil {
		result.Error = err.Error()
		result.LatencyMs = int(time.Since(start).Milliseconds())
		return result
	}

	// 2. Execute the request
	resp, err := c.client.Do(req)

	// 3. Record latency immediately after the call
	result.LatencyMs = int(time.Since(start).Milliseconds())

	if err != nil {
		// This captures DNS errors, connection refused, and context timeouts
		result.Error = err.Error()
		return result
	}
	defer resp.Body.Close()

	// 4. Populate success metrics
	result.StatusCode = resp.StatusCode
	result.OK = resp.StatusCode >= 200 && resp.StatusCode < 300

	return result
}
