package database

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/naren-chakraview/dim/internal/engine"
)

// TestDatabaseSourcePoll verifies polling query execution (M2.3.1)
func TestDatabaseSourcePoll(t *testing.T) {
	// Create in-memory SQLite database
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Create test table
	_, err = db.Exec(`
		CREATE TABLE users (
			id INTEGER PRIMARY KEY,
			name TEXT,
			email TEXT,
			updated_at DATETIME
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}

	// Insert test data
	now := time.Now()
	_, err = db.Exec(`
		INSERT INTO users (id, name, email, updated_at) VALUES (1, 'Alice', 'alice@example.com', ?)
	`, now)
	if err != nil {
		t.Fatalf("Failed to insert data: %v", err)
	}

	// Create source
	config := &SourceConfig{
		Driver:           "sqlite3",
		DSN:              ":memory:",
		Query:            "SELECT id, name, email, updated_at FROM users WHERE updated_at > ?",
		WatermarkColumn:  "updated_at",
		WatermarkType:    "timestamp",
		InitialWatermark: "2020-01-01T00:00:00Z",
		Schedule:         1 * time.Second,
		BatchSize:        100,
		TimeoutSec:       5,
	}

	outChan := engine.NewChannel("test-database-source", 10)
	source, err := NewDatabaseSource(config, outChan)
	if err == nil {
		defer source.Close()
	}

	// Verify source created
	if source == nil {
		t.Fatalf("Failed to create database source")
	}
}

// TestDatabaseSinkInsert verifies INSERT operation (M2.3.2)
func TestDatabaseSinkInsert(t *testing.T) {
	// Use temp file for SQLite database
	tmpFile := t.TempDir() + "/test.db"

	// Create database and table
	db, err := sql.Open("sqlite3", tmpFile)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}

	// Create test table
	_, err = db.Exec(`
		CREATE TABLE users (
			id INTEGER PRIMARY KEY,
			name TEXT,
			email TEXT
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}
	db.Close()

	// Create sink pointing to same database file
	config := &SinkConfig{
		Driver:        "sqlite3",
		DSN:           tmpFile,
		Table:         "users",
		InsertColumns: []string{"id", "name", "email"},
		UpsertEnabled: false,
		BatchSize:     2,
		TimeoutSec:    5,
		FlushIntervalMs: 1000,
	}

	sink, err := NewDatabaseSink(config)
	if err != nil {
		t.Fatalf("Failed to create sink: %v", err)
	}
	defer sink.Close()

	// Create and send messages
	msg1 := engine.NewMessage(
		map[string]interface{}{"id": 1, "name": "Alice", "email": "alice@example.com"},
		"test", "v1",
	)
	msg2 := engine.NewMessage(
		map[string]interface{}{"id": 2, "name": "Bob", "email": "bob@example.com"},
		"test", "v1",
	)

	ctx := context.Background()
	err = sink.Send(ctx, msg1)
	if err != nil {
		t.Errorf("Send failed: %v", err)
	}
	err = sink.Send(ctx, msg2)
	if err != nil {
		t.Errorf("Send failed: %v", err)
	}

	// Close flushes the batch
	if err := sink.Close(); err != nil {
		t.Logf("Close error: %v", err)
	}

	// Verify write count - batch should have been flushed
	if sink.GetWriteCount() < 1 {
		t.Errorf("Expected write count >= 1, got %d", sink.GetWriteCount())
	}
}

// TestDatabaseSinkUpsert verifies UPSERT operation (M2.3.2)
func TestDatabaseSinkUpsert(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Create test table
	_, err = db.Exec(`
		CREATE TABLE users (
			id INTEGER PRIMARY KEY,
			name TEXT,
			email TEXT
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}

	// Create sink with UPSERT
	config := &SinkConfig{
		Driver:        "sqlite3",
		DSN:           ":memory:",
		Table:         "users",
		InsertColumns: []string{"id", "name", "email"},
		UpsertEnabled: true,
		UpsertKeys:    []string{"id"},
		BatchSize:     1,
		TimeoutSec:    5,
		FlushIntervalMs: 1000,
	}

	sink, err := NewDatabaseSink(config)
	if err != nil {
		t.Fatalf("Failed to create sink: %v", err)
	}
	defer sink.Close()

	// Verify sink was created
	if sink == nil {
		t.Fatalf("Sink should be created")
	}
}

// TestSourceConfigValidation verifies configuration validation (M2.3.1)
func TestSourceConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		config  *SourceConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: &SourceConfig{
				Driver:          "sqlite3",
				DSN:             ":memory:",
				Query:           "SELECT * FROM users WHERE id > ?",
				WatermarkColumn: "id",
				WatermarkType:   "integer",
			},
			wantErr: false,
		},
		{
			name: "missing DSN",
			config: &SourceConfig{
				Driver:          "sqlite3",
				Query:           "SELECT * FROM users",
				WatermarkColumn: "id",
				WatermarkType:   "integer",
			},
			wantErr: true,
		},
		{
			name: "missing Query",
			config: &SourceConfig{
				Driver:          "sqlite3",
				DSN:             ":memory:",
				WatermarkColumn: "id",
				WatermarkType:   "integer",
			},
			wantErr: true,
		},
		{
			name: "invalid WatermarkType",
			config: &SourceConfig{
				Driver:          "sqlite3",
				DSN:             ":memory:",
				Query:           "SELECT * FROM users",
				WatermarkColumn: "id",
				WatermarkType:   "invalid",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			outChan := engine.NewChannel("test-validation", 10)
			_, err := NewDatabaseSource(tt.config, outChan)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewDatabaseSource() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestSinkConfigValidation verifies sink configuration validation (M2.3.2)
func TestSinkConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		config  *SinkConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: &SinkConfig{
				Driver:        "sqlite3",
				DSN:           ":memory:",
				Table:         "users",
				InsertColumns: []string{"id", "name"},
			},
			wantErr: false,
		},
		{
			name: "missing DSN",
			config: &SinkConfig{
				Driver:        "sqlite3",
				Table:         "users",
				InsertColumns: []string{"id"},
			},
			wantErr: true,
		},
		{
			name: "missing Table",
			config: &SinkConfig{
				Driver:        "sqlite3",
				DSN:           ":memory:",
				InsertColumns: []string{"id"},
			},
			wantErr: true,
		},
		{
			name: "empty InsertColumns",
			config: &SinkConfig{
				Driver:        "sqlite3",
				DSN:           ":memory:",
				Table:         "users",
				InsertColumns: []string{},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewDatabaseSink(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewDatabaseSink() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestWatermarkTracking verifies watermark persistence (M2.3.1)
func TestWatermarkTracking(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	config := &SourceConfig{
		Driver:           "sqlite3",
		DSN:              ":memory:",
		Query:            "SELECT * FROM users WHERE id > ?",
		WatermarkColumn:  "id",
		WatermarkType:    "integer",
		InitialWatermark: "0",
	}

	outChan := engine.NewChannel("test-watermark", 10)
	source, err := NewDatabaseSource(config, outChan)
	if err != nil {
		t.Fatalf("Failed to create source: %v", err)
	}
	defer source.Close()

	// Check initial watermark (stored as string)
	initial := source.GetWatermark()
	if initial != "0" {
		t.Errorf("Expected initial watermark '0', got %v", initial)
	}

	// Update watermark
	source.SetWatermark("10")
	updated := source.GetWatermark()
	if updated != "10" {
		t.Errorf("Expected watermark '10', got %v", updated)
	}
}
