package config

import (
	"encoding/json"
	"testing"
)

// TestNewContractStore verifies store initialization
func TestNewContractStore(t *testing.T) {
	store := NewContractStore()
	if store == nil {
		t.Fatal("NewContractStore returned nil")
	}
	if store.routeContracts == nil {
		t.Error("routeContracts map is nil")
	}
	if store.compiledSchemas == nil {
		t.Error("compiledSchemas map is nil")
	}
}

// TestLoadContractsEmpty verifies loading empty contract list
func TestLoadContractsEmpty(t *testing.T) {
	store := NewContractStore()
	err := store.LoadContracts("test-route", []ContractSpec{})
	if err != nil {
		t.Errorf("LoadContracts should accept empty list, got error: %v", err)
	}
}

// TestLoadContractsSingleValidContract verifies loading a valid contract
func TestLoadContractsSingleValidContract(t *testing.T) {
	store := NewContractStore()

	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"amount": map[string]interface{}{
				"type": "number",
				"minimum": 0.01,
			},
		},
		"required": []string{"amount"},
	}

	contract := ContractSpec{
		ID:      "payment-schema",
		Version: "1.0.0",
		Schema:  schema,
	}

	err := store.LoadContracts("payment-route", []ContractSpec{contract})
	if err != nil {
		t.Fatalf("LoadContracts failed: %v", err)
	}

	// Verify contract is stored
	contracts := store.GetContractsByRoute("payment-route")
	if len(contracts) != 1 {
		t.Fatalf("Expected 1 contract, got %d", len(contracts))
	}
	if contracts[0].ID != "payment-schema" {
		t.Errorf("Expected contract ID 'payment-schema', got %q", contracts[0].ID)
	}
}

// TestLoadContractsMultipleContracts verifies loading multiple contracts on one route
func TestLoadContractsMultipleContracts(t *testing.T) {
	store := NewContractStore()

	schema1 := map[string]interface{}{"type": "object"}
	schema2 := map[string]interface{}{"type": "array"}

	contracts := []ContractSpec{
		{ID: "contract-1", Version: "1.0.0", Schema: schema1},
		{ID: "contract-2", Version: "2.0.0", Schema: schema2},
	}

	err := store.LoadContracts("test-route", contracts)
	if err != nil {
		t.Fatalf("LoadContracts failed: %v", err)
	}

	retrieved := store.GetContractsByRoute("test-route")
	if len(retrieved) != 2 {
		t.Fatalf("Expected 2 contracts, got %d", len(retrieved))
	}
}

// TestLoadContractsEmptyID verifies rejection of contract with empty ID
func TestLoadContractsEmptyID(t *testing.T) {
	store := NewContractStore()

	schema := map[string]interface{}{"type": "object"}
	contract := ContractSpec{
		ID:      "", // Empty ID
		Version: "1.0.0",
		Schema:  schema,
	}

	err := store.LoadContracts("test-route", []ContractSpec{contract})
	if err == nil {
		t.Error("LoadContracts should reject contract with empty ID")
	}
}

// TestLoadContractsEmptyVersion verifies rejection of contract with empty version
func TestLoadContractsEmptyVersion(t *testing.T) {
	store := NewContractStore()

	schema := map[string]interface{}{"type": "object"}
	contract := ContractSpec{
		ID:      "test-contract",
		Version: "", // Empty version
		Schema:  schema,
	}

	err := store.LoadContracts("test-route", []ContractSpec{contract})
	if err == nil {
		t.Error("LoadContracts should reject contract with empty version")
	}
}

// TestLoadContractsNilSchema verifies rejection of contract with nil schema
func TestLoadContractsNilSchema(t *testing.T) {
	store := NewContractStore()

	contract := ContractSpec{
		ID:      "test-contract",
		Version: "1.0.0",
		Schema:  nil, // Nil schema
	}

	err := store.LoadContracts("test-route", []ContractSpec{contract})
	if err == nil {
		t.Error("LoadContracts should reject contract with nil schema")
	}
}

