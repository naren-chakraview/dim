package observability

import (
	"context"
	"testing"
	"time"
)

func TestTracingProvider_NewTracingProvider(t *testing.T) {
	tp := NewTracingProvider("stdout", "", 0.1)
	if tp == nil {
		t.Fatal("expected non-nil TracingProvider")
	}
	if !tp.enabled {
		t.Fatal("expected tracing to be enabled by default")
	}
	if tp.exporter != "stdout" {
		t.Errorf("expected exporter 'stdout', got %q", tp.exporter)
	}
	if tp.sampleRate != 0.1 {
		t.Errorf("expected sample_rate 0.1, got %.2f", tp.sampleRate)
	}
}

func TestTracingProvider_NormalizeSampleRate(t *testing.T) {
	tests := []struct {
		input    float64
		expected float64
	}{
		{-0.5, 0},    // Negative becomes 0
		{0, 0},       // 0 is valid
		{0.5, 0.5},   // Middle value
		{1.0, 1.0},   // 1.0 is valid
		{1.5, 1.0},   // > 1.0 becomes 1.0
	}

	for _, tc := range tests {
		tp := NewTracingProvider("stdout", "", tc.input)
		if tp.sampleRate != tc.expected {
			t.Errorf("sample_rate %f: expected %f, got %f", tc.input, tc.expected, tp.sampleRate)
		}
	}
}

func TestTracingProvider_StartMessageSpan(t *testing.T) {
	tp := NewTracingProvider("stdout", "", 1.0) // 100% sampling

	_, span := tp.StartMessageSpan(context.Background(), "test-route", "route-v1-hash", "corr-123", "contract-v1", nil)

	if span == nil {
		t.Fatal("expected non-nil span")
	}
	if span.TraceID == "" {
		t.Fatal("expected non-empty trace_id")
	}
	if span.SpanID == "" {
		t.Fatal("expected non-empty span_id")
	}
	if span.Name != "message_processing" {
		t.Errorf("expected name 'message_processing', got %q", span.Name)
	}
	if span.Status != "ok" {
		t.Errorf("expected status 'ok', got %q", span.Status)
	}

	// Check attributes
	if span.Attributes["correlation_id"] != "corr-123" {
		t.Errorf("expected correlation_id 'corr-123', got %v", span.Attributes["correlation_id"])
	}
	if span.Attributes["route_name"] != "test-route" {
		t.Errorf("expected route_name 'test-route', got %v", span.Attributes["route_name"])
	}
	if span.Attributes["route_version"] != "route-v1-hash" {
		t.Errorf("expected route_version 'route-v1-hash', got %v", span.Attributes["route_version"])
	}
	if span.Attributes["contract_version"] != "contract-v1" {
		t.Errorf("expected contract_version 'contract-v1', got %v", span.Attributes["contract_version"])
	}

	// Note: context is returned but we only check the span in this test
}

func TestTracingProvider_SamplingRate(t *testing.T) {
	// Test with 0% sampling (no spans recorded)
	tp := NewTracingProvider("stdout", "", 0.0)

	_, span := tp.StartMessageSpan(context.Background(), "test-route", "route-v1", "corr-123", "", nil)
	if span != nil {
		t.Error("expected nil span with 0% sampling")
	}
}

