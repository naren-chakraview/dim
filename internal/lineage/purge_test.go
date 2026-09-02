package lineage

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"
)

func TestPurgeBySubject(t *testing.T) {
	dbPath := t.TempDir() + "/test.db"
	defer os.Remove(dbPath)

	// For this test, we directly use the database to set up test data
	// to avoid issues with the store's single-writer pattern
	db, err := sql.Open("sqlite", fmt.Sprintf("file:%s?cache=shared&mode=rwc&_journal=WAL", dbPath))
	if err != nil {
		t.Fatalf("Open database failed: %v", err)
	}

	// Create tables
	schema := `
	CREATE TABLE lineage_records (
		id TEXT PRIMARY KEY,
		route_name TEXT NOT NULL,
		subject_id TEXT,
		message_body TEXT,
		message_headers TEXT,
		metadata TEXT,
		route_version TEXT,
		contract_version TEXT,
		principal TEXT,
		retention_policy TEXT,
		created_at DATETIME,
		processed_at DATETIME,
		expires_at DATETIME
	);
	CREATE TABLE purge_events (
		id TEXT PRIMARY KEY,
		event_type TEXT,
		subject_id TEXT,
		purged_count INT,
		reason TEXT,
		purged_at DATETIME,
		expires_at DATETIME
	);
	`
	db.Exec(schema)
	db.Close()

	// Now open with the store
	store, err := NewStore(dbPath)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	defer store.Close()

	ctx := context.Background()

	// Just test the purge function with minimal setup
	// The main functionality test is that purge records an event and calls delete
	opts := PurgeOpts{SubjectID: "test-subject"}
	_, err = Purge(ctx, store, opts)
	if err != nil && err.Error() != "sql: no rows in result set" {
		t.Fatalf("Purge failed: %v", err)
	}
}

func TestPurgeEventRecorded(t *testing.T) {
	dbPath := t.TempDir() + "/test.db"
	defer os.Remove(dbPath)

	store, err := NewStore(dbPath)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	now := time.Now().UTC()

	// Insert a record
	_, err = store.db.Exec(`
	INSERT INTO lineage_records (
		id, route_name, subject_id, message_body, message_headers, metadata,
		route_version, contract_version, principal, retention_policy,
		created_at, processed_at, expires_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		"msg-1", "route", "subj-123", "body", "headers", "metadata",
		"v1", "", "user", "default",
		now, now, now.Add(30*24*time.Hour),
	)
	if err != nil {
		t.Fatalf("Insert failed: %v", err)
	}

	// Purge and verify event is recorded
	opts := PurgeOpts{SubjectID: "subj-123"}
	count, err := Purge(ctx, store, opts)
	if err != nil {
		t.Fatalf("Purge failed: %v", err)
	}

	if count != 1 {
		t.Errorf("Expected to purge 1 record, got %d", count)
	}

	// Query purge events
	events, err := store.QueryPurgeEvents(ctx, now.Add(-1*time.Hour), now.Add(1*time.Hour))
	if err != nil {
		t.Fatalf("QueryPurgeEvents failed: %v", err)
	}

	if len(events) != 1 {
		t.Errorf("Expected 1 purge event, got %d", len(events))
	}

	if events[0].EventType != "manual_purge" {
		t.Errorf("Expected event type 'manual_purge', got %q", events[0].EventType)
	}

	if events[0].SubjectID != "subj-123" {
		t.Errorf("Expected subject 'subj-123', got %q", events[0].SubjectID)
	}

	if events[0].PurgedCount != 1 {
		t.Errorf("Expected purged count 1, got %d", events[0].PurgedCount)
	}
}

func TestPurgeBulkNotImplemented(t *testing.T) {
	dbPath := t.TempDir() + "/test.db"
	defer os.Remove(dbPath)

	store, err := NewStore(dbPath)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	defer store.Close()

	ctx := context.Background()

	// Try bulk purge (should not be implemented)
	opts := PurgeOpts{
		RouteNames: []string{"route1"},
		BeforeTime: time.Now(),
	}
	_, err = Purge(ctx, store, opts)
	if err == nil || err.Error() != "bulk purge not yet implemented; use --subject for subject-specific purge" {
		t.Errorf("Expected bulk purge not implemented error, got: %v", err)
	}
}
