package cluster

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

// DedupStore defines the interface for shared message deduplication across cluster instances.
// Implementations handle concurrent dedup key checks from multiple instances without duplicates.
type DedupStore interface {
	// Check looks up a dedup key. Returns true if key exists and not expired, false if new/expired.
	// If the key is new, this call also reserves it for the TTL duration.
	Check(ctx context.Context, key string) (bool, error)

	// Delete removes a dedup key (used for explicit eviction or cleanup).
	Delete(ctx context.Context, key string) error

	// Stop gracefully shuts down the store.
	Stop(ctx context.Context) error
}

// InMemoryDedupStore is a single-instance in-memory dedup store (default for non-clustered mode).
type InMemoryDedupStore struct {
	keySet map[string]time.Time
	lock   sync.RWMutex
	ttl    time.Duration
	stopCh chan struct{}
	wg     sync.WaitGroup
}

// NewInMemoryDedupStore creates an in-memory dedup store with TTL-based eviction.
func NewInMemoryDedupStore(ttl time.Duration) *InMemoryDedupStore {
	if ttl <= 0 {
		ttl = 60 * time.Minute
	}

	store := &InMemoryDedupStore{
		keySet: make(map[string]time.Time),
		ttl:    ttl,
		stopCh: make(chan struct{}),
	}

	// Start background eviction goroutine
	store.wg.Add(1)
	go store.evictionLoop()

	return store
}

// Check looks up a dedup key and marks it as seen if new.
func (s *InMemoryDedupStore) Check(ctx context.Context, key string) (bool, error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	expiry, exists := s.keySet[key]
	if exists && time.Now().Before(expiry) {
		// Key found and not expired: duplicate
		return true, nil
	}

	// Key is new or expired: add to set
	expireTime := time.Now().Add(s.ttl)
	s.keySet[key] = expireTime

	// Warn if set grows too large
	if len(s.keySet) > 100000 {
		log.Printf("WARNING: dedup set size (%d) exceeds 100k keys; consider shorter TTL", len(s.keySet))
	}

	return false, nil
}

// Delete removes a dedup key.
func (s *InMemoryDedupStore) Delete(ctx context.Context, key string) error {
	s.lock.Lock()
	defer s.lock.Unlock()
	delete(s.keySet, key)
	return nil
}

// evictionLoop runs in a background goroutine and periodically cleans expired keys.
func (s *InMemoryDedupStore) evictionLoop() {
	defer s.wg.Done()

	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopCh:
			return
		case <-ticker.C:
			s.evictExpiredKeys()
		}
	}
}

// evictExpiredKeys removes all expired keys from the dedup set.
func (s *InMemoryDedupStore) evictExpiredKeys() {
	s.lock.Lock()
	defer s.lock.Unlock()

	now := time.Now()
	expired := 0

	for key, expiry := range s.keySet {
		if now.After(expiry) {
			delete(s.keySet, key)
			expired++
		}
	}

	if expired > 0 {
		log.Printf("dedup: evicted %d expired keys (set size now: %d)", expired, len(s.keySet))
	}
}

// Stop gracefully shuts down the eviction goroutine.
func (s *InMemoryDedupStore) Stop(ctx context.Context) error {
	close(s.stopCh)
	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("dedup store shutdown timeout: %w", ctx.Err())
	}
}

// RedisDedupStore is a cluster-wide dedup store backed by Redis.
type RedisDedupStore struct {
	// TODO: Implement Redis backend in M3.1.2 follow-up
	// Placeholder for now - will be implemented with Redis client
}

// PostgresDedupStore is a cluster-wide dedup store backed by PostgreSQL.
type PostgresDedupStore struct {
	// TODO: Implement Postgres backend in M3.1.2 follow-up
	// Placeholder for now - will be implemented with PostgreSQL client
}
