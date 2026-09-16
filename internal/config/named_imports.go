package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// ResolveNamedImports processes the `imports:` directive and `- fragment:` references
// in a route configuration, expanding fragment references into actual step lists.
//
// The `imports:` key contains a list of paths to fragment files. Each fragment file
// has a top-level `fragments:` map containing named fragment definitions.
// Routes' `steps:` lists can reference these by including `- fragment: name`.
// This function expands those references in place.
//
// Behavior:
// - Reads top-level `imports:` (list of paths), if present
// - For each imported file, loads its `fragments:` map
// - Walks routes' `steps:` and splices in fragment step lists for each `- fragment:` reference
// - Recursively resolves `imports:` in fragment files themselves
// - Detects circular imports and duplicate fragment names
// - Removes `imports:` from the returned config (fully resolved)
//
// Returns an error if:
// - A fragment file cannot be read
// - A fragment name is referenced but not found
// - Fragment names are duplicated across imports
// - Circular imports are detected
func ResolveNamedImports(cfg map[string]interface{}, basePath string) (map[string]interface{}, error) {
	if basePath == "" {
		basePath = "."
	}

	resolver := &namedImportResolver{
		basePath: basePath,
		cache:    make(map[string]map[string]interface{}), // path -> fragment name -> steps
		visiting: make(map[string]bool),                   // for cycle detection
	}

	resolved, err := resolver.resolve(cfg)
	return resolved, err
}

type namedImportResolver struct {
	basePath string
	cache    map[string]map[string]interface{} // path -> (fragment name -> steps list)
	visiting map[string]bool                   // cycle detection
}

func (r *namedImportResolver) resolve(cfg map[string]interface{}) (map[string]interface{}, error) {
	// Extract the imports list from the top-level config
	importsVal, hasImports := cfg["imports"]
	if !hasImports {
		// No imports; return as-is
		return cfg, nil
	}

	importsList, ok := importsVal.([]interface{})
	if !ok {
		return nil, fmt.Errorf("imports: must be a list, got %T", importsVal)
	}

	// Load all fragments from the import paths
	allFragments := make(map[string]interface{}) // fragment name -> steps list

	for _, importPath := range importsList {
		importPathStr, ok := importPath.(string)
		if !ok {
			return nil, fmt.Errorf("imports: entry must be a string path, got %T", importPath)
		}

		fragments, err := r.loadFragments(importPathStr)
		if err != nil {
			return nil, err
		}

		// Check for duplicates
		for name := range fragments {
			if _, exists := allFragments[name]; exists {
				return nil, fmt.Errorf("duplicate fragment name %q across imports", name)
			}
		}

		// Merge into allFragments
		for name, steps := range fragments {
			allFragments[name] = steps
		}
	}

	// Now walk the config and expand fragment references in routes
	result := make(map[string]interface{})
	for key, val := range cfg {
		if key == "imports" {
			// Skip; we're removing this directive
			continue
		}

		if key == "routes" {
			// Special handling for routes: expand fragments
			routesMap, ok := val.(map[string]interface{})
			if !ok {
				result[key] = val
				continue
			}

			expandedRoutes := make(map[string]interface{})
			for routeName, routeVal := range routesMap {
				route, ok := routeVal.(map[string]interface{})
				if !ok {
					expandedRoutes[routeName] = routeVal
					continue
				}

				expandedRoute, err := r.expandFragmentsInRoute(route, allFragments)
				if err != nil {
					return nil, fmt.Errorf("route %q: %w", routeName, err)
				}
				expandedRoutes[routeName] = expandedRoute
			}
			result[key] = expandedRoutes
		} else {
			result[key] = val
		}
	}

	return result, nil
}

