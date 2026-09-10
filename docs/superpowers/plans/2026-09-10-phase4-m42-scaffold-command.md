# Phase 4, M4.2 — `dimctl scaffold` Command Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build `dimctl scaffold` command that generates a complete domain directory pre-wired with mandatory governance fragments, starter auth declarations, error paths, and skeleton routes — all passing `dimctl validate` out of the box.

**Architecture:** The scaffold command is a pure CLI tool that:
1. Accepts command-line flags: `--domain <name>`, `--source <type>`, `--sink <type>`, `--with-contract` (optional)
2. Generates a domain directory structure under `domains/<domain>/`
3. Creates `DOMAIN.yaml` with domain metadata
4. Creates a starter route YAML with governance fragment import
5. Creates `error-path.yaml` template if `--with-contract` is specified
6. All output passes `dimctl validate` with zero edits required

**Tech Stack:** Go (no new dependencies required — uses existing `cobra`, `yaml`, and `config` packages).

**Spec:** `design/phase-4-implementation-plan.md` (M4.2, §4.2), `self-service-feasibility-study.md` (§5.1), existing `dimctl validate` command.

## Global Constraints

- Output directory must not exist (fail if already present — no clobbering)
- All generated files must pass `dimctl validate` without modification
- Governance fragment import is mandatory (all routes must import `../governance/fragments.yaml`)
- Starter templates cover three common shapes: source→sink passthrough, source→transform→sink, source→contract-enforced sink
- No runtime changes required; this is pure CLI scaffolding over existing schema and imports mechanism

---

## File Structure

**Backend (Go):**
```
cmd/dimctl/scaffold.go              # `dimctl scaffold` subcommand entry point
cmd/dimctl/scaffold_templates.go    # Template definitions and rendering
cmd/dimctl/scaffold_test.go         # Scaffolding tests
```

**Generated artifacts (not in repo, produced by scaffold):**
```
domains/<domain>/
  ├── DOMAIN.yaml                   # Domain metadata
  ├── <domain>-route.yaml           # Starter route (passes dimctl validate)
  └── error-path.yaml               # Error handling template (if --with-contract)
```

---

## Task Breakdown

### Task 1: Define scaffold templates

**Files:**
- Create: `cmd/dimctl/scaffold_templates.go`

**Steps:**

- [ ] **Step 1: Define three template shapes**

Create template definitions for three common route patterns:

```go
// Template for passthrough route (source → sink)
var passthroughTemplate = `version: 1
imports:
  - ../governance/fragments.yaml

sources:
  input:
    type: http

routes:
  passthrough:
    from: input
    steps: []

sinks:
  output:
    type: file
    path: ./output/messages.jsonl
`

// Template for transform route (source → transform step → sink)
var transformTemplate = `version: 1
imports:
  - ../governance/fragments.yaml

sources:
  input:
    type: http

routes:
  with-transform:
    from: input
    steps:
      - translate:
          expr: { ... }  # User fills in JSONata expression

sinks:
  output:
    type: file
    path: ./output/messages.jsonl
`

// Template for contract-enforced route
var contractTemplate = `version: 1
imports:
  - ../governance/fragments.yaml

sources:
  input:
    type: http

contracts:
  data-contract:
    schema: |
      {
        "type": "object",
        "properties": {
          "id": { "type": "string" },
          "amount": { "type": "number" }
        }
      }

routes:
  with-contract:
    from: input
    steps: []

sinks:
  output:
    type: file
    path: ./output/messages.jsonl
    enforce: true
    contract: data-contract
`
```

- [ ] **Step 2: Define domain metadata template**

```go
var domainMetadataTemplate = `# Domain metadata
domain:
  name: {{.DomainName}}
  owner: user@example.com
  slack_channel: "#{{.DomainName}}-team"
  description: {{.DomainName}} data processing
  owner_team: {{.DomainName}}-platform
`
```

- [ ] **Step 3: Commit**

```bash
git add cmd/dimctl/scaffold_templates.go
git commit -m "feat: define scaffold templates for three common route shapes"
```

