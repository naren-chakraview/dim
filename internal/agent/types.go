package agent

import "time"

// AgentInterfaceVersion is the published interface version
const AgentInterfaceVersion = "1.0.0"

// ResponseEnvelope wraps all operation responses
type ResponseEnvelope struct {
	InterfaceVersion    string        `json:"interface_version"`
	Timestamp           time.Time     `json:"timestamp"`
	Result              interface{}   `json:"result,omitempty"`
	Error               *OperationErr `json:"error,omitempty"`
	DeprecationNotice   string        `json:"deprecation_notice,omitempty"`
}

// OperationErr represents an operation error
type OperationErr struct {
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Details interface{} `json:"details,omitempty"`
}

// --- Validate Operation ---

type ValidateRequest struct {
	RouteConfigPath string `json:"route_config_path"`
	StrictMode      bool   `json:"strict_mode,omitempty"`
}

type ValidateResponse struct {
	Valid       bool                 `json:"valid"`
	Errors      []ValidationError    `json:"errors"`
	Warnings    []string             `json:"warnings"`
	RouteVersion string              `json:"route_version,omitempty"`
}

type ValidationError struct {
	Code          string `json:"code"`
	Message       string `json:"message"`
	Path          string `json:"path"`
	SuggestedFix  string `json:"suggested_fix,omitempty"`
}

// --- Test Operation ---

type TestRequest struct {
	RouteConfigPath string `json:"route_config_path"`
	FixturesPath    string `json:"fixtures_path"`
	TimeoutMs       int    `json:"timeout_ms,omitempty"`
}

type TestResponse struct {
	Passed      bool         `json:"passed"`
	TestResults []TestResult `json:"test_results"`
	Summary     TestSummary  `json:"summary"`
}

type TestResult struct {
	Name       string      `json:"name"`
	Passed     bool        `json:"passed"`
	DurationMs int64       `json:"duration_ms"`
	Error      string      `json:"error,omitempty"`
	Expected   interface{} `json:"expected,omitempty"`
	Actual     interface{} `json:"actual,omitempty"`
}

type TestSummary struct {
	Total      int   `json:"total"`
	Passed     int   `json:"passed"`
	Failed     int   `json:"failed"`
	Skipped    int   `json:"skipped"`
	DurationMs int64 `json:"duration_ms"`
}

// --- Scaffold Operation ---

type ScaffoldRequest struct {
	Domain       string `json:"domain"`
	SourceType   string `json:"source_type,omitempty"`
	SinkType     string `json:"sink_type,omitempty"`
	WithContract bool   `json:"with_contract,omitempty"`
	Template     string `json:"template,omitempty"`
}

type ScaffoldResponse struct {
	Success       bool     `json:"success"`
	DomainPath    string   `json:"domain_path"`
	FilesCreated  []string `json:"files_created"`
	NextSteps     []string `json:"next_steps"`
}

// --- Lineage Operation ---

type LineageRequest struct {
	MessageID string `json:"message_id"`
}

type LineageResponse struct {
	Found           bool             `json:"found"`
	LineageRecords  []LineageRecord  `json:"lineage_records"`
}

type LineageRecord struct {
	ID        string    `json:"id"`
	MessageID string    `json:"message_id"`
	RouteName string    `json:"route_name"`
	Domain    string    `json:"domain"`
	CreatedAt time.Time `json:"created_at"`
	Step      string    `json:"step"`
	Status    string    `json:"status"` // "success" or "error"
}

// --- Provenance Operation ---

type ProvenanceRequest struct {
	SubjectID string `json:"subject_id"`
}

type ProvenanceResponse struct {
	Found             bool                `json:"found"`
	ProvenanceChain   []ProvenanceRecord  `json:"provenance_chain"`
}

type ProvenanceRecord struct {
	ID        string    `json:"id"`
	SubjectID string    `json:"subject_id"`
	MessageID string    `json:"message_id"`
	RouteName string    `json:"route_name"`
	CreatedAt time.Time `json:"created_at"`
	Status    string    `json:"status"`
}

// --- Capabilities Operation ---

type CapabilitiesRequest struct {
	FilterType string `json:"filter_type,omitempty"` // "adapter" | "step" | "eip"
}

type CapabilitiesResponse struct {
	Capabilities  []Capability `json:"capabilities"`
	SchemaVersion string       `json:"schema_version"`
}

type Capability struct {
	Name           string      `json:"name"`
	Type           string      `json:"type"` // "adapter" | "step" | "eip"
	ConfigSchema   interface{} `json:"config_schema"`
	Description    string      `json:"description"`
	Example        interface{} `json:"example,omitempty"`
}
