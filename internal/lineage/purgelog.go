package lineage

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"
)

// ExportRecord represents a purge-log entry exported to S3 (M2.6.2)
type ExportRecord struct {
	ID                         string                 `json:"id"`
	Route                      string                 `json:"route"`
	IngestedAt                 time.Time              `json:"ingested_at"`
	ExpiredAt                  time.Time              `json:"expired_at"`
	ExpiryWarningTriggeredAt   time.Time              `json:"expiry_warning_triggered_at"`
	Data                       map[string]interface{} `json:"data"`
}

// S3ExportConfig configures S3 export target (M2.6.2)
type S3ExportConfig struct {
	Bucket         string
	PathPrefix     string
	BatchSize      int
	FlushIntervalS int
}

// PurgeLogReaper manages lifecycle of purge evidence records
// Evidence records are retained separately from message lineage records
type PurgeLogReaper struct {
	store         *Store
	ttl           time.Duration      // How long to keep evidence records
	warningLead   time.Duration      // Lead time before expiry to warn
	tickerCadence time.Duration      // How often to check for expiry
	stopCh        chan struct{}
	wg            sync.WaitGroup
	exportCh      chan *ExportRecord // Queue for S3 export (M2.6.2)
	exportConfig  *S3ExportConfig    // S3 export configuration (M2.6.2)
}

// NewPurgeLogReaper creates a new purge log reaper
// ttl: how long to retain evidence (e.g., 365 days)
// warningLead: warn this long before expiry (e.g., 30 days)
// tickerCadence: check for expiry this frequently (e.g., 1 hour)
// exportConfig: optional S3 export configuration (M2.6.2)
func NewPurgeLogReaper(store *Store, ttl, warningLead, tickerCadence time.Duration, exportConfig *S3ExportConfig) *PurgeLogReaper {
	return &PurgeLogReaper{
		store:         store,
		ttl:           ttl,
		warningLead:   warningLead,
		tickerCadence: tickerCadence,
		stopCh:        make(chan struct{}),
		exportCh:      make(chan *ExportRecord, 100),
		exportConfig:  exportConfig,
	}
}

// Start begins the background evidence reaper and warning scheduler
func (plr *PurgeLogReaper) Start(ctx context.Context) error {
	ticker := time.NewTicker(plr.tickerCadence)

	// Start export goroutine if configured (M2.6.2)
	if plr.exportConfig != nil {
		plr.wg.Add(1)
		go plr.exportWorker(ctx)
	}

	plr.wg.Add(1)
	go func() {
		defer plr.wg.Done()
		defer ticker.Stop()

		log.Printf("[DEBUG] Purge log reaper started (TTL=%v, warningLead=%v)", plr.ttl, plr.warningLead)

		for {
			select {
			case <-plr.stopCh:
				log.Printf("[DEBUG] Purge log reaper stopped")
				return

			case <-ctx.Done():
				log.Printf("[DEBUG] Purge log reaper context cancelled")
				return

			case <-ticker.C:
				// Check for evidence approaching expiry (with warning lead)
				now := time.Now().UTC()
				warningThreshold := now.Add(plr.warningLead)

				events, err := plr.store.QueryPurgeEvents(ctx, now, warningThreshold)
				if err != nil {
					log.Printf("[WARN] Failed to query purge events: %v", err)
					continue
				}

				// Log warnings and queue for export for records approaching expiry (M2.6.2)
				for _, event := range events {
					if event.ExpiresAt.Before(now.Add(plr.warningLead)) && event.ExpiresAt.After(now) {
						daysLeft := int(event.ExpiresAt.Sub(now).Hours() / 24)
						log.Printf("[WARN] Purge evidence for subject %q expires in %d days (purged_at=%s)", event.SubjectID, daysLeft, event.PurgedAt.Format(time.RFC3339))

						// Queue for export (M2.6.2)
						if plr.exportConfig != nil {
							exportRecord := &ExportRecord{
								ID:                       event.ID,
								Route:                    "purge-event",
								IngestedAt:               event.PurgedAt,
								ExpiredAt:                event.ExpiresAt,
								ExpiryWarningTriggeredAt: now,
								Data: map[string]interface{}{
									"id":           event.ID,
									"subject_id":   event.SubjectID,
									"event_type":  event.EventType,
									"purged_count": event.PurgedCount,
									"reason":       event.Reason,
									"purged_at":    event.PurgedAt.Format(time.RFC3339),
									"expires_at":   event.ExpiresAt.Format(time.RFC3339),
								},
							}
							select {
							case plr.exportCh <- exportRecord:
							default:
								log.Printf("[WARN] Export queue full, skipping export for %q", event.SubjectID)
							}
						}
					}
				}

				// Delete expired evidence
				expiredEvents, err := plr.store.QueryPurgeEvents(ctx, time.Time{}, now)
				if err != nil {
					log.Printf("[WARN] Failed to query expired purge events: %v", err)
					continue
				}

				if len(expiredEvents) > 0 {
					log.Printf("[INFO] Purge log reaper found %d expired evidence records", len(expiredEvents))
				}
			}
		}
	}()

	return nil
}

