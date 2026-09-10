# Task 3: Integrate route validation into CI pipeline

## Files to Modify

- Modify: `.github/workflows/ci.yml`

## Interfaces

- Consumes: Scripts created by Task 2 (`scripts/validate-routes.sh`, `scripts/test-routes.sh`, `scripts/check-mandatory-fragments.sh`)
- Produces: Updated CI pipeline with route validation as merge gates (MERGE GATE 8, 9, 10)

## Steps to Execute

### Step 1: Open .github/workflows/ci.yml for editing

The file currently has 7 merge gates (build, vet, test, race, vulnerabilities, cross-compile, upload).
You will add 3 new gates between "Test" and "Cross-compile" steps.

### Step 2: Locate the insertion point

Find this section in ci.yml:

```yaml
      # MERGE GATE 5: Race detector must pass
      - name: Test with race detector
        run: go test -race ./...

      # MERGE GATE 6: Vulnerability check
      - name: Check for vulnerabilities
        run: go run golang.org/x/vuln/cmd/govulncheck@latest ./...
```

After the "Test with race detector" step but BEFORE "Check for vulnerabilities", insert the three new gates.

### Step 3: Add MERGE GATE 8 (Validate route configurations)

Insert this block:

```yaml
      # MERGE GATE 8: Validate all route YAML files changed in this PR
      - name: Validate route configurations
        run: bash scripts/validate-routes.sh
        if: |
          github.event_name == 'pull_request' ||
          github.ref == 'refs/heads/main' ||
          github.ref == 'refs/heads/master'
```

### Step 4: Add MERGE GATE 9 (Test route configurations)

Insert this block (after MERGE GATE 8):

```yaml
      # MERGE GATE 9: Test all route configurations with their fixture tests
      - name: Test route configurations
        run: bash scripts/test-routes.sh
        if: |
          github.event_name == 'pull_request' ||
          github.ref == 'refs/heads/main' ||
          github.ref == 'refs/heads/master'
```

### Step 5: Add MERGE GATE 10 (Mandatory governance fragment check)

Insert this block (after MERGE GATE 9):

```yaml
      # MERGE GATE 10: Mandatory governance fragment check
      - name: Check mandatory governance fragments
        run: bash scripts/check-mandatory-fragments.sh
        if: |
          github.event_name == 'pull_request' ||
          github.ref == 'refs/heads/main' ||
          github.ref == 'refs/heads/master'
```

### Step 6: Verify workflow syntax

Run GitHub's actionlint to verify the workflow is valid:

```bash
# Install or use pre-installed actionlint
go run github.com/rhysd/actionlint/cmd/actionlint@latest .github/workflows/ci.yml
```

Expected: No errors or critical issues reported.

If actionlint is not available, verify the YAML syntax manually:

```bash
python3 -c "import yaml; yaml.safe_load(open('.github/workflows/ci.yml'))" && echo "✅ YAML valid"
```

Or use a YAML linter:

```bash
docker run --rm -v $(pwd):/workspace sdehaes/yamllint .github/workflows/ci.yml
```

The key is to ensure the YAML is structurally correct.

### Step 7: Verify the script references work

```bash
# Ensure the scripts exist and are executable
test -x scripts/validate-routes.sh && echo "✅ validate-routes.sh exists and is executable"
test -x scripts/test-routes.sh && echo "✅ test-routes.sh exists and is executable"
test -x scripts/check-mandatory-fragments.sh && echo "✅ check-mandatory-fragments.sh exists and is executable"
```

### Step 8: Commit the updated CI workflow

```bash
git add .github/workflows/ci.yml
git commit -m "ci: add route validation and testing gates for M4.1"
```

## Success Criteria

- `.github/workflows/ci.yml` is updated with three new steps
- Steps are named: "Validate route configurations", "Test route configurations", "Check mandatory governance fragments"
- Each step includes the `if:` condition (runs on PR and main/master pushes)
- Step commands reference the scripts created in Task 2: `bash scripts/validate-routes.sh`, etc.
- YAML syntax is valid (no errors from actionlint or YAML parser)
- Commit message is: "ci: add route validation and testing gates for M4.1"
- Git log shows the commit

## Notes

- The `if:` condition ensures these gates only run when it matters (PR merges and main/master pushes), not on every push to feature branches
- The gate numbering (8, 9, 10) follows from the existing 7 gates; adjust comments if needed
- These gates run AFTER the Go build/test gates but BEFORE the cross-compile step, ensuring route issues are caught early
