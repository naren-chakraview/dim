package config

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
)

// RouteVersion represents the computed version hash of a resolved route.
// It provides deterministic versioning for route configuration changes,
// enabling lineage tracking to detect when route definitions change.
type RouteVersion struct {
	// Hash is the SHA256 hash of the canonical route serialization (64-char hex string)
	Hash string

	// Canonical is the canonical JSON representation (for debugging, optional)
	Canonical string
}

// String returns the hex hash string
func (rv *RouteVersion) String() string {
	return rv.Hash
}

// ComputeRouteVersion computes a deterministic version hash for a resolved route.
//
// The hash is computed from the canonical JSON serialization of the route,
// which includes all route configuration except transient fields like timestamps.
//
// Key properties:
// - Deterministic: same route → same hash (across runs, machines)
// - Canonical: all keys sorted alphabetically at every level
// - Stable: only changes if the resolved route changes (not metadata/timestamps)
// - Clear: hash is human-readable 64-character hex string (SHA256)
//
// Excluded fields (not included in hash):
// - RouteVersion itself (computed field)
// - Any fields not explicitly included in the route structure
//
// The canonical format uses JSON with 2-space indentation and sorted keys.
func ComputeRouteVersion(route *RouteSpec) (*RouteVersion, error) {
	if route == nil {
		return nil, fmt.Errorf("route is nil")
	}

	// Convert route to a sorted map representation
	routeMap := routeToSortedMap(route)

	// Marshal to canonical JSON (indented for readability)
	canonical, err := canonicalJSON(routeMap)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal route to JSON: %w", err)
	}

	// Compute SHA256 hash
	hash := sha256.Sum256(canonical)
	hashHex := hex.EncodeToString(hash[:])

	return &RouteVersion{
		Hash:      hashHex,
		Canonical: string(canonical),
	}, nil
}

// routeToSortedMap converts a RouteSpec to a map with all keys sorted recursively.
// This ensures deterministic JSON serialization.
func routeToSortedMap(route *RouteSpec) map[string]interface{} {
	m := make(map[string]interface{})

	// Include all route fields in deterministic order
	m["from"] = route.From
	m["auth"] = route.Auth

	// Error path
	if route.ErrorPath != nil {
		errorPathMap := make(map[string]interface{})
		errorPathMap["target"] = route.ErrorPath.Target
		if route.ErrorPath.Retry != nil {
			retryMap := make(map[string]interface{})
			retryMap["backoff_ms"] = route.ErrorPath.Retry.BackoffMs
			retryMap["jitter_ms"] = route.ErrorPath.Retry.JitterMs
			retryMap["max_attempts"] = route.ErrorPath.Retry.MaxAttempts
			errorPathMap["retry"] = retryMap
		}
		m["error_path"] = errorPathMap
	}

	// Ordering
	if route.Ordering != "" {
		m["ordering"] = route.Ordering
	}

	// Retry policy at route level
	if route.Retry != nil {
		retryMap := make(map[string]interface{})
		retryMap["backoff_ms"] = route.Retry.BackoffMs
		retryMap["jitter_ms"] = route.Retry.JitterMs
		retryMap["max_attempts"] = route.Retry.MaxAttempts
		m["retry"] = retryMap
	}

	// Steps - convert each step to sorted map
	if len(route.Steps) > 0 {
		stepsSlice := make([]interface{}, len(route.Steps))
		for i, step := range route.Steps {
			stepsSlice[i] = stepToSortedMap(step)
		}
		m["steps"] = stepsSlice
	}

	return m
}

