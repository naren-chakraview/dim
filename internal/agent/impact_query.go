package agent

import (
	"fmt"

	"github.com/naren-chakraview/dim/internal/lineage"
)

// ImpactQueryRequest asks what would be affected by a change
type ImpactQueryRequest struct {
	ChangeType    string `json:"change_type"`    // "contract" | "sink" | "source" | "step-type"
	ChangeName    string `json:"change_name"`    // Name of what's changing
	ChangeVersion string `json:"change_version,omitempty"` // New version (for version bumps)
}

// ImpactQueryResponse describes what would be affected
type ImpactQueryResponse struct {
	ChangeType     string                       `json:"change_type"`
	ChangeName     string                       `json:"change_name"`
	AffectedRoutes []string                     `json:"affected_routes"`  // Route names
	AffectedSinks  []string                     `json:"affected_sinks"`   // Sink names
	References     []lineage.ImpactReference    `json:"references"`       // Detailed references
	UncertainRefs  []lineage.ImpactReference    `json:"uncertain_refs"`   // Refs we can't determine statically
	Summary        string                       `json:"summary"`          // Human-readable summary
	ImpactLevel    string                       `json:"impact_level"`     // "low" | "medium" | "high" | "critical"
	Confidence     float32                      `json:"confidence"`       // 0.0-1.0: how complete is this analysis
}

// QueryImpact determines what would be affected by a change
func QueryImpact(req ImpactQueryRequest, index *lineage.RouteImpactIndex) (*ImpactQueryResponse, *OperationErr) {
	if req.ChangeType == "" || req.ChangeName == "" {
		return nil, &OperationErr{
			Code:    "INVALID_REQUEST",
			Message: "change_type and change_name are required",
		}
	}

	resp := &ImpactQueryResponse{
		ChangeType:     req.ChangeType,
		ChangeName:     req.ChangeName,
		AffectedRoutes: []string{},
		AffectedSinks:  []string{},
		References:     []lineage.ImpactReference{},
		UncertainRefs:  []lineage.ImpactReference{},
	}

	// Get references based on change type
	var refs []lineage.ImpactReference
	switch req.ChangeType {
	case "contract":
		refs = index.ContractReferences[req.ChangeName]
	case "sink":
		refs = index.SinkReferences[req.ChangeName]
	case "source":
		refs = index.SourceReferences[req.ChangeName]
	case "step-type":
		refs = index.StepTypeReferences[req.ChangeName]
	default:
		return nil, &OperationErr{
			Code:    "INVALID_REQUEST",
			Message: fmt.Sprintf("unknown change_type: %s", req.ChangeType),
		}
	}

	// Separate static from uncertain references
	affectedRoutesMap := make(map[string]bool)
	affectedSinksMap := make(map[string]bool)

	for _, ref := range refs {
		if ref.IsStatic {
			resp.References = append(resp.References, ref)
			if ref.SourceType == "route" {
				affectedRoutesMap[ref.SourceName] = true
			} else if ref.SourceType == "sink" {
				affectedSinksMap[ref.SourceName] = true
			}
		} else {
			resp.UncertainRefs = append(resp.UncertainRefs, ref)
		}
	}

	// Convert maps to slices
	for route := range affectedRoutesMap {
		resp.AffectedRoutes = append(resp.AffectedRoutes, route)
	}
	for sink := range affectedSinksMap {
		resp.AffectedSinks = append(resp.AffectedSinks, sink)
	}

	// Calculate impact level
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

	// Calculate confidence (0.0-1.0)
	uncertainCount := len(resp.UncertainRefs)
	totalCount := len(resp.References) + uncertainCount
	if totalCount == 0 {
		resp.Confidence = 1.0 // No references = complete picture
	} else {
		resp.Confidence = float32(len(resp.References)) / float32(totalCount)
	}

	// Build summary
	resp.Summary = buildImpactSummary(resp)

	return resp, nil
}

// buildImpactSummary builds a human-readable summary
func buildImpactSummary(resp *ImpactQueryResponse) string {
	if len(resp.AffectedRoutes) == 0 && len(resp.AffectedSinks) == 0 {
		return fmt.Sprintf("No routes or sinks directly reference %s '%s'", resp.ChangeType, resp.ChangeName)
	}

	summary := fmt.Sprintf("Changing %s '%s' would affect: ", resp.ChangeType, resp.ChangeName)
	if len(resp.AffectedRoutes) > 0 {
		summary += fmt.Sprintf("%d routes", len(resp.AffectedRoutes))
	}
	if len(resp.AffectedSinks) > 0 {
		if len(resp.AffectedRoutes) > 0 {
			summary += ", "
		}
		summary += fmt.Sprintf("%d sinks", len(resp.AffectedSinks))
	}

	if len(resp.UncertainRefs) > 0 {
		summary += fmt.Sprintf(". Additionally, %d uncertain references (dynamic names) not included in this analysis", len(resp.UncertainRefs))
	}

	summary += fmt.Sprintf(" (Confidence: %.0f%%)", resp.Confidence*100)
	return summary
}
