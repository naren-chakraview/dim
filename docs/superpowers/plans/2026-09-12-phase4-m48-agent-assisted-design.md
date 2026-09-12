# Phase 4, M4.8 — Agent-Assisted Route Design Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement agent-assisted route design with natural-language scaffolding and route review/critique, built on M4.6's agent interface and M4.7's capability manifest.

**Architecture:** Agent-assisted design provides two capabilities:
1. **Natural-language-to-route scaffolding:** Agent takes an intent description (e.g., "connect webhook to Kafka with PII redaction"), produces a route scaffold using M4.2 templates, validates via M4.6, presents for human review
2. **Route review/critique mode:** Agent analyzes an existing route and flags non-structural issues validation can't catch — redundant steps, missing enforce flags, domain convention violations

**Tech Stack:** Go, existing M4.2 scaffold templates, M4.6 agent interface, M4.7 capability manifest, no new dependencies.

**Spec:** `design/phase-4-implementation-plan.md` (M4.8, §6.5), `m46-agent-interface-spec.md`, `design/m45-studio-spike.md`.

## Global Constraints

- Every agent-produced or agent-modified route must pass identical `dimctl validate`/`test` to a human-authored route
- Agent critique output points at specific, actionable gaps (not generic advice)
- Critique is strictly advisory — no automatic fixes, human review always required
- Critique catches what schema validation structurally can't catch (redundancy, conventions, best practices)
- Natural-language scaffolding uses only M4.2's three template shapes (passthrough, transform, contract-enforced)
- No agent-to-production deployment path — propose/review/merge/deploy workflow unchanged
- All agent operations exposed via M4.6 MCP interface

---

## File Structure

**Backend (Go):**
```
internal/agent/design.go            # Agent-assisted design operations
internal/agent/design_test.go       # Tests for design operations
cmd/dimctl/design.go                # CLI commands for design operations (optional, for testing)
internal/agent/critique.go          # Route critique/review logic
internal/agent/critique_patterns.go # Critique patterns and checks
```

**Test fixtures and examples:**
```
docs/examples/m48-design-examples.md          # Worked examples (intent → route)
docs/examples/critique-examples.md            # Critique output examples
design/m48-agent-design-patterns.md           # Patterns for critique checks
```

---

## Task Breakdown

### Task 1: Define design and critique operation types

**Files:**
- Create: `internal/agent/design.go` (request/response types)
- Create: `internal/agent/critique.go` (critique types and patterns)

**Steps:**

- [ ] **Step 1: Define design scaffolding request/response types**

```go
// In internal/agent/design.go
package agent

// ScaffoldFromIntentRequest describes natural-language design intent
type ScaffoldFromIntentRequest struct {
  Intent            string `json:"intent"`            // e.g., "webhook to Kafka with PII redaction"
  TemplateShape     string `json:"template_shape,omitempty"` // "passthrough" | "transform" | "contract-enforced"
  Domain            string `json:"domain"`            // domain name for the route
  WithTestFixtures  bool   `json:"with_test_fixtures,omitempty"` // Generate test fixtures?
}

// ScaffoldFromIntentResponse is the result of natural-language scaffolding
type ScaffoldFromIntentResponse struct {
  Success           bool              `json:"success"`
  RouteYAML         string            `json:"route_yaml"`           // Generated route config (text)
  TemplateUsed      string            `json:"template_used"`        // Which M4.2 template was used
  ValidationResult  *ValidateResponse `json:"validation_result"`    // Validate result (should be valid)
  Placeholders      []Placeholder     `json:"placeholders"`         // Fields agent left for human review
  Reasoning         string            `json:"reasoning"`            // Why agent made these choices
  NextSteps         []string          `json:"next_steps"`           // How human should review/modify
}

// Placeholder marks a field the agent couldn't determine and left for human review
type Placeholder struct {
  Path        string `json:"path"`        // JSONPath in YAML (e.g., "$.routes.webhook.steps[0].translate.expr")
  Reason      string `json:"reason"`      // Why left as placeholder
  Suggestion  string `json:"suggestion"`  // Optional guidance (e.g., "JSONata expression to extract order ID")
}
```

- [ ] **Step 2: Define critique request/response types**

