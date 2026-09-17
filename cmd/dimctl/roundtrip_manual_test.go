package main

import (
	"strings"
	"testing"
)

// TestRoundTripPreservesOrderAndComments manually tests T2.1 requirements
func TestRoundTripPreservesOrderAndComments(t *testing.T) {
	originalYAML := `# Route definition with non-alphabetical key order
version: 1

# Source configuration
sources:
  input-api:  # HTTP source
    type: http
    port: 8080

# Sink configuration
sinks:
  output-storage:  # File sink
    type: file
    path: ./output.jsonl

# Route definitions
routes:
  main-route:
    from: input-api
    to: output-storage
    steps:
      - translate:  # Transform the data
          expr: "payload | { id, value: .amount }"
`

	// Parse with preservation
	wrapper, err := ParseYAMLWithPreservation(originalYAML)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Reconstruct from node
	reconstructed, err := ReconstructYAMLFromNode(wrapper)
	if err != nil {
		t.Fatalf("Reconstruct failed: %v", err)
	}

	t.Logf("=== ORIGINAL ===\n%s\n", originalYAML)
	t.Logf("=== RECONSTRUCTED ===\n%s\n", reconstructed)

	// Verify key elements are present
	checks := []struct {
		name     string
		required string
	}{
		{"version key", "version: 1"},
		{"sources comment", "# Source configuration"},
		{"sources key", "sources:"},
		{"input-api key", "input-api:"},
		{"HTTP source comment", "# HTTP source"},
		{"type field", "type: http"},
		{"sinks comment", "# Sink configuration"},
		{"sinks key", "sinks:"},
		{"output-storage key", "output-storage:"},
		{"File sink comment", "# File sink"},
		{"routes comment", "# Route definitions"},
		{"translate comment", "# Transform the data"},
	}

	for _, check := range checks {
		if !strings.Contains(reconstructed, check.required) {
			t.Errorf("Missing in reconstruction: %s (expected: %q)", check.name, check.required)
		}
	}

	// Check key order is preserved (version should come before sources)
	versionPos := strings.Index(reconstructed, "version:")
	sourcesPos := strings.Index(reconstructed, "sources:")
	if versionPos < 0 || sourcesPos < 0 || versionPos >= sourcesPos {
		t.Error("Key order not preserved: 'version' should come before 'sources'")
	}

	// Check sources comes before sinks
	sinksPos := strings.Index(reconstructed, "sinks:")
	if sourcesPos < 0 || sinksPos < 0 || sourcesPos >= sinksPos {
		t.Error("Key order not preserved: 'sources' should come before 'sinks'")
	}

	// Check sinks comes before routes
	routesPos := strings.Index(reconstructed, "routes:")
	if sinksPos < 0 || routesPos < 0 || sinksPos >= routesPos {
		t.Error("Key order not preserved: 'sinks' should come before 'routes'")
	}

	t.Log("✓ Round-trip fidelity test passed")
}
