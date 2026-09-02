package engine

import (
	"context"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"
)

// GenerationState represents the lifecycle state of a generation
type GenerationState int

const (
	StateActive GenerationState = iota
	StateDraining
	StateExpired
)

// String returns a human-readable string for the generation state
func (s GenerationState) String() string {
	switch s {
	case StateActive:
		return "Active"
	case StateDraining:
		return "Draining"
	case StateExpired:
		return "Expired"
	default:
		return "Unknown"
	}
}

// Generation represents a single lifecycle of an executor (route config version).
// When a route config changes, a new Generation is created with a new Executor.
// The old generation transitions to Draining state while the new one becomes Active.
// Messages sent to the old generation complete via its Executor's drain logic,
// while new messages are routed to the active generation.
type Generation struct {
	ID               int64              // Generation sequence number (incrementing)
	State            GenerationState    // Active/Draining/Expired
	Executor         *Executor          // Wrapped executor
	RouteVersion     string             // SHA256 hash of route config (for lineage tracking)
	CreatedAt        time.Time          // When this generation was created
	DrainingStarted  time.Time          // When state changed to Draining (zero if still Active)
	DrainingDeadline time.Time          // Timeout deadline for draining (zero if still Active)
	DoneCh           chan struct{}      // Closed when draining complete (for waiting)
	errorCh          chan error         // Errors encountered during draining
	drainOnce        sync.Once          // Ensure drain logic runs once

	mu sync.RWMutex // Protect State field and time fields
}

// NewGeneration creates a new Generation wrapping the given executor.
func NewGeneration(id int64, executor *Executor, routeVersion string) *Generation {
	return &Generation{
		ID:           id,
		State:        StateActive,
		Executor:     executor,
		RouteVersion: routeVersion,
		CreatedAt:    time.Now(),
		DoneCh:       make(chan struct{}),
		errorCh:      make(chan error, 1),
	}
}

// IsActive returns true if this generation is currently accepting new messages.
func (g *Generation) IsActive() bool {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.State == StateActive
}

// GetInputChannel returns the executor's input channel.
// Safe to call regardless of generation state (but only Active generations receive new messages).
func (g *Generation) GetInputChannel() *Channel {
	return g.Executor.GetInputChannel()
}

// StartDraining transitions the generation from Active to Draining and starts a background drain task.
// Returns immediately; the caller does not block on drain completion.
func (g *Generation) StartDraining(ctx context.Context) {
	g.mu.Lock()
	if g.State != StateActive {
		// Already draining or expired; nothing to do
		g.mu.Unlock()
		return
	}
	g.State = StateDraining
	g.DrainingStarted = time.Now()
	g.DrainingDeadline = time.Now().Add(30 * time.Second)
	g.mu.Unlock()

	// Start background drain task (only once)
	g.drainOnce.Do(func() {
		go g.drainInBackground(ctx)
	})
}

// drainInBackground polls the executor's InFlightCount until it reaches zero or deadline passes.
// Closes DoneCh when draining completes, sends any error to errorCh.
func (g *Generation) drainInBackground(ctx context.Context) {
	defer close(g.DoneCh)

	g.mu.RLock()
	deadline := g.DrainingDeadline
	routeVersion := g.RouteVersion
	g.mu.RUnlock()

	for {
		inFlight := g.Executor.InFlightCount()
		if inFlight == 0 {
			shortVersion := routeVersion
			if len(routeVersion) > 16 {
				shortVersion = routeVersion[:16]
			}
			log.Printf("[INFO] Generation %d drained successfully (route_version=%s)", g.ID, shortVersion)
			return
		}

		if time.Now().After(deadline) {
			err := fmt.Errorf("generation %d drain timeout: %d messages still in flight", g.ID, inFlight)
			log.Printf("[WARN] %v", err)
			select {
			case g.errorCh <- err:
			default:
			}
			return
		}

		// Poll every 10ms to check in-flight count
		select {
		case <-ctx.Done():
			// Context cancelled; stop draining
			return
		case <-time.After(10 * time.Millisecond):
		}
	}
}

// WaitForDrained blocks until the generation has drained or the deadline passes.
// Returns nil if drain completed successfully, or an error if timeout occurred.
// Does not block if the generation is still Active.
func (g *Generation) WaitForDrained(ctx context.Context) error {
	g.mu.RLock()
	if g.State == StateActive {
		g.mu.RUnlock()
		return nil // Not draining yet
	}
	g.mu.RUnlock()

	select {
	case <-g.DoneCh:
		// Drain completed; check if there was an error
		select {
		case err := <-g.errorCh:
			return err
		default:
			return nil
		}
	case <-ctx.Done():
		return ctx.Err()
	}
}

// GenerationManager coordinates multiple generations for a single route.
// It manages the lifecycle of generations, handles takeovers (when route config changes),
// and enforces a concurrent draining cap to avoid resource exhaustion.
type GenerationManager struct {
	routeName              string
	generations            []*Generation      // Ordered list: most recent first (index 0 is active)
	activeGeneration       *Generation        // Quick pointer to active generation (generations[0])
	nextGenerationID       int64              // Incrementing ID for next generation
	concurrentDrainingCap  int                // Max simultaneous draining generations (default 3)
	currentlyDraining      int32              // Atomic counter of currently draining generations

	mu sync.RWMutex // Protect generations list and active pointer
}