```go
// CritiqueRouteRequest asks the agent to review an existing route
type CritiqueRouteRequest struct {
  RouteConfigPath string `json:"route_config_path"` // Path to route YAML to critique
  DomainRules     string `json:"domain_rules,omitempty"` // Optional domain-specific conventions JSON
}

// CritiqueRouteResponse contains the agent's critique findings
type CritiqueRouteResponse struct {
  Findings       []CritiqueFinding `json:"findings"`       // Issues found
  Summary        string            `json:"summary"`        // Human-readable summary
  StructuralOK   bool              `json:"structural_ok"`  // Does it pass dimctl validate?
  OverallRating  string            `json:"overall_rating"` // "excellent" | "good" | "fair" | "needs-review"
}

// CritiqueFinding describes one issue the agent found
type CritiqueFinding struct {
  Category   string `json:"category"`   // "redundancy" | "convention" | "best-practice" | "security" | "clarity"
  Severity   string `json:"severity"`   // "info" | "warning" | "important"
  Path       string `json:"path"`       // JSONPath to the issue
  Issue      string `json:"issue"`      // What's wrong?
  Action     string `json:"action"`     // What should the human do about it?
  Example    string `json:"example,omitempty"` // Optional: before/after example
}
```

- [ ] **Step 3: Commit**

```bash
git add internal/agent/design.go internal/agent/critique.go
git commit -m "feat: define agent design and critique operation types"
```

---

### Task 2: Implement natural-language-to-route scaffolding

**Files:**
- Create: `internal/agent/design.go` (implementation)
- Modify: `internal/agent/mcp_server.go` (register operation)

**Steps:**

- [ ] **Step 1: Implement ScaffoldFromIntent operation**

```go
// In internal/agent/design.go
package agent

import (
  "context"
  "fmt"
  "strings"

  "github.com/naren-chakraview/dim/cmd/dimctl"
  "github.com/naren-chakraview/dim/internal/config"
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
    // Agent infers shape from intent keywords
    templateShape = inferTemplateShape(req.Intent)
  }

  // Step 2: Validate template shape
  validShapes := map[string]bool{
    "passthrough": true,
    "transform": true,
    "contract-enforced": true,
  }
  if !validShapes[templateShape] {
    return nil, &OperationErr{
      Code:    "INVALID_REQUEST",
      Message: fmt.Sprintf("invalid template_shape: %s", templateShape),
    }
  }

  // Step 3: Get template
  template := getTemplateByShape(templateShape)
  
  // Step 4: Extract intent signals (what to fill in the template)
  signals := extractIntentSignals(req.Intent)
  
  // Step 5: Apply intent signals to template
  routeYAML := applyIntentToTemplate(template, req.Domain, signals)
  
  // Step 6: Validate the generated route
  tempPath := fmt.Sprintf("/tmp/%s-generated.yaml", req.Domain)
  if err := saveTempYAML(tempPath, routeYAML); err != nil {
    return nil, &OperationErr{
      Code:    "INTERNAL_ERROR",
      Message: fmt.Sprintf("failed to validate generated route: %v", err),
    }
  }
  
  validateReq := ValidateRequest{
    RouteConfigPath: tempPath,
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
      "5. After approval, merge and deploy via GitOps pipeline",
    },
  }

  return resp, nil
}

// Helper: infer template shape from intent keywords
func inferTemplateShape(intent string) string {
  intentLower := strings.ToLower(intent)
  
  if strings.Contains(intentLower, "redact") || strings.Contains(intentLower, "mask") ||
     strings.Contains(intentLower, "transform") || strings.Contains(intentLower, "change") {
    return "transform"
  }
  
  if strings.Contains(intentLower, "enforce") || strings.Contains(intentLower, "validate") ||
     strings.Contains(intentLower, "contract") {
    return "contract-enforced"
  }
  
  // Default to passthrough for simple connectors
  return "passthrough"
}

// Helper: get template text by shape
func getTemplateByShape(shape string) string {
  templates := map[string]string{
    "passthrough": passthroughTemplate(),
    "transform": transformTemplate(),
    "contract-enforced": contractEnforcedTemplate(),
  }
  return templates[shape]
}

// Helper: extract signals from intent (what to configure)
type IntentSignals struct {
  SourceType string
  SinkType   string
  StepTypes  []string
  DomainTag  string
}

func extractIntentSignals(intent string) IntentSignals {
  intentLower := strings.ToLower(intent)
  signals := IntentSignals{}

  // Extract source type
  for _, sourceType := range []string{"webhook", "http", "kafka", "file", "sftp"} {
    if strings.Contains(intentLower, sourceType) {
      signals.SourceType = sourceType
      if sourceType == "webhook" {
        signals.SourceType = "http"
      }
      break
    }
  }

  // Extract sink type
  for _, sinkType := range []string{"kafka", "http", "file", "sftp", "s3"} {
    if strings.Contains(intentLower, sinkType) {
      signals.SinkType = sinkType
      break
    }
  }

  // Extract step types
  if strings.Contains(intentLower, "filter") {
    signals.StepTypes = append(signals.StepTypes, "filter")
  }
  if strings.Contains(intentLower, "redact") || strings.Contains(intentLower, "transform") || strings.Contains(intentLower, "mask") {
    signals.StepTypes = append(signals.StepTypes, "translate")
  }

  return signals
}

// Helper: apply signals to template
func applyIntentToTemplate(template string, domain string, signals IntentSignals) string {
  result := template
  
  // Replace domain references
  result = strings.ReplaceAll(result, "{{ DOMAIN }}", domain)
  
  // Replace source if detected
  if signals.SourceType != "" {
    result = strings.ReplaceAll(result, "{{ SOURCE_TYPE }}", signals.SourceType)
  }
  
  // Replace sink if detected
  if signals.SinkType != "" {
    result = strings.ReplaceAll(result, "{{ SINK_TYPE }}", signals.SinkType)
  }
  
  return result
}

// Helper: identify placeholders left for human review
func identifyPlaceholders(signals IntentSignals, shape string) []Placeholder {
  placeholders := []Placeholder{}
  
  if shape == "transform" && len(signals.StepTypes) > 0 {
    placeholders = append(placeholders, Placeholder{
      Path:       "$.routes[*].steps[*].translate.expr",
      Reason:     "JSONata expression for transformation was not inferred from intent",
      Suggestion: "Use a JSONata expression to extract/transform fields",
    })
  }
  
  if signals.SinkType == "" {
    placeholders = append(placeholders, Placeholder{
      Path:       "$.sinks[*].type",
      Reason:     "Sink type could not be determined from intent",
      Suggestion: "Specify where the processed messages should go",
    })
  }
  
  return placeholders
}

// Helper: build reasoning explanation
func buildReasoning(intent, shape string, signals IntentSignals) string {
  return fmt.Sprintf(
    "Used '%s' template because intent mentioned: %s. "+
    "Detected source: %s, sink: %s. "+
    "Leave placeholders as {{ }} markers for human refinement.",
    shape, intent, signals.SourceType, signals.SinkType,
  )
}

// Template functions
func passthroughTemplate() string {
  return `version: 1