// TestLoadContractsInvalidJSONSchema verifies rejection of invalid JSON Schema
func TestLoadContractsInvalidJSONSchema(t *testing.T) {
	store := NewContractStore()

	// Invalid schema: "type" must be a string or array, not a number
	invalidSchema := map[string]interface{}{
		"type": 123,
	}

	contract := ContractSpec{
		ID:      "test-contract",
		Version: "1.0.0",
		Schema:  invalidSchema,
	}

	err := store.LoadContracts("test-route", []ContractSpec{contract})
	if err == nil {
		t.Error("LoadContracts should reject invalid JSON Schema")
	}
}

// TestLoadContractsJSONStringSchema verifies loading schema from JSON string
func TestLoadContractsJSONStringSchema(t *testing.T) {
	store := NewContractStore()

	schemaJSON := `{
		"type": "object",
		"properties": {
			"name": {"type": "string"}
		}
	}`

	contract := ContractSpec{
		ID:      "json-string-contract",
		Version: "1.0.0",
		Schema:  schemaJSON,
	}

	err := store.LoadContracts("test-route", []ContractSpec{contract})
	if err != nil {
		t.Fatalf("LoadContracts failed with JSON string schema: %v", err)
	}

	// Verify it can be retrieved
	retrieved, err := store.GetContractByID("test-route", "json-string-contract")
	if err != nil {
		t.Fatalf("GetContractByID failed: %v", err)
	}
	if retrieved.ID != "json-string-contract" {
		t.Errorf("Expected contract ID 'json-string-contract', got %q", retrieved.ID)
	}
}

// TestLoadContractsInvalidJSONString verifies rejection of invalid JSON string
func TestLoadContractsInvalidJSONString(t *testing.T) {
	store := NewContractStore()

	invalidJSON := `{invalid json}`

	contract := ContractSpec{
		ID:      "bad-json-contract",
		Version: "1.0.0",
		Schema:  invalidJSON,
	}

	err := store.LoadContracts("test-route", []ContractSpec{contract})
	if err == nil {
		t.Error("LoadContracts should reject invalid JSON string")
	}
}

// TestGetContractByID verifies contract retrieval by ID
func TestGetContractByID(t *testing.T) {
	store := NewContractStore()

	schema := map[string]interface{}{"type": "object"}
	contract := ContractSpec{
		ID:      "my-contract",
		Version: "1.0.0",
		Schema:  schema,
	}

	store.LoadContracts("test-route", []ContractSpec{contract})

	retrieved, err := store.GetContractByID("test-route", "my-contract")
	if err != nil {
		t.Fatalf("GetContractByID failed: %v", err)
	}
	if retrieved.ID != "my-contract" {
		t.Errorf("Expected ID 'my-contract', got %q", retrieved.ID)
	}
}

// TestGetContractByIDNotFound verifies error when contract not found
func TestGetContractByIDNotFound(t *testing.T) {
	store := NewContractStore()

	_, err := store.GetContractByID("nonexistent-route", "nonexistent-contract")
	if err == nil {
		t.Error("GetContractByID should return error for nonexistent contract")
	}
}

// TestGetContractsByRoute verifies contract retrieval by route
func TestGetContractsByRoute(t *testing.T) {
	store := NewContractStore()

	schema := map[string]interface{}{"type": "object"}
	contracts := []ContractSpec{
		{ID: "contract-1", Version: "1.0.0", Schema: schema},
		{ID: "contract-2", Version: "2.0.0", Schema: schema},
	}

	store.LoadContracts("test-route", contracts)

	retrieved := store.GetContractsByRoute("test-route")
	if len(retrieved) != 2 {
		t.Fatalf("Expected 2 contracts, got %d", len(retrieved))
	}
}

// TestGetContractsByRouteNotFound verifies empty list for nonexistent route
func TestGetContractsByRouteNotFound(t *testing.T) {
	store := NewContractStore()

	retrieved := store.GetContractsByRoute("nonexistent-route")
	if retrieved != nil && len(retrieved) > 0 {
		t.Errorf("Expected empty list for nonexistent route, got %d contracts", len(retrieved))
	}
}

