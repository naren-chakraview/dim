//go:build integration

package steps

import (
	"context"
	"testing"

	"github.com/naren-chakraview/dim/internal/adapters/claimcheck"
	"github.com/naren-chakraview/dim/internal/config"
	"github.com/naren-chakraview/dim/internal/engine"
)

// TestClaimCheckOrderWithAttachmentWorkflow demonstrates claim-check in order processing (M3.2.4)
// Simulates: Order message with invoice PDF → lightweight message → compliance step retrieves PDF
func TestClaimCheckOrderWithAttachmentWorkflow(t *testing.T) {
	store := claimcheck.NewInMemoryClaimCheckStore()
	ctx := context.Background()

	// Simulate large invoice PDF (5 MB)
	largeInvoicePDF := make([]byte, 5*1024*1024)
	for i := range largeInvoicePDF {
		largeInvoicePDF[i] = byte(i % 256)
	}

	// Step 1: Order message arrives with large attachment
	orderMessage := engine.NewMessage(
		map[string]interface{}{
			"order_id":    "ORD-2026-001",
			"customer":    "ACME Corp",
			"amount":      50000.00,
			"invoice_pdf": string(largeInvoicePDF), // Large payload
		},
		"order-processing",
		"v1",
	)
	orderMessage.Metadata.CorrelationID = "corr-2026-001"

	// Step 2: Claim-check step - store large payload, keep lightweight message
	checkSpec := &config.ClaimCheckSpec{
		PayloadField:    "body",
		RemovePayload:   true, // Remove payload from message body
		TicketFieldPath: "_claim_check",
	}

	checkStep, err := NewClaimCheckStep(checkSpec, store)
	if err != nil {
		t.Fatalf("Failed to create claim-check step: %v", err)
	}

	checkedMsg, err := checkStep.Execute(ctx, orderMessage)
	if err != nil {
		t.Fatalf("Claim-check execute failed: %v", err)
	}

	// Verify message is now lightweight
	bodyMap, _ := checkedMsg.Body.(map[string]interface{})
	if _, exists := bodyMap["invoice_pdf"]; exists {
		t.Error("Large invoice_pdf should be removed from lightweight message")
	}

	// Verify ticket is present
	ticketInfo, ok := bodyMap["_claim_check"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected claim-check ticket in body")
	}

	ticket, _ := ticketInfo["ticket"].(string)
	size, _ := ticketInfo["size"].(int64)
	if size < 5*1024*1024 {
		t.Errorf("Expected size ~5MB, got %d bytes", size)
	}

	t.Logf("✓ Order message stored with ticket %s (size: %d bytes)", ticket, size)

	// Step 3: Message travels through routing, filtering, etc. (no large payload overhead)
	// Correlation ID preserved through pipeline
	if checkedMsg.Metadata.CorrelationID != "corr-2026-001" {
		t.Error("Correlation ID should be preserved")
	}

	// Step 4: Compliance step needs to retrieve PDF for validation
	resolveSpec := &config.ClaimResolveSpec{
		TicketPath:      "_claim_check",
		OutputFieldPath: "body",
	}

	resolveStep, err := NewClaimResolveStep(resolveSpec, store)
	if err != nil {
		t.Fatalf("Failed to create resolve step: %v", err)
	}

	resolved, err := resolveStep.Execute(ctx, checkedMsg)
	if err != nil {
		t.Fatalf("Resolve execute failed: %v", err)
	}

	// Verify payload is restored
	if resolved.Body == nil {
		t.Fatal("Expected payload to be restored")
	}

	t.Logf("✓ Compliance step retrieved payload from claim-check store")
}

// TestClaimCheckMultipleAttachments verifies handling multiple large payloads (M3.2.4)
func TestClaimCheckMultipleAttachments(t *testing.T) {
	store := claimcheck.NewInMemoryClaimCheckStore()
	ctx := context.Background()

	// Document package with multiple large files
	docPackage := map[string]interface{}{
		"package_id":      "PKG-2026-001",
		"document_count":  3,
		"invoice":         make([]byte, 2*1024*1024),    // 2 MB
		"receipt":         make([]byte, 1*1024*1024),    // 1 MB
		"compliance_cert": make([]byte, 1024*1024),      // 1 MB
	}

	msg := engine.NewMessage(docPackage, "document-processing", "v1")

	// Claim-check the entire document package
	checkStep, err := NewClaimCheckStep(&config.ClaimCheckSpec{
		RemovePayload: true,
	}, store)
	if err != nil {
		t.Fatalf("Failed to create claim-check step: %v", err)
	}

	checked, err := checkStep.Execute(ctx, msg)
	if err != nil {
		t.Fatalf("Claim-check execute failed: %v", err)
	}

	// Verify lightweight message
	bodyMap, _ := checked.Body.(map[string]interface{})
	if _, exists := bodyMap["invoice"]; exists {
		t.Error("Large attachments should be removed")
	}

	ticketInfo, _ := bodyMap["_claim_check"].(map[string]interface{})
	size, _ := ticketInfo["size"].(int64)
	if size < 4*1024*1024 {
		t.Errorf("Expected size ~4MB total, got %d bytes", size)
	}

	t.Logf("✓ Multi-document package stored with total size: %d bytes", size)

	// Resolve and verify
	resolveStep, _ := NewClaimResolveStep(&config.ClaimResolveSpec{}, store)
	resolved, err := resolveStep.Execute(ctx, checked)
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}

	if resolved.Body == nil {
		t.Fatal("Expected payload to be restored")
	}
}

// TestClaimCheckVerificationFailure verifies integrity checks work (M3.2.4)
func TestClaimCheckVerificationFailure(t *testing.T) {
	store := claimcheck.NewInMemoryClaimCheckStore()
	ctx := context.Background()

	// Store original payload
	payload := []byte("sensitive data that should not be tampered with")
	metadata := claimcheck.ClaimCheckMetadata{
		ContentType: "text/plain",
	}

	ticket, _ := store.Store(ctx, payload, metadata)

	// Retrieve and verify hash
	_, meta, _ := store.Retrieve(ctx, ticket)
	valid, _ := store.Verify(ctx, ticket, meta.Hash)
	if !valid {
		t.Error("Verification should succeed for unmodified payload")
	}

	// Verify with wrong hash should fail
	valid, _ = store.Verify(ctx, ticket, "sha256:wronghash")
	if valid {
		t.Error("Verification should fail with wrong hash")
	}

	t.Logf("✓ Integrity verification working correctly")
}

// TestClaimCheckCleanup verifies deletion works for cleanup (M3.2.4)
func TestClaimCheckCleanup(t *testing.T) {
	store := claimcheck.NewInMemoryClaimCheckStore()
	ctx := context.Background()

	// Store payload
	payload := []byte("temporary data")
	metadata := claimcheck.ClaimCheckMetadata{
		ContentType: "text/plain",
	}

	ticket, _ := store.Store(ctx, payload, metadata)

	// Verify it exists
	_, _, err := store.Retrieve(ctx, ticket)
	if err != nil {
		t.Fatal("Payload should exist after store")
	}

	// Delete
	err = store.Delete(ctx, ticket)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Verify it's gone
	_, _, err = store.Retrieve(ctx, ticket)
	if err == nil {
		t.Error("Payload should not exist after delete")
	}

	t.Logf("✓ Cleanup successful - garbage collection working")
}
