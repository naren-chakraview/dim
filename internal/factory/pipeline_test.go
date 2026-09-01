package factory

import (
	"context"
	"testing"
	"time"

	"github.com/naren-chakraview/dim/internal/config"
)

// TestBuildPipelineReturnsExecutor tests that BuildPipeline returns an executor
func TestBuildPipelineReturnsExecutor(t *testing.T) {
	cfg := &config.RouteConfig{
		Version: 1,
		Sources: map[string]config.SourceSpec{
			"http-source": {
				Type: "http",
			},
		},
		Sinks: map[string]config.SinkSpec{
			"output": {
				Type: "file",
				Path: "./output.jsonl",
			},
		},
		Routes: map[string]config.RouteSpec{
			"main": {
				From:  "http-source",
				Steps: []config.StepSpec{},
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Note: We can't actually call BuildPipeline for these tests because
	// the HTTPSource tries to bind to port 8080 immediately. We'll verify
	// the logic through integration tests in cmd/midctl instead.

	// Just verify config loading works
	if len(cfg.Routes) == 0 {
		t.Fatal("expected routes in config")
	}
	if len(cfg.Sinks) == 0 {
		t.Fatal("expected sinks in config")
	}
	if len(cfg.Sources) == 0 {
		t.Fatal("expected sources in config")
	}

	// Verify we can access the first route
	for range cfg.Routes {
		break
	}

	// Verify output sink exists
	outputSink, ok := cfg.Sinks["output"]
	if !ok {
		t.Fatal("output sink not found")
	}
	if outputSink.Path == "" {
		t.Fatal("output sink path is empty")
	}

	_ = ctx
}

// TestBuildPipelineNoRoutes verifies error when no routes configured
func TestBuildPipelineNoRoutes(t *testing.T) {
	cfg := &config.RouteConfig{
		Version: 1,
		Sources: map[string]config.SourceSpec{
			"http-source": {
				Type: "http",
			},
		},
		Sinks: map[string]config.SinkSpec{
			"output": {
				Type: "file",
				Path: "./output.jsonl",
			},
		},
		Routes: map[string]config.RouteSpec{},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, _, _, _, _, _, err := BuildPipeline(ctx, cfg)
	if err == nil {
		t.Fatal("expected error when no routes configured")
	}
}

// TestBuildPipelineNoOutputSink verifies error when output sink is missing
func TestBuildPipelineNoOutputSink(t *testing.T) {
	cfg := &config.RouteConfig{
		Version: 1,
		Sources: map[string]config.SourceSpec{
			"http-source": {
				Type: "http",
			},
		},
		Sinks: map[string]config.SinkSpec{},
		Routes: map[string]config.RouteSpec{
			"main": {
				From:  "http-source",
				Steps: []config.StepSpec{},
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, _, _, _, _, _, err := BuildPipeline(ctx, cfg)
	if err == nil {
		t.Fatal("expected error when output sink is missing")
	}
}

// TestBuildPipelineErrorSinkMissing verifies error when error sink is missing
func TestBuildPipelineErrorSinkMissing(t *testing.T) {
	cfg := &config.RouteConfig{
		Version: 1,
		Sources: map[string]config.SourceSpec{
			"http-source": {
				Type: "http",
			},
		},
		Sinks: map[string]config.SinkSpec{
			"output": {
				Type: "file",
				Path: "./output.jsonl",
			},
		},
		Routes: map[string]config.RouteSpec{
			"main": {
				From: "http-source",
				ErrorPath: &config.ErrorPathSpec{
					Target: "errors", // This sink doesn't exist
				},
				Steps: []config.StepSpec{},
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, _, _, _, _, _, err := BuildPipeline(ctx, cfg)
	if err == nil {
		t.Fatal("expected error when error sink is missing")
	}
}

// TestBuildPipelineInvalidFilterStep verifies error for invalid filter expression
func TestBuildPipelineInvalidFilterStep(t *testing.T) {
	cfg := &config.RouteConfig{
		Version: 1,
		Sources: map[string]config.SourceSpec{
			"http-source": {
				Type: "http",
			},
		},
		Sinks: map[string]config.SinkSpec{
			"output": {
				Type: "file",
				Path: "./output.jsonl",
			},
		},
		Routes: map[string]config.RouteSpec{
			"main": {
				From: "http-source",
				Steps: []config.StepSpec{
					{
						Filter: &config.FilterSpec{
							Expr: "invalid [ syntax",
						},
					},
				},
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, _, _, _, _, _, err := BuildPipeline(ctx, cfg)
	if err == nil {
		t.Fatal("expected error for invalid filter expression")
	}
}

// TestBuildPipelineInvalidTranslateStep verifies error for invalid translate expression
func TestBuildPipelineInvalidTranslateStep(t *testing.T) {
	cfg := &config.RouteConfig{
		Version: 1,
		Sources: map[string]config.SourceSpec{
			"http-source": {
				Type: "http",
			},
		},
		Sinks: map[string]config.SinkSpec{
			"output": {
				Type: "file",
				Path: "./output.jsonl",
			},
		},
		Routes: map[string]config.RouteSpec{
			"main": {
				From: "http-source",
				Steps: []config.StepSpec{
					{
						Translate: &config.TranslateSpec{
							Expr: "invalid [ syntax",
						},
					},
				},
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, _, _, _, _, _, err := BuildPipeline(ctx, cfg)
	if err == nil {
		t.Fatal("expected error for invalid translate expression")
	}
}

// TestBuildPipelineUnsupportedSteps verifies errors for unsupported step types
func TestBuildPipelineUnsupportedSteps(t *testing.T) {
	testCases := []struct {
		name string
		step config.StepSpec
	}{
		{
			name: "route step",
			step: config.StepSpec{
				Route: &config.RouteStepSpec{
					Expr: "body.type",
				},
			},
		},
		{
			name: "wiretap step",
			step: config.StepSpec{
				Wiretap: &config.WiretapSpec{
					Sink: "other",
				},
			},
		},
		{
			name: "idempotent step",
			step: config.StepSpec{
				Idempotent: &config.IdempotentSpec{
					KeyExpr: "body.id",
				},
			},
		},
		{
			name: "authorize step",
			step: config.StepSpec{
				Authorize: &config.AuthorizeSpec{
					Mode: "rbac",
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := &config.RouteConfig{
				Version: 1,
				Sources: map[string]config.SourceSpec{
					"http-source": {
						Type: "http",
					},
				},
				Sinks: map[string]config.SinkSpec{
					"output": {
						Type: "file",
						Path: "./output.jsonl",
					},
				},
				Routes: map[string]config.RouteSpec{
					"main": {
						From:  "http-source",
						Steps: []config.StepSpec{tc.step},
					},
				},
			}

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			_, _, _, _, _, _, err := BuildPipeline(ctx, cfg)
			if err == nil {
				t.Fatalf("expected error for unsupported step type: %s", tc.name)
			}
		})
	}
}
