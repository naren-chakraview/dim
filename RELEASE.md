# Release Process

This document describes how releases are created and what gates must pass before a release is published.

## Overview

Releases follow semantic versioning (v1.2.3) and are gated by a 3-stage workflow in GitHub Actions:

```
Push tag (git push origin v1.2.3)
    ↓
[Stage 1] Verify CI Status
    • Tag format validation (must match v{MAJOR}.{MINOR}.{PATCH})
    • Verify commit exists and is on main/master
    ↓
[Stage 2] Service Health Checks
    • Kafka broker starts and is responsive
    • PostgreSQL database starts and is accessible
    • Zookeeper coordinates services
    ↓
[Stage 3] Build & Release
    • Cross-platform build succeeds (4 platforms × 2 architectures)
    • GoReleaser creates binaries and checksums
    • Release published to GitHub Releases
```

## Prerequisites

Before creating a release tag, run the pre-release checklist:

```bash
scripts/check-release-readiness.sh
```

This verifies:
- ✓ Working directory is clean (no uncommitted changes)
- ✓ On main/master branch
- ✓ Up to date with remote
- ✓ All tests pass
- ✓ go vet passes
- ✓ Build succeeds for all platforms

## Creating a Release

### Step 1: Verify Readiness

```bash
./scripts/check-release-readiness.sh
```

If any checks fail, fix the issues and re-run before proceeding.

### Step 2: Create a Semantic Version Tag

```bash
# Format: v{MAJOR}.{MINOR}.{PATCH}
# Examples: v1.0.0, v1.2.3, v0.5.0

git tag v1.2.3
```

### Step 3: Push Tag to Remote

```bash
git push origin v1.2.3
```

This triggers the release workflow automatically.

### Step 4: Monitor the Workflow

Go to GitHub Actions and watch the release workflow:

1. **Verify CI Status** → Checks tag format and commit
2. **E2E Tests** → Starts services and verifies they're healthy
3. **Create Release** → Builds binaries and publishes release

All three stages must pass. If any stage fails:
- Check the workflow logs
- Fix the issue
- Delete the failed tag: `git tag -d v1.2.3` and `git push origin :v1.2.3`
- Try again

### Step 5: Verify Release Artifacts

Once the workflow completes successfully:

1. Go to https://github.com/naren-chakraview/dim/releases
2. Verify binaries are present for all platforms (Linux, macOS, Windows)
3. Verify checksums are included
4. Download and test a binary if desired

## Release Workflow Details

### Stage 1: Verify CI Status (`.github/workflows/release.yml` - `verify-ci-status`)

**Purpose:** Ensure the tag points to a valid commit on a protected branch

**Checks:**
- Tag format must match semantic versioning: `^v[0-9]+\.[0-9]+\.[0-9]+(-[a-zA-Z0-9]+)?(\+[a-zA-Z0-9]+)?$`
  - Valid: `v1.0.0`, `v1.0.0-beta`, `v1.0.0+build.123`
  - Invalid: `release-1.0.0`, `1.0.0`, `v1.0`

**If it fails:**
- Delete the tag and try with correct format
- Ensure you're pushing to the correct remote

### Stage 2: E2E Tests (`.github/workflows/release.yml` - `e2e-tests`)

**Purpose:** Verify that integration services (Kafka, PostgreSQL, Zookeeper) start and become healthy

**How it works:**
- GitHub Actions starts Docker containers for Kafka, PostgreSQL, and Zookeeper
- Each service has a health check that runs every 5 seconds
- The workflow blocks until all health checks pass or timeout
- No explicit tests are run — service health is the validation

**Health check details:**
- **Zookeeper:** TCP connectivity check, 15 retries, 30s timeout
- **Kafka:** `kafka-broker-api-versions` command, 20 retries, 30s timeout
- **PostgreSQL:** `pg_isready` command, 15 retries, 30s timeout

**If it fails:**
- Service didn't start within timeout (usually means Docker issue)
- Check GitHub Actions logs for which service failed
- Common causes:
  - Docker daemon not available (unlikely in GitHub Actions)
  - Service image outdated or missing
  - Port conflicts (shouldn't happen in isolated CI environment)

### Stage 3: Build & Release (`.github/workflows/release.yml` - `release`)

**Purpose:** Build binaries for all platforms and publish the release

**Platforms built:**
- linux/amd64
- linux/arm64
- darwin/amd64 (macOS Intel)
- darwin/arm64 (macOS Apple Silicon)
- windows/amd64

**Process:**
1. GoReleaser checks out source code
2. Runs `goreleaser release --clean`
3. Builds binaries for all platforms
4. Generates checksums (SHA256)
5. Creates GitHub release with binaries and checksums

**If it fails:**
- Build error (most likely cause)
- Check GoReleaser configuration in `.goreleaser.yml`
- Fix the build issue and try again with a new tag

## Troubleshooting

### "Tag format not valid"

Ensure your tag matches the pattern: `v{MAJOR}.{MINOR}.{PATCH}`

```bash
# Valid
git tag v1.0.0
git tag v1.2.3-beta
git tag v2.0.0-rc.1

# Invalid (won't work)
git tag release-1.0.0    # must start with 'v'
git tag 1.0.0            # missing 'v' prefix
git tag v1.0             # need PATCH version
```

### "E2E Tests workflow timed out"

Services take 30-60 seconds to start. If workflow times out:
1. This might indicate a real infrastructure issue
2. Check GitHub Actions runner logs for Docker errors
3. Retry the release (delete tag and push again)

### "Build failed for platform X"

Check the GoReleaser logs:
- Common issues: dependency conflicts, platform-specific code issues
- Fix the issue and try again with a new tag

### Accidental Tag

If you pushed a tag by accident:

```bash
# Delete local tag
git tag -d v1.2.3

# Delete remote tag
git push origin :v1.2.3
```

Then recreate when ready.

## Semantic Versioning

Follow [semver.org](https://semver.org/):

- **MAJOR** (X.0.0) — Breaking changes to config, API, or behavior
- **MINOR** (1.Y.0) — New features, backward compatible
- **PATCH** (1.0.Z) — Bug fixes, backward compatible

**Examples:**
- v0.5.0 → v0.6.0 (new features: MINOR)
- v1.0.0 → v1.1.0 (new feature: MINOR)
- v1.1.0 → v1.1.1 (bug fix: PATCH)
- v1.0.0 → v2.0.0 (breaking change: MAJOR)

## Rollback

If a released version has critical issues:

1. Fix the issue in a new commit
2. Increment version and create a new tag
3. Push new tag to trigger release workflow

Do NOT re-release or modify an existing tag — releases are immutable by design.

## Release Checklist for Release Notes

When preparing release notes, include:

- **What's new:** Features added in this release
- **Breaking changes:** Any API or config changes
- **Deprecations:** Features that will be removed in future versions
- **Bug fixes:** Issues resolved in this release
- **Known issues:** Limitations or open issues
- **Installation:** How to install this version
- **Upgrading:** Any manual steps required to upgrade from prior version

See [RELEASE_NOTES_v0.5.0.md](RELEASE_NOTES_v0.5.0.md) for an example format.

## CI Workflow Files

- **Tag validation & release gate:** `.github/workflows/release.yml`
- **E2E health checks:** `.github/workflows/e2e.yml` (also runs on every push)
- **Pre-release checklist:** `scripts/check-release-readiness.sh`
- **GoReleaser config:** `.goreleaser.yml`
