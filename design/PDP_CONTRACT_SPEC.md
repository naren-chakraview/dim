# PDP (Policy Decision Point) Contract Specification

**Version:** 1.0  
**Date:** 2026-09-04  
**Status:** Formal specification for Phase 1 (M1.1)  
**Author:** dim Team

## Overview

This document defines the neutral, engine-agnostic contract between dim and a Policy Decision Point (PDP). A PDP is an external service or embedded engine that makes authorization decisions (allow/deny) and returns obligations (data redaction, audit requirements, etc.).

This specification is intentionally **not OPA-specific**, enabling bring-your-own-PDP implementations. OPA is the reference implementation, but other engines (AWS ABAC, Azure AD, custom policy systems) can conform to this contract.

## 1. Core Concepts

### 1.1 Decision Request

A **decision request** is sent from dim to the PDP when an `authorize` step with `mode: pbac` needs a policy decision. It contains:

- **Principal**: Who is making the request (authenticated identity)
- **Action**: What they're trying to do (e.g., "process_order", "export_data")
- **Resource**: What they're acting on (the message context)
- **Context**: Additional context (route, stage, message metadata)

### 1.2 Decision Response

A **decision response** from the PDP contains:

- **Decision**: `allow` or `deny`
- **Obligations**: Additional requirements if allowed (data redaction, audit logging, etc.)
- **Reason**: Human-readable explanation (for audit logs)
- **Metadata**: Machine-readable policy reference (for compliance)

### 1.3 Obligations

**Obligations** are conditions the PDP imposes on an allowed decision:

- **Data Redaction**: Remove specific fields from the message body
- **Audit Logging**: Write additional audit records
- **Rate Limiting**: Throttle the request
- **Data Classification**: Tag the message with sensitivity level

## 2. Contract Specification (OpenAPI 3.0)

```yaml
openapi: 3.0.0
info:
  title: dim Policy Decision Point (PDP) Contract
  version: "1.0"
  description: "Neutral, engine-agnostic authorization contract for dim"

servers:
  - url: http://localhost:8181  # Default OPA endpoint
    description: PDP server

paths:
  /v1/data/dim/authorize:
    post:
      summary: Request authorization decision
      description: |
        Request a policy decision from the PDP. The PDP evaluates
        the principal, action, resource, and context against policies
        and returns an allow/deny decision with optional obligations.
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: "#/components/schemas/DecisionRequest"
      responses:
        "200":
          description: Policy decision with obligations
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/DecisionResponse"
        "400":
          description: Malformed request
        "500":
          description: PDP internal error

components:
  schemas:
    DecisionRequest:
      type: object
      required:
        - principal
        - action
        - resource
      properties:
        principal:
          $ref: "#/components/schemas/Principal"
        action:
          type: string
          description: |
            The action being performed. Examples:
            - process_order
            - export_data
            - modify_route
            - query_lineage
        resource:
          $ref: "#/components/schemas/Resource"
        context:
          $ref: "#/components/schemas/Context"
      example:
        principal:
          subject: user@example.com
          roles:
            - seller
            - partner
        action: process_order
        resource:
          type: message
          id: msg-12345
          route: ingest-orders
        context:
          timestamp: "2026-09-04T12:34:56Z"
          correlation_id: corr-98765

    Principal:
      type: object
      required:
        - subject
      properties:
        subject:
          type: string
          description: Unique identifier (email, service account, etc.)
        roles:
          type: array
          items:
            type: string
          description: |
            Roles assigned to this principal.
            May be empty for role-agnostic policies.
            Examples: ["seller", "admin", "viewer"]
        attributes:
          type: object
          additionalProperties: true
          description: |
            Additional attributes about the principal.
            Examples: department, team, region, data_classification_level

    Resource:
      type: object
      required:
        - type
      properties:
        type:
          type: string
          description: |
            Resource type being accessed.
            Examples: message, route, lineage
        id:
          type: string
          description: Resource identifier (message ID, route name, etc.)
        classification:
          type: string
          description: |
            Data classification of this resource.
            Examples: public, internal, confidential, pci, phi
        route:
          type: string
          description: Route name (for message resources)
        stage:
          type: string
          description: Pipeline stage where decision is being made

    Context:
      type: object
      properties:
        timestamp:
          type: string
          format: date-time
          description: When the authorization was requested
        correlation_id:
          type: string
          description: Message correlation ID for audit traceability
        environment:
          type: string
          description: Deployment environment (dev, staging, prod)
        additional_context:
          type: object
          additionalProperties: true
          description: Policy-specific context (key-value pairs)

    DecisionResponse:
      type: object
      required:
        - decision
      properties:
        decision:
          type: string
          enum:
            - allow
            - deny
          description: Authorization decision
        obligations:
          type: array
          items:
            $ref: "#/components/schemas/Obligation"
          description: |
            Obligations that apply if decision is allow.
            Empty if no obligations.
        reason:
          type: string
          description: |
            Human-readable explanation of the decision.
            Used for audit logs and debugging.
            Examples:
            - "User lacks required role: admin"
            - "Data classification exceeds clearance level"
            - "Rate limit exceeded for this principal"
        metadata:
          type: object
          additionalProperties: true
          description: |
            Machine-readable metadata for compliance.
            Examples:
              policy_id: "opa-policy-v2-1-3"
              rule_name: "data_classification_check"
              trace_id: "trace-abc123"

    Obligation:
      type: object
      required:
        - type
      properties:
        type:
          type: string
          enum:
            - redact_fields
            - audit_log
            - rate_limit
            - data_classification_tag
          description: Type of obligation
        parameters:
          type: object
          additionalProperties: true
          description: |
            Type-specific parameters.
            Examples (by type):
            - redact_fields: { fields: ["ssn", "credit_card"] }
            - audit_log: { level: "INFO", tags: ["high-risk"] }
            - rate_limit: { requests_per_minute: 100 }
            - data_classification_tag: { level: "confidential" }
```

