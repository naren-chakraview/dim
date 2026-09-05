# Compilation Errors Postmortem

## Summary
The M3.5 release pipeline discovered that the codebase had 30+ pre-existing compilation errors despite Phase 2 being marked "complete" with "all tests passing". These errors prevented `go build ./...` from succeeding.

## Root Causes

### 1. **API Refactoring Without Full Coverage**
The engine, adapters, and factory packages underwent API changes (likely between Phase 1 and Phase 2) that were not comprehensively applied:
- Old `engine.Engine` / `engine.NewEngine` still referenced in dimctl/replay.go
- Old `engine.Result` type still used in HTTP adapter (should be `adapters.Result`)
- Factory function return types changed but not all call sites updated
- RouteSpec model changed but contract validation code not updated

### 2. **Dependency Version Incompatibilities**
External dependencies had API incompatibilities with existing code:
- **Kafka (v0.4.51):** Missing fields (Compression, Timestamp, RequiredAcks), missing methods (Config(), ReadOffsetFromPartition)
- **AWS SDK v2:** S3 Client Close() doesn't exist, Body type changed from []byte to io.ReadCloser
- Code was written for different versions and never updated after dependency upgrades

### 3. **CI Not Running Full Builds**
The CI workflow (`go build ./...`) should have caught all these errors, but apparently was not executing or not blocking merges:
- CI status not visible in commit history
- Commits merged despite compilation failures
- No pre-commit hooks or merge gates preventing broken code

## Why It Wasn't Caught

1. **Tests Only**, Not **Build**
   - `go test ./...` was claimed to pass
   - But partial/dead code packages may not be executed in tests
   - `go build ./...` is more comprehensive than `go test ./...`

2. **Tests Passing ≠ Code Compiles**
   - Tests import and use only portions of the codebase
   - Unreferenced code (replay.go unused paths, S3 source functions) wouldn't be tested
   - `go build ./cmd/dimd && go build ./cmd/dimctl` only builds mains, not all packages

3. **No Merge Gate**
   - CI results not enforced at merge time
   - PRs merged without CI passing
   - No "must pass" requirement before merging

## Fixes Applied

### Code Fixes
1. S3 adapter: proper io.ReadCloser handling, removed Close()
2. Kafka adapter: removed unsupported fields/methods, version-compatible code
3. HTTP adapter: correct Result type imports
4. dimd/dimctl: fixed factory function calls, stubbed unsupported APIs
5. Removed unused imports throughout

### Process Fixes (This Document)
This postmortem documents the incident. Future: implement merge gates.

## Prevention: Merge Gate Implementation

To prevent this in the future, we need:

1. **CI Enforcement** (`.github/workflows/ci.yml`)
   - Add `go build ./...` explicitly (currently implicit in cross-compile)
   - Run `go vet ./...` 
   - Run `golangci-lint` for additional checks
   - Make CI results required for merge

2. **Pre-Commit Hooks** (`.git/hooks/pre-commit`)
   - Local developer sanity checks before push
   - `go build ./...` must succeed
   - `go vet ./...` must pass

3. **Branch Protection** (GitHub)
   - Require CI checks to pass before merge
   - Require code reviews

4. **Additional Linters** (`golangci-lint`)
   - Unused imports/variables
   - Ineffectual assignments
   - Type mismatches

## Lessons Learned

1. **"Tests pass" doesn't mean "code compiles"** — Always run full build checks
2. **Dead code detection matters** — Unused code paths hide compilation errors until relied upon
3. **Dependency upgrades need full testing** — API changes must be applied to all call sites
4. **Merge gates are essential** — Code that doesn't compile should never be merged
5. **Verify what "complete" actually means** — Phase 2 claimed complete, but had blocking issues

## Files Affected

- `internal/adapters/s3/s3_source.go` — 10 lines changed
- `internal/adapters/s3/s3_sink.go` — 3 lines changed
- `internal/adapters/kafka/kafka_sink.go` — 15 lines changed
- `internal/adapters/kafka/kafka_source.go` — 15 lines changed
- `internal/adapters/http/http_sink.go` — 2 lines changed
- `cmd/dimd/main.go` — 12 lines changed
- `cmd/dimctl/main.go` — 8 lines changed
- `cmd/dimctl/replay.go` — 18 lines changed

**Total:** ~83 lines across 8 files

## Follow-Up Actions
See `.github/workflows/ci.yml` and `pre-commit` hook configuration for merge gate implementation.
