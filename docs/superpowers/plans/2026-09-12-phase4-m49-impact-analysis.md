# Phase 4, M4.9 — Structured Lineage/Impact-Analysis Query Surface Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement structured impact-analysis queries that let agents answer "what routes/sinks would be affected by changing this contract/sink/source" without manual YAML grepping.

**Architecture:** Impact analysis builds on M4.7's capability manifest and M4.6's interface by adding a static cross-route reference index queried at design/review time:
1. **Static reference index** — Post-import resolution, enumerate which routes/steps reference which sources, sinks, contracts
2. **Impact query surface** — "What routes reference contract X?", "What sinks enforce contract Y?"
3. **Uncertainty surfacing** — Flag dynamically-resolved references (JSONata-computed names) that can't be statically analyzed

**Tech Stack:** Go, existing route config parsing, no new dependencies.

**Spec:** `design/phase-4-implementation-plan.md` (M4.9, §6.6), existing lineage architecture.

## Global Constraints

- Static index built from resolved route configs (post-imports), same point `route_version` is computed
- Every reference must be traceable to specific YAML location (for agent to suggest edits)
- Dynamically-resolved references (JSONata expressions, variable substitution) explicitly flagged as "cannot determine statically"
- No false confidence — better to say "unknown" than to silently omit uncertain references
- All queries exposed via M4.6 MCP interface
- Queries return structured, versioned responses (consistent with M4.6 versioning)

---

## File Structure

**Backend (Go):**
```
internal/lineage/impact_index.go        # Static reference index builder
internal/lineage/impact_index_test.go   # Index tests
internal/agent/impact_query.go          # Impact query operation
internal/agent/impact_query_test.go     # Query tests
```

**Generated/cached:**
```
# Index may be cached during route compilation for performance
# Rebuild triggered by route config changes, contract schema changes
```

---

## Task Breakdown

### Task 1: Define impact index and query types

**Files:**
- Create: `internal/lineage/impact_index.go` (types and index builder)
- Create: `internal/agent/impact_query.go` (query types)

**Steps:**

- [ ] **Step 1: Define impact reference types**

```go
// In internal/lineage/impact_index.go
package lineage

import (
  "github.com/naren-chakraview/dim/internal/config"
)

// ImpactReference represents a reference from a route to an external component
type ImpactReference struct {
  SourceType       string // "route" | "sink" | "source" | "step"
  SourcePath       string // JSONPath in config (e.g., "$.routes.webhook.steps[0].translate")
  SourceName       string // Route/step/sink name in config
  
  TargetType       string // "contract" | "sink" | "source" | "step-type" | "connection"
  TargetName       string // What's being referenced
  
  IsStatic         bool   // True if reference is deterministic, false if dynamic (JSONata computed)
  Uncertainty      string // Reason if not static (e.g., "computed from JSONata expression")
  
  LineNumber       int    // For error reporting
  ConfigPath       string // Path to route YAML file
}

// RouteImpactIndex maps targets to the routes/components that reference them
type RouteImpactIndex struct {
  ContractReferences   map[string][]ImpactReference // contract name -> [references]
  SinkReferences       map[string][]ImpactReference // sink name -> [references]
  SourceReferences     map[string][]ImpactReference // source name -> [references]
  ConnectionReferences map[string][]ImpactReference // connection name -> [references]
  StepTypeReferences   map[string][]ImpactReference // step type -> [references]
  
  AllReferences        []ImpactReference // Flat list for iteration
  GeneratedAt          string             // RFC3339 timestamp
  Version              string             // Compatible with M4.6 InterfaceVersion
}

// BuildImpactIndex scans all routes and builds the cross-reference index
func BuildImpactIndex(routes map[string]*config.RouteConfig, configPath string) (*RouteImpactIndex, error) {
  index := &RouteImpactIndex{
    ContractReferences:   make(map[string][]ImpactReference),
    SinkReferences:       make(map[string][]ImpactReference),
    SourceReferences:     make(map[string][]ImpactReference),
    ConnectionReferences: make(map[string][]ImpactReference),
    StepTypeReferences:   make(map[string][]ImpactReference),
    AllReferences:        []ImpactReference{},
  }

  // Iterate routes and extract references
  for routeName, route := range routes {
    // Extract source references
    if route.From != "" {
      index.addReference(ImpactReference{
        SourceType: "route",
        SourceName: routeName,
        TargetType: "source",
        TargetName: route.From,
        IsStatic:   true,
      })
    }

    // Extract sink references from error_path
    if route.ErrorPath != nil && route.ErrorPath.Target != "" {
      index.addReference(ImpactReference{
        SourceType: "route",
        SourceName: routeName,
        TargetType: "sink",
        TargetName: route.ErrorPath.Target,
        IsStatic:   true,
      })
    }

    // Extract references from steps
    for i, step := range route.Steps {
      // Step type reference
      stepType := inferStepType(step)
      if stepType != "" {
        index.addReference(ImpactReference{
          SourceType:   "step",
          SourcePath:   fmt.Sprintf("$.routes.%s.steps[%d]", routeName, i),
          SourceName:   fmt.Sprintf("%s.steps[%d]", routeName, i),
          TargetType:   "step-type",
          TargetName:   stepType,
          IsStatic:     true,
        })
      }
    }
  }

  // Iterate sinks and extract contract enforcement references
  for sinkName, sink := range routes["sinks"] {
    if sink.Enforce && sink.Contract != "" {
      index.addReference(ImpactReference{
        SourceType: "sink",
        SourceName: sinkName,
        TargetType: "contract",
        TargetName: sink.Contract,
        IsStatic:   true,
      })
    }
  }

  index.GeneratedAt = time.Now().UTC().Format(time.RFC3339)
  index.Version = sdk.CurrentInterfaceVersion
  return index, nil
}

// Helper: add reference to index with deduplication
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

// Helper: infer step type from step structure
func inferStepType(step *config.Step) string {
  if step.Filter != nil {
    return "filter"
  }
  if step.Translate != nil {
    return "translate"
  }
  if step.Authorize != nil {
    return "authorize"
  }
  if step.Log != nil {
    return "log"
  }
  // Add more step types as needed
  return ""
}
```

