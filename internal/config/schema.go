package config

// RouteConfig represents the complete route configuration loaded from YAML
type RouteConfig struct {
	Version        int                       `yaml:"version" json:"version"`
	Sources        map[string]SourceSpec     `yaml:"sources" json:"sources"`
	Sinks          map[string]SinkSpec       `yaml:"sinks" json:"sinks"`
	Routes         map[string]RouteSpec      `yaml:"routes" json:"routes"`
	Observability  *ObservabilityConfig      `yaml:"observability,omitempty" json:"observability,omitempty"`
}

// SourceSpec defines a named message input
type SourceSpec struct {
	Type  string                 `yaml:"type" json:"type"`
	Path  string                 `yaml:"path,omitempty" json:"path,omitempty"`
	URL   string                 `yaml:"url,omitempty" json:"url,omitempty"`
	Extra map[string]interface{} `yaml:",inline" json:"extra,omitempty"`
}

// SinkSpec defines a named message output
type SinkSpec struct {
	Type  string                 `yaml:"type" json:"type"`
	Path  string                 `yaml:"path,omitempty" json:"path,omitempty"`
	URL   string                 `yaml:"url,omitempty" json:"url,omitempty"`
	Extra map[string]interface{} `yaml:",inline" json:"extra,omitempty"`
}

// RouteSpec defines a pipeline (from source to sinks via steps)
type RouteSpec struct {
	From      string            `yaml:"from" json:"from"`
	Auth      interface{}       `yaml:"auth" json:"auth"` // can be string "none" or object with steps
	ErrorPath *ErrorPathSpec    `yaml:"error_path" json:"error_path"`
	Steps     []StepSpec        `yaml:"steps" json:"steps"`
	Retry     *RetryPolicy      `yaml:"retry,omitempty" json:"retry,omitempty"`
	Ordering  string            `yaml:"ordering,omitempty" json:"ordering,omitempty"` // "required" or "none" (default: "none")
	Contracts []ContractSpec    `yaml:"contracts,omitempty" json:"contracts,omitempty"` // Data contracts (M0.3.5)

	// Lineage tracking fields (M0.4)
	RetentionPolicy     string `yaml:"retention_policy,omitempty" json:"retention_policy,omitempty"`
	RetentionPolicyExpr string `yaml:"retention_policy_expr,omitempty" json:"retention_policy_expr,omitempty"`
	SubjectIDExpr       string `yaml:"subject_id_expr,omitempty" json:"subject_id_expr,omitempty"`

	// RouteVersion is the deterministic hash of this resolved route configuration.
	// Computed automatically during route loading after fragment resolution.
	// Used for lineage tracking to detect when route definitions change.
	// Format: SHA256 hash as lowercase hex string (64 characters).
	RouteVersion string `yaml:"-" json:"-"`
}

// ErrorPathSpec defines error handling (target sink and retry policy)
type ErrorPathSpec struct {
	Target string       `yaml:"target" json:"target"` // sink name
	Retry  *RetryPolicy `yaml:"retry,omitempty" json:"retry,omitempty"`
}

// RetryPolicy defines retry behavior
type RetryPolicy struct {
	MaxAttempts int `yaml:"max_attempts" json:"max_attempts"`
	BackoffMs   int `yaml:"backoff_ms,omitempty" json:"backoff_ms,omitempty"`
	JitterMs    int `yaml:"jitter_ms,omitempty" json:"jitter_ms,omitempty"`
}

// StepSpec represents a single pipeline step
type StepSpec struct {
	// One of these fields is populated based on step type
	Filter    *FilterSpec    `yaml:"filter,omitempty" json:"filter,omitempty"`
	Translate *TranslateSpec `yaml:"translate,omitempty" json:"translate,omitempty"`
	Route     *RouteStepSpec `yaml:"route,omitempty" json:"route,omitempty"`
	Wiretap   *WiretapSpec   `yaml:"wiretap,omitempty" json:"wiretap,omitempty"`
	Idempotent *IdempotentSpec `yaml:"idempotent,omitempty" json:"idempotent,omitempty"`
	Authorize *AuthorizeSpec `yaml:"authorize,omitempty" json:"authorize,omitempty"`
	Contract  *ContractStepSpec `yaml:"contract,omitempty" json:"contract,omitempty"` // Contract validation step (M0.3.5)

	// Extra fields for extensibility
	Extra map[string]interface{} `yaml:",inline" json:"-"`
}

