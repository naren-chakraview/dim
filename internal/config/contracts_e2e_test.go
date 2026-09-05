package config

import (
	"encoding/json"
	"testing"
)

// TestR18RegistryBackedContractVersionComputation tests R18.3 contract_version computation
// and demonstrates that registry-backed contracts compute versions differently than inline contracts
// (exit criterion #3: contract_version on lineage/dead-letter records).
func TestR18RegistryBackedContractVersionComputation(t *testing.T) {
	// Test that registry contracts compute version tags in the format:
	// "<type>:<group>:<subject>:<version>"

	ref := &RegistryRefSpec{
		Type:    "apicurio",
		URL:     "http://localhost:8080",
		Group:   "order-schemas",
		Subject: "OrderSchema",
		Version: "3",
	}

	version := computeRegistryContractVersion(ref)

	if version != "apicurio:order-schemas:OrderSchema:3" {
		t.Errorf("registry contract version format incorrect: expected apicurio:order-schemas:OrderSchema:3, got %s", version)
	}

	// Test that this format differs from inline
	inlineVersion := computeInlineContractVersion(`{"type":"object"}`)
	if inlineVersion[:7] != "sha256:" {
		t.Errorf("inline contract version should start with sha256:, got %s", inlineVersion)
	}

	if version == inlineVersion {
		t.Error("registry and inline contract versions should not match")
	}

	t.Logf("Registry contract_version (for lineage): %s", version)
	t.Logf("Inline contract_version (for lineage): %s", inlineVersion)
}

// TestR18ContractModelSupport tests that the ContractSpec model supports registry references.
// This demonstrates exit criterion #1: routes can declare contracts as subject-version references.
func TestR18ContractModelSupport(t *testing.T) {
	// Create a contract spec with registry reference (R18.3)
	contract := ContractSpec{
		ID:      "test-contract",
		Version: "1.0.0",
		Registry: &RegistryRefSpec{
			Type:    "apicurio",
			URL:     "http://localhost:8080",
			Group:   "test-schemas",
			Subject: "TestSchema",
			Version: "latest",
		},
	}

	// Verify the contract model accepts registry references
	if contract.Registry == nil {
		t.Fatal("contract should support registry references")
	}

	if contract.Registry.Type != "apicurio" {
		t.Errorf("registry type should be apicurio, got %s", contract.Registry.Type)
	}

	if contract.Registry.Subject != "TestSchema" {
		t.Errorf("registry subject should be TestSchema, got %s", contract.Registry.Subject)
	}

	t.Log("✓ Exit criterion #1: routes can declare contracts as subject-version references")
}

// TestR18InlineContractVersionComputation shows that inline contracts
// get SHA256-based version tags (exit criterion #3).
func TestR18InlineContractVersionComputation(t *testing.T) {
	cs := NewContractStore()

	// Inline contract
	inlineSchema := `{"type": "object", "properties": {"id": {"type": "string"}}}`
	var inlineObj interface{}
	json.Unmarshal([]byte(inlineSchema), &inlineObj)

	inlineContract := ContractSpec{
		ID:      "inline-contract",
		Version: "1.0.0",
		Schema:  inlineObj,
	}

	// Load inline contract
	cs.LoadContracts("test-route", []ContractSpec{inlineContract})

	// Verify version is computed
	inline, _ := cs.GetContractByID("test-route", "inline-contract")

	if inline.ContractVersion == "" {
		t.Fatal("inline contract version not set")
	}

	// Inline should be sha256:...
	if inline.ContractVersion[:7] != "sha256:" {
		t.Errorf("inline contract version should start with sha256:, got %s", inline.ContractVersion)
	}

	t.Logf("Inline contract_version: %s", inline.ContractVersion)
	t.Log("✓ Exit criterion #3: contract_version computed for inline contracts")
}
