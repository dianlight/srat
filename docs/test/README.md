<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->

**Table of Contents** _generated with [DocToc](https://github.com/thlorenz/doctoc)_

- [Test Cases](#test-cases)
  - [Naming Convention](#naming-convention)
  - [Directory Structure](#directory-structure)
  - [Test Case Format](#test-case-format)
    - [1. Preparation](#1-preparation)
    - [2. Execution](#2-execution)
    - [3. Validation (Optional)](#3-validation-optional)
  - [Execution Environment](#execution-environment)
  - [Running Tests](#running-tests)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

# Test Cases

This directory contains test case specifications for the SRAT project. Test cases are designed to be executed on remote test environments with Home Assistant and the SRAT addon.

## Naming Convention

Test case files follow the pattern `<func-id>.<test-id>_<description>.md`, where:

- `func-id`: 3-digit number identifying the UI macro functionality (e.g., `001` for share operations, `002` for disk operations)
- `test-id`: 3-digit number identifying the specific test within the functionality
- `description`: lowercase with hyphens replacing spaces

Example: `001.001_create-a-share.md` - first share creation test

## Directory Structure

- `docs/test/backend/` - Backend test cases (API, service, GORM helpers)
- `docs/test/frontend/` - Frontend test cases (UI, components, RTK queries)

## Test Case Format

Every test case must contain exactly 3 phases:

### 1. Preparation

- Steps necessary to set and check the environment necessary for the test
- If preparation fails and cannot be completed, the test is **skipped**
- Include specific commands, state checks, and prerequisites
- Must verify the test environment is in a known good state before execution

### 2. Execution

- Steps to test and assert the correct execution
- Must include specific assertions and expected outcomes
- Scripts should minimize token usage and alternative interpretation
- Include expected vs. actual outcome comparisons

### 3. Validation (Optional)

- Final validation to test and validate the execution
- Used to confirm the test objective was met
- May include post-condition checks, state verification, or cleanup

## Execution Environment

- Tests are designed for remote execution with Home Assistant OS
- SRAT addon must be installed and running
- Access to Samba shares, disks, and Home Assistant components as specified
- Tests may require specific addon configurations or addon states

## Running Tests

Test cases can be executed using the Home Assistant test framework or manually via SSH/terminal access to the addon container.