func TestTracingProvider_RecordStepExecution(t *testing.T) {
	tp := NewTracingProvider("stdout", "", 1.0)

	// Start a message span first
	ctx, msgSpan := tp.StartMessageSpan(context.Background(), "test-route", "route-v1", "corr-123", "", nil)

	// Record a step execution
	stepSpan := tp.RecordStepExecution(ctx, 0, "filter", "filter_0", 25*time.Millisecond, true, nil)

	if stepSpan == nil {
		t.Fatal("expected non-nil step span")
	}
	if stepSpan.ParentSpanID != msgSpan.SpanID {
		t.Errorf("expected parent_span_id %q, got %q", msgSpan.SpanID, stepSpan.ParentSpanID)
	}
	if stepSpan.Name != "step_execution" {
		t.Errorf("expected name 'step_execution', got %q", stepSpan.Name)
	}
	if stepSpan.Duration != 25*time.Millisecond {
		t.Errorf("expected duration 25ms, got %v", stepSpan.Duration)
	}

	// Check attributes
	if stepSpan.Attributes["step_index"] != 0 {
		t.Errorf("expected step_index 0, got %v", stepSpan.Attributes["step_index"])
	}
	if stepSpan.Attributes["step_type"] != "filter" {
		t.Errorf("expected step_type 'filter', got %v", stepSpan.Attributes["step_type"])
	}
	if stepSpan.Attributes["step_name"] != "filter_0" {
		t.Errorf("expected step_name 'filter_0', got %v", stepSpan.Attributes["step_name"])
	}
	if stepSpan.Status != "ok" {
		t.Errorf("expected status 'ok', got %q", stepSpan.Status)
	}
}

func TestTracingProvider_EndSpan(t *testing.T) {
	tp := NewTracingProvider("stdout", "", 1.0)

	_, span := tp.StartMessageSpan(context.Background(), "test-route", "route-v1", "corr-123", "", nil)
	if span.StartTime.IsZero() {
		t.Fatal("expected start time to be set")
	}

	// End the span
	tp.EndSpan(span)

	if span.EndTime.IsZero() {
		t.Fatal("expected end time to be set")
	}
	if span.Duration <= 0 {
		t.Errorf("expected positive duration, got %v", span.Duration)
	}
}

func TestTracingProvider_AddSpanEvent(t *testing.T) {
	tp := NewTracingProvider("stdout", "", 1.0)

	_, span := tp.StartMessageSpan(context.Background(), "test-route", "route-v1", "corr-123", "", nil)

	// Add an event
	attrs := map[string]interface{}{"key": "value"}
	tp.AddSpanEvent(span, "test-event", attrs)

	if len(span.Events) != 1 {
		t.Errorf("expected 1 event, got %d", len(span.Events))
	}
	if span.Events[0].Name != "test-event" {
		t.Errorf("expected event name 'test-event', got %q", span.Events[0].Name)
	}
}

func TestTracingProvider_RecordSpanError(t *testing.T) {
	tp := NewTracingProvider("stdout", "", 1.0)

	_, span := tp.StartMessageSpan(context.Background(), "test-route", "route-v1", "corr-123", "", nil)

	// Record an error
	testErr := &stepError{msg: "test error"}
	tp.RecordSpanError(span, testErr)

	if span.Status != "error" {
		t.Errorf("expected status 'error', got %q", span.Status)
	}
	if span.Error != testErr {
		t.Errorf("expected error to be set")
	}
}

func TestTracingProvider_GetSpan(t *testing.T) {
	tp := NewTracingProvider("stdout", "", 1.0)

	_, span := tp.StartMessageSpan(context.Background(), "test-route", "route-v1", "corr-123", "", nil)

	// Retrieve the span
	retrieved := tp.GetSpan(span.SpanID)
	if retrieved == nil {
		t.Fatal("expected to retrieve span")
	}
	if retrieved.SpanID != span.SpanID {
		t.Errorf("expected span_id %q, got %q", span.SpanID, retrieved.SpanID)
	}
}

func TestTracingProvider_GetAllSpans(t *testing.T) {
	tp := NewTracingProvider("stdout", "", 1.0)

	// Start multiple spans
	_, _ = tp.StartMessageSpan(context.Background(), "route-a", "v1", "corr-1", "", nil)
	_, _ = tp.StartMessageSpan(context.Background(), "route-b", "v1", "corr-2", "", nil)

	// Get all spans
	allSpans := tp.GetAllSpans()
	if len(allSpans) < 2 {
		t.Errorf("expected at least 2 spans, got %d", len(allSpans))
	}
}

