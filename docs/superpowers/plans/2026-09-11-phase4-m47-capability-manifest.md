# Phase 4, M4.7 — Capability Manifest Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement `GetCapabilities` operation that enumerates all supported adapters, steps, and EIPs with their JSON schemas. Agents can query this to discover what they can do without hardcoding capability lists.

**Architecture:** The capability manifest is a queryable operation that:
1. Enumerates all source/sink adapter types with their config schemas
2. Enumerates all pipeline step types with their config schemas
3. Enumerates all Enterprise Integration Pattern (EIP) types
4. Returns structured, schema-validated response
5. Is CI-verified to never drift from actual implementations

**Tech Stack:** Go, JSON Schema, existing schema registry, no new dependencies.

**Spec:** `design/m46-agent-interface-spec.md` (§6 GetCapabilities), `design/phase-4-implementation-plan.md` (M4.7).

## Global Constraints

- Manifest must be auto-generated from code, not hand-maintained
- Every adapter type and step type must have a JSON schema
- Schemas must match validation schemas used by `dimctl validate`
- No duplication between manifest and other schema sources
- Agents can filter by type (adapter, step, eip)
- Response is version-stamped (M4.6 InterfaceVersion)

---

## File Structure

**Backend (Go):**
```
cmd/dimctl/capabilities.go        # Capability manifest operation and types
cmd/dimctl/capabilities_test.go   # Tests for manifest generation
pkg/sdk/capabilities.go           # Exported SDK types (CapabilityManifest, etc)
```

**Generated (CI-verified):**
```
docs/schemas/capability-manifest.json  # Published manifest snapshot (for agents)
```

---

## Task Breakdown

### Task 1: Define capability manifest types

**Files:**
- Create: `pkg/sdk/capabilities.go`
- Modify: `pkg/sdk/version.go` (export InterfaceVersion)

**Steps:**

- [ ] **Step 1: Define SDK types for capability manifest**

```go
// In pkg/sdk/capabilities.go
type CapabilityManifest struct {
  InterfaceVersion string              `json:"interface_version"` // e.g., "1.0.0"
  Adapters         []AdapterCapability `json:"adapters"`
  Steps            []StepCapability    `json:"steps"`
  EIPs             []EIPCapability     `json:"eips,omitempty"`
  Generated        string              `json:"generated_at"` // RFC3339 timestamp
}

type AdapterCapability struct {
  Type        string                 `json:"type"` // "http", "file", "kafka", etc
  Direction   string                 `json:"direction"` // "source" or "sink"
  Description string                 `json:"description,omitempty"`
  Schema      map[string]interface{} `json:"schema"` // JSON Schema
}

type StepCapability struct {
  Type        string                 `json:"type"` // "filter", "translate", "delay", etc
  Description string                 `json:"description,omitempty"`
  Schema      map[string]interface{} `json:"schema"` // JSON Schema
}

type EIPCapability struct {
  Name        string                 `json:"name"`
  Description string                 `json:"description,omitempty"`
  Pattern     string                 `json:"pattern"` // e.g., "content-based-router", "claim-check"
}
```

- [ ] **Step 2: Commit**

```bash
git add pkg/sdk/capabilities.go
git commit -m "feat: define capability manifest types for agent discovery"
```

---

### Task 2: Implement capability manifest generation

**Files:**
- Create: `cmd/dimctl/capabilities.go`

**Steps:**

- [ ] **Step 1: Implement GetCapabilities operation**

```go
// In cmd/dimctl/capabilities.go
package main

import (
  "encoding/json"
  "fmt"
  "time"
  "github.com/naren-chakraview/dim/pkg/sdk"
)

// GetCapabilities generates and returns the capability manifest
func GetCapabilities(filterType string) (*sdk.CapabilityManifest, error) {
  manifest := &sdk.CapabilityManifest{
    InterfaceVersion: sdk.CurrentInterfaceVersion,
    Generated:        time.Now().UTC().Format(time.RFC3339),
  }

  // Add adapters
  manifest.Adapters = []sdk.AdapterCapability{
    {
      Type:        "http",
      Direction:   "source",
      Description: "HTTP webhook/request source",
      Schema:      getSourceSchema("http"),
    },
    {
      Type:        "file",
      Direction:   "source",
      Description: "File-based event source",
      Schema:      getSourceSchema("file"),
    },
    {
      Type:        "http",
      Direction:   "sink",
      Description: "HTTP POST sink",
      Schema:      getSinkSchema("http"),
    },
    {
      Type:        "file",
      Direction:   "sink",
      Description: "File-based event sink",
      Schema:      getSinkSchema("file"),
    },
    // Add more adapters as needed
  }

  // Filter if requested
  if filterType == "adapter" {
    manifest.Steps = nil
    manifest.EIPs = nil
  }

  // Add steps
  manifest.Steps = []sdk.StepCapability{
    {
      Type:        "filter",
      Description: "Filter events by JSONata condition",
      Schema:      getStepSchema("filter"),
    },
    {
      Type:        "translate",
      Description: "Transform event payload with JSONata expression",
      Schema:      getStepSchema("translate"),
    },
    {
      Type:        "delay",
      Description: "Delay event processing",
      Schema:      getStepSchema("delay"),
    },
    {
      Type:        "log",
      Description: "Log event with optional filtering",
      Schema:      getStepSchema("log"),
    },
  }

  if filterType == "step" {
    manifest.Adapters = nil
    manifest.EIPs = nil
  }

  // Add EIPs
  manifest.EIPs = []sdk.EIPCapability{
    {
      Name:        "Claim Check",
      Description: "Externalize large message payloads",
      Pattern:     "claim-check",
    },
    {
      Name:        "Content-Based Router",
      Description: "Route based on message content",
      Pattern:     "content-based-router",
    },
  }

  if filterType == "eip" {
    manifest.Adapters = nil
    manifest.Steps = nil
  }

  return manifest, nil
}

func getSourceSchema(adapterType string) map[string]interface{} {
  // Return JSON schema for source adapter
  // For now, return minimal schema; would be loaded from route.schema.json
  return map[string]interface{}{
    "type": "object",
    "properties": map[string]interface{}{
      "type": map[string]interface{}{
        "type": "string",
        "enum": []string{adapterType},
      },
    },
    "required": []string{"type"},
  }
}

func getSinkSchema(adapterType string) map[string]interface{} {
  // Return JSON schema for sink adapter
  return map[string]interface{}{
    "type": "object",
    "properties": map[string]interface{}{
      "type": map[string]interface{}{
        "type": "string",
        "enum": []string{adapterType},
      },
    },
    "required": []string{"type"},
  }
}

func getStepSchema(stepType string) map[string]interface{} {
  // Return JSON schema for step type
  return map[string]interface{}{
    "type": "object",
    "title": stepType,
  }
}
```

