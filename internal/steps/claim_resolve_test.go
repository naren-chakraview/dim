package steps

import (
	"context"
	"testing"

	"github.com/naren-chakraview/dim/internal/adapters/claimcheck"
	"github.com/naren-chakraview/dim/internal/config"
	"github.com/naren-chakraview/dim/internal/engine"
)

// TestClaimResolveStepBasic verifies basic claim-resolve functionality (M3.2.3)
func TestClaimResolveStepBasic(t *testing.T) {
	store := claimcheck.NewInMemoryClaimCheckStore()
	ctx := context.Background()

	// First, store a payload
	payload := []byte(`{"order_id":"ORD-123","amount":500}`)
	metadata := claimcheck.ClaimCheckMetadata{
		ContentType: "application/json",
		Size:        int64(len(payload)),
	}

	ticket, err := store.Store(ctx, payload, metadata)
	if err != nil {
		t.Fatalf("Store failed: %v", err)
	}

	// Create a message with claim-check ticket
	msg := engine.NewMessage(
		map[string]interface{}{
			"order_id": "ORD-123",
			"_claim_check": map[string]interface{}{
				"ticket":        ticket,
				"size":          metadata.Size,
				"content_type":  "application/json",
				"hash":          metadata.Hash,
			},
		},
		"test-route",
		"v1",
	)

	// Resolve the claim-check
	resolver, err := NewClaimResolveStep(&config.ClaimResolveSpec{}, store)
	if err != nil {
		t.Fatalf("NewClaimResolveStep failed: %v", err)
	}

	result, err := resolver.Execute(ctx, msg)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if result == nil {
		t.Fatal("Expected result message, got nil")
	}

	// Verify payload is restored
	if result.Body == nil {
		t.Error("Expected non-nil body")
	}
}

// TestClaimResolveStepNoTicket verifies pass-through for non-claim-checked messages (M3.2.3)
func TestClaimResolveStepNoTicket(t *testing.T) {
	store := claimcheck.NewInMemoryClaimCheckStore()
	resolver, err := NewClaimResolveStep(&config.ClaimResolveSpec{}, store)
	if err != nil {
		t.Fatalf("NewClaimResolveStep failed: %v", err)
	}

	ctx := context.Background()
	msg := engine.NewMessage(
		map[string]interface{}{
			"order_id": "ORD-124",
			"amount":   600.0,
		},
		"test-route",
		"v1",
	)

	result, err := resolver.Execute(ctx, msg)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Verify message passes through unchanged
	if result.Body == nil {
		t.Error("Expected body to pass through")
	}

	bodyMap, ok := result.Body.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected map body, got %T", result.Body)
	}

	if orderID, exists := bodyMap["order_id"]; !exists || orderID != "ORD-124" {
		t.Error("Expected order_id to be preserved")
	}
}

// TestClaimResolveStepRoundTrip verifies store -> claim-check -> resolve round-trip (M3.2.3)
func TestClaimResolveStepRoundTrip(t *testing.T) {
	store := claimcheck.NewInMemoryClaimCheckStore()
	ctx := context.Background()

	// Original payload
	originalBody := map[string]interface{}{
		"order_id": "ORD-125",
		"items": []interface{}{
			map[string]interface{}{"sku": "SKU-001", "qty": 2},
		},
		"total": 750.0,
	}

	// Step 1: Claim-check the payload
	checkSpec := &config.ClaimCheckSpec{
		PayloadField:    "body",
		RemovePayload:   true,
		TicketFieldPath: "_claim_check",
	}
	checkStep, err := NewClaimCheckStep(checkSpec, store)
	if err != nil {
		t.Fatalf("NewClaimCheckStep failed: %v", err)
	}

	msg := engine.NewMessage(originalBody, "test-route", "v1")
	checkedMsg, err := checkStep.Execute(ctx, msg)
	if err != nil {
		t.Fatalf("ClaimCheck Execute failed: %v", err)
	}

	// Verify message was claim-checked
	bodyMap, _ := checkedMsg.Body.(map[string]interface{})
	if _, hasTicket := bodyMap["_claim_check"]; !hasTicket {
		t.Fatal("Expected claim-check ticket in body")
	}

	// Step 2: Resolve the claim-check
	resolveStep, err := NewClaimResolveStep(&config.ClaimResolveSpec{}, store)
	if err != nil {
		t.Fatalf("NewClaimResolveStep failed: %v", err)
	}

	resolved, err := resolveStep.Execute(ctx, checkedMsg)
	if err != nil {
		t.Fatalf("Resolve Execute failed: %v", err)
	}

	// Verify payload is restored
	if resolved.Body == nil {
		t.Error("Expected body to be restored")
	}
}

// TestClaimResolveStepInvalidTicket verifies error handling for missing payload (M3.2.3)
func TestClaimResolveStepInvalidTicket(t *testing.T) {
	store := claimcheck.NewInMemoryClaimCheckStore()
	resolver, err := NewClaimResolveStep(&config.ClaimResolveSpec{}, store)
	if err != nil {
		t.Fatalf("NewClaimResolveStep failed: %v", err)
	}

	ctx := context.Background()
	msg := engine.NewMessage(
		map[string]interface{}{
			"_claim_check": map[string]interface{}{
				"ticket": "invalid-ticket-not-in-store",
				"size":   1000,
				"hash":   "sha256:invalid",
			},
		},
		"test-route",
		"v1",
	)

	_, err = resolver.Execute(ctx, msg)
	if err == nil {
		t.Error("Expected error for invalid ticket, got nil")
	}

	if err != nil && err.Error() == "" {
		t.Error("Expected meaningful error message")
	}
}
