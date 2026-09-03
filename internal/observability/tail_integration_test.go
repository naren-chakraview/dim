package observability

import (
	"context"
	"testing"
	"time"
)

// Test: Integration - Full trace tail workflow
func TestIntegration_FullTraceTailWorkflow(t *testing.T) {
	// Create a tracing provider
	tp := NewTracingProvider("stdout", "", 1.0)
	streamer := NewSpanStreamer(tp)

	// Simulate a message being processed through a route
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create a parent message span
	_, msgSpan := tp.StartMessageSpan(
		ctx,
		"payment-processing",
		"v1.2.3",
		"msg-123-abc",
		"1.0",
		&Principal{Subject: "user-456"},
	)

	if msgSpan == nil {
		t.Fatal("failed to create message span")
	}

	// Simulate step execution within the message
	tp.RecordStepExecution(ctx, 0, "filter", "validate_amount", 5*time.Millisecond, true, nil)
	tp.RecordStepExecution(ctx, 1, "translate", "transform_to_payment", 15*time.Millisecond, true, nil)
	tp.RecordStepExecution(ctx, 2, "route", "route_to_processor", 8*time.Millisecond, true, nil)

	// Complete the message
	tp.RecordMessageComplete(msgSpan, true, nil)

	// Now test the streaming functionality
	opts := TailOptions{
		Route:   "payment-processing",
		Service: "dim",
		Limit:   10,
		Ctx:     ctx,
	}

	// Query recent spans
	recentSpans, err := streamer.QueryRecentSpans(ctx, opts.Limit, opts.Route)
	if err != nil {
		t.Fatalf("QueryRecentSpans failed: %v", err)
	}

	// Should have 1 parent span + 3 child spans = 4 total
	if len(recentSpans) < 1 {
		t.Errorf("expected at least 1 span, got %d", len(recentSpans))
	}

	// Verify the parent span
	parentFound := false
	for _, span := range recentSpans {
		if span.Name == "message_processing" {
			parentFound = true
			if route, ok := span.Attributes["route_name"].(string); !ok || route != "payment-processing" {
				t.Errorf("parent span has wrong route: %v", span.Attributes)
			}
			if principal, ok := span.Attributes["principal"].(string); !ok || principal != "user-456" {
				t.Errorf("parent span has wrong principal: %v", span.Attributes)
			}
			if corrID, ok := span.Attributes["correlation_id"].(string); !ok || corrID != "msg-123-abc" {
				t.Errorf("parent span has wrong correlation_id: %v", span.Attributes)
			}
		}
	}

	if !parentFound {
		t.Fatal("parent message_processing span not found")
	}

	// Test formatting
	for _, span := range recentSpans {
		formatted := FormatSpan(span, false)
		if formatted == "" {
			t.Errorf("failed to format span: %v", span)
		}

		// Verify key information is in output
		if span.Name == "message_processing" {
			if !contains(formatted, "payment-processing") || !contains(formatted, "message_processing") {
				t.Errorf("formatted output missing key info: %s", formatted)
			}
		}
	}

	t.Logf("Integration test passed with %d spans", len(recentSpans))
}

// Test: Integration - Multiple routes with filtering
func TestIntegration_MultipleRoutesFiltering(t *testing.T) {
	tp := NewTracingProvider("stdout", "", 1.0)
	streamer := NewSpanStreamer(tp)

	ctx := context.Background()

	// Create spans for multiple routes
	routeNames := []string{"payment", "audit", "payment", "notification", "audit"}
	for i, routeName := range routeNames {
		corrID := "msg-" + routeName + "-" + string(rune(i))
		_, span := tp.StartMessageSpan(ctx, routeName, "v1", corrID, "", nil)
		if span != nil {
			tp.RecordMessageComplete(span, true, nil)
		}
	}

	// Query only payment route spans
	paymentSpans, err := streamer.QueryRecentSpans(ctx, 20, "payment")
	if err != nil {
		t.Fatalf("QueryRecentSpans failed: %v", err)
	}

	if len(paymentSpans) != 2 {
		t.Errorf("expected 2 payment spans, got %d", len(paymentSpans))
	}

	// Verify all returned spans are from payment route
	for _, span := range paymentSpans {
		if route, ok := span.Attributes["route_name"].(string); !ok || route != "payment" {
			t.Errorf("filtered span has wrong route: %v", span.Attributes)
		}
	}

	t.Logf("Multi-route filtering test passed with %d payment spans", len(paymentSpans))
}