- [ ] **Step 2: Define impact query types**

```go
// In internal/agent/impact_query.go
package agent

import "github.com/naren-chakraview/dim/internal/lineage"

// ImpactQueryRequest asks what would be affected by a change
type ImpactQueryRequest struct {
  ChangeType string `json:"change_type"` // "contract" | "sink" | "source" | "step-type"
  ChangeName string `json:"change_name"` // Name of what's changing
  ChangeVersion string `json:"change_version,omitempty"` // New version (for version bumps)
}

// ImpactQueryResponse describes what would be affected
type ImpactQueryResponse struct {
  ChangeType      string                           `json:"change_type"`
  ChangeName      string                           `json:"change_name"`
  AffectedRoutes  []string                         `json:"affected_routes"`  // Route names
  AffectedSinks   []string                         `json:"affected_sinks"`   // Sink names
  References      []lineage.ImpactReference        `json:"references"`       // Detailed references
  UncertainRefs   []lineage.ImpactReference        `json:"uncertain_refs"`   // Refs we can't determine statically
  Summary         string                           `json:"summary"`          // Human-readable summary
  ImpactLevel     string                           `json:"impact_level"`     // "low" | "medium" | "high" | "critical"
  Confidence      float32                          `json:"confidence"`       // 0.0-1.0: how complete is this analysis
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
    ChangeType:    req.ChangeType,
    ChangeName:    req.ChangeName,
    AffectedRoutes: []string{},
    AffectedSinks: []string{},
    References:    []lineage.ImpactReference{},
    UncertainRefs: []lineage.ImpactReference{},
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

// Helper: build human-readable summary
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
```

- [ ] **Step 3: Commit**

```bash
git add internal/lineage/impact_index.go internal/agent/impact_query.go
git commit -m "feat: define impact index and query operation types"
```

---

### Task 2: Implement impact index builder

**Files:**
- Modify: `internal/lineage/impact_index.go` (full implementation)
- Create: `internal/lineage/impact_index_test.go` (tests)

**Steps:**

- [ ] **Step 1: Complete impact index builder implementation**