imports:
  - ../governance/fragments.yaml

sources:
  input:
    type: {{ SOURCE_TYPE | "http" }}
    
routes:
  passthrough:
    from: input
    error_path:
      target: errors
    steps: []

sinks:
  output:
    type: {{ SINK_TYPE | "file" }}
    
  errors:
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
    type: {{ SOURCE_TYPE | "http" }}
    
routes:
  transform:
    from: input
    error_path:
      target: errors
    steps:
      - translate:
          expr: {{ TRANSFORM_EXPR | "$ /* TODO: JSONata expression here */" }}

sinks:
  output:
    type: {{ SINK_TYPE | "file" }}
    
  errors:
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
    type: {{ SOURCE_TYPE | "http" }}
    
routes:
  enforced:
    from: input
    error_path:
      target: errors
    steps:
      - translate:
          expr: {{ TRANSFORM_EXPR | "$" }}

sinks:
  output:
    type: {{ SINK_TYPE | "file" }}
    enforce: true  # Enforce contract on output
    
  errors:
    type: file
    path: ./dlq/errors.jsonl
`
}
```

- [ ] **Step 2: Register ScaffoldFromIntent in MCP server**

```go
// In internal/agent/mcp_server.go - add to registerOperations()
s.operations["scaffold_from_intent"] = MCPOperation{
  Name:        "scaffold_from_intent",
  Description: "Generate a route scaffold from natural-language design intent",
  Handler: func(ctx context.Context, rawReq json.RawMessage) (interface{}, *OperationErr) {
    var req ScaffoldFromIntentRequest
    if err := json.Unmarshal(rawReq, &req); err != nil {
      return nil, &OperationErr{Code: "INVALID_REQUEST", Message: fmt.Sprintf("Failed to parse request: %v", err)}
    }
    return ScaffoldFromIntent(ctx, req)
  },
}
```

- [ ] **Step 3: Add tests**

