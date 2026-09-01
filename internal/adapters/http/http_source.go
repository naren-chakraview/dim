package http

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/naren-chakraview/dim/internal/engine"
)

// HTTPSource is an HTTP source adapter that listens for incoming HTTP requests
// and converts them to messages that are sent to an output channel.
type HTTPSource struct {
	server   *http.Server
	outChan  *engine.Channel
	closed   chan struct{}
	listener net.Listener
}

// NewHTTPSource creates a new HTTP source adapter listening on the given port
// at the specified path, sending messages to the output channel.
//
// The adapter will listen on the specified port with the path being the endpoint
// that accepts HTTP POST requests with JSON bodies.
func NewHTTPSource(port int, path string, outChan *engine.Channel) (*HTTPSource, error) {
	if outChan == nil {
		return nil, fmt.Errorf("output channel cannot be nil")
	}

	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return nil, fmt.Errorf("failed to listen on port %d: %w", port, err)
	}

	src := &HTTPSource{
		outChan:  outChan,
		closed:   make(chan struct{}),
		listener: listener,
	}

	// Create the HTTP server with the handler
	mux := http.NewServeMux()
	mux.HandleFunc(path, src.handleIngest)

	src.server = &http.Server{
		Handler: mux,
	}

	return src, nil
}

// Start begins listening for HTTP requests on the configured endpoint.
// It blocks until the context is cancelled, at which point it performs
// a graceful shutdown.
func (s *HTTPSource) Start(ctx context.Context) error {
	// Start the server in a goroutine
	errChan := make(chan error, 1)
	go func() {
		// Serve using the existing listener
		if err := s.server.Serve(s.listener); err != nil && err != http.ErrServerClosed {
			errChan <- err
		}
	}()

	// Wait for context cancellation or a server error
	select {
	case err := <-errChan:
		return fmt.Errorf("HTTP server error: %w", err)
	case <-ctx.Done():
		// Gracefully shutdown the server
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := s.server.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("HTTP server shutdown error: %w", err)
		}
		close(s.closed)
		return ctx.Err()
	}
}

// Close closes the HTTP source listener and performs a graceful shutdown.
func (s *HTTPSource) Close() error {
	if s.server != nil {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := s.server.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("failed to shutdown HTTP server: %w", err)
		}
	}
	return nil
}

// handleIngest handles incoming HTTP requests at the configured path.
// It expects a JSON body, converts it to a Message with metadata,
// and sends it to the output channel.
func (s *HTTPSource) handleIngest(w http.ResponseWriter, r *http.Request) {
	// Only accept POST and GET requests
	if r.Method != http.MethodPost && r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	// Parse the request body as JSON
	var body interface{}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"error": "malformed JSON"})
		return
	}

	// Create a message with the parsed body and set metadata
	// Note: route and routeVersion will be set by the executor context
	// For the HTTP source, we set them to empty for now; they'll be populated
	// by the route configuration when processing
	msg := &engine.Message{
		Headers: make(map[string]interface{}),
		Body:    body,
		Metadata: engine.Metadata{
			CorrelationID: generateHTTPCorrelationID(),
			IngestedAt:    time.Now().UTC(),
			Route:         "", // Will be set by route configuration
			RouteVersion:  "", // Will be set by route configuration
		},
	}

	// Send the message to the output channel
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := s.outChan.Send(ctx, msg); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"error": "failed to ingest message"})
		return
	}

	// Return 202 Accepted on success
	w.WriteHeader(http.StatusAccepted)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"correlation_id": msg.Metadata.CorrelationID,
		"status":         "accepted",
	})
}

// generateHTTPCorrelationID generates a correlation ID for HTTP ingestion.
// Using the same strategy as the engine's generateCorrelationID
func generateHTTPCorrelationID() string {
	return time.Now().UTC().Format("20060102150405000000")
}