```go
// Enhanced implementation in internal/lineage/impact_index.go
package lineage

import (
  "context"
  "fmt"
  "time"

  "github.com/naren-chakraview/dim/internal/config"
  "github.com/naren-chakraview/dim/pkg/sdk"
)

// ExtractAllReferences recursively extracts all references from a route config
func ExtractAllReferences(route *config.RouteConfig, routeName string) []ImpactReference {
  var refs []ImpactReference

  // Source reference
  if route.From != "" {
    refs = append(refs, ImpactReference{
      SourceType:   "route",
      SourceName:   routeName,
      TargetType:   "source",
      TargetName:   route.From,
      IsStatic:     true,
      ConfigPath:   routeName,
    })
  }

  // Error path sink reference
  if route.ErrorPath != nil && route.ErrorPath.Target != "" {
    refs = append(refs, ImpactReference{
      SourceType:   "route",
      SourceName:   routeName,
      TargetType:   "sink",
      TargetName:   route.ErrorPath.Target,
      IsStatic:     true,
      ConfigPath:   routeName,
    })
  }

  // Step references
  for i, step := range route.Steps {
    stepPath := fmt.Sprintf("$.routes.%s.steps[%d]", routeName, i)
    
    // Translate step: check for JSONata expressions
    if step.Translate != nil {
      // Mark translate steps (step type reference)
      refs = append(refs, ImpactReference{
        SourceType:     "step",
        SourcePath:     stepPath + ".translate",
        SourceName:     fmt.Sprintf("%s.steps[%d]", routeName, i),
        TargetType:     "step-type",
        TargetName:     "translate",
        IsStatic:       true,
        ConfigPath:     routeName,
      })

      // Check if expression contains dynamic references
      if containsDynamicReferences(step.Translate.Expr) {
        refs = append(refs, ImpactReference{
          SourceType:     "step",
          SourcePath:     stepPath + ".translate.expr",
          TargetType:     "connection", // Dynamic reference
          IsStatic:       false,
          Uncertainty:    "expression contains JSONata; actual references cannot be determined statically",
          ConfigPath:     routeName,
        })
      }
    }

    // Filter step
    if step.Filter != nil {
      refs = append(refs, ImpactReference{
        SourceType:   "step",
        SourcePath:   stepPath + ".filter",
        SourceName:   fmt.Sprintf("%s.steps[%d]", routeName, i),
        TargetType:   "step-type",
        TargetName:   "filter",
        IsStatic:     true,
        ConfigPath:   routeName,
      })
    }

    // Authorize step
    if step.Authorize != nil {
      refs = append(refs, ImpactReference{
        SourceType:   "step",
        SourcePath:   stepPath + ".authorize",
        TargetType:   "step-type",
        TargetName:   "authorize",
        IsStatic:     true,
        ConfigPath:   routeName,
      })
    }
  }

  return refs
}

// Helper: detect if expression likely contains dynamic references
func containsDynamicReferences(expr string) bool {
  // Simple heuristic: check for JSONata patterns like $..., body., etc.
  return len(expr) > 0 && (expr[0] == '$' || expr[0] == '#')
}

// LoadAndBuildIndex loads route configs and builds the impact index
func LoadAndBuildIndex(ctx context.Context, domainPaths []string) (*RouteImpactIndex, error) {
  index := &RouteImpactIndex{
    ContractReferences:   make(map[string][]ImpactReference),
    SinkReferences:       make(map[string][]ImpactReference),
    SourceReferences:     make(map[string][]ImpactReference),
    ConnectionReferences: make(map[string][]ImpactReference),
    StepTypeReferences:   make(map[string][]ImpactReference),
    AllReferences:        []ImpactReference{},
  }

  // Load all routes from domain paths
  allRoutes := make(map[string]*config.RouteConfig)
  for _, domainPath := range domainPaths {
    // Load route configs from domain path
    // This would use existing config loading logic
    cfg, err := config.LoadRouteConfig(domainPath)
    if err != nil {
      return nil, fmt.Errorf("failed to load route config from %s: %w", domainPath, err)
    }

    for routeName, route := range cfg.Routes {
      allRoutes[routeName] = route
    }
  }

  // Extract all references
  for routeName, route := range allRoutes {
    refs := ExtractAllReferences(route, routeName)
    for _, ref := range refs {
      index.addReference(ref)
    }
  }

  index.GeneratedAt = time.Now().UTC().Format(time.RFC3339)
  index.Version = sdk.CurrentInterfaceVersion

  return index, nil
}
```

