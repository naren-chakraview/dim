package agent

import (
	"context"
	"fmt"

	"github.com/naren-chakraview/dim/internal/config"
)

// CritiqueRouteRequest asks the agent to review an existing route
type CritiqueRouteRequest struct {
	RouteConfigPath string `json:"route_config_path"`          // Path to route YAML to critique
	DomainRules     string `json:"domain_rules,omitempty"`     // Optional domain-specific conventions JSON
}

// CritiqueRouteResponse contains the agent's critique findings
type CritiqueRouteResponse struct {
	Findings       []CritiqueFinding `json:"findings"`         // Issues found
	Summary        string            `json:"summary"`          // Human-readable summary
	StructuralOK   bool              `json:"structural_ok"`    // Does it pass dimctl validate?
	OverallRating  string            `json:"overall_rating"`   // "excellent" | "good" | "fair" | "needs-review"
}

// CritiqueFinding describes one issue the agent found
type CritiqueFinding struct {
	Category  string `json:"category"`  // "redundancy" | "convention" | "best-practice" | "security" | "clarity"
	Severity  string `json:"severity"`  // "info" | "warning" | "important"
	Path      string `json:"path"`      // JSONPath to the issue
	Issue     string `json:"issue"`     // What's wrong?
	Action    string `json:"action"`    // What should the human do about it?
	Example   string `json:"example,omitempty"` // Optional: before/after example
}

// CritiqueRoute analyzes an existing route and flags non-structural issues
func CritiqueRoute(ctx context.Context, req CritiqueRouteRequest) (*CritiqueRouteResponse, *OperationErr) {
	if req.RouteConfigPath == "" {
		return nil, &OperationErr{
			Code:    "INVALID_REQUEST",
			Message: "route_config_path is required",
		}
	}

	// Step 1: Load the route (must be structurally valid first)
	cfg, err := config.LoadRouteConfig(req.RouteConfigPath)
	if err != nil {
		return nil, &OperationErr{
			Code:    "VALIDATION_FAILED",
			Message: fmt.Sprintf("Failed to load route config: %v", err),
		}
	}

	// Step 2: Run structural validation
	validateReq := ValidateRequest{
		RouteConfigPath: req.RouteConfigPath,
		StrictMode:      false,
	}
	validateResp, validationErr := ValidateRoute(ctx, validateReq)
	if validationErr != nil {
		return nil, validationErr
	}

	// Step 3: Run critique patterns
	var allFindings []CritiqueFinding
	for _, pattern := range critiquPatterns {
		findings := pattern.Check(cfg)
		allFindings = append(allFindings, findings...)
	}

	// Step 4: Apply domain-specific rules if provided
	// For MVP, domain rules parsing is optional future work
	_ = req.DomainRules

	// Step 5: Calculate overall rating
	overallRating := calculateOverallRating(allFindings, validateResp)

	// Step 6: Build response
	resp := &CritiqueRouteResponse{
		Findings:      allFindings,
		Summary:       buildCritiqueSummary(allFindings),
		StructuralOK:  validateResp.Valid,
		OverallRating: overallRating,
	}

	return resp, nil
}
