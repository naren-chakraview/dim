package lineage

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestReaperCreation(t *testing.T) {
	dbPath := t.TempDir() + "/test.db"
	defer os.Remove(dbPath)

	store, err := NewStore(dbPath)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	defer store.Close()

	resolver := NewRetentionResolver(map[string]time.Duration{
		"default": 30 * 24 * time.Hour,
	})

	cadences := map[string]time.Duration{
		"default": 1 * time.Hour,
	}

	reaper := NewReaper(store, resolver, cadences)
	if reaper == nil {
		t.Error("NewReaper returned nil")
	}
}

func TestReaperStart(t *testing.T) {
	dbPath := t.TempDir() + "/test.db"
	defer os.Remove(dbPath)

	store, err := NewStore(dbPath)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	defer store.Close()

	resolver := NewRetentionResolver(map[string]time.Duration{
		"default": 30 * 24 * time.Hour,
	})

	cadences := map[string]time.Duration{
		"default": 10 * time.Millisecond,
	}

	reaper := NewReaper(store, resolver, cadences)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err = reaper.Start(ctx)
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Let it run briefly
	time.Sleep(100 * time.Millisecond)

	err = reaper.Stop()
	if err != nil {
		t.Fatalf("Stop failed: %v", err)
	}
}

func TestReaperCleanup(t *testing.T) {
	dbPath := t.TempDir() + "/test.db"
	defer os.Remove(dbPath)

	store, err := NewStore(dbPath)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	now := time.Now().UTC()

	// Insert an expired record
	expiredTime := now.Add(-2 * time.Hour)
	_, err = store.db.Exec(`
	INSERT INTO lineage_records (
		id, route_name, subject_id, message_body, message_headers, metadata,
		route_version, contract_version, principal, retention_policy,
		created_at, processed_at, expires_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		"expired-id", "route", "subj-123", "body", "headers", "metadata",
		"v1", "", "user", "default",
		expiredTime, expiredTime, expiredTime,
	)
	if err != nil {
		t.Fatalf("Insert failed: %v", err)
	}

	// Verify it exists
	expired, err := store.QueryExpiredRecords(ctx, now)
	if err != nil || len(expired) != 1 {
		t.Error("Expired record not found")
	}

	// Reap manually
	_, err = store.DeleteBySubject(ctx, "subj-123")
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Verify it's gone
	expired, err = store.QueryExpiredRecords(ctx, now)
	if err != nil || len(expired) != 0 {
		t.Error("Expired record was not deleted")
	}
}