- [ ] **Step 2: Write comprehensive tests**

```go
// In internal/lineage/impact_index_test.go
func TestExtractSourceReference(t *testing.T) {
  route := &config.RouteConfig{
    From: "http-input",
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
  }
  
  route2 := &config.RouteConfig{
    From: "webhook", // Same source
    ErrorPath: &config.ErrorPath{Target: "dlq"}, // Same sink
  }

  routes := map[string]*config.RouteConfig{
    "route1": route1,
    "route2": route2,
  }

  index := &RouteImpactIndex{
    SourceReferences: make(map[string][]ImpactReference),
    SinkReferences:   make(map[string][]ImpactReference),
  }

  for routeName, route := range routes {
    refs := ExtractAllReferences(route, routeName)
    for _, ref := range refs {
      index.addReference(ref)
    }
  }

  // Should have 2 routes referencing webhook source
  if len(index.SourceReferences["webhook"]) != 2 {
    t.Fatalf("expected 2 webhook source refs, got %d", len(index.SourceReferences["webhook"]))
  }

  // Should have 2 routes referencing dlq sink
  if len(index.SinkReferences["dlq"]) != 2 {
    t.Fatalf("expected 2 dlq sink refs, got %d", len(index.SinkReferences["dlq"]))
  }
}
```

- [ ] **Step 3: Commit**

```bash
git add internal/lineage/impact_index.go internal/lineage/impact_index_test.go
git commit -m "feat: implement impact index builder with reference extraction"
```

---

### Task 3: Implement impact query operation

**Files:**
- Modify: `internal/agent/impact_query.go` (full implementation)
- Create: `internal/agent/impact_query_test.go` (tests)
- Modify: `internal/agent/mcp_server.go` (register operation)

**Steps:**

- [ ] **Step 1: Complete impact query implementation**

(See full implementation in Task 1 Step 2, expand QueryImpact function)

- [ ] **Step 2: Write tests**

```go
// In internal/agent/impact_query_test.go
func TestQueryContractImpact(t *testing.T) {
  // Build test index
  index := &lineage.RouteImpactIndex{
    ContractReferences: map[string][]lineage.ImpactReference{
      "payment-contract": {
        {SourceType: "sink", SourceName: "kafka-output", TargetType: "contract", TargetName: "payment-contract", IsStatic: true},
        {SourceType: "sink", SourceName: "file-backup", TargetType: "contract", TargetName: "payment-contract", IsStatic: true},
      },
    },
  }

  req := ImpactQueryRequest{
    ChangeType: "contract",
    ChangeName: "payment-contract",
  }

  resp, err := QueryImpact(req, index)
  if err != nil {
    t.Fatalf("unexpected error: %v", err)
  }

  if len(resp.AffectedSinks) != 2 {
    t.Fatalf("expected 2 affected sinks, got %d", len(resp.AffectedSinks))
  }

  if resp.Confidence != 1.0 {
    t.Fatalf("expected 100%% confidence, got %.0f%%", resp.Confidence*100)
  }
}

func TestQueryWithUncertainReferences(t *testing.T) {
  index := &lineage.RouteImpactIndex{
    ContractReferences: map[string][]lineage.ImpactReference{
      "my-contract": {
        {SourceType: "route", SourceName: "route1", IsStatic: true},
        {SourceType: "route", SourceName: "route2", IsStatic: false, Uncertainty: "computed from JSONata"},
      },
    },
  }

  req := ImpactQueryRequest{
    ChangeType: "contract",
    ChangeName: "my-contract",
  }

  resp, err := QueryImpact(req, index)
  if err != nil {
    t.Fatalf("unexpected error: %v", err)
  }

  if len(resp.UncertainRefs) != 1 {
    t.Fatalf("expected 1 uncertain reference, got %d", len(resp.UncertainRefs))
  }

  if resp.Confidence != 0.5 {
    t.Fatalf("expected 50%% confidence, got %.0f%%", resp.Confidence*100)
  }
}

func TestQueryNoReferences(t *testing.T) {
  index := &lineage.RouteImpactIndex{
    ContractReferences: make(map[string][]lineage.ImpactReference),
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
```

