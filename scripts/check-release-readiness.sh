#!/bin/bash
# Pre-Release Readiness Checker
# Verifies that all requirements are met before creating a release tag
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

echo "🔍 Release Readiness Check"
echo "=========================="
echo ""

FAILED=0

# Check 1: Git status is clean
echo "✓ Checking git status..."
if [ -n "$(git -C "$PROJECT_ROOT" status --porcelain)" ]; then
	echo "  ✗ Working directory has uncommitted changes"
	FAILED=$((FAILED + 1))
else
	echo "  ✓ Working directory is clean"
fi

# Check 2: Current branch is main/master
echo "✓ Checking branch..."
BRANCH=$(git -C "$PROJECT_ROOT" rev-parse --abbrev-ref HEAD)
if [ "$BRANCH" != "master" ] && [ "$BRANCH" != "main" ]; then
	echo "  ✗ Not on main/master branch (current: $BRANCH)"
	FAILED=$((FAILED + 1))
else
	echo "  ✓ On $BRANCH branch"
fi

# Check 3: Up to date with remote
echo "✓ Checking remote sync..."
if ! git -C "$PROJECT_ROOT" diff --quiet HEAD origin/$(git -C "$PROJECT_ROOT" rev-parse --abbrev-ref HEAD); then
	echo "  ✗ Branch is not up to date with remote"
	FAILED=$((FAILED + 1))
else
	echo "  ✓ Up to date with remote"
fi

# Check 4: All tests pass
echo "✓ Running all tests..."
if ! cd "$PROJECT_ROOT" && go test ./... -timeout 30s >/dev/null 2>&1; then
	echo "  ✗ Tests failed"
	FAILED=$((FAILED + 1))
else
	echo "  ✓ All tests pass"
fi

# Check 5: Vet passes
echo "✓ Running go vet..."
if ! cd "$PROJECT_ROOT" && go vet ./... >/dev/null 2>&1; then
	echo "  ✗ Vet checks failed"
	FAILED=$((FAILED + 1))
else
	echo "  ✓ Vet checks pass"
fi

# Check 6: Build succeeds for all platforms
echo "✓ Building for all platforms..."
FAILED_BUILD=0
for OS in linux darwin windows; do
	for ARCH in amd64 arm64; do
		if ! GOOS=$OS GOARCH=$ARCH go build -o /tmp/dim-build ./cmd/dimd >/dev/null 2>&1; then
			echo "  ✗ Build failed for $OS/$ARCH"
			FAILED_BUILD=$((FAILED_BUILD + 1))
		fi
	done
done

if [ $FAILED_BUILD -eq 0 ]; then
	echo "  ✓ All platform builds succeed"
else
	FAILED=$((FAILED + 1))
fi

# Check 7: Version is properly set
echo "✓ Checking version..."
if grep -q "version" "$PROJECT_ROOT/pkg/sdk/version.go" 2>/dev/null; then
	echo "  ✓ Version defined in SDK"
else
	echo "  ⚠ Version not found in SDK (non-critical)"
fi

# Summary
echo ""
echo "=========================="
if [ $FAILED -eq 0 ]; then
	echo "✅ Release is ready!"
	echo ""
	echo "Next steps:"
	echo "  1. Create a semantic version tag: git tag v1.2.3"
	echo "  2. Push tag to remote: git push origin v1.2.3"
	echo "  3. Verify release workflow runs successfully"
	exit 0
else
	echo "❌ Release readiness check failed ($FAILED checks failed)"
	echo ""
	echo "Please fix the issues above before creating a release tag."
	exit 1
fi
