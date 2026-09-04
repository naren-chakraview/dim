# OpenTelemetry Integration (R7)

This document explains the OpenTelemetry integration for dim's Tier 2 observability.

## Status

**Phase 0:** Hand-rolled in-process span tracking (tracing.go)
**Phase 1 (R7):** Real OpenTelemetry SDK + OTLP export (in progress)

## What Changed

### Problem (Phase 0)
- `internal/observability/tracing.go` had hand-rolled `Span`/`TracingProvider`
- No actual OpenTelemetry SDK
- No OTLP export capability → Tier 2 composition (Tempo/Jaeger/Grafana) impossible
- Release notes claimed "OpenTelemetry distributed tracing" but this wasn't real

### Solution (Phase 1 — R7)
1. **OTelProvider** (`otel_integration.go`) — Real OpenTelemetry SDK with OTLP export
   - Uses `go.opentelemetry.io/otel` SDK
   - OTLP/gRPC exporter for collector integration
   - Sampling rate support
   - Resource attributes (service name, version, instance ID)
   - Two modes: real OTLP export or stdout logging for dev

2. **OTelAdapter** (`otel_adapter.go`) — Backward compatibility bridge
   - Adapts legacy TracingProvider API calls to real OTel spans
   - Gradual migration path: existing code can use adapter while new code uses OTel directly
   - Records spans as OTel events
   - Tracks active spans for end/error callbacks

## How to Use

### Starting Real OpenTelemetry (Phase 1+)

```go
import "github.com/naren-chakraview/dim/internal/observability"

ctx := context.Background()

// Option 1: Export to real OTLP collector
otelProvider, err := observability.NewOTelProvider(
    ctx,
    "http://localhost:4317",  // Otel Collector endpoint
    "dim",                      // Service name
    1.0,                        // Sample rate (100%)
)

// Option 2: Log to stdout (development)
otelProvider, err := observability.NewOTelProviderStdout("dim", 1.0)

defer otelProvider.Shutdown(ctx)

// Create adapter for backward compatibility
adapter := observability.NewOTelAdapter(otelProvider)

// Legacy code can now use:
spanCtx, legacySpan := adapter.AdaptStartMessageSpan(
    ctx, "order-processing", "v1.2.3", "msg-123", "v1.0", nil,
)
```

### New Code (Direct OTel Usage)

```go
import "go.opentelemetry.io/otel/attribute"

ctx, span := otelProvider.StartSpan(ctx, "translate_step",
    attribute.String("step.name", "validate"),
    attribute.Int("step.index", 1),
)
defer span.End()
```

## Dependencies to Add

The following packages are required for R7. Add them to `go.mod`:

```
go.opentelemetry.io/otel@latest
go.opentelemetry.io/otel/exporters/otlp/otlptrace@latest
go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc@latest
go.opentelemetry.io/otel/sdk@latest
go.opentelemetry.io/otel/semconv@latest
google.golang.org/grpc@latest  (for OTLP/gRPC)
```

### Adding Dependencies

```bash
go get -u go.opentelemetry.io/otel
go get -u go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc
go get -u go.opentelemetry.io/otel/sdk
go get -u go.opentelemetry.io/otel/semconv/v1.17.0
```

## Tier 2 Observability Flow

With R7 complete, the flow is:

```
dim engine (dimd)
  ├─ OTel SDK emits spans
  ├─ OTLP/gRPC exporter sends to collector
  │
  └─> OpenTelemetry Collector
      (http://localhost:4317 default)
        └─> Tempo / Jaeger (backend trace storage)
            └─> Grafana Node Graph panel (visualization)
```

## Phase 1 vs Phase 0

| Feature | Phase 0 | Phase 1 (R7) |
|---------|---------|------------|
| In-process span tracking | ✅ Hand-rolled | ✅ Real OTel SDK |
| OTLP export | ❌ None | ✅ OTLP/gRPC |
| Tempo/Jaeger integration | ❌ Impossible | ✅ Yes |
| Grafana Node Graph | ❌ No real traces | ✅ Yes (with Tempo) |
| Backward compatibility | N/A | ✅ OTelAdapter |

## Testing

To test the integration:

1. Start an OTel Collector (Docker):
```bash
docker run -p 4317:4317 \
  -v $(pwd)/otel-config.yaml:/etc/otel/config.yaml \
  otel/opentelemetry-collector:latest
```

2. Run dim with OTel configured:
```bash
# In code or config:
OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4317
```

3. Send messages through the pipeline

4. Spans should appear in the backend (Tempo, Jaeger, etc.)

## Future Work

- [ ] Integrate OTelAdapter into executor (currently traces still use hand-rolled API)
- [ ] Add OTel metrics (Prometheus integration already done, need real OTel metrics SDK)
- [ ] Deprecate hand-rolled TracingProvider (after all callers migrated)
- [ ] Add context propagation for distributed tracing across services
- [ ] Add baggage for cross-cutting concerns (request IDs, user context)

## References

- [OpenTelemetry Go SDK](https://pkg.go.dev/go.opentelemetry.io/otel)
- [OTLP Specification](https://opentelemetry.io/docs/specs/otlp/)
- [Tempo Documentation](https://grafana.com/docs/tempo/)
- [Jaeger Getting Started](https://www.jaegertracing.io/docs/latest/getting-started/)
