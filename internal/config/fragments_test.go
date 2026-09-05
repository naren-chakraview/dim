package config

import (
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"
)

// TestLoadSingleFragment tests loading a single fragment file without imports
func TestLoadSingleFragment(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a simple fragment
	fragmentYAML := `
version: 1
sources:
  http-in:
    type: http
    path: /ingest
sinks:
  output:
    type: file
    path: /tmp/output.jsonl
routes:
  hello:
    from: http-in
    auth: none
    error_path:
      target: output
    steps:
      - filter:
          expr: 'body.status = "active"'
`

	fragmentPath := filepath.Join(tmpDir, "base.yaml")
	if err := os.WriteFile(fragmentPath, []byte(fragmentYAML), 0644); err != nil {
		t.Fatalf("failed to write fragment file: %v", err)
	}

	// Load the fragment
	resolver := NewFragmentResolver(tmpDir)
	data, err := os.ReadFile(fragmentPath)
	if err != nil {
		t.Fatalf("failed to read fragment: %v", err)
	}

	var config map[string]interface{}
	if err := yamlUnmarshal(data, &config); err != nil {
		t.Fatalf("failed to parse YAML: %v", err)
	}

	resolved, err := resolver.ResolveImports(config)
	if err != nil {
		t.Fatalf("failed to resolve imports: %v", err)
	}

	// Verify structure
	if version, ok := resolved["version"]; !ok || version != 1 {
		t.Errorf("expected version 1, got %v", version)
	}

	if sources, ok := resolved["sources"]; !ok {
		t.Errorf("expected sources in resolved config")
	} else {
		sourcesMap := sources.(map[string]interface{})
		if _, ok := sourcesMap["http-in"]; !ok {
			t.Errorf("expected http-in source")
		}
	}
}

// TestResolveImportSingleLevel tests resolving a single level of imports
func TestResolveImportSingleLevel(t *testing.T) {
	tmpDir := t.TempDir()

	// Create base fragment
	baseYAML := `
version: 1
sources:
  http-in:
    type: http
    path: /ingest
sinks:
  output:
    type: file
    path: /tmp/output.jsonl
routes:
  hello:
    from: http-in
    auth: none
    error_path:
      target: output
    steps:
      - filter:
          expr: 'body.status = "active"'
`

	basePath := filepath.Join(tmpDir, "base.yaml")
	if err := os.WriteFile(basePath, []byte(baseYAML), 0644); err != nil {
		t.Fatalf("failed to write base fragment: %v", err)
	}

	// Create a fragment that imports the base
	authYAML := `
$import: base.yaml
routes:
  hello:
    auth: none
    steps:
      - authorize:
          mode: rbac
          require_roles: [admin]
`

	authPath := filepath.Join(tmpDir, "auth.yaml")
	if err := os.WriteFile(authPath, []byte(authYAML), 0644); err != nil {
		t.Fatalf("failed to write auth fragment: %v", err)
	}

	// Load and resolve the auth fragment
	resolver := NewFragmentResolver(tmpDir)
	data, err := os.ReadFile(authPath)
	if err != nil {
		t.Fatalf("failed to read auth fragment: %v", err)
	}

	var config map[string]interface{}
	if err := yamlUnmarshal(data, &config); err != nil {
		t.Fatalf("failed to parse YAML: %v", err)
	}

	resolved, err := resolver.ResolveImports(config)
	if err != nil {
		t.Fatalf("failed to resolve imports: %v", err)
	}

	// Verify that both base sources/sinks and auth-specific steps are present
	if sources, ok := resolved["sources"]; !ok {
		t.Errorf("expected sources in resolved config")
	} else {
		sourcesMap := sources.(map[string]interface{})
		if _, ok := sourcesMap["http-in"]; !ok {
			t.Errorf("expected http-in source from base")
		}
	}

	if routes, ok := resolved["routes"]; !ok {
		t.Errorf("expected routes in resolved config")
	} else {
		routesMap := routes.(map[string]interface{})
		if helloRoute, ok := routesMap["hello"]; !ok {
			t.Errorf("expected hello route")
		} else {
			helloMap := helloRoute.(map[string]interface{})
			if steps, ok := helloMap["steps"]; ok {
				stepsSlice := steps.([]interface{})
				// Should have 2 steps: filter from base + authorize from auth
				if len(stepsSlice) != 2 {
					t.Errorf("expected 2 steps, got %d", len(stepsSlice))
				}
			} else {
				t.Errorf("expected steps in hello route")
			}
		}
	}
}

