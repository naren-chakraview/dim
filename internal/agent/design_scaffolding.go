package agent

import (
	"context"
	"fmt"
	"os"
	"strings"
)

// ScaffoldFromIntent generates a route scaffold from natural-language intent
func ScaffoldFromIntent(ctx context.Context, req ScaffoldFromIntentRequest) (*ScaffoldFromIntentResponse, *OperationErr) {
	if req.Intent == "" {
		return nil, &OperationErr{
			Code:    "INVALID_REQUEST",
			Message: "intent is required",
		}
	}

	if req.Domain == "" {
		return nil, &OperationErr{
			Code:    "INVALID_REQUEST",
			Message: "domain is required",
		}
	}

	// Step 1: Select template shape
	templateShape := req.TemplateShape
	if templateShape == "" {
		templateShape = inferTemplateShape(req.Intent)
	}

	// Step 2: Validate template shape
	validShapes := map[string]bool{
		"passthrough":        true,
		"transform":          true,
		"contract-enforced":  true,
	}
	if !validShapes[templateShape] {
		return nil, &OperationErr{
			Code:    "INVALID_REQUEST",
			Message: fmt.Sprintf("invalid template_shape: %s", templateShape),
		}
	}

	// Step 3: Get template
	template := getTemplateByShape(templateShape)

	// Step 4: Extract intent signals
	signals := extractIntentSignals(req.Intent)

	// Step 5: Apply intent signals to template
	routeYAML := applyIntentToTemplate(template, req.Domain, signals)

	// Step 6: Validate the generated route (write to temp file)
	tempFile, err := os.CreateTemp("", "route-scaffold-*.yaml")
	if err != nil {
		return nil, &OperationErr{
			Code:    "INTERNAL_ERROR",
			Message: fmt.Sprintf("failed to create temp file: %v", err),
		}
	}
	defer os.Remove(tempFile.Name())

	if _, err := tempFile.WriteString(routeYAML); err != nil {
		tempFile.Close()
		return nil, &OperationErr{
			Code:    "INTERNAL_ERROR",
			Message: fmt.Sprintf("failed to write route to temp file: %v", err),
		}
	}
	tempFile.Close()

	validateReq := ValidateRequest{
		RouteConfigPath: tempFile.Name(),
		StrictMode:      false,
	}
	validateResp, validationErr := ValidateRoute(ctx, validateReq)
	if validationErr != nil {
		return nil, validationErr
	}

	// Step 7: Build response
	resp := &ScaffoldFromIntentResponse{
		Success:          validateResp.Valid,
		RouteYAML:        routeYAML,
		TemplateUsed:     templateShape,
		ValidationResult: validateResp,
		Placeholders:     identifyPlaceholders(signals, templateShape),
		Reasoning:        buildReasoning(req.Intent, templateShape, signals),
		NextSteps: []string{
			"1. Review the generated route YAML",
			"2. Fill in any placeholders marked with {{ }} comments",
			"3. Run `dimctl test` with fixtures to verify behavior",
			"4. Create a PR for domain team review",
			"5. After approval, merge to main via GitOps pipeline",
		},
	}

	return resp, nil
}

// inferTemplateShape infers which template shape to use from intent keywords
func inferTemplateShape(intent string) string {
	intentLower := strings.ToLower(intent)

	if strings.Contains(intentLower, "redact") || strings.Contains(intentLower, "mask") ||
		strings.Contains(intentLower, "transform") || strings.Contains(intentLower, "modify") ||
		strings.Contains(intentLower, "extract") {
		return "transform"
	}

	if strings.Contains(intentLower, "enforce") || strings.Contains(intentLower, "validate") ||
		strings.Contains(intentLower, "contract") {
		return "contract-enforced"
	}

	// Default to passthrough for simple connectors
	return "passthrough"
}

// getTemplateByShape returns the template text for a given shape
func getTemplateByShape(shape string) string {
	templates := map[string]string{
		"passthrough":       passthroughTemplate(),
		"transform":         transformTemplate(),
		"contract-enforced": contractEnforcedTemplate(),
	}
	if t, ok := templates[shape]; ok {
		return t
	}
	return passthroughTemplate() // fallback
}

// IntentSignals extracted from natural language
type IntentSignals struct {
	SourceType string
	SinkType   string
	StepTypes  []string
}

// extractIntentSignals pulls key signals from the intent description
func extractIntentSignals(intent string) IntentSignals {
	intentLower := strings.ToLower(intent)
	signals := IntentSignals{}

	// Extract source type (only valid schema types: http, file, sftp, exec)
	// Check for keywords that map to sources
	if strings.Contains(intentLower, "webhook") || strings.Contains(intentLower, "request") || strings.Contains(intentLower, "http") {
		signals.SourceType = "http"
	} else if strings.Contains(intentLower, "file") {
		signals.SourceType = "file"
	} else if strings.Contains(intentLower, "sftp") {
		signals.SourceType = "sftp"
	} else if strings.Contains(intentLower, "exec") || strings.Contains(intentLower, "command") {
		signals.SourceType = "exec"
	}

	// Extract sink type (only valid schema types)
	sinkTypes := []string{"http", "file", "sftp", "exec"}  // Valid types in schema
	for _, sinkType := range sinkTypes {
		if strings.Contains(intentLower, sinkType) {
			signals.SinkType = sinkType
			break
		}
	}

	// Extract step types needed
	if strings.Contains(intentLower, "filter") {
		signals.StepTypes = append(signals.StepTypes, "filter")
	}
	if strings.Contains(intentLower, "redact") || strings.Contains(intentLower, "transform") ||
		strings.Contains(intentLower, "mask") || strings.Contains(intentLower, "extract") ||
		strings.Contains(intentLower, "modify") {
		signals.StepTypes = append(signals.StepTypes, "translate")
	}

	return signals
}

