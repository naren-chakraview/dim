package config

import (
	"encoding/json"
	"fmt"

	"github.com/santhosh-tekuri/jsonschema/v5"
)

// ContractStore manages contract definitions indexed by route and contract ID
type ContractStore struct {
	// routeContracts maps route name -> contract list
	routeContracts map[string][]ContractSpec

	// compiledSchemas caches compiled JSON schemas by route + contract ID
	compiledSchemas map[string]*jsonschema.Schema
}

// NewContractStore creates a new empty contract store
func NewContractStore() *ContractStore {
	return &ContractStore{
		routeContracts:  make(map[string][]ContractSpec),
		compiledSchemas: make(map[string]*jsonschema.Schema),
	}
}

// LoadContracts loads and validates contracts from a route configuration
// Returns error if any contract has invalid JSON Schema or other issues
func (cs *ContractStore) LoadContracts(routeName string, contracts []ContractSpec) error {
	if len(contracts) == 0 {
		return nil
	}

	// Store contracts for this route
	cs.routeContracts[routeName] = contracts

	// Validate and compile each contract
	for i, contract := range contracts {
		// Validate contract ID is not empty
		if contract.ID == "" {
			return fmt.Errorf("route %s: contract at index %d has empty ID", routeName, i)
		}

		// Validate contract version is not empty
		if contract.Version == "" {
			return fmt.Errorf("route %s: contract %q has empty version", routeName, contract.ID)
		}

		// Validate schema
		if contract.Schema == nil {
			return fmt.Errorf("route %s: contract %q has nil schema", routeName, contract.ID)
		}

		// Convert schema to JSON if needed
		schemaJSON, err := cs.normalizeSchema(contract.Schema)
		if err != nil {
			return fmt.Errorf("route %s: contract %q: %w", routeName, contract.ID, err)
		}

		// Compile the schema
		// URL can be empty for inline schemas
		schema, err := jsonschema.CompileString("", schemaJSON)
		if err != nil {
			return fmt.Errorf("route %s: contract %q: invalid JSON Schema: %w", routeName, contract.ID, err)
		}

		// Cache the compiled schema
		cacheKey := cs.cacheKey(routeName, contract.ID)
		cs.compiledSchemas[cacheKey] = schema
	}

	return nil
}

// GetContractsByRoute returns all contracts for a given route
func (cs *ContractStore) GetContractsByRoute(routeName string) []ContractSpec {
	return cs.routeContracts[routeName]
}

// GetContractByID returns a specific contract by route and contract ID
// Returns error if contract is not found
func (cs *ContractStore) GetContractByID(routeName, contractID string) (*ContractSpec, error) {
	contracts, ok := cs.routeContracts[routeName]
	if !ok {
		return nil, fmt.Errorf("route %q not found in contract store", routeName)
	}

	for i := range contracts {
		if contracts[i].ID == contractID {
			return &contracts[i], nil
		}
	}

	return nil, fmt.Errorf("contract %q not found in route %q", contractID, routeName)
}

// GetCompiledSchema returns the compiled JSON Schema for a contract
// Returns error if contract is not found
func (cs *ContractStore) GetCompiledSchema(routeName, contractID string) (*jsonschema.Schema, error) {
	cacheKey := cs.cacheKey(routeName, contractID)
	schema, ok := cs.compiledSchemas[cacheKey]
	if !ok {
		return nil, fmt.Errorf("compiled schema for contract %q in route %q not found", contractID, routeName)
	}
	return schema, nil
}

// normalizeSchema converts schema to JSON string for compilation
// Handles both YAML objects and JSON strings
func (cs *ContractStore) normalizeSchema(schema interface{}) (string, error) {
	switch v := schema.(type) {
	case string:
		// Already a JSON string
		// Validate it's valid JSON
		var obj interface{}
		if err := json.Unmarshal([]byte(v), &obj); err != nil {
			return "", fmt.Errorf("schema string is not valid JSON: %w", err)
		}
		return v, nil

	case map[string]interface{}:
		// YAML object, convert to JSON
		jsonBytes, err := json.Marshal(v)
		if err != nil {
			return "", fmt.Errorf("failed to marshal schema object to JSON: %w", err)
		}
		return string(jsonBytes), nil

	default:
		// Try generic marshaling
		jsonBytes, err := json.Marshal(v)
		if err != nil {
			return "", fmt.Errorf("failed to marshal schema to JSON: %w", err)
		}
		return string(jsonBytes), nil
	}
}

// cacheKey generates a cache key for compiled schemas
func (cs *ContractStore) cacheKey(routeName, contractID string) string {
	return routeName + ":" + contractID
}

// ValidateMessage validates a message body against a contract schema
// Returns an error if validation fails
func (cs *ContractStore) ValidateMessage(routeName, contractID string, body interface{}) error {
	schema, err := cs.GetCompiledSchema(routeName, contractID)
	if err != nil {
		return err
	}

	return schema.Validate(body)
}
