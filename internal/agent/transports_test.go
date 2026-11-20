package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func postJSON(t *testing.T, url string, body interface{}) map[string]interface{} {
	t.Helper()
	b, _ := json.Marshal(body)
	resp, err := http.Post(url, "application/json", bytes.NewReader(b))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	var out map[string]interface{}
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatalf("bad JSON (HTTP %d): %s", resp.StatusCode, data)
	}
	return out
}

// Every operation the server exposes must have a request type, or a tool or skill
// would be advertised without an input schema.
func TestEveryOperationHasAnInputSchema(t *testing.T) {
	s := NewMCPServer()
	for _, op := range s.ListOperations() {
		if _, err := InputSchema(op.Name); err != nil {
			t.Errorf("operation %s: %v", op.Name, err)
		}
	}
}

// Required fields come from fields without omitempty.
func TestInputSchemaRequiredFields(t *testing.T) {
	schema, err := InputSchema("validate_route")
	if err != nil {
		t.Fatal(err)
	}
	required, _ := schema["required"].([]string)
	if !reflect.DeepEqual(required, []string{"route_config_path"}) {
		t.Fatalf("validate_route required = %v, want [route_config_path]", required)
	}
	props := schema["properties"].(map[string]interface{})
	if _, ok := props["route_config_path"]; !ok {
		t.Fatal("route_config_path missing from properties")
	}
}

// Each transport returns the same result for the same operation. The result is compared,
// not the envelope, since the envelope carries a timestamp.
func TestTransportsServeIdenticalResults(t *testing.T) {
	srv := httptest.NewServer(NewMCPServer().Handler())
	defer srv.Close()
	args := map[string]interface{}{"filter_type": "adapter"}

	// REST
	rest := postJSON(t, srv.URL+"/call", map[string]interface{}{"operation": "get_capabilities", "request": args})
	restResult := rest["result"]

	// MCP tools/call
	mcp := postJSON(t, srv.URL+"/mcp", map[string]interface{}{
		"jsonrpc": "2.0", "id": 1, "method": "tools/call",
		"params": map[string]interface{}{"name": "get_capabilities", "arguments": args},
	})
	mcpResult := mcp["result"].(map[string]interface{})
	if mcpResult["isError"] != false {
		t.Fatalf("MCP tools/call reported an error: %v", mcpResult)
	}
	text := mcpResult["content"].([]interface{})[0].(map[string]interface{})["text"].(string)
	var mcpEnv map[string]interface{}
	json.Unmarshal([]byte(text), &mcpEnv)

	// A2A SendMessage
	a2a := postJSON(t, srv.URL+"/a2a", map[string]interface{}{
		"jsonrpc": "2.0", "id": 2, "method": "SendMessage",
		"params": map[string]interface{}{"message": map[string]interface{}{
			"messageId": "m1", "role": "ROLE_USER",
			"parts": []interface{}{map[string]interface{}{"data": map[string]interface{}{
				"operation": "get_capabilities", "request": args,
			}}},
		}},
	})
	task := a2a["result"].(map[string]interface{})["task"].(map[string]interface{})
	if state := task["status"].(map[string]interface{})["state"]; state != "TASK_STATE_COMPLETED" {
		t.Fatalf("A2A task state = %v, want COMPLETED", state)
	}
	a2aResult := task["artifacts"].([]interface{})[0].(map[string]interface{})["parts"].([]interface{})[0].(map[string]interface{})["data"].(map[string]interface{})["result"]

	if !reflect.DeepEqual(restResult, mcpEnv["result"]) {
		t.Errorf("MCP result differs from REST:\nREST: %v\nMCP:  %v", restResult, mcpEnv["result"])
	}
	if !reflect.DeepEqual(restResult, a2aResult) {
		t.Errorf("A2A result differs from REST:\nREST: %v\nA2A:  %v", restResult, a2aResult)
	}
}

