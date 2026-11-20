package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// These tests drive the studio through its registered mux (newMux), using the
// same request sequence as the web UI: GET /api/routes, GET /api/route, then
// POST /api/save with {route, filePath}. They assert on the bytes written to
// the route file, not only on the response.

const liveRouteYAML = `version: 1
imports:
  - ../governance/fragments.yaml
sources:
  orders-in:
    type: file
    path: ./orders.jsonl
routes:
  order-payment:
    from: orders-in
    error_path:
      target: error
    steps:
      # keep this comment
      - fragment: governance-baseline
      - filter:
          expr: true
sinks:
  output:
    type: file
    path: ./output/payments-processed.jsonl
  error:
    type: file
    path: ./output/payments-dlq.jsonl
`

// noImportRouteYAML has no imports, so validation runs without the fragment
// file. The silent-no-op cases use it so they are not masked by import errors.
const noImportRouteYAML = `version: 1
sources:
  orders-in:
    type: file
    path: ./orders.jsonl
routes:
  order-payment:
    from: orders-in
    auth: none
    error_path:
      target: error
    steps:
      # keep this comment
      - filter:
          expr: true
      - translate:
          expr: "payload | {id: .id}"
sinks:
  output:
    type: file
    path: ./output/payments-processed.jsonl
  error:
    type: file
    path: ./output/payments-dlq.jsonl
`

// liveWorkspace creates a temp workspace holding the real governance fragment
// and the given route, and returns a studio mux bound to it.
func liveWorkspace(t *testing.T, routeYAML string) (*http.ServeMux, string) {
	t.Helper()
	workDir := t.TempDir()
	govDir := filepath.Join(workDir, "domains", "governance")
	payDir := filepath.Join(workDir, "domains", "payments")
	for _, d := range []string{govDir, payDir} {
		if err := os.MkdirAll(d, 0755); err != nil {
			t.Fatal(err)
		}
	}
	frag, err := os.ReadFile(filepath.Join("..", "..", "domains", "governance", "fragments.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(govDir, "fragments.yaml"), frag, 0644); err != nil {
		t.Fatal(err)
	}
	routePath := filepath.Join(payDir, "order-payment.yaml")
	if err := os.WriteFile(routePath, []byte(routeYAML), 0644); err != nil {
		t.Fatal(err)
	}
	s := &StudioServer{workDir: workDir}
	return s.newMux(), routePath
}

func muxGET(t *testing.T, mux http.Handler, url string) []byte {
	t.Helper()
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, url, nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET %s: status %d: %s", url, rec.Code, rec.Body.String())
	}
	return rec.Body.Bytes()
}

// uiLoadRoute reproduces what the UI does on load: it returns the discovered
// file path and the route's data map from /api/route.
func uiLoadRoute(t *testing.T, mux http.Handler) (filePath string, data map[string]interface{}) {
	t.Helper()
	var list struct {
		Routes map[string]struct {
			FilePath string `json:"filePath"`
		} `json:"routes"`
	}
	if err := json.Unmarshal(muxGET(t, mux, "/api/routes"), &list); err != nil {
		t.Fatal(err)
	}
	filePath = list.Routes["order-payment"].FilePath
	if filePath == "" {
		t.Fatal("order-payment not listed by /api/routes")
	}
	var route struct {
		Data map[string]interface{} `json:"data"`
	}
	if err := json.Unmarshal(muxGET(t, mux, "/api/route?route=order-payment"), &route); err != nil {
		t.Fatal(err)
	}
	return filePath, route.Data
}

// uiSave posts exactly what the UI posts to /api/save.
func uiSave(t *testing.T, mux http.Handler, route map[string]interface{}, filePath string) map[string]interface{} {
	t.Helper()
	body, _ := json.Marshal(map[string]interface{}{"route": route, "filePath": filePath})
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/save", bytes.NewReader(body)))
	var resp map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("bad JSON from /api/save (status %d): %s", rec.Code, rec.Body.String())
	}
	return resp
}

// stepsOf returns the order-payment steps list inside the loaded route data.
func stepsOf(t *testing.T, data map[string]interface{}) []interface{} {
	t.Helper()
	routes := data["routes"].(map[string]interface{})
	return routes["order-payment"].(map[string]interface{})["steps"].([]interface{})
}

func filterStep(t *testing.T, data map[string]interface{}) map[string]interface{} {
	t.Helper()
	for _, st := range stepsOf(t, data) {
		if m, ok := st.(map[string]interface{}); ok {
			if f, ok := m["filter"].(map[string]interface{}); ok {
				return f
			}
		}
	}
	t.Fatal("no filter step found")
	return nil
}

