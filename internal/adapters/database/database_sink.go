package database

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/naren-chakraview/dim/internal/engine"
)

// DatabaseSink writes messages to a database table (M2.3.2)
// Supports both INSERT and UPSERT operations
type DatabaseSink struct {
	db         *sql.DB
	config     *SinkConfig
	batch      []*engine.Message
	batchMu    sync.Mutex
	writeCount int64
	ticker     *time.Ticker
	done       chan struct{}
	wg         sync.WaitGroup
}

// SinkConfig defines configuration for database sink
type SinkConfig struct {
	Driver        string   // postgres, mysql, sqlite
	DSN           string   // Data Source Name
	Table         string   // Target table name
	InsertColumns []string // Columns to insert
	UpsertEnabled bool     // Use UPSERT instead of INSERT
	UpsertKeys    []string // Key columns for upsert
	BatchSize     int      // Batch writes after N messages
	TimeoutSec    int      // Write timeout
	FlushIntervalMs int    // Flush batch every N ms
}

// NewDatabaseSink creates a new database sink
func NewDatabaseSink(config *SinkConfig) (*DatabaseSink, error) {
	if config.DSN == "" {
		return nil, fmt.Errorf("DSN is required")
	}
	if config.Table == "" {
		return nil, fmt.Errorf("Table is required")
	}
	if len(config.InsertColumns) == 0 {
		return nil, fmt.Errorf("InsertColumns is required")
	}

	if config.BatchSize == 0 {
		config.BatchSize = 50
	}
	if config.TimeoutSec == 0 {
		config.TimeoutSec = 10
	}
	if config.FlushIntervalMs == 0 {
		config.FlushIntervalMs = 5000
	}

	// Open database connection
	db, err := sql.Open(config.Driver, config.DSN)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Test connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	ds := &DatabaseSink{
		db:     db,
		config: config,
		batch:  make([]*engine.Message, 0, config.BatchSize),
		done:   make(chan struct{}),
		ticker: time.NewTicker(time.Duration(config.FlushIntervalMs) * time.Millisecond),
	}

	log.Printf("[INFO] Database sink created: driver=%s table=%s batch_size=%d", config.Driver, config.Table, config.BatchSize)

	// Start background flush goroutine
	ds.wg.Add(1)
	go ds.flushLoop()

	return ds, nil
}

// Send adds a message to the batch and flushes if full
func (ds *DatabaseSink) Send(ctx context.Context, msg *engine.Message) error {
	ds.batchMu.Lock()
	defer ds.batchMu.Unlock()

	ds.batch = append(ds.batch, msg)

	// Flush if batch is full
	if len(ds.batch) >= ds.config.BatchSize {
		return ds.flushBatch(ctx)
	}

	return nil
}

// flushLoop periodically flushes batches
func (ds *DatabaseSink) flushLoop() {
	defer ds.wg.Done()

	for {
		select {
		case <-ds.done:
			// Final flush on close
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(ds.config.TimeoutSec)*time.Second)
			ds.batchMu.Lock()
			_ = ds.flushBatch(ctx)
			ds.batchMu.Unlock()
			cancel()
			return

		case <-ds.ticker.C:
			ds.batchMu.Lock()
			if len(ds.batch) > 0 {
				ctx, cancel := context.WithTimeout(context.Background(), time.Duration(ds.config.TimeoutSec)*time.Second)
				_ = ds.flushBatch(ctx)
				cancel()
			}
			ds.batchMu.Unlock()
		}
	}
}

// flushBatch writes all messages in the batch to the database
// Must be called with batchMu locked
func (ds *DatabaseSink) flushBatch(ctx context.Context) error {
	if len(ds.batch) == 0 {
		return nil
	}

	// Build parameterized query
	query := ds.buildInsertQuery(len(ds.batch))

	// Flatten values
	values := make([]interface{}, 0, len(ds.batch)*len(ds.config.InsertColumns))
	for _, msg := range ds.batch {
		rowMap := msg.Body.(map[string]interface{})
		for _, col := range ds.config.InsertColumns {
			values = append(values, rowMap[col])
		}
	}

	// Execute insert
	_, err := ds.db.ExecContext(ctx, query, values...)
	if err != nil {
		return fmt.Errorf("batch insert failed: %w", err)
	}

	writeCount := len(ds.batch)
	ds.writeCount += int64(writeCount)

	log.Printf("[DEBUG] Batch flushed: %d rows (total: %d)", writeCount, ds.writeCount)

	// Clear batch
	ds.batch = ds.batch[:0]

	return nil
}

// buildInsertQuery builds a parameterized INSERT or UPSERT query
func (ds *DatabaseSink) buildInsertQuery(rowCount int) string {
	var query string

	// Build column list
	columnList := ""
	for i, col := range ds.config.InsertColumns {
		if i > 0 {
			columnList += ", "
		}
		columnList += col
	}

	// Build value placeholders
	valuePlaceholders := ""
	for r := 0; r < rowCount; r++ {
		if r > 0 {
			valuePlaceholders += ", "
		}
		valuePlaceholders += "("
		for c := 0; c < len(ds.config.InsertColumns); c++ {
			if c > 0 {
				valuePlaceholders += ", "
			}
			valuePlaceholders += fmt.Sprintf("$%d", r*len(ds.config.InsertColumns)+c+1)
		}
		valuePlaceholders += ")"
	}

	if ds.config.UpsertEnabled {
		// UPSERT (INSERT ... ON CONFLICT ... DO UPDATE)
		conflictColumns := ""
		for i, key := range ds.config.UpsertKeys {
			if i > 0 {
				conflictColumns += ", "
			}
			conflictColumns += key
		}

		updateSet := ""
		for i, col := range ds.config.InsertColumns {
			if i > 0 {
				updateSet += ", "
			}
			updateSet += fmt.Sprintf("%s = EXCLUDED.%s", col, col)
		}

		query = fmt.Sprintf(
			"INSERT INTO %s (%s) VALUES %s ON CONFLICT (%s) DO UPDATE SET %s",
			ds.config.Table, columnList, valuePlaceholders, conflictColumns, updateSet,
		)
	} else {
		// Regular INSERT
		query = fmt.Sprintf(
			"INSERT INTO %s (%s) VALUES %s",
			ds.config.Table, columnList, valuePlaceholders,
		)
	}

	return query
}

// Close flushes remaining batch and closes the sink
func (ds *DatabaseSink) Close() error {
	ds.ticker.Stop()

	// Only close once
	select {
	case <-ds.done:
		// Already closed
	default:
		close(ds.done)
	}

	ds.wg.Wait()

	if ds.db != nil {
		return ds.db.Close()
	}
	return nil
}

// GetWriteCount returns total messages written
func (ds *DatabaseSink) GetWriteCount() int64 {
	return ds.writeCount
}

// HealthCheck verifies database connection
func (ds *DatabaseSink) HealthCheck(ctx context.Context) error {
	return ds.db.PingContext(ctx)
}
