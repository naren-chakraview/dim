package steps

import (
	"context"
	"testing"

	"github.com/naren-chakraview/dim/internal/config"
	"github.com/naren-chakraview/dim/internal/engine"
)

// setupContractStoreWithPaymentSchema creates a contract store with a payment schema
func setupContractStoreWithPaymentSchema(t *testing.T) *config.ContractStore {
	store := config.NewContractStore()

	schema := map[string]interface{}{
		"type": "object",
		"required": []string{"amount", "currency", "account_id"},
		"properties": map[string]interface{}{
			"amount": map[string]interface{}{
				"type":    "number",
				"minimum": 0.01,
			},
			"currency": map[string]interface{}{
				"type":    "string",
				"pattern": "^[A-Z]{3}$",
			},
			"account_id": map[string]interface{}{
				"type": "string",
			},
		},
	}

	contract := config.ContractSpec{
		ID:          "payment-schema",
		Version:     "1.0.0",
		Schema:      schema,
		OnViolation: "payment-dlq",
		Strict:      false,
	}

	err := store.LoadContracts("payment-processor", []config.ContractSpec{contract})
	if err != nil {
		t.Fatalf("Failed to load contract: %v", err)
	}

	return store
}

// setupContractStoreWithStrictContract creates a contract store with strict mode enabled
func setupContractStoreWithStrictContract(t *testing.T) *config.ContractStore {
	store := config.NewContractStore()

	schema := map[string]interface{}{
		"type": "object",
		"required": []string{"amount"},
		"properties": map[string]interface{}{
			"amount": map[string]interface{}{
				"type": "number",
			},
		},
	}

	contract := config.ContractSpec{
		ID:          "strict-contract",
		Version:     "2.0.0",
		Schema:      schema,
		OnViolation: "error-dlq",
		Strict:      true,
	}

	err := store.LoadContracts("strict-route", []config.ContractSpec{contract})
	if err != nil {
		t.Fatalf("Failed to load contract: %v", err)
	}

	return store
}

// Test1: Conforming message passes validation
func TestContractStepConformingMessage(t *testing.T) {
	store := setupContractStoreWithPaymentSchema(t)
	step, err := NewContractStep(store, "payment-processor", "payment-schema")
	if err != nil {
		t.Fatalf("NewContractStep failed: %v", err)
	}

	ctx := context.Background()
	msg := engine.NewMessage(map[string]interface{}{
		"amount":      100.50,
		"currency":    "USD",
		"account_id":  "acc-123",
	}, "payment-processor", "v1")

	result, err := step.Execute(ctx, msg)

	if err != nil {
		t.Fatalf("Execute should succeed with conforming message, got error: %v", err)
	}

	if result == nil {
		t.Error("Expected message to pass through, but got nil")
	}

	if result.Metadata.ContractVersion != "1.0.0" {
		t.Errorf("Expected contract_version '1.0.0', got %q", result.Metadata.ContractVersion)
	}
}

// Test2: Non-conforming message detected as violation
func TestContractStepNonConformingMessage(t *testing.T) {
	store := setupContractStoreWithPaymentSchema(t)
	step, err := NewContractStep(store, "payment-processor", "payment-schema")
	if err != nil {
		t.Fatalf("NewContractStep failed: %v", err)
	}

	ctx := context.Background()
	msg := engine.NewMessage(map[string]interface{}{
		"amount": 100.50,
		// Missing currency and account_id
	}, "payment-processor", "v1")

	result, err := step.Execute(ctx, msg)

	// Non-strict mode should not return error
	if err != nil {
		t.Fatalf("Non-strict mode should not return error: %v", err)
	}

	// Message should pass through with violation marker
	if result == nil {
		t.Error("Non-strict mode should pass through message with violation marker")
	}

	if result.Metadata.ContractViolation == nil {
		t.Error("Expected ContractViolation marker to be set")
	}
}

// Test3: Contract ID lookup works
func TestContractStepContractLookup(t *testing.T) {
	store := config.NewContractStore()

	schema := map[string]interface{}{"type": "object"}
	contract := config.ContractSpec{
		ID:      "test-contract",
		Version: "1.0.0",
		Schema:  schema,
	}

	store.LoadContracts("test-route", []config.ContractSpec{contract})

	step, err := NewContractStep(store, "test-route", "test-contract")
	if err != nil {
		t.Fatalf("NewContractStep failed: %v", err)
	}

	if step.contractID != "test-contract" {
		t.Errorf("Expected contractID 'test-contract', got %q", step.contractID)
	}
}

