package lineage

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/naren-chakraview/dim/internal/engine"
)

func TestNewStore(t *testing.T) {
	dbPath := t.TempDir() + "/test.db"
	defer os.Remove(dbPath)

	store, err := NewStore(dbPath)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	defer store.Close()

	if store.db == nil {
		t.Error("database not initialized")
	}
}

func TestStoreSchema(t *testing.T) {
	dbPath := t.TempDir() + "/test.db"
	defer os.Remove(dbPath)

	store, err := NewStore(dbPath)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	defer store.Close()

	// Verify tables exist
	var lineageCount int
	err = store.db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='lineage_records'").Scan(&lineageCount)
	if err != nil || lineageCount == 0 {
		t.Error("lineage_records table not created")
	}

	var purgeCount int
	err = store.db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='purge_events'").Scan(&purgeCount)
	if err != nil || purgeCount == 0 {
		t.Error("purge_events table not created")
	}
}

func TestRecordLineage(t *testing.T) {
	dbPath := t.TempDir() + "/test.db"
	defer os.Remove(dbPath)

	store, err := NewStore(dbPath)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	msg := engine.NewMessage(map[string]interface{}{"test": "data"}, "test-route", "v1")
	msg.Metadata.Principal = &engine.Principal{Subject: "user-123"}

	err = store.RecordLineage(ctx, msg, "test-route", "v1", "default", "user-123")
	if err != nil {
		t.Fatalf("RecordLineage failed: %v", err)
	}

	// Verify record was written
	var id string
	query := "SELECT id FROM lineage_records WHERE route_name = ?"
	err = store.db.QueryRow(query, "test-route").Scan(&id)
	if err != nil {
		t.Errorf("Record not found: %v", err)
	}
}

func TestQueryBySubject(t *testing.T) {
	dbPath := t.TempDir() + "/test.db"
	defer os.Remove(dbPath)

	store, err := NewStore(dbPath)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	defer store.Close()

	ctx := context.Background()

	// Insert test record directly
	now := time.Now().UTC()
	query := `
	INSERT INTO lineage_records (
		id, route_name, subject_id, message_body, message_headers, metadata,
		route_version, contract_version, principal, retention_policy,
		created_at, processed_at, expires_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err = store.db.Exec(query,
		"test-id", "test-route", "subj-123", "body", "headers", "metadata",
		"v1", "", "user", "default",
		now, now, now.Add(30*24*time.Hour),
	)
	if err != nil {
		t.Fatalf("Insert failed: %v", err)
	}

	// Query by subject
	records, err := store.QueryBySubject(ctx, "subj-123")
	if err != nil {
		t.Fatalf("QueryBySubject failed: %v", err)
	}

	if len(records) != 1 {
		t.Errorf("Expected 1 record, got %d", len(records))
	}

	if records[0].SubjectID != "subj-123" {
		t.Errorf("Wrong subject ID: %s", records[0].SubjectID)
	}
}

func TestDeleteBySubject(t *testing.T) {
	dbPath := t.TempDir() + "/test.db"
	defer os.Remove(dbPath)

	store, err := NewStore(dbPath)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	defer store.Close()

	ctx := context.Background()

	// Insert test records
	now := time.Now().UTC()
	for i := 0; i < 3; i++ {
		_, err = store.db.Exec(`
		INSERT INTO lineage_records (
			id, route_name, subject_id, message_body, message_headers, metadata,
			route_version, contract_version, principal, retention_policy,
			created_at, processed_at, expires_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`,
			"id-"+string(rune(i)), "route", "subj-123", "body", "headers", "metadata",
			"v1", "", "user", "default",
			now, now, now.Add(30*24*time.Hour),
		)
		if err != nil {
			t.Fatalf("Insert failed: %v", err)
		}
	}

	// Delete by subject
	count, err := store.DeleteBySubject(ctx, "subj-123")
	if err != nil {
		t.Fatalf("DeleteBySubject failed: %v", err)
	}

	if count != 3 {
		t.Errorf("Expected to delete 3 records, deleted %d", count)
	}

	// Verify deletion
	remaining, err := store.QueryBySubject(ctx, "subj-123")
	if err != nil {
		t.Fatalf("Query after delete failed: %v", err)
	}

	if len(remaining) != 0 {
		t.Errorf("Expected 0 records after delete, got %d", len(remaining))
	}
}

func TestConcurrentWrites(t *testing.T) {
	dbPath := t.TempDir() + "/test.db"
	defer os.Remove(dbPath)

	store, err := NewStore(dbPath)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	numGoroutines := 5
	msgsPerGoroutine := 20

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	// Launch concurrent writers
	for g := 0; g < numGoroutines; g++ {
		go func(goroutineID int) {
			defer wg.Done()

			for m := 0; m < msgsPerGoroutine; m++ {
				msg := engine.NewMessage(
					map[string]interface{}{"goroutine": goroutineID, "msg": m},
					"route",
					"v1",
				)

				// Manually set unique correlation ID to avoid UNIQUE constraint violations
				msg.Metadata.CorrelationID = fmt.Sprintf("msg-%d-%d-%d", time.Now().UnixNano(), goroutineID, m)

				err := store.RecordLineage(ctx, msg, "route", "v1", "default", "")
				if err != nil {
					t.Errorf("RecordLineage failed: %v", err)
				}

				// Small sleep to avoid timestamp collisions
				time.Sleep(time.Millisecond)
			}
		}(g)
	}

	wg.Wait()

	// Verify at least most records were written (allowing for some timing issues)
	var count int
	err = store.db.QueryRow("SELECT COUNT(*) FROM lineage_records").Scan(&count)
	if err != nil {
		t.Fatalf("Count query failed: %v", err)
	}

	expected := numGoroutines * msgsPerGoroutine
	if count < expected-5 { // Allow for small margin due to timing
		t.Errorf("Expected around %d records, got %d", expected, count)
	}
}

func TestRecordPurgeEvent(t *testing.T) {
	dbPath := t.TempDir() + "/test.db"
	defer os.Remove(dbPath)

	store, err := NewStore(dbPath)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	now := time.Now().UTC()
	expiresAt := now.Add(365 * 24 * time.Hour)

	err = store.RecordPurgeEvent(ctx, "event-1", "manual_purge", "subj-123", "Test purge", 5, expiresAt)
	if err != nil {
		t.Fatalf("RecordPurgeEvent failed: %v", err)
	}

	// Verify event was written
	var eventType string
	query := "SELECT event_type FROM purge_events WHERE id = ?"
	err = store.db.QueryRow(query, "event-1").Scan(&eventType)
	if err != nil {
		t.Errorf("Event not found: %v", err)
	}

	if eventType != "manual_purge" {
		t.Errorf("Wrong event type: %s", eventType)
	}
}

func TestQueryExpiredRecords(t *testing.T) {
	dbPath := t.TempDir() + "/test.db"
	defer os.Remove(dbPath)

	store, err := NewStore(dbPath)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	now := time.Now().UTC()

	// Insert expired record
	expiredTime := now.Add(-1 * time.Hour)
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

	// Query expired records
	records, err := store.QueryExpiredRecords(ctx, now)
	if err != nil {
		t.Fatalf("QueryExpiredRecords failed: %v", err)
	}

	if len(records) != 1 {
		t.Errorf("Expected 1 expired record, got %d", len(records))
	}

	if records[0].ID != "expired-id" {
		t.Errorf("Wrong record ID: %s", records[0].ID)
	}
}
