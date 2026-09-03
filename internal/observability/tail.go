package observability

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"
)

// TailOptions configures span tailing behavior
type TailOptions struct {
	Route      string        // Filter to specific route (optional)
	Service    string        // Service name (default: dim)
	Limit      int           // Number of recent spans to show before streaming (default: 20)
	Endpoint   string        // OTLP exporter endpoint (from env or config)
	Ctx        context.Context
}

// SpanStreamer handles streaming spans from a tracing provider
type SpanStreamer struct {
	provider *TracingProvider
	mu       sync.RWMutex
}

// NewSpanStreamer creates a new span streamer
func NewSpanStreamer(provider *TracingProvider) *SpanStreamer {
	return &SpanStreamer{
		provider: provider,
		mu:       sync.RWMutex{},
	}
}

// QueryRecentSpans retrieves N most recent spans, optionally filtered by route
func (ss *SpanStreamer) QueryRecentSpans(ctx context.Context, limit int, route string) ([]*Span, error) {
	if ss.provider == nil {
		return nil, fmt.Errorf("tracing provider not initialized")
	}

	if limit <= 0 {
		limit = 20
	}

	// Get all spans from provider
	allSpans := ss.provider.GetAllSpans()

	// Filter by route if specified
	var filtered []*Span
	for _, span := range allSpans {
		if route != "" {
			if routeName, ok := span.Attributes["route_name"].(string); ok && routeName == route {
				filtered = append(filtered, span)
			}
		} else {
			filtered = append(filtered, span)
		}
	}

	// Sort by start time (most recent first)
	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].StartTime.After(filtered[j].StartTime)
	})

	// Limit to requested count
	if len(filtered) > limit {
		filtered = filtered[:limit]
	}

	// Reverse to show oldest first
	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].StartTime.Before(filtered[j].StartTime)
	})

	return filtered, nil
}

// StreamSpans returns a channel that streams spans from the provider
// For MVP, this simulates streaming by checking for new spans periodically
func (ss *SpanStreamer) StreamSpans(ctx context.Context, opts TailOptions) (<-chan *Span, error) {
	if ss.provider == nil {
		return nil, fmt.Errorf("tracing provider not initialized")
	}

	spanCh := make(chan *Span, 10)

	go func() {
		defer close(spanCh)

		// Track which spans we've already sent
		seenSpanIDs := make(map[string]bool)
		ticker := time.NewTicker(500 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				// Get all spans and send new ones
				allSpans := ss.provider.GetAllSpans()
				for _, span := range allSpans {
					if seenSpanIDs[span.SpanID] {
						continue
					}

					// Filter by route if specified
					if opts.Route != "" {
						if routeName, ok := span.Attributes["route_name"].(string); ok && routeName != opts.Route {
							continue
						}
					}

					seenSpanIDs[span.SpanID] = true
					select {
					case spanCh <- span:
					case <-ctx.Done():
						return
					}
				}
			}
		}
	}()

	return spanCh, nil
}

// FormatSpan formats a span for terminal output with color codes
func FormatSpan(span *Span, isLive bool) string {
	if span == nil {
		return ""
	}

	// Status icon and color
	statusIcon, colorCode := getStatusIcon(span.Status)

	// Extract key attributes
	routeName := ""
	if rn, ok := span.Attributes["route_name"].(string); ok {
		routeName = rn
	}

	spanName := span.Name
	duration := span.Duration.Milliseconds()

	// Format timestamp with milliseconds (RFC3339)
	timestamp := span.StartTime.Format("2006-01-02T15:04:05.000Z07:00")

	// Build attribute string
	attrs := formatSpanAttributes(span)

	// Color codes for terminal output (ANSI)
	reset := "\033[0m"

	// Format main span line
	if attrs != "" {
		return fmt.Sprintf("%s[%s]%s %s | %s | %dms | %s (%s)",
			colorCode, timestamp, reset, routeName, spanName, duration, statusIcon, attrs)
	}

	return fmt.Sprintf("%s[%s]%s %s | %s | %dms | %s",
		colorCode, timestamp, reset, routeName, spanName, duration, statusIcon)
}

