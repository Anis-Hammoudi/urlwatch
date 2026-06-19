package checker

import (
	"context"

	"urlwatch/internal/domain"
)

// MockChecker implements domain.Checker for testing purposes without hitting the network.
type MockChecker struct {
	// CheckFunc allows us to inject custom behavior for specific tests
	CheckFunc func(ctx context.Context, url string) domain.CheckResult
}

// Check satisfies the domain.Checker interface.
func (m *MockChecker) Check(ctx context.Context, url string) domain.CheckResult {
	if m.CheckFunc != nil {
		return m.CheckFunc(ctx, url)
	}

	// Default behavior if no custom function is provided
	return domain.CheckResult{
		URL:        url,
		StatusCode: 200,
		OK:         true,
		LatencyMs:  10, // Hardcoded fake latency
	}
}
