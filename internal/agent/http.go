package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Handler serves every wire protocol for this server: MCP at /mcp, A2A at /a2a with
// its agent card, and the REST interface (/health, /operations, /call). All of them
// reach CallOperation.
func (s *MCPServer) Handler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(s.TestConnection(ctx))
	})

	mux.HandleFunc("/operations", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"operations": s.ListOperations()})
	})

	mux.HandleFunc("/call", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var req struct {
			Operation string          `json:"operation"`
			Request   json.RawMessage `json:"request"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
		defer cancel()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(s.CallOperation(ctx, req.Operation, req.Request))
	})

	mux.HandleFunc("/mcp", s.HandleMCP)
	mux.HandleFunc("/a2a", s.HandleA2A)
	mux.HandleFunc("/.well-known/agent-card.json", s.HandleAgentCard)
	return mux
}
