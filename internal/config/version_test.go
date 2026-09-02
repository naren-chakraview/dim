package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// TestComputeRouteVersionDeterminism verifies that identical routes produce identical hashes
func TestComputeRouteVersionDeterminism(t *testing.T) {
	// Create a simple route
	route := &RouteSpec{
		From: "webhook-in",
		Auth: "none",
		ErrorPath: &ErrorPathSpec{
			Target: "error-sink",
			Retry: &RetryPolicy{
				MaxAttempts: 3,
				BackoffMs:   100,
				JitterMs:    50,
			},
		},
		Steps: []StepSpec{
			{
				Filter: &FilterSpec{
					Expr: `body.status = "active"`,
				},
			},
			{
				Translate: &TranslateSpec{
					Expr: `{ id: body.id, name: body.name }`,
				},
			},
		},
	}

	// Compute version 3 times and verify all hashes are identical
	hashes := make([]string, 3)
	for i := 0; i < 3; i++ {
		rv, err := ComputeRouteVersion(route)
		if err != nil {
			t.Fatalf("iteration %d: failed to compute version: %v", i, err)
		}
		hashes[i] = rv.Hash
	}

	if hashes[0] != hashes[1] || hashes[1] != hashes[2] {
		t.Errorf("hashes not deterministic: %s, %s, %s", hashes[0], hashes[1], hashes[2])
	}
}

// TestComputeRouteVersionHashFormat verifies hash is hex string, 64 characters (SHA256)
func TestComputeRouteVersionHashFormat(t *testing.T) {
	route := &RouteSpec{
		From: "in",
		Auth: "none",
		ErrorPath: &ErrorPathSpec{
			Target: "out",
		},
		Steps: []StepSpec{},
	}

	rv, err := ComputeRouteVersion(route)
	if err != nil {
		t.Fatalf("failed to compute version: %v", err)
	}

	// Hash should be 64 characters (SHA256 hex)
	if len(rv.Hash) != 64 {
		t.Errorf("expected hash length 64, got %d", len(rv.Hash))
	}

	// Hash should be lowercase hex
	hexRegex := regexp.MustCompile(`^[a-f0-9]{64}$`)
	if !hexRegex.MatchString(rv.Hash) {
		t.Errorf("hash does not match hex format: %s", rv.Hash)
	}
}

