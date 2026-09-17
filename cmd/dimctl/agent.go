package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/naren-chakraview/dim/internal/agent"
	"github.com/spf13/cobra"
)

var agentCmd = &cobra.Command{
	Use:   "agent",
	Short: "AI agent interface server",
	Long:  "Start or interact with the DIM agent interface (MCP-compatible)",
}

var agentServeCmd = &cobra.Command{
	Use:   "serve [--addr localhost:9090]",
	Short: "Start the agent interface server",
	Long:  "Start the MCP-compatible agent server listening for requests from Claude or other AI clients",
	RunE: func(cmd *cobra.Command, args []string) error {
		addr, _ := cmd.Flags().GetString("addr")
		return serveAgent(addr)
	},
}

// serveAgent starts the agent interface server
func serveAgent(addr string) error {
	// Create MCP server
	mcpServer := agent.NewMCPServer()

	// Create HTTP server that wraps MCP protocol
	mux := http.NewServeMux()

	// Health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		
		response := mcpServer.TestConnection(ctx)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	})

	// List operations endpoint
	mux.HandleFunc("/operations", func(w http.ResponseWriter, r *http.Request) {
		operations := mcpServer.ListOperations()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"operations": operations,
		})
	})

	// Call operation endpoint
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

		// Execute operation with timeout
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		response := mcpServer.CallOperation(ctx, req.Operation, req.Request)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	})

	// Create HTTP server
	server := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	// Start server in background
	serverErrors := make(chan error, 1)
	go func() {
		log.Printf("Agent server listening on http://%s", addr)
		serverErrors <- server.ListenAndServe()
	}()

	// Wait for signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-serverErrors:
		if err != http.ErrServerClosed {
			return fmt.Errorf("server error: %w", err)
		}
		return nil
	case sig := <-sigChan:
		log.Printf("Received signal: %v, shutting down", sig)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return server.Shutdown(ctx)
	}
}

// init adds agent subcommands
func init() {
	agentServeCmd.Flags().String("addr", "localhost:9090", "Address to listen on (host:port)")
	agentCmd.AddCommand(agentServeCmd)
}
