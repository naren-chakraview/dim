package cluster

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"
)

// Cluster represents a dimd cluster instance with all necessary coordination components.
type Cluster struct {
	Config              *ClusterConfig
	DedupStore          DedupStore
	LineageBackend      LineageBackend
	GenerationTracker   GenerationTracker
}

// NewCluster initializes a cluster with the appropriate backends based on configuration.
func NewCluster(config *ClusterConfig) (*Cluster, error) {
	if config == nil {
		return nil, fmt.Errorf("cluster config is required")
	}

	// Select dedup store based on cluster mode
	var dedupStore DedupStore
	if config.IsClustered() {
		// In clustered mode, use Postgres backend
		dsn := os.Getenv("DIMD_DEDUP_DSN")
		if dsn == "" {
			// Fall back to in-memory if DSN not provided
			log.Printf("[WARN] DIMD_DEDUP_DSN not set; using in-memory dedup (not production-ready)")
			dedupStore = NewInMemoryDedupStore(60 * time.Minute)
		} else {
			pgStore, err := NewPostgresDedupStore(dsn, 60*time.Minute)
			if err != nil {
				return nil, fmt.Errorf("failed to initialize postgres dedup store: %w", err)
			}
			log.Printf("[INFO] postgres dedup store initialized")
			dedupStore = pgStore
		}
	} else {
		// Single instance: use in-memory dedup
		dedupStore = NewInMemoryDedupStore(60 * time.Minute)
	}

	// Select lineage backend based on cluster mode
	var lineageBackend LineageBackend
	if config.IsClustered() {
		// In clustered mode, use Postgres backend for lineage
		dsn := os.Getenv("DIMD_DEDUP_DSN")
		if dsn == "" {
			// Fall back to no-op if DSN not provided
			log.Printf("[WARN] DIMD_DEDUP_DSN not set; using no-op lineage backend")
			lineageBackend = NewNoOpLineageBackend()
		} else {
			pgBackend, err := NewPostgresLineageBackend(dsn)
			if err != nil {
				return nil, fmt.Errorf("failed to initialize postgres lineage backend: %w", err)
			}
			log.Printf("[INFO] postgres lineage backend initialized")
			lineageBackend = pgBackend
		}
	} else {
		// Single instance: lineage disabled in cluster package (uses existing SQLite directly)
		lineageBackend = NewNoOpLineageBackend()
	}

	// Select generation tracker based on cluster mode
	var genTracker GenerationTracker
	if config.IsClustered() {
		// Multi-instance: use staggered tracker for now (coordinated can be added later)
		genTracker = NewStaggeredGenerationTracker(1)
	} else {
		// Single instance: use local tracker
		genTracker = NewLocalGenerationTracker(1)
	}

	cluster := &Cluster{
		Config:            config,
		DedupStore:        dedupStore,
		LineageBackend:    lineageBackend,
		GenerationTracker: genTracker,
	}

	log.Printf("[INFO] cluster initialized: mode=%s, instances=%d, current=%s, clustered=%v",
		config.Mode, len(config.Instances), config.CurrentID, config.IsClustered())

	return cluster, nil
}

// Close gracefully shuts down all cluster components.
func (c *Cluster) Close(ctx context.Context) error {
	var errs []error

	if err := c.DedupStore.Stop(ctx); err != nil {
		errs = append(errs, fmt.Errorf("dedup store shutdown: %w", err))
	}

	if err := c.LineageBackend.Close(ctx); err != nil {
		errs = append(errs, fmt.Errorf("lineage backend shutdown: %w", err))
	}

	if err := c.GenerationTracker.Close(ctx); err != nil {
		errs = append(errs, fmt.Errorf("generation tracker shutdown: %w", err))
	}

	if len(errs) > 0 {
		return fmt.Errorf("cluster shutdown errors: %v", errs)
	}

	log.Printf("[INFO] cluster shutdown complete")
	return nil
}
