package lineage

import (
	"context"
)

// ProvenanceChain represents the lineage chain for a message or subject
type ProvenanceChain struct {
	Records []LineageRecord
}

// GetProvenance retrieves the provenance chain for a message by correlation ID
// Returns all lineage records for the subject_id associated with the message
func GetProvenance(ctx context.Context, store *Store, messageID string) (*ProvenanceChain, error) {
	// Query the message by ID
	query := `
	SELECT id, route_name, subject_id, message_body, message_headers, metadata,
	       route_version, contract_version, principal, retention_policy,
	       created_at, processed_at, expires_at
	FROM lineage_records
	WHERE id = ?
	`

	var record LineageRecord
	err := store.db.QueryRowContext(ctx, query, messageID).Scan(
		&record.ID, &record.RouteName, &record.SubjectID, &record.MessageBody, &record.MessageHeaders, &record.Metadata,
		&record.RouteVersion, &record.ContractVersion, &record.Principal, &record.RetentionPolicy,
		&record.CreatedAt, &record.ProcessedAt, &record.ExpiresAt,
	)
	if err != nil {
		return nil, err
	}

	// Query all records for this subject_id
	records, err := store.QueryBySubject(ctx, record.SubjectID)
	if err != nil {
		return nil, err
	}

	return &ProvenanceChain{
		Records: records,
	}, nil
}

// GetProvenanceBySubject retrieves the provenance chain for a subject
func GetProvenanceBySubject(ctx context.Context, store *Store, subjectID string) (*ProvenanceChain, error) {
	records, err := store.QueryBySubject(ctx, subjectID)
	if err != nil {
		return nil, err
	}

	return &ProvenanceChain{
		Records: records,
	}, nil
}