func TestMCPProtocolBasics(t *testing.T) {
	srv := httptest.NewServer(NewMCPServer().Handler())
	defer srv.Close()

	init := postJSON(t, srv.URL+"/mcp", map[string]interface{}{"jsonrpc": "2.0", "id": 1, "method": "initialize",
		"params": map[string]interface{}{"protocolVersion": MCPProtocolVersion}})
	if got := init["result"].(map[string]interface{})["protocolVersion"]; got != MCPProtocolVersion {
		t.Fatalf("protocolVersion = %v", got)
	}

	list := postJSON(t, srv.URL+"/mcp", map[string]interface{}{"jsonrpc": "2.0", "id": 2, "method": "tools/list"})
	tools := list["result"].(map[string]interface{})["tools"].([]interface{})
	if len(tools) != len(operationRequestTypes) {
		t.Fatalf("tools/list returned %d tools, want %d", len(tools), len(operationRequestTypes))
	}
	for _, tool := range tools {
		tm := tool.(map[string]interface{})
		if _, ok := tm["inputSchema"]; !ok {
			t.Errorf("tool %v has no inputSchema", tm["name"])
		}
	}

	unknown := postJSON(t, srv.URL+"/mcp", map[string]interface{}{"jsonrpc": "2.0", "id": 3, "method": "tools/call",
		"params": map[string]interface{}{"name": "no_such_tool"}})
	if unknown["error"] == nil {
		t.Fatal("unknown tool did not return a JSON-RPC error")
	}

	// A notification (no id) gets no body.
	b, _ := json.Marshal(map[string]interface{}{"jsonrpc": "2.0", "method": "notifications/initialized"})
	resp, err := http.Post(srv.URL+"/mcp", "application/json", bytes.NewReader(b))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("notification status = %d, want 202", resp.StatusCode)
	}
}

func TestA2AAgentCardAndSkills(t *testing.T) {
	srv := httptest.NewServer(NewMCPServer().Handler())
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/.well-known/agent-card.json")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var card map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&card)
	ifaces := card["supportedInterfaces"].([]interface{})
	if got := ifaces[0].(map[string]interface{})["url"]; got != "http://"+srv.Listener.Addr().String()+"/a2a" {
		t.Fatalf("interface url = %v", got)
	}
	if n := len(card["skills"].([]interface{})); n != len(operationRequestTypes) {
		t.Fatalf("card has %d skills, want %d", n, len(operationRequestTypes))
	}

	bad := postJSON(t, srv.URL+"/a2a", map[string]interface{}{"jsonrpc": "2.0", "id": 1, "method": "SendMessage",
		"params": map[string]interface{}{"message": map[string]interface{}{"parts": []interface{}{
			map[string]interface{}{"data": map[string]interface{}{"operation": "no_such_skill"}}}}}})
	if bad["error"] == nil {
		t.Fatal("unknown skill did not return a JSON-RPC error")
	}
}

// A request with a field the schema does not describe is rejected, not run on defaults.
func TestUnknownRequestFieldIsRejected(t *testing.T) {
	s := NewMCPServer()
	env := s.CallOperation(context.Background(), "get_capabilities", json.RawMessage(`{"filter": "adapter"}`))
	if env.Error == nil || env.Error.Code != "INVALID_REQUEST" {
		t.Fatalf("unknown field: got %+v, want INVALID_REQUEST", env.Error)
	}
	env = s.CallOperation(context.Background(), "get_capabilities", json.RawMessage(`{"filter_type": "adapter"}`))
	if env.Error != nil {
		t.Fatalf("valid request failed: %+v", env.Error)
	}
}

// A required field that is missing is rejected before the handler runs.
func TestMissingRequiredFieldIsRejected(t *testing.T) {
	s := NewMCPServer()
	env := s.CallOperation(context.Background(), "validate_route", json.RawMessage(`{}`))
	if env.Error == nil || env.Error.Code != "INVALID_REQUEST" {
		t.Fatalf("missing route_config_path: got %+v, want INVALID_REQUEST", env.Error)
	}
}
