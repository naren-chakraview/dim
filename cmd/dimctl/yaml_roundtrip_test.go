package main

import (
	"strings"
	"testing"
)

func TestRoundTripPreservesComments(t *testing.T) {
	yaml := `# This is a comment
version: 1
# Route comment
routes:
  my-route:  # inline comment
    from: source
`

	data, comments, err := ParseYAMLWithComments(yaml)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if data["version"] != 1 {
		t.Errorf("version mismatch: got %v", data["version"])
	}

	// Reconstruct YAML
	reconstructed, err := ReconstructYAML(data, comments)
	if err != nil {
		t.Fatalf("reconstruct failed: %v", err)
	}

	// Comments should be preserved
	if !strings.Contains(reconstructed, "# This is a comment") {
		t.Error("top-level comment lost")
	}
	if !strings.Contains(reconstructed, "# inline comment") {
		t.Error("inline comment lost")
	}
}

func TestParseYAMLWithComments(t *testing.T) {
	yaml := `version: 1
sources:
  test:
    type: http
routes:
  test-route:
    from: test
    steps: []
`

	data, _, err := ParseYAMLWithComments(yaml)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if data["version"] != 1 {
		t.Errorf("expected version 1, got %v", data["version"])
	}

	routes, ok := data["routes"].(map[string]interface{})
	if !ok {
		t.Errorf("routes should be a map, got %T", data["routes"])
	}

	if _, ok := routes["test-route"]; !ok {
		t.Error("test-route not found in routes")
	}
}

func TestReconstructYAML(t *testing.T) {
	data := map[string]interface{}{
		"version": 1,
		"routes": map[string]interface{}{
			"test-route": map[string]interface{}{
				"from": "source",
			},
		},
	}

	reconstructed, err := ReconstructYAML(data, make(map[string]string))
	if err != nil {
		t.Fatalf("reconstruct failed: %v", err)
	}

	if !strings.Contains(reconstructed, "version: 1") {
		t.Error("version field not in reconstructed YAML")
	}

	if !strings.Contains(reconstructed, "test-route") {
		t.Error("test-route not in reconstructed YAML")
	}
}
