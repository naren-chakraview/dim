package observability

import (
	"fmt"

	"github.com/naren-chakraview/dim/internal/config"
)

// Config holds parsed observability configuration
type Config struct {
	// Metrics settings
	MetricsEnabled   bool
	PrometheusAddr   string

	// Tracing settings
	TracingEnabled   bool
	OtelExporter     string
	JaegerEndpoint   string
	SampleRate       float64
}

// DefaultConfig returns the default observability configuration (disabled)
func DefaultConfig() *Config {
	return &Config{
		MetricsEnabled:   false,
		PrometheusAddr:   ":2112",
		TracingEnabled:   false,
		OtelExporter:     "stdout",
		JaegerEndpoint:   "http://localhost:14268/api/traces",
		SampleRate:       0.1, // 10% sampling by default
	}
}

// FromRouteConfig parses observability config from RouteConfig
func FromRouteConfig(cfg *config.RouteConfig) *Config {
	obsConfig := DefaultConfig()

	if cfg.Observability == nil {
		return obsConfig
	}

	// Parse metrics config
	if cfg.Observability.Metrics != nil {
		obsConfig.MetricsEnabled = cfg.Observability.Metrics.Enabled
		if cfg.Observability.Metrics.PrometheusAddr != "" {
			obsConfig.PrometheusAddr = cfg.Observability.Metrics.PrometheusAddr
		}
	}

	// Parse tracing config
	if cfg.Observability.Tracing != nil {
		obsConfig.TracingEnabled = cfg.Observability.Tracing.Enabled
		if cfg.Observability.Tracing.OtelExporter != "" {
			obsConfig.OtelExporter = cfg.Observability.Tracing.OtelExporter
		}
		if cfg.Observability.Tracing.JaegerEndpoint != "" {
			obsConfig.JaegerEndpoint = cfg.Observability.Tracing.JaegerEndpoint
		}
		if cfg.Observability.Tracing.SampleRate > 0 {
			obsConfig.SampleRate = cfg.Observability.Tracing.SampleRate
		}
	}

	return obsConfig
}

// Validate checks the observability configuration for correctness
func (c *Config) Validate() error {
	// Validate Prometheus address if metrics enabled
	if c.MetricsEnabled && c.PrometheusAddr == "" {
		return fmt.Errorf("prometheus_addr is required when metrics are enabled")
	}

	// Validate exporter if tracing enabled
	if c.TracingEnabled {
		validExporters := map[string]bool{
			"stdout": true,
			"jaeger": true,
			"datadog": true,
		}
		if !validExporters[c.OtelExporter] {
			return fmt.Errorf("invalid otel_exporter: %s (must be stdout, jaeger, or datadog)", c.OtelExporter)
		}

		// Validate Jaeger endpoint if using Jaeger exporter
		if c.OtelExporter == "jaeger" && c.JaegerEndpoint == "" {
			return fmt.Errorf("jaeger_endpoint is required when using jaeger exporter")
		}

		// Validate sample rate
		if c.SampleRate < 0 || c.SampleRate > 1 {
			return fmt.Errorf("sample_rate must be between 0.0 and 1.0 (got %.2f)", c.SampleRate)
		}
	}

	return nil
}

// String returns a human-readable representation of the config
func (c *Config) String() string {
	return fmt.Sprintf(
		"ObservabilityConfig{metrics_enabled=%v, prometheus_addr=%q, tracing_enabled=%v, otel_exporter=%q, sample_rate=%.2f}",
		c.MetricsEnabled,
		c.PrometheusAddr,
		c.TracingEnabled,
		c.OtelExporter,
		c.SampleRate,
	)
}
