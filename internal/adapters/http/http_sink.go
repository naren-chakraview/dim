package http

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/naren-chakraview/dim/internal/adapters"
	"github.com/naren-chakraview/dim/internal/engine"
	"github.com/naren-chakraview/dim/internal/obo"
)

// HTTPSink implements the Sink interface for HTTP endpoints with OBO support (M1.3.3)
type HTTPSink struct {
	url              string
	client           *http.Client
	timeout          time.Duration
	tokenExchanger   *obo.TokenExchanger // For OBO (M1.3.3)
	oboScope         string              // OBO scope (e.g., "read:orders")
	oboAudience      string              // OBO audience (e.g., "api.internal")
	oboTTL           int                 // OBO token TTL in seconds
	routeVersion     string              // Current route version for OBO token claims
}

// HTTPSinkConfig defines HTTP sink configuration
type HTTPSinkConfig struct {
	URL        string // Target HTTP endpoint
	Timeout    int    // Request timeout in milliseconds (default: 5000)
	OBO        *OBOConfig
}

// OBOConfig defines On-Behalf-Of token exchange configuration (M1.3.3)
type OBOConfig struct {
	Scope    string // Requested scope (e.g., "read:orders")
	Audience string // Intended recipient service
	TTL      int    // Token lifetime in seconds (default: 3600)
}

// NewHTTPSink creates a new HTTP sink adapter
func NewHTTPSink(config *HTTPSinkConfig) (*HTTPSink, error) {
	if config == nil {
		return nil, fmt.Errorf("HTTP sink config cannot be nil")
	}
	if config.URL == "" {
		return nil, fmt.Errorf("HTTP URL required")
	}

	timeout := time.Duration(config.Timeout) * time.Millisecond
	if timeout == 0 {
		timeout = 5 * time.Second
	}

	sink := &HTTPSink{
		url:     config.URL,
		timeout: timeout,
		client: &http.Client{
			Timeout: timeout,
		},
	}

	// Configure OBO if provided (M1.3.3)
	if config.OBO != nil {
		if config.OBO.Scope != "" {
			sink.tokenExchanger = obo.NewTokenExchanger("") // Secret set via SetOBOSecret
			sink.oboScope = config.OBO.Scope
			sink.oboAudience = config.OBO.Audience
			sink.oboTTL = config.OBO.TTL
			if sink.oboTTL == 0 {
				sink.oboTTL = 3600 // Default 1 hour
			}
		}
	}

	return sink, nil
}

// SetOBOSecret configures the shared secret for OBO token signing (M1.3.3)
func (s *HTTPSink) SetOBOSecret(secret string) {
	if s.tokenExchanger == nil {
		s.tokenExchanger = obo.NewTokenExchanger(secret)
	}
}

// SetRouteVersion sets the current route version for OBO token claims (M1.3.3)
func (s *HTTPSink) SetRouteVersion(version string) {
	s.routeVersion = version
}

// Write sends messages to the HTTP endpoint (M1.3.3 - with OBO support)
func (s *HTTPSink) Write(ctx context.Context, msgs []*engine.Message) []adapters.Result {
	results := make([]adapters.Result, len(msgs))

	for i, msg := range msgs {
		// Prepare authorization header
		authHeader := ""

		// If OBO is configured, exchange token (M1.3.3)
		if s.tokenExchanger != nil && msg.Metadata.Principal != nil && msg.Metadata.Principal.Token != "" {
			exchangeReq := &obo.ExchangeRequest{
				OriginalToken: msg.Metadata.Principal.Token,
				Scope:         s.oboScope,
				Audience:      s.oboAudience,
				TTL:           s.oboTTL,
			}

			exchangeResp, err := s.tokenExchanger.Exchange(ctx, exchangeReq)
			if err != nil {
				results[i] = adapters.Result{
					Message: msg,
					Error:   fmt.Errorf("OBO token exchange failed: %w", err),
					Success: false,
				}
				continue
			}

			authHeader = "Bearer " + exchangeResp.AccessToken
		} else if msg.Metadata.Principal != nil && msg.Metadata.Principal.Token != "" {
			// No OBO: use original token
			authHeader = "Bearer " + msg.Metadata.Principal.Token
		}

		// Prepare request body (message body as JSON)
		bodyBytes, err := json.Marshal(msg.Body)
		if err != nil {
			results[i] = adapters.Result{
				Message: msg,
				Error:   fmt.Errorf("failed to marshal message body: %w", err),
				Success: false,
			}
			continue
		}

		// Create HTTP request
		req, err := http.NewRequestWithContext(ctx, "POST", s.url, bytes.NewReader(bodyBytes))
		if err != nil {
			results[i] = adapters.Result{
				Message: msg,
				Error:   fmt.Errorf("failed to create HTTP request: %w", err),
				Success: false,
			}
			continue
		}

		// Set headers
		req.Header.Set("Content-Type", "application/json")
		if authHeader != "" {
			req.Header.Set("Authorization", authHeader)
		}

		// Add correlation ID if available
		if msg.Metadata.CorrelationID != "" {
			req.Header.Set("X-Correlation-ID", msg.Metadata.CorrelationID)
		}

		// Send request
		resp, err := s.client.Do(req)
		if err != nil {
			results[i] = adapters.Result{
				Message: msg,
				Error:   fmt.Errorf("HTTP request failed: %w", err),
				Success: false,
			}
			continue
		}

		// Check response status
		if resp.StatusCode >= 400 {
			bodyText, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			results[i] = adapters.Result{
				Message: msg,
				Error:   fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(bodyText)),
				Success: false,
			}
			continue
		}

		resp.Body.Close()
		results[i] = adapters.Result{
			Message: msg,
			Error:   nil,
			Success: true,
		}
	}

	return results
}

// HealthCheck verifies the HTTP endpoint is reachable
func (s *HTTPSink) HealthCheck(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, "GET", s.url, nil)
	if err != nil {
		return fmt.Errorf("failed to create health check request: %w", err)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 500 {
		return fmt.Errorf("health check returned %d", resp.StatusCode)
	}

	return nil
}

// Checkpoint returns adapter state (HTTP has no stateful offset)
func (s *HTTPSink) Checkpoint() map[string]int64 {
	return make(map[string]int64)
}

// Close closes the HTTP sink (no resources to close)
func (s *HTTPSink) Close() error {
	// HTTP client is stateless, nothing to close
	return nil
}
