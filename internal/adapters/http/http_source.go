package http

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/naren-chakraview/dim/internal/authz"
	"github.com/naren-chakraview/dim/internal/engine"
)

// HTTPSource is an HTTP source adapter that listens for incoming HTTP requests
// and converts them to messages that are sent to an output channel.
type HTTPSource struct {
	server         *http.Server
	outChan        *engine.Channel
	closed         chan struct{}
	listener       net.Listener
	jwtValidator   *authz.JWTValidator
	requireAuth    bool
}

// NewHTTPSource creates a new HTTP source adapter listening on the given port
// at the specified path, sending messages to the output channel.
//
// The adapter will listen on the specified port with the path being the endpoint
// that accepts HTTP POST requests with JSON bodies.
// JWT validation is optional; if not configured, principals will be nil.
func NewHTTPSource(port int, path string, outChan *engine.Channel) (*HTTPSource, error) {
	return NewHTTPSourceWithAuth(port, path, outChan, nil, false)
}

// NewHTTPSourceWithAuth creates an HTTP source adapter with optional JWT validation
// jwtValidator can be nil to skip JWT validation
// requireAuth determines if 401 is returned when no valid auth is present
func NewHTTPSourceWithAuth(port int, path string, outChan *engine.Channel, jwtValidator *authz.JWTValidator, requireAuth bool) (*HTTPSource, error) {
	if outChan == nil {
		return nil, fmt.Errorf("output channel cannot be nil")
	}

	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return nil, fmt.Errorf("failed to listen on port %d: %w", port, err)
	}

	src := &HTTPSource{
		outChan:      outChan,
		closed:       make(chan struct{}),
		listener:     listener,
		jwtValidator: jwtValidator,
		requireAuth:  requireAuth,
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
// Route detection: X-Route header, path component, or query parameter
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

	// Detect route from HTTP request
	// Priority: X-Route header > path > query parameter
	routeName := s.detectRoute(r)

	// Extract principal from Authorization header if JWT validator is configured
	var principal *engine.Principal
	if s.jwtValidator != nil {
		authHeader := r.Header.Get("Authorization")
		authzPrincipal, err := authz.ExtractPrincipalFromHeader(authHeader, s.jwtValidator)
		if err != nil {
			// Invalid token present
			w.WriteHeader(http.StatusUnauthorized)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
			return
		}
		// If no token present and auth is required, reject
		if authzPrincipal == nil && s.requireAuth {
			w.WriteHeader(http.StatusUnauthorized)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
			return
		}
		// Convert authz.Principal to engine.Principal
		if authzPrincipal != nil {
			principal = &engine.Principal{
				Subject: authzPrincipal.Subject,
				Roles:   authzPrincipal.Roles,
				Claims:  convertAttributesToClaims(authzPrincipal.Attributes),
			}
		}
	}

	// Create a message with the parsed body and set metadata
	msg := &engine.Message{
		Headers: make(map[string]interface{}),
		Body:    body,
		Metadata: engine.Metadata{
			CorrelationID: generateHTTPCorrelationID(),
			IngestedAt:    time.Now().UTC(),
			Route:         routeName, // Set from HTTP request
			RouteVersion:  "",        // Will be set by executor
			Principal:     principal, // Set from Authorization header
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
		"route":          routeName,
		"status":         "accepted",
	})
}

// detectRoute extracts the route name from the HTTP request.
// Detection order:
// 1. X-Route header (explicit)
// 2. Path component: /ingest/{routeName}
// 3. Query parameter: ?route=routeName
// 4. Empty string if not found (router will need to handle)
func (s *HTTPSource) detectRoute(r *http.Request) string {
	// Check X-Route header
	if route := r.Header.Get("X-Route"); route != "" {
		return route
	}

	// Check query parameter
	if route := r.URL.Query().Get("route"); route != "" {
		return route
	}

	// Check path component: /ingest/{routeName}
	// Path format: /ingest or /ingest/{routeName} or /ingest/{routeName}/...
	path := r.URL.Path
	parts := strings.Split(path, "/")

	// Format: ["", "ingest", "routeName", ...]
	if len(parts) > 2 && parts[1] == "ingest" && parts[2] != "" {
		return parts[2]
	}

	// No route detected
	return ""
}

// generateHTTPCorrelationID generates a correlation ID for HTTP ingestion.
// Using the same strategy as the engine's generateCorrelationID
func generateHTTPCorrelationID() string {
	return time.Now().UTC().Format("20060102150405000000")
}

// convertAttributesToClaims converts string attributes map to interface{} claims map
func convertAttributesToClaims(attrs map[string]string) map[string]interface{} {
	if len(attrs) == 0 {
		return nil
	}
	claims := make(map[string]interface{})
	for k, v := range attrs {
		claims[k] = v
	}
	return claims
}
