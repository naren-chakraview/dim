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

	_, span := tp.StartMessageSpan(context.Background(), "test-route", "msg-123")

	if span == nil {
		t.Fatal("expected non-nil span")
	}
	if span.TraceID == "" {
		t.Fatal("expected non-empty trace_id")
	}
	if span.SpanID == "" {
		t.Fatal("expected non-empty span_id")
	}
	if span.Name != "message_processing.test-route" {
		t.Errorf("expected name 'message_processing.test-route', got %q", span.Name)
	}
	if span.Status != "ok" {
		t.Errorf("expected status 'ok', got %q", span.Status)
	}

	// Note: context is returned but we only check the span in this test
}

func TestTracingProvider_SamplingRate(t *testing.T) {
	// Test with 0% sampling (no spans recorded)
	tp := NewTracingProvider("stdout", "", 0.0)

	_, span := tp.StartMessageSpan(context.Background(), "test-route", "msg-123")
	if span != nil {
		t.Error("expected nil span with 0% sampling")
	}
}

func TestTracingProvider_RecordStepExecution(t *testing.T) {
	tp := NewTracingProvider("stdout", "", 1.0)

	// Start a message span first
	ctx, msgSpan := tp.StartMessageSpan(context.Background(), "test-route", "msg-123")

	// Record a step execution
	stepSpan := tp.RecordStepExecution(ctx, "filter", 25)

	if stepSpan == nil {
		t.Fatal("expected non-nil step span")
	}
	if stepSpan.ParentSpanID != msgSpan.SpanID {
		t.Errorf("expected parent_span_id %q, got %q", msgSpan.SpanID, stepSpan.ParentSpanID)
	}
	if stepSpan.Name != "step_execution.filter" {
		t.Errorf("expected name 'step_execution.filter', got %q", stepSpan.Name)
	}
	if stepSpan.Duration != 25*time.Millisecond {
		t.Errorf("expected duration 25ms, got %v", stepSpan.Duration)
	}
}

func TestTracingProvider_EndSpan(t *testing.T) {
	tp := NewTracingProvider("stdout", "", 1.0)

	_, span := tp.StartMessageSpan(context.Background(), "test-route", "msg-123")
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

	_, span := tp.StartMessageSpan(context.Background(), "test-route", "msg-123")

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

	_, span := tp.StartMessageSpan(context.Background(), "test-route", "msg-123")

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

	_, span := tp.StartMessageSpan(context.Background(), "test-route", "msg-123")

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
	_, _ = tp.StartMessageSpan(context.Background(), "route-a", "msg-1")
	_, _ = tp.StartMessageSpan(context.Background(), "route-b", "msg-2")

	// Get all spans
	allSpans := tp.GetAllSpans()
	if len(allSpans) < 2 {
		t.Errorf("expected at least 2 spans, got %d", len(allSpans))
	}
}

func TestTracingProvider_NestedSpans(t *testing.T) {
	tp := NewTracingProvider("stdout", "", 1.0)

	// Start a message span
	ctx, msgSpan := tp.StartMessageSpan(context.Background(), "test-route", "msg-123")

	// Record nested step spans
	stepSpan1 := tp.RecordStepExecution(ctx, "filter", 10)
	stepSpan2 := tp.RecordStepExecution(ctx, "translate", 15)

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

	_, span1 := tp.StartMessageSpan(context.Background(), "test-route", "msg-1")
	if span1 == nil {
		t.Fatal("expected span before disable")
	}

	tp.Disable()

	_, span2 := tp.StartMessageSpan(context.Background(), "test-route", "msg-2")
	if span2 != nil {
		t.Error("expected nil span after disable")
	}
}

func TestTracingProvider_Reset(t *testing.T) {
	tp := NewTracingProvider("stdout", "", 1.0)

	// Create some spans
	_, _ = tp.StartMessageSpan(context.Background(), "test-route", "msg-123")

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
		_, span := tp.StartMessageSpan(context.Background(), "test-route", "msg")
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
		_, span := tp.StartMessageSpan(context.Background(), "test-route", "msg")
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
				tp.StartMessageSpan(context.Background(), "route", "msg")
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