// TestComputeRouteVersionChangeDetection verifies that any change to route produces different hash
func TestComputeRouteVersionChangeDetection(t *testing.T) {
	baseRoute := &RouteSpec{
		From: "webhook-in",
		Auth: "none",
		ErrorPath: &ErrorPathSpec{
			Target: "error-sink",
		},
		Steps: []StepSpec{
			{
				Filter: &FilterSpec{
					Expr: `body.status = "active"`,
				},
			},
		},
	}

	// Compute base hash
	baseRV, err := ComputeRouteVersion(baseRoute)
	if err != nil {
		t.Fatalf("failed to compute base version: %v", err)
	}

	tests := []struct {
		name            string
		modifyRoute     func(*RouteSpec)
		shouldChange    bool
	}{
		{
			name:         "change from source",
			modifyRoute:  func(r *RouteSpec) { r.From = "different-source" },
			shouldChange: true,
		},
		{
			name:         "change auth",
			modifyRoute:  func(r *RouteSpec) { r.Auth = "oauth2" },
			shouldChange: true,
		},
		{
			name:         "change error path target",
			modifyRoute:  func(r *RouteSpec) { r.ErrorPath.Target = "different-sink" },
			shouldChange: true,
		},
		{
			name:         "change filter expression",
			modifyRoute:  func(r *RouteSpec) { r.Steps[0].Filter.Expr = `body.status = "inactive"` },
			shouldChange: true,
		},
		{
			name: "add retry policy",
			modifyRoute: func(r *RouteSpec) {
				r.ErrorPath.Retry = &RetryPolicy{
					MaxAttempts: 5,
					BackoffMs:   200,
				}
			},
			shouldChange: true,
		},
		{
			name: "add step",
			modifyRoute: func(r *RouteSpec) {
				r.Steps = append(r.Steps, StepSpec{
					Translate: &TranslateSpec{
						Expr: `body`,
					},
				})
			},
			shouldChange: true,
		},
		{
			name: "change retry backoff",
			modifyRoute: func(r *RouteSpec) {
				if r.ErrorPath.Retry == nil {
					r.ErrorPath.Retry = &RetryPolicy{}
				}
				r.ErrorPath.Retry.BackoffMs = 500
			},
			shouldChange: true,
		},
		{
			name:         "add ordering",
			modifyRoute:  func(r *RouteSpec) { r.Ordering = "required" },
			shouldChange: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a copy of the base route
			modifiedRoute := &RouteSpec{
				From:     baseRoute.From,
				Auth:     baseRoute.Auth,
				Ordering: baseRoute.Ordering,
				Steps:    make([]StepSpec, len(baseRoute.Steps)),
			}
			if baseRoute.ErrorPath != nil {
				modifiedRoute.ErrorPath = &ErrorPathSpec{
					Target: baseRoute.ErrorPath.Target,
				}
				if baseRoute.ErrorPath.Retry != nil {
					modifiedRoute.ErrorPath.Retry = &RetryPolicy{
						MaxAttempts: baseRoute.ErrorPath.Retry.MaxAttempts,
						BackoffMs:   baseRoute.ErrorPath.Retry.BackoffMs,
						JitterMs:    baseRoute.ErrorPath.Retry.JitterMs,
					}
				}
			}
			if baseRoute.Retry != nil {
				modifiedRoute.Retry = &RetryPolicy{
					MaxAttempts: baseRoute.Retry.MaxAttempts,
					BackoffMs:   baseRoute.Retry.BackoffMs,
					JitterMs:    baseRoute.Retry.JitterMs,
				}
			}
			copy(modifiedRoute.Steps, baseRoute.Steps)

			// Deep copy steps
			for i, step := range baseRoute.Steps {
				if step.Filter != nil {
					modifiedRoute.Steps[i].Filter = &FilterSpec{Expr: step.Filter.Expr}
				}
				if step.Translate != nil {
					modifiedRoute.Steps[i].Translate = &TranslateSpec{Expr: step.Translate.Expr}
				}
				if step.Wiretap != nil {
					modifiedRoute.Steps[i].Wiretap = &WiretapSpec{Sink: step.Wiretap.Sink}
				}
				if step.Idempotent != nil {
					modifiedRoute.Steps[i].Idempotent = &IdempotentSpec{KeyExpr: step.Idempotent.KeyExpr}
				}
				if step.Route != nil {
					modifiedRoute.Steps[i].Route = &RouteStepSpec{
						Expr:    step.Route.Expr,
						Default: step.Route.Default,
						Cases:   step.Route.Cases,
					}
				}
				if step.Authorize != nil {
					modifiedRoute.Steps[i].Authorize = &AuthorizeSpec{
						Mode:         step.Authorize.Mode,
						Expr:         step.Authorize.Expr,
						RequireRoles: step.Authorize.RequireRoles,
					}
				}
			}

			// Apply modification
			tt.modifyRoute(modifiedRoute)

			// Compute hash
			modifiedRV, err := ComputeRouteVersion(modifiedRoute)
			if err != nil {
				t.Fatalf("failed to compute modified version: %v", err)
			}

			if tt.shouldChange {
				if baseRV.Hash == modifiedRV.Hash {
					t.Errorf("expected hash to change, but it remained %s", baseRV.Hash)
				}
			} else {
				if baseRV.Hash != modifiedRV.Hash {
					t.Errorf("expected hash to remain the same, but changed from %s to %s", baseRV.Hash, modifiedRV.Hash)
				}
			}
		})
	}
}