## 3. JSON Schema (for validation)

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "title": "dim PDP Contract",
  "type": "object",
  "definitions": {
    "principal": {
      "type": "object",
      "required": ["subject"],
      "properties": {
        "subject": { "type": "string" },
        "roles": {
          "type": "array",
          "items": { "type": "string" }
        },
        "attributes": {
          "type": "object",
          "additionalProperties": true
        }
      }
    },
    "resource": {
      "type": "object",
      "required": ["type"],
      "properties": {
        "type": { "type": "string" },
        "id": { "type": "string" },
        "classification": { "type": "string" },
        "route": { "type": "string" },
        "stage": { "type": "string" }
      }
    },
    "context": {
      "type": "object",
      "properties": {
        "timestamp": { "type": "string", "format": "date-time" },
        "correlation_id": { "type": "string" },
        "environment": { "type": "string" },
        "additional_context": {
          "type": "object",
          "additionalProperties": true
        }
      }
    },
    "obligation": {
      "type": "object",
      "required": ["type"],
      "properties": {
        "type": {
          "type": "string",
          "enum": ["redact_fields", "audit_log", "rate_limit", "data_classification_tag"]
        },
        "parameters": {
          "type": "object",
          "additionalProperties": true
        }
      }
    },
    "decisionRequest": {
      "type": "object",
      "required": ["principal", "action", "resource"],
      "properties": {
        "principal": { "$ref": "#/definitions/principal" },
        "action": { "type": "string" },
        "resource": { "$ref": "#/definitions/resource" },
        "context": { "$ref": "#/definitions/context" }
      }
    },
    "decisionResponse": {
      "type": "object",
      "required": ["decision"],
      "properties": {
        "decision": {
          "type": "string",
          "enum": ["allow", "deny"]
        },
        "obligations": {
          "type": "array",
          "items": { "$ref": "#/definitions/obligation" }
        },
        "reason": { "type": "string" },
        "metadata": {
          "type": "object",
          "additionalProperties": true
        }
      }
    }
  }
}
```

## 4. Example Flows

### 4.1 Allow with Obligations

```json
{
  "request": {
    "principal": {
      "subject": "alice@example.com",
      "roles": ["seller", "partner"]
    },
    "action": "process_order",
    "resource": {
      "type": "message",
      "id": "order-12345",
      "route": "order-processing",
      "classification": "internal"
    },
    "context": {
      "timestamp": "2026-09-04T12:34:56Z",
      "correlation_id": "corr-98765",
      "environment": "prod"
    }
  },
  "response": {
    "decision": "allow",
    "obligations": [
      {
        "type": "audit_log",
        "parameters": {
          "level": "INFO",
          "tags": ["order-processing", "seller-action"]
        }
      }
    ],
    "reason": "User has seller role; order processing allowed",
    "metadata": {
      "policy_id": "opa-order-processing-v1",
      "rule_name": "seller_can_process_own_orders"
    }
  }
}
```

### 4.2 Deny Decision

```json
{
  "request": {
    "principal": {
      "subject": "bob@example.com",
      "roles": ["viewer"]
    },
    "action": "export_data",
    "resource": {
      "type": "message",
      "id": "pii-record-456",
      "classification": "phi"
    },
    "context": {
      "timestamp": "2026-09-04T12:34:56Z",
      "correlation_id": "corr-11111"
    }
  },
  "response": {
    "decision": "deny",
    "reason": "Viewer role cannot export PHI-classified data",
    "metadata": {
      "policy_id": "opa-data-protection-v2",
      "rule_name": "phi_export_restriction"
    }
  }
}
```

## 5. Versioning

**Contract Version:** 1.0

- **Major versions** (1.x → 2.x) indicate breaking changes to request/response structure
- **Minor versions** (1.0 → 1.1) indicate backward-compatible additions (new obligation types, new context fields)
- **Patch versions** (1.0 → 1.0.1) indicate clarifications with no semantic changes

The PDP and dim must agree on contract version before communication begins.

## 6. BYO-PDP Conformance Guide

To implement a policy engine that conforms to this contract:

### 6.1 Minimum Requirements

1. **HTTP/REST endpoint** accepting POST requests
2. **Accept** decision requests matching the DecisionRequest schema
3. **Return** decision responses matching the DecisionResponse schema
4. **Validate** that `principal.subject` and `action` are non-empty strings
5. **Support** at least one obligation type (`audit_log`)

### 6.2 Recommended Enhancements

1. **Support multiple obligation types** (redact_fields, rate_limit, data_classification_tag)
2. **Include reason** and metadata in every response
3. **Handle context.additional_context** for domain-specific policies
4. **Implement request timeouts** (recommend: 500ms-2s)
5. **Log policy evaluations** for debugging and compliance

### 6.3 Error Handling

- Return HTTP 400 for malformed requests (invalid JSON, missing required fields)
- Return HTTP 500 for internal errors (policy syntax errors, backend unavailable)
- Include error description in response body:
  ```json
  {
    "error": "policy_syntax_error",
    "message": "Invalid Rego rule at line 42",
    "request_id": "req-abc123"
  }
  ```

### 6.4 Testing Your Implementation

Test that your PDP correctly:

1. **Accepts valid DecisionRequests**
2. **Rejects invalid requests** with HTTP 400
3. **Returns decisions** matching DecisionResponse schema
4. **Supports obligations** (at least audit_log)
5. **Provides reason and metadata** for traceability
6. **Handles edge cases** (empty roles, missing context fields, null principal attributes)

## 7. Design Rationale

### Why engine-agnostic?

- Organizations have existing policy infrastructure (AWS ABAC, Azure AD, OPA, Rego, custom)
- No single engine fits all use cases
- This contract enables bring-your-own-PDP without requiring deep dim knowledge

### Why REST/HTTP?

- Simplifies deployment (separate service, sidecary, embedded)
- Language-agnostic (any language can implement this endpoint)
- Integrates with existing monitoring/observability

### Why obligations?

- Decisions aren't binary: allow-with-redaction is different from allow-all-data
- Enables fine-grained security (rate limits, classifications, audit requirements)
- Separates policy decision from policy enforcement (dim applies obligations)

## 8. Reference Implementation

OPA reference implementation (M1.2) will:
- Deploy OPA standalone or embedded
- Accept this contract's DecisionRequests
- Evaluate Rego policies against principal, action, resource
- Return DecisionResponses with obligations extracted from policy output

See `M1.2_PBAC_OPA_REFERENCE.md` for OPA-specific implementation details.

## 9. Version History

| Version | Date | Changes |
|---------|------|---------|
| 1.0 | 2026-09-04 | Initial specification: decision request/response, obligations, BYO-PDP conformance |

## 10. Questions & Feedback

This specification is open for feedback. Submit issues or questions to:
- GitHub Issues: https://github.com/naren-chakraview/dim/issues
- Design Review: design/

## Appendix: Real-World Policy Examples

### Example 1: Role-Based Data Access

```yaml
# Policy: Users can only access data in their region
policy:
  - if: principal.attributes.region == resource.region
    then: allow
  - obligations:
      - audit_log: { level: "INFO" }
  - else: deny
```

### Example 2: Classification-Based Redaction

```yaml
# Policy: Viewer role can read, but PII fields are redacted
policy:
  - if: principal.roles.contains("viewer") and resource.classification == "pii"
    then:
      - allow
      - obligations:
          - redact_fields: { fields: ["ssn", "credit_card", "phone"] }
  - else: deny
```

### Example 3: Rate Limiting by Role

```yaml
# Policy: Free tier gets rate limits, premium doesn't
policy:
  - if: principal.attributes.tier == "free"
    then:
      - allow
      - obligations:
          - rate_limit: { requests_per_minute: 100 }
  - else: allow  # Premium: no rate limit
```
