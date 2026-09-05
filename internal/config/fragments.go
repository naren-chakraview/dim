package config

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"

	"gopkg.in/yaml.v3"
)

// FragmentResolver handles loading and resolution of YAML fragments with $import directives
// and parameter substitution via ${PARAM:name} syntax
type FragmentResolver struct {
	BasePath   string                     // Base directory for resolving relative imports
	Cache      map[string]interface{}     // Cached parsed fragments (key: absolute path)
	Visiting   map[string]bool            // Currently visiting nodes (for cycle detection)
	Parameters map[string]interface{}     // Current parameter scope (for substitution)
}

// paramRegex matches ${PARAM:name} patterns
var paramRegex = regexp.MustCompile(`\$\{PARAM:([^}]+)\}`)

// NewFragmentResolver creates a new fragment resolver with the given base path
func NewFragmentResolver(basePath string) *FragmentResolver {
	if basePath == "" {
		basePath = "."
	}
	return &FragmentResolver{
		BasePath:   basePath,
		Cache:      make(map[string]interface{}),
		Visiting:   make(map[string]bool),
		Parameters: make(map[string]interface{}),
	}
}

// ResolveImports recursively resolves all $import directives in a config, merging fragments
// and substituting parameters. Returns a fully-resolved config with all imports resolved,
// merged, and parameters substituted.
func (fr *FragmentResolver) ResolveImports(config map[string]interface{}) (map[string]interface{}, error) {
	// Extract parameters from current config (route-level overrides)
	routeParams := extractParams(config)

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

		// Extract fragment-level parameter defaults
		fragmentParams := extractParams(resolvedImport)

		// Merge parameters: route overrides fragment
		mergedParams := make(map[string]interface{})
		for k, v := range fragmentParams {
			mergedParams[k] = v
		}
		for k, v := range routeParams {
			mergedParams[k] = v
		}

		// Update resolver's parameter scope
		oldParams := fr.Parameters
		fr.Parameters = mergedParams
		defer func() {
			fr.Parameters = oldParams
		}()

		// Recursively resolve imports in the loaded fragment
		if hasImport(resolvedImport) {
			resolvedImport, err = fr.ResolveImports(resolvedImport)
			if err != nil {
				return nil, err
			}
		}

		// Remove $import and $params directives from current config
		currentWithoutDirectives := make(map[string]interface{})
		for k, v := range config {
			if k != "$import" && k != "$params" {
				currentWithoutDirectives[k] = v
			}
		}

		// Remove $params from fragment
		resolvedWithoutParams := make(map[string]interface{})
		for k, v := range resolvedImport {
			if k != "$params" {
				resolvedWithoutParams[k] = v
			}
		}

		// Merge current config on top of imported config (current overrides imported)
		merged, err := MergeConfigs(resolvedWithoutParams, currentWithoutDirectives)
		if err != nil {
			return nil, err
		}

		// Substitute parameters in the merged config
		substituted, err := fr.substituteParams(merged, mergedParams)
		if err != nil {
			return nil, err
		}

		result, ok := substituted.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("internal error: substituted config is not a map")
		}
		return result, nil
	}

	// No import, but might have parameters to substitute
	mergedParams := make(map[string]interface{})
	for k, v := range fr.Parameters {
		mergedParams[k] = v
	}
	for k, v := range routeParams {
		mergedParams[k] = v
	}

	// Remove $params directive
	withoutParams := make(map[string]interface{})
	for k, v := range config {
		if k != "$params" {
			withoutParams[k] = v
		}
	}

	substituted, err := fr.substituteParams(withoutParams, mergedParams)
	if err != nil {
		return nil, err
	}

	result, ok := substituted.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("internal error: substituted config is not a map")
	}
	return result, nil
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

// extractParams extracts the $params section from a config, returning a map of parameters
// Parameters can be of any type (string, number, boolean, object, array)
func extractParams(config map[string]interface{}) map[string]interface{} {
	if paramsInterface, ok := config["$params"]; ok {
		if params, ok := paramsInterface.(map[string]interface{}); ok {
			return params
		}
	}
	return make(map[string]interface{})
}

// substituteParams recursively substitutes ${PARAM:name} references in a config
// using values from the parameters map. Returns an error if a parameter is undefined.
func (fr *FragmentResolver) substituteParams(config interface{}, params map[string]interface{}) (interface{}, error) {
	switch v := config.(type) {
	case string:
		return fr.substituteString(v, params)

	case map[string]interface{}:
		result := make(map[string]interface{})
		for k, val := range v {
			substituted, err := fr.substituteParams(val, params)
			if err != nil {
				return nil, err
			}
			result[k] = substituted
		}
		return result, nil

	case []interface{}:
		result := make([]interface{}, len(v))
		for i, val := range v {
			substituted, err := fr.substituteParams(val, params)
			if err != nil {
				return nil, err
			}
			result[i] = substituted
		}
		return result, nil

	default:
		// For numbers, booleans, null, etc., return as-is
		return config, nil
	}
}

// substituteString substitutes ${PARAM:name} patterns in a string
// If the entire string is a parameter reference and has a non-string value,
// that value is returned (type conversion). Otherwise, substitutions are string-replaced.
func (fr *FragmentResolver) substituteString(s string, params map[string]interface{}) (interface{}, error) {
	// Check if entire string is a single parameter reference
	matches := paramRegex.FindStringSubmatch(s)
	if matches != nil && matches[0] == s {
		// Entire string is a parameter reference; return the typed value
		paramName := matches[1]
		if value, ok := params[paramName]; ok {
			return value, nil
		}
		return nil, fmt.Errorf("undefined parameter '%s'", paramName)
	}

	// Partial substitution: replace ${PARAM:name} with string values
	result := paramRegex.ReplaceAllStringFunc(s, func(match string) string {
		submatches := paramRegex.FindStringSubmatch(match)
		if len(submatches) != 2 {
			return match
		}

		paramName := submatches[1]
		if value, ok := params[paramName]; ok {
			return valueToString(value)
		}

		// Return original if not found; error will be caught on re-validation
		return match
	})

	// Verify all parameter references were substituted
	if paramRegex.MatchString(result) {
		remainingMatches := paramRegex.FindAllStringSubmatch(result, -1)
		if len(remainingMatches) > 0 {
			paramName := remainingMatches[0][1]
			return nil, fmt.Errorf("undefined parameter '%s'", paramName)
		}
	}

	return result, nil
}

// valueToString converts a parameter value to a string for substitution
func valueToString(v interface{}) string {
	switch val := v.(type) {
	case string:
		return val
	case bool:
		return strconv.FormatBool(val)
	case int, int32, int64:
		return fmt.Sprintf("%v", val)
	case float64:
		// Format float without unnecessary decimals
		s := strconv.FormatFloat(val, 'f', -1, 64)
		return s
	default:
		// For complex types, return YAML representation
		return fmt.Sprintf("%v", val)
	}
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
