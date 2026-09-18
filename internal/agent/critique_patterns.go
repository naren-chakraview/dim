package agent

import (
	"fmt"
	"strings"

	"github.com/naren-chakraview/dim/internal/config"
)

// CritiquePattern defines a rule the critique engine checks
type CritiquePattern struct {
	Name        string
	Category    string // "redundancy" | "convention" | "best-practice" | "security" | "clarity"
	Severity    string // "info" | "warning" | "important"
	Description string
	Check       func(*config.RouteConfig) []CritiqueFinding
}

var critiquPatterns = []CritiquePattern{
	{
		Name:        "redundant_translate_steps",
		Category:    "redundancy",
		Severity:    "warning",
		Description: "Multiple consecutive translate steps can usually be combined",
		Check:       checkRedundantTranslateSteps,
	},
	{
		Name:        "contract_without_enforce",
		Category:    "best-practice",
		Severity:    "important",
		Description: "Contract referenced but not strictly enforced",
		Check:       checkContractEnforcement,
	},
	{
		Name:        "missing_error_path",
		Category:    "convention",
		Severity:    "warning",
		Description: "Route missing error_path configuration",
		Check:       checkErrorPath,
	},
	{
		Name:        "missing_auth_declaration",
		Category:    "security",
		Severity:    "important",
		Description: "Route missing auth declaration (should be 'none' if intentional)",
		Check:       checkAuthDeclaration,
	},
	{
		Name:        "unused_imports",
		Category:    "clarity",
		Severity:    "info",
		Description: "Import statement that references no fragments actually used",
		Check:       checkUnusedImports,
	},
	{
		Name:        "missing_lineage_retention_policy",
		Category:    "best-practice",
		Severity:    "info",
		Description: "Route missing retention_policy for lineage tracking",
		Check:       checkLineageRetentionPolicy,
	},
}

// checkRedundantTranslateSteps detects multiple consecutive translate steps
func checkRedundantTranslateSteps(cfg *config.RouteConfig) []CritiqueFinding {
	var findings []CritiqueFinding

	// For each route, check if it has multiple translate steps
	for routeName, route := range cfg.Routes {
		translateCount := 0
		var lastTranslateIndices []int

		for i, step := range route.Steps {
			if step.Translate != nil {
				translateCount++
				lastTranslateIndices = append(lastTranslateIndices, i)
			}
		}

		// Check if we have consecutive translate steps
		if translateCount > 1 {
			hasConsecutive := false
			for i := 0; i < len(lastTranslateIndices)-1; i++ {
				if lastTranslateIndices[i+1] == lastTranslateIndices[i]+1 {
					hasConsecutive = true
					break
				}
			}

			if hasConsecutive {
				findings = append(findings, CritiqueFinding{
					Category: "redundancy",
					Severity: "warning",
					Path:     fmt.Sprintf("$.routes.%s.steps", routeName),
					Issue:    fmt.Sprintf("Found %d translate steps; consecutive ones can be combined", translateCount),
					Action:   "Combine multiple translate steps into a single expression for clarity and performance",
					Example:  "// Before: [translate(expr1), translate(expr2)]\n// After: translate(($ | expr1) | expr2)",
				})
			}
		}
	}

	return findings
}

// checkContractEnforcement detects routes with contracts not strictly enforced
func checkContractEnforcement(cfg *config.RouteConfig) []CritiqueFinding {
	var findings []CritiqueFinding

	// Check if routes reference contracts without strict enforcement in contract steps
	for routeName, route := range cfg.Routes {
		if len(route.Contracts) > 0 {
			// Route has contracts defined; check if any are enforced with strict: true
			hasStrictEnforcement := false
			for _, step := range route.Steps {
				if step.Contract != nil && step.Contract.Strict {
					hasStrictEnforcement = true
					break
				}
			}

			// If contracts exist but no contract step has strict: true, flag it
			if !hasStrictEnforcement {
				findings = append(findings, CritiqueFinding{
					Category: "best-practice",
					Severity: "important",
					Path:     fmt.Sprintf("$.routes.%s", routeName),
					Issue:    fmt.Sprintf("Route '%s' defines contracts but does not enforce them (no contract step with strict: true)", routeName),
					Action:   "Add a contract validation step with strict: true to enforce data contracts",
					Example:  "steps:\n  - contract:\n      id: payment-contract\n      strict: true  # Fail pipeline on violation",
				})
			}
		}
	}

	return findings
}

