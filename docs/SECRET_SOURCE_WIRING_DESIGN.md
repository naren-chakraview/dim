# Design: Wiring Domain-Scoped Secrets into File/SFTP Sources

Status: open design task. Follow-up to the T1.13 cross-domain secret fix
(fail-closed resolver, typed denial errors, construction-time reference validation).

## Current state (verified in code)

- `internal/factory/pipeline.go` builds no `FileSource` or SFTP source. Its only
  use of `adapters/file` is `file.NewFileSink`. Source construction there is HTTP-only.
- Nothing outside tests calls `NewFileSourceWithDomain` or `NewSFTPSourceWithDomain`.
  Unscoped `NewFileSource` / `NewSFTPSource` have no production callers either.
- No production `SecretStore` exists. `NewInMemoryStore` is used only in tests and in
  `cmd/dimctl/secret_audit.go`.
- Result: no live route is exposed today. The domain-scoped path is latent until a
  source type is wired into the factory and a secret store is populated from config.

## Problem

Wiring is not a one-line change. It needs three things that do not exist yet:

1. A route-spec source type (`file` / `sftp`) that the factory can dispatch on.
2. A secret store populated from configuration, with each entry's domain set.
3. A way to pass each route's domain and the shared resolver into source construction.

## Questions to answer

- What is the route-spec shape for file and SFTP sources (fields, where secrets are referenced)?
- Where does the secret store get its data: a secrets file, environment, or an external vault?
  Who owns registration of domain-scoped entries?
- Should unscoped constructors (`NewFileSource`, `NewSFTPSource`) be removed or kept
  behind an explicit opt-in? They still use the environment fallback.
- Password is the only secret-bearing source field today. Are key-file paths or other
  fields also secret references?

## Constraints carried over from the fix

- A denied cross-domain reference must never fall back to an environment variable.
- Validation runs at construction (`ValidateSecretReferences`); denials surface at
  poll time as `ErrCrossDomainDenied` if validation was bypassed.
- Resolver failures are returned as errors, never as partial output.

## Out of scope

- Changing the secret-domain policy itself (`domain.name` remains blocked).
