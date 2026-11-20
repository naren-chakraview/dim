package factory

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/naren-chakraview/dim/internal/config"
	"github.com/naren-chakraview/dim/internal/engine"
	"github.com/naren-chakraview/dim/internal/tenant"
)

func loadCfgFromString(t *testing.T, body string) *config.RouteConfig {
	t.Helper()
	path := filepath.Join(t.TempDir(), "route.yaml")
	if err := os.WriteFile(path, []byte(body), 0644); err != nil {
		t.Fatal(err)
	}
	cfg, err := config.LoadRouteConfig(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	return cfg
}

// A file source configured in a route's sources: receives a file dropped into its
// directory, and the route's output reaches the sink. Nothing here is mocked: the
// poller, router, executor and file sink all run.
func TestFileSourceDeliversThroughRouteToSink(t *testing.T) {
	dir := t.TempDir()
	inDir := filepath.Join(dir, "in")
	outPath := filepath.Join(dir, "out.jsonl")
	if err := os.MkdirAll(inDir, 0755); err != nil {
		t.Fatal(err)
	}
	cfg := loadCfgFromString(t, `version: 1
sources:
  landing:
    type: file
    path: `+inDir+`
    schedule: 100ms
sinks:
  output:
    type: file
    path: `+outPath+`
  errors:
    type: file
    path: `+filepath.Join(dir, "errors.jsonl")+`
routes:
  ingest:
    from: landing
    auth: none
    error_path:
      target: errors
    steps:
      - translate:
          expr: "{\"landed\": true}"
`)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	gens, router, sources, sinks, err := BuildMultiRoutePipelineWithTracing(ctx, cfg, nil, tenant.NewManager())
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	if len(sources) != 1 {
		t.Fatalf("got %d sources, want 1 (the file source)", len(sources))
	}
	for _, src := range sources {
		go src.Start(ctx)
	}
	for _, sink := range sinks {
		if err := sink.Start(ctx); err != nil {
			t.Fatal(err)
		}
	}
	if err := router.Start(ctx); err != nil {
		t.Fatal(err)
	}
	for _, gm := range gens {
		go gm.GetActiveGeneration().Executor.Run(ctx)
	}

	// Publish a file atomically, as a producer should.
	staged := filepath.Join(dir, "staged.json")
	if err := os.WriteFile(staged, []byte(`{"order": 1}`), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(staged, filepath.Join(inDir, "order.json")); err != nil {
		t.Fatal(err)
	}

	deadline := time.Now().Add(10 * time.Second)
	for {
		b, _ := os.ReadFile(outPath)
		if strings.Contains(string(b), `"landed":true`) {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("no output from the file source reached the sink; output: %q", string(b))
		}
		time.Sleep(50 * time.Millisecond)
	}
}

// A source shared by routes in two domains is refused: its secrets could not be
// scoped to a single domain.
func TestSourceSharedAcrossDomainsIsRefused(t *testing.T) {
	cfg := loadCfgFromString(t, `version: 1
sources:
  shared:
    type: file
    path: /tmp
sinks:
  output:
    type: file
    path: /tmp/o.jsonl
  errors:
    type: file
    path: /tmp/e.jsonl
routes:
  a:
    from: shared
    domain: payments
    auth: none
    error_path:
      target: errors
    steps:
      - filter:
          expr: "true"
  b:
    from: shared
    domain: fraud
    auth: none
    error_path:
      target: errors
    steps:
      - filter:
          expr: "true"
`)
	_, err := buildSourceBindings(context.Background(), cfg, secretResolver())
	if err == nil || !strings.Contains(err.Error(), "one domain") {
		t.Fatalf("shared source across domains: got %v, want a one-domain error", err)
	}
}

// A route that names a source the config does not define is an error, not an
// implicit HTTP source.
func TestRouteReadingUndefinedSourceIsRefused(t *testing.T) {
	cfg := loadCfgFromString(t, `version: 1
sources:
  real:
    type: file
    path: /tmp
sinks:
  output:
    type: file
    path: /tmp/o.jsonl
  errors:
    type: file
    path: /tmp/e.jsonl
routes:
  a:
    from: missing
    auth: none
    error_path:
      target: errors
    steps:
      - filter:
          expr: "true"
`)
	_, err := buildSourceBindings(context.Background(), cfg, secretResolver())
	if err == nil || !strings.Contains(err.Error(), `"missing"`) {
		t.Fatalf("undefined source: got %v, want an error naming the source", err)
	}
}

var _ = engine.NewChannel // keep the engine import for future route-level assertions
