package pdp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/naren-chakraview/dim/internal/steps"
)

// OPAAdapter translates between dim's neutral PDP contract (M1.1) and OPA's Rego policies (M1.2.3)
type OPAAdapter struct {
	endpoint string
	timeout  time.Duration
	client   *http.Client
}

// NewOPAAdapter creates a new OPA adapter for a given endpoint
func NewOPAAdapter(endpoint string, timeoutMs int) (*OPAAdapter, error) {
	if endpoint == "" {
		return nil, fmt.Errorf("OPA endpoint cannot be empty")
	}

	timeout := time.Duration(timeoutMs) * time.Millisecond
	if timeout == 0 {
		timeout = 5 * time.Second
	}

	return &OPAAdapter{
		endpoint: endpoint,
		timeout:  timeout,
		client: &http.Client{
			Timeout: timeout,
		},
	}, nil
}

// Evaluate sends a decision request to OPA and returns the decision
// Follows the neutral contract from design/PDP_CONTRACT_SPEC.md
func (oa *OPAAdapter) Evaluate(ctx context.Context, req *steps.PDPDecisionRequest) (*steps.PDPDecisionResponse, error) {
	// Convert neutral request to OPA input format
	opaInput := oa.translateRequest(req)

	// Call OPA
	opaResult, err := oa.callOPA(ctx, opaInput)
	if err != nil {
		return nil, fmt.Errorf("OPA evaluation failed: %w", err)
	}

	// Translate OPA output back to neutral response
	resp := oa.translateResponse(opaResult)
	return resp, nil
}

// translateRequest converts a neutral DecisionRequest to OPA input format
// OPA expects input.principal, input.action, input.resource, input.context
func (oa *OPAAdapter) translateRequest(req *steps.PDPDecisionRequest) map[string]interface{} {
	return map[string]interface{}{
		"principal": map[string]interface{}{
			"subject":    req.Principal.Subject,
			"roles":      req.Principal.Roles,
			"attributes": req.Principal.Attributes,
		},
		"action":   req.Action,
		"resource": map[string]interface{}{
			"type":  req.Resource.Type,
			"id":    req.Resource.ID,
			"route": req.Resource.Route,
			"stage": req.Resource.Stage,
		},
		"context": map[string]interface{}{
			"timestamp":      req.Context.Timestamp,
			"correlation_id": req.Context.CorrelationID,
			"environment":    req.Context.Environment,
		},
	}
}

// callOPA makes an HTTP request to OPA's /v1/data/dim/authorize endpoint
// OPA API expects: POST /v1/data/dim/authorize with input in body
// Returns the OPA result (usually a map with "result" key)
func (oa *OPAAdapter) callOPA(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
	// OPA API body format
	reqBody := map[string]interface{}{
		"input": input,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal OPA request: %w", err)
	}

	// Create HTTP request with context timeout
	httpReq, err := http.NewRequestWithContext(ctx, "POST", oa.endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create OPA request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	// Execute request
	resp, err := oa.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("OPA request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read OPA response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("OPA returned status %d: %s", resp.StatusCode, string(respBody))
	}

	// Unmarshal OPA response
	var opaResp map[string]interface{}
	if err := json.Unmarshal(respBody, &opaResp); err != nil {
		return nil, fmt.Errorf("failed to parse OPA response: %w", err)
	}

	return opaResp, nil
}

// translateResponse converts OPA output to a neutral DecisionResponse
// OPA policies should set:
// - result.allow (boolean)
// - result.reason (string, optional)
// - result.obligations (array, optional)
func (oa *OPAAdapter) translateResponse(opaResult map[string]interface{}) *steps.PDPDecisionResponse {
	resp := &steps.PDPDecisionResponse{
		Decision: "deny", // default
		Metadata: make(map[string]interface{}),
	}

	// Extract result from OPA response
	result, ok := opaResult["result"]
	if !ok {
		resp.Reason = "OPA returned no result"
		return resp
	}

	resultMap, ok := result.(map[string]interface{})
	if !ok {
		resp.Reason = "OPA result is not a map"
		return resp
	}

	// Check allow decision
	if allow, ok := resultMap["allow"].(bool); ok && allow {
		resp.Decision = "allow"
	} else {
		resp.Decision = "deny"
	}

	// Extract reason
	if reason, ok := resultMap["reason"].(string); ok {
		resp.Reason = reason
	}

	// Extract obligations (OPA policy sets this as array of objects)
	if obligations, ok := resultMap["obligations"].([]interface{}); ok {
		for _, obl := range obligations {
			if oblMap, ok := obl.(map[string]interface{}); ok {
				obligation := steps.PDPObligation{
					Type:       fmt.Sprintf("%v", oblMap["type"]),
					Parameters: make(map[string]interface{}),
				}
				if params, ok := oblMap["parameters"].(map[string]interface{}); ok {
					obligation.Parameters = params
				}
				resp.Obligations = append(resp.Obligations, obligation)
			}
		}
	}

	// Store OPA metadata for audit
	resp.Metadata["opa_endpoint"] = oa.endpoint

	return resp
}
