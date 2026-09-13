package lineage

import (
	"testing"
)

func TestImpactIndexEmpty(t *testing.T) {
	index := &RouteImpactIndex{
		ContractReferences:   make(map[string][]ImpactReference),
		SinkReferences:       make(map[string][]ImpactReference),
		SourceReferences:     make(map[string][]ImpactReference),
		ConnectionReferences: make(map[string][]ImpactReference),
		StepTypeReferences:   make(map[string][]ImpactReference),
	}

	if len(index.AllReferences) != 0 {
		t.Fatal("empty index should have no references")
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

func TestAddReference(t *testing.T) {
	index := &RouteImpactIndex{
		ContractReferences:   make(map[string][]ImpactReference),
		SinkReferences:       make(map[string][]ImpactReference),
		SourceReferences:     make(map[string][]ImpactReference),
		ConnectionReferences: make(map[string][]ImpactReference),
		StepTypeReferences:   make(map[string][]ImpactReference),
		AllReferences:        []ImpactReference{},
	}

	ref := ImpactReference{
		SourceType: "route",
		SourceName: "test-route",
		TargetType: "source",
		TargetName: "input",
		IsStatic:   true,
	}

	index.addReference(ref)

	if len(index.SourceReferences["input"]) != 1 {
		t.Fatal("reference not added to index")
	}

	if len(index.AllReferences) != 1 {
		t.Fatal("reference not added to all references")
	}
}
