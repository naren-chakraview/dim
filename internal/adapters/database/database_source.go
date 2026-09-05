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

// DatabaseSource is a polling-based database source adapter (M2.3.1)
// Executes parameterized queries with watermark-based incremental polling
type DatabaseSource struct {
	db                *sql.DB
	config            *SourceConfig
	outChan           *engine.Channel
	closed            chan struct{}
	wg                sync.WaitGroup
	currentWatermark  interface{}
	mu                sync.RWMutex
	lastPollTime      time.Time
}

// SourceConfig defines configuration for database polling source
type SourceConfig struct {
	Driver           string        // postgres, mysql, sqlite
	DSN              string        // Data Source Name (connection string)
	Query            string        // Parameterized query with $1 placeholder
	WatermarkColumn  string        // Column to track for incremental pulls
	WatermarkType    string        // "timestamp" or "integer"
	InitialWatermark string        // Initial watermark value
	Schedule         time.Duration // Poll interval
	BatchSize        int           // Rows per query
	TimeoutSec       int           // Query timeout
}

// NewDatabaseSource creates a new database polling source
func NewDatabaseSource(config *SourceConfig, outChan *engine.Channel) (*DatabaseSource, error) {
	if config.DSN == "" {
		return nil, fmt.Errorf("DSN is required")
	}
	if config.Query == "" {
		return nil, fmt.Errorf("Query is required")
	}
	if config.WatermarkColumn == "" {
		return nil, fmt.Errorf("WatermarkColumn is required")
	}
	if config.WatermarkType != "timestamp" && config.WatermarkType != "integer" {
		return nil, fmt.Errorf("WatermarkType must be 'timestamp' or 'integer'")
	}

	if config.Schedule == 0 {
		config.Schedule = 5 * time.Minute
	}
	if config.BatchSize == 0 {
		config.BatchSize = 100
	}
	if config.TimeoutSec == 0 {
		config.TimeoutSec = 30
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

	ds := &DatabaseSource{
		db:               db,
		config:           config,
		outChan:          outChan,
		closed:           make(chan struct{}),
		currentWatermark: config.InitialWatermark,
		lastPollTime:     time.Now().UTC(),
	}

	log.Printf("[INFO] Database source created: driver=%s query_timeout=%ds schedule=%v", config.Driver, config.TimeoutSec, config.Schedule)

	return ds, nil
}

// Start begins polling the database on schedule
func (ds *DatabaseSource) Start(ctx context.Context) error {
	ds.wg.Add(1)
	defer ds.wg.Done()

	ticker := time.NewTicker(ds.config.Schedule)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			close(ds.closed)
			return ctx.Err()

		case <-ticker.C:
			// Execute poll
			if err := ds.poll(ctx); err != nil {
				log.Printf("[WARN] Poll failed: %v", err)
				// Continue polling on next tick
			}
			ds.lastPollTime = time.Now().UTC()
		}
	}
}

// poll executes one query and sends results as messages
func (ds *DatabaseSource) poll(ctx context.Context) error {
	queryCtx, cancel := context.WithTimeout(ctx, time.Duration(ds.config.TimeoutSec)*time.Second)
	defer cancel()

	// Execute parameterized query with watermark
	ds.mu.RLock()
	currentWatermark := ds.currentWatermark
	ds.mu.RUnlock()

	rows, err := ds.db.QueryContext(queryCtx, ds.config.Query, currentWatermark)
	if err != nil {
		return fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	// Get column names
	columns, err := rows.Columns()
	if err != nil {
		return fmt.Errorf("failed to get columns: %w", err)
	}

	rowCount := 0
	maxWatermark := currentWatermark

	// Process each row
	for rows.Next() {
		// Scan row into generic interface slice
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range columns {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return fmt.Errorf("failed to scan row: %w", err)
		}

		// Convert row to JSON object
		rowMap := make(map[string]interface{})
		for i, col := range columns {
			var v interface{}
			val := values[i]
			b, ok := val.([]byte)
			if ok {
				v = string(b)
			} else {
				v = val
			}
			rowMap[col] = v

			// Track max watermark
			if col == ds.config.WatermarkColumn {
				maxWatermark = v
			}
		}

		// Create message
		msg := engine.NewMessage(rowMap, "database-source", "")

		// Add watermark to metadata
		if ds.config.WatermarkColumn != "" {
			msg.Metadata.Stage = fmt.Sprintf("watermark=%v", maxWatermark)
		}

		// Send to output channel
		if err := ds.outChan.Send(queryCtx, msg); err != nil {
			return fmt.Errorf("failed to send message: %w", err)
		}

		rowCount++
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("row iteration error: %w", err)
	}

	// Update watermark if rows were processed
	if rowCount > 0 {
		ds.mu.Lock()
		ds.currentWatermark = maxWatermark
		ds.mu.Unlock()

		log.Printf("[DEBUG] Poll complete: %d rows, watermark=%v", rowCount, maxWatermark)
	}

	return nil
}

// Close closes the database source
func (ds *DatabaseSource) Close() error {
	if ds.db != nil {
		if err := ds.db.Close(); err != nil {
			return fmt.Errorf("failed to close database: %w", err)
		}
	}
	select {
	case <-ds.closed:
	default:
		// closed channel not yet available; Start() was not called
	}
	ds.wg.Wait()
	return nil
}

// GetWatermark returns the current watermark value
func (ds *DatabaseSource) GetWatermark() interface{} {
	ds.mu.RLock()
	defer ds.mu.RUnlock()
	return ds.currentWatermark
}

// SetWatermark sets the watermark value (for resumption)
func (ds *DatabaseSource) SetWatermark(watermark interface{}) {
	ds.mu.Lock()
	defer ds.mu.Unlock()
	ds.currentWatermark = watermark
}

// HealthCheck verifies database connection
func (ds *DatabaseSource) HealthCheck(ctx context.Context) error {
	return ds.db.PingContext(ctx)
}

// GetLastPollTime returns when the last poll occurred
func (ds *DatabaseSource) GetLastPollTime() time.Time {
	ds.mu.RLock()
	defer ds.mu.RUnlock()
	return ds.lastPollTime
}