---

### Task 2: Implement `dimctl scaffold` subcommand

**Files:**
- Create: `cmd/dimctl/scaffold.go`
- Modify: `cmd/dimctl/main.go` (add subcommand)

**Steps:**

- [ ] **Step 1: Create scaffold subcommand with flags**

In `cmd/dimctl/scaffold.go`:

```go
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"text/template"

	"github.com/spf13/cobra"
)

var scaffoldCmd = &cobra.Command{
	Use:   "scaffold data-product --domain <name> --source <type> --sink <type> [--with-contract]",
	Short: "Generate a domain directory scaffold with starter route",
	Long:  "Generate a complete domain directory pre-wired with governance fragments and starter routes",
	RunE:  runScaffold,
}

var (
	domainName  string
	sourceType  string
	sinkType    string
	withContract bool
	templateType string
)

func init() {
	scaffoldCmd.Flags().StringVar(&domainName, "domain", "", "Domain name (required)")
	scaffoldCmd.Flags().StringVar(&sourceType, "source", "http", "Source type (http, file, kafka, amqp)")
	scaffoldCmd.Flags().StringVar(&sinkType, "sink", "file", "Sink type (file, kafka, http)")
	scaffoldCmd.Flags().BoolVar(&withContract, "with-contract", false, "Generate contract enforcement template")
	scaffoldCmd.Flags().StringVar(&templateType, "template", "passthrough", "Template shape (passthrough, transform, contract)")
	
	scaffoldCmd.MarkFlagRequired("domain")
}

func runScaffold(cmd *cobra.Command, args []string) error {
	if domainName == "" {
		return fmt.Errorf("--domain is required")
	}

	// Create domain directory
	domainPath := filepath.Join("domains", domainName)
	if _, err := os.Stat(domainPath); err == nil {
		return fmt.Errorf("domain directory already exists: %s", domainPath)
	}

	if err := os.MkdirAll(domainPath, 0755); err != nil {
		return fmt.Errorf("failed to create domain directory: %w", err)
	}

	// Generate files
	if err := generateDomainMetadata(domainPath, domainName); err != nil {
		return fmt.Errorf("failed to generate domain metadata: %w", err)
	}

	if err := generateRoute(domainPath, domainName, templateType); err != nil {
		return fmt.Errorf("failed to generate route: %w", err)
	}

	if withContract {
		if err := generateContractTemplate(domainPath); err != nil {
			return fmt.Errorf("failed to generate contract template: %w", err)
		}
	}

	fmt.Printf("✓ Generated domain scaffold at %s\n", domainPath)
	fmt.Printf("✓ Run 'dimctl validate domains/%s/*.yaml' to verify\n", domainName)
	return nil
}

func generateDomainMetadata(domainPath, domainName string) error {
	// Use domain metadata template
	data := map[string]string{
		"DomainName": domainName,
	}
	tmpl, _ := template.New("domain").Parse(domainMetadataTemplate)
	
	file, err := os.Create(filepath.Join(domainPath, "DOMAIN.yaml"))
	if err != nil {
		return err
	}
	defer file.Close()
	
	return tmpl.Execute(file, data)
}

func generateRoute(domainPath, domainName, templateType string) error {
	var routeYAML string
	switch templateType {
	case "passthrough":
		routeYAML = passthroughTemplate
	case "transform":
		routeYAML = transformTemplate
	case "contract":
		routeYAML = contractTemplate
	default:
		return fmt.Errorf("unknown template type: %s", templateType)
	}

	routePath := filepath.Join(domainPath, fmt.Sprintf("%s-route.yaml", domainName))
	return os.WriteFile(routePath, []byte(routeYAML), 0644)
}

func generateContractTemplate(domainPath string) error {
	contractPath := filepath.Join(domainPath, "error-path.yaml")
	return os.WriteFile(contractPath, []byte(errorPathTemplate), 0644)
}

var errorPathTemplate = `version: 1

