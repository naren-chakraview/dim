package observability

import (
	"context"
	"testing"
	"time"
)

// TestTracingIntegrationEndToEnd tests the complete tracing flow
// from message ingestion through step execution
func TestTracingIntegrationEndToEnd(t *testing.T) {
	// Initialize tracing provider
	tp := NewTracingProvider("stdout", "", 1.0) // 100% sampling for testing

	// Simulate message ingestion
	routeName := "payment-route"
	routeVersion := "sha256:abc123def456789"
	correlationID := "txn-123-456-789"
	contractVersion := "payment-v3.0"
	principal := &Principal{
		Subject: "payment-service@internal",
		Roles:   []string{"processor", "admin"},
	}

	// Start message processing span
	ctx, msgSpan := tp.StartMessageSpan(context.Background(), routeName, routeVersion, correlationID, contractVersion, principal)

	if msgSpan == nil {
		t.Fatal("expected message span to be created")
	}

	// Verify message span attributes
	verifyAttribute(t, msgSpan, "correlation_id", correlationID)
	verifyAttribute(t, msgSpan, "route_name", routeName)
	verifyAttribute(t, msgSpan, "route_version", routeVersion)
	verifyAttribute(t, msgSpan, "contract_version", contractVersion)
	verifyAttribute(t, msgSpan, "principal", principal.Subject)

	roles, ok := msgSpan.Attributes["principal.roles"].([]string)
	if !ok || len(roles) != 2 {
		t.Errorf("expected principal.roles to be [processor, admin], got %v", msgSpan.Attributes["principal.roles"])
	}

	// Simulate step execution - filter
	stepSpan1 := tp.RecordStepExecution(ctx, 0, "filter", "filter_validate_amount", 15*time.Millisecond, true, nil)

	if stepSpan1 == nil {
		t.Fatal("expected step span 1 to be created")
	}
	verifyNestedSpan(t, stepSpan1, msgSpan.SpanID, 0, "filter")

	// Simulate step execution - translate
	stepSpan2 := tp.RecordStepExecution(ctx, 1, "translate", "translate_to_internal_format", 25*time.Millisecond, true, nil)

	if stepSpan2 == nil {
		t.Fatal("expected step span 2 to be created")
	}
	verifyNestedSpan(t, stepSpan2, msgSpan.SpanID, 1, "translate")

	// Simulate step execution - authorize
	stepSpan3 := tp.RecordStepExecution(ctx, 2, "authorize", "rbac_check", 10*time.Millisecond, true, nil)

	if stepSpan3 == nil {
		t.Fatal("expected step span 3 to be created")
	}
	verifyNestedSpan(t, stepSpan3, msgSpan.SpanID, 2, "authorize")

	// Complete message processing
	tp.RecordMessageComplete(msgSpan, true, nil)

	if msgSpan.Status != "ok" {
		t.Errorf("expected message span status 'ok', got %q", msgSpan.Status)
	}
	if msgSpan.EndTime.IsZero() {
		t.Error("expected message span end time to be set")
	}

	// Verify all spans are recorded
	allSpans := tp.GetAllSpans()
	if len(allSpans) < 4 {
		t.Errorf("expected at least 4 spans (1 message + 3 steps), got %d", len(allSpans))
	}

	// Verify span hierarchy
	spansByID := make(map[string]*Span)
	for _, span := range allSpans {
		spansByID[span.SpanID] = span
	}

	// Find child spans
	childSpans := 0
	for _, span := range allSpans {
		if span.ParentSpanID == msgSpan.SpanID {
			childSpans++
		}
	}
	if childSpans < 3 {
		t.Errorf("expected at least 3 child spans, got %d", childSpans)
	}
}

