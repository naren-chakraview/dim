package config

// RouteConfig represents the complete route configuration loaded from YAML
type RouteConfig struct {
	Version int                    `yaml:"version" json:"version"`
	Sources map[string]SourceSpec  `yaml:"sources" json:"sources"`
	Sinks   map[string]SinkSpec    `yaml:"sinks" json:"sinks"`
	Routes  map[string]RouteSpec   `yaml:"routes" json:"routes"`
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

	// Extra fields for extensibility
	Extra map[string]interface{} `yaml:",inline" json:"-"`
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
	Mode  string      `yaml:"mode" json:"mode"` // "rbac" or "abac"
	Rules interface{} `yaml:"rules" json:"rules"`
}