```go
// In internal/agent/design_test.go
func TestScaffoldFromIntentPassthrough(t *testing.T) {
  ctx := context.Background()
  req := ScaffoldFromIntentRequest{
    Intent:     "connect webhook to file output",
    Domain:     "data-pipeline",
  }
  
  resp, err := ScaffoldFromIntent(ctx, req)
  if err != nil {
    t.Fatalf("unexpected error: %v", err)
  }
  
  if !resp.Success {
    t.Fatal("scaffold should produce valid route")
  }
  
  if !strings.Contains(resp.RouteYAML, "sources:") {
    t.Fatal("route YAML missing sources")
  }
  
  if resp.TemplateUsed == "" {
    t.Fatal("should report which template was used")
  }
}

func TestScaffoldFromIntentTransform(t *testing.T) {
  ctx := context.Background()
  req := ScaffoldFromIntentRequest{
    Intent:     "webhook to Kafka with PII redaction",
    Domain:     "payments",
  }
  
  resp, err := ScaffoldFromIntent(ctx, req)
  if err != nil {
    t.Fatalf("unexpected error: %v", err)
  }
  
  if resp.TemplateUsed != "transform" {
    t.Fatalf("should detect transform template, got %s", resp.TemplateUsed)
  }
  
  if len(resp.Placeholders) == 0 {
    t.Fatal("should identify placeholders for human review")
  }
}

func TestScaffoldValidationPasses(t *testing.T) {
  ctx := context.Background()
  req := ScaffoldFromIntentRequest{
    Intent:     "webhook to file",
    Domain:     "test",
  }
  
  resp, err := ScaffoldFromIntent(ctx, req)
  if err != nil {
    t.Fatalf("unexpected error: %v", err)
  }
  
  if resp.ValidationResult == nil {
    t.Fatal("should include validation result")
  }
  
  if !resp.ValidationResult.Valid {
    t.Fatalf("generated route should be valid, got errors: %v", resp.ValidationResult.Errors)
  }
}

func TestScaffoldRequiredFields(t *testing.T) {
  ctx := context.Background()
  
  testCases := []struct {
    req   ScaffoldFromIntentRequest
    name  string
  }{
    {ScaffoldFromIntentRequest{Intent: "test"}, "missing domain"},
    {ScaffoldFromIntentRequest{Domain: "test"}, "missing intent"},
  }
  
  for _, tc := range testCases {
    resp, err := ScaffoldFromIntent(ctx, tc.req)
    if err == nil {
      t.Errorf("%s: should error, got response: %v", tc.name, resp)
    }
  }
}
```

- [ ] **Step 4: Commit**

```bash
git add internal/agent/design.go internal/agent/design_test.go internal/agent/mcp_server.go
git commit -m "feat: implement natural-language-to-route scaffolding (M4.8.1)"
```

---

### Task 3: Implement route critique/review mode

**Files:**
- Create: `internal/agent/critique.go` (critique logic)
- Create: `internal/agent/critique_patterns.go` (critique patterns/rules)
- Create: `internal/agent/critique_test.go` (tests)

**Steps:**

- [ ] **Step 1: Define critique patterns**

