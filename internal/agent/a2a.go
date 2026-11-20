package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"time"
)

// A2A (Agent2Agent) binding. The card describes the agent's skills; SendMessage runs
// one operation and returns the result as a task. Each skill is one operation.

// AgentCard returns the A2A agent card for the server reachable at baseURL.
func (s *MCPServer) AgentCard(baseURL string) map[string]interface{} {
	ops := s.ListOperations()
	sort.Slice(ops, func(i, j int) bool { return ops[i].Name < ops[j].Name })
	skills := make([]map[string]interface{}, 0, len(ops))
	for _, op := range ops {
		skills = append(skills, map[string]interface{}{
			"id":          op.Name,
			"name":        op.Name,
			"description": op.Description,
			"tags":        []string{"dim", "route"},
		})
	}
	return map[string]interface{}{
		"name":        "dim",
		"description": "DIM route validation, testing, scaffolding, lineage, provenance and impact analysis",
		"version":     AgentInterfaceVersion,
		"supportedInterfaces": []map[string]interface{}{{
			"url":             baseURL + "/a2a",
			"protocolBinding": "JSONRPC",
			"protocolVersion": A2AProtocolVersion,
		}},
		"capabilities":       map[string]interface{}{"streaming": false},
		"defaultInputModes":  []string{"application/json"},
		"defaultOutputModes": []string{"application/json"},
		"skills":             skills,
	}
}

// HandleAgentCard serves the agent card. baseURL is derived from the request's Host.
func (s *MCPServer) HandleAgentCard(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(s.AgentCard("http://" + r.Host))
}

// HandleA2A serves the A2A JSON-RPC endpoint. It implements SendMessage.
func (s *MCPServer) HandleA2A(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "A2A requests are POSTed as JSON-RPC", http.StatusMethodNotAllowed)
		return
	}
	var req rpcRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeRPC(w, rpcResponse{JSONRPC: "2.0", ID: json.RawMessage("null"),
			Error: &rpcError{Code: rpcParseError, Message: err.Error()}})
		return
	}
	if req.JSONRPC != "2.0" || len(req.ID) == 0 {
		writeRPC(w, rpcResponse{JSONRPC: "2.0", ID: json.RawMessage("null"),
			Error: &rpcError{Code: rpcInvalidRequest, Message: "A2A requests need jsonrpc 2.0 and an id"}})
		return
	}
	if req.Method != "SendMessage" {
		writeRPC(w, rpcResponse{JSONRPC: "2.0", ID: req.ID,
			Error: &rpcError{Code: rpcMethodNotFound, Message: fmt.Sprintf("method %q not supported", req.Method)}})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()
	result, rpcErr := s.sendMessage(ctx, req.Params)
	resp := rpcResponse{JSONRPC: "2.0", ID: req.ID}
	if rpcErr != nil {
		resp.Error = rpcErr
	} else {
		resp.Result = result
	}
	writeRPC(w, resp)
}

// sendMessage runs the operation a message asks for and returns it as a task.
// The message carries one data part: {"operation": "...", "request": {...}}.
func (s *MCPServer) sendMessage(ctx context.Context, params json.RawMessage) (interface{}, *rpcError) {
	var p struct {
		Message struct {
			MessageID string `json:"messageId"`
			Parts     []struct {
				Text string                 `json:"text"`
				Data map[string]interface{} `json:"data"`
			} `json:"parts"`
		} `json:"message"`
	}
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, &rpcError{Code: rpcInvalidParams, Message: err.Error()}
	}

	var call struct {
		Operation string          `json:"operation"`
		Request   json.RawMessage `json:"request"`
	}
	found := false
	for _, part := range p.Message.Parts {
		if part.Data != nil {
			raw, _ := json.Marshal(part.Data)
			if err := json.Unmarshal(raw, &call); err == nil && call.Operation != "" {
				found = true
				break
			}
		}
		if part.Text != "" {
			if err := json.Unmarshal([]byte(part.Text), &call); err == nil && call.Operation != "" {
				found = true
				break
			}
		}
	}
	if !found {
		return nil, &rpcError{Code: rpcInvalidParams, Message: `message needs a part with {"operation": ..., "request": ...}`}
	}
	if _, ok := s.operations[call.Operation]; !ok {
		return nil, &rpcError{Code: rpcInvalidParams, Message: fmt.Sprintf("unknown skill %q", call.Operation)}
	}
	if len(call.Request) == 0 {
		call.Request = json.RawMessage("{}")
	}

	env := s.CallOperation(ctx, call.Operation, call.Request)
	var envMap map[string]interface{}
	raw, _ := json.Marshal(env)
	_ = json.Unmarshal(raw, &envMap)

	state := "TASK_STATE_COMPLETED"
	if env.Error != nil {
		state = "TASK_STATE_FAILED"
	}
	id := fmt.Sprintf("task-%d", time.Now().UnixNano())
	return map[string]interface{}{
		"task": map[string]interface{}{
			"id":        id,
			"contextId": p.Message.MessageID,
			"status":    map[string]interface{}{"state": state},
			"artifacts": []interface{}{map[string]interface{}{
				"artifactId": id + "-result",
				"parts":      []interface{}{map[string]interface{}{"data": envMap}},
			}},
		},
	}, nil
}
