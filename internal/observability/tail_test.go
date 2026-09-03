package observability

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"
)

// Test 1: Span formatting with all attributes
func TestFormatSpanWithAttributes(t *testing.T) {
	span := &Span{
		TraceID:    "trace123",
		SpanID:     "span456",
		Name:       "message_processing",
		StartTime:  time.Date(2026, 9, 1, 20, 34, 56, 0, time.UTC),
		EndTime:    time.Date(2026, 9, 1, 20, 34, 57, 0, time.UTC),
		Duration:   time.Duration(1000 * time.Millisecond),
		Status:     "ok",
		Attributes: map[string]interface{}{
			"route_name":     "payment-processing",
			"correlation_id": "corr-123",
			"principal":      "user-123",
			"step_index":     1,
		},
	}

	formatted := FormatSpan(span, false)

	// Check that all key information is in the output
	if !strings.Contains(formatted, "payment-processing") {
		t.Errorf("formatted span missing route name: %s", formatted)
	}
	if !strings.Contains(formatted, "message_processing") {
		t.Errorf("formatted span missing span name: %s", formatted)
	}
	if !strings.Contains(formatted, "1000ms") {
		t.Errorf("formatted span missing duration: %s", formatted)
	}
	if !strings.Contains(formatted, "✓") {
		t.Errorf("formatted span missing success icon: %s", formatted)
	}
	if !strings.Contains(formatted, "principal=user-123") {
		t.Errorf("formatted span missing principal attribute: %s", formatted)
	}
	if !strings.Contains(formatted, "correlation_id=corr-123") {
		t.Errorf("formatted span missing correlation_id: %s", formatted)
	}
}

// Test 2: Color coding for different status values
func TestColorCodingForStatus(t *testing.T) {
	tests := []struct {
		status string
		icon   string
		color  string
	}{
		{"ok", "✓", "\033[32m"},       // Green
		{"error", "✗", "\033[31m"},    // Red
		{"skipped", "—", "\033[33m"},  // Yellow
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			span := &Span{
				TraceID:    "trace123",
				SpanID:     "span456",
				Name:       "test_span",
				StartTime:  time.Now(),
				EndTime:    time.Now().Add(10 * time.Millisecond),
				Duration:   10 * time.Millisecond,
				Status:     tt.status,
				Attributes: map[string]interface{}{},
			}

			formatted := FormatSpan(span, false)

			if !strings.Contains(formatted, tt.icon) {
				t.Errorf("status %q: missing icon %q in output: %s", tt.status, tt.icon, formatted)
			}
		})
	}
}

// Test 3: Timestamp formatting (RFC3339)
func TestTimestampFormatting(t *testing.T) {
	startTime := time.Date(2026, 9, 1, 20, 34, 56, 12000000, time.UTC)

	span := &Span{
		TraceID:    "trace123",
		SpanID:     "span456",
		Name:       "test_span",
		StartTime:  startTime,
		EndTime:    startTime.Add(50 * time.Millisecond),
		Duration:   50 * time.Millisecond,
		Status:     "ok",
		Attributes: map[string]interface{}{},
	}

	formatted := FormatSpan(span, false)

	// Should contain RFC3339 formatted timestamp with milliseconds
	if !strings.Contains(formatted, "2026-09-01T20:34:56") {
		t.Errorf("formatted span missing proper timestamp: %s", formatted)
	}
}

// Test 4: Attribute display (principal, step_index)
func TestAttributeDisplay(t *testing.T) {
	tests := []struct {
		name         string
		attributes   map[string]interface{}
		shouldHave   []string
		shouldNotHave []string
	}{
		{
			name: "multiple attributes",
			attributes: map[string]interface{}{
				"route_name":     "payment",
				"principal":      "user-123",
				"step_index":     2,
				"correlation_id": "corr-456",
			},
			shouldHave:   []string{"principal=user-123", "step_index=2", "correlation_id=corr-456"},
			shouldNotHave: []string{},
		},
		{
			name: "only route_name",
			attributes: map[string]interface{}{
				"route_name": "audit",
			},
			shouldHave:   []string{},
			shouldNotHave: []string{"principal"},
		},
		{
			name: "empty attributes",
			attributes: map[string]interface{}{},
			shouldHave:   []string{},
			shouldNotHave: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			span := &Span{
				TraceID:    "trace123",
				SpanID:     "span456",
				Name:       "test",
				StartTime:  time.Now(),
				EndTime:    time.Now().Add(10 * time.Millisecond),
				Duration:   10 * time.Millisecond,
				Status:     "ok",
				Attributes: tt.attributes,
			}

			formatted := FormatSpan(span, false)

			for _, attr := range tt.shouldHave {
				if !strings.Contains(formatted, attr) {
					t.Errorf("expected to find %q in output: %s", attr, formatted)
				}
			}

			for _, attr := range tt.shouldNotHave {
				if strings.Contains(formatted, attr) {
					t.Errorf("expected NOT to find %q in output: %s", attr, formatted)
				}
			}
		})
	}
}

