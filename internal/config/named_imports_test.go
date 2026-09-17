package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveNamedImports_SingleImport(t *testing.T) {
	// Test: a route that imports a fragment file and references a fragment by name
	// should have the fragment steps spliced into its steps list

	tmpDir := t.TempDir()

	// Create fragment file
	fragmentFile := filepath.Join(tmpDir, "fragments.yaml")
	fragmentContent := `version: 1
fragments:
  governance-baseline:
    - authorize:
        mode: rbac
        require_roles: [admin]
    - filter:
        expr: "true"
`
	if err := os.WriteFile(fragmentFile, []byte(fragmentContent), 0644); err != nil {
		t.Fatalf("Failed to write fragment file: %v", err)
	}

	// Create route config that imports and references the fragment
	cfg := map[string]interface{}{
		"version": 1,
		"imports": []interface{}{
			"fragments.yaml",
		},
		"routes": map[string]interface{}{
			"test": map[string]interface{}{
				"from":  "input",
				"auth":  "none",
				"steps": []interface{}{
					map[string]interface{}{
						"fragment": "governance-baseline",
					},
					map[string]interface{}{
						"translate": map[string]interface{}{
							"expr": "body | {id}",
						},
					},
				},
			},
		},
	}

	// Resolve named imports
	resolved, err := ResolveNamedImports(cfg, tmpDir)
	if err != nil {
		t.Fatalf("ResolveNamedImports failed: %v", err)
	}

	// Verify imports are removed
	if _, hasImports := resolved["imports"]; hasImports {
		t.Errorf("Expected imports to be removed, but it's still present")
	}

	// Verify route steps were expanded
	routes := resolved["routes"].(map[string]interface{})
	testRoute := routes["test"].(map[string]interface{})
	steps := testRoute["steps"].([]interface{})

	// Should have 3 steps: 2 from governance-baseline + 1 original translate
	if len(steps) != 3 {
		t.Errorf("Expected 3 steps after expansion, got %d", len(steps))
	}

	// First step should be the authorize from governance-baseline
	firstStep := steps[0].(map[string]interface{})
	if _, hasAuthorize := firstStep["authorize"]; !hasAuthorize {
		t.Errorf("Expected first step to be authorize from fragment, got: %v", firstStep)
	}

	// Last step should be the original translate
	lastStep := steps[2].(map[string]interface{})
	if _, hasTranslate := lastStep["translate"]; !hasTranslate {
		t.Errorf("Expected last step to be translate, got: %v", lastStep)
	}
}

func TestResolveNamedImports_UnknownFragment(t *testing.T) {
	// Test: referencing a non-existent fragment should return error

	tmpDir := t.TempDir()

	cfg := map[string]interface{}{
		"version": 1,
		"imports": []interface{}{
			"fragments.yaml", // This file doesn't exist, so no fragments available
		},
		"routes": map[string]interface{}{
			"test": map[string]interface{}{
				"from":  "input",
				"auth":  "none",
				"steps": []interface{}{
					map[string]interface{}{
						"fragment": "nonexistent", // References a fragment that doesn't exist
					},
				},
			},
		},
	}

	_, err := ResolveNamedImports(cfg, tmpDir)
	if err == nil {
		t.Errorf("Expected error for non-existent fragment, got nil")
	}
	if err != nil && err.Error() != "route \"test\": fragment \"nonexistent\" not found (available: )" {
		// Check if error contains the expected message (may vary slightly)
		if !containsErrorString(err.Error(), "nonexistent") || !containsErrorString(err.Error(), "not found") {
			t.Errorf("Expected error about missing fragment, got: %v", err)
		}
	}
}

func TestResolveNamedImports_NoImports(t *testing.T) {
	// Test: config without imports should be returned as-is

	cfg := map[string]interface{}{
		"version": 1,
		"routes": map[string]interface{}{
			"test": map[string]interface{}{
				"from": "input",
				"auth": "none",
			},
		},
	}

	resolved, err := ResolveNamedImports(cfg, ".")
	if err != nil {
		t.Fatalf("ResolveNamedImports failed: %v", err)
	}

	if resolved["version"] != cfg["version"] {
		t.Errorf("Config was modified unexpectedly")
	}
}

