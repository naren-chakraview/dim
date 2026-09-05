package cluster

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

// GenerationTracker coordinates hot-reload generations across cluster instances.
// It ensures that either all instances transition to a new generation (coordinated),
// or each transitions independently (staggered), based on configuration.
type GenerationTracker interface {
	// CurrentGeneration returns the current generation version for this instance.
	CurrentGeneration(ctx context.Context) (int64, error)

	// CanTransition checks if this instance can transition to the next generation.
	// Returns true if allowed, false if must wait for other instances.
	CanTransition(ctx context.Context) (bool, error)

	// Transition atomically advances to the next generation.
	// In coordinated mode, waits for all instances to reach the same generation first.
	// In staggered mode, transitions immediately.
	Transition(ctx context.Context, newVersion int64) error

	// Close gracefully shuts down the tracker.
	Close(ctx context.Context) error
}

// StaggeredGenerationTracker allows each instance to transition independently.
// No coordination needed — perfect for gradual, rolling updates.
type StaggeredGenerationTracker struct {
	currentGen int64
	lock       sync.RWMutex
}

// NewStaggeredGenerationTracker creates a tracker for independent transitions.
func NewStaggeredGenerationTracker(initialGen int64) *StaggeredGenerationTracker {
	return &StaggeredGenerationTracker{
		currentGen: initialGen,
	}
}

// CurrentGeneration returns the current generation.
func (t *StaggeredGenerationTracker) CurrentGeneration(ctx context.Context) (int64, error) {
	t.lock.RLock()
	defer t.lock.RUnlock()
	return t.currentGen, nil
}

// CanTransition always returns true in staggered mode (no coordination).
func (t *StaggeredGenerationTracker) CanTransition(ctx context.Context) (bool, error) {
	return true, nil
}

// Transition atomically advances to the next generation.
func (t *StaggeredGenerationTracker) Transition(ctx context.Context, newVersion int64) error {
	t.lock.Lock()
	defer t.lock.Unlock()

	if newVersion <= t.currentGen {
		return fmt.Errorf("cannot transition to older generation: current=%d, requested=%d", t.currentGen, newVersion)
	}

	oldGen := t.currentGen
	t.currentGen = newVersion
	log.Printf("[INFO] generation transition (staggered): %d -> %d", oldGen, newVersion)

	return nil
}

// Close is a no-op for staggered tracker.
func (t *StaggeredGenerationTracker) Close(ctx context.Context) error {
	return nil
}

// CoordinatedGenerationTracker synchronizes generation transitions across all cluster instances.
// All instances must reach the same generation before any can advance further.
// This ensures zero-downtime coordinated config pushes at the cost of slower rollouts.
type CoordinatedGenerationTracker struct {
	currentGen     int64
	clusterConfig  *ClusterConfig
	lock           sync.RWMutex
	syncTimeout    time.Duration
	lastSyncTime   time.Time
	// TODO: implement inter-instance heartbeat protocol for generation synchronization
}

// NewCoordinatedGenerationTracker creates a tracker that coordinates across instances.
func NewCoordinatedGenerationTracker(initialGen int64, clusterConfig *ClusterConfig) *CoordinatedGenerationTracker {
	return &CoordinatedGenerationTracker{
		currentGen:   initialGen,
		clusterConfig: clusterConfig,
		syncTimeout:  30 * time.Second,
		lastSyncTime: time.Now(),
	}
}

// CurrentGeneration returns the current generation.
func (t *CoordinatedGenerationTracker) CurrentGeneration(ctx context.Context) (int64, error) {
	t.lock.RLock()
	defer t.lock.RUnlock()
	return t.currentGen, nil
}

// CanTransition checks if all other instances are ready for transition.
// For now, always returns true (full implementation requires inter-instance coordination).
func (t *CoordinatedGenerationTracker) CanTransition(ctx context.Context) (bool, error) {
	t.lock.RLock()
	defer t.lock.RUnlock()

	// TODO: Implement generation sync with other instances via heartbeat protocol
	// Return false if any other instance is not at the current generation
	// This prevents cascading updates and ensures coordinated transitions

	return true, nil
}

// Transition atomically advances to the next generation.
// In coordinated mode, this should wait for all instances to synchronize.
func (t *CoordinatedGenerationTracker) Transition(ctx context.Context, newVersion int64) error {
	t.lock.Lock()
	defer t.lock.Unlock()

	if newVersion <= t.currentGen {
		return fmt.Errorf("cannot transition to older generation: current=%d, requested=%d", t.currentGen, newVersion)
	}

	// TODO: Broadcast new generation to all other instances
	// Wait for acknowledgments before confirming transition locally

	oldGen := t.currentGen
	t.currentGen = newVersion
	t.lastSyncTime = time.Now()
	log.Printf("[INFO] generation transition (coordinated): %d -> %d", oldGen, newVersion)

	return nil
}

// Close gracefully shuts down the tracker.
func (t *CoordinatedGenerationTracker) Close(ctx context.Context) error {
	return nil
}

// LocalGenerationTracker is a no-op tracker for single-instance deployments.
type LocalGenerationTracker struct {
	currentGen int64
	lock       sync.RWMutex
}

// NewLocalGenerationTracker creates a tracker for single-instance mode.
func NewLocalGenerationTracker(initialGen int64) *LocalGenerationTracker {
	return &LocalGenerationTracker{
		currentGen: initialGen,
	}
}

func (t *LocalGenerationTracker) CurrentGeneration(ctx context.Context) (int64, error) {
	t.lock.RLock()
	defer t.lock.RUnlock()
	return t.currentGen, nil
}

func (t *LocalGenerationTracker) CanTransition(ctx context.Context) (bool, error) {
	return true, nil
}

func (t *LocalGenerationTracker) Transition(ctx context.Context, newVersion int64) error {
	t.lock.Lock()
	defer t.lock.Unlock()

	if newVersion <= t.currentGen {
		return fmt.Errorf("cannot transition to older generation: current=%d, requested=%d", t.currentGen, newVersion)
	}

	oldGen := t.currentGen
	t.currentGen = newVersion
	log.Printf("[INFO] generation transition (local): %d -> %d", oldGen, newVersion)

	return nil
}

func (t *LocalGenerationTracker) Close(ctx context.Context) error {
	return nil
}