// Test4: Multiple contracts on single route
func TestContractStepMultipleContracts(t *testing.T) {
	store := config.NewContractStore()

	schema1 := map[string]interface{}{"type": "object"}
	schema2 := map[string]interface{}{"type": "array"}

	contracts := []config.ContractSpec{
		{ID: "contract-1", Version: "1.0.0", Schema: schema1},
		{ID: "contract-2", Version: "2.0.0", Schema: schema2},
	}

	store.LoadContracts("multi-route", contracts)

	step1, err := NewContractStep(store, "multi-route", "contract-1")
	if err != nil {
		t.Fatalf("NewContractStep for contract-1 failed: %v", err)
	}
	if step1.contractID != "contract-1" {
		t.Errorf("Expected contractID 'contract-1', got %q", step1.contractID)
	}

	step2, err := NewContractStep(store, "multi-route", "contract-2")
	if err != nil {
		t.Fatalf("NewContractStep for contract-2 failed: %v", err)
	}
	if step2.contractID != "contract-2" {
		t.Errorf("Expected contractID 'contract-2', got %q", step2.contractID)
	}
}

// Test5: Contract version stamped correctly
func TestContractStepVersionStamping(t *testing.T) {
	store := setupContractStoreWithPaymentSchema(t)
	step, err := NewContractStep(store, "payment-processor", "payment-schema")
	if err != nil {
		t.Fatalf("NewContractStep failed: %v", err)
	}

	ctx := context.Background()
	msg := engine.NewMessage(map[string]interface{}{
		"amount":      50.00,
		"currency":    "EUR",
		"account_id":  "acc-456",
	}, "payment-processor", "v1")

	result, err := step.Execute(ctx, msg)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if result.Metadata.ContractVersion != "1.0.0" {
		t.Errorf("Expected contract_version '1.0.0', got %q", result.Metadata.ContractVersion)
	}
}

// Test6: Violation marker set on metadata
func TestContractStepViolationMarker(t *testing.T) {
	store := setupContractStoreWithPaymentSchema(t)
	step, err := NewContractStep(store, "payment-processor", "payment-schema")
	if err != nil {
		t.Fatalf("NewContractStep failed: %v", err)
	}

	ctx := context.Background()
	msg := engine.NewMessage(map[string]interface{}{
		"invalid": "message",
	}, "payment-processor", "v1")

	result, err := step.Execute(ctx, msg)

	if err != nil {
		t.Fatalf("Non-strict mode should not return error: %v", err)
	}

	if result.Metadata.ContractViolation == nil {
		t.Fatal("Expected ContractViolation to be set")
	}

	violation, ok := result.Metadata.ContractViolation.(*ViolationInfo)
	if !ok {
		t.Fatal("ContractViolation is not a *ViolationInfo")
	}

	if violation.ContractID != "payment-schema" {
		t.Errorf("Expected ContractID 'payment-schema', got %q", violation.ContractID)
	}

	if violation.Severity != "warning" {
		t.Errorf("Expected Severity 'warning', got %q", violation.Severity)
	}

	if violation.Reason == "" {
		t.Error("Expected non-empty Reason")
	}
}

// Test7: Strict mode returns error
func TestContractStepStrictMode(t *testing.T) {
	store := setupContractStoreWithStrictContract(t)
	step, err := NewContractStep(store, "strict-route", "strict-contract")
	if err != nil {
		t.Fatalf("NewContractStep failed: %v", err)
	}

	ctx := context.Background()
	msg := engine.NewMessage(map[string]interface{}{
		"invalid": "message",
	}, "strict-route", "v1")

	result, err := step.Execute(ctx, msg)

	if err == nil {
		t.Error("Strict mode should return error on validation failure")
	}

	if result != nil {
		t.Error("Strict mode should return nil message on validation failure")
	}

	if !IsContractViolation(err) {
		t.Error("Error should be a ContractViolationError")
	}

	violation := GetContractViolation(err)
	if violation == nil {
		t.Fatal("Expected to extract violation info from error")
	}

	if violation.Severity != "error" {
		t.Errorf("Expected Severity 'error', got %q", violation.Severity)
	}
}

