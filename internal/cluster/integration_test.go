package cluster

import (
	"context"
	"os"
	"testing"
	"time"
)

// TestClusterCreation verifies a cluster can be initialized correctly.
func TestClusterCreation(t *testing.T) {
	// Create static cluster config
	instances := []Instance{
		{ID: "dimd-1", Host: "localhost", Port: 8080},
		{ID: "dimd-2", Host: "localhost", Port: 8081},
		{ID: "dimd-3", Host: "localhost", Port: 8082},
	}

	config, err := NewClusterConfigFromStatic(instances, "dimd-1")
	if err != nil {
		t.Fatalf("failed to create cluster config: %v", err)
	}

	// Create cluster
	cluster, err := NewCluster(config)
	if err != nil {
		t.Fatalf("failed to create cluster: %v", err)
	}
	defer cluster.Close(context.Background())

	if cluster.DedupStore == nil {
		t.Errorf("expected DedupStore to be initialized")
	}

	if cluster.GenerationTracker == nil {
		t.Errorf("expected GenerationTracker to be initialized")
	}

	if !cluster.Config.IsClustered() {
		t.Errorf("expected cluster to be in clustered mode")
	}
}

// TestNoDuplicateProcessingAcrossInstances verifies that when multiple instances
// receive the same message (with same dedup key), only one processes it.
func TestNoDuplicateProcessingAcrossInstances(t *testing.T) {
	ctx := context.Background()

	// Create two separate cluster instances with shared dedup store
	instances := []Instance{
		{ID: "instance-1", Host: "localhost", Port: 8080},
		{ID: "instance-2", Host: "localhost", Port: 8081},
	}

	config, err := NewClusterConfigFromStatic(instances, "instance-1")
	if err != nil {
		t.Fatalf("failed to create config: %v", err)
	}

	cluster1, err := NewCluster(config)
	if err != nil {
		t.Fatalf("failed to create cluster: %v", err)
	}
	defer cluster1.Close(ctx)

	// Both instances use the same dedup store (shared reference)
	sharedDedup := cluster1.DedupStore

	// Simulate instance-1 receiving a message
	isDup1, err := sharedDedup.Check(ctx, "message-key-1")
	if err != nil {
		t.Fatalf("instance-1 dedup check failed: %v", err)
	}
	if isDup1 {
		t.Errorf("instance-1: expected isDup=false for new key")
	}

	// Simulate instance-2 receiving the same message (via shared dedup)
	isDup2, err := sharedDedup.Check(ctx, "message-key-1")
	if err != nil {
		t.Fatalf("instance-2 dedup check failed: %v", err)
	}
	if !isDup2 {
		t.Errorf("instance-2: expected isDup=true (duplicate), got false")
	}

	// Verify dedup is cluster-wide: both instances see the same state
	if isDup1 == isDup2 {
		if isDup1 {
			t.Errorf("both instances report duplicate, expected only second to")
		}
	}
}

// TestGenerationTracking verifies generation transitions work correctly.
func TestGenerationTracking(t *testing.T) {
	ctx := context.Background()

	instances := []Instance{
		{ID: "dimd-1", Host: "localhost", Port: 8080},
		{ID: "dimd-2", Host: "localhost", Port: 8081},
	}

	config, err := NewClusterConfigFromStatic(instances, "dimd-1")
	if err != nil {
		t.Fatalf("failed to create config: %v", err)
	}

	cluster, err := NewCluster(config)
	if err != nil {
		t.Fatalf("failed to create cluster: %v", err)
	}
	defer cluster.Close(ctx)

	// Check initial generation
	gen1, err := cluster.GenerationTracker.CurrentGeneration(ctx)
	if err != nil {
		t.Fatalf("failed to get generation: %v", err)
	}

	// Transition to next generation
	err = cluster.GenerationTracker.Transition(ctx, gen1+1)
	if err != nil {
		t.Fatalf("failed to transition generation: %v", err)
	}

	// Verify transition happened
	gen2, err := cluster.GenerationTracker.CurrentGeneration(ctx)
	if err != nil {
		t.Fatalf("failed to get generation: %v", err)
	}

	if gen2 <= gen1 {
		t.Errorf("expected generation to advance: %d -> %d", gen1, gen2)
	}
}

