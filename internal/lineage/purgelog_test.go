package lineage

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestPurgeLogReaperCreation(t *testing.T) {
	dbPath := t.TempDir() + "/test.db"
	defer os.Remove(dbPath)

	store, err := NewStore(dbPath)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	defer store.Close()

	reaper := NewPurgeLogReaper(
		store,
		365*24*time.Hour,  // 1 year TTL
		30*24*time.Hour,   // 30 day warning lead
		1*time.Hour,       // 1 hour check cadence
	)

	if reaper == nil {
		t.Error("NewPurgeLogReaper returned nil")
	}

	if reaper.ttl != 365*24*time.Hour {
		t.Errorf("Wrong TTL: expected 365d, got %v", reaper.ttl)
	}
}

func TestPurgeLogReaperStart(t *testing.T) {
	dbPath := t.TempDir() + "/test.db"
	defer os.Remove(dbPath)

	store, err := NewStore(dbPath)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	defer store.Close()

	reaper := NewPurgeLogReaper(
		store,
		365*24*time.Hour,
		30*24*time.Hour,
		10*time.Millisecond,  // Short cadence for testing
	)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err = reaper.Start(ctx)
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	time.Sleep(50 * time.Millisecond)

	err = reaper.Stop()
	if err != nil {
		t.Fatalf("Stop failed: %v", err)
	}
}

func TestPurgeLogWarningThreshold(t *testing.T) {
	dbPath := t.TempDir() + "/test.db"
	defer os.Remove(dbPath)

	store, err := NewStore(dbPath)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	now := time.Now().UTC()

	// Insert purge event that will expire soon
	warningLead := 30 * 24 * time.Hour
	expiresAt := now.Add(15 * 24 * time.Hour) // Expires in 15 days (within 30-day lead)

	err = store.RecordPurgeEvent(
		ctx,
		"event-1",
		"manual_purge",
		"subj-123",
		"Test purge",
		5,
		expiresAt,
	)
	if err != nil {
		t.Fatalf("RecordPurgeEvent failed: %v", err)
	}

	// Query events in warning window
	warningThreshold := now.Add(warningLead)
	events, err := store.QueryPurgeEvents(ctx, now, warningThreshold)
	if err != nil {
		t.Fatalf("QueryPurgeEvents failed: %v", err)
	}

	if len(events) != 1 {
		t.Errorf("Expected 1 event in warning window, got %d", len(events))
	}

	// Verify event details
	if events[0].ID != "event-1" {
		t.Errorf("Wrong event ID: %s", events[0].ID)
	}

	if events[0].PurgedCount != 5 {
		t.Errorf("Wrong purged count: expected 5, got %d", events[0].PurgedCount)
	}
}
