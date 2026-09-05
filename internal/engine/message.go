package engine

import (
	"time"
)

// Message is the core data structure flowing through the pipeline
// Immutable after creation; steps produce new messages to output channels
type Message struct {
	// Headers are mutable key-value pairs (e.g., HTTP headers, metadata)
	Headers map[string]interface{} `json:"headers,omitempty"`

	// Body is the main payload (JSON-like structure)
	Body interface{} `json:"body"`

	// Metadata is immutable operational data attached at ingest
	Metadata Metadata `json:"metadata"`
}

// Metadata carries provenance and context information for a message
type Metadata struct {
	// CorrelationID links a message through all stages (for tracing/lineage)
	CorrelationID string `json:"correlation_id"`

	// IngestedAt is when the message entered the system (UTC)
	IngestedAt time.Time `json:"ingested_at"`

	// Route name this message is flowing through
	Route string `json:"route"`

	// RouteVersion is the compiled DAG version processing this message
	RouteVersion string `json:"route_version"`

	// ContractVersion is the schema version checked (if applicable)
	ContractVersion string `json:"contract_version,omitempty"`

	// ContractViolation is set if contract validation failed in non-strict mode (M0.3.6)
	ContractViolation interface{} `json:"-"` // *steps.ViolationInfo but imported as interface{} to avoid circular dependency

	// Stage is the current step in the pipeline (for debugging/observability)
	Stage string `json:"stage,omitempty"`

	// Principal is the authenticated caller (if available)
	Principal *Principal `json:"principal,omitempty"`

	// ReplayCount tracks how many times this message has been replayed (M1.8)
	ReplayCount int `json:"replay_count,omitempty"`

	// ReplayHistory records each replay operation (M1.8.3)
	ReplayHistory []*ReplayEntry `json:"replay_history,omitempty"`

	// ErrorType records the type of error that caused DLQ (M1.8)
	ErrorType string `json:"error_type,omitempty"`

	// AggregatorFacet tracks aggregation metadata (M2.1.2)
	AggregatorFacet interface{} `json:"aggregator_facet,omitempty"`
}

// Principal represents an authenticated identity
type Principal struct {
	Subject string                 `json:"subject"`
	Roles   []string               `json:"roles,omitempty"`
	Claims  map[string]interface{} `json:"claims,omitempty"`
	Token   string                 `json:"token,omitempty"`
}

// NewMessage constructs a message with required fields
func NewMessage(body interface{}, route, routeVersion string) *Message {
	return &Message{
		Headers: make(map[string]interface{}),
		Body:    body,
		Metadata: Metadata{
			CorrelationID: generateCorrelationID(),
			IngestedAt:    time.Now().UTC(),
			Route:         route,
			RouteVersion:  routeVersion,
		},
	}
}

// Copy creates a shallow copy of the message (for mutations at each step)
// Mutations to Body/Headers create new data; Metadata is shared
func (m *Message) Copy() *Message {
	if m == nil {
		return nil
	}
	newHeaders := make(map[string]interface{})
	for k, v := range m.Headers {
		newHeaders[k] = v
	}
	return &Message{
		Headers:  newHeaders,
		Body:     m.Body, // Shallow copy; steps should replace this, not mutate
		Metadata: m.Metadata,
	}
}

func generateCorrelationID() string {
	// Phase 0: simple timestamp-based ID. Phase 1+ can use UUIDv4
	return time.Now().UTC().Format("20060102150405000000")
}

// ReplayEntry records a replay operation for audit trail (M1.8.3)
type ReplayEntry struct {
	Timestamp time.Time `json:"timestamp"`
	Attempt   int       `json:"attempt"`
	Initiator string    `json:"initiator"`
}
