package observability

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"sync"
	"time"
)

// Span represents an OpenTelemetry span with attributes
type Span struct {
	TraceID      string
	SpanID       string
	ParentSpanID string
	Name         string
	StartTime    time.Time
	EndTime      time.Time
	Duration     time.Duration
	Attributes   map[string]interface{}
	Events       []SpanEvent
	Status       string // "ok", "error"
	Error        error
}

// SpanEvent represents an event within a span
type SpanEvent struct {
	Name       string
	Timestamp  time.Time
	Attributes map[string]interface{}
}

// TracingProvider manages OpenTelemetry tracing
type TracingProvider struct {
	enabled       bool
	exporter      string // stdout, jaeger, datadog
	jaegerEndpoint string
	sampleRate    float64
	spans         map[string]*Span
	mu            *sync.RWMutex
	traceIDCounter uint64
	spanIDCounter  uint64
}

// NewTracingProvider creates a new tracing provider
func NewTracingProvider(exporter string, jaegerEndpoint string, sampleRate float64) *TracingProvider {
	// Validate sample rate
	if sampleRate < 0 {
		sampleRate = 0
	}
	if sampleRate > 1 {
		sampleRate = 1
	}

	// Default exporter is stdout
	if exporter == "" {
		exporter = "stdout"
	}

	tp := &TracingProvider{
		enabled:        true,
		exporter:       exporter,
		jaegerEndpoint: jaegerEndpoint,
		sampleRate:     sampleRate,
		spans:          make(map[string]*Span),
		mu:             &sync.RWMutex{},
		traceIDCounter: uint64(time.Now().UnixNano()),
		spanIDCounter:  uint64(time.Now().UnixNano()),
	}

	log.Printf("[DEBUG] tracing provider initialized: exporter=%s, sample_rate=%.2f", exporter, sampleRate)

	return tp
}

// StartMessageSpan creates a new span for message processing
func (tp *TracingProvider) StartMessageSpan(ctx context.Context, route string, messageID string) (context.Context, *Span) {
	if !tp.enabled {
		return ctx, nil
	}

	// Check sampling rate
	if rand.Float64() > tp.sampleRate {
		return ctx, nil
	}

	span := &Span{
		TraceID:    tp.generateTraceID(),
		SpanID:     tp.generateSpanID(),
		Name:       fmt.Sprintf("message_processing.%s", route),
		StartTime:  time.Now(),
		Attributes: make(map[string]interface{}),
		Events:     make([]SpanEvent, 0),
		Status:     "ok",
	}

	span.Attributes["route"] = route
	span.Attributes["message_id"] = messageID

	tp.mu.Lock()
	tp.spans[span.SpanID] = span
	tp.mu.Unlock()

	// Log if stdout exporter
	if tp.exporter == "stdout" {
		log.Printf("[TRACE] start_span trace_id=%s span_id=%s name=%s", span.TraceID, span.SpanID, span.Name)
	}

	// Attach trace context to context
	ctx = context.WithValue(ctx, "otel_trace_id", span.TraceID)
	ctx = context.WithValue(ctx, "otel_span_id", span.SpanID)

	return ctx, span
}

// RecordStepExecution records a step execution as a nested span
func (tp *TracingProvider) RecordStepExecution(ctx context.Context, stepName string, durationMs int64) *Span {
	if !tp.enabled {
		return nil
	}

	// Extract parent span context from context
	parentSpanID, ok := ctx.Value("otel_span_id").(string)
	if !ok {
		return nil
	}

	span := &Span{
		TraceID:      ctx.Value("otel_trace_id").(string),
		SpanID:       tp.generateSpanID(),
		ParentSpanID: parentSpanID,
		Name:         fmt.Sprintf("step_execution.%s", stepName),
		StartTime:    time.Now(),
		Duration:     time.Duration(durationMs) * time.Millisecond,
		Attributes:   make(map[string]interface{}),
		Events:       make([]SpanEvent, 0),
		Status:       "ok",
	}

	span.Attributes["step"] = stepName
	span.Attributes["duration_ms"] = durationMs
	span.EndTime = span.StartTime.Add(span.Duration)

	tp.mu.Lock()
	tp.spans[span.SpanID] = span
	tp.mu.Unlock()

	// Log if stdout exporter
	if tp.exporter == "stdout" {
		log.Printf("[TRACE] step_execution parent_span=%s span_id=%s name=%s duration_ms=%d",
			parentSpanID, span.SpanID, span.Name, durationMs)
	}

	return span
}

