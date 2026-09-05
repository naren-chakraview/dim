# Contributing to dim

## Merge Gates & Quality Standards

This project enforces strict code quality standards through CI merge gates and pre-commit hooks. **All checks must pass before code can be merged.**

### Setting up Pre-Commit Hooks

When you first clone the repository, configure git to use the project's hooks:

```bash
git config core.hooksPath .githooks
```

This enables automatic checks before each commit.

### Merge Gate Checks

These checks run in CI and **must pass** before a PR can merge:

1. **Build**: `go build ./...` - Full codebase compilation
2. **Vet**: `go vet ./...` - Go static analysis
3. **Lint**: `golangci-lint run` - Comprehensive linting
4. **Test**: `go test ./...` - All unit tests
5. **Race**: `go test -race ./...` - Race condition detection
6. **Vulnerabilities**: `govulncheck ./...` - Dependency vulnerability scan
7. **Cross-compile**: Build for linux/amd64, linux/arm64, darwin/amd64, darwin/arm64, windows/amd64

### Local Pre-Commit Checks

The `.githooks/pre-commit` script runs locally before you can push and checks:

- ✓ Code compiles
- ✓ `go vet` passes
- ✓ Tests pass
- ⚠ Lists untracked files

If any check fails, fix the issue and re-run `git commit`.

### Running Checks Manually

```bash
# Full build
go build ./...

# Static analysis
go vet ./...

# Comprehensive linting
golangci-lint run

# Unit tests
go test ./...

# Race detector
go test -race ./...

# Vulnerability check
go run golang.org/x/vuln/cmd/govulncheck@latest ./...

# Cross-compile
GOOS=linux GOARCH=amd64 go build ./cmd/dimd
GOOS=linux GOARCH=arm64 go build ./cmd/dimd
GOOS=darwin GOARCH=amd64 go build ./cmd/dimd
GOOS=darwin GOARCH=arm64 go build ./cmd/dimd
GOOS=windows GOARCH=amd64 go build ./cmd/dimd
```

## Why These Gates Exist

The project once shipped code that didn't compile (see `COMPILATION_ERRORS_POSTMORTEM.md`). These gates prevent that:

- **Build check** catches all compilation errors immediately
- **Vet + Lint** catch type mismatches, unused variables, API incompatibilities
- **Test + Race** find logic errors and concurrency bugs
- **Vulnerability check** catches known dependency exploits
- **Cross-compile** ensures binaries work on all supported platforms

## Before Opening a PR

1. Configure pre-commit hooks: `git config core.hooksPath .githooks`
2. Make your changes
3. Run `go build ./...` to verify compilation
4. Run `go test ./...` to verify tests pass
5. Run `golangci-lint run` to catch style/quality issues
6. Commit and push (pre-commit hook will run again)

## CI Requirements

Every PR must pass all CI checks before merge. The CI workflow is defined in `.github/workflows/ci.yml` and runs:

- On every push to main/master
- On every pull request against main/master
- Results are required for merge (enforced by GitHub branch protection)

## Questions?

Refer to:
- `COMPILATION_ERRORS_POSTMORTEM.md` - Why these standards exist
- `.github/workflows/ci.yml` - Full CI pipeline definition
- `.githooks/pre-commit` - Local pre-commit checks
- `.golangci.yml` - Linter configuration
