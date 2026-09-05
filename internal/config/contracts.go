package config

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/naren-chakraview/dim/internal/schema"
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

// LoadContracts loads and validates contracts from a route configuration.
// Resolves registry-backed contracts by fetching from the registry (R18.3).
// Returns error if any contract has invalid JSON Schema or registry resolution fails.
func (cs *ContractStore) LoadContracts(routeName string, contracts []ContractSpec) error {
	return cs.LoadContractsWithContext(context.Background(), routeName, contracts)
}

// LoadContractsWithContext is like LoadContracts but accepts a context for registry operations (R18.3).
func (cs *ContractStore) LoadContractsWithContext(ctx context.Context, routeName string, contracts []ContractSpec) error {
	if len(contracts) == 0 {
		return nil
	}

	// Store contracts for this route (will be updated with resolved contract_version)
	cs.routeContracts[routeName] = contracts

	// Validate and compile each contract
	for i := range contracts {
		contract := &contracts[i]
		// Validate contract ID is not empty
		if contract.ID == "" {
			return fmt.Errorf("route %s: contract at index %d has empty ID", routeName, i)
		}

		// Validate contract version is not empty
		if contract.Version == "" {
			return fmt.Errorf("route %s: contract %q has empty version", routeName, contract.ID)
		}

		// Resolve schema and contract_version based on whether it's inline or registry-backed
		var schemaJSON string
		var err error

		if contract.Registry != nil {
			// Registry-backed contract (R18.3)
			schemaJSON, err = cs.resolveRegistrySchema(ctx, contract.Registry)
			if err != nil {
				return fmt.Errorf("route %s: contract %q: failed to resolve registry schema: %w", routeName, contract.ID, err)
			}

			// Compute contract_version for registry-backed contract
			contract.ContractVersion = computeRegistryContractVersion(contract.Registry)
		} else {
			// Inline contract
			if contract.Schema == nil {
				return fmt.Errorf("route %s: contract %q has nil schema", routeName, contract.ID)
			}

			schemaJSON, err = cs.normalizeSchema(contract.Schema)
			if err != nil {
				return fmt.Errorf("route %s: contract %q: %w", routeName, contract.ID, err)
			}

			// Compute contract_version for inline contract
			contract.ContractVersion = computeInlineContractVersion(schemaJSON)
		}

		// Compile the schema
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

// resolveRegistrySchema resolves a registry-backed schema (R18.3).
func (cs *ContractStore) resolveRegistrySchema(ctx context.Context, ref *RegistryRefSpec) (string, error) {
	client, err := schema.NewRegistryClient(ctx, &schema.SchemaReference{
		Type:    ref.Type,
		URL:     ref.URL,
		Group:   ref.Group,
		Subject: ref.Subject,
		Version: ref.Version,
	})
	if err != nil {
		return "", err
	}

	schemaBytes, _, err := client.GetSchema(ctx, ref.Group, ref.Subject, ref.Version)
	if err != nil {
		return "", err
	}

	return string(schemaBytes), nil
}

// computeInlineContractVersion computes a version tag for an inline contract (R18.3).
// Format: "sha256:<hex-digest>"
func computeInlineContractVersion(schemaJSON string) string {
	hash := sha256.Sum256([]byte(schemaJSON))
	return "sha256:" + hex.EncodeToString(hash[:])
}

// computeRegistryContractVersion computes a version tag for a registry-backed contract (R18.3).
// Format: "<type>:<group>:<subject>:<version>"
func computeRegistryContractVersion(ref *RegistryRefSpec) string {
	// Note: In a real implementation, this would also fetch the resolved version ID
	// from the registry if version is "latest". For now, we use "latest" as-is.
	// This should be improved in a follow-up to use the actual resolved version ID.
	return fmt.Sprintf("%s:%s:%s:%s", ref.Type, ref.Group, ref.Subject, ref.Version)
}