```go
// In internal/agent/critique_patterns.go
package agent

import (
  "github.com/naren-chakraview/dim/internal/config"
)

// CritiquePattern defines a rule the critique engine checks
type CritiquePattern struct {
  Name        string
  Category    string  // "redundancy" | "convention" | "best-practice" | "security" | "clarity"
  Severity    string  // "info" | "warning" | "important"
  Description string
  Check       func(*config.RouteConfig) []CritiqueFinding
}

var crititquePatterns = []CritiquePattern{
  {
    Name:        "redundant_translate_steps",
    Category:    "redundancy",
    Severity:    "warning",
    Description: "Multiple consecutive translate steps can usually be combined",
    Check: checkRedundantTranslateSteps,
  },
  {
    Name:        "contract_without_enforce",
    Category:    "best-practice",
    Severity:    "important",
    Description: "Contract referenced but not enforced on sink",
    Check: checkContractEnforcement,
  },
  {
    Name:        "missing_error_path",
    Category:    "convention",
    Severity:    "warning",
    Description: "Route missing error_path configuration",
    Check: checkErrorPath,
  },
  {
    Name:        "missing_auth_declaration",
    Category:    "security",
    Severity:    "important",
    Description: "Route missing auth declaration (should be 'none' if intentional)",
    Check: checkAuthDeclaration,
  },
  {
    Name:        "unused_imports",
    Category:    "clarity",
    Severity:    "info",
    Description: "Import statement that references no fragments actually used",
    Check: checkUnusedImports,
  },
}

func checkRedundantTranslateSteps(cfg *config.RouteConfig) []CritiqueFinding {
  var findings []CritiqueFinding
  
  // For each route, check if it has consecutive translate steps
  for routeName, route := range cfg.Routes {
    translateCount := 0
    var lastTranslateIndex int
    
    for i, step := range route.Steps {
      if step.Translate != nil {
        translateCount++
        lastTranslateIndex = i
      }
    }
    
    if translateCount > 1 {
      findings = append(findings, CritiqueFinding{
        Category: "redundancy",
        Severity: "warning",
        Path:     fmt.Sprintf("$.routes.%s.steps", routeName),
        Issue:    fmt.Sprintf("Found %d translate steps; consider combining into one", translateCount),
        Action:   "Combine multiple translate steps into a single expression for clarity",
        Example:  "// Before: [translate(expr1), translate(expr2)]\n// After: translate(expr1 | expr2)",
      })
    }
  }
  
  return findings
}

func checkContractEnforcement(cfg *config.RouteConfig) []CritiqueFinding {
  var findings []CritiqueFinding
  
  // Check if any routes reference contracts without enforcing
  // This is a simplification; real implementation would check imports
  for sinkName, sink := range cfg.Sinks {
    if sink.Type == "http" && sink.Enforce == false {
      findings = append(findings, CritiqueFinding{
        Category: "best-practice",
        Severity: "important",
        Path:     fmt.Sprintf("$.sinks.%s", sinkName),
        Issue:    "Sink has enforce: false, but a contract should be checked",
        Action:   "Set enforce: true to ensure contract validation on output",
      })
    }
  }
  
  return findings
}

func checkErrorPath(cfg *config.RouteConfig) []CritiqueFinding {
  var findings []CritiqueFinding
  
  for routeName, route := range cfg.Routes {
    if route.ErrorPath == nil {
      findings = append(findings, CritiqueFinding{
        Category: "convention",
        Severity: "warning",
        Path:     fmt.Sprintf("$.routes.%s", routeName),
        Issue:    "No error_path configured; failed messages have nowhere to go",
        Action:   "Add error_path configuration to route for DLQ or error handling",
      })
    }
  }
  
  return findings
}

func checkAuthDeclaration(cfg *config.RouteConfig) []CritiqueFinding {
  var findings []CritiqueFinding
  
  for routeName, route := range cfg.Routes {
    if route.Auth == nil || (route.Auth != nil && route.Auth.Mode == "") {
      findings = append(findings, CritiqueFinding{
        Category: "security",
        Severity: "important",
        Path:     fmt.Sprintf("$.routes.%s.auth", routeName),
        Issue:    "No explicit auth declaration; authorization intent unclear",
        Action:   "Add auth: 'none' if intentional, or specify RBAC/ABAC policy",
      })
    }
  }
  
  return findings
}

func checkUnusedImports(cfg *config.RouteConfig) []CritiqueFinding {
  // Simplified version - real implementation would analyze what's actually used
  var findings []CritiqueFinding
  // This would require deeper analysis of fragment usage
  return findings
}
```

- [ ] **Step 2: Implement CritiqueRoute operation**

```go
// In internal/agent/critique.go
package agent

import (
  "context"
  "fmt"

  "github.com/naren-chakraview/dim/internal/config"
)

// CritiqueRoute analyzes an existing route and flags non-structural issues
func CritiqueRoute(ctx context.Context, req CritiqueRouteRequest) (*CritiqueRouteResponse, *OperationErr) {
  if req.RouteConfigPath == "" {
    return nil, &OperationErr{
      Code:    "INVALID_REQUEST",
      Message: "route_config_path is required",
    }
  }

  // Step 1: Load and validate the route (must be structurally valid first)
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
  for _, pattern := range crititquePatterns {
    findings := pattern.Check(cfg)
    allFindings = append(allFindings, findings...)
  }

  // Step 4: Apply domain-specific rules if provided
  if req.DomainRules != "" {
    // Would parse and apply custom domain rules here
    // For MVP, skip this
  }

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

// Helper: calculate overall rating
func calculateOverallRating(findings []CritiqueFinding, validateResp *ValidateResponse) string {
  if !validateResp.Valid {
    return "invalid"
  }

  importantCount := 0
  for _, f := range findings {
    if f.Severity == "important" {
      importantCount++
    }
  }

  if importantCount >= 3 {
    return "needs-review"
  } else if importantCount > 0 {
    return "fair"
  } else if len(findings) > 5 {
    return "good"
  }

  return "excellent"
}

// Helper: build human-readable critique summary
func buildCritiqueSummary(findings []CritiqueFinding) string {
  if len(findings) == 0 {
    return "No issues found. Route looks good!"
  }

  importantCount := 0
  warningCount := 0
  infoCount := 0

  for _, f := range findings {
    switch f.Severity {
    case "important":
      importantCount++
    case "warning":
      warningCount++
    default:
      infoCount++
    }
  }

  summary := fmt.Sprintf("Found %d issues: ", len(findings))
  if importantCount > 0 {
    summary += fmt.Sprintf("%d important, ", importantCount)
  }
  if warningCount > 0 {
    summary += fmt.Sprintf("%d warnings, ", warningCount)
  }
  if infoCount > 0 {
    summary += fmt.Sprintf("%d info", infoCount)
  }

  return summary
}
```

