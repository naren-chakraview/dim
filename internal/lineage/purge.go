package lineage

import (
	"context"
	"fmt"
	"log"
	"time"
)

// PurgeOpts specifies options for manual purge operations
type PurgeOpts struct {
	SubjectID  string        // If set, purge all records for this subject only
	RouteNames []string      // Optional: limit to specific routes
	BeforeTime time.Time     // Optional: only purge before this time
}

// Purge manually deletes lineage records based on options
// Records subject IDs are purged if SubjectID is specified
// Otherwise, bulk purge by route/time is performed
// Returns count of deleted records
func Purge(ctx context.Context, store *Store, opts PurgeOpts) (int, error) {
	if opts.SubjectID != "" {
		// Subject-specific purge
		count, err := store.DeleteBySubject(ctx, opts.SubjectID)
		if err != nil {
			return 0, fmt.Errorf("failed to purge by subject: %w", err)
		}

		// Record purge event
		eventID := fmt.Sprintf("purge-%d-%s", time.Now().UnixNano(), opts.SubjectID)
		purgeErr := store.RecordPurgeEvent(
			ctx,
			eventID,
			"manual_purge",
			opts.SubjectID,
			"Manual purge",
			count,
			time.Now().UTC().Add(365*24*time.Hour), // 1-year evidence retention
		)
		if purgeErr != nil {
			log.Printf("[WARN] Failed to record purge event: %v", purgeErr)
		}

		return count, nil
	}

	// Bulk purge (by route/time) - for now, simplified implementation
	// In production, would need more sophisticated bulk delete query
	log.Printf("[WARN] Bulk purge by route/time not yet implemented")
	return 0, fmt.Errorf("bulk purge not yet implemented; use --subject for subject-specific purge")
}