// TestResolveImportRecursive tests recursive import resolution (fragment importing fragment)
func TestResolveImportRecursive(t *testing.T) {
	tmpDir := t.TempDir()

	// Create base fragment
	baseYAML := `
version: 1
sources:
  http-in:
    type: http
    path: /ingest
sinks:
  output:
    type: file
    path: /tmp/output.jsonl
routes:
  hello:
    from: http-in
    auth: none
    error_path:
      target: output
    steps: []
`

	basePath := filepath.Join(tmpDir, "base.yaml")
	if err := os.WriteFile(basePath, []byte(baseYAML), 0644); err != nil {
		t.Fatalf("failed to write base fragment: %v", err)
	}

	// Create auth fragment that imports base
	authYAML := `
$import: base.yaml
routes:
  hello:
    auth: none
    steps:
      - authorize:
          mode: rbac
          require_roles: [admin]
`

	authPath := filepath.Join(tmpDir, "auth.yaml")
	if err := os.WriteFile(authPath, []byte(authYAML), 0644); err != nil {
		t.Fatalf("failed to write auth fragment: %v", err)
	}

	// Create retry fragment that imports auth
	retryYAML := `
$import: auth.yaml
routes:
  hello:
    auth: none
    steps:
      - translate:
          expr: 'body'
`

	retryPath := filepath.Join(tmpDir, "retry.yaml")
	if err := os.WriteFile(retryPath, []byte(retryYAML), 0644); err != nil {
		t.Fatalf("failed to write retry fragment: %v", err)
	}

	// Load and resolve the retry fragment (should recursively resolve through auth -> base)
	resolver := NewFragmentResolver(tmpDir)
	data, err := os.ReadFile(retryPath)
	if err != nil {
		t.Fatalf("failed to read retry fragment: %v", err)
	}

	var config map[string]interface{}
	if err := yamlUnmarshal(data, &config); err != nil {
		t.Fatalf("failed to parse YAML: %v", err)
	}

	resolved, err := resolver.ResolveImports(config)
	if err != nil {
		t.Fatalf("failed to resolve imports: %v", err)
	}

	// Verify that base sources/sinks are present and all steps are accumulated
	if sources, ok := resolved["sources"]; !ok {
		t.Errorf("expected sources in resolved config")
	} else {
		sourcesMap := sources.(map[string]interface{})
		if _, ok := sourcesMap["http-in"]; !ok {
			t.Errorf("expected http-in source from base")
		}
	}

	if routes, ok := resolved["routes"]; !ok {
		t.Errorf("expected routes in resolved config")
	} else {
		routesMap := routes.(map[string]interface{})
		if helloRoute, ok := routesMap["hello"]; !ok {
			t.Errorf("expected hello route")
		} else {
			helloMap := helloRoute.(map[string]interface{})
			if steps, ok := helloMap["steps"]; ok {
				stepsSlice := steps.([]interface{})
				// Should have 3 steps: authorize from auth + translate from retry
				// (base has no steps initially)
				if len(stepsSlice) != 2 {
					t.Errorf("expected 2 steps, got %d", len(stepsSlice))
				}
			}
		}
	}
}