// applyIntentToTemplate substitutes placeholders with extracted signals and defaults
func applyIntentToTemplate(template string, domain string, signals IntentSignals) string {
	result := template

	// Replace domain references
	result = strings.ReplaceAll(result, "{{ DOMAIN }}", domain)

	// Replace source - use detected type or default
	sourceType := "http" // default
	if signals.SourceType != "" {
		// Map common names to schema types
		sourceMapping := map[string]string{
			"webhook": "http",
			"queue":   "kafka",
		}
		if mapped, ok := sourceMapping[signals.SourceType]; ok {
			sourceType = mapped
		} else {
			sourceType = signals.SourceType
		}
	}
	result = strings.ReplaceAll(result, "{{ SOURCE_TYPE }}", sourceType)

	// Replace sink - use detected type or default
	sinkType := "file" // default
	if signals.SinkType != "" {
		sinkType = signals.SinkType
	}
	result = strings.ReplaceAll(result, "{{ SINK_TYPE }}", sinkType)

	// Replace transformation expression with default identity
	transformExpr := "$" // default identity transformation
	result = strings.ReplaceAll(result, "{{ TRANSFORM_EXPR }}", transformExpr)

	return result
}

// identifyPlaceholders identifies fields left for human review
func identifyPlaceholders(signals IntentSignals, shape string) []Placeholder {
	placeholders := []Placeholder{}

	if shape == "transform" && len(signals.StepTypes) > 0 {
		placeholders = append(placeholders, Placeholder{
			Path:       "$.routes[*].steps[*].translate.expr",
			Reason:     "JSONata expression for transformation not inferred from intent",
			Suggestion: "Provide a JSONata expression to extract/transform fields (e.g., `$ | {id: $.order_id, total: $.amount}`)",
		})
	}

	if signals.SinkType == "" {
		placeholders = append(placeholders, Placeholder{
			Path:       "$.sinks[*].type",
			Reason:     "Sink type could not be determined from intent",
			Suggestion: "Specify the destination: http, file, kafka, sftp, etc.",
		})
	}

	if signals.SourceType == "" {
		placeholders = append(placeholders, Placeholder{
			Path:       "$.sources[*].type",
			Reason:     "Source type could not be determined from intent",
			Suggestion: "Specify the source: http (webhook), file, kafka, sftp, exec, etc.",
		})
	}

	return placeholders
}

// buildReasoning explains why the agent made these template/signal choices
func buildReasoning(intent, shape string, signals IntentSignals) string {
	reasoning := fmt.Sprintf(
		"Selected '%s' template based on intent keywords. ",
		shape,
	)

	if signals.SourceType != "" {
		reasoning += fmt.Sprintf("Detected source type: %s. ", signals.SourceType)
	}
	if signals.SinkType != "" {
		reasoning += fmt.Sprintf("Detected sink type: %s. ", signals.SinkType)
	}

	reasoning += "Placeholders ({{ }} markers) indicate fields requiring human judgment."

	return reasoning
}

// Template implementations
func passthroughTemplate() string {
	return `version: 1
imports:
  - ../governance/fragments.yaml

sources:
  input:
    type: {{ SOURCE_TYPE }}

routes:
  passthrough:
    from: input
    error_path:
      target: error
    steps:
      - filter:
          expr: "true"

sinks:
  output:
    type: {{ SINK_TYPE }}

  error:
    type: file
    path: ./dlq/errors.jsonl
`
}

func transformTemplate() string {
	return `version: 1
imports:
  - ../governance/fragments.yaml

sources:
  input:
    type: {{ SOURCE_TYPE }}

routes:
  transform:
    from: input
    error_path:
      target: error
    steps:
      - translate:
          expr: '{{ TRANSFORM_EXPR }}'  # TODO: Implement transformation logic

sinks:
  output:
    type: {{ SINK_TYPE }}

  error:
    type: file
    path: ./dlq/errors.jsonl
`
}

func contractEnforcedTemplate() string {
	return `version: 1
imports:
  - ../governance/fragments.yaml

sources:
  input:
    type: {{ SOURCE_TYPE }}

routes:
  enforced:
    from: input
    error_path:
      target: error
    steps:
      - translate:
          expr: '{{ TRANSFORM_EXPR }}'

sinks:
  output:
    type: {{ SINK_TYPE }}
    enforce: true  # Enforce contract on output

  error:
    type: file
    path: ./dlq/errors.jsonl
`
}
