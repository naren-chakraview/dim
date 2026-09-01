package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/santhosh-tekuri/jsonschema/v5"
	"gopkg.in/yaml.v3"
)

// LoadRouteConfig loads and validates a route configuration from a YAML file
func LoadRouteConfig(path string) (*RouteConfig, error) {
	// Read the file
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse YAML into RouteConfig
	var config RouteConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	// Validate against JSON Schema
	if err := validateRouteConfig(&config); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return &config, nil
}

// validateRouteConfig validates a RouteConfig against the JSON Schema
func validateRouteConfig(config *RouteConfig) error {
	// Convert config to JSON for schema validation
	jsonData, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config to JSON: %w", err)
	}

	// Load the schema
	schema, err := loadSchema()
	if err != nil {
		return fmt.Errorf("failed to load schema: %w", err)
	}

	// Parse JSON for validation
	var obj interface{}
	if err := json.Unmarshal(jsonData, &obj); err != nil {
		return fmt.Errorf("failed to parse JSON: %w", err)
	}

	// Validate
	if err := schema.Validate(obj); err != nil {
		return fmt.Errorf("schema validation error: %w", err)
	}

	return nil
}

// loadSchema loads the JSON Schema from the embedded or file-based schema
func loadSchema() (*jsonschema.Schema, error) {
	// Try to load from standard location relative to go module root
	// For now, we'll compile the schema directly from the file system
	schemaPath := filepath.Join(getSchemaDir(), "route.schema.json")

	data, err := os.ReadFile(schemaPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read schema file at %s: %w", schemaPath, err)
	}

	// Parse and compile the schema
	compiler := jsonschema.NewCompiler()
	if err := compiler.AddResource("route.schema.json", bytes.NewReader(data)); err != nil {
		return nil, fmt.Errorf("failed to parse schema: %w", err)
	}

	schema, err := compiler.Compile("route.schema.json")
	if err != nil {
		return nil, fmt.Errorf("failed to compile schema: %w", err)
	}

	return schema, nil
}

// getSchemaDir returns the path to the schemas directory
// This is relative to the module root (dim/)
func getSchemaDir() string {
	// Try to find the schemas directory by looking up from the current working directory
	// or from the go module root
	currentDir, err := os.Getwd()
	if err != nil {
		return "./schemas"
	}

	// Look for schemas directory in current directory or parent directories
	for {
		schemasPath := filepath.Join(currentDir, "schemas")
		if _, err := os.Stat(schemasPath); err == nil {
			return schemasPath
		}

		parentDir := filepath.Dir(currentDir)
		if parentDir == currentDir {
			// Reached the filesystem root
			break
		}
		currentDir = parentDir
	}

	// Fallback
	return "./schemas"
}
