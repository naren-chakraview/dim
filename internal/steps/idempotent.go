package steps

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/naren-chakraview/dim/internal/engine"
	"github.com/naren-chakraview/dim/internal/expr"
)

// IdempotentStep implements message deduplication using an in-memory key set with TTL-based eviction.
// Messages with duplicate deduplication keys are dropped (return nil).
// Keys are extracted via JSONata expression and remembered for the configured TTL.
type IdempotentStep struct {
	evaluator *expr.Evaluator
	keySet    map[string]time.Time // key -> expiry time
	lock      sync.RWMutex         // protect keySet
	ttl       time.Duration
	stopCh    chan struct{}         // signal to stop eviction goroutine
	wg        sync.WaitGroup        // wait for eviction goroutine
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
		evaluator: evaluator,
		keySet:    make(map[string]time.Time),
		ttl:       ttl,
		stopCh:    make(chan struct{}),
	}

	// Start background eviction goroutine
	step.wg.Add(1)
	go step.evictionLoop()

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

// Stop gracefully shuts down the eviction goroutine.
// This should be called when the step is being cleaned up.
func (is *IdempotentStep) Stop(ctx context.Context) error {
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
}
