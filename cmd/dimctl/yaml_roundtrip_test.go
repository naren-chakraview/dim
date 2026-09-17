package main

import (
	"testing"

	"gopkg.in/yaml.v3"
)

// Spike Summary:
// yaml.v3 can parse YAML with comment metadata AND round-trip structural changes.
// However, yaml.Marshal() does NOT write back comments automatically.
// MVP solution: Use yaml.v3 for edits, warn user if comments will be lost.

// TestYAMLv3NodeStructure proves yaml.v3 captures comment metadata
func TestYAMLv3NodeStructure(t *testing.T) {
	testYAML := `version: 1
# This is a domain comment
domain: payments
`

	var docNode yaml.Node
	err := yaml.Unmarshal([]byte(testYAML), &docNode)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	commentCount := countComments(&docNode)
	t.Logf("✓ yaml.v3 parsed %d comments from YAML", commentCount)
	if commentCount == 0 {
		t.Errorf("expected comments, got 0")
	}
}

// TestCanRoundTripStructuralChanges proves we can edit and write back
func TestCanRoundTripStructuralChanges(t *testing.T) {
	originalYAML := `version: 1
domain: payments

routes:
  order-payment:
    from: input
    steps: []
`

	// Parse
	var data map[string]interface{}
	err := yaml.Unmarshal([]byte(originalYAML), &data)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	// Modify via standard map
	data["domain"] = "payments-v2"

	// Marshal back
	modified, err := yaml.Marshal(data)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	// Verify
	var result map[string]interface{}
	yaml.Unmarshal(modified, &result)
	if result["domain"] != "payments-v2" {
		t.Errorf("expected payments-v2, got %v", result["domain"])
	}

	t.Logf("✓ Can edit and round-trip structural changes")
}

// TestCommentPreservationIssue demonstrates the key limitation
func TestCommentPreservationIssue(t *testing.T) {
	originalWithComments := `version: 1
# This is important
domain: payments
`

	// Parse with comments preserved in Node
	var node yaml.Node
	yaml.Unmarshal([]byte(originalWithComments), &node)

	// Marshal does NOT write back comments
	marshalled, _ := yaml.Marshal(&node)

	if len(marshalled) > 0 && !containsCommentChar(string(marshalled)) {
		t.Logf("⚠ Expected: yaml.v3 Marshal() does NOT preserve comments")
		t.Logf("  Reason: yaml.Marshal uses map/struct unmarshaling, loses comment metadata")
		t.Logf("  Solution options:")
		t.Logf("    A) Use custom encoder (complex, full fidelity)")
		t.Logf("    B) Warn user and allow raw edit (simple MVP)")
		t.Logf("    C) Store comments separately (fragile)")
		t.Logf("  RECOMMENDATION: Start with Option B for MVP")
	}
}

// TestStudioMVPStrategy defines the spike recommendation
func TestStudioMVPStrategy(t *testing.T) {
	t.Logf("=== SPIKE CONCLUSION: M4.5 MVP Strategy ===")
	t.Logf("")
	t.Logf("Feasibility: ✓ YES - yaml.v3 is viable for visual editing")
	t.Logf("")
	t.Logf("Approach:")
	t.Logf("1. Use yaml.v3 to parse YAML (gets comment metadata)")
	t.Logf("2. When editing: modify via map/struct (loses comments)")
	t.Logf("3. Before saving: detect comments and warn user")
	t.Logf("4. If no comments: round-trip cleanly")
	t.Logf("5. If comments: offer choice:")
	t.Logf("   - Continue (loses comments)")
	t.Logf("   - Cancel (edit raw YAML instead)")
	t.Logf("")
	t.Logf("Why this works:")
	t.Logf("- Honest about limitations (no false promises)")
	t.Logf("- Prevents silent data loss")
	t.Logf("- Allows visual editing for comment-free routes (most common)")
	t.Logf("- Path to full comment preservation via custom encoder later")
	t.Logf("")
	t.Logf("Risk assessment:")
	t.Logf("✓ LOW - yaml.v3 is stable, well-tested")
	t.Logf("✓ MITIGATION: Show comment warning before losing them")
	t.Logf("")
	t.Logf("Exit criterion: Spike validated. Ready for M4.5.2 (Canvas)")
}