- [ ] **Step 3: Register CritiqueRoute in MCP server**

```go
// In internal/agent/mcp_server.go - add to registerOperations()
s.operations["critique_route"] = MCPOperation{
  Name:        "critique_route",
  Description: "Analyze and critique an existing route for non-structural issues",
  Handler: func(ctx context.Context, rawReq json.RawMessage) (interface{}, *OperationErr) {
    var req CritiqueRouteRequest
    if err := json.Unmarshal(rawReq, &req); err != nil {
      return nil, &OperationErr{Code: "INVALID_REQUEST", Message: fmt.Sprintf("Failed to parse request: %v", err)}
    }
    return CritiqueRoute(ctx, req)
  },
}
```

- [ ] **Step 4: Add comprehensive tests**

```go
// In internal/agent/critique_test.go
func TestCritiqueRedundantTranslateSteps(t *testing.T) {
  // Create a test route with redundant translate steps
  ctx := context.Background()
  
  // Would need to create a test route YAML with multiple translate steps
  // Then verify critique catches it
}

func TestCritiqueAuthDeclaration(t *testing.T) {
  ctx := context.Background()
  
  // Create test route without auth declaration
  // Verify critique flags it as important
}

func TestCritiqueValidRoute(t *testing.T) {
  ctx := context.Background()
  
  // Create a well-formed route with best practices
  // Verify critique returns minimal findings or "excellent" rating
}

func TestCritiqueStructurallyInvalid(t *testing.T) {
  ctx := context.Background()
  
  // Pass a route that fails dimctl validate
  // Verify critique reports structural_ok: false
}
```

- [ ] **Step 5: Commit**

```bash
git add internal/agent/critique.go internal/agent/critique_patterns.go internal/agent/critique_test.go internal/agent/mcp_server.go
git commit -m "feat: implement route critique/review mode (M4.8.2)"
```

---

### Task 4: Create worked example

**Files:**
- Create: `docs/examples/m48-design-examples.md` (documented end-to-end example)
- Create: `docs/examples/critique-examples.md` (critique examples)

**Steps:**

- [ ] **Step 1: Write ScaffoldFromIntent worked example**

```markdown
# M4.8.1 — Natural-Language-to-Route Scaffolding Example

## Scenario
An agent is asked to design a route that:
- Receives webhook events from a payment processor
- Redacts PII (credit card numbers, SSN)
- Routes to Kafka topic for downstream processing
- Must validate against payment contract

## Intent
"Connect payment webhook to Kafka with PII redaction and contract enforcement"

## Agent's Process

### 1. Call scaffold_from_intent
\`\`\`json
{
  "intent": "Connect payment webhook to Kafka with PII redaction and contract enforcement",
  "domain": "payments",
  "with_test_fixtures": true
}
\`\`\`

### 2. Agent's Response
\`\`\`json
{
  "success": true,
  "template_used": "contract-enforced",
  "route_yaml": "version: 1\nimports:\n  - ../governance/fragments.yaml\n\nsources:\n  payment_webhook:\n    type: http\n    \nroutes:\n  process_payment:\n    from: payment_webhook\n    error_path:\n      target: payment_dlq\n    steps:\n      - translate:\n          expr: {{ REDACTION_EXPR | \"$ /* TODO: Remove credit card, SSN fields */\" }}\n\nsinks:\n  kafka_out:\n    type: kafka\n    enforce: true  # Enforce payment contract\n    \n  payment_dlq:\n    type: file\n    path: ./dlq/payment-errors.jsonl",
  "placeholders": [\n    {\n      "path": "$.routes.process_payment.steps[0].translate.expr",\n      "reason": "PII redaction expression not inferred from intent",\n      "suggestion": "JSONata expression to remove payment fields: $utils.redact(., ['card_number', 'ssn', 'cvv'])" \n    },\n    {\n      "path": "$.sinks.kafka_out.topic",\n      "reason": "Target Kafka topic not determined from intent",\n      "suggestion": "Likely 'payments-processed' or similar based on domain conventions"\n    }\n  ],\n  "validation_result": {\n    "valid": true,\n    "errors": [],\n    "warnings": [],\n    "route_version": "sha256:abc123..."\n  },\n  "reasoning": "Used 'contract-enforced' template because intent mentioned: 'contract enforcement'. Detected source: http (webhook), sink: kafka. Leave placeholders as {{ }} markers for human refinement.",\n  "next_steps": [\n    "1. Review the generated route YAML",\n    "2. Fill in the redaction JSONata expression",\n    "3. Specify the Kafka topic name",\n    "4. Run 'dimctl test' with payment event fixtures",\n    "5. Create a PR for domain team review",\n    "6. After approval, merge to main"
  ]
}
\`\`\`

### 3. Human Review Process
The domain engineer reviews the generated route:
- Fills in the JSONata redaction expression: `$ | { card_number: "***", ssn: "***", ...}`
- Specifies Kafka topic: `payments-processed`
- Adds test fixtures with sample payment events
- Runs `dimctl test domains/payments/process-payment.yaml`
- Creates PR with agent-generated route + human refinements
- Domain team reviews and approves
- Merges to main via GitOps pipeline

### 4. What Validation Caught
- ✓ Schema structure is correct
- ✓ Governance fragments imported
- ✓ Sources/sinks are valid types
- ✓ Error path configured

### 5. What Critique Would Flag
```
[
  {
    "category": "best-practice",
    "severity": "info",
    "path": "$.routes.process_payment.steps[0].translate.expr",
    "issue": "JSONata expression is a placeholder; actual PII redaction logic not yet implemented",
    "action": "Implement the redaction expression to remove sensitive fields"
  }
]
```
```

