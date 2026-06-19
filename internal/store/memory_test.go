package store

import (
	"context"
	"errors"
	"sync"
	"testing"

	"urlwatch/internal/domain"
)

func TestMemoryStore_BasicOperations(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	// 1. Test Get on empty store
	_, err := store.Get(ctx, "non_existent")
	if !errors.Is(err, domain.ErrBatchNotFound) {
		t.Errorf("Expected ErrBatchNotFound, got %v", err)
	}

	// 2. Test Save
	batch := domain.Batch{ID: "batch_1"}
	err = store.Save(ctx, batch)
	if err != nil {
		t.Fatalf("Failed to save batch: %v", err)
	}

	// 3. Test successful Get
	retrieved, err := store.Get(ctx, "batch_1")
	if err != nil {
		t.Fatalf("Failed to get batch: %v", err)
	}
	if retrieved.ID != "batch_1" {
		t.Errorf("Expected batch ID 'batch_1', got '%s'", retrieved.ID)
	}
}

// TestMemoryStore_ConcurrentAccess blasts the store with reads and writes
// to ensure the sync.RWMutex prevents data races.
func TestMemoryStore_ConcurrentAccess(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()
	var wg sync.WaitGroup

	// Write 100 batches concurrently
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(id string) {
			defer wg.Done()
			_ = store.Save(ctx, domain.Batch{ID: id})
		}(string(rune(i))) // Just generating dummy string IDs
	}

	// Read 100 batches concurrently
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(id string) {
			defer wg.Done()
			_, _ = store.Get(ctx, id)
		}(string(rune(i)))
	}

	wg.Wait() // Wait for all reads and writes to finish
}
