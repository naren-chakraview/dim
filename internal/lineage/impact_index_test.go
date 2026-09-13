package lineage

import (
	"testing"

	"github.com/naren-chakraview/dim/internal/config"
)

func TestExtractSourceReference(t *testing.T) {
	route := &config.RouteConfig{
		From:  "http-input",
		Steps: []*config.Step{},
	}

	refs := ExtractAllReferences(route, "test-route")

	if len(refs) != 1 {
		t.Fatalf("expected 1 reference, got %d", len(refs))
	}

	if refs[0].TargetType != "source" || refs[0].TargetName != "http-input" {
		t.Fatalf("wrong source reference: %+v", refs[0])
	}
}

func TestExtractErrorPathSinkReference(t *testing.T) {
	route := &config.RouteConfig{
		From: "input",
		ErrorPath: &config.ErrorPath{
			Target: "dlq-sink",
		},
		Steps: []*config.Step{},
	}

	refs := ExtractAllReferences(route, "test-route")

	hasErrorPathRef := false
	for _, ref := range refs {
		if ref.TargetType == "sink" && ref.TargetName == "dlq-sink" {
			hasErrorPathRef = true
			break
		}
	}

	if !hasErrorPathRef {
		t.Fatal("missing error path sink reference")
	}
}

func TestDetectDynamicReferences(t *testing.T) {
	route := &config.RouteConfig{
		From: "input",
		Steps: []*config.Step{
			{
				Translate: &config.TranslateStep{
					Expr: "body | $connectionName($)",
				},
			},
		},
	}

	refs := ExtractAllReferences(route, "dynamic-route")

	hasDynamicRef := false
	for _, ref := range refs {
		if !ref.IsStatic && ref.Uncertainty != "" {
			hasDynamicRef = true
			break
		}
	}

	if !hasDynamicRef {
		t.Fatal("should detect dynamic reference")
	}
}

func TestImpactIndexBuild(t *testing.T) {
	route1 := &config.RouteConfig{
		From: "webhook",
		ErrorPath: &config.ErrorPath{Target: "dlq"},
		Steps: []*config.Step{},
	}

	route2 := &config.RouteConfig{
		From: "webhook",
		ErrorPath: &config.ErrorPath{Target: "dlq"},
		Steps: []*config.Step{},
	}

	routes := map[string]*config.RouteConfig{
		"route1": route1,
		"route2": route2,
	}

	index := BuildImpactIndex(routes, make(map[string]*config.Sink))

	// Should have 2 routes referencing webhook source
	if len(index.SourceReferences["webhook"]) != 2 {
		t.Fatalf("expected 2 webhook source refs, got %d", len(index.SourceReferences["webhook"]))
	}

	// Should have 2 routes referencing dlq sink
	if len(index.SinkReferences["dlq"]) != 2 {
		t.Fatalf("expected 2 dlq sink refs, got %d", len(index.SinkReferences["dlq"]))
	}
}

func TestContainsDynamicReferences(t *testing.T) {
	tests := []struct {
		expr     string
		expected bool
	}{
		{"$", true},
		{"body", false},
		{"#", true},
		{"$sum(items.price)", true},
		{"", false},
		{"123", false},
	}

	for _, tt := range tests {
		got := containsDynamicReferences(tt.expr)
		if got != tt.expected {
			t.Errorf("containsDynamicReferences(%q) = %v, want %v", tt.expr, got, tt.expected)
		}
	}
}

func TestFilterStepReference(t *testing.T) {
	route := &config.RouteConfig{
		From: "input",
		Steps: []*config.Step{
			{
				Filter: &config.FilterStep{
					Expr: "body.id != null",
				},
			},
		},
	}

	refs := ExtractAllReferences(route, "filter-route")

	hasFilterRef := false
	for _, ref := range refs {
		if ref.TargetType == "step-type" && ref.TargetName == "filter" {
			hasFilterRef = true
			break
		}
	}

	if !hasFilterRef {
		t.Fatal("should extract filter step reference")
	}
}