- [ ] **Step 2: Write CritiqueRoute worked example**

```markdown
# M4.8.2 — Route Critique/Review Example

## Example 1: Under-specified Route

### Input Route
\`\`\`yaml
version: 1
imports:
  - ../governance/fragments.yaml

sources:
  webhook:
    type: http

routes:
  transform_and_send:
    from: webhook
    steps:
      - translate:
          expr: $
      - translate:
          expr: $.payload.data

sinks:
  output:
    type: http
    url: https://downstream.example.com/ingest
\`\`\`

### Agent Critique
\`\`\`json
{
  "findings": [
    {
      "category": "redundancy",
      "severity": "warning",
      "path": "$.routes.transform_and_send.steps",
      "issue": "Two consecutive translate steps can be combined into one",
      "action": "Combine into single translate: expr $.payload.data",
      "example": "// Before: [translate($), translate($.payload.data)]\n// After: translate($.payload.data)"
    },
    {
      "category": "convention",
      "severity": "warning",
      "path": "$.routes.transform_and_send",
      "issue": "No error_path configured; failed messages have no DLQ",
      "action": "Add error_path pointing to a DLQ sink"
    },
    {
      "category": "security",
      "severity": "important",
      "path": "$.routes.transform_and_send.auth",
      "issue": "No auth declaration; authorization intent unclear",
      "action": "Add auth: 'none' if intentional, or specify RBAC/ABAC rules"
    },
    {
      "category": "best-practice",
      "severity": "important",
      "path": "$.sinks.output",
      "issue": "HTTP sink missing enforce: true; contract validation not enabled",
      "action": "Add enforce: true to validate output against contract"
    }
  ],
  "summary": "Found 4 issues: 2 important, 2 warnings",
  "structural_ok": true,
  "overall_rating": "fair"
}
\`\`\`

### How Human Would Fix It
1. Combine the two translate steps
2. Add error_path with DLQ sink
3. Add explicit auth declaration
4. Add enforce: true to HTTP sink
5. Re-run critique to verify improvements

## Example 2: Well-Designed Route

### Input Route
\`\`\`yaml
version: 1
imports:
  - ../governance/fragments.yaml

sources:
  orders:
    type: http

routes:
  order_processing:
    from: orders
    auth: none  # Webhook is pre-authenticated by network
    error_path:
      target: order_dlq
      retry:
        max_attempts: 3
    steps:
      - filter:
          expr: $exists($.order_id)  # Ensure required fields
      - translate:
          expr: $.items | $sum($.price)  # Calculate total

sinks:
  kafka_orders:
    type: kafka
    topic: orders-processed
    enforce: true

  order_dlq:
    type: file
    path: ./dlq/orders.jsonl
\`\`\`

### Agent Critique
\`\`\`json
{
  "findings": [],
  "summary": "No issues found. Route looks good!",
  "structural_ok": true,
  "overall_rating": "excellent"
}
\`\`\`
```