func TestResolveNamedImports_DuplicateFragmentNames(t *testing.T) {
	// Test: importing fragment files with duplicate fragment names should error

	tmpDir := t.TempDir()

	// Create first fragment file with governance-baseline
	fragment1 := filepath.Join(tmpDir, "fragments1.yaml")
	if err := os.WriteFile(fragment1, []byte(`version: 1
fragments:
  governance-baseline:
    - authorize:
        mode: rbac
`), 0644); err != nil {
		t.Fatalf("Failed to write fragment1: %v", err)
	}

	// Create second fragment file with same name
	fragment2 := filepath.Join(tmpDir, "fragments2.yaml")
	if err := os.WriteFile(fragment2, []byte(`version: 1
fragments:
  governance-baseline:
    - filter:
        expr: "true"
`), 0644); err != nil {
		t.Fatalf("Failed to write fragment2: %v", err)
	}

	cfg := map[string]interface{}{
		"version": 1,
		"imports": []interface{}{
			"fragments1.yaml",
			"fragments2.yaml",
		},
		"routes": map[string]interface{}{},
	}

	_, err := ResolveNamedImports(cfg, tmpDir)
	if err == nil {
		t.Errorf("Expected error for duplicate fragment names, got nil")
	}
	if err != nil && !containsErrorString(err.Error(), "duplicate") {
		t.Errorf("Expected duplicate fragment error, got: %v", err)
	}
}

func TestResolveNamedImports_CircularImports(t *testing.T) {
	// Test: circular imports should be detected

	tmpDir := t.TempDir()

	// Create file1 that imports file2
	file1 := filepath.Join(tmpDir, "file1.yaml")
	if err := os.WriteFile(file1, []byte(`version: 1
imports:
  - file2.yaml
fragments:
  fragment1:
    - filter:
        expr: "true"
`), 0644); err != nil {
		t.Fatalf("Failed to write file1: %v", err)
	}

	// Create file2 that imports file1 (cycle!)
	file2 := filepath.Join(tmpDir, "file2.yaml")
	if err := os.WriteFile(file2, []byte(`version: 1
imports:
  - file1.yaml
fragments:
  fragment2:
    - filter:
        expr: "false"
`), 0644); err != nil {
		t.Fatalf("Failed to write file2: %v", err)
	}

	cfg := map[string]interface{}{
		"version": 1,
		"imports": []interface{}{
			"file1.yaml",
		},
		"routes": map[string]interface{}{},
	}

	_, err := ResolveNamedImports(cfg, tmpDir)
	if err == nil {
		t.Errorf("Expected error for circular import, got nil")
	}
	if err != nil && !containsErrorString(err.Error(), "circular") {
		t.Errorf("Expected circular import error, got: %v", err)
	}
}

func TestResolveNamedImports_MissingFragmentFile(t *testing.T) {
	// Test: missing fragment file (not found) should be handled gracefully
	// (in tests, missing files return empty fragment maps; in production, this may error)

	cfg := map[string]interface{}{
		"version": 1,
		"imports": []interface{}{
			"nonexistent-file.yaml",
		},
		"routes": map[string]interface{}{
			"test": map[string]interface{}{
				"from":  "input",
				"auth":  "none",
				"steps": []interface{}{},
			},
		},
	}

	// In the current implementation, missing files are silently treated as having no fragments
	// This allows test routes to load without requiring all fragment files
	resolved, err := ResolveNamedImports(cfg, t.TempDir())
	if err != nil {
		t.Logf("Note: missing fragment file returned error: %v (this is acceptable)", err)
	} else {
		// If no error, the import was treated as empty
		if _, hasImports := resolved["imports"]; hasImports {
			t.Errorf("imports should be removed even with missing file")
		}
	}
}

// Helper function to check if error message contains a string
func containsErrorString(err, substr string) bool {
	for i := 0; i+len(substr) <= len(err); i++ {
		if err[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