// Test8: Non-strict mode continues to success path
func TestContractStepNonStrictContinues(t *testing.T) {
	store := setupContractStoreWithPaymentSchema(t)
	step, err := NewContractStep(store, "payment-processor", "payment-schema")
	if err != nil {
		t.Fatalf("NewContractStep failed: %v", err)
	}

	ctx := context.Background()
	msg := engine.NewMessage(map[string]interface{}{
		"amount": -50.00, // Violates minimum constraint
		"currency": "USD",
		"account_id": "acc-789",
	}, "payment-processor", "v1")

	result, err := step.Execute(ctx, msg)

	// Non-strict should not return error
	if err != nil {
		t.Fatalf("Non-strict mode should not return error: %v", err)
	}

	// Message should still be returned
	if result == nil {
		t.Error("Non-strict mode should return message")
	}

	// Should have violation marker
	if result.Metadata.ContractViolation == nil {
		t.Error("Expected ContractViolation marker")
	}
}

// Test9: Correlation ID preserved through violation
func TestContractStepCorrelationIDPreserved(t *testing.T) {
	store := setupContractStoreWithPaymentSchema(t)
	step, err := NewContractStep(store, "payment-processor", "payment-schema")
	if err != nil {
		t.Fatalf("NewContractStep failed: %v", err)
	}

	ctx := context.Background()
	msg := engine.NewMessage(map[string]interface{}{
		"invalid": "payload",
	}, "payment-processor", "v1")

	originalCorrelationID := msg.Metadata.CorrelationID

	result, err := step.Execute(ctx, msg)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if result.Metadata.CorrelationID != originalCorrelationID {
		t.Errorf("CorrelationID should be preserved: %q != %q", result.Metadata.CorrelationID, originalCorrelationID)
	}
}

// Test10: Missing required fields detected
func TestContractStepMissingRequiredFields(t *testing.T) {
	store := setupContractStoreWithPaymentSchema(t)
	step, err := NewContractStep(store, "payment-processor", "payment-schema")
	if err != nil {
		t.Fatalf("NewContractStep failed: %v", err)
	}

	ctx := context.Background()
	msg := engine.NewMessage(map[string]interface{}{
		"amount": 100.50,
		// Missing currency and account_id
	}, "payment-processor", "v1")

	result, err := step.Execute(ctx, msg)

	if err != nil {
		t.Fatalf("Non-strict should not return error: %v", err)
	}

	if result.Metadata.ContractViolation == nil {
		t.Fatal("Expected ContractViolation for missing required fields")
	}

	violation := result.Metadata.ContractViolation.(*ViolationInfo)
	if violation.ContractID != "payment-schema" {
		t.Errorf("Expected ContractID 'payment-schema', got %q", violation.ContractID)
	}
}

// Test11: Invalid data type detected
func TestContractStepInvalidDataType(t *testing.T) {
	store := setupContractStoreWithPaymentSchema(t)
	step, err := NewContractStep(store, "payment-processor", "payment-schema")
	if err != nil {
		t.Fatalf("NewContractStep failed: %v", err)
	}

	ctx := context.Background()
	msg := engine.NewMessage(map[string]interface{}{
		"amount":     "not-a-number", // Wrong type
		"currency":   "USD",
		"account_id": "acc-123",
	}, "payment-processor", "v1")

	result, err := step.Execute(ctx, msg)

	if err != nil {
		t.Fatalf("Non-strict should not return error: %v", err)
	}

	if result.Metadata.ContractViolation == nil {
		t.Fatal("Expected ContractViolation for invalid data type")
	}
}

// Test12: Invalid schema format rejected at load time
func TestContractStepInvalidSchemaFormat(t *testing.T) {
	store := config.NewContractStore()

	invalidSchema := map[string]interface{}{
		"type": 123, // Invalid: type should be string or array
	}

	contract := config.ContractSpec{
		ID:      "invalid-contract",
		Version: "1.0.0",
		Schema:  invalidSchema,
	}

	err := store.LoadContracts("test-route", []config.ContractSpec{contract})
	if err == nil {
		t.Error("LoadContracts should reject invalid JSON Schema")
	}

	// Attempting to create step with unloaded contract should fail
	_, err = NewContractStep(store, "test-route", "invalid-contract")
	if err == nil {
		t.Error("NewContractStep should fail if contract schema is invalid")
	}
}

