package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"
)

// MCPServer implements the agent-facing MCP interface
type MCPServer struct {
	operations map[string]MCPOperation
	version    string
}

// MCPOperation defines a callable operation
type MCPOperation struct {
	Name        string
	Description string
	Handler     func(context.Context, json.RawMessage) (interface{}, *OperationErr)
}

// NewMCPServer creates a new MCP server
func NewMCPServer() *MCPServer {
	server := &MCPServer{
		operations: make(map[string]MCPOperation),
		version:    AgentInterfaceVersion,
	}

	// Register all operations
	server.registerOperations()

	return server
}

// RegisterOperations registers all available operations
func (s *MCPServer) registerOperations() {
	s.operations["validate_route"] = MCPOperation{
		Name:        "validate_route",
		Description: "Validate a route YAML against JSON schema and governance rules",
		Handler: func(ctx context.Context, rawReq json.RawMessage) (interface{}, *OperationErr) {
			var req ValidateRequest
			if err := json.Unmarshal(rawReq, &req); err != nil {
				return nil, &OperationErr{Code: "INVALID_REQUEST", Message: fmt.Sprintf("Failed to parse request: %v", err)}
			}
			return ValidateRoute(ctx, req)
		},
	}

	s.operations["test_route"] = MCPOperation{
		Name:        "test_route",
		Description: "Run integration tests (fixtures) against a route configuration",
		Handler: func(ctx context.Context, rawReq json.RawMessage) (interface{}, *OperationErr) {
			var req TestRequest
			if err := json.Unmarshal(rawReq, &req); err != nil {
				return nil, &OperationErr{Code: "INVALID_REQUEST", Message: fmt.Sprintf("Failed to parse request: %v", err)}
			}
			return TestRoute(ctx, req)
		},
	}

	s.operations["scaffold_domain"] = MCPOperation{
		Name:        "scaffold_domain",
		Description: "Generate a domain directory with governance imports and starter route",
		Handler: func(ctx context.Context, rawReq json.RawMessage) (interface{}, *OperationErr) {
			var req ScaffoldRequest
			if err := json.Unmarshal(rawReq, &req); err != nil {
				return nil, &OperationErr{Code: "INVALID_REQUEST", Message: fmt.Sprintf("Failed to parse request: %v", err)}
			}
			return ScaffoldDomain(ctx, req)
		},
	}

	s.operations["get_lineage"] = MCPOperation{
		Name:        "get_lineage",
		Description: "Get the lineage chain for a message by ID",
		Handler: func(ctx context.Context, rawReq json.RawMessage) (interface{}, *OperationErr) {
			var req LineageRequest
			if err := json.Unmarshal(rawReq, &req); err != nil {
				return nil, &OperationErr{Code: "INVALID_REQUEST", Message: fmt.Sprintf("Failed to parse request: %v", err)}
			}
			return GetLineage(ctx, req)
		},
	}

	s.operations["get_provenance"] = MCPOperation{
		Name:        "get_provenance",
		Description: "Get the complete provenance chain for a subject ID",
		Handler: func(ctx context.Context, rawReq json.RawMessage) (interface{}, *OperationErr) {
			var req ProvenanceRequest
			if err := json.Unmarshal(rawReq, &req); err != nil {
				return nil, &OperationErr{Code: "INVALID_REQUEST", Message: fmt.Sprintf("Failed to parse request: %v", err)}
			}
			return GetProvenance(ctx, req)
		},
	}

	s.operations["get_capabilities"] = MCPOperation{
		Name:        "get_capabilities",
		Description: "Get enumerable catalog of all supported adapters, steps, and schemas",
		Handler: func(ctx context.Context, rawReq json.RawMessage) (interface{}, *OperationErr) {
			var req CapabilitiesRequest
			if err := json.Unmarshal(rawReq, &req); err != nil {
				return nil, &OperationErr{Code: "INVALID_REQUEST", Message: fmt.Sprintf("Failed to parse request: %v", err)}
			}
			return GetCapabilities(ctx, req)
		},
	}

	s.operations["scaffold_from_intent"] = MCPOperation{
		Name:        "scaffold_from_intent",
		Description: "Generate a route scaffold from natural-language design intent",
		Handler: func(ctx context.Context, rawReq json.RawMessage) (interface{}, *OperationErr) {
			var req ScaffoldFromIntentRequest
			if err := json.Unmarshal(rawReq, &req); err != nil {
				return nil, &OperationErr{Code: "INVALID_REQUEST", Message: fmt.Sprintf("Failed to parse request: %v", err)}
			}
			return ScaffoldFromIntent(ctx, req)
		},
	}

	s.operations["critique_route"] = MCPOperation{
		Name:        "critique_route",
		Description: "Analyze and critique an existing route for non-structural issues",
		Handler: func(ctx context.Context, rawReq json.RawMessage) (interface{}, *OperationErr) {
			var req CritiqueRouteRequest
			if err := json.Unmarshal(rawReq, &req); err != nil {
				return nil, &OperationErr{Code: "INVALID_REQUEST", Message: fmt.Sprintf("Failed to parse request: %v", err)}
			}
			return CritiqueRoute(ctx, req)
		},
	}

	s.operations["query_impact"] = MCPOperation{
		Name:        "query_impact",
		Description: "Analyze what routes and adapters would be affected by changing a contract, sink, source, or step type",
		Handler: func(ctx context.Context, rawReq json.RawMessage) (interface{}, *OperationErr) {
			var req QueryImpactRequest
			if err := json.Unmarshal(rawReq, &req); err != nil {
				return nil, &OperationErr{Code: "INVALID_REQUEST", Message: fmt.Sprintf("Failed to parse request: %v", err)}
			}
			return QueryRouteImpact(ctx, req)
		},
	}
}

