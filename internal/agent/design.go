package agent

// ScaffoldFromIntentRequest describes natural-language design intent
type ScaffoldFromIntentRequest struct {
	Intent           string `json:"intent"`                      // e.g., "webhook to Kafka with PII redaction"
	TemplateShape    string `json:"template_shape,omitempty"`   // "passthrough" | "transform" | "contract-enforced"
	Domain           string `json:"domain"`                      // domain name for the route
	WithTestFixtures bool   `json:"with_test_fixtures,omitempty"` // Generate test fixtures?
}

// ScaffoldFromIntentResponse is the result of natural-language scaffolding
type ScaffoldFromIntentResponse struct {
	Success          bool              `json:"success"`
	RouteYAML        string            `json:"route_yaml"`           // Generated route config (text)
	TemplateUsed     string            `json:"template_used"`        // Which M4.2 template was used
	ValidationResult *ValidateResponse `json:"validation_result"`    // Validate result (should be valid)
	Placeholders     []Placeholder     `json:"placeholders"`         // Fields agent left for human review
	Reasoning        string            `json:"reasoning"`            // Why agent made these choices
	NextSteps        []string          `json:"next_steps"`           // How human should review/modify
}

// Placeholder marks a field the agent couldn't determine and left for human review
type Placeholder struct {
	Path       string `json:"path"`        // JSONPath in YAML (e.g., "$.routes.webhook.steps[0].translate.expr")
	Reason     string `json:"reason"`      // Why left as placeholder
	Suggestion string `json:"suggestion"` // Optional guidance (e.g., "JSONata expression to extract order ID")
}
