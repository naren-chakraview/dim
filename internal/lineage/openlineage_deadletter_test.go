package lineage

import (
	"context"
	"testing"
	"time"
)

// TestOpenLineageDeadLetterOnExportFailure verifies failed events route to dead-letter on export failure (M1.7.4, R20)
func TestOpenLineageDeadLetterOnExportFailure(t *testing.T) {
	deadLetterCh := make(chan interface{}, 10)
	// Use invalid URL that will cause connection error
	ole := NewOpenLineageEmitterWithDeadLetter("http://localhost:0", 10, 5000, deadLetterCh)

	// Create a valid event
	event := &OpenLineageEvent{
		EventType: "COMPLETE",
		EventTime: time.Now().UTC().Format(time.RFC3339),
		Producer:  "test",
		Run:       &Run{RunID: "run1"},
		Job:       &Job{Namespace: "ns", Name: "job1"},
		Outputs: []Dataset{
			{
				Namespace: "test",
				Name:      "output",
				Facets: map[string]interface{}{
					"schema": map[string]interface{}{
						"fields": []map[string]interface{}{
							{"name": "id", "type": "string"},
						},
					},
				},
			},
		},
	}

	ctx := context.Background()
	_ = ole.flushBatch(ctx, []*OpenLineageEvent{event})

	// Verify event was routed to dead-letter (connection failed)
	select {
	case evt := <-deadLetterCh:
		if evt == nil {
			t.Fatalf("Expected event in dead-letter, got nil")
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("Expected event in dead-letter within timeout")
	}
}

// TestOpenLineageDeadLetterNilChannel verifies emitter works with nil dead-letter channel
func TestOpenLineageDeadLetterNilChannel(t *testing.T) {
	ole := NewOpenLineageEmitterWithDeadLetter("http://localhost:5000", 10, 5000, nil)

	event := &OpenLineageEvent{
		EventType: "COMPLETE",
		EventTime: time.Now().UTC().Format(time.RFC3339),
		Producer:  "test",
		Run:       &Run{RunID: "run1"},
		Job:       &Job{Namespace: "ns", Name: "job1"},
	}

	ctx := context.Background()
	err := ole.flushBatch(ctx, []*OpenLineageEvent{event})

	// Error expected due to failed connection, but should not crash with nil DLQ
	_ = err
}

// TestOpenLineageDeadLetterBufferFull verifies behavior when dead-letter queue is full (M1.7.4, R20)
func TestOpenLineageDeadLetterBufferFull(t *testing.T) {
	deadLetterCh := make(chan interface{}, 1) // Small buffer
	ole := NewOpenLineageEmitterWithDeadLetter("http://localhost:5000", 10, 5000, deadLetterCh)

	// Fill the dead-letter buffer
	deadLetterCh <- "existing_event"

	event := &OpenLineageEvent{
		EventType: "COMPLETE",
		EventTime: time.Now().UTC().Format(time.RFC3339),
		Producer:  "test",
		Run:       &Run{RunID: "run1"},
		Job:       &Job{Namespace: "ns", Name: "job1"},
	}

	ctx := context.Background()
	// This would try to route to full dead-letter; the emitter should handle it gracefully
	// by logging a warning rather than blocking
	_ = ole.flushBatch(ctx, []*OpenLineageEvent{event})

	// Verify buffer still has only the existing event
	select {
	case val := <-deadLetterCh:
		if val != "existing_event" {
			t.Errorf("Expected existing_event, got %v", val)
		}
	}

	// No more events should be available (failed emit was non-blocking)
	select {
	case <-deadLetterCh:
		t.Fatalf("Unexpected event in dead-letter")
	case <-time.After(100 * time.Millisecond):
		// Expected: dead-letter has no new event
	}
}

// TestOpenLineageEmitterConstructors verifies both constructor styles work
func TestOpenLineageEmitterConstructors(t *testing.T) {
	// Original constructor (no dead-letter)
	ole1 := NewOpenLineageEmitter("http://localhost:5000", 10, 5000)
	if ole1 == nil {
		t.Fatalf("Expected non-nil emitter from NewOpenLineageEmitter")
	}

	// New constructor with dead-letter
	deadLetterCh := make(chan interface{}, 10)
	ole2 := NewOpenLineageEmitterWithDeadLetter("http://localhost:5000", 10, 5000, deadLetterCh)
	if ole2 == nil {
		t.Fatalf("Expected non-nil emitter from NewOpenLineageEmitterWithDeadLetter")
	}

	// Verify deadLetterCh was set on ole2 but not on ole1
	if ole2.deadLetterCh != deadLetterCh {
		t.Fatalf("Dead-letter channel not set correctly on ole2")
	}
	if ole1.deadLetterCh != nil {
		t.Fatalf("Dead-letter channel should be nil on ole1")
	}
}
