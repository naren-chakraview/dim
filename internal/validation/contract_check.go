package validation

import (
	"fmt"
	"regexp"
	"strings"
)

// ContractChecker provides static validation of JSONata transformations against data contracts.
// It detects whether a JSONata expression is statically analyzable and, if so,
// checks whether the output shape matches the target contract. (M2.7.2)
type ContractChecker struct{}

// NewContractChecker creates a new contract checker.
func NewContractChecker() *ContractChecker {
	return &ContractChecker{}
}

// CheckResult holds the outcome of a static contract check.
type CheckResult struct {
	Analyzable bool     // Whether the expression is statically analyzable
	Errors     []string // Contract mismatch errors
	Warnings   []string // Non-critical issues (e.g., extra fields)
	FieldTypes map[string]string
}

// Check validates a JSONata transform expression against a sink contract.
// Returns CheckResult with Analyzable=false if the expression is outside the static subset.
// If Analyzable=true but there are errors, the contract is violated.
func (cc *ContractChecker) Check(jsonataExpr string, sinkContract map[string]interface{}) CheckResult {
	result := CheckResult{
		FieldTypes: make(map[string]string),
		Errors:     []string{},
		Warnings:   []string{},
	}

	// Step 1: Check if expression is statically analyzable (M2.7.2)
	analyzable, fieldTypes := cc.isAnalyzable(jsonataExpr)
	if !analyzable {
		result.Analyzable = false
		return result
	}

	result.Analyzable = true
	result.FieldTypes = fieldTypes

	// Step 2: Compare output shape to contract (M2.7.2)
	cc.compareToContract(fieldTypes, sinkContract, &result)

	return result
}

// isAnalyzable determines if a JSONata expression is statically analyzable.
// Returns true only if expression matches the simple object/property/literal subset.
func (cc *ContractChecker) isAnalyzable(expr string) (bool, map[string]string) {
	expr = strings.TrimSpace(expr)
	fieldTypes := make(map[string]string)

	// Check for disallowed patterns (function calls, loops, variables, conditionals)
	disallowedPatterns := []string{
		`\$\w+`,              // Variables: $var
		`\(\s*\$`,            // Function calls: ... ($ ...
		`\bfor\b`,            // for loops
		`\bforeach\b`,        // foreach loops
		`\.\s*\(`,            // Map/foreach syntax: .(...) or .(...)
		`\(\s*\w+\s*\)`,      // Function calls: name(...)
		`#\w+\s*\(`,          // Functions: #name(...)
		`\$map\b`,            // $map function
		`\$filter\b`,         // $filter function
		`\$reduce\b`,         // $reduce function
		`regex\s*\(`,         // regex function
	}

	for _, pattern := range disallowedPatterns {
		if matched, _ := regexp.MatchString(pattern, expr); matched {
			return false, fieldTypes
		}
	}

	// Must start with { (object literal)
	if !strings.HasPrefix(strings.TrimSpace(expr), "{") {
		return false, fieldTypes
	}

	// Try to extract object fields - simple regex-based approach
	// Look for pattern: "key": value or key: value
	// Note: Go regex doesn't support backreferences, so we use two patterns
	fieldPattern := regexp.MustCompile(`"(\w+)"\s*:\s*([^,}]+)|'(\w+)'\s*:\s*([^,}]+)|(\w+)\s*:\s*([^,}]+)`)
	matches := fieldPattern.FindAllStringSubmatch(expr, -1)

	for _, match := range matches {
		var fieldName, valueExpr string
		// match groups: full match, "key", value, 'key', value, key, value
		if match[1] != "" {
			// "key": value
			fieldName = match[1]
			valueExpr = strings.TrimSpace(match[2])
		} else if match[3] != "" {
			// 'key': value
			fieldName = match[3]
			valueExpr = strings.TrimSpace(match[4])
		} else {
			// key: value
			fieldName = match[5]
			valueExpr = strings.TrimSpace(match[6])
		}

		if fieldName != "" {
			// Infer type from value expression
			fieldType := cc.inferType(valueExpr)
			fieldTypes[fieldName] = fieldType
		}
	}

	// If we extracted no fields, it's not a valid analyzable object
	if len(fieldTypes) == 0 {
		return false, fieldTypes
	}

	return true, fieldTypes
}