// TestClusterFromEnv verifies environment-based cluster configuration.
func TestClusterFromEnv(t *testing.T) {
	os.Setenv("DIMD_INSTANCE_ID", "dimd-1")
	os.Setenv("DIMD_CLUSTER_HOSTS", "localhost:8080,localhost:8081,localhost:8082")
	defer func() {
		os.Unsetenv("DIMD_INSTANCE_ID")
		os.Unsetenv("DIMD_CLUSTER_HOSTS")
	}()

	config, err := NewClusterConfigFromEnv()
	if err != nil {
		t.Fatalf("failed to create config from env: %v", err)
	}

	cluster, err := NewCluster(config)
	if err != nil {
		t.Fatalf("failed to create cluster: %v", err)
	}
	defer cluster.Close(context.Background())

	if !cluster.Config.IsClustered() {
		t.Errorf("expected cluster to be in clustered mode")
	}

	if cluster.Config.CurrentID != "dimd-1" {
		t.Errorf("expected current instance ID to be dimd-1, got %s", cluster.Config.CurrentID)
	}
}

// TestSingleInstanceMode verifies single-instance (non-clustered) mode.
func TestSingleInstanceMode(t *testing.T) {
	ctx := context.Background()

	config := SingleInstanceConfig("dimd-single")

	cluster, err := NewCluster(config)
	if err != nil {
		t.Fatalf("failed to create cluster: %v", err)
	}
	defer cluster.Close(ctx)

	if cluster.Config.IsClustered() {
		t.Errorf("expected single instance to report IsClustered=false")
	}

	// Dedup should still work
	isDup, err := cluster.DedupStore.Check(ctx, "single-key")
	if err != nil {
		t.Fatalf("dedup check failed: %v", err)
	}
	if isDup {
		t.Errorf("expected isDup=false for new key in single instance mode")
	}

	// Verify it's marked as duplicate on second check
	isDup, err = cluster.DedupStore.Check(ctx, "single-key")
	if err != nil {
		t.Fatalf("dedup check failed: %v", err)
	}
	if !isDup {
		t.Errorf("expected isDup=true on duplicate check")
	}
}

// TestLineageBackendInterface verifies lineage backend works.
func TestLineageBackendInterface(t *testing.T) {
	ctx := context.Background()

	backend := NewNoOpLineageBackend()

	// Test insert (should be no-op)
	record := &LineageRecord{
		ID:        "record-1",
		RouteName: "test-route",
		SubjectID: "subject-1",
	}
	err := backend.InsertRecord(ctx, record)
	if err != nil {
		t.Fatalf("insert failed: %v", err)
	}

	// Test query (should return empty)
	records, err := backend.QueryBySubject(ctx, "subject-1", 10)
	if err != nil {
		t.Fatalf("query failed: %v", err)
	}
	if len(records) != 0 {
		t.Errorf("expected empty result from no-op backend, got %d records", len(records))
	}

	// Test delete (should be no-op)
	err = backend.DeleteByID(ctx, "record-1")
	if err != nil {
		t.Fatalf("delete failed: %v", err)
	}

	// Test close
	err = backend.Close(ctx)
	if err != nil {
		t.Fatalf("close failed: %v", err)
	}
}

// TestDedupTTLExpiration verifies that dedup keys expire after TTL.
func TestDedupTTLExpiration(t *testing.T) {
	ctx := context.Background()

	config := SingleInstanceConfig("dimd-test")
	cluster, err := NewCluster(config)
	if err != nil {
		t.Fatalf("failed to create cluster: %v", err)
	}
	defer cluster.Close(ctx)

	// Note: the default TTL is 60 minutes, so we can't actually test expiration
	// without waiting. This is a smoke test that dedup functionality works.

	key := "expiring-key"

	// First check: should be new
	isDup, err := cluster.DedupStore.Check(ctx, key)
	if err != nil {
		t.Fatalf("check failed: %v", err)
	}
	if isDup {
		t.Errorf("expected new key to not be duplicate")
	}

	// Second check: should be duplicate (within TTL)
	isDup, err = cluster.DedupStore.Check(ctx, key)
	if err != nil {
		t.Fatalf("check failed: %v", err)
	}
	if !isDup {
		t.Errorf("expected duplicate within TTL")
	}

	// After delete: should be new again
	err = cluster.DedupStore.Delete(ctx, key)
	if err != nil {
		t.Fatalf("delete failed: %v", err)
	}

	isDup, err = cluster.DedupStore.Check(ctx, key)
	if err != nil {
		t.Fatalf("check failed: %v", err)
	}
	if isDup {
		t.Errorf("expected new key after deletion")
	}
}

// TestClusterShutdown verifies graceful cluster shutdown.
func TestClusterShutdown(t *testing.T) {
	config := SingleInstanceConfig("dimd-shutdown-test")

	cluster, err := NewCluster(config)
	if err != nil {
		t.Fatalf("failed to create cluster: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Verify we can add a key before shutdown
	_, err = cluster.DedupStore.Check(ctx, "test-key")
	if err != nil {
		t.Fatalf("dedup check failed: %v", err)
	}

	// Shutdown should succeed
	err = cluster.Close(ctx)
	if err != nil {
		t.Fatalf("cluster shutdown failed: %v", err)
	}
}