// checkErrorPath detects routes missing error_path configuration
func checkErrorPath(cfg *config.RouteConfig) []CritiqueFinding {
	var findings []CritiqueFinding

	for routeName, route := range cfg.Routes {
		if route.ErrorPath == nil {
			findings = append(findings, CritiqueFinding{
				Category: "convention",
				Severity: "warning",
				Path:     fmt.Sprintf("$.routes.%s", routeName),
				Issue:    "No error_path configured; failed messages have no dead-letter queue",
				Action:   "Add error_path configuration to route pointing to a DLQ sink for failure handling",
				Example:  "routes:\n  " + routeName + ":\n    error_path:\n      target: dlq_sink\n      retry:\n        max_attempts: 3",
			})
		}
	}

	return findings
}

// checkAuthDeclaration detects routes without explicit auth declaration
func checkAuthDeclaration(cfg *config.RouteConfig) []CritiqueFinding {
	var findings []CritiqueFinding

	for routeName, route := range cfg.Routes {
		// Check if auth is missing (nil is the zero value for interface{})
		if route.Auth == nil {
			findings = append(findings, CritiqueFinding{
				Category: "security",
				Severity: "important",
				Path:     fmt.Sprintf("$.routes.%s.auth", routeName),
				Issue:    "No explicit auth declaration; authorization intent is unclear",
				Action:   "Add auth: 'none' if intentional, or specify RBAC/ABAC policy for route authorization",
				Example:  "routes:\n  " + routeName + ":\n    auth: none  # Webhook is pre-authenticated by network",
			})
		}
	}

	return findings
}

// checkUnusedImports detects imports that might not be used
func checkUnusedImports(cfg *config.RouteConfig) []CritiqueFinding {
	var findings []CritiqueFinding

	// Check for unused imports
	if len(cfg.Imports) == 0 {
		return findings
	}

	// By the time we reach this check, fragments have been resolved and expanded into steps.
	// We can detect if imports were declared but resulted in no steps being added.
	// This is an indirect check: if imports exist but all routes have no steps or only
	// explicitly-defined steps (not from fragments), the imports may be unused.

	// Count total steps across all routes
	var totalSteps int
	for _, route := range cfg.Routes {
		totalSteps += len(route.Steps)
	}

	// If there are imports but no steps, that's suspicious
	if totalSteps == 0 && len(cfg.Imports) > 0 {
		for _, importPath := range cfg.Imports {
			findings = append(findings, CritiqueFinding{
				Category: "clarity",
				Severity: "info",
				Path:     "$.imports",
				Issue:    fmt.Sprintf("Unused import: '%s' (imported but no fragments referenced in routes)", importPath),
				Action:   "Remove this import if no routes reference its fragments",
				Example:  fmt.Sprintf("# Remove this line if not needed:\nimports:\n  # - %s", importPath),
			})
		}
	}

	return findings
}

// checkLineageRetentionPolicy detects routes missing retention_policy for lineage tracking
func checkLineageRetentionPolicy(cfg *config.RouteConfig) []CritiqueFinding {
	var findings []CritiqueFinding

	for routeName, route := range cfg.Routes {
		if route.RetentionPolicy == "" {
			findings = append(findings, CritiqueFinding{
				Category: "best-practice",
				Severity: "info",
				Path:     fmt.Sprintf("$.routes.%s", routeName),
				Issue:    fmt.Sprintf("Route '%s' missing retention_policy for lineage tracking", routeName),
				Action:   "Add retention_policy to route for data lineage and compliance tracking",
				Example:  "routes:\n  " + routeName + ":\n    retention_policy: 30d  # Keep lineage for 30 days",
			})
		}
	}

	return findings
}

// calculateOverallRating calculates a route's overall quality rating
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

// buildCritiqueSummary creates a human-readable summary of findings
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

	parts := []string{}
	if importantCount > 0 {
		parts = append(parts, fmt.Sprintf("%d important", importantCount))
	}
	if warningCount > 0 {
		parts = append(parts, fmt.Sprintf("%d warnings", warningCount))
	}
	if infoCount > 0 {
		parts = append(parts, fmt.Sprintf("%d info", infoCount))
	}

	return fmt.Sprintf("Found %d issues: %s", len(findings), strings.Join(parts, ", "))
}
