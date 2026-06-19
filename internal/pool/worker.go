package pool

import (
	"context"
	"sync"
	"time"

	"urlwatch/internal/domain"
)

// ProcessBatch checks a list of URLs concurrently using a bounded worker pool.
func ProcessBatch(ctx context.Context, checker domain.Checker, batchID string, urls []string, concurrency int) domain.Batch {
	start := time.Now()

	// Sanity check for concurrency limits
	if concurrency < 1 {
		concurrency = 1
	}
	if concurrency > len(urls) {
		concurrency = len(urls) // Optimize: no need for more workers than URLs
	}

	jobs := make(chan string)
	results := make(chan domain.CheckResult)

	// 1. Fan-out: Producer goroutine
	go func() {
		defer close(jobs) // Always close channels when done producing
		for _, url := range urls {
			select {
			case <-ctx.Done():
				return // Context canceled/timed out: stop feeding jobs
			case jobs <- url:
				// Job successfully sent to a worker
			}
		}
	}()

	// 2. The Worker Pool
	var wg sync.WaitGroup
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			// Range safely exits when the 'jobs' channel is closed by the producer
			for url := range jobs {
				// The checker respects ctx natively
				res := checker.Check(ctx, url)
				results <- res
			}
		}()
	}

	// 3. Cleanup: Wait for workers and close results
	go func() {
		wg.Wait()
		close(results) // Closing this signals the fan-in loop below to exit
	}()

	// 4. Fan-in: Collector (runs in the current goroutine)
	var checkResults []domain.CheckResult
	up := 0
	down := 0

	// This loop blocks until 'results' is closed by the cleanup goroutine
	for res := range results {
		checkResults = append(checkResults, res)
		if res.OK {
			up++
		} else {
			down++
		}
	}

	// 5. Construct and return the final Batch
	return domain.Batch{
		ID:        batchID,
		CreatedAt: time.Now().UTC(),
		Summary: domain.Summary{
			Total:      len(urls),
			Up:         up,
			Down:       down,
			DurationMs: int(time.Since(start).Milliseconds()),
		},
		Results: checkResults,
	}
}
