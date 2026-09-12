package agent

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