// NewGenerationManager creates a new manager for a route with an initial executor.
func NewGenerationManager(routeName string, initialExecutor *Executor, routeVersion string, concurrentDrainingCap int) *GenerationManager {
	if concurrentDrainingCap < 1 {
		concurrentDrainingCap = 3 // Default
	}

	initialGen := NewGeneration(1, initialExecutor, routeVersion)
	return &GenerationManager{
		routeName:             routeName,
		generations:           []*Generation{initialGen},
		activeGeneration:      initialGen,
		nextGenerationID:      2,
		concurrentDrainingCap: concurrentDrainingCap,
	}
}

// GetActiveInputChannel returns the input channel of the currently active generation.
// Suitable for high-frequency calls (uses read lock).
func (gm *GenerationManager) GetActiveInputChannel() *Channel {
	gm.mu.RLock()
	defer gm.mu.RUnlock()

	if gm.activeGeneration == nil {
		return nil
	}
	return gm.activeGeneration.GetInputChannel()
}

// GetActiveGeneration returns a pointer to the currently active generation.
// Suitable for less frequent operations like starting an executor.
func (gm *GenerationManager) GetActiveGeneration() *Generation {
	gm.mu.RLock()
	defer gm.mu.RUnlock()
	return gm.activeGeneration
}

// Takeover replaces the active generation with a new one.
// The old active generation transitions to Draining state, and new messages are routed to the new generation.
// Draining is enforced with a concurrent-draining cap; if N generations are already draining,
// the drain task waits before proceeding to respect the cap.
// Returns immediately; draining happens in the background.
func (gm *GenerationManager) Takeover(ctx context.Context, newExecutor *Executor, newRouteVersion string) error {
	gm.mu.Lock()
	oldGeneration := gm.activeGeneration

	// Create new generation
	newGeneration := NewGeneration(gm.nextGenerationID, newExecutor, newRouteVersion)
	gm.nextGenerationID++

	// Make the new generation active
	gm.activeGeneration = newGeneration
	gm.generations = append([]*Generation{newGeneration}, gm.generations...)

	gm.mu.Unlock()

	oldVer, newVer := oldGeneration.RouteVersion, newRouteVersion
	if len(oldVer) > 16 {
		oldVer = oldVer[:16]
	}
	if len(newVer) > 16 {
		newVer = newVer[:16]
	}

	log.Printf("[INFO] Takeover for route %q: gen %d → gen %d (version %s → %s)",
		gm.routeName, oldGeneration.ID, newGeneration.ID,
		oldVer, newVer)

	// Start draining the old generation immediately (synchronously transition to Draining state)
	oldGeneration.StartDraining(ctx)

	// Coordinate the drain supervision with concurrent-draining cap in background
	go func() {
		// Wait if we've hit the concurrent draining cap
		for {
			current := atomic.LoadInt32(&gm.currentlyDraining)
			if current < int32(gm.concurrentDrainingCap) {
				if atomic.CompareAndSwapInt32(&gm.currentlyDraining, current, current+1) {
					break
				}
			} else {
				// Cap reached; wait a bit before retrying
				time.Sleep(100 * time.Millisecond)
			}
		}

		// Wait for the drain to complete (with timeout)
		if err := oldGeneration.WaitForDrained(ctx); err != nil {
			log.Printf("[WARN] Generation %d drain failed: %v", oldGeneration.ID, err)
		}

		// Mark this generation as expired
		oldGeneration.mu.Lock()
		oldGeneration.State = StateExpired
		oldGeneration.mu.Unlock()

		// Decrement the draining counter
		atomic.AddInt32(&gm.currentlyDraining, -1)
	}()

	return nil
}

// DrainAll waits for all generations to drain (used at shutdown).
// Starts draining all non-active generations and waits up to timeout for completion.
func (gm *GenerationManager) DrainAll(ctx context.Context, timeout time.Duration) error {
	gm.mu.RLock()
	gensCopy := make([]*Generation, len(gm.generations))
	copy(gensCopy, gm.generations)
	gm.mu.RUnlock()

	// Mark all generations as draining
	for _, gen := range gensCopy {
		gen.StartDraining(ctx)
	}

	// Wait for all to drain
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	drainedCount := 0
	for _, gen := range gensCopy {
		if err := gen.WaitForDrained(ctx); err != nil {
			log.Printf("[WARN] Generation %d failed to drain: %v", gen.ID, err)
		} else {
			drainedCount++
		}
	}

	if drainedCount < len(gensCopy) {
		return fmt.Errorf("DrainAll: %d/%d generations drained (timeout=%v)", drainedCount, len(gensCopy), timeout)
	}

	log.Printf("[INFO] DrainAll for route %q: all %d generations drained successfully", gm.routeName, len(gensCopy))
	return nil
}