// TestCycleDetection tests that circular imports are detected and rejected
func TestCycleDetection(t *testing.T) {
	tmpDir := t.TempDir()

	// Create fragment A that imports B
	fragmentAYAML := `
$import: fragment_b.yaml
version: 1
sources: {}
sinks: {}
routes: {}
`

	fragmentAPath := filepath.Join(tmpDir, "fragment_a.yaml")
	if err := os.WriteFile(fragmentAPath, []byte(fragmentAYAML), 0644); err != nil {
		t.Fatalf("failed to write fragment A: %v", err)
	}

	// Create fragment B that imports A (creating a cycle)
	fragmentBYAML := `
$import: fragment_a.yaml
version: 1
sources: {}
sinks: {}
routes: {}
`

	fragmentBPath := filepath.Join(tmpDir, "fragment_b.yaml")
	if err := os.WriteFile(fragmentBPath, []byte(fragmentBYAML), 0644); err != nil {
		t.Fatalf("failed to write fragment B: %v", err)
	}

	// Try to resolve - should detect cycle
	resolver := NewFragmentResolver(tmpDir)
	data, err := os.ReadFile(fragmentAPath)
	if err != nil {
		t.Fatalf("failed to read fragment A: %v", err)
	}

	var config map[string]interface{}
	if err := yamlUnmarshal(data, &config); err != nil {
		t.Fatalf("failed to parse YAML: %v", err)
	}

	_, err = resolver.ResolveImports(config)
	if err == nil {
		t.Errorf("expected error for circular import, got nil")
	}

	// Verify error message mentions cycle
	if err.Error() == "" || err.Error() == "nil" {
		t.Errorf("expected meaningful error message for cycle")
	}
}

// TestConfigMergingSourcesAndSinks tests that sources and sinks are merged by key
func TestConfigMergingSourcesAndSinks(t *testing.T) {
	parent := map[string]interface{}{
		"sources": map[string]interface{}{
			"http-in": map[string]interface{}{
				"type": "http",
				"path": "/old",
			},
		},
		"sinks": map[string]interface{}{
			"file-out": map[string]interface{}{
				"type": "file",
				"path": "/tmp/old.jsonl",
			},
		},
	}

	child := map[string]interface{}{
		"sources": map[string]interface{}{
			"http-in": map[string]interface{}{
				"type": "http",
				"path": "/new",
			},
			"kafka-in": map[string]interface{}{
				"type": "kafka",
				"topic": "events",
			},
		},
		"sinks": map[string]interface{}{
			"http-out": map[string]interface{}{
				"type": "http",
				"url": "http://example.com",
			},
		},
	}

	merged, err := MergeConfigs(parent, child)
	if err != nil {
		t.Fatalf("merge failed: %v", err)
	}

	// Check sources: http-in should be overridden, kafka-in should be added, file-out removed
	sourcesMap := merged["sources"].(map[string]interface{})
	if len(sourcesMap) != 2 {
		t.Errorf("expected 2 sources, got %d", len(sourcesMap))
	}

	if httpPath, ok := sourcesMap["http-in"].(map[string]interface{})["path"]; !ok || httpPath != "/new" {
		t.Errorf("expected http-in path to be overridden to /new")
	}

	if _, ok := sourcesMap["kafka-in"]; !ok {
		t.Errorf("expected kafka-in source to be added")
	}

	// Check sinks: file-out should be overridden (not removed, it's still there), http-out added
	sinksMap := merged["sinks"].(map[string]interface{})
	if len(sinksMap) != 2 {
		t.Errorf("expected 2 sinks, got %d", len(sinksMap))
	}

	if _, ok := sinksMap["file-out"]; !ok {
		t.Errorf("expected file-out sink to remain")
	}

	if _, ok := sinksMap["http-out"]; !ok {
		t.Errorf("expected http-out sink to be added")
	}
}

// TestConfigMergingSteps tests that steps are concatenated, not overridden
func TestConfigMergingSteps(t *testing.T) {
	parent := map[string]interface{}{
		"steps": []interface{}{
			map[string]interface{}{
				"filter": map[string]interface{}{
					"expr": `body.status = "active"`,
				},
			},
		},
	}

	child := map[string]interface{}{
		"steps": []interface{}{
			map[string]interface{}{
				"translate": map[string]interface{}{
					"expr": "body",
				},
			},
		},
	}

	merged, err := MergeConfigs(parent, child)
	if err != nil {
		t.Fatalf("merge failed: %v", err)
	}

	stepsSlice := merged["steps"].([]interface{})
	if len(stepsSlice) != 2 {
		t.Errorf("expected 2 steps, got %d", len(stepsSlice))
	}

	// Verify order: parent filter first, then child translate
	if filter, ok := stepsSlice[0].(map[string]interface{})["filter"]; !ok {
		t.Errorf("expected first step to be filter")
	} else {
		if expr, ok := filter.(map[string]interface{})["expr"]; !ok || expr != `body.status = "active"` {
			t.Errorf("expected filter expr to match parent")
		}
	}

	if translate, ok := stepsSlice[1].(map[string]interface{})["translate"]; !ok {
		t.Errorf("expected second step to be translate")
	} else {
		if expr, ok := translate.(map[string]interface{})["expr"]; !ok || expr != "body" {
			t.Errorf("expected translate expr to match child")
		}
	}
}