func mustRead(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func assertRefused(t *testing.T, resp map[string]interface{}) {
	t.Helper()
	if resp["success"] != false {
		t.Fatalf("expected refusal, got success=%v (%v)", resp["success"], resp)
	}
}

func TestLiveSaveScalarEditKeepsCommentsAndIndent(t *testing.T) {
	mux, routePath := liveWorkspace(t, liveRouteYAML)
	filePath, data := uiLoadRoute(t, mux)
	filterStep(t, data)["expr"] = "false"

	resp := uiSave(t, mux, data, filePath)
	if resp["success"] != true {
		t.Fatalf("save failed: %v", resp)
	}

	got := mustRead(t, routePath)
	if !strings.Contains(got, "      # keep this comment\n") {
		t.Errorf("comment lost:\n%s", got)
	}
	if !strings.Contains(got, "\n  orders-in:\n    type: file\n") {
		t.Errorf("indentation changed from 2-space:\n%s", got)
	}
	if !strings.Contains(got, "          expr: \"false\"\n") {
		t.Errorf("edit not written:\n%s", got)
	}
	// Exactly one line may differ from the original file.
	want := strings.Split(liveRouteYAML, "\n")
	have := strings.Split(got, "\n")
	if len(want) != len(have) {
		t.Fatalf("line count changed: %d -> %d", len(want), len(have))
	}
	diffs := 0
	for i := range want {
		if want[i] != have[i] {
			diffs++
		}
	}
	if diffs != 1 {
		t.Errorf("expected exactly 1 changed line, got %d:\n%s", diffs, got)
	}
}

func TestLiveSaveNoImportRouteKeepsIndentAndComments(t *testing.T) {
	mux, routePath := liveWorkspace(t, noImportRouteYAML)
	filePath, data := uiLoadRoute(t, mux)
	filterStep(t, data)["expr"] = "false"

	resp := uiSave(t, mux, data, filePath)
	if resp["success"] != true {
		t.Fatalf("save failed: %v", resp)
	}
	got := mustRead(t, routePath)
	if !strings.Contains(got, "      # keep this comment\n") {
		t.Errorf("comment lost:\n%s", got)
	}
	if !strings.Contains(got, "\n  orders-in:\n    type: file\n") {
		t.Errorf("indentation changed from 2-space:\n%s", got)
	}
	if !strings.Contains(got, "          expr: \"false\"\n") {
		t.Errorf("edit not written:\n%s", got)
	}
}

func TestLiveSaveRefusesAddFieldInsideStep(t *testing.T) {
	mux, routePath := liveWorkspace(t, noImportRouteYAML)
	before := mustRead(t, routePath)
	filePath, data := uiLoadRoute(t, mux)
	filterStep(t, data)["note"] = "added"

	resp := uiSave(t, mux, data, filePath)
	assertRefused(t, resp)
	if !strings.Contains(strings.Join(anyStrings(resp["errors"]), " "), "add/remove") {
		t.Errorf("refusal does not explain the add: %v", resp["errors"])
	}
	if mustRead(t, routePath) != before {
		t.Error("file was modified despite refusal")
	}
}

func TestLiveSaveRefusesRemoveFieldInsideStep(t *testing.T) {
	mux, routePath := liveWorkspace(t, noImportRouteYAML)
	before := mustRead(t, routePath)
	filePath, data := uiLoadRoute(t, mux)
	delete(filterStep(t, data), "expr")

	resp := uiSave(t, mux, data, filePath)
	assertRefused(t, resp)
	if mustRead(t, routePath) != before {
		t.Error("file was modified despite refusal")
	}
}

func TestLiveSaveRefusesScalarToMapAndList(t *testing.T) {
	for name, val := range map[string]interface{}{
		"map":  map[string]interface{}{"lang": "jsonata"},
		"list": []interface{}{"a", "b"},
	} {
		t.Run(name, func(t *testing.T) {
			mux, routePath := liveWorkspace(t, noImportRouteYAML)
			before := mustRead(t, routePath)
			filePath, data := uiLoadRoute(t, mux)
			filterStep(t, data)["expr"] = val

			resp := uiSave(t, mux, data, filePath)
			assertRefused(t, resp)
			if mustRead(t, routePath) != before {
				t.Error("file was modified despite refusal")
			}
		})
	}
}

func TestLiveSaveRefusesUnknownFilePath(t *testing.T) {
	mux, routePath := liveWorkspace(t, noImportRouteYAML)
	before := mustRead(t, routePath)
	_, data := uiLoadRoute(t, mux)
	filterStep(t, data)["expr"] = "false"

	resp := uiSave(t, mux, data, filepath.Join(t.TempDir(), "elsewhere.yaml"))
	assertRefused(t, resp)
	if mustRead(t, routePath) != before {
		t.Error("route file modified by save to unknown path")
	}
}

func TestLegacySaveRouteEndpointIsUnreachable(t *testing.T) {
	mux, routePath := liveWorkspace(t, liveRouteYAML)
	before := mustRead(t, routePath)
	body := `{"routes":{"order-payment":{}}}`
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/routes/?route=order-payment", strings.NewReader(body)))
	if rec.Code == http.StatusOK {
		t.Fatalf("legacy /api/routes/ POST still reachable: %d %s", rec.Code, rec.Body.String())
	}
	if mustRead(t, routePath) != before {
		t.Error("legacy endpoint modified the route file")
	}
}

func anyStrings(v interface{}) []string {
	var out []string
	if arr, ok := v.([]interface{}); ok {
		for _, x := range arr {
			if s, ok := x.(string); ok {
				out = append(out, s)
			}
		}
	}
	return out
}