func TestTracingProvider_NestedSpans(t *testing.T) {
	tp := NewTracingProvider("stdout", "", 1.0)

	// Start a message span
	ctx, msgSpan := tp.StartMessageSpan(context.Background(), "test-route", "route-v1", "corr-123", "", nil)

	// Record nested step spans
	stepSpan1 := tp.RecordStepExecution(ctx, 0, "filter", "filter_0", 10*time.Millisecond, true, nil)
	stepSpan2 := tp.RecordStepExecution(ctx, 1, "translate", "translate_1", 15*time.Millisecond, true, nil)

	if stepSpan1.ParentSpanID != msgSpan.SpanID {
		t.Errorf("expected step1 parent_span_id %q, got %q", msgSpan.SpanID, stepSpan1.ParentSpanID)
	}
	if stepSpan2.ParentSpanID != msgSpan.SpanID {
		t.Errorf("expected step2 parent_span_id %q, got %q", msgSpan.SpanID, stepSpan2.ParentSpanID)
	}
	if stepSpan1.SpanID == stepSpan2.SpanID {
		t.Error("expected different span IDs for different steps")
	}
}

func TestTracingProvider_Disable(t *testing.T) {
	tp := NewTracingProvider("stdout", "", 1.0)

	_, span1 := tp.StartMessageSpan(context.Background(), "test-route", "route-v1", "corr-1", "", nil)
	if span1 == nil {
		t.Fatal("expected span before disable")
	}

	tp.Disable()

	_, span2 := tp.StartMessageSpan(context.Background(), "test-route", "route-v1", "corr-2", "", nil)
	if span2 != nil {
		t.Error("expected nil span after disable")
	}
}

func TestTracingProvider_Reset(t *testing.T) {
	tp := NewTracingProvider("stdout", "", 1.0)

	// Create some spans
	_, _ = tp.StartMessageSpan(context.Background(), "test-route", "route-v1", "corr-123", "", nil)

	allSpans1 := tp.GetAllSpans()
	if len(allSpans1) == 0 {
		t.Fatal("expected spans before reset")
	}

	tp.Reset()

	allSpans2 := tp.GetAllSpans()
	if len(allSpans2) != 0 {
		t.Errorf("expected no spans after reset, got %d", len(allSpans2))
	}
}

func TestTracingProvider_MultipleExporters(t *testing.T) {
	exporters := []string{"stdout", "jaeger", "datadog"}

	for _, exporter := range exporters {
		tp := NewTracingProvider(exporter, "http://localhost:14268/api/traces", 1.0)
		if tp.exporter != exporter {
			t.Errorf("expected exporter %q, got %q", exporter, tp.exporter)
		}
	}
}

func TestTracingProvider_DefaultExporter(t *testing.T) {
	tp := NewTracingProvider("", "", 0.1)
	if tp.exporter != "stdout" {
		t.Errorf("expected default exporter 'stdout', got %q", tp.exporter)
	}
}

func TestTracingProvider_TraceIDUniqueness(t *testing.T) {
	tp := NewTracingProvider("stdout", "", 1.0)

	traceIDs := make(map[string]bool)
	for i := 0; i < 10; i++ {
		_, span := tp.StartMessageSpan(context.Background(), "test-route", "v1", "corr", "", nil)
		if traceIDs[span.TraceID] {
			t.Errorf("duplicate trace_id: %s", span.TraceID)
		}
		traceIDs[span.TraceID] = true
	}
}

func TestTracingProvider_SpanIDUniqueness(t *testing.T) {
	tp := NewTracingProvider("stdout", "", 1.0)

	spanIDs := make(map[string]bool)
	for i := 0; i < 10; i++ {
		_, span := tp.StartMessageSpan(context.Background(), "test-route", "v1", "corr", "", nil)
		if spanIDs[span.SpanID] {
			t.Errorf("duplicate span_id: %s", span.SpanID)
		}
		spanIDs[span.SpanID] = true
	}
}