// TestConfigMergingOverride tests that non-special fields override (child wins)
func TestConfigMergingOverride(t *testing.T) {
	parent := map[string]interface{}{
		"version": 1,
		"custom":  "parent-value",
	}

	child := map[string]interface{}{
		"version": 1,
		"custom":  "child-value",
	}

	merged, err := MergeConfigs(parent, child)
	if err != nil {
		t.Fatalf("merge failed: %v", err)
	}

	if custom, ok := merged["custom"]; !ok || custom != "child-value" {
		t.Errorf("expected custom field to be overridden by child value")
	}
}

// TestRelativePathResolution tests that import paths are resolved relative to the importing file
func TestRelativePathResolution(t *testing.T) {
	tmpDir := t.TempDir()
	subDir := filepath.Join(tmpDir, "subdir")
	if err := os.Mkdir(subDir, 0755); err != nil {
		t.Fatalf("failed to create subdir: %v", err)
	}

	// Create base fragment in subdirectory
	baseYAML := `
version: 1
sources: {}
sinks: {}
routes: {}
`

	basePath := filepath.Join(subDir, "base.yaml")
	if err := os.WriteFile(basePath, []byte(baseYAML), 0644); err != nil {
		t.Fatalf("failed to write base fragment: %v", err)
	}

	// Create auth fragment in parent directory that imports relative path
	authYAML := `
$import: subdir/base.yaml
version: 1
sources: {}
sinks: {}
routes: {}
`

	authPath := filepath.Join(tmpDir, "auth.yaml")
	if err := os.WriteFile(authPath, []byte(authYAML), 0644); err != nil {
		t.Fatalf("failed to write auth fragment: %v", err)
	}

	// Resolve from the auth fragment location
	resolver := NewFragmentResolver(tmpDir)
	data, err := os.ReadFile(authPath)
	if err != nil {
		t.Fatalf("failed to read auth fragment: %v", err)
	}

	var config map[string]interface{}
	if err := yamlUnmarshal(data, &config); err != nil {
		t.Fatalf("failed to parse YAML: %v", err)
	}

	resolved, err := resolver.ResolveImports(config)
	if err != nil {
		t.Fatalf("failed to resolve imports: %v", err)
	}

	if version, ok := resolved["version"]; !ok || version != 1 {
		t.Errorf("expected resolved config to have version 1")
	}
}

// TestMissingFragmentFile tests that missing fragment files are handled gracefully
func TestMissingFragmentFile(t *testing.T) {
	tmpDir := t.TempDir()

	// Create fragment that imports non-existent file
	authYAML := `
$import: nonexistent.yaml
version: 1
sources: {}
sinks: {}
routes: {}
`

	authPath := filepath.Join(tmpDir, "auth.yaml")
	if err := os.WriteFile(authPath, []byte(authYAML), 0644); err != nil {
		t.Fatalf("failed to write auth fragment: %v", err)
	}

	// Try to resolve - should fail with meaningful error
	resolver := NewFragmentResolver(tmpDir)
	data, err := os.ReadFile(authPath)
	if err != nil {
		t.Fatalf("failed to read auth fragment: %v", err)
	}

	var config map[string]interface{}
	if err := yamlUnmarshal(data, &config); err != nil {
		t.Fatalf("failed to parse YAML: %v", err)
	}

	_, err = resolver.ResolveImports(config)
	if err == nil {
		t.Errorf("expected error for missing fragment file, got nil")
	}
}

