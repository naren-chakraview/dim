package observability

import (
	"context"
	"fmt"
	"log"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// This file provides backward compatibility between the hand-rolled TracingProvider
// API and the real OpenTelemetry SDK. It allows gradual migration without breaking
// existing code.

// OTelAdapter bridges TracingProvider calls to real OpenTelemetry spans
type OTelAdapter struct {
	otelProvider *OTelProvider
	activeSpans  map[string]trace.Span // Track active spans by span ID
}

// NewOTelAdapter creates an adapter for the OTel provider
func NewOTelAdapter(otelProvider *OTelProvider) *OTelAdapter {
	return &OTelAdapter{
		otelProvider: otelProvider,
		activeSpans:  make(map[string]trace.Span),
	}
}

// AdaptStartMessageSpan adapts the legacy StartMessageSpan API to real OTel
// This allows existing code to work with the real provider
func (a *OTelAdapter) AdaptStartMessageSpan(ctx context.Context, routeName, routeVersion, correlationID, contractVersion string, principal *Principal) (context.Context, *Span) {
	// Prepare OTel attributes
	attrs := []attribute.KeyValue{
		attribute.String("correlation_id", correlationID),
		attribute.String("route.name", routeName),
	}

	if routeVersion != "" {
		attrs = append(attrs, attribute.String("route.version", routeVersion))
	}
	if contractVersion != "" {
		attrs = append(attrs, attribute.String("contract.version", contractVersion))
	}

	if principal != nil {
		attrs = append(attrs, attribute.String("principal.subject", principal.Subject))
		if len(principal.Roles) > 0 {
			// Convert roles to comma-separated string for attribute
			rolesStr := ""
			for i, role := range principal.Roles {
				if i > 0 {
					rolesStr += ","
				}
				rolesStr += role
			}
			attrs = append(attrs, attribute.String("principal.roles", rolesStr))
		}
	}

	// Start real OTel span
	spanCtx, otelSpan := a.otelProvider.StartSpan(ctx, "message_processing", attrs...)

	// Create legacy Span object for backward compatibility
	legacySpan := &Span{
		TraceID:    extractTraceID(spanCtx),
		SpanID:     extractSpanID(spanCtx),
		Name:       "message_processing",
		Attributes: make(map[string]interface{}),
		Events:     make([]SpanEvent, 0),
		Status:     "ok",
	}

	// Store OTel span reference for later end/error recording
	a.activeSpans[legacySpan.SpanID] = otelSpan

	return spanCtx, legacySpan
}

// AdaptRecordStepExecution adapts step execution recording to real OTel
func (a *OTelAdapter) AdaptRecordStepExecution(ctx context.Context, spanID string, stepIndex int, stepType, stepName string, duration int64, success bool, stepErr error) {
	otelSpan, exists := a.activeSpans[spanID]
	if !exists || otelSpan == nil {
		return
	}

	// Record step as event on the main span
	attrs := []attribute.KeyValue{
		attribute.Int("step.index", stepIndex),
		attribute.String("step.type", stepType),
		attribute.String("step.name", stepName),
		attribute.Int64("step.duration_ms", duration),
	}

	if stepErr != nil {
		attrs = append(attrs, attribute.String("step.error", stepErr.Error()))
		otelSpan.SetStatus(trace.Status{Code: trace.StatusCodeError})
		otelSpan.RecordError(stepErr)
	}

	otelSpan.AddEvent("step_executed", trace.WithAttributes(attrs...))
}

// AdaptEndSpan adapts span ending to real OTel
func (a *OTelAdapter) AdaptEndSpan(spanID string, success bool, endErr error) {
	otelSpan, exists := a.activeSpans[spanID]
	if !exists || otelSpan == nil {
		return
	}

	if !success || endErr != nil {
		if endErr != nil {
			otelSpan.SetStatus(trace.Status{Code: trace.StatusCodeError})
			otelSpan.RecordError(endErr)
		}
	}

	otelSpan.End()
	delete(a.activeSpans, spanID)
}

// extractTraceID extracts trace ID from context (returns hex representation)
func extractTraceID(ctx context.Context) string {
	span := trace.SpanFromContext(ctx)
	if span == nil {
		return ""
	}
	return span.SpanContext().TraceID().String()
}

// extractSpanID extracts span ID from context (returns hex representation)
func extractSpanID(ctx context.Context) string {
	span := trace.SpanFromContext(ctx)
	if span == nil {
		return ""
	}
	return span.SpanContext().SpanID().String()
}

// Phase 1 Migration Path:
// 1. Current code calls TracingProvider.StartMessageSpan (hand-rolled)
// 2. We add OTelAdapter as a bridge
// 3. Gradually migrate callers to use OTelAdapter + real OTel provider
// 4. Eventually replace TracingProvider completely with OTel SDK
//
// This keeps Tier 2 observability (Tempo/Jaeger/Grafana) working while
// maintaining backward compatibility with Phase 0 code.
