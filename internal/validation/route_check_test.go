package validation

import (
	"testing"

	"github.com/naren-chakraview/dim/internal/config"
)

func TestPerformStaticContractChecks(t *testing.T) {
	tests := []struct {
		name       string
		route      *config.RouteSpec
		contracts  []config.ContractSpec
		wantWarnings int
		description string
	}{
		{
			name: "translate before contract with type mismatch",
			route: &config.RouteSpec{
				From: "input",
				Steps: []config.StepSpec{
					{
						Translate: &config.TranslateSpec{
							Expr: `{ "amount": "not-a-number" }`, // String literal, but contract expects number
						},
					},
					{
						Contract: &config.ContractStepSpec{
							ID: "order-contract",
						},
					},
				},
			},
			contracts: []config.ContractSpec{
				{
					ID:      "order-contract",
					Version: "1.0.0",
					Schema: map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"amount": map[string]interface{}{
								"type": "number",
							},
						},
						"required": []interface{}{"amount"},
					},
				},
			},
			wantWarnings: 1,
			description: "Should warn when translate emits wrong type",
		},
		{
			name: "translate before contract with correct type",
			route: &config.RouteSpec{
				From: "input",
				Steps: []config.StepSpec{
					{
						Translate: &config.TranslateSpec{
							Expr: `{ "amount": 42, "id": "order-123" }`, // Correct types
						},
					},
					{
						Contract: &config.ContractStepSpec{
							ID: "order-contract",
						},
					},
				},
			},
			contracts: []config.ContractSpec{
				{
					ID:      "order-contract",
					Version: "1.0.0",
					Schema: map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"amount": map[string]interface{}{
								"type": "number",
							},
							"id": map[string]interface{}{
								"type": "string",
							},
						},
						"required": []interface{}{"amount", "id"},
					},
				},
			},
			wantWarnings: 0,
			description: "Should not warn when types match",
		},
		{
			name: "complex JSONata expression",
			route: &config.RouteSpec{
				From: "input",
				Steps: []config.StepSpec{
					{
						Translate: &config.TranslateSpec{
							Expr: `$map(items, function($item) { $item.price * $item.qty })`, // Complex, outside analyzable subset
						},
					},
					{
						Contract: &config.ContractStepSpec{
							ID: "order-contract",
						},
					},
				},
			},
			contracts: []config.ContractSpec{
				{
					ID:      "order-contract",
					Version: "1.0.0",
					Schema: map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"total": map[string]interface{}{
								"type": "number",
							},
						},
					},
				},
			},
			wantWarnings: 0,
			description: "Should skip complex expressions (best-effort, partial validation)",
		},
		{
			name: "contract step without prior translate",
			route: &config.RouteSpec{
				From: "input",
				Steps: []config.StepSpec{
					{
						Filter: &config.FilterSpec{
							Expr: "body.id != null",
						},
					},
					{
						Contract: &config.ContractStepSpec{
							ID: "order-contract",
						},
					},
				},
			},
			contracts: []config.ContractSpec{
				{
					ID:      "order-contract",
					Version: "1.0.0",
					Schema: map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"id": map[string]interface{}{
								"type": "string",
							},
						},
					},
				},
			},
			wantWarnings: 0,
			description: "Should skip contract validation when no prior translate",
		},
		{
			name: "no contract step",
			route: &config.RouteSpec{
				From: "input",
				Steps: []config.StepSpec{
					{
						Translate: &config.TranslateSpec{
							Expr: `{ "amount": "wrong" }`,
						},
					},
				},
			},
			contracts: []config.ContractSpec{
				{
					ID:      "order-contract",
					Version: "1.0.0",
					Schema: map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"amount": map[string]interface{}{
								"type": "number",
							},
						},
					},
				},
			},
			wantWarnings: 0,
			description: "Should not warn when there's no contract step",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.RouteConfig{
				Version: 1,
				Routes: map[string]config.RouteSpec{
					"test-route": {
						From:      tt.route.From,
						Steps:     tt.route.Steps,
						Contracts: tt.contracts,
					},
				},
			}

			warnings := PerformStaticContractChecks(cfg)

			if len(warnings) != tt.wantWarnings {
				t.Errorf("Expected %d warnings, got %d. Description: %s", tt.wantWarnings, len(warnings), tt.description)
				for i, w := range warnings {
					t.Logf("  Warning %d: %s", i+1, w)
				}
			}
		})
	}
}

func TestContractSpecToFlatFieldTypes(t *testing.T) {
	tests := []struct {
		name      string
		contract  *config.ContractSpec
		wantError bool
		wantFields map[string]bool // field names that should exist
	}{
		{
			name: "basic schema",
			contract: &config.ContractSpec{
				ID: "test-contract",
				Schema: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"id": map[string]interface{}{
							"type": "string",
						},
						"amount": map[string]interface{}{
							"type": "number",
						},
					},
					"required": []interface{}{"id"},
				},
			},
			wantError: false,
			wantFields: map[string]bool{
				"id":     true,
				"amount": true,
			},
		},
		{
			name: "schema as JSON string",
			contract: &config.ContractSpec{
				ID: "test-contract",
				Schema: `{
					"type": "object",
					"properties": {
						"name": {"type": "string"}
					}
				}`,
			},
			wantError: false,
			wantFields: map[string]bool{
				"name": true,
			},
		},
		{
			name: "nil schema",
			contract: &config.ContractSpec{
				ID:     "test-contract",
				Schema: nil,
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := contractSpecToFlatFieldTypes(tt.contract)

			if (err != nil) != tt.wantError {
				t.Errorf("Expected error: %v, got: %v", tt.wantError, err)
			}

			if tt.wantError {
				return
			}

			for fieldName := range tt.wantFields {
				if _, ok := result[fieldName]; !ok {
					t.Errorf("Expected field %q in result", fieldName)
				}
			}
		})
	}
}
