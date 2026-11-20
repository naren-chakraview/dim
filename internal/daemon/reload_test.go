package daemon

import (
	"context"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/naren-chakraview/dim/internal/config"
	"github.com/naren-chakraview/dim/internal/engine"
	"github.com/naren-chakraview/dim/internal/factory"
	"github.com/naren-chakraview/dim/internal/tenant"
)

const reloadRouteYAML = `version: 1
sources:
  in:
    type: http
    url: http://localhost/ingest
sinks:
  output:
    type: file
    path: %s
  errors:
    type: file
    path: %s
routes:
  r1:
    from: in
    auth: none
    error_path:
      target: errors
    steps:
      - translate:
          expr: "{\"stage\": \"%s\"}"
`

func writeRoute(t *testing.T, path, outPath, stage string) {
	t.Helper()
	body := strings.Replace(reloadRouteYAML, "%s", outPath, 1)
	body = strings.Replace(body, "%s", filepath.Join(filepath.Dir(outPath), "errors.jsonl"), 1)
	body = strings.Replace(body, "%s", stage, 1)
	if err := os.WriteFile(path, []byte(body), 0644); err != nil {
		t.Fatal(err)
	}
}

func waitForLines(t *testing.T, path string, want int) string {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for {
		b, _ := os.ReadFile(path)
		if n := strings.Count(string(b), "\n"); n >= want {
			return string(b)
		}
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %d line(s) in %s; have: %q", want, path, string(b))
		}
		time.Sleep(20 * time.Millisecond)
	}
}

// After a hot reload, the route must still process messages and write them to
// its sink. Before the fix the reloaded executor was never run and wrote to a
// channel no sink read.
func TestReloadedRouteKeepsRoutingAndSinkOutput(t *testing.T) {
	probe, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("DIM_HTTP_PORT", strconv.Itoa(probe.Addr().(*net.TCPAddr).Port))
	probe.Close()

	dir := t.TempDir()
	outPath := filepath.Join(dir, "out.jsonl")
	routePath := filepath.Join(dir, "route.yaml")
	writeRoute(t, routePath, outPath, "v1")

	cfg, err := config.LoadRouteConfig(routePath)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	gens, router, sources, sinks, err := factory.BuildMultiRoutePipelineWithTracing(ctx, cfg, nil, tenant.NewManager())
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	_ = sources // the HTTP source is not needed: messages are sent into the route directly
	for _, sink := range sinks {
		if err := sink.Start(ctx); err != nil {
			t.Fatal(err)
		}
	}
	if err := router.Start(ctx); err != nil {
		t.Fatal(err)
	}

	state := &reloadState{
		cfg:     cfg,
		gens:    gens,
		exits:   make(chan exitResult, 16),
		cancels: make(map[*engine.Executor]context.CancelFunc),
	}
	for name, gm := range gens {
		state.startExecutor(ctx, name, gm.GetActiveGeneration().Executor)
	}

	send := func(stage string) {
		msg := engine.NewMessage(map[string]interface{}{"n": stage}, "r1", "")
		if err := gens["r1"].GetActiveInputChannel().Send(ctx, msg); err != nil {
			t.Fatalf("send: %v", err)
		}
	}

	send("before")
	waitForLines(t, outPath, 1)

	writeRoute(t, routePath, outPath, "v2")
	state.reload(ctx, routePath)
	if got := gens["r1"].GetActiveGeneration().RouteVersion; got == "" {
		t.Fatal("reload did not take over")
	}

	send("after")
	out := waitForLines(t, outPath, 2)
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if !strings.Contains(lines[1], `"stage":"v2"`) {
		t.Fatalf("message after reload was not processed by the new config; output:\n%s", out)
	}
}

// Two routes must both reach the shared output sink. Before the fix, only the
// first route built was connected to it.
func TestTwoRoutesBothReachSharedSink(t *testing.T) {
	probe, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("DIM_HTTP_PORT", strconv.Itoa(probe.Addr().(*net.TCPAddr).Port))
	probe.Close()

	dir := t.TempDir()
	outPath := filepath.Join(dir, "out.jsonl")
	routePath := filepath.Join(dir, "two.yaml")
	body := `version: 1
sources:
  in:
    type: http
    url: http://localhost/ingest
sinks:
  output:
    type: file
    path: ` + outPath + `
  errors:
    type: file
    path: ` + filepath.Join(dir, "errors.jsonl") + `
routes:
  alpha:
    from: in
    auth: none
    error_path:
      target: errors
    steps:
      - translate:
          expr: "{\"route\": \"alpha\"}"
  beta:
    from: in
    auth: none
    error_path:
      target: errors
    steps:
      - translate:
          expr: "{\"route\": \"beta\"}"
`
	if err := os.WriteFile(routePath, []byte(body), 0644); err != nil {
		t.Fatal(err)
	}
	cfg, err := config.LoadRouteConfig(routePath)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	gens, router, _, sinks, err := factory.BuildMultiRoutePipelineWithTracing(ctx, cfg, nil, tenant.NewManager())
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	for _, sink := range sinks {
		if err := sink.Start(ctx); err != nil {
			t.Fatal(err)
		}
	}
	if err := router.Start(ctx); err != nil {
		t.Fatal(err)
	}
	state := &reloadState{cfg: cfg, gens: gens, exits: make(chan exitResult, 4), cancels: make(map[*engine.Executor]context.CancelFunc)}
	for name, gm := range gens {
		state.startExecutor(ctx, name, gm.GetActiveGeneration().Executor)
	}

	for _, name := range []string{"alpha", "beta"} {
		msg := engine.NewMessage(map[string]interface{}{"n": name}, name, "")
		if err := gens[name].GetActiveInputChannel().Send(ctx, msg); err != nil {
			t.Fatal(err)
		}
	}
	out := waitForLines(t, outPath, 2)
	if !strings.Contains(out, `"route":"alpha"`) || !strings.Contains(out, `"route":"beta"`) {
		t.Fatalf("shared sink missing a route's output:\n%s", out)
	}
}