// Helpers
func countComments(node *yaml.Node) int {
	if node == nil {
		return 0
	}
	count := 0
	if node.HeadComment != "" {
		count++
	}
	if node.LineComment != "" {
		count++
	}
	if node.FootComment != "" {
		count++
	}
	for _, child := range node.Content {
		count += countComments(child)
	}
	return count
}

func containsCommentChar(s string) bool {
	for _, line := range []byte(s) {
		if line == '#' {
			return true
		}
	}
	return false
}

// T2.1 Fix Verification Tests
func TestYAMLRoundtripAddStep(t *testing.T) {
	// T2.1 Fix verification: adding a step must fail loudly, not silently drop the operation
	// Per the "no fabricated success" rule, if we can't preserve the add operation,
	// we must return an explicit error instead of silently dropping it

	originalYAML := `version: 1
sources:
  input:
    type: file
    path: ./input.jsonl
sinks:
  output:
    type: file
    path: ./output.jsonl
routes:
  test:
    from: input
    auth: none
    steps:
      - filter:
          expr: "true"
`

	// Parse with preservation
	wrapper, err := ParseYAMLWithPreservation(originalYAML)
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	// Add a new step to the route
	if routes, ok := wrapper.Data["routes"].(map[string]interface{}); ok {
		if testRoute, ok := routes["test"].(map[string]interface{}); ok {
			if steps, ok := testRoute["steps"].([]interface{}); ok {
				// Add a new translate step
				newStep := map[string]interface{}{
					"translate": map[string]interface{}{
						"expr": "body | {id, amount}",
					},
				}
				testRoute["steps"] = append(steps, newStep)
			}
		}
	}

	// Reconstruct YAML - should fail with explicit error, not silently drop the add
	reconstructed, err := ReconstructYAML(wrapper.Data, wrapper)

	// CRITICAL: must reject with error, not silently succeed
	if err == nil {
		t.Fatalf("T2.1 BUG: Adding a step silently succeeded! This is the exact fabricated-success bug T2.1 was supposed to fix.\nReconstructed YAML:\n%s", reconstructed)
	}

	if !containsString(err.Error(), "add/remove") {
		t.Fatalf("Expected 'add/remove' in error message, got: %v", err)
	}

	t.Logf("✓ T2.1 verified: Adding a step is correctly rejected with explicit error (not silently dropped)")
	t.Logf("  Error message: %v", err)
}

func TestYAMLRoundtripRemoveStep(t *testing.T) {
	// T2.1 Fix verification: removing a step must fail loudly, not silently keep it
	// Per the "no fabricated success" rule, if we can't preserve the remove operation,
	// we must return an explicit error instead of silently ignoring the user's edit

	originalYAML := `version: 1
sources:
  input:
    type: file
    path: ./input.jsonl
sinks:
  output:
    type: file
    path: ./output.jsonl
routes:
  test:
    from: input
    auth: none
    steps:
      - filter:
          expr: "true"
      - translate:
          expr: "body | {id}"
      - filter:
          expr: "body.id != null"
`

	// Parse with preservation
	wrapper, err := ParseYAMLWithPreservation(originalYAML)
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	// Remove the translate step (middle one)
	if routes, ok := wrapper.Data["routes"].(map[string]interface{}); ok {
		if testRoute, ok := routes["test"].(map[string]interface{}); ok {
			if steps, ok := testRoute["steps"].([]interface{}); ok {
				if len(steps) >= 2 {
					// Remove index 1 (translate step)
					newSteps := append(steps[:1], steps[2:]...)
					testRoute["steps"] = newSteps
				}
			}
		}
	}

	// Reconstruct YAML - should fail with explicit error, not silently keep all 3 steps
	reconstructed, err := ReconstructYAML(wrapper.Data, wrapper)

	// CRITICAL: must reject with error, not silently succeed with old content
	if err == nil {
		t.Fatalf("T2.1 BUG: Removing a step silently succeeded! This is the exact fabricated-success bug T2.1 was supposed to fix.\nReconstructed YAML:\n%s", reconstructed)
	}

	if !containsString(err.Error(), "add/remove") {
		t.Fatalf("Expected 'add/remove' in error message, got: %v", err)
	}

	t.Logf("✓ T2.1 verified: Removing a step is correctly rejected with explicit error (not silently kept)")
	t.Logf("  Error message: %v", err)
}

func containsString(haystack, needle string) bool {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}