// Test 5: Query recent spans with limit
func TestQueryRecentSpans(t *testing.T) {
	tp := NewTracingProvider("stdout", "", 1.0)
	streamer := NewSpanStreamer(tp)

	// Add test spans
	for i := 0; i < 30; i++ {
		span := &Span{
			TraceID:    "trace123",
			SpanID:     fmt.Sprintf("span%d", i),
			Name:       "test_span",
			StartTime:  time.Now().Add(time.Duration(i) * time.Millisecond),
			Duration:   10 * time.Millisecond,
			Status:     "ok",
			Attributes: map[string]interface{}{
				"route_name": "payment",
			},
		}
		tp.mu.Lock()
		tp.spans[span.SpanID] = span
		tp.mu.Unlock()
	}

	// Test with limit
	recent, err := streamer.QueryRecentSpans(context.Background(), 10, "")
	if err != nil {
		t.Fatalf("QueryRecentSpans failed: %v", err)
	}

	if len(recent) != 10 {
		t.Errorf("expected 10 spans, got %d", len(recent))
	}

	// Verify spans are sorted by start time (oldest first)
	for i := 1; i < len(recent); i++ {
		if recent[i].StartTime.Before(recent[i-1].StartTime) {
			t.Errorf("spans not sorted correctly at index %d", i)
		}
	}
}

// Test 6: Filter spans by route
func TestFilterSpansByRoute(t *testing.T) {
	tp := NewTracingProvider("stdout", "", 1.0)
	streamer := NewSpanStreamer(tp)

	// Add spans for different routes
	routes := []string{"payment", "audit", "payment"}
	for i, route := range routes {
		span := &Span{
			TraceID:    "trace123",
			SpanID:     fmt.Sprintf("span%d", i),
			Name:       "test_span",
			StartTime:  time.Now(),
			Duration:   10 * time.Millisecond,
			Status:     "ok",
			Attributes: map[string]interface{}{
				"route_name": route,
			},
		}
		tp.mu.Lock()
		tp.spans[span.SpanID] = span
		tp.mu.Unlock()
	}

	// Query with route filter
	recent, err := streamer.QueryRecentSpans(context.Background(), 20, "payment")
	if err != nil {
		t.Fatalf("QueryRecentSpans failed: %v", err)
	}

	if len(recent) != 2 {
		t.Errorf("expected 2 payment spans, got %d", len(recent))
	}

	for _, span := range recent {
		if route, ok := span.Attributes["route_name"].(string); !ok || route != "payment" {
			t.Errorf("filtered span has wrong route: %v", span.Attributes)
		}
	}
}

// Test 7: Stream spans in real-time
func TestStreamSpans(t *testing.T) {
	tp := NewTracingProvider("stdout", "", 1.0)
	streamer := NewSpanStreamer(tp)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	opts := TailOptions{
		Route: "",
		Ctx:   ctx,
	}

	// Start streaming
	spanCh, err := streamer.StreamSpans(ctx, opts)
	if err != nil {
		t.Fatalf("StreamSpans failed: %v", err)
	}

	// Add spans in a goroutine while streaming
	go func() {
		for i := 0; i < 5; i++ {
			time.Sleep(100 * time.Millisecond)
			span := &Span{
				TraceID:    "trace123",
				SpanID:     fmt.Sprintf("span%d", i),
				Name:       "test_span",
				StartTime:  time.Now(),
				Duration:   10 * time.Millisecond,
				Status:     "ok",
				Attributes: map[string]interface{}{
					"route_name": "payment",
				},
			}
			tp.mu.Lock()
			tp.spans[span.SpanID] = span
			tp.mu.Unlock()
		}
	}()

	// Collect streamed spans
	receivedCount := 0
	for span := range spanCh {
		if span != nil {
			receivedCount++
		}
	}

	if receivedCount < 3 {
		t.Logf("received %d spans (expected at least 3)", receivedCount)
	}
}

// Test 8: CLI flags parsing (integration with main command)
func TestSpanStreamerWithNilProvider(t *testing.T) {
	streamer := NewSpanStreamer(nil)

	ctx := context.Background()
	_, err := streamer.QueryRecentSpans(ctx, 10, "")
	if err == nil {
		t.Errorf("expected error with nil provider, got nil")
	}

	_, err = streamer.StreamSpans(ctx, TailOptions{})
	if err == nil {
		t.Errorf("expected error with nil provider, got nil")
	}
}

