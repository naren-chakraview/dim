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

// TriggerBasedCDCSource polls changelog table for database changes (M2.3.4a)
// Captures before/after states from trigger-written changelog
type TriggerBasedCDCSource struct {
	db                *sql.DB
	config            *TriggerCDCConfig
	outChan           *engine.Channel
	closed            chan struct{}
	wg                sync.WaitGroup
	currentWatermark  interface{}
	mu                sync.RWMutex
	lastPollTime      time.Time
}

// TriggerCDCConfig defines configuration for trigger-based CDC
type TriggerCDCConfig struct {
	Driver           string        // postgres, mysql, sqlite
	DSN              string        // Data Source Name
	ChangelogTable   string        // Changelog table name (e.g., dim_cdc_changelog)
	WatermarkColumn  string        // Column tracking change time (e.g., changed_at)
	WatermarkType    string        // "timestamp" or "integer"
	InitialWatermark string        // Initial watermark value
	Schedule         time.Duration // Poll interval
	BatchSize        int           // Rows per query
	TimeoutSec       int           // Query timeout
}

// NewTriggerBasedCDCSource creates a new trigger-based CDC source
func NewTriggerBasedCDCSource(config *TriggerCDCConfig, outChan *engine.Channel) (*TriggerBasedCDCSource, error) {
	if config.DSN == "" {
		return nil, fmt.Errorf("DSN is required")
	}
	if config.ChangelogTable == "" {
		return nil, fmt.Errorf("ChangelogTable is required")
	}
	if config.WatermarkColumn == "" {
		return nil, fmt.Errorf("WatermarkColumn is required")
	}
	if config.WatermarkType != "timestamp" && config.WatermarkType != "integer" {
		return nil, fmt.Errorf("WatermarkType must be 'timestamp' or 'integer'")
	}

	if config.Schedule == 0 {
		config.Schedule = 30 * time.Second
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

	source := &TriggerBasedCDCSource{
		db:               db,
		config:           config,
		outChan:          outChan,
		closed:           make(chan struct{}),
		currentWatermark: config.InitialWatermark,
		lastPollTime:     time.Now().UTC(),
	}

	log.Printf("[INFO] Trigger-based CDC source created: table=%s watermark=%s schedule=%v", config.ChangelogTable, config.WatermarkColumn, config.Schedule)

	return source, nil
}

// Start begins polling the changelog table on schedule
func (s *TriggerBasedCDCSource) Start(ctx context.Context) error {
	s.wg.Add(1)
	defer s.wg.Done()

	ticker := time.NewTicker(s.config.Schedule)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			close(s.closed)
			return ctx.Err()

		case <-ticker.C:
			if err := s.pollChangelog(ctx); err != nil {
				log.Printf("[WARN] Changelog poll failed: %v", err)
			}
			s.lastPollTime = time.Now().UTC()
		}
	}
}

// pollChangelog executes one query and sends CDC events as messages
func (s *TriggerBasedCDCSource) pollChangelog(ctx context.Context) error {
	queryCtx, cancel := context.WithTimeout(ctx, time.Duration(s.config.TimeoutSec)*time.Second)
	defer cancel()

	// Build parameterized query for changelog table
	query := fmt.Sprintf(`
		SELECT table_name, operation, record_id, changed_at, before_state, after_state
		FROM %s
		WHERE %s > $1
		ORDER BY %s ASC
		LIMIT $2
	`, s.config.ChangelogTable, s.config.WatermarkColumn, s.config.WatermarkColumn)

	s.mu.RLock()
	currentWatermark := s.currentWatermark
	s.mu.RUnlock()

	rows, err := s.db.QueryContext(queryCtx, query, currentWatermark, s.config.BatchSize)
	if err != nil {
		return fmt.Errorf("changelog query failed: %w", err)
	}
	defer rows.Close()

	rowCount := 0
	maxWatermark := currentWatermark

	for rows.Next() {
		var (
			tableName   string
			operation   string
			recordID    interface{}
			changedAt   interface{}
			beforeState sql.NullString
			afterState  sql.NullString
		)

		if err := rows.Scan(&tableName, &operation, &recordID, &changedAt, &beforeState, &afterState); err != nil {
			return fmt.Errorf("failed to scan row: %w", err)
		}

		// Build CDC event message
		cdcEvent := map[string]interface{}{
			"table":      tableName,
			"operation":  operation,
			"record_id":  recordID,
			"changed_at": changedAt,
		}

		// Parse JSON states if available
		if beforeState.Valid {
			cdcEvent["before"] = beforeState.String
		}
		if afterState.Valid {
			cdcEvent["after"] = afterState.String
		}

		// Create message
		msg := engine.NewMessage(cdcEvent, "database-cdc-trigger", "")

		// Add CDC facet to metadata
		msg.Metadata.Stage = fmt.Sprintf("cdc_operation=%s table=%s watermark=%v", operation, tableName, changedAt)

		// Send to output channel
		if err := s.outChan.Send(queryCtx, msg); err != nil {
			return fmt.Errorf("failed to send message: %w", err)
		}

		maxWatermark = changedAt
		rowCount++
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("row iteration error: %w", err)
	}

	// Update watermark if rows were processed
	if rowCount > 0 {
		s.mu.Lock()
		s.currentWatermark = maxWatermark
		s.mu.Unlock()

		log.Printf("[DEBUG] Changelog poll complete: %d events, watermark=%v", rowCount, maxWatermark)
	}

	return nil
}

// Close closes the CDC source
func (s *TriggerBasedCDCSource) Close() error {
	if s.db != nil {
		if err := s.db.Close(); err != nil {
			return fmt.Errorf("failed to close database: %w", err)
		}
	}
	select {
	case <-s.closed:
	default:
	}
	s.wg.Wait()
	return nil
}

// GetWatermark returns the current watermark value
func (s *TriggerBasedCDCSource) GetWatermark() interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.currentWatermark
}

// SetWatermark sets the watermark value (for resumption)
func (s *TriggerBasedCDCSource) SetWatermark(watermark interface{}) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.currentWatermark = watermark
}

// HealthCheck verifies database connection
func (s *TriggerBasedCDCSource) HealthCheck(ctx context.Context) error {
	return s.db.PingContext(ctx)
}

// GetLastPollTime returns when the last poll occurred
func (s *TriggerBasedCDCSource) GetLastPollTime() time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.lastPollTime
}
