package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// FragmentResolver handles loading and resolution of YAML fragments with $import directives
type FragmentResolver struct {
	BasePath string                     // Base directory for resolving relative imports
	Cache    map[string]interface{}     // Cached parsed fragments (key: absolute path)
	Visiting map[string]bool            // Currently visiting nodes (for cycle detection)
}

// NewFragmentResolver creates a new fragment resolver with the given base path
func NewFragmentResolver(basePath string) *FragmentResolver {
	if basePath == "" {
		basePath = "."
	}
	return &FragmentResolver{
		BasePath: basePath,
		Cache:    make(map[string]interface{}),
		Visiting: make(map[string]bool),
	}
}

// ResolveImports recursively resolves all $import directives in a config, merging fragments
// Returns a fully-resolved config with all imports resolved and merged
func (fr *FragmentResolver) ResolveImports(config map[string]interface{}) (map[string]interface{}, error) {
	// Check if there's an $import directive at the top level
	if importPath, ok := config["$import"]; ok {
		importPathStr, ok := importPath.(string)
		if !ok {
			return nil, fmt.Errorf("invalid $import directive: must be a string, got %T", importPath)
		}

		// Load and resolve the imported fragment
		resolvedImport, err := fr.resolveImportPath(importPathStr)
		if err != nil {
			return nil, err
		}

		// Recursively resolve imports in the loaded fragment
		if hasImport(resolvedImport) {
			resolvedImport, err = fr.ResolveImports(resolvedImport)
			if err != nil {
				return nil, err
			}
		}

		// Remove the $import directive from the current config
		currentWithoutImport := make(map[string]interface{})
		for k, v := range config {
			if k != "$import" {
				currentWithoutImport[k] = v
			}
		}

		// Merge current config on top of imported config (current overrides imported)
		return MergeConfigs(resolvedImport, currentWithoutImport)
	}

	return config, nil
}

// resolveImportPath loads and parses a fragment file, handling relative path resolution
// and cycle detection
func (fr *FragmentResolver) resolveImportPath(importPath string) (map[string]interface{}, error) {
	// Resolve the path relative to the base path if it's relative
	absPath := importPath
	if !filepath.IsAbs(importPath) {
		absPath = filepath.Join(fr.BasePath, importPath)
	}

	// Normalize the path to detect cycles correctly
	absPath, err := filepath.Abs(absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve import path %s: %w", importPath, err)
	}

	// Check for cycles
	if fr.Visiting[absPath] {
		return nil, fmt.Errorf("circular import detected: %s", absPath)
	}

	// Check cache
	if cached, ok := fr.Cache[absPath]; ok {
		return cached.(map[string]interface{}), nil
	}

	// Mark as visiting for cycle detection
	fr.Visiting[absPath] = true
	defer func() {
		fr.Visiting[absPath] = false
	}()

	// Read the file
	data, err := os.ReadFile(absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read fragment file %s: %w", absPath, err)
	}

	// Parse YAML
	var fragment map[string]interface{}
	if err := yaml.Unmarshal(data, &fragment); err != nil {
		return nil, fmt.Errorf("failed to parse YAML fragment %s: %w", absPath, err)
	}

	if fragment == nil {
		fragment = make(map[string]interface{})
	}

	// Recursively resolve imports in this fragment, but change basePath to the directory
	// containing this fragment so relative paths are resolved correctly
	oldBasePath := fr.BasePath
	fr.BasePath = filepath.Dir(absPath)
	defer func() {
		fr.BasePath = oldBasePath
	}()

	// Recursively resolve imports in the fragment
	if hasImport(fragment) {
		fragment, err = fr.ResolveImports(fragment)
		if err != nil {
			return nil, err
		}
	}

	// Cache the resolved fragment
	fr.Cache[absPath] = fragment

	return fragment, nil
}

// hasImport checks if a config map has an $import directive
func hasImport(config map[string]interface{}) bool {
	_, ok := config["$import"]
	return ok
}

// HasCycle detects circular imports and returns the cycle path if found
// Returns (hasCycle bool, cyclePath []string)
func (fr *FragmentResolver) HasCycle(path string) (bool, []string) {
	visited := make(map[string]bool)
	visiting := make(map[string]bool)
	var cyclePath []string

	absPath, err := filepath.Abs(path)
	if err != nil {
		return false, nil
	}

	found := fr.detectCycle(absPath, visited, visiting, &cyclePath)
	return found, cyclePath
}

// detectCycle recursively checks for cycles in imports
func (fr *FragmentResolver) detectCycle(path string, visited map[string]bool, visiting map[string]bool, cyclePath *[]string) bool {
	if visiting[path] {
		// Found a cycle
		*cyclePath = append(*cyclePath, path)
		return true
	}

	if visited[path] {
		return false
	}

	visiting[path] = true

	// Read and parse the file to find its imports
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}

	var fragment map[string]interface{}
	if err := yaml.Unmarshal(data, &fragment); err != nil {
		return false
	}

	// Check for $import directive
	if importPath, ok := fragment["$import"]; ok {
		importPathStr, ok := importPath.(string)
		if !ok {
			return false
		}

		absImportPath := importPathStr
		if !filepath.IsAbs(importPathStr) {
			absImportPath = filepath.Join(filepath.Dir(path), importPathStr)
		}

		absImportPath, err := filepath.Abs(absImportPath)
		if err != nil {
			return false
		}

		*cyclePath = append(*cyclePath, path)
		if fr.detectCycle(absImportPath, visited, visiting, cyclePath) {
			return true
		}
		*cyclePath = (*cyclePath)[:len(*cyclePath)-1]
	}

	visiting[path] = false
	visited[path] = true
	return false
}