// ContractStepSpec defines a contract validation step
type ContractStepSpec struct {
	ID     string `yaml:"id" json:"id"`                           // contract ID to validate against
	Strict bool   `yaml:"strict,omitempty" json:"strict,omitempty"` // fail pipeline on violation? (default: false)
}

// FilterSpec defines a filter step (boolean predicate)
type FilterSpec struct {
	Expr string `yaml:"expr" json:"expr"`
}

// TranslateSpec defines a translate step (JSONata transformation)
type TranslateSpec struct {
	Expr string `yaml:"expr" json:"expr"`
}

// RouteStepSpec defines a content-based router step
type RouteStepSpec struct {
	Expr     string                   `yaml:"expr" json:"expr"` // evaluates to route name
	Cases    map[string]RouteCase     `yaml:"cases,omitempty" json:"cases,omitempty"`
	Default  string                   `yaml:"default,omitempty" json:"default,omitempty"` // default route
}

// RouteCase represents a single case in a router
type RouteCase struct {
	Target string `yaml:"target" json:"target"`
}

// WiretapSpec defines a wiretap step (duplicate to secondary sink)
type WiretapSpec struct {
	Sink string `yaml:"sink" json:"sink"` // sink to duplicate to
}

// IdempotentSpec defines an idempotent deduplication step
type IdempotentSpec struct {
	KeyExpr string `yaml:"key_expr" json:"key_expr"` // JSONata to extract dedup key
}

// AuthorizeSpec defines an authorization step
type AuthorizeSpec struct {
	Mode         string   `yaml:"mode" json:"mode"`                   // "rbac" or "abac"
	RequireRoles []string `yaml:"require_roles,omitempty" json:"require_roles,omitempty"` // for RBAC mode
	Expr         string   `yaml:"expr,omitempty" json:"expr,omitempty"`                   // for ABAC mode
}

// ContractSpec defines a data contract with JSON Schema validation (M0.3.5-7)
type ContractSpec struct {
	ID          string                 `yaml:"id" json:"id"`                               // contract identifier
	Version     string                 `yaml:"version" json:"version"`                     // semver: 1.0.0
	Schema      interface{}            `yaml:"schema" json:"schema"`                       // JSON Schema (can be YAML object or JSON string)
	OnViolation string                 `yaml:"on_violation" json:"on_violation"`           // sink name or route for violations
	Strict      bool                   `yaml:"strict,omitempty" json:"strict,omitempty"` // fail pipeline on violation? (default: false)
	Extra       map[string]interface{} `yaml:",inline" json:"-"`                           // extensibility
}

// ObservabilityConfig defines observability settings for metrics and tracing
type ObservabilityConfig struct {
	Metrics *MetricsConfig `yaml:"metrics,omitempty" json:"metrics,omitempty"`
	Tracing *TracingConfig `yaml:"tracing,omitempty" json:"tracing,omitempty"`
}

// MetricsConfig defines Prometheus metrics settings
type MetricsConfig struct {
	Enabled         bool   `yaml:"enabled,omitempty" json:"enabled,omitempty"`
	PrometheusAddr  string `yaml:"prometheus_addr,omitempty" json:"prometheus_addr,omitempty"` // default: ":2112"
}

// TracingConfig defines OpenTelemetry tracing settings
type TracingConfig struct {
	Enabled       bool    `yaml:"enabled,omitempty" json:"enabled,omitempty"`
	OtelExporter  string  `yaml:"otel_exporter,omitempty" json:"otel_exporter,omitempty"` // stdout, jaeger, datadog
	JaegerEndpoint string `yaml:"jaeger_endpoint,omitempty" json:"jaeger_endpoint,omitempty"` // for Jaeger exporter
	SampleRate    float64 `yaml:"sample_rate,omitempty" json:"sample_rate,omitempty"`   // default: 0.1 (10%)
}
