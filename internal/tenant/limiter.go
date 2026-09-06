package tenant

import (
	"sync"
	"time"
)

// MessageRateLimiter enforces per-domain message rate limits
type MessageRateLimiter struct {
	mu      sync.RWMutex
	manager *Manager

	// Per-tenant rate tracking
	windowStart map[string]time.Time
	windowCount map[string]int
}

// NewMessageRateLimiter creates a new rate limiter
func NewMessageRateLimiter(manager *Manager) *MessageRateLimiter {
	return &MessageRateLimiter{
		manager:     manager,
		windowStart: make(map[string]time.Time),
		windowCount: make(map[string]int),
	}
}

// AllowMessage checks if a message is allowed for the given domain
// Returns (allowed, currentRate) where currentRate is msgs/sec
func (l *MessageRateLimiter) AllowMessage(domain string) (bool, float64) {
	if domain == "" {
		return true, 0 // No domain specified, allow
	}

	cfg := l.manager.GetTenant(domain)
	if cfg.MessageRateLimit == 0 {
		return true, 0 // Unlimited
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	windowStart, exists := l.windowStart[domain]

	// Initialize or reset window if needed
	if !exists || now.Sub(windowStart) > time.Second {
		l.windowStart[domain] = now
		l.windowCount[domain] = 0
	}

	currentCount := l.windowCount[domain]
	currentRate := float64(currentCount) / now.Sub(l.windowStart[domain]).Seconds()

	// Check if within limit
	if currentCount >= cfg.MessageRateLimit {
		return false, currentRate
	}

	// Increment and allow
	l.windowCount[domain]++
	return true, currentRate
}

// GetCurrentRate returns the current message rate for a domain (msgs/sec)
func (l *MessageRateLimiter) GetCurrentRate(domain string) float64 {
	l.mu.RLock()
	defer l.mu.RUnlock()

	windowStart, exists := l.windowStart[domain]
	if !exists {
		return 0
	}

	count := l.windowCount[domain]
	elapsed := time.Since(windowStart).Seconds()
	if elapsed == 0 {
		return 0
	}

	return float64(count) / elapsed
}

// WorkerSlotManager manages per-domain worker slot allocation
type WorkerSlotManager struct {
	mu      sync.Mutex
	manager *Manager

	// Per-tenant slot tracking
	slotsUsed map[string]int
	slotsQueue map[string]int // Queued messages waiting for slot
}

// NewWorkerSlotManager creates a new slot manager
func NewWorkerSlotManager(manager *Manager) *WorkerSlotManager {
	return &WorkerSlotManager{
		manager:    manager,
		slotsUsed:  make(map[string]int),
		slotsQueue: make(map[string]int),
	}
}

// AcquireSlot attempts to acquire a worker slot for a domain
// Returns true if slot acquired, false if would need to queue
func (w *WorkerSlotManager) AcquireSlot(domain string) bool {
	if domain == "" {
		return true // No domain specified, allow
	}

	cfg := w.manager.GetTenant(domain)
	if cfg.WorkerSlots == 0 {
		return true // Unlimited
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	used := w.slotsUsed[domain]
	if used >= cfg.WorkerSlots {
		// No slot available
		w.slotsQueue[domain]++
		return false
	}

	w.slotsUsed[domain]++
	return true
}

// ReleaseSlot releases a worker slot for a domain
func (w *WorkerSlotManager) ReleaseSlot(domain string) {
	if domain == "" {
		return
	}

	cfg := w.manager.GetTenant(domain)
	if cfg.WorkerSlots == 0 {
		return // Unlimited
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	if w.slotsUsed[domain] > 0 {
		w.slotsUsed[domain]--
	}
}

// GetSlotUsage returns (used, queued, limit) for a domain
func (w *WorkerSlotManager) GetSlotUsage(domain string) (used, queued int) {
	if domain == "" {
		return 0, 0
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	return w.slotsUsed[domain], w.slotsQueue[domain]
}

// ResetQueueCount resets the queued count (called when queue is processed)
func (w *WorkerSlotManager) ResetQueueCount(domain string) {
	if domain == "" {
		return
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	w.slotsQueue[domain] = 0
}
