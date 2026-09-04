# M1.6: Schema-Registry-Backed Contracts with Apicurio Reference

**Phase:** Phase 1  
**Component:** M1.6 (Schema registry integration)  
**Status:** Design & Implementation Guide  
**Date:** 2026-09-04

## Overview

M1.6 extends Phase 0's inline JSON Schema contracts to support registry-backed contracts. Schemas are stored in a centralized registry (Apicurio as the OSS reference, with BYO support for Confluent/AWS/Azure).

### Key Design Points

1. **Inline contracts still work** — backward compatible
2. **Registry references** via subject-version syntax (Apicurio-compatible)
3. **Compatibility rules delegated to registry** — dim doesn't re-evaluate
4. **contract_version tagging** for registry-backed schemas
5. **Multi-registry support** — pluggable registry backend

## Subtasks

### M1.6.1: Stand Up Local Apicurio Instance

**Objective:** Validate registry integration against a running Apicurio instance.

**Implementation:**

```bash
# Docker Compose for local Apicurio (for dev/test)
version: '3.8'
services:
  apicurio:
    image: apicurio/apicurio-registry:latest
    ports:
      - "8080:8080"
    environment:
      REGISTRY_AUTH_ENABLED: "false"
      REGISTRY_UI_CONFIG_AUTH_ENABLED: "false"
    volumes:
      - apicurio-data:/data
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8080/health"]
      interval: 10s
      timeout: 5s
      retries: 5

volumes:
  apicurio-data:
```

**Testing:**

```bash
# Register a schema
curl -X POST http://localhost:8080/apis/registry/v3/groups/order-schemas/artifacts \
  -H "Content-Type: application/json" \
  -d '{
    "type": "JSON",
    "name": "OrderSchema",
    "content": "{\"type\":\"object\",\"properties\":{\"order_id\":{\"type\":\"string\"},...}}"
  }'

# List schemas
curl http://localhost:8080/apis/registry/v3/groups/order-schemas/artifacts

# Fetch schema by ID
curl http://localhost:8080/apis/registry/v3/groups/order-schemas/artifacts/OrderSchema/versions
```

### M1.6.2: Extend Contract Model

**Current (Phase 0):** Inline only

```yaml
contract:
  name: OrderContract
  version: 1
  schema: |
    {
      "type": "object",
      "properties": {...}
    }
```

**Extended (Phase 1):** Registry reference

```yaml
contract:
  name: OrderContract
  registry:
    type: apicurio  # or confluent, aws-glue, azure-schema-registry
    url: http://localhost:8080
    group: order-schemas
    subject: OrderSchema
    version: latest  # or specific version: 3
```

**Implementation steps:**

1. Add `RegistryConfig` struct to config loader:
```go
type RegistryConfig struct {
  Type    string // "apicurio", "confluent", "aws-glue", "azure"
  URL     string
  Group   string
  Subject string
  Version string // "latest" or version number
}

type ContractConfig struct {
  Name     string         // Existing field
  Inline   string         // Existing: inline schema
  Registry RegistryConfig // New: registry reference
}
```

2. Update config loader to handle both inline and registry contracts
3. Add registry client interface (pluggable)
4. Implement Apicurio client

### M1.6.3: Compatibility-Rule Delegation

**Key principle:** dim does NOT re-evaluate compatibility; the registry does.

**Flow:**

```
dim (attempting to update contract version)
  ↓
Apicurio Registry (checks compatibility rules)
  ↓
Result: compatible ✓ or incompatible ✗
  ↓
Apicurio returns: new version ID or error
  ↓
dim surfaces error clearly to route config
```

**Example:**

```yaml
# Old schema (v1)
properties:
  order_id: {type: string}
  amount: {type: number}

# New schema (v2, attempted)
properties:
  order_id: {type: string}
  amount: {type: number}
  currency: {type: string, required: true}  # ← NEW REQUIRED FIELD

# Result: Apicurio rejects (backward-incompatible: new required field)
# Error: "Cannot register schema: new required property 'currency' added"
```

**Implementation:**

```go
func (s *RegistryStore) ValidateSchema(ctx context.Context, group, subject string, schema []byte) (newVersion string, err error) {
  // Call registry validation endpoint
  // Return new version ID or error
  // dim surfaces error as: "Contract violation: <registry-reason>"
}
```

### M1.6.4: contract_version Tagging

**Phase 0 (inline):**
```
contract_version = SHA256(inline_schema)
```

**Phase 1 (registry-backed):**
```
contract_version = "<registry>:<group>:<subject>:<version_id>"
Example: apicurio:order-schemas:OrderSchema:3
```

**Implementation:**

```go
func ContractVersion(cfg *ContractConfig) (string, error) {
  if cfg.Inline != "" {
    // Phase 0: hash inline
    return sha256(cfg.Inline), nil
  }
  
  if cfg.Registry != nil {
    // Phase 1: registry reference + version
    client := NewRegistryClient(cfg.Registry.Type, cfg.Registry.URL)
    version, err := client.GetLatestVersion(ctx, cfg.Registry.Group, cfg.Registry.Subject)
    if err != nil {
      return "", err
    }
    return fmt.Sprintf("%s:%s:%s:%s", 
      cfg.Registry.Type, cfg.Registry.Group, cfg.Registry.Subject, version), nil
  }
  
  return "", fmt.Errorf("no contract specified")
}
```

## Design Decisions

### Why Delegate Compatibility to Registry?

- **Single source of truth**: Registry owns compatibility rules
- **No duplication**: dim doesn't need to re-implement registry logic
- **Future-proof**: Registry can add new compatibility modes; dim just calls it
- **Simplicity**: dim is a consumer, not a policy arbiter

### Why Support Multiple Registries?

- Organizations already have one: Confluent, AWS Glue, Azure Schema Registry, OPA
- Apicurio is the open-source reference
- BYO interface lets custom registries integrate

### Why Keep Inline Contracts?

- Not all users need a registry
- Simple routes benefit from simplicity
- Zero new infrastructure for Phase 0 users

## Testing

### Unit Tests
- Registry client mock returns valid schemas
- Contract version computation for inline vs. registry-backed
- Error handling for registry connectivity failures
- Compatibility rule delegation (schema rejected by registry → dim error)

### Integration Tests (M1.6.1)
- Stand up local Apicurio
- Register schema via API
- Fetch schema by reference in route config
- Attempt incompatible schema update → verify rejection

## Migration Path

**Phase 0 routes:** Continue using inline contracts (no change needed)

**Phase 1 adoption (optional):**
```yaml
# Convert from inline to registry:
# Before:
contract:
  name: OrderContract
  version: 1
  schema: |
    {...}

# After:
contract:
  name: OrderContract
  registry:
    type: apicurio
    url: http://apicurio:8080
    group: order-schemas
    subject: OrderSchema
    version: latest
```

## Success Criteria

✅ M1.6.1: Local Apicurio instance runs via Docker Compose; schemas can be registered and fetched  
✅ M1.6.2: Routes can reference registry-backed contracts alongside inline contracts  
✅ M1.6.3: Incompatible schema updates are rejected by registry, error is surfaced clearly  
✅ M1.6.4: contract_version correctly captures registry reference and version  

## References

- **Apicurio Registry:** https://www.apicur.io/
- **JSON Schema:** https://json-schema.org/
- **Registry API:** https://www.apicur.io/registry/docs/
- **Design §11:** Data contracts (design/eip-middleware-design.md §11)