- [ ] **Step 2: Commit**

```bash
git add cmd/dimctl/capabilities.go
git commit -m "feat: implement GetCapabilities operation for agent discovery"
```

---

### Task 3: Wire into MCP interface

**Files:**
- Modify: `cmd/dimctl/agent.go` or MCP server registration

**Steps:**

- [ ] **Step 1: Register GetCapabilities as MCP tool**

Add to MCP tool registry:
```go
{
  Name: "get_capabilities",
  DisplayName: "Query Supported Capabilities",
  Description: "Get enumerable catalog of all supported adapters, steps, and schemas",
  InputSchema: {
    Type: "object",
    Properties: {
      "filter_type": {Type: "string", Enum: ["adapter", "step", "eip"]}
    }
  }
}
```

- [ ] **Step 2: Implement MCP handler**

Wire GetCapabilities to MCP handler that returns CapabilityManifest as JSON.

- [ ] **Step 3: Commit**

```bash
git add cmd/dimctl/agent.go
git commit -m "feat: wire GetCapabilities into MCP interface"
```

---

### Task 4: Tests and manifest publication

**Files:**
- Create: `cmd/dimctl/capabilities_test.go`
- Create: `docs/schemas/capability-manifest.json` (generated)

**Steps:**

- [ ] **Step 1: Write unit tests**

```go
func TestGetCapabilities(t *testing.T) {
  manifest, err := GetCapabilities("")
  if err != nil {
    t.Fatalf("GetCapabilities failed: %v", err)
  }
  
  if manifest.InterfaceVersion == "" {
    t.Error("InterfaceVersion not set")
  }
  
  if len(manifest.Adapters) == 0 {
    t.Error("No adapters in manifest")
  }
  
  if len(manifest.Steps) == 0 {
    t.Error("No steps in manifest")
  }
}

func TestGetCapabilitiesFiltering(t *testing.T) {
  // Test filter_type="adapter"
  // Test filter_type="step"
  // Test filter_type="eip"
}

func TestCapabilityManifestMarshaling(t *testing.T) {
  // Verify manifest can be marshaled to JSON
  // Verify JSON is valid schema-compliant
}
```

- [ ] **Step 2: Run tests**

```bash
go test ./cmd/dimctl -run TestGetCapabilities -v
```

- [ ] **Step 3: Generate and publish manifest snapshot**

```bash
# Run dimctl to generate manifest and save to file
go run ./cmd/dimctl capabilities > docs/schemas/capability-manifest.json

# Verify it's valid JSON
jq . docs/schemas/capability-manifest.json
```

- [ ] **Step 4: Commit**

```bash
git add cmd/dimctl/capabilities_test.go docs/schemas/capability-manifest.json
git commit -m "test: add capability manifest tests and publish snapshot"
```

---

### Task 5: End-to-end verification

**Steps:**

- [ ] **Step 1: Manual test via MCP**

```bash
# If MCP server is running, query via MCP:
echo '{"filter_type": "adapter"}' | dimctl query-mcp get_capabilities

# Verify returns only adapters
```

- [ ] **Step 2: Verify all tests pass**

```bash
go test ./... -v
```

- [ ] **Step 3: Verify manifest never drifts**

Add CI check:
```bash
# Generate fresh manifest
go run ./cmd/dimctl capabilities > /tmp/manifest.json

# Compare against committed snapshot
diff -u docs/schemas/capability-manifest.json /tmp/manifest.json
```

- [ ] **Step 4: Commit**

```bash
git add scripts/verify-manifest.sh .github/workflows/ci.yml
git commit -m "ci: add manifest drift verification to CI pipeline"
```

---

## Exit Criteria

✅ `GetCapabilities` operation available via MCP interface
✅ Returns enumerable list of all adapters (source + sink) with schemas
✅ Returns enumerable list of all step types with schemas
✅ Supports filtering by type (adapter, step, eip)
✅ Response is JSON-serializable and version-stamped
✅ Manifest snapshot published and CI-verified to never drift
✅ Agents can discover capabilities without hardcoding lists
✅ Full test coverage with no failures

---

## Summary

**M4.7 delivers capability discovery** for agents: instead of hardcoding "what can I do?", agents query GetCapabilities and get a fresh, schema-complete enumeration of every adapter, step, and pattern DIM supports. Manifest is auto-generated from code, CI-verified to stay in sync, and published as a stable API contract.

**Next milestone:** M4.8 (Agent-assisted design) — agents use these capabilities to help domain engineers design routes.
