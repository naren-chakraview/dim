# Agent Interface Versioning Policy

**Version:** 1.0.0  
**Effective Date:** 2026-09-11  
**Policy Lifecycle:** Until superseded by a newer version

---

## Overview

The Agent-Facing Interface uses **Semantic Versioning (SemVer)** to communicate changes to external agents.

**Version Format:** `MAJOR.MINOR.PATCH`
- **MAJOR:** Breaking changes (agents must update)
- **MINOR:** Backward-compatible additions (agents can ignore)
- **PATCH:** Bug fixes (no action needed)

**Current Version:** `1.0.0`

---

## Version Compatibility

### Major Version (X.y.z)

**When it changes:** Removing operations, incompatible request/response changes, error format changes.

**Agent behavior:**
- **MUST** update before using interface
- Older agents receive `VERSION_MISMATCH` error
- Server will not process requests with old version

**Example transitions:**
- `1.0.0` → `2.0.0`: Removes `validate_route` operation
- `1.0.0` → `2.0.0`: Changes `route_config_path` to `file_path`
- `1.0.0` → `2.0.0`: Changes error format from flat to nested

### Minor Version (x.Y.z)

**When it changes:** Adding new operations, adding optional parameters, expanding response types.

**Agent behavior:**
- Can ignore (operations your agent uses work unchanged)
- Can adopt (new operations become available)
- No action required

**Example transitions:**
- `1.0.0` → `1.1.0`: Adds new `analyze_impact` operation
- `1.0.0` → `1.1.0`: Adds optional `include_warnings` parameter to `validate_route`
- `1.0.0` → `1.1.0`: Adds `suggestions` field to validation errors

### Patch Version (x.y.Z)

**When it changes:** Bug fixes, performance improvements, documentation.

**Agent behavior:**
- Transparent — no action needed
- Behavior is identical to previous patch version
- Can ignore or update

**Example transitions:**
- `1.0.0` → `1.0.1`: Fixes timeout calculation in `test_route`
- `1.0.0` → `1.0.1`: Improves error message clarity
- `1.0.0` → `1.0.1`: Optimizes `get_capabilities` response time

---

## Deprecation Process

When an operation or parameter needs to be removed, it goes through a deprecation phase:

### Phase 1: Announced (6 months before removal)

**What happens:**
- Operation continues to work
- Response includes `deprecation_notice` field
- Notice contains removal date and migration path

**Example response:**
```json
{
  "interface_version": "1.0.0",
  "deprecation_notice": "validate_route.strict_mode will be removed in v2.0.0 (2027-03-11). Use strict_mode parameter in request instead.",
  "result": { ... }
}
```

**Agent action:**
```python
if response.get("deprecation_notice"):
    log_warning(response["deprecation_notice"])
    # Continue using — still works
    # Update your code to use replacement
```

### Phase 2: Removed (on announced date)

**What happens:**
- Operation no longer works
- Requests receive `NOT_IMPLEMENTED` error
- `interface_version` bumps to next Major (e.g., 2.0.0)

**Example error:**
```json
{
  "interface_version": "2.0.0",
  "error": {
    "code": "NOT_IMPLEMENTED",
    "message": "Operation 'old_operation' has been removed. Use 'new_operation' instead."
  }
}
```

**Agent action:**
- Must update to use replacement operation
- Must update expected version check

---

## Agent Version Checking

Agents **MUST** verify version before proceeding:

### Minimum Check (Required)

```python
def call_operation(operation: str, request: dict) -> dict:
    response = client.call_operation(operation, request)
    
    # REQUIRED: Check version first
    if response["interface_version"] != EXPECTED_VERSION:
        raise VersionMismatchError(
            f"Expected {EXPECTED_VERSION}, got {response['interface_version']}"
        )
    
    # Then process result or error
    ...
```

### Recommended: Version Range Check

```python
from packaging import version

EXPECTED_VERSION = "1.0.0"
COMPATIBLE_RANGE = ">=1.0.0,<2.0.0"

def check_version(response_version: str) -> bool:
    """Check if response version is compatible with agent."""
    return version.parse(response_version) in version.SpecifierSet(COMPATIBLE_RANGE)

def call_operation(operation: str, request: dict) -> dict:
    response = client.call_operation(operation, request)
    
    if not check_version(response["interface_version"]):
        raise IncompatibleVersion(
            f"Agent requires {COMPATIBLE_RANGE}, "
            f"server has {response['interface_version']}"
        )
    
    # Safe to proceed
    ...
```

---

## Release Schedule

### Regular Releases (Patch & Minor)

- **Schedule:** As needed (no fixed cadence)
- **Notice:** None required (backward compatible)
- **Agent action:** Update or ignore

### Major Releases (Breaking Changes)

- **Schedule:** Minimum 12 months between major versions
- **Notice:** 6 months advance notice with migration path
- **Agent action:** Required update before using interface

**Rationale:** Agents depend on interfaces; breaking changes must be announced with time to adapt.

---

## Migration Guide

### How to Update When MAJOR Version Changes

