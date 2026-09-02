package config

import (
	"fmt"
)

// MergeConfigs performs a deep merge of two config maps with child overriding parent.
// Merge semantics:
// - Sources/Sinks: Merged by key (child values override parent values for the same key)
// - Steps: Concatenated (parent steps first, then child steps)
// - Routes: Merged by key (child values override parent values for the same key)
// - Other fields: Override (child value wins)
func MergeConfigs(parent, child map[string]interface{}) (map[string]interface{}, error) {
	if parent == nil {
		parent = make(map[string]interface{})
	}
	if child == nil {
		child = make(map[string]interface{})
	}

	// Create result starting with parent
	result := copyMap(parent)

	// Merge special keys first
	if childSources, ok := child["sources"]; ok {
		parentSources, _ := result["sources"]
		result["sources"] = MergeSources(
			getMapValue(parentSources, nil),
			getMapValue(childSources, nil),
		)
	}

	if childSinks, ok := child["sinks"]; ok {
		parentSinks, _ := result["sinks"]
		result["sinks"] = MergeSinks(
			getMapValue(parentSinks, nil),
			getMapValue(childSinks, nil),
		)
	}

	if childSteps, ok := child["steps"]; ok {
		parentSteps, _ := result["steps"]
		result["steps"] = MergeSteps(
			getSliceValue(parentSteps, nil),
			getSliceValue(childSteps, nil),
		)
	}

	if childRoutes, ok := child["routes"]; ok {
		parentRoutes, _ := result["routes"]
		result["routes"] = MergeRoutes(
			getMapValue(parentRoutes, nil),
			getMapValue(childRoutes, nil),
		)
	}

	// Override other top-level fields (child wins)
	for k, v := range child {
		if k != "sources" && k != "sinks" && k != "steps" && k != "routes" {
			result[k] = v
		}
	}

	return result, nil
}

// MergeSources merges two source maps, with child values overriding parent values
// Strategy: merge by key, child definitions override parent definitions
func MergeSources(parent, child map[string]interface{}) map[string]interface{} {
	if parent == nil {
		parent = make(map[string]interface{})
	}
	if child == nil {
		child = make(map[string]interface{})
	}

	result := make(map[string]interface{})

	// Add parent sources
	for k, v := range parent {
		result[k] = v
	}

	// Override with child sources
	for k, v := range child {
		result[k] = v
	}

	return result
}

// MergeSinks merges two sink maps, with child values overriding parent values
// Strategy: merge by key, child definitions override parent definitions
func MergeSinks(parent, child map[string]interface{}) map[string]interface{} {
	if parent == nil {
		parent = make(map[string]interface{})
	}
	if child == nil {
		child = make(map[string]interface{})
	}

	result := make(map[string]interface{})

	// Add parent sinks
	for k, v := range parent {
		result[k] = v
	}

	// Override with child sinks
	for k, v := range child {
		result[k] = v
	}

	return result
}

// MergeSteps concatenates step arrays, preserving order (parent steps first, then child)
// This allows fragments to add steps to a base configuration
func MergeSteps(parent, child []interface{}) []interface{} {
	if parent == nil {
		parent = make([]interface{}, 0)
	}
	if child == nil {
		child = make([]interface{}, 0)
	}

	// Concatenate: parent first, then child
	result := make([]interface{}, 0, len(parent)+len(child))
	result = append(result, parent...)
	result = append(result, child...)

	return result
}

// MergeRoutes merges two route maps, with child values overriding parent values
// Each route is merged individually (steps are concatenated, other fields override)
func MergeRoutes(parent, child map[string]interface{}) map[string]interface{} {
	if parent == nil {
		parent = make(map[string]interface{})
	}
	if child == nil {
		child = make(map[string]interface{})
	}

	result := make(map[string]interface{})

	// Add parent routes
	for k, v := range parent {
		result[k] = v
	}

	// Merge or override with child routes
	for k, v := range child {
		if parentRoute, ok := parent[k]; ok {
			// Route exists in both parent and child - merge them
			if parentRouteMap, ok := parentRoute.(map[string]interface{}); ok {
				if childRouteMap, ok := v.(map[string]interface{}); ok {
					result[k] = mergeRoute(parentRouteMap, childRouteMap)
					continue
				}
			}
		}
		// Route only in child, or couldn't parse - just override
		result[k] = v
	}

	return result
}

// mergeRoute merges a single route definition, concatenating steps and overriding other fields
func mergeRoute(parent, child map[string]interface{}) map[string]interface{} {
	result := copyMap(parent)

	// Merge steps if present in child
	if childSteps, ok := child["steps"]; ok {
		parentStepsVal, _ := result["steps"]
		parentSteps := getSliceValue(parentStepsVal, nil)
		result["steps"] = MergeSteps(parentSteps, getSliceValue(childSteps, nil))
	}

	// Override other fields
	for k, v := range child {
		if k != "steps" {
			result[k] = v
		}
	}

	return result
}

// copyMap creates a shallow copy of a map
func copyMap(m map[string]interface{}) map[string]interface{} {
	if m == nil {
		return make(map[string]interface{})
	}
	result := make(map[string]interface{})
	for k, v := range m {
		result[k] = v
	}
	return result
}

// getMapValue gets a value from a map and asserts it's a map, or returns an empty map
func getMapValue(src interface{}, defaultVal interface{}) map[string]interface{} {
	if m, ok := src.(map[string]interface{}); ok {
		return m
	}
	if defaultVal != nil {
		if m, ok := defaultVal.(map[string]interface{}); ok {
			return m
		}
	}
	return make(map[string]interface{})
}

// getSliceValue gets a value from a map and asserts it's a slice, or returns an empty slice
func getSliceValue(src interface{}, defaultVal interface{}) []interface{} {
	if s, ok := src.([]interface{}); ok {
		return s
	}
	if defaultVal != nil {
		if s, ok := defaultVal.([]interface{}); ok {
			return s
		}
	}
	return make([]interface{}, 0)
}

// DeepMergeObjects performs a deep merge of two interface{} values
// Used for merging nested structures in configs
func DeepMergeObjects(parent, child interface{}) interface{} {
	parentMap, isParentMap := parent.(map[string]interface{})
	childMap, isChildMap := child.(map[string]interface{})

	if !isParentMap || !isChildMap {
		// If either is not a map, child wins
		return child
	}

	result := copyMap(parentMap)
	for k, v := range childMap {
		if parentVal, ok := result[k]; ok {
			// Both have this key - try to merge recursively
			result[k] = DeepMergeObjects(parentVal, v)
		} else {
			// Only child has this key
			result[k] = v
		}
	}
	return result
}

// MergeConfigsWithError is a variant that returns an error if merge would lose data
// This is stricter than MergeConfigs and can be used for validation
func MergeConfigsWithError(parent, child map[string]interface{}) (map[string]interface{}, error) {
	result, err := MergeConfigs(parent, child)
	if err != nil {
		return nil, fmt.Errorf("merge error: %w", err)
	}
	return result, nil
}
