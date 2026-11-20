package agent

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/naren-chakraview/dim/internal/config"
)

// TestCheckUnusedImports loads real route and fragment files, so the check sees
// the same usage data the loader records in production.
func TestCheckUnusedImports(t *testing.T) {
	const route = `version: 1
imports:
%s
sources:
  input:
    type: file
    path: ./input.jsonl
sinks:
  output:
    type: file
    path: ./output.jsonl
routes:
  route1:
    from: input
    auth: none
    error_path:
      target: output
    steps:
%s
`
	const frags = `version: 1
fragments:
  %s:
    - filter:
        expr: "true"
`
	tests := []struct {
		name       string
		imports    []string // import paths, each with its fragment name
		steps      string   // YAML steps block
		wantUnused []string
	}{
		{
			name:       "import referenced by a step",
			imports:    []string{"used"},
			steps:      "      - fragment: frag-used\n      - filter:\n          expr: \"true\"",
			wantUnused: nil,
		},
		{
			// The case the old check missed: the route has steps, but one import is unused.
			name:       "import declared but unused while route has other steps",
			imports:    []string{"used", "unused"},
			steps:      "      - fragment: frag-used\n      - filter:\n          expr: \"true\"",
			wantUnused: []string{"unused/fragments.yaml"},
		},
	}

	// Through the agent operation itself (the path agents call), not only the helper.
	t.Run("CritiqueRoute reports the unused import", func(t *testing.T) {
		dir := t.TempDir()
		for _, name := range []string{"used", "unused"} {
			os.MkdirAll(filepath.Join(dir, name), 0755)
			os.WriteFile(filepath.Join(dir, name, "fragments.yaml"), []byte(strings.Replace(frags, "%s", "frag-"+name, 1)), 0644)
		}
		body := strings.Replace(route, "%s", "  - used/fragments.yaml\n  - unused/fragments.yaml", 1)
		body = strings.Replace(body, "%s", "      - fragment: frag-used\n      - filter:\n          expr: \"true\"", 1)
		routePath := filepath.Join(dir, "route.yaml")
		os.WriteFile(routePath, []byte(body), 0644)

		resp, operErr := CritiqueRoute(context.Background(), CritiqueRouteRequest{RouteConfigPath: routePath})
		if operErr != nil {
			t.Fatalf("CritiqueRoute: %v", operErr.Message)
		}
		found := false
		for _, f := range resp.Findings {
			if strings.Contains(f.Issue, "Unused import: 'unused/fragments.yaml'") {
				found = true
			}
		}
		if !found {
			t.Fatalf("CritiqueRoute did not report the unused import; findings=%+v", resp.Findings)
		}
	})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			var importLines string
			for _, name := range tt.imports {
				if err := os.MkdirAll(filepath.Join(dir, name), 0755); err != nil {
					t.Fatal(err)
				}
				frag := "frag-" + name
				if err := os.WriteFile(filepath.Join(dir, name, "fragments.yaml"), []byte(strings.Replace(frags, "%s", frag, 1)), 0644); err != nil {
					t.Fatal(err)
				}
				importLines += "  - " + name + "/fragments.yaml\n"
			}
			routePath := filepath.Join(dir, "route.yaml")
			body := strings.Replace(route, "%s", strings.TrimSuffix(importLines, "\n"), 1)
			body = strings.Replace(body, "%s", tt.steps, 1)
			if err := os.WriteFile(routePath, []byte(body), 0644); err != nil {
				t.Fatal(err)
			}

			cfg, err := config.LoadRouteConfig(routePath)
			if err != nil {
				t.Fatalf("LoadRouteConfig: %v", err)
			}
			if !reflect.DeepEqual(cfg.UnusedImports, tt.wantUnused) {
				t.Fatalf("UnusedImports = %v, want %v", cfg.UnusedImports, tt.wantUnused)
			}

			findings := checkUnusedImports(cfg)
			if (len(findings) > 0) != (len(tt.wantUnused) > 0) {
				t.Errorf("checkUnusedImports got %d findings, want unused=%v", len(findings), tt.wantUnused)
			}
		})
	}
}
