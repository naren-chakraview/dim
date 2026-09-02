package lineage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/naren-chakraview/dim/internal/engine"
	_ "modernc.org/sqlite"
)

// LineageRecord represents a single message lineage entry
type LineageRecord struct {
	ID              string    `json:"id"`
	RouteName       string    `json:"route_name"`
	SubjectID       string    `json:"subject_id"`
	MessageBody     string    `json:"message_body"`
	MessageHeaders  string    `json:"message_headers"`
	Metadata        string    `json:"metadata"`
	RouteVersion    string    `json:"route_version"`
	ContractVersion string    `json:"contract_version"`
	Principal       string    `json:"principal"`
	RetentionPolicy string    `json:"retention_policy"`
	CreatedAt       time.Time `json:"created_at"`
	ProcessedAt     time.Time `json:"processed_at"`
	ExpiresAt       time.Time `json:"expires_at"`
}

// PurgeEvent represents a purge/reap event for evidence logging
type PurgeEvent struct {
	ID        string    `json:"id"`
	EventType string    `json:"event_type"` // "manual_purge", "auto_reap"
	SubjectID string    `json:"subject_id"` // NULL if bulk purge
	PurgedCount int      `json:"purged_count"`
	Reason    string    `json:"reason"`
	PurgedAt  time.Time `json:"purged_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

// writeRequest is an internal type for queueing write operations
type writeRequest struct {
	op     string                 // "insert_lineage", "delete_lineage", "insert_purge_event"
	data   map[string]interface{} // operation parameters
	respCh chan error             // return error to caller
}

// Store manages embedded SQLite lineage storage with single-writer pattern
type Store struct {
	dbPath    string
	db        *sql.DB
	writeQ    chan writeRequest // Single-writer queue
	writeDone chan struct{}     // Signal writer goroutine to stop
	writeErr  chan error        // Capture writer goroutine errors
	wg        sync.WaitGroup
}

// NewStore initializes the lineage store with embedded SQLite
func NewStore(dbPath string) (*Store, error) {
	// Open database with WAL mode for concurrent reads + single writer
	db, err := sql.Open("sqlite", fmt.Sprintf("file:%s?cache=shared&mode=rwc&_journal=WAL", dbPath))
	if err != nil {
		return nil, fmt.Errorf("failed to open SQLite database: %w", err)
	}

	// Set connection pool parameters
	db.SetMaxOpenConns(1) // Single writer enforced here
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(0)

	// Initialize schema
	if err := initSchema(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	s := &Store{
		dbPath:    dbPath,
		db:        db,
		writeQ:    make(chan writeRequest, 100), // Buffered queue
		writeDone: make(chan struct{}),
		writeErr:  make(chan error, 1),
	}

	// Start single-writer goroutine
	s.wg.Add(1)
	go s.writerLoop()

	return s, nil
}

// initSchema creates the schema if it doesn't exist
func initSchema(db *sql.DB) error {
	schema := `
	CREATE TABLE IF NOT EXISTS lineage_records (
		id TEXT PRIMARY KEY,
		route_name TEXT NOT NULL,
		subject_id TEXT,
		message_body TEXT NOT NULL,
		message_headers TEXT,
		metadata TEXT,
		route_version TEXT NOT NULL,
		contract_version TEXT,
		principal TEXT,
		retention_policy TEXT NOT NULL,
		created_at DATETIME NOT NULL,
		processed_at DATETIME NOT NULL,
		expires_at DATETIME NOT NULL
	);

	CREATE INDEX IF NOT EXISTS idx_route_time ON lineage_records (route_name, created_at DESC);
	CREATE INDEX IF NOT EXISTS idx_subject_id ON lineage_records (subject_id);
	CREATE INDEX IF NOT EXISTS idx_expires_at ON lineage_records (expires_at);
	CREATE INDEX IF NOT EXISTS idx_subject_id_route ON lineage_records (subject_id, route_name);

	CREATE TABLE IF NOT EXISTS purge_events (
		id TEXT PRIMARY KEY,
		event_type TEXT NOT NULL,
		subject_id TEXT,
		purged_count INT,
		reason TEXT,
		purged_at DATETIME NOT NULL,
		expires_at DATETIME NOT NULL
	);

	CREATE INDEX IF NOT EXISTS idx_purge_purged_at ON purge_events (purged_at);
	CREATE INDEX IF NOT EXISTS idx_purge_expires_at ON purge_events (expires_at);
	`

	if _, err := db.Exec(schema); err != nil {
		return fmt.Errorf("failed to create schema: %w", err)
	}

	return nil
}

// writerLoop is the single-writer goroutine that processes all write operations
func (s *Store) writerLoop() {
	defer s.wg.Done()
	defer log.Printf("[DEBUG] lineage store writer goroutine exiting")

	for {
		select {
		case <-s.writeDone:
			// Drain remaining requests on close
			for {
				select {
				case req := <-s.writeQ:
					req.respCh <- fmt.Errorf("store closed")
				default:
					return
				}
			}

		case req := <-s.writeQ:
			err := s.executeWrite(req)
			req.respCh <- err
		}
	}
}

// executeWrite performs the actual write operation
func (s *Store) executeWrite(req writeRequest) error {
	switch req.op {
	case "insert_lineage":
		return s.insertLineage(
			req.data["id"].(string),
			req.data["route_name"].(string),
			req.data["subject_id"].(string),
			req.data["message_body"].(string),
			req.data["message_headers"].(string),
			req.data["metadata"].(string),
			req.data["route_version"].(string),
			req.data["contract_version"].(string),
			req.data["principal"].(string),
			req.data["retention_policy"].(string),
			req.data["created_at"].(time.Time),
			req.data["processed_at"].(time.Time),
			req.data["expires_at"].(time.Time),
		)

	case "delete_by_subject":
		return s.deleteBySubjectDB(req.data["subject_id"].(string))

	case "insert_purge_event":
		return s.insertPurgeEventDB(
			req.data["id"].(string),
			req.data["event_type"].(string),
			req.data["subject_id"].(string),
			req.data["purged_count"].(int),
			req.data["reason"].(string),
			req.data["purged_at"].(time.Time),
			req.data["expires_at"].(time.Time),
		)

	default:
		return fmt.Errorf("unknown write operation: %s", req.op)
	}
}

// insertLineage performs the actual insert
func (s *Store) insertLineage(id, routeName, subjectID, msgBody, msgHeaders, metadata, routeVersion, contractVersion, principal, retentionPolicy string, createdAt, processedAt, expiresAt time.Time) error {
	query := `
	INSERT INTO lineage_records (
		id, route_name, subject_id, message_body, message_headers, metadata,
		route_version, contract_version, principal, retention_policy,
		created_at, processed_at, expires_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := s.db.Exec(query,
		id, routeName, subjectID, msgBody, msgHeaders, metadata,
		routeVersion, contractVersion, principal, retentionPolicy,
		createdAt, processedAt, expiresAt,
	)
	return err
}

// deleteBySubjectDB performs the actual delete by subject
func (s *Store) deleteBySubjectDB(subjectID string) error {
	_, err := s.db.Exec("DELETE FROM lineage_records WHERE subject_id = ?", subjectID)
	return err
}

// insertPurgeEventDB performs the actual purge event insert
func (s *Store) insertPurgeEventDB(id, eventType, subjectID string, purgedCount int, reason string, purgedAt, expiresAt time.Time) error {
	query := `
	INSERT INTO purge_events (
		id, event_type, subject_id, purged_count, reason, purged_at, expires_at
	) VALUES (?, ?, ?, ?, ?, ?, ?)
	`

	_, err := s.db.Exec(query, id, eventType, subjectID, purgedCount, reason, purgedAt, expiresAt)
	return err
}

// RecordLineage queues a lineage record write via the single-writer pattern
func (s *Store) RecordLineage(
	ctx context.Context,
	msg *engine.Message,
	routeName, routeVersion, retentionPolicy, principal string,
) error {
	// Serialize message data
	msgBody, _ := json.Marshal(msg.Body)
	msgHeaders, _ := json.Marshal(msg.Headers)
	metadata, _ := json.Marshal(msg.Metadata)

	// Extract contract version and principal if available
	contractVersion := msg.Metadata.ContractVersion
	principalStr := ""
	if msg.Metadata.Principal != nil {
		principalStr = msg.Metadata.Principal.Subject
	}

	// Calculate expires_at based on retention policy TTL
	// (This would be calculated by caller with RetentionResolver)
	expiresAt := time.Now().Add(30 * 24 * time.Hour) // Default: 30 days

	req := writeRequest{
		op: "insert_lineage",
		data: map[string]interface{}{
			"id":                msg.Metadata.CorrelationID,
			"route_name":        routeName,
			"subject_id":        "", // Will be set by caller
			"message_body":      string(msgBody),
			"message_headers":   string(msgHeaders),
			"metadata":          string(metadata),
			"route_version":     routeVersion,
			"contract_version":  contractVersion,
			"principal":         principalStr,
			"retention_policy":  retentionPolicy,
			"created_at":        msg.Metadata.IngestedAt,
			"processed_at":      time.Now().UTC(),
			"expires_at":        expiresAt,
		},
		respCh: make(chan error, 1),
	}

	select {
	case s.writeQ <- req:
		return <-req.respCh
	case <-ctx.Done():
		return ctx.Err()
	}
}

// QueryBySubject queries lineage records by subject ID (read-only, no lock)
func (s *Store) QueryBySubject(ctx context.Context, subjectID string) ([]LineageRecord, error) {
	query := `
	SELECT id, route_name, subject_id, message_body, message_headers, metadata,
	       route_version, contract_version, principal, retention_policy,
	       created_at, processed_at, expires_at
	FROM lineage_records
	WHERE subject_id = ?
	ORDER BY created_at ASC
	`

	rows, err := s.db.QueryContext(ctx, query, subjectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []LineageRecord
	for rows.Next() {
		var r LineageRecord
		if err := rows.Scan(
			&r.ID, &r.RouteName, &r.SubjectID, &r.MessageBody, &r.MessageHeaders, &r.Metadata,
			&r.RouteVersion, &r.ContractVersion, &r.Principal, &r.RetentionPolicy,
			&r.CreatedAt, &r.ProcessedAt, &r.ExpiresAt,
		); err != nil {
			return nil, err
		}
		records = append(records, r)
	}

	return records, rows.Err()
}

// QueryByRoute queries lineage records by route within a time range
func (s *Store) QueryByRoute(ctx context.Context, routeName string, since, until time.Time) ([]LineageRecord, error) {
	query := `
	SELECT id, route_name, subject_id, message_body, message_headers, metadata,
	       route_version, contract_version, principal, retention_policy,
	       created_at, processed_at, expires_at
	FROM lineage_records
	WHERE route_name = ? AND created_at >= ? AND created_at <= ?
	ORDER BY created_at ASC
	`

	rows, err := s.db.QueryContext(ctx, query, routeName, since, until)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []LineageRecord
	for rows.Next() {
		var r LineageRecord
		if err := rows.Scan(
			&r.ID, &r.RouteName, &r.SubjectID, &r.MessageBody, &r.MessageHeaders, &r.Metadata,
			&r.RouteVersion, &r.ContractVersion, &r.Principal, &r.RetentionPolicy,
			&r.CreatedAt, &r.ProcessedAt, &r.ExpiresAt,
		); err != nil {
			return nil, err
		}
		records = append(records, r)
	}

	return records, rows.Err()
}

// DeleteBySubject queues a delete by subject ID
func (s *Store) DeleteBySubject(ctx context.Context, subjectID string) (int, error) {
	// Get count before deletion for return value
	countQuery := "SELECT COUNT(*) FROM lineage_records WHERE subject_id = ?"
	var count int
	if err := s.db.QueryRowContext(ctx, countQuery, subjectID).Scan(&count); err != nil {
		return 0, err
	}

	// Queue deletion
	req := writeRequest{
		op: "delete_by_subject",
		data: map[string]interface{}{
			"subject_id": subjectID,
		},
		respCh: make(chan error, 1),
	}

	select {
	case s.writeQ <- req:
		if err := <-req.respCh; err != nil {
			return 0, err
		}
		return count, nil
	case <-ctx.Done():
		return 0, ctx.Err()
	}
}

// QueryExpiredRecords queries records that have expired
func (s *Store) QueryExpiredRecords(ctx context.Context, until time.Time) ([]LineageRecord, error) {
	query := `
	SELECT id, route_name, subject_id, message_body, message_headers, metadata,
	       route_version, contract_version, principal, retention_policy,
	       created_at, processed_at, expires_at
	FROM lineage_records
	WHERE expires_at <= ?
	ORDER BY expires_at ASC
	`

	rows, err := s.db.QueryContext(ctx, query, until)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []LineageRecord
	for rows.Next() {
		var r LineageRecord
		if err := rows.Scan(
			&r.ID, &r.RouteName, &r.SubjectID, &r.MessageBody, &r.MessageHeaders, &r.Metadata,
			&r.RouteVersion, &r.ContractVersion, &r.Principal, &r.RetentionPolicy,
			&r.CreatedAt, &r.ProcessedAt, &r.ExpiresAt,
		); err != nil {
			return nil, err
		}
		records = append(records, r)
	}

	return records, rows.Err()
}

// RecordPurgeEvent queues a purge event record
func (s *Store) RecordPurgeEvent(ctx context.Context, eventID, eventType, subjectID, reason string, purgedCount int, expiresAt time.Time) error {
	req := writeRequest{
		op: "insert_purge_event",
		data: map[string]interface{}{
			"id":           eventID,
			"event_type":   eventType,
			"subject_id":   subjectID,
			"purged_count": purgedCount,
			"reason":       reason,
			"purged_at":    time.Now().UTC(),
			"expires_at":   expiresAt,
		},
		respCh: make(chan error, 1),
	}

	select {
	case s.writeQ <- req:
		return <-req.respCh
	case <-ctx.Done():
		return ctx.Err()
	}
}

// QueryPurgeEvents queries purge events within a time range
func (s *Store) QueryPurgeEvents(ctx context.Context, since, until time.Time) ([]PurgeEvent, error) {
	query := `
	SELECT id, event_type, subject_id, purged_count, reason, purged_at, expires_at
	FROM purge_events
	WHERE purged_at >= ? AND purged_at <= ?
	ORDER BY purged_at ASC
	`

	rows, err := s.db.QueryContext(ctx, query, since, until)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []PurgeEvent
	for rows.Next() {
		var e PurgeEvent
		if err := rows.Scan(
			&e.ID, &e.EventType, &e.SubjectID, &e.PurgedCount, &e.Reason, &e.PurgedAt, &e.ExpiresAt,
		); err != nil {
			return nil, err
		}
		events = append(events, e)
	}

	return events, rows.Err()
}

// Close gracefully shuts down the store
func (s *Store) Close() error {
	close(s.writeDone)
	s.wg.Wait()

	if err := s.db.Close(); err != nil {
		return fmt.Errorf("failed to close database: %w", err)
	}

	return nil
}