// FormatSpanHierarchy formats a span and its child spans
func FormatSpanHierarchy(span *Span, children []*Span, indent string) string {
	if span == nil {
		return ""
	}

	lines := []string{FormatSpan(span, false)}

	// Sort children by start time
	sort.Slice(children, func(i, j int) bool {
		return children[i].StartTime.Before(children[j].StartTime)
	})

	// Add child spans with indentation
	for _, child := range children {
		statusIcon, colorCode := getStatusIcon(child.Status)
		reset := "\033[0m"

		timestamp := child.StartTime.Format("2006-01-02T15:04:05.000Z07:00")
		duration := child.Duration.Milliseconds()
		attrs := formatSpanAttributes(child)

		line := fmt.Sprintf("%s  %s[%s]%s %s | %dms | %s",
			indent, colorCode, timestamp, reset, child.Name, duration, statusIcon)
		if attrs != "" {
			line += fmt.Sprintf(" (%s)", attrs)
		}
		lines = append(lines, line)
	}

	return formatLines(lines)
}

// getStatusIcon returns status icon and ANSI color code
func getStatusIcon(status string) (string, string) {
	// ANSI color codes
	green := "\033[32m"   // OK
	red := "\033[31m"     // ERROR
	yellow := "\033[33m"  // SKIPPED
	reset := "\033[0m"

	switch status {
	case "ok":
		return green + "✓" + reset, green
	case "error":
		return red + "✗" + reset, red
	case "skipped":
		return yellow + "—" + reset, yellow
	default:
		return "?", ""
	}
}

// formatSpanAttributes extracts and formats key attributes
func formatSpanAttributes(span *Span) string {
	if span == nil || len(span.Attributes) == 0 {
		return ""
	}

	attrs := []string{}

	// Include key attributes in order of importance
	keyAttrs := []string{"principal", "step_index", "correlation_id", "step_name", "route_version"}

	for _, key := range keyAttrs {
		if val, ok := span.Attributes[key]; ok {
			switch v := val.(type) {
			case string:
				attrs = append(attrs, fmt.Sprintf("%s=%s", key, v))
			case int, int64:
				attrs = append(attrs, fmt.Sprintf("%s=%v", key, v))
			case float64:
				attrs = append(attrs, fmt.Sprintf("%s=%.2f", key, v))
			}
		}
	}

	if len(attrs) == 0 {
		return ""
	}

	return joinAttributes(attrs)
}

// joinAttributes joins attributes into a comma-separated string
func joinAttributes(attrs []string) string {
	if len(attrs) == 0 {
		return ""
	}
	result := ""
	for i, attr := range attrs {
		if i > 0 {
			result += ", "
		}
		result += attr
	}
	return result
}

// formatLines joins lines with newlines
func formatLines(lines []string) string {
	result := ""
	for i, line := range lines {
		if i > 0 {
			result += "\n"
		}
		result += line
	}
	return result
}

// FilterSpansByRoute filters spans to only those matching the route name
func FilterSpansByRoute(spans []*Span, route string) []*Span {
	if route == "" {
		return spans
	}

	filtered := make([]*Span, 0, len(spans))
	for _, span := range spans {
		if routeName, ok := span.Attributes["route_name"].(string); ok && routeName == route {
			filtered = append(filtered, span)
		}
	}
	return filtered
}

// FindChildSpans finds all spans that are children of a given parent span
func FindChildSpans(parentSpan *Span, allSpans []*Span) []*Span {
	if parentSpan == nil {
		return nil
	}

	children := make([]*Span, 0)
	for _, span := range allSpans {
		if span.ParentSpanID == parentSpan.SpanID {
			children = append(children, span)
		}
	}
	return children
}

// GetSpansByTraceID retrieves all spans with a given trace ID
func GetSpansByTraceID(traceID string, allSpans []*Span) []*Span {
	if traceID == "" {
		return nil
	}

	spans := make([]*Span, 0)
	for _, span := range allSpans {
		if span.TraceID == traceID {
			spans = append(spans, span)
		}
	}
	return spans
}