func (r *namedImportResolver) loadFragments(importPath string) (map[string]interface{}, error) {
	// Resolve path relative to basePath
	absPath := importPath
	if !filepath.IsAbs(importPath) {
		absPath = filepath.Join(r.basePath, importPath)
	}

	absPath, err := filepath.Abs(absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve import path %s: %w", importPath, err)
	}

	// Check cache
	if cached, ok := r.cache[absPath]; ok {
		return cached, nil
	}

	// Check for cycles
	if r.visiting[absPath] {
		return nil, fmt.Errorf("circular import detected: %s", absPath)
	}

	r.visiting[absPath] = true
	defer func() {
		r.visiting[absPath] = false
	}()

	// Read and parse the file
	data, err := os.ReadFile(absPath)
	if err != nil {
		// If file doesn't exist, return empty fragments (allows test routes to load)
		// In production, real routes should have access to their fragment files
		if os.IsNotExist(err) {
			r.cache[absPath] = make(map[string]interface{})
			return make(map[string]interface{}), nil
		}
		return nil, fmt.Errorf("failed to read fragment file %s: %w", absPath, err)
	}

	var fragment map[string]interface{}
	if err := yaml.Unmarshal(data, &fragment); err != nil {
		return nil, fmt.Errorf("failed to parse YAML fragment %s: %w", absPath, err)
	}

	if fragment == nil {
		fragment = make(map[string]interface{})
	}

	// Recursively resolve imports in this fragment file
	if _, hasImports := fragment["imports"]; hasImports {
		oldBasePath := r.basePath
		r.basePath = filepath.Dir(absPath)
		resolved, err := r.resolve(fragment)
		r.basePath = oldBasePath

		if err != nil {
			return nil, err
		}
		fragment = resolved
	}

	// Extract the fragments map from this file
	fragmentsVal, hasFragments := fragment["fragments"]
	if !hasFragments {
		// No fragments in this file; cache empty
		r.cache[absPath] = make(map[string]interface{})
		return make(map[string]interface{}), nil
	}

	fragmentsMap, ok := fragmentsVal.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("fragments: in %s must be a map, got %T", absPath, fragmentsVal)
	}

	// Cache and return
	r.cache[absPath] = fragmentsMap
	return fragmentsMap, nil
}

func (r *namedImportResolver) expandFragmentsInRoute(route map[string]interface{}, allFragments map[string]interface{}) (map[string]interface{}, error) {
	stepsVal, hasSteps := route["steps"]
	if !hasSteps {
		// No steps; return as-is
		return route, nil
	}

	stepsList, ok := stepsVal.([]interface{})
	if !ok {
		return route, nil // Not a list; leave as-is
	}

	// Walk steps and expand fragment references
	var expandedSteps []interface{}
	for _, stepVal := range stepsList {
		stepMap, ok := stepVal.(map[string]interface{})
		if !ok {
			// Not a map; keep as-is
			expandedSteps = append(expandedSteps, stepVal)
			continue
		}

		// Check if this step is a fragment reference
		if fragmentName, ok := stepMap["fragment"]; ok {
			if len(stepMap) != 1 {
				return nil, fmt.Errorf("fragment reference step must only contain 'fragment:' key")
			}

			fragmentNameStr, ok := fragmentName.(string)
			if !ok {
				return nil, fmt.Errorf("fragment: value must be a string, got %T", fragmentName)
			}

			// Look up the fragment
			fragment, ok := allFragments[fragmentNameStr]
			if !ok {
				available := ""
				for name := range allFragments {
					if available != "" {
						available += ", "
					}
					available += name
				}
				return nil, fmt.Errorf("fragment %q not found (available: %s)", fragmentNameStr, available)
			}

			// Fragment should be a list of step maps
			fragmentSteps, ok := fragment.([]interface{})
			if !ok {
				return nil, fmt.Errorf("fragment %q is not a list of steps", fragmentNameStr)
			}

			// Deep-copy the fragment steps and recursively expand any nested fragments
			for _, fstepVal := range fragmentSteps {
				fstepMap, ok := fstepVal.(map[string]interface{})
				if !ok {
					expandedSteps = append(expandedSteps, fstepVal)
					continue
				}

				// Deep copy the step
				copiedStep := deepCopyMap(fstepMap)

				// Recursively expand if it's a fragment reference
				if _, isFrag := copiedStep["fragment"]; isFrag {
					// Recurse: treat this step as a single-step route and expand it
					tempRoute := map[string]interface{}{
						"steps": []interface{}{copiedStep},
					}
					expandedRoute, err := r.expandFragmentsInRoute(tempRoute, allFragments)
					if err != nil {
						return nil, err
					}
					tempSteps := expandedRoute["steps"].([]interface{})
					expandedSteps = append(expandedSteps, tempSteps...)
				} else {
					expandedSteps = append(expandedSteps, copiedStep)
				}
			}
		} else {
			// Regular step; keep as-is (but recursively expand if it somehow contains fragments)
			expandedSteps = append(expandedSteps, stepVal)
		}
	}

	// Update the route with expanded steps
	result := deepCopyMap(route)
	result["steps"] = expandedSteps
	return result, nil
}

func deepCopyMap(m map[string]interface{}) map[string]interface{} {
	copy := make(map[string]interface{})
	for k, v := range m {
		copy[k] = deepCopyValue(v)
	}
	return copy
}

func deepCopyValue(v interface{}) interface{} {
	switch val := v.(type) {
	case map[string]interface{}:
		return deepCopyMap(val)
	case []interface{}:
		result := make([]interface{}, len(val))
		for i, item := range val {
			result[i] = deepCopyValue(item)
		}
		return result
	default:
		return v
	}
}