// TestTracingIntegrationErrorHandling tests error recording in tracing
func TestTracingIntegrationErrorHandling(t *testing.T) {
	tp := NewTracingProvider("stdout", "", 1.0)

	routeName := "payment-route"
	ctx, msgSpan := tp.StartMessageSpan(
		context.Background(),
		routeName,
		"sha256:version",
		"txn-789",
		"payment-v2.0",
		nil,
	)

	// Simulate first two steps succeeding
	_ = tp.RecordStepExecution(ctx, 0, "filter", "filter_0", 10*time.Millisecond, true, nil)
	_ = tp.RecordStepExecution(ctx, 1, "translate", "translate_1", 20*time.Millisecond, true, nil)

	// Simulate third step failing
	testErr := &stepError{msg: "insufficient funds"}
	step3Span := tp.RecordStepExecution(ctx, 2, "authorize", "authorize_2", 5*time.Millisecond, false, testErr)

	if step3Span.Status != "error" {
		t.Errorf("expected step span status 'error', got %q", step3Span.Status)
	}
	if step3Span.Error != testErr {
		t.Error("expected step span error to be set")
	}

	// Complete message with error
	tp.RecordMessageComplete(msgSpan, false, testErr)

	if msgSpan.Status != "error" {
		t.Errorf("expected message span status 'error', got %q", msgSpan.Status)
	}
	if msgSpan.Error != testErr {
		t.Error("expected message span error to be set")
	}

	// Verify all spans are recorded
	allSpans := tp.GetAllSpans()
	if len(allSpans) != 4 {
		t.Errorf("expected 4 spans, got %d", len(allSpans))
	}

	// Count error spans
	errorSpans := 0
	for _, span := range allSpans {
		if span.Status == "error" {
			errorSpans++
		}
	}
	if errorSpans != 2 {
		t.Errorf("expected 2 error spans (message + step), got %d", errorSpans)
	}
}

// TestTracingIntegrationAttributeCarrying tests attribute propagation through nested spans
func TestTracingIntegrationAttributeCarrying(t *testing.T) {
	tp := NewTracingProvider("stdout", "", 1.0)

	principal := &Principal{
		Subject: "api-gateway@internal",
		Roles:   []string{"gateway"},
	}

	ctx, msgSpan := tp.StartMessageSpan(
		context.Background(),
		"api-route",
		"sha256:api-v1",
		"api-req-123",
		"api-contract-v1",
		principal,
	)

	// Verify message span has all attributes
	expectedAttrs := map[string]interface{}{
		"correlation_id":    "api-req-123",
		"route_name":        "api-route",
		"route_version":     "sha256:api-v1",
		"contract_version":  "api-contract-v1",
		"principal":         "api-gateway@internal",
	}

	for key, expected := range expectedAttrs {
		if val, ok := msgSpan.Attributes[key]; !ok || val != expected {
			t.Errorf("attribute %q: expected %v, got %v (ok=%v)", key, expected, val, ok)
		}
	}

	// Record step and verify it has step-specific attributes but same trace context
	stepSpan := tp.RecordStepExecution(ctx, 0, "filter", "filter_0", 15*time.Millisecond, true, nil)

	// Verify step span has its own attributes
	if stepSpan.Attributes["step_index"] != 0 {
		t.Errorf("expected step_index 0, got %v", stepSpan.Attributes["step_index"])
	}
	if stepSpan.Attributes["step_type"] != "filter" {
		t.Errorf("expected step_type 'filter', got %v", stepSpan.Attributes["step_type"])
	}

	// Verify trace context is carried (same trace ID, same parent span)
	if stepSpan.TraceID != msgSpan.TraceID {
		t.Errorf("expected same trace_id, got %q vs %q", msgSpan.TraceID, stepSpan.TraceID)
	}
	if stepSpan.ParentSpanID != msgSpan.SpanID {
		t.Errorf("expected parent_span_id to be message span id, got %q vs %q", msgSpan.SpanID, stepSpan.ParentSpanID)
	}
}

// Helper functions

func verifyAttribute(t *testing.T, span *Span, key string, expected interface{}) {
	t.Helper()
	val, ok := span.Attributes[key]
	if !ok || val != expected {
		t.Errorf("attribute %q: expected %v, got %v (ok=%v)", key, expected, val, ok)
	}
}

func verifyNestedSpan(t *testing.T, span *Span, expectedParentID string, expectedIndex int, expectedType string) {
	t.Helper()

	if span.ParentSpanID != expectedParentID {
		t.Errorf("expected parent_span_id %q, got %q", expectedParentID, span.ParentSpanID)
	}
	if span.Attributes["step_index"] != expectedIndex {
		t.Errorf("expected step_index %d, got %v", expectedIndex, span.Attributes["step_index"])
	}
	if span.Attributes["step_type"] != expectedType {
		t.Errorf("expected step_type %q, got %v", expectedType, span.Attributes["step_type"])
	}
	if span.Status != "ok" {
		t.Errorf("expected status 'ok', got %q", span.Status)
	}
}
