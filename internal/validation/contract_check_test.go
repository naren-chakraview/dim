package validation

import (
	"testing"
)

func TestStaticAnalyzableSubset(t *testing.T) {
	tests := []struct {
		name       string
		expr       string
		analyzable bool
		fields     []string
	}{
		{
			name:       "Simple object construction",
			expr:       `{ "order_id": body.id, "total": body.amount }`,
			analyzable: true,
			fields:     []string{"order_id", "total"},
		},
		{
			name:       "With literals",
			expr:       `{ "status": "processed", "count": 42 }`,
			analyzable: true,
			fields:     []string{"status", "count"},
		},
		{
			name:       "With arithmetic",
			expr:       `{ "total": body.amount / 100 }`,
			analyzable: true,
			fields:     []string{"total"},
		},
		{
			name:       "Function call - not analyzable",
			expr:       `{ "result": $count(body.items) }`,
			analyzable: false,
			fields:     []string{},
		},
		{
			name:       "Map function - not analyzable",
			expr:       `{ "items": body.items.($map(this, .)) }`,
			analyzable: false,
			fields:     []string{},
		},
		{
			name:       "Foreach - not analyzable",
			expr:       `{ "fields": body.items.(name & id) }`,
			analyzable: false,
			fields:     []string{},
		},
		{
			name:       "Variable - not analyzable",
			expr:       `{ "value": $custom_var }`,
			analyzable: false,
			fields:     []string{},
		},
		{
			name:       "Ternary operator - analyzable",
			expr:       `{ "value": body.x ? body.y : 0 }`,
			analyzable: true,
			fields:     []string{"value"},
		},
	}

	checker := NewContractChecker()

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			analyzable, fields := checker.isAnalyzable(tc.expr)

			if analyzable != tc.analyzable {
				t.Errorf("isAnalyzable: expected %v, got %v", tc.analyzable, analyzable)
			}

			if analyzable {
				for _, field := range tc.fields {
					if _, ok := fields[field]; !ok {
						t.Errorf("expected field %q not found", field)
					}
				}
			}
		})
	}
}

func TestContractCheck_MatchingContract(t *testing.T) {
	checker := NewContractChecker()

	expr := `{ "order_id": body.id, "status": "new" }`
	contract := map[string]interface{}{
		"order_id": "number",
		"status":  "string",
	}

	result := checker.Check(expr, contract)

	if !result.Analyzable {
		t.Error("expression should be analyzable")
	}

	if len(result.Errors) > 0 {
		t.Errorf("unexpected errors: %v", result.Errors)
	}
}

func TestContractCheck_MissingRequiredField(t *testing.T) {
	checker := NewContractChecker()

	expr := `{ "order_id": body.id }`
	contract := map[string]interface{}{
		"order_id": "number",
		"status":  "string", // required
	}

	result := checker.Check(expr, contract)

	if !result.Analyzable {
		t.Error("expression should be analyzable")
	}

	if len(result.Errors) == 0 {
		t.Error("should detect missing required field 'status'")
	}
}

func TestContractCheck_TypeMismatch(t *testing.T) {
	checker := NewContractChecker()

	expr := `{ "order_id": "not-a-number" }`
	contract := map[string]interface{}{
		"order_id": "number",
	}

	result := checker.Check(expr, contract)

	if !result.Analyzable {
		t.Error("expression should be analyzable")
	}

	if len(result.Errors) == 0 {
		t.Error("should detect type mismatch")
	}
}

func TestContractCheck_NotAnalyzable(t *testing.T) {
	checker := NewContractChecker()

	// Complex JSONata that can't be statically analyzed
	expr := `{ "items": body.items.( { "name": name, "count": $count(purchases) } ) }`
	contract := map[string]interface{}{
		"items": "array",
	}

	result := checker.Check(expr, contract)

	if result.Analyzable {
		t.Error("expression should NOT be analyzable (contains function call)")
	}

	// Errors should be empty since we can't analyze it
	if len(result.Errors) > 0 {
		t.Errorf("should not report errors for non-analyzable expression, got: %v", result.Errors)
	}
}

func TestContractCheck_ExtraFields(t *testing.T) {
	checker := NewContractChecker()

	expr := `{ "order_id": body.id, "extra_field": "value" }`
	contract := map[string]interface{}{
		"order_id": "number",
	}

	result := checker.Check(expr, contract)

	if !result.Analyzable {
		t.Error("expression should be analyzable")
	}

	if len(result.Warnings) == 0 {
		t.Error("should warn about extra fields")
	}
}

func TestInferType(t *testing.T) {
	tests := []struct {
		expr     string
		expected string
	}{
		{`"hello"`, "string"},
		{`'world'`, "string"},
		{`42`, "number"},
		{`3.14`, "number"},
		{`true`, "boolean"},
		{`false`, "boolean"},
		{`null`, "null"},
		{`[1, 2, 3]`, "array"},
		{`body.field`, "unknown"},
		{`body.amount / 100`, "number"},
		{`"name: " & body.name`, "string"},
		{`body.x ? body.y : 0`, "unknown"},
	}

	checker := NewContractChecker()

	for _, tc := range tests {
		t.Run(tc.expr, func(t *testing.T) {
			typ := checker.inferType(tc.expr)
			if typ != tc.expected {
				t.Errorf("inferType: expected %q, got %q", tc.expected, typ)
			}
		})
	}
}
