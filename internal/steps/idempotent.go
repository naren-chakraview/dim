package steps

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/naren-chakraview/dim/internal/cluster"
	"github.com/naren-chakraview/dim/internal/engine"
	"github.com/naren-chakraview/dim/internal/expr"
)

// IdempotentStep implements message deduplication using a pluggable dedup store.
// Messages with duplicate deduplication keys are dropped (return nil).
// Keys are extracted via JSONata expression and checked via the dedup store.
type IdempotentStep struct {
	evaluator  *expr.Evaluator
	dedupStore cluster.DedupStore
	// Legacy fields for backwards compatibility (when no dedupStore is provided)
	keySet    map[string]time.Time // key -> expiry time
	lock      sync.RWMutex         // protect keySet
	ttl       time.Duration
	stopCh    chan struct{}         // signal to stop eviction goroutine
	wg        sync.WaitGroup        // wait for eviction goroutine
	legacy    bool                  // true if using built-in dedup, false if using dedupStore
}

// NewIdempotentStep creates a new idempotent step from a JSONata key expression.
// The key expression is compiled at construction time to catch syntax errors early.
// ttlMinutes specifies how long to remember dedup keys (default 60 if <= 0).
// A background goroutine is started to evict expired keys every 5 minutes.
func NewIdempotentStep(keyExpr string, ttlMinutes int) (*IdempotentStep, error) {
	evaluator, err := expr.CompileExpression(keyExpr)
	if err != nil {
		return nil, fmt.Errorf("failed to compile idempotent key expression: %w", err)
	}

	// Default TTL: 60 minutes
	ttl := time.Duration(ttlMinutes) * time.Minute
	if ttl <= 0 {
		ttl = 60 * time.Minute
	}

	step := &IdempotentStep{
		evaluator:  evaluator,
		keySet:     make(map[string]time.Time),
		ttl:        ttl,
		stopCh:     make(chan struct{}),
		legacy:     true,
		dedupStore: nil,
	}

	// Start background eviction goroutine for legacy mode
	step.wg.Add(1)
	go step.evictionLoop()

	return step, nil
}

// NewIdempotentStepWithStore creates an idempotent step using an external dedup store (for cluster mode).
func NewIdempotentStepWithStore(keyExpr string, dedupStore cluster.DedupStore) (*IdempotentStep, error) {
	evaluator, err := expr.CompileExpression(keyExpr)
	if err != nil {
		return nil, fmt.Errorf("failed to compile idempotent key expression: %w", err)
	}

	step := &IdempotentStep{
		evaluator:  evaluator,
		dedupStore: dedupStore,
		keySet:     nil,
		legacy:     false,
	}

	return step, nil
}

// Execute evaluates the idempotent key expression against the message.
// Returns:
// - (msg, nil) if the key is new or expired (unique message, pass through)
// - (nil, nil) if the key has been seen and not expired (duplicate, drop message)
// - (nil, error) if the key expression evaluation fails
func (is *IdempotentStep) Execute(ctx context.Context, msg *engine.Message) (*engine.Message, error) {
	if msg == nil {
		return nil, nil
	}

	// Build context for JSONata evaluation
	context := map[string]interface{}{
		"body":     msg.Body,
		"headers":  msg.Headers,
		"metadata": msg.Metadata,
	}

	// Evaluate the key expression
	result, err := is.evaluator.Eval(context)
	if err != nil {
		return nil, fmt.Errorf("idempotent key expression evaluation failed: %w", err)
	}

	// Convert result to string (dedup key)
	key := fmt.Sprintf("%v", result)

	if is.legacy {
		// Use legacy in-memory dedup store
		return is.executeWithLegacyStore(ctx, msg, key)
	} else {
		// Use pluggable dedup store (cluster mode)
		return is.executeWithDedupStore(ctx, msg, key)
	}
}

// executeWithDedupStore checks dedup using the pluggable store.
func (is *IdempotentStep) executeWithDedupStore(ctx context.Context, msg *engine.Message, key string) (*engine.Message, error) {
	isDup, err := is.dedupStore.Check(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("dedup store check failed: %w", err)
	}

	if isDup {
		// Key found and not expired: duplicate message, drop it
		return nil, nil
	}

	// Key is new or expired: pass through
	return msg, nil
}

// executeWithLegacyStore checks dedup using the legacy in-memory store.
func (is *IdempotentStep) executeWithLegacyStore(ctx context.Context, msg *engine.Message, key string) (*engine.Message, error) {
	// Check if key exists and not expired
	is.lock.RLock()
	expiry, exists := is.keySet[key]
	is.lock.RUnlock()

	if exists && time.Now().Before(expiry) {
		// Key found and not expired: duplicate message, drop it
		return nil, nil
	}

	// Key is new or expired: add to set and pass through
	expireTime := time.Now().Add(is.ttl)
	is.lock.Lock()
	is.keySet[key] = expireTime

	// Warn if set grows too large (bounded memory check)
	if len(is.keySet) > 100000 {
		log.Printf("WARNING: idempotent dedup set size (%d) exceeds 100k keys; consider shorter TTL", len(is.keySet))
	}

	is.lock.Unlock()

	return msg, nil
}

// evictionLoop runs in a background goroutine and periodically cleans expired keys.
// It runs every 5 minutes and removes all keys where the current time is past the expiry time.
func (is *IdempotentStep) evictionLoop() {
	defer is.wg.Done()

	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-is.stopCh:
			return
		case <-ticker.C:
			is.evictExpiredKeys()
		}
	}
}

// evictExpiredKeys removes all expired keys from the dedup set.
func (is *IdempotentStep) evictExpiredKeys() {
	is.lock.Lock()
	defer is.lock.Unlock()

	now := time.Now()
	expired := 0

	for key, expiry := range is.keySet {
		if now.After(expiry) {
			delete(is.keySet, key)
			expired++
		}
	}

	if expired > 0 {
		log.Printf("idempotent: evicted %d expired keys (set size now: %d)", expired, len(is.keySet))
	}
}

// Stop gracefully shuts down the idempotent step.
// For legacy mode, stops the eviction goroutine.
// For dedup store mode, stops the underlying dedup store.
func (is *IdempotentStep) Stop(ctx context.Context) error {
	if is.legacy {
		close(is.stopCh)
		done := make(chan struct{})
		go func() {
			is.wg.Wait()
			close(done)
		}()

		select {
		case <-done:
			return nil
		case <-ctx.Done():
			return fmt.Errorf("idempotent step shutdown timeout: %w", ctx.Err())
		}
	} else {
		// Stop the dedupStore
		return is.dedupStore.Stop(ctx)
	}
}
