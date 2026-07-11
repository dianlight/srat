---
name: prepare-refactor
description: 'Guided pre/post test safety harness for code refactors. Before any refactor, ask the user whether to run a "prepare check": identify all impacted functions (even those not directly modified), ensure each has a test, create missing tests, record pre-refactor test state in a tracking doc under docs/refactors/, then run the same tests after the refactor and verify nothing broke. Triggers on: "prepare refactor", "refactor check", "refactor safety check", "pre refactor", "prepare and check refactor", or whenever a task type is REFACTOR.'
argument-hint: 'Refactor scope description or path to task file (e.g. "service/filesystem", "task 018")'
---

# Prepare & Check Refactor

Structured safety harness that wraps any code refactor with a pre-analysis, missing-test creation, baseline recording, and post-refactor verification gate.

## When to Use

- User asks to perform a refactor of any scope
- A `docs/tasks/*.md` task has type `[REFACTOR]`
- Any work that reorganises, renames, moves, or structurally changes existing code without adding new user-visible features

**Every time a task or work is identified as a refactor, ask the user:**

> "This looks like a refactor. Would you like to run a prepare check before starting?
> A prepare check identifies all impacted functions (including indirect ones), ensures each has a test, records their current pass/fail state, and validates them again after the refactor."

If the user declines, proceed directly to the refactor and skip to Phase 5 (post-refactor run) if tests exist. Record the decision in the tracking doc.

---

## Outcome

- A `docs/refactors/<slug>.md` tracking document recording the full refactor lifecycle
- All functions impacted by the refactor identified (direct + indirect)
- Every impacted function covered by at least one test (new tests created where missing)
- Pre-refactor baseline test results recorded
- Post-refactor test results verified and compared against baseline
- All discrepancies resolved or explicitly accepted by the user

---

## Procedure

### Phase 0 — Ask the User

Before doing any analysis, always ask:

```
This task/work is a code refactor. Would you like to run a prepare check?

A prepare check will:
  1. Identify all functions impacted by the refactor (direct + indirect callers/dependants)
  2. Ensure each has at least one test covering the impacted behaviour
  3. Create any missing tests
  4. Record the current pass/fail baseline for every identified test
  5. After the refactor, re-run those tests and verify nothing broke

Proceed with prepare check?
```

Choices: `Yes (Recommended)` / `No — skip prepare, run tests after only` / `No — skip all checks`

Record the user's choice in the tracking document.

---

### Phase 1 — Create the Tracking Document

1. Determine a short slug for the refactor (from the task title, file path, or user description). Example: `filesystem-adapter-registry`, `share-service-persistence`.
2. Create `docs/refactors/<slug>.md` using the template below.
3. Fill in the header fields immediately; leave test tables empty until Phase 2.

#### Tracking Document Template

```markdown
# Refactor: <Title>

<!-- DOCTOC SKIP -->

**Date:** YYYY-MM-DD  
**Status:** 🔍 Analysis  
**Prepare Check:** Yes | No (skipped by user)  
**Linked Task:** docs/tasks/NNN_<title>.md (if applicable)  
**Scope:** <Brief description of what is being changed>

---

## Impacted Functions

| # | Function / Symbol | File | Caller / Reason Impacted | Has Test? | Test File |
|---|-------------------|------|--------------------------|-----------|-----------|
|   |                   |      |                          |           |           |

---

## Pre-Refactor Test Baseline

| Test Name | File | Status Before | Notes |
|-----------|------|---------------|-------|
|           |      |               |       |

---

## Post-Refactor Test Results

| Test Name | File | Status Before | Status After | Result | Notes |
|-----------|------|---------------|--------------|--------|-------|
|           |      |               |              |        |       |

---

## Decisions & Notes

<!-- Record user decisions (skip tests, ignore failures, etc.) and important findings here -->

---

## Checklist

- [ ] Tracking document created
- [ ] Impacted functions identified (direct)
- [ ] Impacted functions identified (indirect callers/dependants)
- [ ] All impacted functions have at least one test
- [ ] Missing tests created
- [ ] Pre-refactor baseline run and recorded
- [ ] Refactor implemented
- [ ] Post-refactor tests run
- [ ] All tests pass (or failures accepted by user)
- [ ] Tracking document finalised
```

---

### Phase 2 — Identify All Impacted Functions

Perform a **two-level impact analysis**:

#### 2a. Direct Impact (functions being modified)

List every function, method, type, or symbol that will be **directly changed** by the refactor:

- Renamed symbols
- Moved packages/files
- Signature changes
- Deleted functions
- Restructured types

Use `grep`, `gopls`, LSP symbol references, or file inspection to enumerate these.

#### 2b. Indirect Impact (callers and dependants)

For every directly impacted symbol, find all **callers and dependants** that are _not_ themselves being modified:

- Functions that call a renamed/moved function
- Structs that embed a changed type
- Interfaces that the changed type satisfies
- Test helpers that wrap the changed code

