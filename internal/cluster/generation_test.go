package cluster

import (
	"context"
	"testing"
)

func TestStaggeredGenerationTracker(t *testing.T) {
	ctx := context.Background()
	tracker := NewStaggeredGenerationTracker(1)

	// Check initial generation
	gen, err := tracker.CurrentGeneration(ctx)
	if err != nil {
		t.Fatalf("failed to get current generation: %v", err)
	}
	if gen != 1 {
		t.Errorf("expected initial generation=1, got %d", gen)
	}

	// Can always transition in staggered mode
	canTrans, err := tracker.CanTransition(ctx)
	if err != nil {
		t.Fatalf("failed to check transition: %v", err)
	}
	if !canTrans {
		t.Errorf("expected CanTransition=true in staggered mode")
	}

	// Transition to next generation
	err = tracker.Transition(ctx, 2)
	if err != nil {
		t.Fatalf("failed to transition: %v", err)
	}

	gen, err = tracker.CurrentGeneration(ctx)
	if err != nil {
		t.Fatalf("failed to get current generation: %v", err)
	}
	if gen != 2 {
		t.Errorf("expected generation=2 after transition, got %d", gen)
	}
}

func TestStaggeredGenerationTracker_RegressionError(t *testing.T) {
	ctx := context.Background()
	tracker := NewStaggeredGenerationTracker(5)

	// Should reject transition to older generation
	err := tracker.Transition(ctx, 3)
	if err == nil {
		t.Errorf("expected error when transitioning to older generation")
	}
}

func TestCoordinatedGenerationTracker(t *testing.T) {
	ctx := context.Background()
	instances := []Instance{
		{ID: "dimd-1", Host: "localhost", Port: 8080},
		{ID: "dimd-2", Host: "localhost", Port: 8081},
	}
	config, _ := NewClusterConfigFromStatic(instances, "dimd-1")

	tracker := NewCoordinatedGenerationTracker(1, config)

	// Check initial generation
	gen, err := tracker.CurrentGeneration(ctx)
	if err != nil {
		t.Fatalf("failed to get current generation: %v", err)
	}
	if gen != 1 {
		t.Errorf("expected initial generation=1, got %d", gen)
	}

	// Transition to next generation
	err = tracker.Transition(ctx, 2)
	if err != nil {
		t.Fatalf("failed to transition: %v", err)
	}

	gen, err = tracker.CurrentGeneration(ctx)
	if err != nil {
		t.Fatalf("failed to get current generation: %v", err)
	}
	if gen != 2 {
		t.Errorf("expected generation=2 after transition, got %d", gen)
	}
}

func TestLocalGenerationTracker(t *testing.T) {
	ctx := context.Background()
	tracker := NewLocalGenerationTracker(1)

	gen, err := tracker.CurrentGeneration(ctx)
	if err != nil {
		t.Fatalf("failed to get current generation: %v", err)
	}
	if gen != 1 {
		t.Errorf("expected initial generation=1, got %d", gen)
	}

	err = tracker.Transition(ctx, 2)
	if err != nil {
		t.Fatalf("failed to transition: %v", err)
	}

	gen, err = tracker.CurrentGeneration(ctx)
	if err != nil {
		t.Fatalf("failed to get current generation: %v", err)
	}
	if gen != 2 {
		t.Errorf("expected generation=2 after transition, got %d", gen)
	}
}