// Test13: Nil message handling
func TestContractStepNilMessage(t *testing.T) {
	store := setupContractStoreWithPaymentSchema(t)
	step, err := NewContractStep(store, "payment-processor", "payment-schema")
	if err != nil {
		t.Fatalf("NewContractStep failed: %v", err)
	}

	ctx := context.Background()
	result, err := step.Execute(ctx, nil)

	if result != nil {
		t.Error("Execute should return nil for nil message")
	}
	if err != nil {
		t.Error("Execute should not return error for nil message")
	}
}

// Test14: OnViolationSink retrieval
func TestContractStepOnViolationSink(t *testing.T) {
	store := setupContractStoreWithPaymentSchema(t)
	step, err := NewContractStep(store, "payment-processor", "payment-schema")
	if err != nil {
		t.Fatalf("NewContractStep failed: %v", err)
	}

	sink := step.GetOnViolationSink()
	if sink != "payment-dlq" {
		t.Errorf("Expected on_violation sink 'payment-dlq', got %q", sink)
	}
}

// Test15: NewContractStep validation
func TestContractStepCreationValidation(t *testing.T) {
	store := config.NewContractStore()

	// Test nil store
	_, err := NewContractStep(nil, "test-route", "test-contract")
	if err == nil {
		t.Error("NewContractStep should reject nil store")
	}

	// Test empty route name
	_, err = NewContractStep(store, "", "test-contract")
	if err == nil {
		t.Error("NewContractStep should reject empty route name")
	}

	// Test empty contract ID
	_, err = NewContractStep(store, "test-route", "")
	if err == nil {
		t.Error("NewContractStep should reject empty contract ID")
	}

	// Test nonexistent contract
	_, err = NewContractStep(store, "test-route", "nonexistent")
	if err == nil {
		t.Error("NewContractStep should reject nonexistent contract")
	}
}

// Test16: Complex nested schema validation
func TestContractStepComplexSchema(t *testing.T) {
	store := config.NewContractStore()

	// Use a JSON string for the schema to avoid Go type issues with validators
	schemaJSON := `{
		"type": "object",
		"required": ["order"],
		"properties": {
			"order": {
				"type": "object",
				"required": ["total"],
				"properties": {
					"total": {
						"type": "number",
						"minimum": 0
					},
					"items": {
						"type": "array",
						"minItems": 1
					}
				}
			}
		}
	}`

	contract := config.ContractSpec{
		ID:      "order-schema",
		Version: "1.0.0",
		Schema:  schemaJSON,
		Strict:  false,
	}

	store.LoadContracts("order-route", []config.ContractSpec{contract})
	step, err := NewContractStep(store, "order-route", "order-schema")
	if err != nil {
		t.Fatalf("NewContractStep failed: %v", err)
	}

	ctx := context.Background()

	// Valid nested structure
	validMsg := engine.NewMessage(map[string]interface{}{
		"order": map[string]interface{}{
			"items": []interface{}{"item-1", "item-2"},
			"total": 99.99,
		},
	}, "order-route", "v1")

	result, err := step.Execute(ctx, validMsg)
	if err != nil {
		t.Fatalf("Should validate valid nested structure: %v", err)
	}
	if result.Metadata.ContractViolation != nil {
		violation := result.Metadata.ContractViolation.(*ViolationInfo)
		t.Errorf("Valid nested structure should not have violation marker. Got: %s", violation.Reason)
	}

	// Invalid: missing required order field
	invalidMsg := engine.NewMessage(map[string]interface{}{
		"name": "invalid",
	}, "order-route", "v1")

	result, err = step.Execute(ctx, invalidMsg)
	if err != nil {
		t.Fatalf("Non-strict should not return error: %v", err)
	}
	if result.Metadata.ContractViolation == nil {
		t.Error("Invalid nested structure should have violation marker")
	}
}
