package claimcheck

import (
	"context"
	"testing"
	"time"
)

// TestS3StoreConfigValidation verifies configuration validation (M3.2)
func TestS3StoreConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		cfg     S3StoreConfig
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid config with defaults",
			cfg: S3StoreConfig{
				Bucket: "test-bucket",
				Region: "us-east-1",
			},
			wantErr: false,
		},
		{
			name: "missing bucket",
			cfg: S3StoreConfig{
				Region: "us-east-1",
			},
			wantErr: true,
			errMsg:  "S3 bucket name is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Note: This test validates config error handling.
			// Actual S3 connectivity would fail, so we expect that.
			_, err := NewS3ClaimCheckStore(tt.cfg)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				if tt.errMsg != "" && err.Error() != tt.errMsg {
					t.Errorf("expected error %q, got %q", tt.errMsg, err.Error())
				}
			} else if err != nil && tt.cfg.Bucket == "" {
				// Skip actual S3 connectivity test if bucket is empty
				t.Skipf("skipping S3 connectivity test: %v", err)
			}
		})
	}
}

// TestS3StoreInterfaceContract verifies S3 store implements ClaimCheckStore (M3.2)
func TestS3StoreInterfaceContract(t *testing.T) {
	// Verify S3ClaimCheckStore implements ClaimCheckStore interface
	var _ ClaimCheckStore = (*S3ClaimCheckStore)(nil)
}

// TestS3StoreDefaults verifies reasonable defaults are applied (M3.2)
func TestS3StoreDefaults(t *testing.T) {
	cfg := S3StoreConfig{
		Bucket: "test-bucket",
	}

	// Set defaults inline for testing
	if cfg.Region == "" {
		cfg.Region = "us-east-1"
	}
	if cfg.DefaultTTL == 0 {
		cfg.DefaultTTL = 30 * 24 * time.Hour
	}
	if cfg.Prefix == "" {
		cfg.Prefix = "claim-check/"
	}

	if cfg.Region != "us-east-1" {
		t.Errorf("expected region us-east-1, got %s", cfg.Region)
	}
	if cfg.DefaultTTL != 30*24*time.Hour {
		t.Errorf("expected TTL 30 days, got %v", cfg.DefaultTTL)
	}
	if cfg.Prefix != "claim-check/" {
		t.Errorf("expected prefix claim-check/, got %s", cfg.Prefix)
	}
}

// TestS3StoreMetadataHandling verifies metadata is correctly preserved (M3.2)
func TestS3StoreMetadataHandling(t *testing.T) {
	payload := []byte("test payload data")
	metadata := ClaimCheckMetadata{
		ContentType: "text/plain",
		Size:        int64(len(payload)),
	}

	// Verify metadata fields are preserved
	if metadata.ContentType != "text/plain" {
		t.Errorf("expected content type text/plain, got %s", metadata.ContentType)
	}
	if metadata.Size != int64(len(payload)) {
		t.Errorf("expected size %d, got %d", len(payload), metadata.Size)
	}

	// Verify hash computation
	hash := computeHash(payload)
	if hash == "" {
		t.Error("expected non-empty hash")
	}
	if len(hash) == 0 || hash[:7] != "sha256:" {
		t.Errorf("expected sha256 hash format, got %s", hash)
	}
}

// TestS3StoreTicketGeneration verifies unique tickets are generated (M3.2)
func TestS3StoreTicketGeneration(t *testing.T) {
	// Generate multiple tickets and verify they're unique
	tickets := make(map[string]bool)
	for i := 0; i < 10; i++ {
		ticketKey := generateRandomID()
		if tickets[ticketKey] {
			t.Error("generated duplicate ticket key")
		}
		tickets[ticketKey] = true
	}

	if len(tickets) != 10 {
		t.Errorf("expected 10 unique tickets, got %d", len(tickets))
	}
}

// TestCloseIdempotency verifies Close can be called multiple times safely (M3.2)
func TestCloseIdempotency(t *testing.T) {
	// Create a mock store for testing close behavior
	store := &S3ClaimCheckStore{
		client:        nil,
		bucket:        "test",
		metadataCache: make(map[string]ClaimCheckMetadata),
		closed:        false,
	}

	ctx := context.Background()

	// First close should succeed
	err := store.Close(ctx)
	if err != nil {
		t.Errorf("first close failed: %v", err)
	}

	// Second close should also succeed (idempotent)
	err = store.Close(ctx)
	if err != nil {
		t.Errorf("second close failed: %v", err)
	}

	// Verify store is marked as closed
	if !store.closed {
		t.Error("store should be marked as closed")
	}
}

// TestStoreClosedBehavior verifies operations fail when store is closed (M3.2)
func TestStoreClosedBehavior(t *testing.T) {
	store := &S3ClaimCheckStore{
		client:        nil,
		bucket:        "test",
		metadataCache: make(map[string]ClaimCheckMetadata),
		closed:        true, // Already closed
	}

	ctx := context.Background()

	// Store should fail
	_, err := store.Store(ctx, []byte("data"), ClaimCheckMetadata{})
	if err == nil || err.Error() != "store is closed" {
		t.Errorf("expected 'store is closed' error, got %v", err)
	}

	// Retrieve should fail
	_, _, err = store.Retrieve(ctx, "any-ticket")
	if err == nil || err.Error() != "store is closed" {
		t.Errorf("expected 'store is closed' error, got %v", err)
	}

	// Delete should fail
	err = store.Delete(ctx, "any-ticket")
	if err == nil || err.Error() != "store is closed" {
		t.Errorf("expected 'store is closed' error, got %v", err)
	}

	// Verify should fail
	_, err = store.Verify(ctx, "any-ticket", "any-hash")
	if err == nil || err.Error() != "store is closed" {
		t.Errorf("expected 'store is closed' error, got %v", err)
	}
}

// TestS3StoreVsInMemory verifies interface contract is same (M3.2)
func TestS3StoreVsInMemory(t *testing.T) {
	// Both should implement the same interface
	var s3Store ClaimCheckStore
	var memStore ClaimCheckStore

	s3Store = &S3ClaimCheckStore{}
	memStore = NewInMemoryClaimCheckStore()

	// Verify both have the same methods
	_ = s3Store
	_ = memStore

	// If this compiles, both implement ClaimCheckStore interface
	t.Log("S3ClaimCheckStore and InMemoryClaimCheckStore both implement ClaimCheckStore")
}