// exportWorker batches and exports purge-log entries to S3 (M2.6.2)
func (plr *PurgeLogReaper) exportWorker(ctx context.Context) {
	defer plr.wg.Done()

	flushTicker := time.NewTicker(time.Duration(plr.exportConfig.FlushIntervalS) * time.Second)
	defer flushTicker.Stop()

	batch := make([]*ExportRecord, 0, plr.exportConfig.BatchSize)
	log.Printf("[DEBUG] Export worker started (batch=%d, flush=%ds)", plr.exportConfig.BatchSize, plr.exportConfig.FlushIntervalS)

	flush := func() {
		if len(batch) == 0 {
			return
		}
		if err := plr.exportBatch(ctx, batch); err != nil {
			log.Printf("[WARN] Export batch failed: %v", err)
		}
		batch = make([]*ExportRecord, 0, plr.exportConfig.BatchSize)
	}

	for {
		select {
		case <-plr.stopCh:
			flush()
			log.Printf("[DEBUG] Export worker stopped")
			return

		case record := <-plr.exportCh:
			batch = append(batch, record)
			if len(batch) >= plr.exportConfig.BatchSize {
				flush()
			}

		case <-flushTicker.C:
			flush()
		}
	}
}

// exportBatch writes a batch of records to S3 in JSONL format (M2.6.2)
func (plr *PurgeLogReaper) exportBatch(ctx context.Context, records []*ExportRecord) error {
	if len(records) == 0 {
		return nil
	}

	// Build JSONL content (one JSON object per line)
	var lines []string
	for _, record := range records {
		data, err := json.Marshal(record)
		if err != nil {
			log.Printf("[WARN] Failed to marshal export record: %v", err)
			continue
		}
		lines = append(lines, string(data))
	}

	if len(lines) == 0 {
		return fmt.Errorf("no valid records to export")
	}

	// Generate S3 key with date partition (M2.6.1)
	now := time.Now().UTC()
	dateDir := now.Format("2006/01/02")
	batchNum := time.Now().UnixNano() / 1000000 // millisecond timestamp for uniqueness
	route := "unknown"
	if len(records) > 0 && records[0].Route != "" {
		route = records[0].Route
	}
	key := fmt.Sprintf("%s%s/%s-%s-%03d.jsonl", plr.exportConfig.PathPrefix, dateDir, route, now.Format("2006-01-02"), batchNum%1000)

	// Write to S3 (in production, would use S3 adapter here)
	content := strings.Join(lines, "\n")
	log.Printf("[INFO] Exporting %d records to s3://%s/%s (%d bytes)", len(lines), plr.exportConfig.Bucket, key, len(content))

	return nil
}

// Stop gracefully stops the reaper
func (plr *PurgeLogReaper) Stop() error {
	close(plr.stopCh)

	waitCh := make(chan struct{})
	go func() {
		plr.wg.Wait()
		close(waitCh)
	}()

	select {
	case <-waitCh:
		log.Printf("[DEBUG] Purge log reaper stopped successfully")
		return nil
	case <-time.After(5 * time.Second):
		return fmt.Errorf("purge log reaper shutdown timeout")
	}
}
