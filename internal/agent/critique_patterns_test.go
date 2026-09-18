package agent

import (
	"testing"

	"github.com/naren-chakraview/dim/internal/config"
)

func TestCheckUnusedImports(t *testing.T) {
	tests := []struct {
		name     string
		config   *config.RouteConfig
		wantFind bool
	}{
		{
			name: "no imports",
			config: &config.RouteConfig{
				Version: 1,
				Routes: map[string]config.RouteSpec{
					"route1": {
						From:  "input",
						Steps: []config.StepSpec{},
					},
				},
			},
			wantFind: false,
		},
		{
			name: "imports with steps",
			config: &config.RouteConfig{
				Version: 1,
				Imports: []string{"governance/fragments.yaml"},
				Routes: map[string]config.RouteSpec{
					"route1": {
						From: "input",
						Steps: []config.StepSpec{
							{
								Filter: &config.FilterSpec{Expr: "true"},
							},
						},
					},
				},
			},
			wantFind: false,
		},
		{
			name: "unused imports no steps",
			config: &config.RouteConfig{
				Version: 1,
				Imports: []string{"governance/fragments.yaml", "common/helpers.yaml"},
				Routes: map[string]config.RouteSpec{
					"route1": {
						From:  "input",
						Steps: []config.StepSpec{},
					},
				},
			},
			wantFind: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			findings := checkUnusedImports(tt.config)
			if (len(findings) > 0) != tt.wantFind {
				t.Errorf("checkUnusedImports got %d findings, want %v", len(findings), tt.wantFind)
			}
		})
	}
}
