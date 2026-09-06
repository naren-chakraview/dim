package cluster

import (
	"context"
	"testing"
	"time"
)

func TestInMemoryDedupStore_Check(t *testing.T) {
	ctx := context.Background()
	store := NewInMemoryDedupStore(10 * time.Millisecond)
	defer store.Stop(ctx)

	// First check should return false (new key)
	isDup, err := store.Check(ctx, "key1")
	if err != nil {
		t.Fatalf("first check failed: %v", err)
	}
	if isDup {
		t.Errorf("expected isDup=false for new key, got %v", isDup)
	}

	// Second check of same key should return true (duplicate)
	isDup, err = store.Check(ctx, "key1")
	if err != nil {
		t.Fatalf("second check failed: %v", err)
	}
	if !isDup {
		t.Errorf("expected isDup=true for duplicate key, got %v", isDup)
	}

	// Different key should return false
	isDup, err = store.Check(ctx, "key2")
	if err != nil {
		t.Fatalf("third check failed: %v", err)
	}
	if isDup {
		t.Errorf("expected isDup=false for different key, got %v", isDup)
	}
}

func TestInMemoryDedupStore_Expiration(t *testing.T) {
	ctx := context.Background()
	store := NewInMemoryDedupStore(50 * time.Millisecond)
	defer store.Stop(ctx)

	// Check key
	isDup, err := store.Check(ctx, "expiring_key")
	if err != nil {
		t.Fatalf("check failed: %v", err)
	}
	if isDup {
		t.Errorf("expected isDup=false for new key")
	}

	// Key should exist immediately
	isDup, err = store.Check(ctx, "expiring_key")
	if err != nil {
		t.Fatalf("check failed: %v", err)
	}
	if !isDup {
		t.Errorf("expected isDup=true before expiration")
	}

	// Wait for expiration
	time.Sleep(100 * time.Millisecond)

	// Key should now be treated as new (expired)
	isDup, err = store.Check(ctx, "expiring_key")
	if err != nil {
		t.Fatalf("check failed: %v", err)
	}
	if isDup {
		t.Errorf("expected isDup=false after expiration")
	}
}

func TestInMemoryDedupStore_Delete(t *testing.T) {
	ctx := context.Background()
	store := NewInMemoryDedupStore(1 * time.Minute)
	defer store.Stop(ctx)

	// Add key
	isDup, err := store.Check(ctx, "key_to_delete")
	if err != nil {
		t.Fatalf("check failed: %v", err)
	}
	if isDup {
		t.Errorf("expected isDup=false for new key")
	}

	// Delete key
	err = store.Delete(ctx, "key_to_delete")
	if err != nil {
		t.Fatalf("delete failed: %v", err)
	}

	// After deletion, key should be new again
	isDup, err = store.Check(ctx, "key_to_delete")
	if err != nil {
		t.Fatalf("check failed: %v", err)
	}
	if isDup {
		t.Errorf("expected isDup=false after deletion")
	}
}
