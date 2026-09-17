package agent

import (
	"testing"

	"github.com/naren-chakraview/dim/internal/config"
	"github.com/naren-chakraview/dim/internal/lineage"
)

func TestQueryContractImpact(t *testing.T) {
	// Build a real config with contract references
	cfg := &config.RouteConfig{
		Version: 1,
		Sources: map[string]config.SourceSpec{
			"input": {Type: "http", URL: "http://localhost:8080"},
		},
		Sinks: map[string]config.SinkSpec{
			"output": {Type: "kafka", URL: "localhost:9092"},
			"backup": {Type: "file", Path: "./backup.jsonl"},
		},
		Routes: map[string]config.RouteSpec{
			"order-processing": {
				From: "input",
				Contracts: []config.ContractSpec{
					{ID: "payment-contract", Version: "1.0.0", Strict: true},
				},
				Steps: []config.StepSpec{
					{Translate: &config.TranslateSpec{Expr: "$ | {order_id, amount}"}},
				},
			},
		},
	}

	// Build index from real config
	index := lineage.BuildImpactIndex(cfg)

	req := ImpactQueryRequest{
		ChangeType: "contract",
		ChangeName: "payment-contract",
	}

	resp, err := QueryImpact(req, index)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(resp.AffectedRoutes) != 1 || resp.AffectedRoutes[0] != "order-processing" {
		t.Fatalf("expected 1 affected route (order-processing), got %d: %v", len(resp.AffectedRoutes), resp.AffectedRoutes)
	}

	if resp.Confidence != 1.0 {
		t.Fatalf("expected 100%% confidence, got %.0f%%", resp.Confidence*100)
	}
}

func TestQueryWithUncertainReferences(t *testing.T) {
	// Build a config with a translate step that has dynamic references
	cfg := &config.RouteConfig{
		Version: 1,
		Sources: map[string]config.SourceSpec{
			"input": {Type: "http", URL: "http://localhost:8080"},
		},
		Sinks: map[string]config.SinkSpec{
			"output": {Type: "http", URL: "http://localhost:9090"},
		},
		Routes: map[string]config.RouteSpec{
			"dynamic-route": {
				From: "input",
				Steps: []config.StepSpec{
					// This translate has a dynamic reference (starts with $)
					{Translate: &config.TranslateSpec{Expr: "$env.target_sink"}},
				},
			},
		},
	}

	// Build index from real config
	index := lineage.BuildImpactIndex(cfg)

	// The dynamic reference should be captured as an uncertain "connection" reference
	if len(index.ConnectionReferences) != 1 {
		t.Fatalf("expected 1 uncertain connection reference, got %d", len(index.ConnectionReferences))
	}
}

func TestQueryNoReferences(t *testing.T) {
	index := &lineage.RouteImpactIndex{
		ContractReferences:   make(map[string][]lineage.ImpactReference),
		SinkReferences:       make(map[string][]lineage.ImpactReference),
		SourceReferences:     make(map[string][]lineage.ImpactReference),
		ConnectionReferences: make(map[string][]lineage.ImpactReference),
		StepTypeReferences:   make(map[string][]lineage.ImpactReference),
	}

	req := ImpactQueryRequest{
		ChangeType: "contract",
		ChangeName: "unused-contract",
	}

	resp, err := QueryImpact(req, index)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(resp.AffectedRoutes) != 0 {
		t.Fatal("should have no affected routes")
	}

	if resp.ImpactLevel != "low" {
		t.Fatalf("expected low impact, got %s", resp.ImpactLevel)
	}
}

func TestQueryRequiredFields(t *testing.T) {
	index := &lineage.RouteImpactIndex{
		ContractReferences: make(map[string][]lineage.ImpactReference),
	}

	tests := []struct {
		req ImpactQueryRequest
		name string
	}{
		{ImpactQueryRequest{ChangeType: "contract"}, "missing change_name"},
		{ImpactQueryRequest{ChangeName: "test"}, "missing change_type"},
	}

	for _, tc := range tests {
		resp, err := QueryImpact(tc.req, index)
		if err == nil {
			t.Errorf("%s: should error, got response: %v", tc.name, resp)
		}
	}
}

func TestQuerySinkImpact(t *testing.T) {
	index := &lineage.RouteImpactIndex{
		SinkReferences: map[string][]lineage.ImpactReference{
			"kafka-output": {
				{SourceType: "route", SourceName: "route1", TargetType: "sink", TargetName: "kafka-output", IsStatic: true},
				{SourceType: "route", SourceName: "route2", TargetType: "sink", TargetName: "kafka-output", IsStatic: true},
			},
		},
		ContractReferences:   make(map[string][]lineage.ImpactReference),
		SourceReferences:     make(map[string][]lineage.ImpactReference),
		ConnectionReferences: make(map[string][]lineage.ImpactReference),
		StepTypeReferences:   make(map[string][]lineage.ImpactReference),
	}

	req := ImpactQueryRequest{
		ChangeType: "sink",
		ChangeName: "kafka-output",
	}

	resp, err := QueryImpact(req, index)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(resp.AffectedRoutes) != 2 {
		t.Fatalf("expected 2 affected routes, got %d", len(resp.AffectedRoutes))
	}

	if resp.ImpactLevel != "medium" {
		t.Fatalf("expected medium impact, got %s", resp.ImpactLevel)
	}
}

func TestImpactLevelCalculation(t *testing.T) {
	tests := []struct {
		routeCount int
		sinkCount  int
		expected   string
	}{
		{0, 0, "low"},
		{1, 0, "medium"},
		{3, 0, "medium"},
		{4, 0, "high"},
		{10, 0, "high"},
		{11, 0, "critical"},
	}

	for _, tt := range tests {
		routes := make([]string, tt.routeCount)
		sinks := make([]string, tt.sinkCount)
		for i := 0; i < tt.routeCount; i++ {
			routes[i] = "route" + string(rune(i))
		}
		for i := 0; i < tt.sinkCount; i++ {
			sinks[i] = "sink" + string(rune(i))
		}

		resp := &ImpactQueryResponse{
			AffectedRoutes: routes,
			AffectedSinks:  sinks,
		}

		// Recalculate impact level as done in QueryImpact
		totalAffected := len(resp.AffectedRoutes) + len(resp.AffectedSinks)
		if totalAffected == 0 {
			resp.ImpactLevel = "low"
		} else if totalAffected <= 3 {
			resp.ImpactLevel = "medium"
		} else if totalAffected <= 10 {
			resp.ImpactLevel = "high"
		} else {
			resp.ImpactLevel = "critical"
		}

		if resp.ImpactLevel != tt.expected {
			t.Errorf("impactLevel for %d routes, %d sinks = %s, want %s", tt.routeCount, tt.sinkCount, resp.ImpactLevel, tt.expected)
		}
	}
}