// TestInvalidYAMLInFragment tests that invalid YAML in fragments is handled
func TestInvalidYAMLInFragment(t *testing.T) {
	tmpDir := t.TempDir()

	// Create fragment with invalid YAML (unclosed string/quote)
	authYAML := `
version: 1
sources:
  http-in:
    type: "unclosed string
sinks: {}
routes: {}
`

	authPath := filepath.Join(tmpDir, "auth.yaml")
	if err := os.WriteFile(authPath, []byte(authYAML), 0644); err != nil {
		t.Fatalf("failed to write auth fragment: %v", err)
	}

	// Try to resolve - should fail with meaningful error
	data, err := os.ReadFile(authPath)
	if err != nil {
		t.Fatalf("failed to read auth fragment: %v", err)
	}

	var config map[string]interface{}
	if err := yamlUnmarshal(data, &config); err == nil {
		t.Errorf("expected error for invalid YAML")
	}
}

// TestIntegrationLoadRouteConfigWithImports tests loading a route config with imports through the loader
func TestIntegrationLoadRouteConfigWithImports(t *testing.T) {
	tmpDir := t.TempDir()

	// Create base fragment
	baseYAML := `
version: 1
sources:
  http-in:
    type: http
    path: /ingest
    method: POST
sinks:
  output:
    type: file
    path: /tmp/output.jsonl
routes:
  hello:
    from: http-in
    auth: none
    error_path:
      target: output
      retry:
        max_attempts: 1
    steps:
      - filter:
          expr: 'body.status = "active"'
`

	basePath := filepath.Join(tmpDir, "base.yaml")
	if err := os.WriteFile(basePath, []byte(baseYAML), 0644); err != nil {
		t.Fatalf("failed to write base fragment: %v", err)
	}

	// Create route config that imports base
	routeYAML := `
$import: base.yaml
routes:
  hello:
    auth: none
    steps:
      - translate:
          expr: '{ message: body }'
`

	routePath := filepath.Join(tmpDir, "route.yaml")
	if err := os.WriteFile(routePath, []byte(routeYAML), 0644); err != nil {
		t.Fatalf("failed to write route config: %v", err)
	}

	// Load through the standard loader
	config, err := LoadRouteConfig(routePath)
	if err != nil {
		t.Fatalf("failed to load route config: %v", err)
	}

	// Verify merged configuration
	if config.Version != 1 {
		t.Errorf("expected version 1, got %d", config.Version)
	}

	if _, ok := config.Sources["http-in"]; !ok {
		t.Errorf("expected http-in source from base")
	}

	if _, ok := config.Routes["hello"]; !ok {
		t.Errorf("expected hello route")
	}

	route := config.Routes["hello"]
	// Should have 2 steps: filter from base + translate from route
	if len(route.Steps) != 2 {
		t.Errorf("expected 2 steps, got %d", len(route.Steps))
	}

	if route.Steps[0].Filter == nil || route.Steps[0].Filter.Expr != `body.status = "active"` {
		t.Errorf("expected first step to be filter from base")
	}

	if route.Steps[1].Translate == nil || route.Steps[1].Translate.Expr != `{ message: body }` {
		t.Errorf("expected second step to be translate from route")
	}
}

