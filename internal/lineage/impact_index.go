package lineage

import (
	"fmt"
	"time"

	"github.com/naren-chakraview/dim/internal/config"
)

// ImpactReference represents a reference from a route to an external component
type ImpactReference struct {
	SourceType   string // "route" | "sink" | "source" | "step"
	SourcePath   string // JSONPath in config (e.g., "$.routes.webhook.steps[0].translate")
	SourceName   string // Route/step/sink name in config

	TargetType  string // "contract" | "sink" | "source" | "step-type" | "connection"
	TargetName  string // What's being referenced

	IsStatic     bool   // True if reference is deterministic, false if dynamic (JSONata computed)
	Uncertainty  string // Reason if not static (e.g., "computed from JSONata expression")

	LineNumber int    // For error reporting
	ConfigPath string // Path to route YAML file
}

// RouteImpactIndex maps targets to the routes/components that reference them
type RouteImpactIndex struct {
	ContractReferences   map[string][]ImpactReference // contract name -> [references]
	SinkReferences       map[string][]ImpactReference // sink name -> [references]
	SourceReferences     map[string][]ImpactReference // source name -> [references]
	ConnectionReferences map[string][]ImpactReference // connection name -> [references]
	StepTypeReferences   map[string][]ImpactReference // step type -> [references]

	AllReferences []ImpactReference // Flat list for iteration
	GeneratedAt   string             // RFC3339 timestamp
	Version       string             // Compatible with M4.6 InterfaceVersion
}

// addReference adds a reference to the index
func (idx *RouteImpactIndex) addReference(ref ImpactReference) {
	switch ref.TargetType {
	case "contract":
		idx.ContractReferences[ref.TargetName] = append(idx.ContractReferences[ref.TargetName], ref)
	case "sink":
		idx.SinkReferences[ref.TargetName] = append(idx.SinkReferences[ref.TargetName], ref)
	case "source":
		idx.SourceReferences[ref.TargetName] = append(idx.SourceReferences[ref.TargetName], ref)
	case "connection":
		idx.ConnectionReferences[ref.TargetName] = append(idx.ConnectionReferences[ref.TargetName], ref)
	case "step-type":
		idx.StepTypeReferences[ref.TargetName] = append(idx.StepTypeReferences[ref.TargetName], ref)
	}
	idx.AllReferences = append(idx.AllReferences, ref)
}

// ExtractAllReferences extracts all references from a route spec
func ExtractAllReferences(route config.RouteSpec, routeName string) []ImpactReference {
	var refs []ImpactReference

	// Source reference
	if route.From != "" {
		refs = append(refs, ImpactReference{
			SourceType: "route",
			SourceName: routeName,
			TargetType: "source",
			TargetName: route.From,
			IsStatic:   true,
			ConfigPath: routeName,
		})
	}

	// Error path sink reference
	if route.ErrorPath != nil && route.ErrorPath.Target != "" {
		refs = append(refs, ImpactReference{
			SourceType: "route",
			SourceName: routeName,
			TargetType: "sink",
			TargetName: route.ErrorPath.Target,
			IsStatic:   true,
			ConfigPath: routeName,
		})
	}

	// Step references
	for i, step := range route.Steps {
		stepPath := fmt.Sprintf("$.routes.%s.steps[%d]", routeName, i)

		// Filter step
		if step.Filter != nil {
			refs = append(refs, ImpactReference{
				SourceType: "step",
				SourcePath: stepPath + ".filter",
				SourceName: fmt.Sprintf("%s.steps[%d]", routeName, i),
				TargetType: "step-type",
				TargetName: "filter",
				IsStatic:   true,
				ConfigPath: routeName,
			})
		}

		// Translate step
		if step.Translate != nil {
			refs = append(refs, ImpactReference{
				SourceType: "step",
				SourcePath: stepPath + ".translate",
				SourceName: fmt.Sprintf("%s.steps[%d]", routeName, i),
				TargetType: "step-type",
				TargetName: "translate",
				IsStatic:   true,
				ConfigPath: routeName,
			})

			// Check if expression contains dynamic references
			if containsDynamicReferences(step.Translate.Expr) {
				refs = append(refs, ImpactReference{
					SourceType:  "step",
					SourcePath:  stepPath + ".translate.expr",
					TargetType:  "connection",
					IsStatic:    false,
					Uncertainty: "expression contains JSONata; actual references cannot be determined statically",
					ConfigPath:  routeName,
				})
			}
		}

		// Authorize step
		if step.Authorize != nil {
			refs = append(refs, ImpactReference{
				SourceType: "step",
				SourcePath: stepPath + ".authorize",
				TargetType: "step-type",
				TargetName: "authorize",
				IsStatic:   true,
				ConfigPath: routeName,
			})
		}
	}

	return refs
}

// containsDynamicReferences detects if expression contains dynamic references
func containsDynamicReferences(expr string) bool {
	if len(expr) == 0 {
		return false
	}
	// Simple heuristic: check for JSONata patterns
	return expr[0] == '$' || expr[0] == '#'
}

// BuildImpactIndex builds an impact index from route specs
func BuildImpactIndex(cfg *config.RouteConfig) *RouteImpactIndex {
	index := &RouteImpactIndex{
		ContractReferences:   make(map[string][]ImpactReference),
		SinkReferences:       make(map[string][]ImpactReference),
		SourceReferences:     make(map[string][]ImpactReference),
		ConnectionReferences: make(map[string][]ImpactReference),
		StepTypeReferences:   make(map[string][]ImpactReference),
		AllReferences:        []ImpactReference{},
	}

	// Extract references from routes
	if cfg != nil && cfg.Routes != nil {
		for routeName, route := range cfg.Routes {
			refs := ExtractAllReferences(route, routeName)
			for _, ref := range refs {
				index.addReference(ref)
			}
		}
	}

	index.GeneratedAt = time.Now().UTC().Format(time.RFC3339)
	index.Version = "1.0.0" // TODO: use actual interface version from SDK

	return index
}
