package cluster

import (
	"context"
	"time"
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
