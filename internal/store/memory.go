package store

import (
	"context"
	"sync"

	"urlwatch/internal/domain"
)

// MemoryStore is a thread-safe, in-memory implementation of domain.Store.
type MemoryStore struct {
	mu      sync.RWMutex
	batches map[string]domain.Batch
}

// NewMemoryStore initializes and returns a new MemoryStore.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		batches: make(map[string]domain.Batch),
	}
}

// Save stores a batch in memory.
// It uses a full Lock() because it modifies the map.
func (s *MemoryStore) Save(ctx context.Context, b domain.Batch) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// In a real database, we would check for context cancellation here.
	// Since in-memory map assignment is instantaneous, we just do it.
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		s.batches[b.ID] = b
		return nil
	}
}

// Get retrieves a batch by its ID.
// It uses RLock() to allow multiple concurrent reads without blocking each other.
func (s *MemoryStore) Get(ctx context.Context, id string) (domain.Batch, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	select {
	case <-ctx.Done():
		return domain.Batch{}, ctx.Err()
	default:
		batch, exists := s.batches[id]
		if !exists {
			// Using our sentinel error from the domain package
			return domain.Batch{}, domain.ErrBatchNotFound
		}
		return batch, nil
	}
}
