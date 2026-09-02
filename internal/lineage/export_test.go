package lineage

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"
)

func TestExportCSV(t *testing.T) {
	dbPath := t.TempDir() + "/test.db"
	defer os.Remove(dbPath)

	store, err := NewStore(dbPath)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	now := time.Now().UTC()

	// Insert test records
	for i := 0; i < 2; i++ {
		_, err := store.db.Exec(`
		INSERT INTO lineage_records (
			id, route_name, subject_id, message_body, message_headers, metadata,
			route_version, contract_version, principal, retention_policy,
			created_at, processed_at, expires_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`,
			"id-"+string(rune(i)),
			"route",
			"subj-"+string(rune(i)),
			"body",
			"headers",
			"metadata",
			"v1",
			"",
			"user",
			"default",
			now,
			now,
			now.Add(30*24*time.Hour),
		)
		if err != nil {
			t.Fatalf("Insert failed: %v", err)
		}
	}

	// Export to CSV
	var buf bytes.Buffer
	err = Export(ctx, store, "csv", now.Add(-1*time.Hour), now.Add(1*time.Hour), &buf)
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	// Parse CSV
	reader := csv.NewReader(strings.NewReader(buf.String()))
	rows, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("CSV parsing failed: %v", err)
	}

	// Verify header and records
	if len(rows) != 3 { // header + 2 records
		t.Errorf("Expected 3 rows (header + 2 records), got %d", len(rows))
	}

	if rows[0][0] != "id" {
		t.Errorf("Expected 'id' in header, got %q", rows[0][0])
	}

	if rows[1][1] != "route" {
		t.Errorf("Expected 'route' in route_name column, got %q", rows[1][1])
	}
}

func TestExportNDJSON(t *testing.T) {
	dbPath := t.TempDir() + "/test.db"
	defer os.Remove(dbPath)

	store, err := NewStore(dbPath)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	now := time.Now().UTC()

	// Insert test record
	_, err = store.db.Exec(`
	INSERT INTO lineage_records (
		id, route_name, subject_id, message_body, message_headers, metadata,
		route_version, contract_version, principal, retention_policy,
		created_at, processed_at, expires_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		"test-id",
		"test-route",
		"test-subject",
		"test-body",
		"test-headers",
		"test-metadata",
		"v1",
		"",
		"user",
		"default",
		now,
		now,
		now.Add(30*24*time.Hour),
	)
	if err != nil {
		t.Fatalf("Insert failed: %v", err)
	}

	// Export to NDJSON
	var buf bytes.Buffer
	err = Export(ctx, store, "ndjson", now.Add(-1*time.Hour), now.Add(1*time.Hour), &buf)
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	// Parse NDJSON
	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 1 {
		t.Errorf("Expected 1 line, got %d", len(lines))
	}

	var record LineageRecord
	err = json.Unmarshal([]byte(lines[0]), &record)
	if err != nil {
		t.Fatalf("JSON parsing failed: %v", err)
	}

	if record.ID != "test-id" {
		t.Errorf("Expected ID 'test-id', got %q", record.ID)
	}

	if record.RouteName != "test-route" {
		t.Errorf("Expected route 'test-route', got %q", record.RouteName)
	}
}

func TestExportEmptyRecords(t *testing.T) {
	dbPath := t.TempDir() + "/test.db"
	defer os.Remove(dbPath)

	store, err := NewStore(dbPath)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	now := time.Now().UTC()

	// Export with no records
	var buf bytes.Buffer
	err = Export(ctx, store, "csv", now.Add(-1*time.Hour), now.Add(1*time.Hour), &buf)
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	// Should only have header
	reader := csv.NewReader(strings.NewReader(buf.String()))
	rows, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("CSV parsing failed: %v", err)
	}

	if len(rows) != 1 {
		t.Errorf("Expected 1 row (header only), got %d", len(rows))
	}
}

func TestExportInvalidFormat(t *testing.T) {
	dbPath := t.TempDir() + "/test.db"
	defer os.Remove(dbPath)

	store, err := NewStore(dbPath)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	now := time.Now().UTC()

	var buf bytes.Buffer
	err = Export(ctx, store, "invalid-format", now, now, &buf)
	if err == nil {
		t.Error("Expected error for invalid format")
	}

	if !strings.Contains(err.Error(), "unsupported export format") {
		t.Errorf("Expected 'unsupported export format' error, got %q", err.Error())
	}
}
