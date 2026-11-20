package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// fakeViewer serves the same /debug/routes document the daemon's viewer serves.
func fakeViewer(t *testing.T, body string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/debug/routes" {
			http.NotFound(w, r)
			return
		}
		w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return srv
}

func getLive(t *testing.T, mux http.Handler, route string) map[string]interface{} {
	t.Helper()
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/live?route="+route, nil))
	var out map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("bad JSON (status %d): %s", rec.Code, rec.Body.String())
	}
	return out
}

func TestLiveReturnsRouteStateFromViewer(t *testing.T) {
	viewerDoc := `{"routes":[{"name":"order-payment","totalProcessed":7,"totalErrors":1,"p50LatencyMs":2.5},` +
		`{"name":"other","totalProcessed":99}],"pairs":[]}`
	v := fakeViewer(t, viewerDoc)
	t.Setenv("DIM_VIEWER_URL", v.URL)
	mux, _ := liveWorkspace(t, liveRouteYAML)

	got := getLive(t, mux, "order-payment")
	if got["available"] != true {
		t.Fatalf("available = %v, want true: %v", got["available"], got)
	}
	state := got["state"].(map[string]interface{})
	if state["name"] != "order-payment" || state["totalProcessed"] != float64(7) || state["totalErrors"] != float64(1) {
		t.Fatalf("state = %v, want order-payment with processed 7, errors 1", state)
	}
}

func TestLiveReportsRouteNotInViewer(t *testing.T) {
	v := fakeViewer(t, `{"routes":[{"name":"other"}],"pairs":[]}`)
	t.Setenv("DIM_VIEWER_URL", v.URL)
	mux, _ := liveWorkspace(t, liveRouteYAML)

	got := getLive(t, mux, "order-payment")
	if got["available"] != false || !strings.Contains(got["reason"].(string), "not running") {
		t.Fatalf("got %v, want available=false with 'not running'", got)
	}
}

// An unreachable viewer is reported as unavailable, never as zero counts.
func TestLiveReportsViewerUnreachable(t *testing.T) {
	t.Setenv("DIM_VIEWER_URL", "http://127.0.0.1:1") // nothing listens here
	mux, _ := liveWorkspace(t, liveRouteYAML)

	got := getLive(t, mux, "order-payment")
	if got["available"] != false || got["state"] != nil {
		t.Fatalf("got %v, want available=false and no state", got)
	}
	if !strings.Contains(got["reason"].(string), "viewer") {
		t.Fatalf("reason does not name the viewer: %v", got["reason"])
	}
}

func TestLiveRejectsWrongMethod(t *testing.T) {
	mux, _ := liveWorkspace(t, liveRouteYAML)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/live?route=order-payment", nil))
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("POST /api/live status = %d, want 405 (read-only)", rec.Code)
	}
}
