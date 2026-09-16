package validation

import (
	"encoding/json"
	"fmt"

	"github.com/naren-chakraview/dim/internal/config"
)

// PerformStaticContractChecks runs static validation on translate steps followed by contract steps.
// It walks each route's steps, and whenever a translate step is followed by a contract validation step,
// it checks the JSONata expression against the contract schema for type mismatches.
// Returns a slice of formatted warning messages suitable for CLI/agent output.
// Per the mandatory caveat (M2.7.3), this is BEST-EFFORT and PARTIAL — complex JSONata,
// dynamic schemas, and uncertain references are silently skipped with no warning.
func PerformStaticContractChecks(cfg *config.RouteConfig) []string {
	var warnings []string
	checker := NewContractChecker()

	for routeName, route := range cfg.Routes {
		if len(route.Steps) == 0 {
			continue
		}

		// Build a map of contract ID -> ContractSpec for this route
		contractMap := make(map[string]*config.ContractSpec)
		for i := range route.Contracts {
			contractMap[route.Contracts[i].ID] = &route.Contracts[i]
		}

		// Walk steps; track the most recent translate step
		var lastTranslateExpr string
		for _, step := range route.Steps {
			// If this is a translate step, remember its expression
			if step.Translate != nil {
				lastTranslateExpr = step.Translate.Expr
			}

			// If this is a contract step, validate the prior translate against it
			if step.Contract != nil {
				if lastTranslateExpr == "" {
					// No translate step yet; skip silently (not an error; contract might not need a translate)
					continue
				}

				// Resolve the contract
				contract, ok := contractMap[step.Contract.ID]
				if !ok {
					// Contract not declared in this route; this should have been caught by schema validation
					// Skip silently per best-effort semantics
					continue
				}

				// Convert the contract schema to a flat field-type map
				flatSchema, err := contractSpecToFlatFieldTypes(contract)
				if err != nil {
					// Schema parsing error; skip silently per best-effort semantics
					continue
				}

				// Run the checker
				result := checker.Check(lastTranslateExpr, flatSchema)

				// Collect warnings/errors only if the expression was analyzable
				if !result.Analyzable {
					// Complex JSONata outside the simple-literal subset; skip
					continue
				}

				// Format and collect any real warnings/errors from the check
				for _, w := range result.Warnings {
					warnings = append(warnings, fmt.Sprintf("route %s: %s", routeName, w))
				}
				for _, e := range result.Errors {
					warnings = append(warnings, fmt.Sprintf("route %s: %s", routeName, e))
				}

				// Reset for the next contract step
				lastTranslateExpr = ""
			}
		}
	}

	return warnings
}

// contractSpecToFlatFieldTypes converts a ContractSpec's JSON Schema to a flat map[string]interface{}
// suitable for ContractChecker.Check(). Mirrors the pattern of ContractSpec.ToSchemaDatasetFacet()
// but returns the raw field-type info rather than an OpenLineage facet.
func contractSpecToFlatFieldTypes(cs *config.ContractSpec) (map[string]interface{}, error) {
	if cs.Schema == nil {
		return nil, fmt.Errorf("contract %q has no schema", cs.ID)
	}

	// Parse schema JSON (handles both YAML object and JSON string)
	var schemaObj map[string]interface{}
	switch v := cs.Schema.(type) {
	case string:
		if err := json.Unmarshal([]byte(v), &schemaObj); err != nil {
			return nil, fmt.Errorf("contract %q: invalid schema JSON: %w", cs.ID, err)
		}
	case map[string]interface{}:
		schemaObj = v
	default:
		schemaBytes, err := json.Marshal(v)
		if err != nil {
			return nil, fmt.Errorf("contract %q: failed to marshal schema: %w", cs.ID, err)
		}
		if err := json.Unmarshal(schemaBytes, &schemaObj); err != nil {
			return nil, fmt.Errorf("contract %q: invalid schema JSON: %w", cs.ID, err)
		}
	}

	// Extract field properties and required list
	result := make(map[string]interface{})

	if properties, ok := schemaObj["properties"].(map[string]interface{}); ok {
		requiredSet := make(map[string]bool)
		if reqList, ok := schemaObj["required"].([]interface{}); ok {
			for _, r := range reqList {
				if s, ok := r.(string); ok {
					requiredSet[s] = true
				}
			}
		}

		// For each property, extract its type info
		for fieldName, propVal := range properties {
			if propObj, ok := propVal.(map[string]interface{}); ok {
				// Store the field info as a nested map for ContractChecker.Check() to inspect
				fieldInfo := make(map[string]interface{})

				// Extract type
				if t, ok := propObj["type"].(string); ok {
					fieldInfo["type"] = t
				} else if typeArray, ok := propObj["type"].([]interface{}); ok && len(typeArray) > 0 {
					// Handle ["type", "null"] case; take the first non-null type
					if ts, ok := typeArray[0].(string); ok {
						fieldInfo["type"] = ts
					}
				}

				// Mark required status
				fieldInfo["required"] = requiredSet[fieldName]

				result[fieldName] = fieldInfo
			}
		}
	}

	return result, nil
}

// StaticCheckCaveat is the mandatory disclaimer text appended to validation output (M2.7.3).
// Shared between CLI and agent so both surfaces emit identical wording.
const StaticCheckCaveat = `
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
⚠️  STATIC CONTRACT CONFORMANCE CHECK CAVEAT (M2.7)
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

  This check is a BEST-EFFORT, PARTIAL validation only.

  • Only simple JSONata transformations can be analyzed (no functions, loops,
    conditionals, or dynamic features).

  • Passing this check does NOT guarantee runtime behavior — use enforce: true
    on the contract for a complete runtime guarantee.

  • This static check is NOT a substitute for the runtime enforce: true check.

  • See design documentation (M2.7) for details on the analyzable subset.

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
`
