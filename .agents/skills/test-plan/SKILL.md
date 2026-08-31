---
name: test-plan
description: Test plan custom skill to help creation of test cases for SRAT project
argument-hint: "Describe what test cases are needed (e.g., 'create share tests, include backend and frontend')"
---

# Test Plan Custom Skill

## Description

This skill helps create test cases for the SRAT project following the established conventions and formats. It provides guidance on test case structure, naming conventions, and the 3-phase format (Preparation, Execution, Validation).

## When to Use

- When creating new test cases for SRAT backend or frontend functionality
- When following the test case naming convention: `<func-id>.<test-id>_<description>.md`
- When documenting test cases with the required 3-phase format
- When ensuring test cases are suitable for remote execution with Home Assistant and the SRAT addon

## How to Use

Provide an argument describing the test cases needed, for example:

```
Test: "Create share creation and deletion tests for backend and frontend"
Custom component: "include custom component: no"
```

Or more specifically:

```
Test: "001.001_create-a-share backend API test, 001.002_delete-a-share backend test, 001.001_create-a-share-ui frontend UI test"
Custom component: "include custom component: no"
```

## Test Case Creation Guidelines

### Naming Convention

Test case files follow the pattern `<func-id>.<test-id>_<description>.md`, where:

- `func-id`: 3-digit number identifying the UI macro functionality (e.g., `001` for share operations, `002` for disk operations)
- `test-id`: 3-digit number identifying the specific test within the functionality
- `description`: lowercase with hyphens replacing spaces

Example: `001.001_create-a-share.md` - first share creation test

### Directory Structure

- `docs/test/backend/` - Backend test cases (API, service, GORM helpers)
- `docs/test/frontend/` - Frontend test cases (UI, components, RTK queries)

### 3-Phase Format

Every test case must contain exactly 3 phases:

#### 1. Preparation

- Steps necessary to set and check the environment necessary for the test
- If preparation fails and cannot be completed, the test is **skipped**
- Include specific commands, state checks, and prerequisites
- Must verify the test environment is in a known good state before execution

#### 2. Execution

- Steps to test and assert the correct execution
- Must include specific assertions and expected outcomes
- Scripts should minimize token usage and alternative interpretation
- Include expected vs. actual outcome comparisons

#### 3. Validation (Optional)

- Final validation to test and validate the execution
- Used to confirm the test objective was met
- May include post-condition checks, state verification, or cleanup

### Execution Environment

- Tests are designed for remote execution with Home Assistant OS
- SRAT addon must be installed and running
- Access to Samba shares, disks, and Home Assistant components as specified
- Tests may require specific addon configurations or addon states

### Example Test Case Structure

```markdown
# Test Case: 001.001_create-a-share

## Test Objective

Verify that a new SMB share can be created successfully.

## Preparation

1. Ensure the SRAT addon is running: `addon ls`
2. Verify a data disk is mounted: `df -h`
3. Set up test environment variables:

```bash
export SRAT_TEST_MODE=true
export TEST_SHARE_NAME="test-share-$(date +%s)"
```

**If any preparation step fails**, the test must be **skipped** and documented in test logs.

## Execution

1. Create a new share via API:
   ```bash
   curl -X POST http://localhost:8090/api/shares \
     -H "Content-Type: application/json" \
     -d "{\"name\": \"${TEST_SHARE_NAME}\", \"path\": \"/mnt/testvolume\"}"
   ```
2. Verify API response returns HTTP 201 Created

**Assertions:**
- API returns status code 201
- Share appears in share listing

## Validation

1. Verify share is listed in the SRAT frontend UI
2. Confirm share properties match what was created
3. Clean up: remove test share via API

**Success criteria:** All assertions pass and share is confirmed created and then cleaned up.
```

## Related Skills

- `test-remote-environment` - For executing test cases on remote Home Assistant environment
- `create-task` - For creating task planning documents
- `start-task-work` - For starting implementation from existing tasks

## Usage Examples

### Example 1: Create backend share tests

```
Test: "Create 001.001 and 001.002 backend share tests for create and delete operations"
Custom component: "include custom component: no"
```

### Example 2: Create frontend share UI tests

```
Test: "Create 001.001 and 001.002 frontend share UI tests for create and edit operations"
Custom component: "include custom component: no"
```

### Example 3: Create comprehensive test suite

```
Test: "Create full test suite: backend create/share delete, frontend create/share edit, disk health check"
Custom component: "include custom component: no"
```

## Return Values

This skill provides:

- Test case files created in `docs/test/` directory following conventions
- Test plan documentation in `docs/test/README.md`
- 3-phase test case structure (Preparation, Execution, Validation)
- Naming convention guidance and directory structure
- Ready-to-use test case templates for common SRAT functionalities

## Error Handling

This skill gracefully handles:

- Missing or invalid test case descriptions
- Invalid func-id or test-id numbers
- Missing required directories
- Conflicting test case names

All errors provide actionable guidance for creating test cases correctly.