// TestValidateMessageConforming verifies validation of conforming message
func TestValidateMessageConforming(t *testing.T) {
	store := NewContractStore()

	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"amount": map[string]interface{}{
				"type":    "number",
				"minimum": 0.01,
			},
		},
		"required": []string{"amount"},
	}

	contract := ContractSpec{
		ID:      "payment-schema",
		Version: "1.0.0",
		Schema:  schema,
	}

	store.LoadContracts("payment-route", []ContractSpec{contract})

	// Conforming message
	body := map[string]interface{}{
		"amount": 100.50,
	}

	err := store.ValidateMessage("payment-route", "payment-schema", body)
	if err != nil {
		t.Fatalf("ValidateMessage should accept conforming message, got error: %v", err)
	}
}

// TestValidateMessageNonConforming verifies detection of non-conforming message
func TestValidateMessageNonConforming(t *testing.T) {
	store := NewContractStore()

	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"amount": map[string]interface{}{
				"type":    "number",
				"minimum": 0.01,
			},
		},
		"required": []string{"amount"},
	}

	contract := ContractSpec{
		ID:      "payment-schema",
		Version: "1.0.0",
		Schema:  schema,
	}

	store.LoadContracts("payment-route", []ContractSpec{contract})

	// Non-conforming message (missing required field)
	body := map[string]interface{}{
		"name": "payment",
	}

	err := store.ValidateMessage("payment-route", "payment-schema", body)
	if err == nil {
		t.Error("ValidateMessage should reject non-conforming message")
	}
}

// TestValidateMessageInvalidType verifies detection of invalid data type
func TestValidateMessageInvalidType(t *testing.T) {
	store := NewContractStore()

	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"amount": map[string]interface{}{
				"type": "number",
			},
		},
		"required": []string{"amount"},
	}

	contract := ContractSpec{
		ID:      "payment-schema",
		Version: "1.0.0",
		Schema:  schema,
	}

	store.LoadContracts("payment-route", []ContractSpec{contract})

	// Non-conforming message (wrong type for amount)
	body := map[string]interface{}{
		"amount": "not-a-number",
	}

	err := store.ValidateMessage("payment-route", "payment-schema", body)
	if err == nil {
		t.Error("ValidateMessage should reject message with invalid type")
	}
}

// TestValidateMessageMinimumConstraint verifies numeric constraints
func TestValidateMessageMinimumConstraint(t *testing.T) {
	store := NewContractStore()

	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"amount": map[string]interface{}{
				"type":    "number",
				"minimum": 0.01,
			},
		},
		"required": []string{"amount"},
	}

	contract := ContractSpec{
		ID:      "payment-schema",
		Version: "1.0.0",
		Schema:  schema,
	}

	store.LoadContracts("payment-route", []ContractSpec{contract})

	// Amount below minimum
	body := map[string]interface{}{
		"amount": 0.00,
	}

	err := store.ValidateMessage("payment-route", "payment-schema", body)
	if err == nil {
		t.Error("ValidateMessage should reject message with amount below minimum")
	}
}

// TestNormalizeSchemaFromMap verifies schema normalization from map
func TestNormalizeSchemaFromMap(t *testing.T) {
	store := NewContractStore()

	schema := map[string]interface{}{
		"type": "object",
	}

	jsonStr, err := store.normalizeSchema(schema)
	if err != nil {
		t.Fatalf("normalizeSchema failed: %v", err)
	}

	// Verify it's valid JSON
	var obj interface{}
	if err := json.Unmarshal([]byte(jsonStr), &obj); err != nil {
		t.Errorf("normalizeSchema produced invalid JSON: %v", err)
	}
}

// TestNormalizeSchemaFromString verifies schema normalization from JSON string
func TestNormalizeSchemaFromString(t *testing.T) {
	store := NewContractStore()

	schemaJSON := `{"type":"object"}`

	json, err := store.normalizeSchema(schemaJSON)
	if err != nil {
		t.Fatalf("normalizeSchema failed: %v", err)
	}

	if json != schemaJSON {
		t.Errorf("normalizeSchema should preserve JSON string unchanged")
	}
}

// TestNormalizeSchemaInvalidString verifies rejection of invalid JSON string
func TestNormalizeSchemaInvalidString(t *testing.T) {
	store := NewContractStore()

	invalidJSON := `{invalid}`

	_, err := store.normalizeSchema(invalidJSON)
	if err == nil {
		t.Error("normalizeSchema should reject invalid JSON")
	}
}
