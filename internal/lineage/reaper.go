package lineage

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

// Reaper automatically removes expired lineage records based on retention policies
type Reaper struct {
	store      *Store
	resolver   *RetentionResolver
	cadences   map[string]time.Duration // policy -> check cadence
	tickers    map[string]*time.Ticker
	stopCh     chan struct{}
	wg         sync.WaitGroup
}

// NewReaper creates a new reaper for automatic record expiry
// cadences: maps policy names to check intervals (e.g., "30-days" -> 1 hour)
func NewReaper(store *Store, resolver *RetentionResolver, cadences map[string]time.Duration) *Reaper {
	return &Reaper{
		store:      store,
		resolver:   resolver,
		cadences:   cadences,
		tickers:    make(map[string]*time.Ticker),
		stopCh:     make(chan struct{}),
	}
}

// Start begins background reapers for all configured policies
func (r *Reaper) Start(ctx context.Context) error {
	for policyName, cadence := range r.cadences {
		if cadence <= 0 {
			continue // Skip zero/negative cadences
		}

		ticker := time.NewTicker(cadence)
		r.tickers[policyName] = ticker

		r.wg.Add(1)
		go r.reaperLoop(ctx, policyName, ticker)
	}

	log.Printf("[DEBUG] Reaper started with %d policies", len(r.cadences))
	return nil
}

// reaperLoop runs the reaper for a single policy on its cadence
func (r *Reaper) reaperLoop(ctx context.Context, policyName string, ticker *time.Ticker) {
	defer r.wg.Done()
	defer ticker.Stop()

	log.Printf("[DEBUG] Reaper loop started for policy %q", policyName)

	for {
		select {
		case <-r.stopCh:
			log.Printf("[DEBUG] Reaper loop stopped for policy %q", policyName)
			return

		case <-ctx.Done():
			log.Printf("[DEBUG] Reaper loop context cancelled for policy %q", policyName)
			return

		case <-ticker.C:
			// Query expired records
			expired, err := r.store.QueryExpiredRecords(ctx, time.Now().UTC())
			if err != nil {
				log.Printf("[WARN] Failed to query expired records for policy %q: %v", policyName, err)
				continue
			}

			if len(expired) == 0 {
				continue
			}

			// Delete expired records (grouped by subject_id for efficiency)
			subjects := make(map[string]bool)
			for _, record := range expired {
				subjects[record.SubjectID] = true
			}

			totalDeleted := 0
			for subjectID := range subjects {
				if subjectID == "" {
					continue // Skip records with empty subject_id
				}

				count, err := r.store.DeleteBySubject(ctx, subjectID)
				if err != nil {
					log.Printf("[WARN] Failed to delete records for subject %q: %v", subjectID, err)
					continue
				}

				totalDeleted += count

				// Record reap event
				eventID := fmt.Sprintf("reap-%d-%s", time.Now().UnixNano(), subjectID)
				purgeErr := r.store.RecordPurgeEvent(
					ctx,
					eventID,
					"auto_reap",
					subjectID,
					fmt.Sprintf("Automatic reap for policy %q", policyName),
					count,
					time.Now().UTC().Add(365*24*time.Hour), // 1-year evidence retention
				)
				if purgeErr != nil {
					log.Printf("[WARN] Failed to record reap event for subject %q: %v", subjectID, purgeErr)
				}
			}

			if totalDeleted > 0 {
				log.Printf("[INFO] Reaper deleted %d expired records for policy %q", totalDeleted, policyName)
			}
		}
	}
}

// Stop gracefully stops all reapers
func (r *Reaper) Stop() error {
	close(r.stopCh)

	// Wait for all reaper loops to finish
	waitCh := make(chan struct{})
	go func() {
		r.wg.Wait()
		close(waitCh)
	}()

	// Wait with timeout
	select {
	case <-waitCh:
		log.Printf("[DEBUG] All reapers stopped successfully")
		return nil
	case <-time.After(10 * time.Second):
		return fmt.Errorf("reaper shutdown timeout")
	}
}
