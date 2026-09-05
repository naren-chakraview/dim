package config

import (
	"encoding/json"
	"testing"
)

// TestContractToSchemaDatasetFacet verifies automatic schema facet generation (M1.7.3, R20)
func TestContractToSchemaDatasetFacet(t *testing.T) {
	contract := &ContractSpec{
		ID:      "order-contract",
		Version: "1.0.0",
		Schema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"order_id": map[string]interface{}{
					"type":        "string",
					"description": "Unique order identifier",
				},
				"customer_name": map[string]interface{}{
					"type": "string",
				},
				"total": map[string]interface{}{
					"type": "number",
				},
				"currency": map[string]interface{}{
					"type": "string",
				},
				"timestamp": map[string]interface{}{
					"type": "string",
				},
			},
			"required": []interface{}{"order_id", "customer_name", "total"},
		},
	}

	facet, err := contract.ToSchemaDatasetFacet()
	if err != nil {
		t.Fatalf("Failed to generate schema facet: %v", err)
	}

	if facet == nil {
		t.Fatalf("Expected non-nil facet")
	}

	// Verify field count
	if len(facet.Fields) != 5 {
		t.Errorf("Expected 5 fields, got %d", len(facet.Fields))
	}

	// Check required fields
	requiredFields := map[string]bool{
		"order_id":      true,
		"customer_name": true,
		"total":         true,
	}

	fieldMap := make(map[string]SchemaField)
	for _, f := range facet.Fields {
		fieldMap[f.Name] = f
	}

	// Verify order_id (required, has description)
	if f, ok := fieldMap["order_id"]; ok {
		if f.Type != "string" {
			t.Errorf("order_id: expected type string, got %s", f.Type)
		}
		if f.Nullable {
			t.Errorf("order_id: expected nullable=false")
		}
		if f.Description != "Unique order identifier" {
			t.Errorf("order_id: expected description 'Unique order identifier', got %q", f.Description)
		}
	} else {
		t.Errorf("order_id field not found in facet")
	}

	// Verify customer_name (required)
	if f, ok := fieldMap["customer_name"]; ok {
		if f.Nullable {
			t.Errorf("customer_name: expected nullable=false")
		}
	} else {
		t.Errorf("customer_name field not found in facet")
	}

	// Verify currency (optional)
	if f, ok := fieldMap["currency"]; ok {
		if !f.Nullable {
			t.Errorf("currency: expected nullable=true")
		}
	} else {
		t.Errorf("currency field not found in facet")
	}

	// Verify all required fields are marked non-nullable
	for _, f := range facet.Fields {
		if requiredFields[f.Name] && f.Nullable {
			t.Errorf("%s: required field should not be nullable", f.Name)
		}
	}
}

// TestContractToSchemaDatasetFacetWithStringSchema verifies facet generation from JSON string schema
func TestContractToSchemaDatasetFacetWithStringSchema(t *testing.T) {
	contract := &ContractSpec{
		ID:      "test-contract",
		Version: "1.0.0",
		Schema: `{
			"type": "object",
			"properties": {
				"id": {"type": "string"},
				"value": {"type": "integer"}
			},
			"required": ["id"]
		}`,
	}

	facet, err := contract.ToSchemaDatasetFacet()
	if err != nil {
		t.Fatalf("Failed to generate schema facet: %v", err)
	}

	if len(facet.Fields) != 2 {
		t.Errorf("Expected 2 fields, got %d", len(facet.Fields))
	}

	// Verify id is required
	for _, f := range facet.Fields {
		if f.Name == "id" && f.Nullable {
			t.Errorf("id should not be nullable (it's required)")
		}
		if f.Name == "value" && !f.Nullable {
			t.Errorf("value should be nullable (it's not required)")
		}
	}
}

// TestContractToSchemaDatasetFacetNilSchema verifies error handling for missing schema
func TestContractToSchemaDatasetFacetNilSchema(t *testing.T) {
	contract := &ContractSpec{
		ID:      "test",
		Version: "1.0.0",
		Schema:  nil,
	}

	facet, err := contract.ToSchemaDatasetFacet()
	if err == nil {
		t.Fatalf("Expected error for nil schema, got nil")
	}
	if facet != nil {
		t.Fatalf("Expected nil facet for nil schema")
	}
}

// TestSchemaDatasetFacetMarshal verifies facet can be marshaled to JSON (for OpenLineage)
func TestSchemaDatasetFacetMarshal(t *testing.T) {
	facet := &SchemaDatasetFacet{
		Fields: []SchemaField{
			{
				Name:        "id",
				Type:        "string",
				Description: "User ID",
				Nullable:    false,
			},
			{
				Name:     "email",
				Type:     "string",
				Nullable: true,
			},
		},
	}

	jsonBytes, err := json.Marshal(facet)
	if err != nil {
		t.Fatalf("Failed to marshal facet: %v", err)
	}

	if len(jsonBytes) == 0 {
		t.Fatalf("Expected non-empty JSON")
	}

	// Verify JSON structure by unmarshaling
	var unmarshaled SchemaDatasetFacet
	err = json.Unmarshal(jsonBytes, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal facet: %v", err)
	}

	if len(unmarshaled.Fields) != 2 {
		t.Errorf("Expected 2 fields after unmarshal, got %d", len(unmarshaled.Fields))
	}
}

// TestContractToSchemaDatasetFacetNoProperties verifies facet generation with empty schema
func TestContractToSchemaDatasetFacetNoProperties(t *testing.T) {
	contract := &ContractSpec{
		ID:      "empty-contract",
		Version: "1.0.0",
		Schema: map[string]interface{}{
			"type": "object",
		},
	}

	facet, err := contract.ToSchemaDatasetFacet()
	if err != nil {
		t.Fatalf("Failed to generate schema facet: %v", err)
	}

	if len(facet.Fields) != 0 {
		t.Errorf("Expected 0 fields for empty schema, got %d", len(facet.Fields))
	}
}
