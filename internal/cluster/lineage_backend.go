package cluster

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"
)

// LineageBackend defines the interface for storing and querying lineage records across cluster instances.
// Implementations can be SQLite (single-instance default) or PostgreSQL (cluster-wide).
type LineageBackend interface {
	// InsertRecord stores a lineage record.
	InsertRecord(ctx context.Context, record *LineageRecord) error

	// QueryBySubject retrieves all lineage records for a given subject ID.
	QueryBySubject(ctx context.Context, subjectID string, limit int) ([]*LineageRecord, error)

	// QueryByRoute retrieves all lineage records for a given route name.
	QueryByRoute(ctx context.Context, routeName string, limit int) ([]*LineageRecord, error)

	// DeleteByID removes a lineage record by ID (for purge operations).
	DeleteByID(ctx context.Context, id string) error

	// Close gracefully shuts down the backend.
	Close(ctx context.Context) error
}

// LineageRecord represents a message lineage entry in cluster mode.
// Compatible with the existing per-instance record structure.
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
	InstanceID      string    `json:"instance_id"` // Which cluster instance processed this
}

// NoOpLineageBackend is a no-op implementation (for testing or disabled lineage).
type NoOpLineageBackend struct{}

// NewNoOpLineageBackend creates a no-op backend that discards all records.
func NewNoOpLineageBackend() *NoOpLineageBackend {
	return &NoOpLineageBackend{}
}

func (n *NoOpLineageBackend) InsertRecord(ctx context.Context, record *LineageRecord) error {
	return nil
}

func (n *NoOpLineageBackend) QueryBySubject(ctx context.Context, subjectID string, limit int) ([]*LineageRecord, error) {
	return []*LineageRecord{}, nil
}

func (n *NoOpLineageBackend) QueryByRoute(ctx context.Context, routeName string, limit int) ([]*LineageRecord, error) {
	return []*LineageRecord{}, nil
}

func (n *NoOpLineageBackend) DeleteByID(ctx context.Context, id string) error {
	return nil
}

func (n *NoOpLineageBackend) Close(ctx context.Context) error {
	return nil
}

// PostgresLineageBackend stores lineage records in a centralized Postgres database.
type PostgresLineageBackend struct {
	db *sql.DB
}

// NewPostgresLineageBackend creates a Postgres-backed lineage backend.
func NewPostgresLineageBackend(dsn string) (*PostgresLineageBackend, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to postgres: %w", err)
	}

	// Verify connection works
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("postgres ping failed: %w", err)
	}

	return &PostgresLineageBackend{db: db}, nil
}

// InsertRecord stores a lineage record in Postgres.
func (p *PostgresLineageBackend) InsertRecord(ctx context.Context, record *LineageRecord) error {
	query := `
		INSERT INTO lineage_records
		(id, route_name, subject_id, message_body, message_headers, metadata,
		 route_version, contract_version, principal, retention_policy, instance_id,
		 created_at, processed_at, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		ON CONFLICT (id) DO NOTHING
	`

	_, err := p.db.ExecContext(ctx, query,
		record.ID, record.RouteName, record.SubjectID,
		record.MessageBody, record.MessageHeaders, record.Metadata,
		record.RouteVersion, record.ContractVersion, record.Principal,
		record.RetentionPolicy, record.InstanceID,
		record.CreatedAt, record.ProcessedAt, record.ExpiresAt,
	)

	if err != nil {
		return fmt.Errorf("failed to insert lineage record: %w", err)
	}

	return nil
}

// QueryBySubject retrieves all lineage records for a given subject ID.
func (p *PostgresLineageBackend) QueryBySubject(ctx context.Context, subjectID string, limit int) ([]*LineageRecord, error) {
	if limit <= 0 {
		limit = 100
	}

	query := `
		SELECT id, route_name, subject_id, message_body, message_headers, metadata,
		       route_version, contract_version, principal, retention_policy, instance_id,
		       created_at, processed_at, expires_at
		FROM lineage_records
		WHERE subject_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`

	rows, err := p.db.QueryContext(ctx, query, subjectID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query lineage by subject: %w", err)
	}
	defer rows.Close()

	var records []*LineageRecord
	for rows.Next() {
		record := &LineageRecord{}
		err := rows.Scan(
			&record.ID, &record.RouteName, &record.SubjectID,
			&record.MessageBody, &record.MessageHeaders, &record.Metadata,
			&record.RouteVersion, &record.ContractVersion, &record.Principal,
			&record.RetentionPolicy, &record.InstanceID,
			&record.CreatedAt, &record.ProcessedAt, &record.ExpiresAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan lineage record: %w", err)
		}
		records = append(records, record)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("query error: %w", err)
	}

	return records, nil
}

// QueryByRoute retrieves all lineage records for a given route name.
func (p *PostgresLineageBackend) QueryByRoute(ctx context.Context, routeName string, limit int) ([]*LineageRecord, error) {
	if limit <= 0 {
		limit = 100
	}

	query := `
		SELECT id, route_name, subject_id, message_body, message_headers, metadata,
		       route_version, contract_version, principal, retention_policy, instance_id,
		       created_at, processed_at, expires_at
		FROM lineage_records
		WHERE route_name = $1
		ORDER BY created_at DESC
		LIMIT $2
	`

	rows, err := p.db.QueryContext(ctx, query, routeName, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query lineage by route: %w", err)
	}
	defer rows.Close()

	var records []*LineageRecord
	for rows.Next() {
		record := &LineageRecord{}
		err := rows.Scan(
			&record.ID, &record.RouteName, &record.SubjectID,
			&record.MessageBody, &record.MessageHeaders, &record.Metadata,
			&record.RouteVersion, &record.ContractVersion, &record.Principal,
			&record.RetentionPolicy, &record.InstanceID,
			&record.CreatedAt, &record.ProcessedAt, &record.ExpiresAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan lineage record: %w", err)
		}
		records = append(records, record)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("query error: %w", err)
	}

	return records, nil
}

// DeleteByID removes a lineage record by ID.
func (p *PostgresLineageBackend) DeleteByID(ctx context.Context, id string) error {
	query := "DELETE FROM lineage_records WHERE id = $1"
	_, err := p.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete lineage record: %w", err)
	}
	return nil
}

// Close gracefully closes the Postgres connection.
func (p *PostgresLineageBackend) Close(ctx context.Context) error {
	return p.db.Close()
}