**Scenario:** v1.0.0 → v2.0.0 removes `validate_route` and replaces it with `validate_config`

**Steps:**

1. **Read deprecation notice** (from v1.x.x responses):
   ```
   validate_route will be removed in v2.0.0. 
   Use validate_config with identical parameters.
   ```

2. **During deprecation phase** (6 months before v2.0.0):
   ```python
   # OLD: Still works, but deprecated
   response = agent.call_operation("validate_route", {...})
   
   # NEW: Start using replacement
   response = agent.call_operation("validate_config", {...})
   ```

3. **After v2.0.0 release:**
   ```python
   # OLD: No longer works
   response = agent.call_operation("validate_route", {...})
   # Error: NOT_IMPLEMENTED
   
   # NEW: Must use
   response = agent.call_operation("validate_config", {...})
   ```

4. **Update version check:**
   ```python
   # Before:
   EXPECTED_VERSION = "1.0.0"
   
   # After:
   EXPECTED_VERSION = "2.0.0"
   COMPATIBLE_RANGE = ">=2.0.0,<3.0.0"
   ```

---

## Version Compatibility Matrix

| Agent Version | Server 1.0.0 | Server 1.1.0 | Server 1.0.1 | Server 2.0.0 |
|---------------|-------------|-------------|-------------|-------------|
| Agent for 1.0.0 | ✅ Works | ✅ Works | ✅ Works | ❌ Fail |
| Agent for 1.1.0 | ❌ Fail | ✅ Works | ✅ Works | ❌ Fail |
| Agent for 2.0.0 | ❌ Fail | ❌ Fail | ❌ Fail | ✅ Works |

**Legend:**
- ✅ Works: Agent and server compatible
- ❌ Fail: Incompatible; agent gets VERSION_MISMATCH error

---

## Deprecation Timeline Example

### Case: Removing `validate_route.strict_mode` in v2.0.0

**T=0 (Today):** v1.0.0 released
```python
# This works
response = agent.call_operation("validate_route", {
    "route_config_path": "...",
    "strict_mode": True  # Optional parameter
})
```

**T+3 months:** v1.1.0 released (minor: backward compatible)
```python
# Still works, but deprecation_notice in response
response = agent.call_operation("validate_route", {
    "route_config_path": "...",
    "strict_mode": True
})
# Response includes:
# "deprecation_notice": "strict_mode will be removed in v2.0.0 (2027-03-11)"
```

**T+6 months:** v2.0.0 released (major: breaking change)
```python
# OLD: No longer works
response = agent.call_operation("validate_route", {
    "route_config_path": "...",
    "strict_mode": True  # Parameter removed
})
# Error: INVALID_REQUEST - "strict_mode is not supported in v2.0.0"

# NEW: Mandatory update
response = agent.call_operation("validate_route", {
    "route_config_path": "..."
    # strict_mode moved to separate operation: strict_validate_route
})
```

---

## Stability Guarantees

The Agent Interface commits to:

### Guaranteed Stable (Until Major Bump)

- ✅ Operation names (within v1.x.x)
- ✅ Request field names (within v1.x.x)
- ✅ Error codes (within v1.x.x)
- ✅ Response structure (ResponseEnvelope format)
- ✅ Versioning scheme (SemVer)

### Not Guaranteed Stable (Can Change)

- ❌ CLI command names (`dimctl validate` vs `dimctl check`)
- ❌ Internal package API (`internal/config` API)
- ❌ Endpoint paths (implementation detail)
- ❌ Operation performance (speed may improve)

---

## Questions & Answers

**Q: What if I need a version newer than my agent supports?**  
A: You must update your agent's version check. Get the updated agent code from the reference implementation.

**Q: Can I ignore a deprecation notice?**  
A: For 6 months, yes — the operation continues to work. After the removal date, you must update.

**Q: What if a deprecation date passes and I haven't updated?**  
A: Your agent receives `NOT_IMPLEMENTED` errors for removed operations. Update immediately.

**Q: How often will Major versions change?**  
A: As needed, but never more than once per 12 months per major cycle. Breaking changes require 6 months notice.

**Q: Can I use an agent built for v1.0.0 with v1.5.0 server?**  
A: Yes. Your agent checks for v1.0.0 exactly, or v1.x.x in general (if using version range). Either way, backward compatibility is guaranteed.

**Q: What about security patches?**  
A: Security fixes bump patch version and are released immediately. No deprecation process.

---

## Summary

| Term | Policy |
|------|--------|
| Current Version | 1.0.0 |
| Next Major | Unknown (at least 1 year away) |
| Deprecation Notice | 6 months before removal |
| Breaking Changes | MAJOR version only |
| Backward Compat | Guaranteed until MAJOR bump |
| Version Checking | REQUIRED in every agent |

---

## References

- [Integration Guide](INTEGRATION_GUIDE.md) — How to use the interface
- [Worked Example](worked_example.md) — Real-world agent scenario
- [M4.6 Specification](../design/m46-agent-interface-spec.md) — Complete API spec
- [SemVer](https://semver.org) — Official semantic versioning spec