- [ ] **Step 4: Register in MCP server**

```go
// In internal/agent/mcp_server.go - add to registerOperations()
s.operations["query_impact"] = MCPOperation{
  Name:        "query_impact",
  Description: "Analyze impact of a proposed change (contract/sink/source update) on routes",
  Handler: func(ctx context.Context, rawReq json.RawMessage) (interface{}, *OperationErr) {
    var req ImpactQueryRequest
    if err := json.Unmarshal(rawReq, &req); err != nil {
      return nil, &OperationErr{Code: "INVALID_REQUEST", Message: fmt.Sprintf("Failed to parse request: %v", err)}
    }
    
    // Load or get cached impact index
    index := getImpactIndex(ctx)
    return QueryImpact(req, index)
  },
}
```

- [ ] **Step 5: Commit**

```bash
git add internal/agent/impact_query.go internal/agent/impact_query_test.go internal/agent/mcp_server.go
git commit -m "feat: implement impact query operation (M4.9.2)"
```

---

### Task 4: Create worked examples

**Files:**
- Create: `docs/examples/m49-impact-analysis-examples.md`

**Steps:**

- [ ] **Step 1: Write impact analysis examples**

```markdown
# M4.9 — Impact Analysis Query Examples

## Example 1: Contract Version Bump

### Scenario
Payments team wants to bump the payment contract from v1.0 to v2.0. What routes/sinks need updates?

### Query
\`\`\`json
{
  "change_type": "contract",
  "change_name": "payment-contract",
  "change_version": "2.0"
}
\`\`\`

### Response
\`\`\`json
{
  "change_type": "contract",
  "change_name": "payment-contract",
  "affected_routes": ["process-payment", "validate-refund"],
  "affected_sinks": ["kafka-payments", "dlq"],
  "references": [
    {
      "source_type": "sink",
      "source_name": "kafka-payments",
      "target_type": "contract",
      "target_name": "payment-contract",
      "is_static": true,
      "config_path": "domains/payments/payment-routes.yaml"
    },
    {
      "source_type": "sink",
      "source_name": "dlq",
      "target_type": "contract",
      "target_name": "payment-contract",
      "is_static": true
    }
  ],
  "uncertain_refs": [],
  "summary": "Changing contract 'payment-contract' would affect: 2 routes, 2 sinks (Confidence: 100%)",
  "impact_level": "high",
  "confidence": 1.0
}
\`\`\`

### What This Means
- 2 routes directly reference payment-contract
- 2 sinks enforce it
- Version bump requires testing these routes + CI validation
- No uncertain references = complete picture

## Example 2: Sink Configuration Change

### Scenario
Kafka team migrates the payments topic. Impact on routes?

### Query
\`\`\`json
{
  "change_type": "sink",
  "change_name": "kafka-payments"
}
\`\`\`

### Response
\`\`\`json
{
  "change_type": "sink",
  "change_name": "kafka-payments",
  "affected_routes": ["process-payment", "batch-export"],
  "affected_sinks": [],
  "references": [
    {
      "source_type": "route",
      "source_name": "process-payment",
      "target_type": "sink",
      "target_name": "kafka-payments",
      "is_static": true
    },
    {
      "source_type": "route",
      "source_name": "batch-export",
      "target_type": "sink",
      "target_name": "kafka-payments",
      "is_static": true
    }
  ],
  "uncertain_refs": [],
  "summary": "Changing sink 'kafka-payments' would affect: 2 routes (Confidence: 100%)",
  "impact_level": "medium",
  "confidence": 1.0
}
\`\`\`

## Example 3: With Uncertain References

### Scenario
Trying to determine impact of changing a dynamically-resolved connection.

### Query
\`\`\`json
{
  "change_type": "connection",
  "change_name": "message-broker"
}
\`\`\`

### Response
\`\`\`json
{
  "change_type": "connection",
  "change_name": "message-broker",
  "affected_routes": ["known-route"],
  "references": [
    {
      "source_type": "route",
      "source_name": "known-route",
      "target_type": "connection",
      "target_name": "message-broker",
      "is_static": true
    }
  ],
  "uncertain_refs": [
    {
      "source_type": "step",
      "source_path": "$.routes.dynamic-route.steps[0].translate",
      "target_type": "connection",
      "is_static": false,
      "uncertainty": "connection name computed from JSONata expression; cannot determine statically"
    }
  ],
  "summary": "Changing connection 'message-broker' would affect: 1 route. Additionally, 1 uncertain reference (dynamic names) not included in this analysis (Confidence: 50%)",
  "impact_level": "medium",
  "confidence": 0.5
}
\`\`\`

### What This Means
- Known impact: 1 route
- Unknown impact: 1 more route might be affected, but we can't know without running it
- Agent should flag this to human: "50% confidence; some routes use dynamic connection names"
```