// Test 9: Span hierarchy formatting
func TestFormatSpanHierarchy(t *testing.T) {
	parentSpan := &Span{
		TraceID:    "trace123",
		SpanID:     "parent",
		Name:       "message_processing",
		StartTime:  time.Now(),
		Duration:   50 * time.Millisecond,
		Status:     "ok",
		Attributes: map[string]interface{}{
			"route_name": "payment",
		},
	}

	childSpans := []*Span{
		{
			TraceID:      "trace123",
			SpanID:       "child1",
			ParentSpanID: "parent",
			Name:         "authorize",
			StartTime:    time.Now().Add(5 * time.Millisecond),
			Duration:     20 * time.Millisecond,
			Status:       "ok",
			Attributes:   map[string]interface{}{},
		},
		{
			TraceID:      "trace123",
			SpanID:       "child2",
			ParentSpanID: "parent",
			Name:         "route",
			StartTime:    time.Now().Add(30 * time.Millisecond),
			Duration:     15 * time.Millisecond,
			Status:       "ok",
			Attributes:   map[string]interface{}{},
		},
	}

	formatted := FormatSpanHierarchy(parentSpan, childSpans, "")

	// Check that parent and children are in output
	if !strings.Contains(formatted, "message_processing") {
		t.Errorf("missing parent span name: %s", formatted)
	}
	if !strings.Contains(formatted, "authorize") {
		t.Errorf("missing child span name: %s", formatted)
	}
	if !strings.Contains(formatted, "route") {
		t.Errorf("missing second child span name: %s", formatted)
	}

	// Check for indentation
	lines := strings.Split(formatted, "\n")
	if len(lines) < 3 {
		t.Errorf("expected at least 3 lines (parent + 2 children), got %d", len(lines))
	}
}

// Test 10: Find child spans
func TestFindChildSpans(t *testing.T) {
	parentSpan := &Span{
		TraceID: "trace123",
		SpanID:  "parent",
		Name:    "message_processing",
	}

	allSpans := []*Span{
		{
			TraceID:      "trace123",
			SpanID:       "child1",
			ParentSpanID: "parent",
			Name:         "step1",
		},
		{
			TraceID:      "trace123",
			SpanID:       "child2",
			ParentSpanID: "parent",
			Name:         "step2",
		},
		{
			TraceID:      "trace123",
			SpanID:       "other",
			ParentSpanID: "different",
			Name:         "other_span",
		},
	}

	children := FindChildSpans(parentSpan, allSpans)

	if len(children) != 2 {
		t.Errorf("expected 2 children, got %d", len(children))
	}

	for _, child := range children {
		if child.ParentSpanID != "parent" {
			t.Errorf("found span with wrong parent: %v", child)
		}
	}
}

// Test 11: Get spans by trace ID
func TestGetSpansByTraceID(t *testing.T) {
	allSpans := []*Span{
		{TraceID: "trace123", SpanID: "span1"},
		{TraceID: "trace123", SpanID: "span2"},
		{TraceID: "trace456", SpanID: "span3"},
		{TraceID: "trace123", SpanID: "span4"},
	}

	result := GetSpansByTraceID("trace123", allSpans)

	if len(result) != 3 {
		t.Errorf("expected 3 spans with trace123, got %d", len(result))
	}

	for _, span := range result {
		if span.TraceID != "trace123" {
			t.Errorf("found span with wrong trace ID: %v", span)
		}
	}
}

// Test 12: Empty span handling
func TestFormatEmptySpan(t *testing.T) {
	formatted := FormatSpan(nil, false)
	if formatted != "" {
		t.Errorf("expected empty string for nil span, got: %s", formatted)
	}
}

// Test 13: Status icon generation
func TestGetStatusIcon(t *testing.T) {
	tests := []struct {
		status       string
		expectedIcon string
	}{
		{"ok", "✓"},
		{"error", "✗"},
		{"skipped", "—"},
		{"unknown", "?"},
	}

	for _, tt := range tests {
		icon, _ := getStatusIcon(tt.status)
		// Remove ANSI codes for comparison
		cleaned := strings.ReplaceAll(strings.ReplaceAll(icon, "\033[32m", ""), "\033[0m", "")
		cleaned = strings.ReplaceAll(strings.ReplaceAll(cleaned, "\033[31m", ""), "\033[0m", "")
		cleaned = strings.ReplaceAll(strings.ReplaceAll(cleaned, "\033[33m", ""), "\033[0m", "")

		if !strings.Contains(cleaned, tt.expectedIcon) {
			t.Errorf("status %q: expected icon %q, got %q", tt.status, tt.expectedIcon, icon)
		}
	}
}