// TestFragmentEquivalenceRouteVersion verifies that a route composed from fragments
// produces the same RouteVersion hash as an equivalent route written inline
func TestFragmentEquivalenceRouteVersion(t *testing.T) {
	tmpDir := t.TempDir()

	// Route 1: Completely inline (no fragments)
	inlineYAML := `
version: 1
sources:
  http-in:
    type: http
sinks:
  output:
    type: file
    path: /tmp/output.jsonl
routes:
  main:
    from: http-in
    auth: none
    error_path:
      target: output
    steps:
      - filter:
          expr: 'body.active = true'
      - translate:
          expr: '{ id: body.id, name: body.name }'
`

	inlinePath := filepath.Join(tmpDir, "inline.yaml")
	if err := os.WriteFile(inlinePath, []byte(inlineYAML), 0644); err != nil {
		t.Fatalf("failed to write inline config: %v", err)
	}

	// Route 2: Composed from fragments with same content
	baseFragmentYAML := `
version: 1
sources:
  http-in:
    type: http
sinks:
  output:
    type: file
    path: /tmp/output.jsonl
routes:
  main:
    from: http-in
    auth: none
    error_path:
      target: output
    steps:
      - filter:
          expr: 'body.active = true'
      - translate:
          expr: '{ id: body.id, name: body.name }'
`

	baseFragPath := filepath.Join(tmpDir, "base_frag.yaml")
	if err := os.WriteFile(baseFragPath, []byte(baseFragmentYAML), 0644); err != nil {
		t.Fatalf("failed to write base fragment: %v", err)
	}

	composedYAML := `
$import: base_frag.yaml
`

	composedPath := filepath.Join(tmpDir, "composed.yaml")
	if err := os.WriteFile(composedPath, []byte(composedYAML), 0644); err != nil {
		t.Fatalf("failed to write composed config: %v", err)
	}

	// Load both configurations
	inlineConfig, err := LoadRouteConfig(inlinePath)
	if err != nil {
		t.Fatalf("failed to load inline config: %v", err)
	}

	composedConfig, err := LoadRouteConfig(composedPath)
	if err != nil {
		t.Fatalf("failed to load composed config: %v", err)
	}

	// Extract main route from both
	inlineRoute, ok := inlineConfig.Routes["main"]
	if !ok {
		t.Fatalf("main route not found in inline config")
	}

	composedRoute, ok := composedConfig.Routes["main"]
	if !ok {
		t.Fatalf("main route not found in composed config")
	}

	// Verify RouteVersion hashes are identical
	if inlineRoute.RouteVersion != composedRoute.RouteVersion {
		t.Errorf("RouteVersion mismatch: inline=%s, composed=%s",
			inlineRoute.RouteVersion, composedRoute.RouteVersion)
	}

	// Both should have non-empty RouteVersion
	if inlineRoute.RouteVersion == "" {
		t.Error("inline route RouteVersion is empty")
	}
	if composedRoute.RouteVersion == "" {
		t.Error("composed route RouteVersion is empty")
	}
}

// Helper function to unmarshal YAML
func yamlUnmarshal(data []byte, v interface{}) error {
	return yaml.Unmarshal(data, v)
}

// TestParameterSubstitutionScalar verifies scalar parameter substitution (M2.4.2)
func TestParameterSubstitutionScalar(t *testing.T) {
	tmpDir := t.TempDir()

	// Create fragment with parameters
	fragmentYAML := `$params:
  retries: 5
  backoff: 2.0

config:
  max_retries: ${PARAM:retries}
  backoff_multiplier: ${PARAM:backoff}
`
	fragmentPath := filepath.Join(tmpDir, "retry.yaml")
	os.WriteFile(fragmentPath, []byte(fragmentYAML), 0644)

	// Create route importing fragment with overrides
	routeYAML := `$import: retry.yaml
$params:
  retries: 10

routes:
  test:
    name: test
`
	routePath := filepath.Join(tmpDir, "route.yaml")
	os.WriteFile(routePath, []byte(routeYAML), 0644)

	fr := NewFragmentResolver(tmpDir)

	var route map[string]interface{}
	data, _ := os.ReadFile(routePath)
	yaml.Unmarshal(data, &route)

	resolved, err := fr.ResolveImports(route)
	if err != nil {
		t.Fatalf("Failed to resolve imports: %v", err)
	}

	// Verify parameters were substituted (as typed values: int 10, float64 2.0)
	config := resolved["config"].(map[string]interface{})
	maxRetries := config["max_retries"]
	switch v := maxRetries.(type) {
	case int:
		if v != 10 {
			t.Errorf("Expected max_retries=10 (overridden), got %d", v)
		}
	case float64:
		if v != 10 {
			t.Errorf("Expected max_retries=10 (overridden), got %v", v)
		}
	default:
		t.Errorf("Expected max_retries to be numeric, got %T: %v", maxRetries, maxRetries)
	}

	backoff := config["backoff_multiplier"]
	if backoffVal, ok := backoff.(float64); !ok || backoffVal != 2.0 {
		t.Errorf("Expected backoff_multiplier=2.0 (from fragment), got %v (type: %T)", backoff, backoff)
	}
}