func TestTracingProvider_ThreadSafety(t *testing.T) {
	tp := NewTracingProvider("stdout", "", 1.0)

	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(idx int) {
			for j := 0; j < 100; j++ {
				tp.StartMessageSpan(context.Background(), "route", "v1", "corr", "", nil)
			}
			done <- true
		}(i)
	}

	for i := 0; i < 10; i++ {
		<-done
	}

	allSpans := tp.GetAllSpans()
	if len(allSpans) < 1000 {
		t.Errorf("expected at least 1000 spans, got %d", len(allSpans))
	}
}

// Stub for testing error types
type stepError struct {
	msg string
}

func (e *stepError) Error() string {
	return e.msg
}

// Test for StartMessageSpan with principal attributes (M0.5.1)
func TestTracingProvider_MessageSpanWithPrincipal(t *testing.T) {
	tp := NewTracingProvider("stdout", "", 1.0)

	principal := &Principal{
		Subject: "user@example.com",
		Roles:   []string{"admin", "viewer"},
	}

	_, span := tp.StartMessageSpan(context.Background(), "secure-route", "route-v2", "corr-456", "contract-v2", principal)

	if span == nil {
		t.Fatal("expected non-nil span")
	}

	// Check principal attributes
	if span.Attributes["principal"] != "user@example.com" {
		t.Errorf("expected principal 'user@example.com', got %v", span.Attributes["principal"])
	}

	roles, ok := span.Attributes["principal.roles"].([]string)
	if !ok {
		t.Errorf("expected principal.roles to be []string, got %T", span.Attributes["principal.roles"])
	}
	if len(roles) != 2 || roles[0] != "admin" || roles[1] != "viewer" {
		t.Errorf("expected roles [admin, viewer], got %v", roles)
	}
}

// Test for RecordMessageComplete (M0.5.1)
func TestTracingProvider_RecordMessageComplete(t *testing.T) {
	tp := NewTracingProvider("stdout", "", 1.0)

	_, span := tp.StartMessageSpan(context.Background(), "test-route", "route-v1", "corr-123", "", nil)

	// Record success
	tp.RecordMessageComplete(span, true, nil)

	if span.Status != "ok" {
		t.Errorf("expected status 'ok', got %q", span.Status)
	}
	if span.EndTime.IsZero() {
		t.Error("expected end_time to be set")
	}
	if span.Duration <= 0 {
		t.Errorf("expected positive duration, got %v", span.Duration)
	}
}

// Test for RecordMessageComplete with error (M0.5.1)
func TestTracingProvider_RecordMessageCompleteWithError(t *testing.T) {
	tp := NewTracingProvider("stdout", "", 1.0)

	_, span := tp.StartMessageSpan(context.Background(), "test-route", "route-v1", "corr-123", "", nil)

	testErr := &stepError{msg: "processing failed"}
	tp.RecordMessageComplete(span, false, testErr)

	if span.Status != "error" {
		t.Errorf("expected status 'error', got %q", span.Status)
	}
	if span.Error != testErr {
		t.Errorf("expected error to be set")
	}
}

// Test for RecordStepExecution with error status (M0.5.1)
func TestTracingProvider_RecordStepExecutionWithError(t *testing.T) {
	tp := NewTracingProvider("stdout", "", 1.0)

	ctx, _ := tp.StartMessageSpan(context.Background(), "test-route", "route-v1", "corr-123", "", nil)

	testErr := &stepError{msg: "step failed"}
	stepSpan := tp.RecordStepExecution(ctx, 1, "translate", "translate_1", 50*time.Millisecond, false, testErr)

	if stepSpan == nil {
		t.Fatal("expected non-nil step span")
	}
	if stepSpan.Status != "error" {
		t.Errorf("expected status 'error', got %q", stepSpan.Status)
	}
	if stepSpan.Error != testErr {
		t.Errorf("expected error to be set")
	}
}

