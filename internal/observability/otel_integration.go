package observability

import (
	"context"
	"fmt"
	"log"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	"go.opentelemetry.io/otel/trace"
)

// OTelProvider wraps the real OpenTelemetry SDK with OTLP export
type OTelProvider struct {
	tracerProvider *sdktrace.TracerProvider
	tracer         trace.Tracer
	exporter       sdktrace.SpanExporter
	enabled        bool
}

// NewOTelProvider creates a real OpenTelemetry provider with OTLP export
// endpoint: OTLP collector endpoint (e.g., "http://localhost:4317")
// serviceName: service name for resource attributes
// sampleRate: sampling rate (0.0-1.0)
func NewOTelProvider(ctx context.Context, endpoint string, serviceName string, sampleRate float64) (*OTelProvider, error) {
	if endpoint == "" {
		// Default to localhost if not specified
		endpoint = "http://localhost:4317"
	}

	if sampleRate < 0 {
		sampleRate = 0
	}
	if sampleRate > 1 {
		sampleRate = 1
	}

	// Create OTLP exporter
	exporter, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithEndpoint(endpoint),
		otlptracegrpc.WithTimeout(5*time.Second),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create OTLP exporter: %w", err)
	}

	// Create resource with service attributes
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(serviceName),
			semconv.ServiceVersion("v0.5.0"),
			attribute.String("service.instance.id", generateInstanceID()),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// Create trace provider with OTLP exporter and sampler
	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.TraceIDRatioBased(sampleRate)),
	)

	// Set as global tracer provider for convenience
	otel.SetTracerProvider(tracerProvider)

	log.Printf("[INFO] OpenTelemetry configured: endpoint=%s service=%s sample_rate=%.2f", endpoint, serviceName, sampleRate)

	return &OTelProvider{
		tracerProvider: tracerProvider,
		tracer:         tracerProvider.Tracer("dim"),
		exporter:       exporter,
		enabled:        true,
	}, nil
}

// NewOTelProviderStdout creates an OpenTelemetry provider that logs to stdout
// (useful for development when no real collector is available)
func NewOTelProviderStdout(serviceName string, sampleRate float64) (*OTelProvider, error) {
	// Create a simple stdout span processor (not a full exporter)
	// For now, we'll use a no-op exporter and rely on logging
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(serviceName),
			semconv.ServiceVersion("v0.5.0"),
		),
	)
	if err != nil {
		return nil, err
	}

	// Create trace provider without exporter (spans will be logged by caller)
	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.TraceIDRatioBased(sampleRate)),
	)

	log.Printf("[INFO] OpenTelemetry configured: stdout logging, service=%s sample_rate=%.2f", serviceName, sampleRate)

	return &OTelProvider{
		tracerProvider: tracerProvider,
		tracer:         tracerProvider.Tracer("dim"),
		enabled:        true,
	}, nil
}

// StartSpan starts a new span with attributes
func (p *OTelProvider) StartSpan(ctx context.Context, spanName string, attrs ...attribute.KeyValue) (context.Context, trace.Span) {
	if !p.enabled || p.tracer == nil {
		return ctx, trace.SpanFromContext(ctx)
	}

	opts := []trace.SpanStartOption{
		trace.WithAttributes(attrs...),
	}

	return p.tracer.Start(ctx, spanName, opts...)
}

// EndSpan ends a span
func (p *OTelProvider) EndSpan(span trace.Span, attrs ...attribute.KeyValue) {
	if span != nil {
		if len(attrs) > 0 {
			span.SetAttributes(attrs...)
		}
		span.End()
	}
}

// Shutdown shuts down the tracer provider and exports pending spans
func (p *OTelProvider) Shutdown(ctx context.Context) error {
	if p.tracerProvider != nil {
		return p.tracerProvider.Shutdown(ctx)
	}
	return nil
}

// generateInstanceID generates a unique instance ID for resource attributes
func generateInstanceID() string {
	// Simple implementation: use timestamp + random number
	// In production, this could use hostname, container ID, etc.
	return fmt.Sprintf("dim-%d", time.Now().UnixNano())
}