- [ ] **Step 2: Commit**

```bash
git add docs/examples/m49-impact-analysis-examples.md
git commit -m "docs: add M4.9 impact analysis examples"
```

---

### Task 5: Integration and verification

**Files:**
- Modify: `internal/agent/agent_test.go` (add M4.9 tests)

**Steps:**

- [ ] **Step 1: Add MCP integration tests**

```go
func TestMCPQueryImpact(t *testing.T) {
  server := NewMCPServer()
  ctx := context.Background()

  req := json.RawMessage(`{
    "change_type": "contract",
    "change_name": "test-contract"
  }`)

  result := server.CallOperation(ctx, "query_impact", req)

  if result.Error != nil {
    t.Fatalf("unexpected error: %v", result.Error)
  }

  var resp agent.ImpactQueryResponse
  respJSON, _ := json.Marshal(result.Result)
  if err := json.Unmarshal(respJSON, &resp); err != nil {
    t.Fatalf("failed to unmarshal response: %v", err)
  }

  if resp.ChangeType != "contract" {
    t.Fatalf("wrong change type: %s", resp.ChangeType)
  }
}

func TestMCPQueryImpactMissingFields(t *testing.T) {
  server := NewMCPServer()
  ctx := context.Background()

  req := json.RawMessage(`{
    "change_type": "contract"
  }`)

  result := server.CallOperation(ctx, "query_impact", req)

  if result.Error == nil {
    t.Fatal("should error on missing change_name")
  }
}
```

- [ ] **Step 2: Verify all tests pass**

```bash
go test ./internal/agent ./internal/lineage -v
```

- [ ] **Step 3: Commit**

```bash
git add internal/agent/agent_test.go
git commit -m "test: add M4.9 MCP integration tests"
```

---

## Exit Criteria

✅ **M4.9.1 Complete:**
- Static impact index built from parsed route configs
- Extracts references from sources, sinks, steps, contracts
- Distinguishes static references from uncertain (dynamically-resolved)
- Index generated at route compilation time (same point as route_version)

✅ **M4.9.2 Complete:**
- Impact query operation accessible via M4.6 MCP interface
- "What routes reference contract/sink/source X?" queries answerable
- Response includes affected components and specific YAML locations
- Uncertainty explicitly surfaced (not silently omitted)

✅ **M4.9.3 Complete:**
- Confidence metric (0.0-1.0) indicates completeness
- Impact level (low/medium/high/critical) summarizes scope
- Worked examples show query/response for various change types
- Tests verify MCP integration and data accuracy

✅ **Global Constraints:**
- ✅ All references traceable to YAML locations
- ✅ Static index built post-imports (same point as route_version)
- ✅ Dynamically-resolved references explicitly flagged as uncertain
- ✅ No false confidence (confidence metric shows what we don't know)
- ✅ All queries versioned and compatible with M4.6

---

## Summary

**M4.9 delivers impact analysis for agents:** Given a proposed contract/sink/source change, enumerate what routes and steps would be affected, without a human manually grepping YAML files. Static index built from resolved configs, explicit about what it can't determine (dynamic references), versioned through M4.6's interface.

**Impact:** Agents can now ask "what if we change X?" and get a reliable, complete answer for all statically-determinable references. For uncertain ones, the query says so — better than silently omitting them or falsely claiming complete knowledge.

