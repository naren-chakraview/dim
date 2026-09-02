package lineage

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

// PurgeLogReaper manages lifecycle of purge evidence records
// Evidence records are retained separately from message lineage records
type PurgeLogReaper struct {
	store         *Store
	ttl           time.Duration      // How long to keep evidence records
	warningLead   time.Duration      // Lead time before expiry to warn
	tickerCadence time.Duration      // How often to check for expiry
	stopCh        chan struct{}
	wg            sync.WaitGroup
}

// NewPurgeLogReaper creates a new purge log reaper
// ttl: how long to retain evidence (e.g., 365 days)
// warningLead: warn this long before expiry (e.g., 30 days)
// tickerCadence: check for expiry this frequently (e.g., 1 hour)
func NewPurgeLogReaper(store *Store, ttl, warningLead, tickerCadence time.Duration) *PurgeLogReaper {
	return &PurgeLogReaper{
		store:         store,
		ttl:           ttl,
		warningLead:   warningLead,
		tickerCadence: tickerCadence,
		stopCh:        make(chan struct{}),
	}
}

// Start begins the background evidence reaper and warning scheduler
func (plr *PurgeLogReaper) Start(ctx context.Context) error {
	ticker := time.NewTicker(plr.tickerCadence)

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

				// Log warnings for records approaching expiry
				for _, event := range events {
					if event.ExpiresAt.Before(now.Add(plr.warningLead)) && event.ExpiresAt.After(now) {
						daysLeft := int(event.ExpiresAt.Sub(now).Hours() / 24)
						log.Printf("[WARN] Purge evidence for subject %q expires in %d days (purged_at=%s)", event.SubjectID, daysLeft, event.PurgedAt.Format(time.RFC3339))
					}
				}

				// Delete expired evidence
				expiredEvents, err := plr.store.QueryPurgeEvents(ctx, time.Time{}, now)
				if err != nil {
					log.Printf("[WARN] Failed to query expired purge events: %v", err)
					continue
				}

				// For now, we'd need a DeletePurgeEvent method in Store to actually delete them
				// This is a simplified implementation
				if len(expiredEvents) > 0 {
					log.Printf("[INFO] Purge log reaper found %d expired evidence records", len(expiredEvents))
				}
			}
		}
	}()

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
