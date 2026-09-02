package lineage

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"time"
)

// Export exports lineage records in CSV or NDJSON format within a time range
func Export(ctx context.Context, store *Store, format string, since, until time.Time, w io.Writer) error {
	// Query all records in time range
	// For simplicity, we'll query all records and filter by route/time
	// In production, would be more efficient to query paginated results

	records, err := store.queryAllRecords(ctx, since, until)
	if err != nil {
		return fmt.Errorf("failed to query records: %w", err)
	}

	switch format {
	case "csv":
		return exportCSV(records, w)
	case "ndjson":
		return exportNDJSON(records, w)
	default:
		return fmt.Errorf("unsupported export format: %s", format)
	}
}

// exportCSV writes lineage records in CSV format
func exportCSV(records []LineageRecord, w io.Writer) error {
	writer := csv.NewWriter(w)
	defer writer.Flush()

	// Write header
	header := []string{
		"id", "route_name", "subject_id", "created_at", "processed_at",
		"route_version", "retention_policy", "contract_version", "principal",
	}
	if err := writer.Write(header); err != nil {
		return err
	}

	// Write records
	for _, record := range records {
		row := []string{
			record.ID,
			record.RouteName,
			record.SubjectID,
			record.CreatedAt.Format(time.RFC3339),
			record.ProcessedAt.Format(time.RFC3339),
			record.RouteVersion,
			record.RetentionPolicy,
			record.ContractVersion,
			record.Principal,
		}
		if err := writer.Write(row); err != nil {
			return err
		}
	}

	return nil
}

// exportNDJSON writes lineage records in NDJSON format (one JSON object per line)
func exportNDJSON(records []LineageRecord, w io.Writer) error {
	encoder := json.NewEncoder(w)
	encoder.SetEscapeHTML(false)

	for _, record := range records {
		if err := encoder.Encode(record); err != nil {
			return err
		}
	}

	return nil
}

// queryAllRecords is a helper to query all records (simplified; in production would paginate)
func (s *Store) queryAllRecords(ctx context.Context, since, until time.Time) ([]LineageRecord, error) {
	query := `
	SELECT id, route_name, subject_id, message_body, message_headers, metadata,
	       route_version, contract_version, principal, retention_policy,
	       created_at, processed_at, expires_at
	FROM lineage_records
	WHERE created_at >= ? AND created_at <= ?
	ORDER BY created_at ASC
	`

	rows, err := s.db.QueryContext(ctx, query, since, until)
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