// TestComputeRouteVersionCanonicalFormat verifies canonical JSON is readable and complete
func TestComputeRouteVersionCanonicalFormat(t *testing.T) {
	route := &RouteSpec{
		From: "webhook-in",
		Auth: "none",
		ErrorPath: &ErrorPathSpec{
			Target: "error-sink",
			Retry: &RetryPolicy{
				MaxAttempts: 3,
				BackoffMs:   100,
				JitterMs:    50,
			},
		},
		Steps: []StepSpec{
			{
				Filter: &FilterSpec{
					Expr: `body.status = "active"`,
				},
			},
			{
				Translate: &TranslateSpec{
					Expr: `{ id: body.id }`,
				},
			},
		},
	}

	rv, err := ComputeRouteVersion(route)
	if err != nil {
		t.Fatalf("failed to compute version: %v", err)
	}

	// Verify canonical JSON contains expected fields
	expectedFields := []string{
		`"from"`,
		`"auth"`,
		`"error_path"`,
		`"target"`,
		`"retry"`,
		`"max_attempts"`,
		`"backoff_ms"`,
		`"jitter_ms"`,
		`"steps"`,
	}

	for _, field := range expectedFields {
		if !strings.Contains(rv.Canonical, field) {
			t.Errorf("expected canonical JSON to contain %q, but got:\n%s", field, rv.Canonical)
		}
	}

	// Verify canonical JSON is valid JSON
	var parsed interface{}
	err = json.Unmarshal([]byte(rv.Canonical), &parsed)
	if err != nil {
		t.Errorf("canonical JSON is not valid JSON: %v\nContent:\n%s", err, rv.Canonical)
	}
}

// TestComputeRouteVersionNilRoute verifies that nil route is handled gracefully
func TestComputeRouteVersionNilRoute(t *testing.T) {
	_, err := ComputeRouteVersion(nil)
	if err == nil {
		t.Errorf("expected error for nil route, got nil")
	}
}

// TestComputeRouteVersionSimpleRoute tests a minimal valid route
func TestComputeRouteVersionSimpleRoute(t *testing.T) {
	route := &RouteSpec{
		From: "in",
		Auth: "none",
		ErrorPath: &ErrorPathSpec{
			Target: "out",
		},
		Steps: []StepSpec{},
	}

	rv, err := ComputeRouteVersion(route)
	if err != nil {
		t.Fatalf("failed to compute version: %v", err)
	}

	if rv.Hash == "" {
		t.Errorf("expected non-empty hash")
	}

	if rv.Canonical == "" {
		t.Errorf("expected non-empty canonical JSON")
	}
}

// TestComputeRouteVersionComplexRoute tests a route with many steps and options
func TestComputeRouteVersionComplexRoute(t *testing.T) {
	route := &RouteSpec{
		From: "webhook-in",
		Auth: map[string]interface{}{
			"mode": "rbac",
			"rules": []map[string]interface{}{
				{"role": "admin", "allow": true},
			},
		},
		ErrorPath: &ErrorPathSpec{
			Target: "error-sink",
			Retry: &RetryPolicy{
				MaxAttempts: 5,
				BackoffMs:   200,
				JitterMs:    100,
			},
		},
		Retry: &RetryPolicy{
			MaxAttempts: 3,
			BackoffMs:   100,
		},
		Ordering: "required",
		Steps: []StepSpec{
			{
				Filter: &FilterSpec{
					Expr: `body.status = "active"`,
				},
			},
			{
				Idempotent: &IdempotentSpec{
					KeyExpr: `body.id`,
				},
			},
			{
				Translate: &TranslateSpec{
					Expr: `{ id: body.id, name: body.name }`,
				},
			},
			{
				Wiretap: &WiretapSpec{
					Sink: "audit-sink",
				},
			},
			{
				Authorize: &AuthorizeSpec{
					Mode:         "rbac",
					RequireRoles: []string{"admin", "developer"},
				},
			},
			{
				Route: &RouteStepSpec{
					Expr: `body.type`,
					Cases: map[string]RouteCase{
						"user":  {Target: "user-sink"},
						"order": {Target: "order-sink"},
					},
					Default: "default-sink",
				},
			},
		},
	}

	rv, err := ComputeRouteVersion(route)
	if err != nil {
		t.Fatalf("failed to compute complex route version: %v", err)
	}

	if rv.Hash == "" {
		t.Errorf("expected non-empty hash for complex route")
	}

	// Verify hash is stable
	rv2, _ := ComputeRouteVersion(route)
	if rv.Hash != rv2.Hash {
		t.Errorf("complex route hash not deterministic: %s vs %s", rv.Hash, rv2.Hash)
	}
}