// TestParameterUndefinedError verifies error on undefined parameter (M2.4.2)
func TestParameterUndefinedError(t *testing.T) {
	tmpDir := t.TempDir()

	// Fragment referencing undefined parameter
	fragmentYAML := `config:
  value: ${PARAM:undefined}
`
	fragmentPath := filepath.Join(tmpDir, "fragment.yaml")
	os.WriteFile(fragmentPath, []byte(fragmentYAML), 0644)

	// Route importing fragment
	routeYAML := `$import: fragment.yaml

routes:
  test:
    name: test
`
	routePath := filepath.Join(tmpDir, "route.yaml")
	os.WriteFile(routePath, []byte(routeYAML), 0644)

	fr := NewFragmentResolver(tmpDir)

	var route map[string]interface{}
	data, _ := os.ReadFile(routePath)
	yaml.Unmarshal(data, &route)

	_, err := fr.ResolveImports(route)
	if err == nil {
		t.Errorf("Expected error for undefined parameter, got none")
	}
	if err.Error() != "undefined parameter 'undefined'" {
		t.Errorf("Expected undefined parameter error, got: %v", err)
	}
}

// TestParameterOverride verifies route params override fragment defaults (M2.4.2)
func TestParameterOverride(t *testing.T) {
	tmpDir := t.TempDir()

	// Fragment with defaults
	fragmentYAML := `$params:
  maxRetries: 3
  timeout: 30

config:
  retries: ${PARAM:maxRetries}
  timeout_sec: ${PARAM:timeout}
`
	fragmentPath := filepath.Join(tmpDir, "policy.yaml")
	os.WriteFile(fragmentPath, []byte(fragmentYAML), 0644)

	// Route overriding parameters
	routeYAML := `$import: policy.yaml
$params:
  maxRetries: 10

routes:
  test:
    name: test
`
	routePath := filepath.Join(tmpDir, "route.yaml")
	os.WriteFile(routePath, []byte(routeYAML), 0644)

	fr := NewFragmentResolver(tmpDir)

	var route map[string]interface{}
	data, _ := os.ReadFile(routePath)
	yaml.Unmarshal(data, &route)

	resolved, err := fr.ResolveImports(route)
	if err != nil {
		t.Fatalf("Failed to resolve imports: %v", err)
	}

	config := resolved["config"].(map[string]interface{})
	retries := config["retries"]
	switch v := retries.(type) {
	case int:
		if v != 10 {
			t.Errorf("Expected retries=10 (overridden), got %d", v)
		}
	case float64:
		if v != 10 {
			t.Errorf("Expected retries=10 (overridden), got %v", v)
		}
	default:
		t.Errorf("Expected retries to be numeric, got %T: %v", retries, retries)
	}

	timeout := config["timeout_sec"]
	switch v := timeout.(type) {
	case int:
		if v != 30 {
			t.Errorf("Expected timeout_sec=30 (from fragment), got %d", v)
		}
	case float64:
		if v != 30 {
			t.Errorf("Expected timeout_sec=30 (from fragment), got %v", v)
		}
	default:
		t.Errorf("Expected timeout_sec to be numeric, got %T: %v", timeout, timeout)
	}
}

// TestParameterPartialSubstitution verifies partial string substitution (M2.4.2)
func TestParameterPartialSubstitution(t *testing.T) {
	tmpDir := t.TempDir()

	// Fragment with embedded parameters
	fragmentYAML := `$params:
  env: "production"
  version: "v2"

config:
  topic: "orders-${PARAM:env}-${PARAM:version}"
  description: "Processing ${PARAM:env} orders with ${PARAM:version}"
`
	fragmentPath := filepath.Join(tmpDir, "config.yaml")
	os.WriteFile(fragmentPath, []byte(fragmentYAML), 0644)

	// Route
	routeYAML := `$import: config.yaml
$params:
  env: "staging"

routes:
  test:
    name: test
`
	routePath := filepath.Join(tmpDir, "route.yaml")
	os.WriteFile(routePath, []byte(routeYAML), 0644)

	fr := NewFragmentResolver(tmpDir)

	var route map[string]interface{}
	data, _ := os.ReadFile(routePath)
	yaml.Unmarshal(data, &route)

	resolved, err := fr.ResolveImports(route)
	if err != nil {
		t.Fatalf("Failed to resolve imports: %v", err)
	}

	config := resolved["config"].(map[string]interface{})
	if topic := config["topic"]; topic != "orders-staging-v2" {
		t.Errorf("Expected topic='orders-staging-v2', got %v", topic)
	}
	if desc := config["description"]; desc != "Processing staging orders with v2" {
		t.Errorf("Expected description='Processing staging orders with v2', got %v", desc)
	}
}
