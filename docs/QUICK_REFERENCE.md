<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->
**Table of Contents**  *generated with [DocToc](https://github.com/thlorenz/doctoc)*

- [Quick Reference Guide - Coverage Updates Implementation](#quick-reference-guide---coverage-updates-implementation)
  - [What Changed?](#what-changed)
    - [📝 Script (`scripts/update-coverage-badges.sh`)](#-script-scriptsupdate-coverage-badgessh)
    - [🔄 Workflow (`.github/workflows/build.yaml`)](#-workflow-githubworkflowsbuildyaml)
  - [Usage Examples](#usage-examples)
    - [Local: Skip tests, just update badges](#local-skip-tests-just-update-badges)
    - [Local: Run tests and update badges](#local-run-tests-and-update-badges)
    - [Local: Invalid input (error handling)](#local-invalid-input-error-handling)
  - [CI/CD Flow](#cicd-flow)
  - [Key Points](#key-points)
  - [Testing](#testing)
  - [Files Modified](#files-modified)
  - [Documentation](#documentation)
  - [Status](#status)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

# Quick Reference Guide - Coverage Updates Implementation

## What Changed?

### 📝 Script (`scripts/update-coverage-badges.sh`)

- Now accepts `--backend VALUE` and `--frontend VALUE` arguments
- Skips tests if both values provided
- Validates numeric input
- Still works without arguments (backward compatible)

### 🔄 Workflow (`.github/workflows/build.yaml`)

- `test-backend` job now outputs coverage value
- `test-frontend` job now outputs coverage value
- New `update-coverage` job processes the outputs
- `build` job depends on `update-coverage` job

---

## Usage Examples

### Local: Skip tests, just update badges

```bash
bash scripts/update-coverage-badges.sh --backend 45.2 --frontend 78.5
```

Output: Updates badges and docs, no tests run

### Local: Run tests and update badges

```bash
bash scripts/update-coverage-badges.sh
```

Output: Runs tests, extracts coverage, updates badges

### Local: Invalid input (error handling)

```bash
bash scripts/update-coverage-badges.sh --backend invalid --frontend 78.5
```

Output: Error message, exit code 1

---

## CI/CD Flow

```txt
test-backend
    ↓ outputs coverage

test-frontend
    ↓ outputs coverage

update-coverage (receives both outputs)
    ├→ Calls script with --backend X --frontend Y
    ├→ Updates README.md (badges)
    ├→ Updates docs/TEST_COVERAGE.md (history)
    ├→ Commits changes if any
    └→ Skips commit on pull requests

build (waits for coverage update)
    └→ Builds application
```

---

## Key Points

✅ **Tests run once** - No duplication  
✅ **Efficient** - Coverage flows between jobs  
✅ **Safe** - Validates input, only commits on main  
✅ **Backward compatible** - Old usage still works  
✅ **Flexible** - Use with or without arguments

---

## Testing

| Test Case              | Command                             | Expected Result               |
| ---------------------- | ----------------------------------- | ----------------------------- |
| Valid args, skip tests | `--backend 45.2 --frontend 78.5`    | Updates badges, skips tests   |
| No args, run tests     | `(no arguments)`                    | Runs tests, extracts coverage |
| Invalid backend        | `--backend invalid --frontend 78.5` | Error message, exit 1         |
| Only backend           | `--backend 45.2`                    | Runs tests (fallback)         |
| Only frontend          | `--frontend 78.5`                   | Runs tests (fallback)         |

---

## Files Modified

- `scripts/update-coverage-badges.sh` (+92 lines)
- `.github/workflows/build.yaml` (+86 lines)

---

## Documentation

- `docs/COVERAGE_UPDATES_IMPLEMENTATION.md` - Main documentation
- `docs/WORKFLOW_COVERAGE_UPDATES.md` - Detailed workflow info
- `docs/WORKFLOW_IMPLEMENTATION_SUMMARY.md` - Implementation summary
- `docs/IMPLEMENTATION_DIFF.md` - Before/after comparison

---

## Status

✅ **Implementation Complete**
✅ **Tested Locally**
✅ **YAML Validated**
✅ **Ready for Deployment**