Use `go_symbol_references`, `grep`, or `gopls` to trace these references.

Fill the **Impacted Functions** table in the tracking document for every symbol found in both 2a and 2b.

---

### Phase 3 — Ensure Test Coverage

For every function in the impacted list:

1. Check whether at least one existing test exercises the function's behaviour that will be affected.
2. Mark `Has Test?` as `Yes` or `No` in the tracking table.
3. For every `No`:
   - **Create a focused test** that covers the specific behaviour at risk from the refactor.
   - Follow the project's testing conventions (see `.opencode/instructions/backend_test.instructions.md` or `fontend_test.instructions.md`).
   - Name the test after the function and scenario: `TestFunctionName_ImpactedBehaviour`.
   - Update the tracking table with the new test file.
4. Run the newly created tests immediately to confirm they pass before the refactor baseline is taken.

> **Rule:** Do not start the refactor until every impacted function has at least one test.

---

### Phase 4 — Record the Pre-Refactor Baseline

Run all tests identified in Phase 3 and record their result in the **Pre-Refactor Test Baseline** table.

For the backend:
```bash
cd backend/src && go test ./... 2>&1 | grep -E "^(ok|FAIL|---)"
# or targeted: go test ./service/... ./api/...
```

For the frontend:
```bash
cd frontend && bun run test 2>&1 | grep -E "(PASS|FAIL|✓|✗)"
```

For the custom component:
```bash
cd custom_components && mise run test
```

For every test:
- `PASS` → record `✅ Pass`
- `FAIL` → record `❌ Fail` and ask the user:

  > "Test `<TestName>` in `<file>` fails **before** the refactor. How should we handle it?
  >
  > 1. Fix now — correct the failing test/code before starting the refactor
  > 2. Fix after — ignore before, fix during/after the refactor
  > 3. Ignore — exclude this test from pre- and post-refactor validation"

  Record the user's decision in **Decisions & Notes** and mark the test accordingly.

Update the tracking document **Status** to `🔧 Ready` once the baseline is complete.

---

### Phase 5 — Perform the Refactor

Implement the refactor following normal project coding conventions:

- Apply changes to directly impacted symbols first
- Update all indirect callers/dependants found in Phase 2b
- Follow instructions in `.opencode/rules.md` and relevant instruction files
- Do **not** add new features or fix unrelated bugs in the same change set
- Update the tracking document checklist item `Refactor implemented` when done

---

### Phase 6 — Run Post-Refactor Tests

Re-run **all** tests recorded in the baseline (excluding those the user chose to ignore in Phase 4).

For each test, compare against the baseline:

| Scenario | Action |
|----------|--------|
| Was `Pass`, now `Pass` | ✅ OK — record in post column |
| Was `Fail` (fix-after), now `Pass` | ✅ Fixed — record and note |
| Was `Pass`, now `Fail` | 🔴 Regression — investigate |
| Was `Fail` (fix-after), still `Fail` | 🔴 Still broken — investigate |

#### When a Post-Refactor Test Fails

1. **Inspect the failure** — read the error message and stack trace carefully.
2. **Determine root cause**:
   - **Code bug**: the refactor introduced a logic or compilation error → fix the code.
   - **Test problem**: the test was brittle and broke due to valid structural change (e.g., renamed export, moved file) → inform the user:

     > "Test `<TestName>` fails after the refactor. The failure appears to be a **test problem** (not a code bug):
     > `<reason>`
     >
     > Should I:
     > 1. Fix the test to match the new structure (recommended)
     > 2. Fix the code so the old test passes (revert the structural change)
     > 3. Accept this failure and note it as a known test debt"

3. Record the decision and resolution in **Decisions & Notes**.
4. After any fix, re-run the failing test and confirm it passes.

---

### Phase 7 — Finalise the Tracking Document

1. Fill in the **Post-Refactor Test Results** table completely.
2. Update the document **Status** to one of:
   - `✅ Complete` — all tests pass (or accepted failures are documented)
   - `⚠️ Complete with Known Issues` — some failures accepted by user
3. Check off all completed checklist items.
4. Summarise the outcome under **Decisions & Notes**.

---

## Completion Checks

- [ ] User was asked about prepare check and decision recorded
- [ ] Tracking document created at `docs/refactors/<slug>.md`
- [ ] All directly impacted functions identified and listed
- [ ] All indirectly impacted callers/dependants identified and listed
- [ ] Every impacted function has at least one test
- [ ] Pre-refactor baseline recorded with pass/fail for every test
- [ ] Any pre-refactor failures resolved per user decision
- [ ] Refactor implemented
- [ ] All baseline tests re-run post-refactor
- [ ] All regressions investigated and resolved or accepted
- [ ] Tracking document finalised with final status

---

## Example Prompts

- `prepare refactor for service/filesystem`
- `prepare and check refactor for task 018`
- `refactor safety check before renaming ShareService`
- `run prepare check — I'm about to refactor the broadcaster`
- `start refactor of the repair service`