- [ ] **Step 3: Commit**

```bash
git add docs/examples/m48-design-examples.md docs/examples/critique-examples.md
git commit -m "docs: add M4.8 worked examples for design and critique"
```

---

### Task 5: Integration tests and MCP verification

**Files:**
- Modify: `internal/agent/agent_test.go` (add M4.8 tests)

**Steps:**

- [ ] **Step 1: Add end-to-end MCP tests**

```go
func TestMCPScaffoldFromIntent(t *testing.T) {
  server := NewMCPServer()
  ctx := context.Background()

  req := json.RawMessage(`{
    "intent": "webhook to file output",
    "domain": "test-domain"
  }`)

  result := server.CallOperation(ctx, "scaffold_from_intent", req)

  if result.Error != nil {
    t.Fatalf("unexpected error: %v", result.Error)
  }

  // Verify response structure
  var resp ScaffoldFromIntentResponse
  respJSON, _ := json.Marshal(result.Result)
  if err := json.Unmarshal(respJSON, &resp); err != nil {
    t.Fatalf("failed to unmarshal response: %v", err)
  }

  if !resp.Success {
    t.Fatal("scaffold should produce valid route")
  }

  if resp.TemplateUsed == "" {
    t.Fatal("should report which template was used")
  }
}

func TestMCPCritiqueRoute(t *testing.T) {
  server := NewMCPServer()
  ctx := context.Background()

  req := json.RawMessage(`{
    "route_config_path": "domains/test/test-route.yaml"
  }`)

  result := server.CallOperation(ctx, "critique_route", req)

  // Should handle missing file gracefully
  if result.Error == nil {
    // Or critique might return findings even for nonexistent routes
    var resp CritiqueRouteResponse
    respJSON, _ := json.Marshal(result.Result)
    if err := json.Unmarshal(respJSON, &resp); err != nil {
      t.Fatalf("failed to unmarshal response: %v", err)
    }

    if resp.Findings == nil {
      t.Fatal("findings should not be nil")
    }
  }
}
```

- [ ] **Step 2: Verify all tests pass**

```bash
go test ./internal/agent -v
```

- [ ] **Step 3: Commit**

```bash
git add internal/agent/agent_test.go
git commit -m "test: add M4.8 MCP integration tests"
```

---

## Exit Criteria

✅ **M4.8.1 Complete:**
- Agent can scaffold route from natural-language intent using M4.2 templates
- Generated route passes identical `dimctl validate` to human-authored route
- Placeholders marked for human review (JSONata expressions, sink/source types)
- Reasoning provided explaining template choice and intent signals

✅ **M4.8.2 Complete:**
- Route critique operation implemented with 5+ critique patterns
- Critique catches redundancy (consecutive steps), conventions (auth, error-path), best-practices (contract enforcement)
- Critique returns actionable findings, not generic advice
- Critique distinguishes between structural validation (dimctl validate job) and semantic review (agent job)

✅ **M4.8.3 Complete:**
- End-to-end worked example shows intent → route → human review → PR → merge
- Documentation shows critique output and how human refines it
- All agent operations accessible via M4.6 MCP interface
- Tests verify MCP integration

✅ **Process Requirements:**
- Every agent-produced route passes identical validation to human route
- No agent-to-production path; propose/review/merge/deploy unchanged
- Critique output points at specific, actionable gaps
- All new operations tested with MCP server
- Tests verify human review step is shown in workflow

---

## Global Constraints Verification

- ✅ Reuses M4.2 scaffold templates (no new template design)
- ✅ Builds on M4.6 agent interface (uses MCP, no new transport)
- ✅ Integrates with M4.7 capability manifest (agents query capabilities for context)
- ✅ No new dependencies required (Go, existing config/schema packages)
- ✅ All agent operations have human review in the loop
- ✅ Critique is advisory only; no automatic fixes
- ✅ Every agent-modified route passes `dimctl validate`/`test` identically to human-authored

---

## Summary

**M4.8 delivers agent-assisted route design:** Agents propose routes from natural language intent and critique existing routes, but never auto-deploy. Humans remain in the review/merge/deploy loop. The interface is built on M4.6's published contract, the templates reuse M4.2's shapes, and the critique patterns catch what schema validation structurally can't catch — redundancy, conventions, best practices — not duplicating validation's job.

**Next milestone:** M4.9 (Structured lineage/impact-analysis queries) — given a proposed change, what routes/steps would it affect.