# Error path template — uncomment and customize as needed
# error_path:
#   steps:
#     - log:
#         level: error
#   sinks:
#     dlq:
#       type: file
#       path: ./dlq/failed-messages.jsonl
`
```

- [ ] **Step 2: Add scaffold subcommand to main.go**

In `cmd/dimctl/main.go`, add to init():

```go
rootCmd.AddCommand(scaffoldCmd)
```

- [ ] **Step 3: Test scaffold command**

```bash
go build -o /tmp/dimctl ./cmd/dimctl
/tmp/dimctl scaffold data-product --domain test-domain --source http --sink file
ls -la domains/test-domain/
/tmp/dimctl validate domains/test-domain/test-domain-route.yaml
```

- [ ] **Step 4: Commit**

```bash
git add cmd/dimctl/scaffold.go cmd/dimctl/main.go
git commit -m "feat: implement dimctl scaffold subcommand with template generation"
```

---

### Task 3: Write tests

**Files:**
- Create: `cmd/dimctl/scaffold_test.go`

**Steps:**

- [ ] **Step 1: Write unit tests**

```go
package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestScaffoldGeneratesDomainMetadata(t *testing.T) {
	tmpDir := t.TempDir()
	os.Chdir(tmpDir)

	// Run scaffold
	// ...verify DOMAIN.yaml is created with correct metadata
}

func TestScaffoldGeneratesValidRoute(t *testing.T) {
	tmpDir := t.TempDir()
	os.Chdir(tmpDir)

	// Run scaffold
	// ...verify route file passes dimctl validate
}

func TestScaffoldFailsIfDirectoryExists(t *testing.T) {
	tmpDir := t.TempDir()
	os.MkdirAll(filepath.Join(tmpDir, "domains", "test"), 0755)
	os.Chdir(tmpDir)

	// Run scaffold --domain test
	// ...should fail with "already exists" error
}

func TestScaffoldWithContract(t *testing.T) {
	tmpDir := t.TempDir()
	os.Chdir(tmpDir)

	// Run scaffold --with-contract
	// ...verify error-path.yaml is generated
}
```

- [ ] **Step 2: Run tests**

```bash
go test ./cmd/dimctl -run TestScaffold -v
```

- [ ] **Step 3: Commit**

```bash
git add cmd/dimctl/scaffold_test.go
git commit -m "test: add scaffold command tests"
```

---

### Task 4: End-to-end verification

**Steps:**

- [ ] **Step 1: Manual test all three template shapes**

```bash
/tmp/dimctl scaffold data-product --domain payments --source http --sink kafka --template passthrough
/tmp/dimctl validate domains/payments/payments-route.yaml

/tmp/dimctl scaffold data-product --domain inventory --source file --sink http --template transform
/tmp/dimctl validate domains/inventory/inventory-route.yaml

/tmp/dimctl scaffold data-product --domain orders --source kafka --sink file --with-contract --template contract
/tmp/dimctl validate domains/orders/orders-route.yaml
```

- [ ] **Step 2: Verify all tests pass**

```bash
go test ./cmd/dimctl -v
```

- [ ] **Step 3: Commit**

```bash
git add -A
git commit -m "test: verify all scaffold outputs pass dimctl validate"
```

---

## Exit Criteria

✅ `dimctl scaffold` command available with flags: `--domain`, `--source`, `--sink`, `--with-contract`
✅ Three template shapes (passthrough, transform, contract) generate correctly
✅ All scaffolded output passes `dimctl validate` with zero edits
✅ Mandatory governance fragment import present in all generated routes
✅ Domain metadata (DOMAIN.yaml) created with correct structure
✅ Full test coverage (unit + e2e)

---

## Summary

**M4.2 delivers a scaffolding tool** that lets domain engineers bootstrap new domains with zero setup friction. Run one command, get a complete, validated, governance-compliant domain structure. No manual file creation, no copy-paste templates, no validation surprises.

**Next milestones:** M4.3 (discovery surface), M4.4 (domain-scoped secrets), or start Track C (M4.6 agent interface).
