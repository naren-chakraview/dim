package cluster

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"sync"
	"time"

	_ "github.com/lib/pq"
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
	db  *sql.DB
	ttl time.Duration
}

// NewPostgresDedupStore creates a Postgres-backed dedup store.
func NewPostgresDedupStore(dsn string, ttl time.Duration) (*PostgresDedupStore, error) {
	if ttl <= 0 {
		ttl = 60 * time.Minute
	}

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to postgres: %w", err)
	}

	// Verify connection works
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("postgres ping failed: %w", err)
	}

	return &PostgresDedupStore{
		db:  db,
		ttl: ttl,
	}, nil
}

// Check looks up a dedup key in Postgres and marks it as seen if new.
func (p *PostgresDedupStore) Check(ctx context.Context, key string) (bool, error) {
	// Check if key exists and is not expired
	query := "SELECT expires_at FROM dedup_store WHERE dedup_key = $1"
	var expiresAt time.Time
	err := p.db.QueryRowContext(ctx, query, key).Scan(&expiresAt)

	if err == sql.ErrNoRows {
		// Key is new: insert it
		insertQuery := "INSERT INTO dedup_store (dedup_key, expires_at) VALUES ($1, $2) ON CONFLICT (dedup_key) DO NOTHING"
		expireTime := time.Now().Add(p.ttl)
		_, err := p.db.ExecContext(ctx, insertQuery, key, expireTime)
		if err != nil {
			return false, fmt.Errorf("failed to insert dedup key: %w", err)
		}
		return false, nil // Key is new
	}

	if err != nil {
		return false, fmt.Errorf("failed to query dedup key: %w", err)
	}

	// Key exists: check if expired
	if time.Now().After(expiresAt) {
		// Key is expired: treat as new and update
		updateQuery := "UPDATE dedup_store SET expires_at = $1 WHERE dedup_key = $2"
		expireTime := time.Now().Add(p.ttl)
		_, err := p.db.ExecContext(ctx, updateQuery, expireTime, key)
		if err != nil {
			return false, fmt.Errorf("failed to update dedup key: %w", err)
		}
		return false, nil // Key was expired, now renewed
	}

	return true, nil // Key exists and is not expired: duplicate
}

// Delete removes a dedup key from Postgres.
func (p *PostgresDedupStore) Delete(ctx context.Context, key string) error {
	query := "DELETE FROM dedup_store WHERE dedup_key = $1"
	_, err := p.db.ExecContext(ctx, query, key)
	return err
}

// Stop gracefully closes the Postgres connection.
func (p *PostgresDedupStore) Stop(ctx context.Context) error {
	return p.db.Close()
}