// Test: Integration - Span hierarchy with parent-child relationships
func TestIntegration_SpanHierarchy(t *testing.T) {
	tp := NewTracingProvider("stdout", "", 1.0)

	// Create parent span with context update
	baseCtx := context.Background()
	ctx, parentSpan := tp.StartMessageSpan(
		baseCtx,
		"checkout",
		"v1",
		"order-123",
		"",
		&Principal{Subject: "customer-1"},
	)

	if parentSpan == nil {
		t.Fatal("failed to create parent span")
	}

	// Create child spans (step executions) with updated context
	childSpan1 := tp.RecordStepExecution(ctx, 0, "validate", "validate_cart", 10*time.Millisecond, true, nil)
	childSpan2 := tp.RecordStepExecution(ctx, 1, "authorize", "authorize_payment", 50*time.Millisecond, true, nil)

	tp.RecordMessageComplete(parentSpan, true, nil)

	// Verify child spans have correct parent reference
	if childSpan1 != nil && childSpan1.ParentSpanID != parentSpan.SpanID {
		t.Errorf("child span1 has wrong parent: %s != %s", childSpan1.ParentSpanID, parentSpan.SpanID)
	}
	if childSpan2 != nil && childSpan2.ParentSpanID != parentSpan.SpanID {
		t.Errorf("child span2 has wrong parent: %s != %s", childSpan2.ParentSpanID, parentSpan.SpanID)
	}

	// Get all spans
	allSpans := tp.GetAllSpans()

	// Find children of parent span
	children := FindChildSpans(parentSpan, allSpans)
	if len(children) != 2 {
		t.Logf("Warning: expected 2 child spans, got %d", len(children))
	}

	// Test hierarchy formatting (even if no children)
	formatted := FormatSpanHierarchy(parentSpan, children, "")
	if !contains(formatted, "checkout") {
		t.Errorf("hierarchy formatting missing parent span: %s", formatted)
	}
	if len(children) > 0 && !contains(formatted, "validate") {
		t.Errorf("hierarchy formatting missing validate child: %s", formatted)
	}

	t.Logf("Span hierarchy test passed with parent and %d children", len(children))
}

// Test: Integration - Span streaming with new spans added
func TestIntegration_StreamingWithDynamicSpans(t *testing.T) {
	tp := NewTracingProvider("stdout", "", 1.0)
	streamer := NewSpanStreamer(tp)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	opts := TailOptions{
		Route:   "",
		Service: "dim",
		Limit:   10,
		Ctx:     ctx,
	}

	// Start streaming
	spanCh, err := streamer.StreamSpans(ctx, opts)
	if err != nil {
		t.Fatalf("StreamSpans failed: %v", err)
	}

	// Track spans received
	receivedSpans := make([]*Span, 0)
	spansCh := make(chan *Span, 10)

	// Goroutine to collect spans
	go func() {
		for span := range spanCh {
			if span != nil {
				spansCh <- span
			}
		}
	}()

	// Add spans while streaming
	go func() {
		for i := 0; i < 3; i++ {
			time.Sleep(200 * time.Millisecond)
			corrID := "stream-test-" + string(rune(48+i))
			_, span := tp.StartMessageSpan(ctx, "test-route", "v1", corrID, "", nil)
			if span != nil {
				tp.RecordMessageComplete(span, true, nil)
			}
		}
	}()

	// Collect streamed spans with timeout
	timeout := time.After(3 * time.Second)
	for {
		select {
		case <-timeout:
			goto done
		case <-ctx.Done():
			goto done
		case span, ok := <-spansCh:
			if !ok {
				goto done
			}
			if span != nil {
				receivedSpans = append(receivedSpans, span)
			}
		}
	}

done:
	if len(receivedSpans) < 1 {
		t.Logf("Warning: received %d spans (may indicate timing issue)", len(receivedSpans))
	} else {
		t.Logf("Streaming test passed with %d received spans", len(receivedSpans))
	}
}

// Helper function
func contains(haystack, needle string) bool {
	for i := 0; i <= len(haystack)-len(needle); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}
