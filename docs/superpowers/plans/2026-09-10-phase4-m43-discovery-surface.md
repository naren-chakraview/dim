# Phase 4, M4.3 — Pre-deployment Discovery Surface Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build `dimctl catalog search` command that queries existing Apicurio registry and OpenLineage catalog infrastructure to help domain engineers discover what contracts, connections, and data products already exist — enabling "what can I build on" queries without asking a platform team.

**Architecture:** The discovery command is a pure CLI tool that:
1. Connects to existing Apicurio registry (already populated with schemas/contracts)
2. Queries OpenLineage catalog for known data products and lineage
3. Provides searchable interface: by domain, contract subject, connection type
4. Returns results in both human-readable (terminal) and machine-readable (JSON) formats
5. No new infrastructure required — leverages existing observability and schema registry

**Tech Stack:** Go, existing HTTP client libraries, JSON unmarshaling.

**Spec:** `design/phase-4-implementation-plan.md` (M4.3, §4.3), `self-service-feasibility-study.md` (§5.2), existing Apicurio/OpenLineage infrastructure.

## Global Constraints

- Query existing infrastructure only (no new databases, no new APIs to write)
- Results must be immediately actionable (filter by domain, connection type, contract subject)
- Output format: human-readable for CLI, structured JSON for agents (M4.7 integration)
- No breaking changes to existing observability systems
- Graceful fallback if registry/catalog unavailable (inform user, don't crash)

---

## File Structure

**Backend (Go):**
```
cmd/dimctl/catalog.go              # `dimctl catalog` subcommand entry point
cmd/dimctl/catalog_search.go       # Search implementation (registry + catalog queries)
cmd/dimctl/catalog_types.go        # Result types for marshaling
cmd/dimctl/catalog_test.go         # Search tests
```

**Configuration (environment vars):**
```
APICURIO_URL       # Apicurio registry endpoint (default: http://localhost:8080)
OPENLINEAGE_URL    # OpenLineage catalog endpoint (default: http://localhost:5000)
```

---

## Task Breakdown

### Task 1: Define catalog data types and Apicurio client

**Files:**
- Create: `cmd/dimctl/catalog_types.go`
- Create: `cmd/dimctl/catalog_search.go` (partial - types and Apicurio client)

**Steps:**

- [ ] **Step 1: Define result types for marshaling**

```go
type CatalogResult struct {
  Type     string      // "contract", "connection", "product"
  Name     string
  Domain   string
  Subject  string      // For contracts
  Schema   interface{} // JSON schema or null
  Updated  string      // Timestamp
  URL      string      // Link to registry/catalog
}

type SearchResults struct {
  Query   string           `json:"query"`
  Count   int              `json:"count"`
  Results []CatalogResult  `json:"results"`
}
```

- [ ] **Step 2: Implement Apicurio registry client**

```go
type ApicurioClient struct {
  baseURL string
  client  *http.Client
}

// QueryContracts searches Apicurio for schemas matching subject
func (c *ApicurioClient) QueryContracts(ctx context.Context, subject string) ([]CatalogResult, error) {
  // GET /apis/registry/v2/search/artifacts?search=<subject>
  // Parse response and return contracts
}

// ListArtifacts returns all schemas in registry
func (c *ApicurioClient) ListArtifacts(ctx context.Context) ([]CatalogResult, error) {
  // GET /apis/registry/v2/artifacts
}
```

- [ ] **Step 3: Commit**

```bash
git add cmd/dimctl/catalog_types.go cmd/dimctl/catalog_search.go
git commit -m "feat: define catalog types and Apicurio registry client"
```

---

### Task 2: Implement OpenLineage catalog queries

**Files:**
- Modify: `cmd/dimctl/catalog_search.go` (add OpenLineage client)

**Steps:**

- [ ] **Step 1: Implement OpenLineage catalog client**

```go
type OpenLineageClient struct {
  baseURL string
  client  *http.Client
}

// QueryDatasets searches OpenLineage for datasets (data products)
func (c *OpenLineageClient) QueryDatasets(ctx context.Context, domain string) ([]CatalogResult, error) {
  // GET /api/v1/namespaces/{domain}/datasets
  // Parse response and return products with lineage
}

// QueryConnections searches for connection types used in lineage
func (c *OpenLineageClient) QueryConnections(ctx context.Context, connType string) ([]CatalogResult, error) {
  // GET /api/v1/connections?type=<type>
}
```

- [ ] **Step 2: Integrate clients into search interface**

```go
func Search(ctx context.Context, query, filterType, filterDomain string) (*SearchResults, error) {
  results := &SearchResults{Query: query}
  
  // Query Apicurio if looking for contracts
  if filterType == "" || filterType == "contract" {
    apicurioResults, _ := apicurio.QueryContracts(ctx, query)
    results.Results = append(results.Results, apicurioResults...)
  }
  
  // Query OpenLineage if looking for products/connections
  if filterType == "" || filterType == "product" {
    olResults, _ := openlineage.QueryDatasets(ctx, filterDomain)
    results.Results = append(results.Results, olResults...)
  }
  
  results.Count = len(results.Results)
  return results, nil
}
```

- [ ] **Step 3: Commit**

```bash
git add cmd/dimctl/catalog_search.go
git commit -m "feat: implement OpenLineage catalog queries"
```

---

### Task 3: Implement `dimctl catalog search` subcommand

**Files:**
- Create: `cmd/dimctl/catalog.go`
- Modify: `cmd/dimctl/main.go` (add subcommand)

**Steps:**

- [ ] **Step 1: Create catalog subcommand with flags**

```go
var catalogCmd = &cobra.Command{
  Use:   "catalog",
  Short: "Discover existing contracts, connections, and data products",
  Long:  "Query Apicurio registry and OpenLineage catalog to discover what you can build on",
}

var searchCmd = &cobra.Command{
  Use:   "search <query> [--type contract|product|connection] [--domain <name>]",
  Short: "Search catalog for contracts, products, or connections",
  RunE:  runSearch,
  Args:  cobra.ExactArgs(1),
}

var (
  searchType   string
  searchDomain string
  outputJSON   bool
)

func init() {
  searchCmd.Flags().StringVar(&searchType, "type", "", "Filter by type: contract, product, connection")
  searchCmd.Flags().StringVar(&searchDomain, "domain", "", "Filter by domain")
  searchCmd.Flags().BoolVar(&outputJSON, "json", false, "Output as JSON")
  
  catalogCmd.AddCommand(searchCmd)
}

func runSearch(cmd *cobra.Command, args []string) error {
  query := args[0]
  
  results, err := Search(context.Background(), query, searchType, searchDomain)
  if err != nil {
    return fmt.Errorf("search failed: %w", err)
  }
  
  if outputJSON {
    return json.NewEncoder(os.Stdout).Encode(results)
  }
  
  // Human-readable output
  fmt.Printf("Found %d items matching '%s':\n\n", results.Count, query)
  for _, r := range results.Results {
    fmt.Printf("  %s/%s [%s]\n", r.Domain, r.Name, r.Type)
    if r.Subject != "" {
      fmt.Printf("    Subject: %s\n", r.Subject)
    }
    fmt.Printf("    Updated: %s\n", r.Updated)
  }
  
  return nil
}
```

- [ ] **Step 2: Add catalog command to main.go**

```go
rootCmd.AddCommand(catalogCmd)
```

- [ ] **Step 3: Test search command**

```bash
go build -o /tmp/dimctl ./cmd/dimctl
/tmp/dimctl catalog search "payment" --type contract --json
/tmp/dimctl catalog search "order" --domain payments
```

- [ ] **Step 4: Commit**

```bash
git add cmd/dimctl/catalog.go cmd/dimctl/main.go
git commit -m "feat: implement dimctl catalog search subcommand"
```

---

### Task 4: Write tests and end-to-end verification

**Files:**
- Create: `cmd/dimctl/catalog_test.go`

**Steps:**

- [ ] **Step 1: Write unit tests for catalog clients**

```go
func TestApicurioQueryContracts(t *testing.T) {
  // Mock Apicurio API response
  // Verify contracts are correctly parsed
}

func TestOpenLineageQueryDatasets(t *testing.T) {
  // Mock OpenLineage API response
  // Verify datasets are correctly parsed
}

func TestSearchFiltering(t *testing.T) {
  // Test search with type filters
  // Test search with domain filters
  // Test combined filters
}

func TestSearchGracefulFallback(t *testing.T) {
  // Test behavior when Apicurio is unavailable
  // Test behavior when OpenLineage is unavailable
  // Verify error messages are helpful
}

func TestSearchOutputFormats(t *testing.T) {
  // Test JSON output parsing
  // Test human-readable formatting
}
```

- [ ] **Step 2: Run tests**

```bash
go test ./cmd/dimctl -run TestCatalog -v
```

- [ ] **Step 3: Manual end-to-end test**

```bash
# Assuming local Apicurio/OpenLineage running
/tmp/dimctl catalog search "payment-contract" --type contract --json
/tmp/dimctl catalog search "orders" --domain payments
/tmp/dimctl catalog search "" --type product  # List all products
```

- [ ] **Step 4: Verify full test suite passes**

```bash
go test ./... -v
```

- [ ] **Step 5: Commit**

```bash
git add cmd/dimctl/catalog_test.go
git commit -m "test: add catalog search tests and end-to-end verification"
```

---

## Exit Criteria

✅ `dimctl catalog search` command available with flags: `--type`, `--domain`, `--json`
✅ Queries Apicurio registry for contracts/schemas
✅ Queries OpenLineage catalog for data products and connections
✅ Results filterable by domain, contract subject, connection type
✅ Output in both human-readable and JSON formats
✅ Graceful fallback if registry/catalog unavailable
✅ Full test coverage with no failures
✅ Domain engineer can discover existing assets without asking platform team

---

## Summary

**M4.3 delivers a discovery surface** that helps domain engineers self-serve: "what contracts and data products already exist that I can build on?" No more blocking conversations with platform teams just to know what's available. Query existing infrastructure (Apicurio + OpenLineage) that's already tracking contracts, schemas, and lineage.

**Next milestones:** M4.4 (domain-scoped secrets), or jump to Track C (M4.6 agent interface, M4.7 capability manifest).