// CallOperation executes an operation by name with the given request
func (s *MCPServer) CallOperation(ctx context.Context, operationName string, rawRequest json.RawMessage) *ResponseEnvelope {
	operation, exists := s.operations[operationName]
	if !exists {
		return &ResponseEnvelope{
			InterfaceVersion: s.version,
			Timestamp:        time.Now().UTC(),
			Error: &OperationErr{
				Code:    "NOT_IMPLEMENTED",
				Message: fmt.Sprintf("Operation '%s' not found", operationName),
			},
		}
	}

	// Call the operation handler
	result, opErr := operation.Handler(ctx, rawRequest)

	// Build response envelope
	envelope := &ResponseEnvelope{
		InterfaceVersion: s.version,
		Timestamp:        time.Now().UTC(),
	}

	if opErr != nil {
		envelope.Error = opErr
	} else {
		envelope.Result = result
	}

	return envelope
}

// ListOperations returns all available operations
func (s *MCPServer) ListOperations() []OperationSchema {
	var schemas []OperationSchema

	for name, op := range s.operations {
		schema := OperationSchema{
			Name:        name,
			DisplayName: toDisplayName(name),
			Description: op.Description,
		}
		schemas = append(schemas, schema)
	}

	return schemas
}

// OperationSchema describes an operation for discovery
type OperationSchema struct {
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	Description string `json:"description"`
}

// Helper: convert operation name to display name
func toDisplayName(name string) string {
	displayNames := map[string]string{
		"validate_route":        "Validate Route Configuration",
		"test_route":            "Test Route with Fixtures",
		"scaffold_domain":       "Scaffold New Domain",
		"get_lineage":           "Query Message Lineage",
		"get_provenance":        "Query Subject Provenance",
		"get_capabilities":      "Query Supported Capabilities",
		"scaffold_from_intent":  "Generate Route from Design Intent",
		"critique_route":        "Critique and Review Route",
		"query_impact":          "Query Impact of Changes",
	}

	if display, exists := displayNames[name]; exists {
		return display
	}

	return name
}

// TestConnection verifies the server is operational
func (s *MCPServer) TestConnection(ctx context.Context) *ResponseEnvelope {
	log.Printf("Agent Interface Server v%s ready", s.version)

	return &ResponseEnvelope{
		InterfaceVersion: s.version,
		Timestamp:        time.Now().UTC(),
		Result: map[string]interface{}{
			"status":       "ok",
			"version":      s.version,
			"operations":   len(s.operations),
		},
	}
}
