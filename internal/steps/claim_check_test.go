package steps

import (
	"context"
	"testing"

	"github.com/naren-chakraview/dim/internal/adapters/claimcheck"
	"github.com/naren-chakraview/dim/internal/config"
	"github.com/naren-chakraview/dim/internal/engine"
)

// TestClaimCheckStepBasic verifies basic claim-check functionality (M3.2.2)
func TestClaimCheckStepBasic(t *testing.T) {
	store := claimcheck.NewInMemoryClaimCheckStore()
	spec := &config.ClaimCheckSpec{
		PayloadField:    "body",
		RemovePayload:   true,
		TicketFieldPath: "_claim_check",
	}

	step, err := NewClaimCheckStep(spec, store)
	if err != nil {
		t.Fatalf("NewClaimCheckStep failed: %v", err)
	}

	ctx := context.Background()
	msg := engine.NewMessage(
		map[string]interface{}{
			"order_id": "ORD-001",
			"amount":   100.0,
			"invoice":  "large pdf content here",
		},
		"order-processing",
		"v1",
	)

	result, err := step.Execute(ctx, msg)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if result == nil {
		t.Fatal("Expected result message, got nil")
	}

	// Verify message has claim-check ticket
	bodyMap, ok := result.Body.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected map body, got %T", result.Body)
	}

	claimCheckInfo, ok := bodyMap["_claim_check"]
	if !ok {
		t.Error("Expected _claim_check in body")
	}

	// Verify ticket structure
	ticketMap, ok := claimCheckInfo.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected ticket to be map, got %T", claimCheckInfo)
	}

	ticket, ok := ticketMap["ticket"].(string)
	if !ok || ticket == "" {
		t.Error("Expected non-empty ticket")
	}

	size, ok := ticketMap["size"].(int64)
	if !ok || size <= 0 {
		t.Errorf("Expected positive size, got %v", size)
	}

	hash, ok := ticketMap["hash"].(string)
	if !ok || hash == "" {
		t.Error("Expected non-empty hash")
	}
}

// TestClaimCheckStepWithoutRemoval verifies claim-check preserves body (M3.2.2)
func TestClaimCheckStepWithoutRemoval(t *testing.T) {
	store := claimcheck.NewInMemoryClaimCheckStore()
	spec := &config.ClaimCheckSpec{
		PayloadField:    "body",
		RemovePayload:   false, // Keep payload in message
		TicketFieldPath: "_claim_check",
	}

	step, err := NewClaimCheckStep(spec, store)
	if err != nil {
		t.Fatalf("NewClaimCheckStep failed: %v", err)
	}

	ctx := context.Background()
	msg := engine.NewMessage(
		map[string]interface{}{
			"order_id": "ORD-002",
			"amount":   200.0,
		},
		"test-route",
		"v1",
	)

	result, err := step.Execute(ctx, msg)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Verify original fields are still in body
	bodyMap, ok := result.Body.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected map body, got %T", result.Body)
	}

	if _, ok := bodyMap["order_id"]; !ok {
		t.Error("Expected order_id in body")
	}

	if _, ok := bodyMap["_claim_check"]; !ok {
		t.Error("Expected _claim_check in body")
	}
}

// TestResolveClaimCheck verifies retrieval of claim-checked payload (M3.2.3)
func TestResolveClaimCheck(t *testing.T) {
	store := claimcheck.NewInMemoryClaimCheckStore()
	spec := &config.ClaimCheckSpec{
		PayloadField:    "body",
		RemovePayload:   true,
		TicketFieldPath: "_claim_check",
	}

	step, err := NewClaimCheckStep(spec, store)
	if err != nil {
		t.Fatalf("NewClaimCheckStep failed: %v", err)
	}

	ctx := context.Background()
	originalBody := map[string]interface{}{
		"order_id":  "ORD-003",
		"amount":    300.0,
		"invoice":   "pdf data",
		"timestamp": "2026-09-06",
	}

	msg := engine.NewMessage(originalBody, "test-route", "v1")

	// Store the payload
	checked, err := step.Execute(ctx, msg)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Resolve the claim-check
	payload, err := ResolveClaimCheck(ctx, store, checked)
	if err != nil {
		t.Fatalf("ResolveClaimCheck failed: %v", err)
	}

	if payload == nil {
		t.Fatal("Expected payload, got nil")
	}

	// Verify payload is not empty
	payloadBytes, ok := payload.([]byte)
	if !ok {
		t.Fatalf("Expected []byte payload, got %T", payload)
	}

	if len(payloadBytes) == 0 {
		t.Error("Expected non-empty payload")
	}
}

// TestClaimCheckStoreInMemory verifies in-memory store implementation (M3.2.2)
func TestClaimCheckStoreInMemory(t *testing.T) {
	store := claimcheck.NewInMemoryClaimCheckStore()
	ctx := context.Background()

	payload := []byte("test payload data")
	metadata := claimcheck.ClaimCheckMetadata{
		ContentType: "text/plain",
		Size:        int64(len(payload)),
	}

	// Store
	ticket, err := store.Store(ctx, payload, metadata)
	if err != nil {
		t.Fatalf("Store failed: %v", err)
	}

	if ticket == "" {
		t.Error("Expected non-empty ticket")
	}

	// Retrieve
	retrieved, retrievedMeta, err := store.Retrieve(ctx, ticket)
	if err != nil {
		t.Fatalf("Retrieve failed: %v", err)
	}

	if string(retrieved) != "test payload data" {
		t.Errorf("Expected 'test payload data', got %q", string(retrieved))
	}

	if retrievedMeta.Size != int64(len(payload)) {
		t.Errorf("Expected size %d, got %d", len(payload), retrievedMeta.Size)
	}

	// Verify
	valid, err := store.Verify(ctx, ticket, retrievedMeta.Hash)
	if err != nil {
		t.Fatalf("Verify failed: %v", err)
	}

	if !valid {
		t.Error("Verification failed")
	}

	// Delete
	err = store.Delete(ctx, ticket)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Verify deletion
	_, _, err = store.Retrieve(ctx, ticket)
	if err == nil {
		t.Error("Expected error after delete, but retrieval succeeded")
	}
}

// TestClaimCheckCorrelationIDPreserved verifies metadata preservation (M3.2.2)
func TestClaimCheckCorrelationIDPreserved(t *testing.T) {
	store := claimcheck.NewInMemoryClaimCheckStore()
	spec := &config.ClaimCheckSpec{
		RemovePayload: true,
	}

	step, err := NewClaimCheckStep(spec, store)
	if err != nil {
		t.Fatalf("NewClaimCheckStep failed: %v", err)
	}

	ctx := context.Background()
	msg := engine.NewMessage(
		map[string]interface{}{"data": "test"},
		"test-route",
		"v1",
	)
	msg.Metadata.CorrelationID = "corr-123"

	result, err := step.Execute(ctx, msg)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if result.Metadata.CorrelationID != "corr-123" {
		t.Errorf("Expected correlation ID 'corr-123', got %q", result.Metadata.CorrelationID)
	}
}