// inferType infers the type of a JSONata value expression.
func (cc *ContractChecker) inferType(valueExpr string) string {
	valueExpr = strings.TrimSpace(valueExpr)

	// String literals
	if (strings.HasPrefix(valueExpr, `"`) && strings.HasSuffix(valueExpr, `"`)) ||
		(strings.HasPrefix(valueExpr, `'`) && strings.HasSuffix(valueExpr, `'`)) {
		return "string"
	}

	// Number literals
	if isNumber(valueExpr) {
		return "number"
	}

	// Boolean literals
	if valueExpr == "true" || valueExpr == "false" {
		return "boolean"
	}

	// Null
	if valueExpr == "null" {
		return "null"
	}

	// Array literals
	if strings.HasPrefix(valueExpr, "[") && strings.HasSuffix(valueExpr, "]") {
		return "array"
	}

	// Ternary: expr ? true_val : false_val - return unknown since we can't know
	if strings.Contains(valueExpr, "?") && strings.Contains(valueExpr, ":") {
		return "unknown"
	}

	// Arithmetic: division and other binary ops
	// Only if it looks like "something / something_else"
	if matched, _ := regexp.MatchString(`\w+\s*/\s*\w+`, valueExpr); matched {
		return "number" // division always produces float
	}

	// String concatenation with & operator
	if matched, _ := regexp.MatchString(`["']\s*&|&\s*["']|\w+\s*&\s*\w+`, valueExpr); matched {
		return "string"
	}

	// Property access (body.field, headers.x, etc.) - default unknown
	return "unknown"
}

// isNumber checks if a string is a numeric literal.
func isNumber(s string) bool {
	s = strings.TrimSpace(s)
	if len(s) == 0 {
		return false
	}
	// Simple check: starts with digit, maybe has decimal and/or minus
	if s[0] == '-' {
		s = s[1:]
	}
	if len(s) == 0 {
		return false
	}
	// Check if it's all digits with at most one decimal point
	parts := strings.Split(s, ".")
	if len(parts) > 2 {
		return false
	}
	for _, part := range parts {
		if part == "" {
			return false
		}
		for _, ch := range part {
			if ch < '0' || ch > '9' {
				return false
			}
		}
	}
	return true
}

// compareToContract checks output fields against sink contract.
func (cc *ContractChecker) compareToContract(fieldTypes map[string]string, sinkContract map[string]interface{}, result *CheckResult) {
	// Extract contract fields and their types
	contractFields := make(map[string]string)
	requiredFields := make(map[string]bool)

	for fieldName, fieldDef := range sinkContract {
		switch def := fieldDef.(type) {
		case string:
			contractFields[fieldName] = def
			requiredFields[fieldName] = true
		case map[string]interface{}:
			if typeVal, ok := def["type"].(string); ok {
				contractFields[fieldName] = typeVal
			}
			if reqVal, ok := def["required"].(bool); ok {
				requiredFields[fieldName] = reqVal
			} else {
				requiredFields[fieldName] = true // default to required
			}
		}
	}

	// Check for missing required fields
	for fieldName := range contractFields {
		if requiredFields[fieldName] {
			if _, exists := fieldTypes[fieldName]; !exists {
				result.Errors = append(result.Errors,
					fmt.Sprintf("missing required field: %q", fieldName))
			}
		}
	}

	// Check for type mismatches
	for fieldName, outputType := range fieldTypes {
		if expectedType, ok := contractFields[fieldName]; ok {
			if outputType != "unknown" && expectedType != "unknown" && outputType != expectedType {
				result.Errors = append(result.Errors,
					fmt.Sprintf("field %q: expected %s, got %s", fieldName, expectedType, outputType))
			}
		} else {
			// Extra field not in contract
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("field %q not in contract (may be ignored by sink)", fieldName))
		}
	}
}
