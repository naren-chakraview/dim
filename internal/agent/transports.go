package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"sort"
	"strings"
	"time"
)

// The operations are served over three wire protocols. All of them call
// MCPServer.CallOperation, so every transport returns the same result for the same
// request (TestTransportsServeIdenticalResults checks this):
//   - MCP: JSON-RPC 2.0 at /mcp (initialize, ping, tools/list, tools/call)
//   - A2A: agent card at /.well-known/agent-card.json and JSON-RPC SendMessage at /a2a
//   - REST: POST /call (see cmd/dimctl/agent.go)

// MCPProtocolVersion is the MCP revision this server speaks.
const MCPProtocolVersion = "2025-06-18"

// A2AProtocolVersion is the A2A revision this server speaks.
const A2AProtocolVersion = "1.0"

// operationRequestTypes maps each operation to the request struct its handler decodes.
// Input schemas are generated from these structs, so a schema cannot drift from the
// request the operation actually accepts.
var operationRequestTypes = map[string]reflect.Type{
	"validate_route":       reflect.TypeOf(ValidateRequest{}),
	"test_route":           reflect.TypeOf(TestRequest{}),
	"scaffold_domain":      reflect.TypeOf(ScaffoldRequest{}),
	"get_lineage":          reflect.TypeOf(LineageRequest{}),
	"get_provenance":       reflect.TypeOf(ProvenanceRequest{}),
	"get_capabilities":     reflect.TypeOf(CapabilitiesRequest{}),
	"scaffold_from_intent": reflect.TypeOf(ScaffoldFromIntentRequest{}),
	"critique_route":       reflect.TypeOf(CritiqueRouteRequest{}),
	"query_impact":         reflect.TypeOf(QueryImpactRequest{}),
}

// InputSchema returns the JSON Schema of an operation's request, generated from the
// request struct's json tags. A field without omitempty is required.
func InputSchema(operation string) (map[string]interface{}, error) {
	t, ok := operationRequestTypes[operation]
	if !ok {
		return nil, fmt.Errorf("no request type for operation %q", operation)
	}
	return structSchema(t), nil
}

func structSchema(t reflect.Type) map[string]interface{} {
	props := map[string]interface{}{}
	required := []string{}
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if f.PkgPath != "" { // unexported
			continue
		}
		tag := f.Tag.Get("json")
		if tag == "-" {
			continue
		}
		name, opts, _ := strings.Cut(tag, ",")
		if name == "" {
			name = f.Name
		}
		props[name] = typeSchema(f.Type)
		if !strings.Contains(opts, "omitempty") {
			required = append(required, name)
		}
	}
	schema := map[string]interface{}{"type": "object", "properties": props, "additionalProperties": false}
	if len(required) > 0 {
		sort.Strings(required)
		schema["required"] = required
	}
	return schema
}

func typeSchema(t reflect.Type) map[string]interface{} {
	switch t.Kind() {
	case reflect.Ptr:
		return typeSchema(t.Elem())
	case reflect.String:
		return map[string]interface{}{"type": "string"}
	case reflect.Bool:
		return map[string]interface{}{"type": "boolean"}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return map[string]interface{}{"type": "integer"}
	case reflect.Float32, reflect.Float64:
		return map[string]interface{}{"type": "number"}
	case reflect.Slice, reflect.Array:
		return map[string]interface{}{"type": "array", "items": typeSchema(t.Elem())}
	case reflect.Map:
		return map[string]interface{}{"type": "object"}
	case reflect.Struct:
		return structSchema(t)
	}
	return map[string]interface{}{} // interface{} and anything else: any value
}

// --- MCP (JSON-RPC 2.0) ---

type rpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type rpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  interface{}     `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

const (
	rpcParseError     = -32700
	rpcInvalidRequest = -32600
	rpcMethodNotFound = -32601
	rpcInvalidParams  = -32602
)

// HandleMCP serves the MCP JSON-RPC endpoint (Streamable HTTP, JSON responses).
func (s *MCPServer) HandleMCP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "MCP requests are POSTed as JSON-RPC", http.StatusMethodNotAllowed)
		return
	}
	var req rpcRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeRPC(w, rpcResponse{JSONRPC: "2.0", ID: json.RawMessage("null"),
			Error: &rpcError{Code: rpcParseError, Message: err.Error()}})
		return
	}
	if req.JSONRPC != "2.0" || req.Method == "" {
		writeRPC(w, rpcResponse{JSONRPC: "2.0", ID: json.RawMessage("null"),
			Error: &rpcError{Code: rpcInvalidRequest, Message: "not a JSON-RPC 2.0 request"}})
		return
	}
	// A notification has no id and gets no response body.
	if len(req.ID) == 0 {
		w.WriteHeader(http.StatusAccepted)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()
	result, rpcErr := s.dispatchMCP(ctx, req)
	resp := rpcResponse{JSONRPC: "2.0", ID: req.ID}
	if rpcErr != nil {
		resp.Error = rpcErr
	} else {
		resp.Result = result
	}
	writeRPC(w, resp)
}

func (s *MCPServer) dispatchMCP(ctx context.Context, req rpcRequest) (interface{}, *rpcError) {
	switch req.Method {
	case "initialize":
		return map[string]interface{}{
			"protocolVersion": MCPProtocolVersion,
			"capabilities":    map[string]interface{}{"tools": map[string]interface{}{}},
			"serverInfo":      map[string]interface{}{"name": "dim", "version": AgentInterfaceVersion},
		}, nil
	case "ping":
		return map[string]interface{}{}, nil
	case "tools/list":
		return map[string]interface{}{"tools": s.mcpTools()}, nil
	case "tools/call":
		var p struct {
			Name      string          `json:"name"`
			Arguments json.RawMessage `json:"arguments"`
		}
		if err := json.Unmarshal(req.Params, &p); err != nil {
			return nil, &rpcError{Code: rpcInvalidParams, Message: err.Error()}
		}
		if _, ok := s.operations[p.Name]; !ok {
			return nil, &rpcError{Code: rpcInvalidParams, Message: fmt.Sprintf("unknown tool %q", p.Name)}
		}
		if len(p.Arguments) == 0 {
			p.Arguments = json.RawMessage("{}")
		}
		env := s.CallOperation(ctx, p.Name, p.Arguments)
		text, _ := json.Marshal(env)
		return map[string]interface{}{
			"content": []interface{}{map[string]interface{}{"type": "text", "text": string(text)}},
			"isError": env.Error != nil,
		}, nil
	}
	return nil, &rpcError{Code: rpcMethodNotFound, Message: fmt.Sprintf("method %q not supported", req.Method)}
}

// mcpTools lists the operations as MCP tools, sorted by name.
func (s *MCPServer) mcpTools() []map[string]interface{} {
	ops := s.ListOperations()
	sort.Slice(ops, func(i, j int) bool { return ops[i].Name < ops[j].Name })
	tools := make([]map[string]interface{}, 0, len(ops))
	for _, op := range ops {
		schema, err := InputSchema(op.Name)
		if err != nil {
			continue // an operation without a request type is not exposed as a tool
		}
		tools = append(tools, map[string]interface{}{
			"name":        op.Name,
			"description": op.Description,
			"inputSchema": schema,
		})
	}
	return tools
}

func writeRPC(w http.ResponseWriter, resp rpcResponse) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// checkRequestFields validates a request's top-level fields against the operation's
// schema: every field must be known, and every required field present. A request that
// does not match is rejected with INVALID_REQUEST, so it never runs on silent defaults.
// An empty request is treated as an empty object.
func checkRequestFields(operation string, raw json.RawMessage) *OperationErr {
	schema, err := InputSchema(operation)
	if err != nil {
		return nil // an operation without a schema is checked by its handler only
	}
	if len(raw) == 0 || string(raw) == "null" {
		raw = json.RawMessage("{}")
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return &OperationErr{Code: "INVALID_REQUEST", Message: fmt.Sprintf("request must be a JSON object: %v", err)}
	}
	props, _ := schema["properties"].(map[string]interface{})
	var unknown []string
	for name := range fields {
		if _, ok := props[name]; !ok {
			unknown = append(unknown, name)
		}
	}
	if len(unknown) > 0 {
		sort.Strings(unknown)
		return &OperationErr{Code: "INVALID_REQUEST",
			Message: fmt.Sprintf("unknown field(s) for %s: %v", operation, unknown)}
	}
	required, _ := schema["required"].([]string)
	var missing []string
	for _, name := range required {
		if _, ok := fields[name]; !ok {
			missing = append(missing, name)
		}
	}
	if len(missing) > 0 {
		return &OperationErr{Code: "INVALID_REQUEST",
			Message: fmt.Sprintf("missing required field(s) for %s: %v", operation, missing)}
	}
	return nil
}
