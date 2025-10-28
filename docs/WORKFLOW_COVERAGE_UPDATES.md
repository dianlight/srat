# Workflow Coverage Updates Implementation

## Overview

The CI/CD workflow has been restructured to separate test execution from coverage badge updates. This improves efficiency and allows coverage data to flow from test jobs to a dedicated update job.

## Key Changes

### 1. Modified Script: `scripts/update-coverage-badges.sh`

**New Features:**
- **CLI Arguments Support**: Script now accepts `--backend COVERAGE` and `--frontend COVERAGE` parameters
- **Conditional Test Execution**: Tests are skipped if coverage values are provided via CLI
- **Input Validation**: Validates that provided coverage values are numeric with optional decimals
- **Backward Compatible**: Running without arguments still executes tests as before

**Usage Examples:**
```bash
# Run tests and update badges (original behavior)
./scripts/update-coverage-badges.sh

# Use provided coverage values without running tests (new behavior)
./scripts/update-coverage-badges.sh --backend 45.2 --frontend 78.5
```

### 2. Enhanced `test-backend` Job

**New Output:**
- Captures test output to `backend/backend_test_output.txt`
- Extracts coverage percentage from test output
- Exports coverage as job output: `coverage`

**Process:**
1. Runs `make test` and captures output
2. Extracts line matching "Total coverage: XX.X%"
3. Stores numeric value as step output for downstream jobs

### 3. Enhanced `test-frontend` Job

**New Output:**
- Captures test output to `frontend/frontend_test_output.txt`
- Extracts coverage percentage from test output table
- Exports coverage as job output: `coverage`

**Process:**
1. Runs `bun test:ci` and captures output
2. Extracts line matching "All files" from coverage table
3. Stores numeric value as step output for downstream jobs

### 4. New `update-coverage` Job

**Purpose:** Centralized coverage badge and documentation updates

**Dependencies:**
- Depends on: `setversion`, `test-backend`, `test-frontend`
- Depended on by: `build`

**Workflow:**
1. Receives coverage data from test jobs as inputs
2. Sets up necessary tools (Go, Bun)
3. Calls `update-coverage-badges.sh` with CLI parameters (no test re-execution)
4. Runs `make docs` for documentation generation
5. Detects changes in `README.md` and `docs/TEST_COVERAGE.md`
6. Commits and pushes changes if any were made (main branch only)

**Git Configuration:**
- Only commits on non-pull-request events
- Uses `github-actions[bot]` as committer
- Includes pull-and-rebase to avoid conflicts

### 5. Updated `build` Job

**Change:** 
- Modified dependency chain from `needs: [setversion, test-backend, test-frontend]`
- To: `needs: [setversion, test-backend, test-frontend, update-coverage]`

**Benefit:** Ensures coverage updates complete before build starts

## Execution Flow

```
setversion
    ↓
    ├→ test-backend (outputs: coverage)
    ├→ test-frontend (outputs: coverage)
    │
    ├→ update-coverage (reads: test outputs, writes: README.md, TEST_COVERAGE.md)
    │   ↓
    └→ build (builds artifacts)
       ↓
    create-release
```

## Benefits

1. **Efficiency**: Tests run once; coverage data flows to badge update job
2. **Separation of Concerns**: Test jobs focus on testing, update job on documentation
3. **Flexibility**: Script can be used standalone with coverage values
4. **No Redundant Execution**: Badge updates don't re-run tests
5. **Atomic Changes**: All coverage files updated in single commit
6. **Error Handling**: Validates coverage inputs before processing

## Testing the Changes

### Locally
```bash
# Test with provided coverage values
bash scripts/update-coverage-badges.sh --backend 35.5 --frontend 72.3

# Test with test execution (original behavior)
bash scripts/update-coverage-badges.sh
```

### In CI
The workflow will automatically:
1. Extract coverage from test outputs
2. Pass values to update-coverage job
3. Update documentation without re-running tests
4. Commit changes automatically

## Files Modified

- `scripts/update-coverage-badges.sh` - Added CLI argument parsing and conditional test execution
- `.github/workflows/build.yaml` - Enhanced test jobs with outputs, added update-coverage job, updated build dependency