// TestVerifyRouteVersionDeterminism tests the helper function for verification
func TestVerifyRouteVersionDeterminism(t *testing.T) {
	route := &RouteSpec{
		From: "in",
		Auth: "none",
		ErrorPath: &ErrorPathSpec{
			Target: "out",
		},
		Steps: []StepSpec{},
	}

	ok, hash, err := VerifyRouteVersionDeterminism(route, 5)
	if err != nil {
		t.Errorf("VerifyRouteVersionDeterminism failed: %v", err)
	}

	if !ok {
		t.Errorf("determinism verification failed")
	}

	if hash == "" {
		t.Errorf("expected non-empty hash from verification")
	}
}

// TestComputeRouteVersionWithoutErrorPathRetry tests route without error path retry
func TestComputeRouteVersionWithoutErrorPathRetry(t *testing.T) {
	route := &RouteSpec{
		From: "in",
		Auth: "none",
		ErrorPath: &ErrorPathSpec{
			Target: "out",
			// No retry
		},
		Steps: []StepSpec{},
	}

	rv, err := ComputeRouteVersion(route)
	if err != nil {
		t.Fatalf("failed to compute version: %v", err)
	}

	// Should still produce a valid hash
	if len(rv.Hash) != 64 {
		t.Errorf("expected 64-char hash, got %d", len(rv.Hash))
	}

	// Should be consistent
	rv2, _ := ComputeRouteVersion(route)
	if rv.Hash != rv2.Hash {
		t.Errorf("hash not deterministic")
	}
}

// TestComputeRouteVersionWithPartialRetry tests with only partial retry fields
func TestComputeRouteVersionWithPartialRetry(t *testing.T) {
	route1 := &RouteSpec{
		From: "in",
		Auth: "none",
		ErrorPath: &ErrorPathSpec{
			Target: "out",
			Retry: &RetryPolicy{
				MaxAttempts: 3,
				BackoffMs:   100,
				// JitterMs is 0
			},
		},
		Steps: []StepSpec{},
	}

	route2 := &RouteSpec{
		From: "in",
		Auth: "none",
		ErrorPath: &ErrorPathSpec{
			Target: "out",
			Retry: &RetryPolicy{
				MaxAttempts: 3,
				BackoffMs:   100,
				JitterMs:    0,
			},
		},
		Steps: []StepSpec{},
	}

	rv1, _ := ComputeRouteVersion(route1)
	rv2, _ := ComputeRouteVersion(route2)

	// Both should produce the same hash (zero values should be consistent)
	if rv1.Hash != rv2.Hash {
		t.Errorf("hash mismatch for routes with same retry config: %s vs %s", rv1.Hash, rv2.Hash)
	}
}

// TestLoadConfigComputesRouteVersions tests that LoadRouteConfig automatically computes route versions
func TestLoadConfigComputesRouteVersions(t *testing.T) {
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

	tmpFile := filepath.Join(t.TempDir(), "test_version.yaml")
	if err := os.WriteFile(tmpFile, []byte(yaml), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	config, err := LoadRouteConfig(tmpFile)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	// Verify route has a version computed
	route, ok := config.Routes["test"]
	if !ok {
		t.Fatalf("expected route 'test' not found")
	}

	if route.RouteVersion == "" {
		t.Errorf("expected RouteVersion to be populated, but got empty string")
	}

	// Verify version is a valid hash (64-char hex)
	hexRegex := regexp.MustCompile(`^[a-f0-9]{64}$`)
	if !hexRegex.MatchString(route.RouteVersion) {
		t.Errorf("RouteVersion does not match hex format: %s", route.RouteVersion)
	}
}

// TestLoadConfigRouteVersionConsistency tests that two identical routes get the same version
func TestLoadConfigRouteVersionConsistency(t *testing.T) {
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
  route1:
    from: in
    auth: none
    error_path:
      target: out
    steps:
      - translate:
          expr: '{ id: body.id }'
  route2:
    from: in
    auth: none
    error_path:
      target: out
    steps:
      - translate:
          expr: '{ id: body.id }'
`

	tmpFile := filepath.Join(t.TempDir(), "test_consistency.yaml")
	if err := os.WriteFile(tmpFile, []byte(yaml), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	config, err := LoadRouteConfig(tmpFile)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	// Both routes should have identical versions
	route1 := config.Routes["route1"]
	route2 := config.Routes["route2"]

	if route1.RouteVersion != route2.RouteVersion {
		t.Errorf("identical routes should have identical versions: %s vs %s", route1.RouteVersion, route2.RouteVersion)
	}
}