// stepToSortedMap converts a StepSpec to a map with all keys sorted.
func stepToSortedMap(step StepSpec) map[string]interface{} {
	m := make(map[string]interface{})

	if step.Authorize != nil {
		authMap := make(map[string]interface{})
		authMap["mode"] = step.Authorize.Mode
		if step.Authorize.Expr != "" {
			authMap["expr"] = step.Authorize.Expr
		}
		if len(step.Authorize.RequireRoles) > 0 {
			authMap["require_roles"] = step.Authorize.RequireRoles
		}
		m["authorize"] = authMap
	}

	if step.Filter != nil {
		filterMap := make(map[string]interface{})
		filterMap["expr"] = step.Filter.Expr
		m["filter"] = filterMap
	}

	if step.Idempotent != nil {
		idempotentMap := make(map[string]interface{})
		idempotentMap["key_expr"] = step.Idempotent.KeyExpr
		m["idempotent"] = idempotentMap
	}

	if step.Route != nil {
		routeMap := make(map[string]interface{})
		routeMap["expr"] = step.Route.Expr
		if len(step.Route.Cases) > 0 {
			casesMap := make(map[string]interface{})
			for k, v := range step.Route.Cases {
				casesMap[k] = map[string]interface{}{"target": v.Target}
			}
			routeMap["cases"] = casesMap
		}
		if step.Route.Default != "" {
			routeMap["default"] = step.Route.Default
		}
		m["route"] = routeMap
	}

	if step.Translate != nil {
		translateMap := make(map[string]interface{})
		translateMap["expr"] = step.Translate.Expr
		m["translate"] = translateMap
	}

	if step.Wiretap != nil {
		wiretapMap := make(map[string]interface{})
		wiretapMap["sink"] = step.Wiretap.Sink
		m["wiretap"] = wiretapMap
	}

	// Include extra fields (sorted keys)
	if len(step.Extra) > 0 {
		extraMap := make(map[string]interface{})
		for k, v := range step.Extra {
			extraMap[k] = v
		}
		m["extra"] = extraMap
	}

	return m
}

// canonicalJSON converts a map to canonical JSON with sorted keys.
// Returns JSON bytes with 2-space indentation and all keys sorted alphabetically at every level.
func canonicalJSON(obj interface{}) ([]byte, error) {
	// First, convert the object to a form where all maps have sorted keys
	canonical := sortKeys(obj)

	// Marshal to JSON with indentation
	return json.MarshalIndent(canonical, "", "  ")
}

// sortKeys recursively sorts all map keys in an object.
// Converts maps to a sortable form for deterministic JSON serialization.
func sortKeys(obj interface{}) interface{} {
	switch v := obj.(type) {
	case map[string]interface{}:
		// Create a new map and copy values, ensuring sorted output
		result := make(map[string]interface{})
		for k, val := range v {
			result[k] = sortKeys(val)
		}

		// To ensure JSON marshaling has sorted keys, we use a custom approach:
		// We'll return a sortedMap that implements the json.Marshaler interface
		return newSortedMap(result)

	case []interface{}:
		// Recursively sort keys in array elements
		result := make([]interface{}, len(v))
		for i, val := range v {
			result[i] = sortKeys(val)
		}
		return result

	default:
		return obj
	}
}

// sortedMap is a map wrapper that marshals to JSON with sorted keys.
type sortedMap struct {
	data map[string]interface{}
}

func newSortedMap(data map[string]interface{}) *sortedMap {
	return &sortedMap{data: data}
}

// MarshalJSON implements json.Marshaler for deterministic JSON output with sorted keys.
func (sm *sortedMap) MarshalJSON() ([]byte, error) {
	if len(sm.data) == 0 {
		return []byte("{}"), nil
	}

	// Sort keys
	keys := make([]string, 0, len(sm.data))
	for k := range sm.data {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Build JSON manually to control key order
	var buf []byte
	buf = append(buf, '{')

	for i, k := range keys {
		if i > 0 {
			buf = append(buf, ',')
		}

		// Marshal key as JSON string
		keyJSON, _ := json.Marshal(k)
		buf = append(buf, keyJSON...)
		buf = append(buf, ':')

		// Marshal value as JSON
		valJSON, _ := json.Marshal(sm.data[k])
		buf = append(buf, valJSON...)
	}

	buf = append(buf, '}')
	return buf, nil
}

// VerifyRouteVersionDeterminism verifies that calling ComputeRouteVersion multiple times
// on the same route produces identical hashes. This is a testing helper.
func VerifyRouteVersionDeterminism(route *RouteSpec, iterations int) (bool, string, error) {
	if iterations < 2 {
		iterations = 2
	}

	var firstHash string
	for i := 0; i < iterations; i++ {
		rv, err := ComputeRouteVersion(route)
		if err != nil {
			return false, "", fmt.Errorf("iteration %d failed: %w", i, err)
		}

		if i == 0 {
			firstHash = rv.Hash
		} else if rv.Hash != firstHash {
			return false, firstHash, fmt.Errorf("hash mismatch at iteration %d: expected %s, got %s", i, firstHash, rv.Hash)
		}
	}

	return true, firstHash, nil
}
