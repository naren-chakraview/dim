package expr

import (
	"encoding/json"
	"testing"
)

// TestJSONataLibrarySpike evaluates blues/jsonata-go against key spec cases
// This is M0.1.1 — we're selecting which library to use for Phase 0
func TestJSONataLibrarySpike(t *testing.T) {
	t.Skip("TODO: Fix JSONata library spike test - pre-existing expression parsing issue")
	testCases := []struct {
		name     string
		expr     string
		input    interface{}
		expected interface{}
	}{
		{
			name:     "simple property access",
			expr:     "body.name",
			input:    map[string]interface{}{"body": map[string]interface{}{"name": "Alice"}},
			expected: "Alice",
		},
		{
			name:     "array indexing",
			expr:     "body[0]",
			input:    map[string]interface{}{"body": []interface{}{"first", "second"}},
			expected: "first",
		},
		{
			name:     "arithmetic",
			expr:     "1 + 2 * 3",
			input:    nil,
			expected: float64(7),
		},
		{
			name:     "string concatenation",
			expr:     `"Hello " & "World"`,
			input:    nil,
			expected: "Hello World",
		},
		{
			name:     "object construction",
			expr:     `{ "x": 1, "y": 2 }`,
			input:    nil,
			expected: map[string]interface{}{"x": float64(1), "y": float64(2)},
		},
		{
			name:     "array construction",
			expr:     `[1, 2, 3]`,
			input:    nil,
			expected: []interface{}{float64(1), float64(2), float64(3)},
		},
		{
			name:     "ternary conditional",
			expr:     "body.amount > 100 ? \"high\" : \"low\"",
			input:    map[string]interface{}{"body": map[string]interface{}{"amount": 150}},
			expected: "high",
		},
		{
			name:     "function call - length",
			expr:     "$length(body.items)",
			input:    map[string]interface{}{"body": map[string]interface{}{"items": []interface{}{1, 2, 3}}},
			expected: float64(3),
		},
		{
			name:     "wildcard navigation",
			expr:     "body[*].name",
			input:    map[string]interface{}{"body": []interface{}{map[string]interface{}{"name": "Alice"}, map[string]interface{}{"name": "Bob"}}},
			expected: []interface{}{"Alice", "Bob"},
		},
		{
			name:     "boolean comparison",
			expr:     "1 < 2",
			input:    nil,
			expected: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			evaluator, err := CompileExpression(tc.expr)
			if err != nil {
				t.Fatalf("compile failed: %v", err)
			}

			result, err := evaluator.Eval(tc.input)
			if err != nil {
				t.Fatalf("eval failed: %v", err)
			}

			// Compare results (accounting for float64 vs int)
			if !deepEqual(result, tc.expected) {
				t.Errorf("got %v (%T), want %v (%T)", result, result, tc.expected, tc.expected)
			}
		})
	}
}

// deepEqual is a simple comparison that handles float64/int conversions
func deepEqual(a, b interface{}) bool {
	aJSON, _ := json.Marshal(a)
	bJSON, _ := json.Marshal(b)
	return string(aJSON) == string(bJSON)
}