// EndSpan marks the end of a span
func (tp *TracingProvider) EndSpan(span *Span) {
	if !tp.enabled || span == nil {
		return
	}

	span.EndTime = time.Now()
	if span.StartTime.Before(span.EndTime) {
		span.Duration = span.EndTime.Sub(span.StartTime)
	}

	// Log if stdout exporter
	if tp.exporter == "stdout" {
		log.Printf("[TRACE] end_span span_id=%s name=%s status=%s duration_ms=%d",
			span.SpanID, span.Name, span.Status, span.Duration.Milliseconds())
	}

	// Export based on configured exporter
	tp.exportSpan(span)
}

// AddSpanEvent adds an event to a span
func (tp *TracingProvider) AddSpanEvent(span *Span, eventName string, attributes map[string]interface{}) {
	if !tp.enabled || span == nil {
		return
	}

	event := SpanEvent{
		Name:       eventName,
		Timestamp:  time.Now(),
		Attributes: attributes,
	}

	span.Events = append(span.Events, event)

	if tp.exporter == "stdout" {
		log.Printf("[TRACE] span_event span_id=%s event=%s", span.SpanID, eventName)
	}
}

// RecordSpanError marks a span as having an error
func (tp *TracingProvider) RecordSpanError(span *Span, err error) {
	if !tp.enabled || span == nil {
		return
	}

	span.Status = "error"
	span.Error = err

	if tp.exporter == "stdout" {
		log.Printf("[TRACE] span_error span_id=%s error=%v", span.SpanID, err)
	}
}

// exportSpan exports a span based on the configured exporter
func (tp *TracingProvider) exportSpan(span *Span) {
	switch tp.exporter {
	case "stdout":
		tp.exportToStdout(span)
	case "jaeger":
		tp.exportToJaeger(span)
	case "datadog":
		tp.exportToDatadog(span)
	default:
		log.Printf("[WARN] unknown exporter: %s", tp.exporter)
	}
}

// exportToStdout logs span to stdout
func (tp *TracingProvider) exportToStdout(span *Span) {
	log.Printf("[TRACE] exported_span trace_id=%s span_id=%s parent_id=%s name=%s status=%s duration_ms=%d",
		span.TraceID, span.SpanID, span.ParentSpanID, span.Name, span.Status, span.Duration.Milliseconds())
}

// exportToJaeger exports span to Jaeger (stub implementation)
func (tp *TracingProvider) exportToJaeger(span *Span) {
	if tp.jaegerEndpoint == "" {
		tp.jaegerEndpoint = "http://localhost:14268/api/traces"
	}
	// In a real implementation, we would send JSON-formatted spans to Jaeger
	log.Printf("[TRACE] exporting to jaeger endpoint=%s trace_id=%s", tp.jaegerEndpoint, span.TraceID)
}

// exportToDatadog exports span to Datadog (stub implementation)
func (tp *TracingProvider) exportToDatadog(span *Span) {
	// In a real implementation, we would send protobuf-formatted spans to Datadog agent
	log.Printf("[TRACE] exporting to datadog trace_id=%s", span.TraceID)
}

// generateTraceID generates a unique trace ID
func (tp *TracingProvider) generateTraceID() string {
	tp.mu.Lock()
	defer tp.mu.Unlock()
	tp.traceIDCounter++
	return fmt.Sprintf("%016x%016x", time.Now().UnixNano(), tp.traceIDCounter)
}

// generateSpanID generates a unique span ID
func (tp *TracingProvider) generateSpanID() string {
	tp.mu.Lock()
	defer tp.mu.Unlock()
	tp.spanIDCounter++
	return fmt.Sprintf("%016x", tp.spanIDCounter)
}

// GetSpan retrieves a span by ID
func (tp *TracingProvider) GetSpan(spanID string) *Span {
	tp.mu.RLock()
	defer tp.mu.RUnlock()
	return tp.spans[spanID]
}

// GetAllSpans returns all recorded spans (for testing)
func (tp *TracingProvider) GetAllSpans() []*Span {
	tp.mu.RLock()
	defer tp.mu.RUnlock()

	spans := make([]*Span, 0, len(tp.spans))
	for _, span := range tp.spans {
		spans = append(spans, span)
	}

	return spans
}

// Disable disables tracing
func (tp *TracingProvider) Disable() {
	tp.mu.Lock()
	defer tp.mu.Unlock()
	tp.enabled = false
	log.Println("[DEBUG] tracing disabled")
}

// Reset clears all spans (for testing)
func (tp *TracingProvider) Reset() {
	tp.mu.Lock()
	defer tp.mu.Unlock()
	tp.spans = make(map[string]*Span)
}
