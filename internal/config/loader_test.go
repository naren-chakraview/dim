package config

import (
	"os"
	"path/filepath"
	"testing"
)

// TestLoadValidRouteConfig tests loading a valid YAML route configuration
func TestLoadValidRouteConfig(t *testing.T) {
	yaml := `
version: 1
sources:
  webhook-in:
    type: http
    path: /ingest
    method: POST
sinks:
  output:
    type: file
    path: /tmp/output.jsonl
routes:
  hello:
    from: webhook-in
    auth: none
    error_path:
      target: output
      retry:
        max_attempts: 1
    steps:
      - translate:
          expr: '{ message: body }'
`

	// Write to temp file
	tmpFile := filepath.Join(t.TempDir(), "test_valid.yaml")
	if err := os.WriteFile(tmpFile, []byte(yaml), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	// Load config
	config, err := LoadRouteConfig(tmpFile)
	if err != nil {
		t.Fatalf("failed to load valid config: %v", err)
	}

	// Verify structure
	if config.Version != 1 {
		t.Errorf("expected version 1, got %d", config.Version)
	}

	if _, ok := config.Sources["webhook-in"]; !ok {
		t.Errorf("expected source 'webhook-in' not found")
	}

	if config.Sources["webhook-in"].Type != "http" {
		t.Errorf("expected source type 'http', got %s", config.Sources["webhook-in"].Type)
	}

	if _, ok := config.Sinks["output"]; !ok {
		t.Errorf("expected sink 'output' not found")
	}

	if _, ok := config.Routes["hello"]; !ok {
		t.Errorf("expected route 'hello' not found")
	}

	route := config.Routes["hello"]
	if route.From != "webhook-in" {
		t.Errorf("expected route.from to be 'webhook-in', got %s", route.From)
	}

	if route.Auth != "none" {
		t.Errorf("expected route.auth to be 'none', got %v", route.Auth)
	}

	if route.ErrorPath.Target != "output" {
		t.Errorf("expected error_path.target to be 'output', got %s", route.ErrorPath.Target)
	}

	if route.ErrorPath.Retry == nil || route.ErrorPath.Retry.MaxAttempts != 1 {
		t.Errorf("expected retry max_attempts to be 1")
	}

	if len(route.Steps) != 1 {
		t.Errorf("expected 1 step, got %d", len(route.Steps))
	}

	if route.Steps[0].Translate == nil || route.Steps[0].Translate.Expr != "{ message: body }" {
		t.Errorf("expected translate step with correct expression")
	}
}

// TestLoadConfigMissingVersion tests that missing version is rejected
func TestLoadConfigMissingVersion(t *testing.T) {
	yaml := `
sources:
  in:
    type: http
sinks:
  out:
    type: file
    path: /tmp/out
routes:
  test:
    from: in
    auth: none
    error_path:
      target: out
    steps: []
`

	tmpFile := filepath.Join(t.TempDir(), "test_no_version.yaml")
	if err := os.WriteFile(tmpFile, []byte(yaml), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	_, err := LoadRouteConfig(tmpFile)
	if err == nil {
		t.Errorf("expected error for missing version, got nil")
	}
}

// TestLoadConfigWrongVersion tests that version other than 1 is rejected
func TestLoadConfigWrongVersion(t *testing.T) {
	yaml := `
version: 2
sources:
  in:
    type: http
sinks:
  out:
    type: file
    path: /tmp/out
routes:
  test:
    from: in
    auth: none
    error_path:
      target: out
    steps: []
`

	tmpFile := filepath.Join(t.TempDir(), "test_wrong_version.yaml")
	if err := os.WriteFile(tmpFile, []byte(yaml), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	_, err := LoadRouteConfig(tmpFile)
	if err == nil {
		t.Errorf("expected error for version 2, got nil")
	}
}

// TestLoadConfigMissingSources tests that missing sources is rejected
func TestLoadConfigMissingSources(t *testing.T) {
	yaml := `
version: 1
sinks:
  out:
    type: file
    path: /tmp/out
routes:
  test:
    from: in
    auth: none
    error_path:
      target: out
    steps: []
`

	tmpFile := filepath.Join(t.TempDir(), "test_no_sources.yaml")
	if err := os.WriteFile(tmpFile, []byte(yaml), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	_, err := LoadRouteConfig(tmpFile)
	if err == nil {
		t.Errorf("expected error for missing sources, got nil")
	}
}

// TestLoadConfigMissingSinks tests that missing sinks is rejected
func TestLoadConfigMissingSinks(t *testing.T) {
	yaml := `
version: 1
sources:
  in:
    type: http
routes:
  test:
    from: in
    auth: none
    error_path:
      target: out
    steps: []
`

	tmpFile := filepath.Join(t.TempDir(), "test_no_sinks.yaml")
	if err := os.WriteFile(tmpFile, []byte(yaml), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	_, err := LoadRouteConfig(tmpFile)
	if err == nil {
		t.Errorf("expected error for missing sinks, got nil")
	}
}

// TestLoadConfigMissingRoutes tests that missing routes is rejected
func TestLoadConfigMissingRoutes(t *testing.T) {
	yaml := `
version: 1
sources:
  in:
    type: http
sinks:
  out:
    type: file
    path: /tmp/out
`

	tmpFile := filepath.Join(t.TempDir(), "test_no_routes.yaml")
	if err := os.WriteFile(tmpFile, []byte(yaml), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	_, err := LoadRouteConfig(tmpFile)
	if err == nil {
		t.Errorf("expected error for missing routes, got nil")
	}
}

// TestLoadConfigMissingAuth tests that missing auth is rejected
func TestLoadConfigMissingAuth(t *testing.T) {
	yaml := `
version: 1
sources:
  in:
    type: http
sinks:
  out:
    type: file
    path: /tmp/out
routes:
  test:
    from: in
    error_path:
      target: out
    steps: []
`

	tmpFile := filepath.Join(t.TempDir(), "test_no_auth.yaml")
	if err := os.WriteFile(tmpFile, []byte(yaml), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	_, err := LoadRouteConfig(tmpFile)
	if err == nil {
		t.Errorf("expected error for missing auth, got nil")
	}
}

// TestLoadConfigMissingErrorPath tests that missing error_path is rejected
func TestLoadConfigMissingErrorPath(t *testing.T) {
	yaml := `
version: 1
sources:
  in:
    type: http
sinks:
  out:
    type: file
    path: /tmp/out
routes:
  test:
    from: in
    auth: none
    steps: []
`

	tmpFile := filepath.Join(t.TempDir(), "test_no_error_path.yaml")
	if err := os.WriteFile(tmpFile, []byte(yaml), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	_, err := LoadRouteConfig(tmpFile)
	if err == nil {
		t.Errorf("expected error for missing error_path, got nil")
	}
}

// TestLoadConfigMissingErrorPathTarget tests that missing error_path target is rejected
func TestLoadConfigMissingErrorPathTarget(t *testing.T) {
	yaml := `
version: 1
sources:
  in:
    type: http
sinks:
  out:
    type: file
    path: /tmp/out
routes:
  test:
    from: in
    auth: none
    error_path:
      retry:
        max_attempts: 1
    steps: []
`

	tmpFile := filepath.Join(t.TempDir(), "test_no_error_target.yaml")
	if err := os.WriteFile(tmpFile, []byte(yaml), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	_, err := LoadRouteConfig(tmpFile)
	if err == nil {
		t.Errorf("expected error for missing error_path target, got nil")
	}
}

// TestLoadConfigMultipleSteps tests loading config with multiple valid steps
func TestLoadConfigMultipleSteps(t *testing.T) {
	yaml := `
version: 1
sources:
  in:
    type: http
sinks:
  out:
    type: file
    path: /tmp/out
routes:
  test:
    from: in
    auth: none
    error_path:
      target: out
    steps:
      - filter:
          expr: 'body.status = "active"'
      - translate:
          expr: '{ id: body.id, name: body.name }'
`

	tmpFile := filepath.Join(t.TempDir(), "test_multi_steps.yaml")
	if err := os.WriteFile(tmpFile, []byte(yaml), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	config, err := LoadRouteConfig(tmpFile)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	route := config.Routes["test"]
	if len(route.Steps) != 2 {
		t.Errorf("expected 2 steps, got %d", len(route.Steps))
	}

	if route.Steps[0].Filter == nil || route.Steps[0].Filter.Expr != `body.status = "active"` {
		t.Errorf("expected first step to be filter")
	}

	if route.Steps[1].Translate == nil || route.Steps[1].Translate.Expr != `{ id: body.id, name: body.name }` {
		t.Errorf("expected second step to be translate")
	}
}

// TestLoadConfigWithAuthObject tests loading config with auth as an object
func TestLoadConfigWithAuthObject(t *testing.T) {
	yaml := `
version: 1
sources:
  in:
    type: http
sinks:
  out:
    type: file
    path: /tmp/out
routes:
  test:
    from: in
    auth:
      mode: rbac
      rules:
        - role: admin
          allow: true
    error_path:
      target: out
    steps:
      - authorize:
          mode: rbac
          rules:
            - role: admin
`

	tmpFile := filepath.Join(t.TempDir(), "test_auth_object.yaml")
	if err := os.WriteFile(tmpFile, []byte(yaml), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	config, err := LoadRouteConfig(tmpFile)
	if err != nil {
		t.Fatalf("failed to load config with auth object: %v", err)
	}

	route := config.Routes["test"]
	if route.Auth == nil {
		t.Errorf("expected auth to be non-nil")
	}

	// Verify auth is a map (parsed from YAML as an object)
	authMap, ok := route.Auth.(map[string]interface{})
	if !ok {
		t.Errorf("expected auth to be a map, got %T", route.Auth)
	}

	if authMap["mode"] != "rbac" {
		t.Errorf("expected auth mode to be 'rbac', got %v", authMap["mode"])
	}
}

// TestLoadConfigWithWiretapStep tests loading config with wiretap step
func TestLoadConfigWithWiretapStep(t *testing.T) {
	yaml := `
version: 1
sources:
  in:
    type: http
sinks:
  out:
    type: file
    path: /tmp/out
  audit:
    type: file
    path: /tmp/audit
routes:
  test:
    from: in
    auth: none
    error_path:
      target: out
    steps:
      - wiretap:
          sink: audit
      - translate:
          expr: 'body'
`

	tmpFile := filepath.Join(t.TempDir(), "test_wiretap.yaml")
	if err := os.WriteFile(tmpFile, []byte(yaml), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	config, err := LoadRouteConfig(tmpFile)
	if err != nil {
		t.Fatalf("failed to load config with wiretap: %v", err)
	}

	route := config.Routes["test"]
	if len(route.Steps) != 2 {
		t.Errorf("expected 2 steps, got %d", len(route.Steps))
	}

	if route.Steps[0].Wiretap == nil || route.Steps[0].Wiretap.Sink != "audit" {
		t.Errorf("expected first step to be wiretap with sink 'audit'")
	}
}

// TestLoadConfigWithIdempotentStep tests loading config with idempotent step
func TestLoadConfigWithIdempotentStep(t *testing.T) {
	yaml := `
version: 1
sources:
  in:
    type: http
sinks:
  out:
    type: file
    path: /tmp/out
routes:
  test:
    from: in
    auth: none
    error_path:
      target: out
    steps:
      - idempotent:
          key_expr: 'body.id'
      - translate:
          expr: 'body'
`

	tmpFile := filepath.Join(t.TempDir(), "test_idempotent.yaml")
	if err := os.WriteFile(tmpFile, []byte(yaml), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	config, err := LoadRouteConfig(tmpFile)
	if err != nil {
		t.Fatalf("failed to load config with idempotent: %v", err)
	}

	route := config.Routes["test"]
	if len(route.Steps) != 2 {
		t.Errorf("expected 2 steps, got %d", len(route.Steps))
	}

	if route.Steps[0].Idempotent == nil || route.Steps[0].Idempotent.KeyExpr != "body.id" {
		t.Errorf("expected first step to be idempotent with key_expr 'body.id'")
	}
}

// TestLoadConfigFileNotFound tests that missing file is handled
func TestLoadConfigFileNotFound(t *testing.T) {
	_, err := LoadRouteConfig("/nonexistent/path/config.yaml")
	if err == nil {
		t.Errorf("expected error for missing file, got nil")
	}
}

// TestLoadConfigInvalidYAML tests that invalid YAML is rejected
func TestLoadConfigInvalidYAML(t *testing.T) {
	yaml := `
version: 1
sources:
  in:
    type: http
    - invalid list item in object
sinks:
  out:
    type: file
`

	tmpFile := filepath.Join(t.TempDir(), "test_invalid_yaml.yaml")
	if err := os.WriteFile(tmpFile, []byte(yaml), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	_, err := LoadRouteConfig(tmpFile)
	if err == nil {
		t.Errorf("expected error for invalid YAML, got nil")
	}
}