// Test for nested step spans (M0.5.1)
func TestTracingProvider_NestedStepSpans(t *testing.T) {
	tp := NewTracingProvider("stdout", "", 1.0)

	ctx, msgSpan := tp.StartMessageSpan(context.Background(), "test-route", "route-v1", "corr-123", "", nil)

	// Record multiple nested step spans
	stepSpan1 := tp.RecordStepExecution(ctx, 0, "filter", "filter_0", 10*time.Millisecond, true, nil)
	stepSpan2 := tp.RecordStepExecution(ctx, 1, "translate", "translate_1", 20*time.Millisecond, true, nil)
	stepSpan3 := tp.RecordStepExecution(ctx, 2, "route", "route_2", 15*time.Millisecond, true, nil)

	// All should have the same parent
	if stepSpan1.ParentSpanID != msgSpan.SpanID {
		t.Errorf("step1: expected parent_span_id %q, got %q", msgSpan.SpanID, stepSpan1.ParentSpanID)
	}
	if stepSpan2.ParentSpanID != msgSpan.SpanID {
		t.Errorf("step2: expected parent_span_id %q, got %q", msgSpan.SpanID, stepSpan2.ParentSpanID)
	}
	if stepSpan3.ParentSpanID != msgSpan.SpanID {
		t.Errorf("step3: expected parent_span_id %q, got %q", msgSpan.SpanID, stepSpan3.ParentSpanID)
	}

	// All should have different span IDs
	ids := map[string]bool{
		stepSpan1.SpanID: true,
		stepSpan2.SpanID: true,
		stepSpan3.SpanID: true,
	}
	if len(ids) != 3 {
		t.Error("expected 3 different span IDs")
	}

	// Verify step indices
	if stepSpan1.Attributes["step_index"] != 0 {
		t.Errorf("expected step_index 0 for stepSpan1, got %v", stepSpan1.Attributes["step_index"])
	}
	if stepSpan2.Attributes["step_index"] != 1 {
		t.Errorf("expected step_index 1 for stepSpan2, got %v", stepSpan2.Attributes["step_index"])
	}
	if stepSpan3.Attributes["step_index"] != 2 {
		t.Errorf("expected step_index 2 for stepSpan3, got %v", stepSpan3.Attributes["step_index"])
	}
}

// Test for span attribute population (M0.5.1)
func TestTracingProvider_SpanAttributePopulation(t *testing.T) {
	tp := NewTracingProvider("stdout", "", 1.0)

	principal := &Principal{
		Subject: "service@internal",
		Roles:   []string{"writer"},
	}

	_, span := tp.StartMessageSpan(
		context.Background(),
		"payment-route",
		"sha256:abc123def456",
		"txn-789-xyz",
		"payment-v3.0",
		principal,
	)

	// Verify all attributes are populated
	tests := []struct {
		key      string
		expected interface{}
	}{
		{"correlation_id", "txn-789-xyz"},
		{"route_name", "payment-route"},
		{"route_version", "sha256:abc123def456"},
		{"contract_version", "payment-v3.0"},
		{"principal", "service@internal"},
	}

	for _, tc := range tests {
		if val, ok := span.Attributes[tc.key]; !ok || val != tc.expected {
			t.Errorf("attribute %q: expected %v, got %v (ok=%v)", tc.key, tc.expected, val, ok)
		}
	}
}

// Test for sampling with various rates (M0.5.1)
func TestTracingProvider_SamplingBehavior(t *testing.T) {
	tests := []struct {
		sampleRate float64
		expectSpan bool
	}{
		{0.0, false}, // 0% should never generate spans
		{0.5, true},  // 50% might generate spans (probabilistic, so we just check it's initialized)
		{1.0, true},  // 100% should always generate spans
	}

	for _, tc := range tests {
		tp := NewTracingProvider("stdout", "", tc.sampleRate)
		_, span := tp.StartMessageSpan(context.Background(), "route", "v1", "corr", "", nil)

		if tc.sampleRate == 0.0 && span != nil {
			t.Errorf("sample_rate=0.0 should never create spans, got %v", span)
		}
		if tc.sampleRate == 1.0 && span == nil {
			t.Error("sample_rate=1.0 should always create spans")
		}
	}
}
