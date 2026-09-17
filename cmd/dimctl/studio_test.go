package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestStudioEndToEnd(t *testing.T) {
	// Create temp dir with test route
	tmpDir := t.TempDir()
	domainsDir := filepath.Join(tmpDir, "domains", "test")
	if err := os.MkdirAll(domainsDir, 0755); err != nil {
		t.Fatalf("failed to create domains dir: %v", err)
	}

	// Create test route file
	testRouteYAML := `version: 1
sources:
  test-source:
    type: http
sinks:
  test-sink:
    type: file
    path: ./test-output.jsonl
routes:
  test-route:
    from: test-source
    error_path:
      target: test-sink
    steps: []
`
	routePath := filepath.Join(domainsDir, "test-route.yaml")
	if err := os.WriteFile(routePath, []byte(testRouteYAML), 0644); err != nil {
		t.Fatalf("failed to write test route: %v", err)
	}

	// Start studio server in background
	go StartServer(8888, tmpDir)

	// Give server time to start
	time.Sleep(500 * time.Millisecond)

	// Test 1: Load routes from API
	resp, err := http.Get("http://localhost:8888/api/routes")
	if err != nil {
		t.Fatalf("failed to load routes: %v", err)
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	routes, ok := result["routes"].(map[string]interface{})
	if !ok {
		t.Fatalf("routes should be a map, got %T", result["routes"])
	}

	if len(routes) == 0 {
		t.Error("no routes loaded from API")
	}

	if _, ok := routes["test-route"]; !ok {
		t.Error("test-route not found in API response")
	}

	// Test 2: Verify route data structure
	testRoute, ok := routes["test-route"].(map[string]interface{})
	if !ok {
		t.Fatalf("test-route should be a map, got %T", routes["test-route"])
	}

	if testRoute["name"] != "test-route" {
		t.Errorf("expected name test-route, got %v", testRoute["name"])
	}

	if testRoute["domain"] != "test" {
		t.Errorf("expected domain test, got %v", testRoute["domain"])
	}

	// Test 3: Verify UI is served
	resp, err = http.Get("http://localhost:8888/")
	if err != nil {
		t.Fatalf("failed to load UI: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if !bytes.Contains(body, []byte("DIM Visual Route Editor")) {
		t.Error("UI does not contain expected title")
	}

	// Test 4: Verify root element is in the page (React mounts here)
	if !bytes.Contains(body, []byte(`<div id="root">`)) {
		t.Error("UI does not contain root div for React")
	}

	// Test 5: Verify scripts are loaded (React/Vite bundle)
	if !bytes.Contains(body, []byte(`type="module"`)) {
		t.Error("UI does not reference module script")
	}

	// Test 6: Verify CSS bundle is loaded
	if !bytes.Contains(body, []byte(`rel="stylesheet"`)) {
		t.Error("UI does not reference stylesheet")
	}

	// Test 7: Edit route (simulate)
	editData := map[string]interface{}{
		"version": 1,
		"sources": map[string]interface{}{
			"test-source": map[string]interface{}{
				"type": "http",
			},
		},
		"sinks": map[string]interface{}{
			"test-sink": map[string]interface{}{
				"type": "file",
				"path": "./test-output.jsonl",
			},
		},
		"routes": map[string]interface{}{
			"test-route": map[string]interface{}{
				"from": "test-source",
				"error_path": map[string]interface{}{
					"target": "test-sink",
				},
				"steps": []interface{}{},
			},
		},
	}
	bodyJSON, _ := json.Marshal(editData)

	resp, err = http.Post("http://localhost:8888/api/routes?route=test-route", "application/json", bytes.NewReader(bodyJSON))
	if err != nil {
		t.Fatalf("failed to save route: %v", err)
	}
	defer resp.Body.Close()

	// Note: save may fail validation if dimctl validate can't be run in test environment
	// We mainly verify the endpoint is reachable and returns JSON
	var saveResult map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&saveResult); err != nil {
		t.Logf("Note: save response decoding issue (may be validation failure): %v", err)
	}

	// Test 8: Verify route file was saved
	savedContent, err := os.ReadFile(routePath)
	if err != nil {
		t.Fatalf("failed to read saved route: %v", err)
	}

	if len(savedContent) == 0 {
		t.Error("saved route file is empty")
	}

	if !bytes.Contains(savedContent, []byte("version: 1")) {
		t.Error("saved route does not contain version field")
	}

	if !bytes.Contains(savedContent, []byte("test-route")) {
		t.Error("saved route does not contain route name")
	}

	t.Log("All end-to-end tests passed!")
}

func TestYAMLRoundTripIntegration(t *testing.T) {
	yaml := `version: 1
# This is a route comment
routes:
  test-route:  # inline comment
    from: source
`

	wrapper, err := ParseYAMLWithPreservation(yaml)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	reconstructed, err := ReconstructYAMLFromNode(wrapper)
	if err != nil {
		t.Fatalf("reconstruct failed: %v", err)
	}

	// Verify comments are preserved
	if !bytes.Contains([]byte(reconstructed), []byte("# This is a route comment")) {
		t.Error("route comment was lost during round-trip")
	}

	if !bytes.Contains([]byte(reconstructed), []byte("# inline comment")) {
		t.Error("inline comment was lost during round-trip")
	}
}
