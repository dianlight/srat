# Spec Delta

## Purpose

Database Recovery Safety, HTTP Request Size Limits, and Goroutine Leak behaviour for the SRAT project.

## ADDED Requirements

### Requirement: Database Recovery Safety, HTTP Request Size Limits, and Goroutine Leak

The system SHALL fix three reliability issues that share the theme of defensive guards around infrastructure components: 1. **`replaceDatabase` infinite recursion** — `NewDB` and `replaceDatabase` can recurse indefinitely when the database path is persistently unwritable, causing a stack overflow.

#### Scenario: Change applied

- **WHEN** the change is applied as described in the proposal
- **THEN** Add a `recovered bool` parameter to `replaceDatabase` (or an internal `newDBWithDepth(depth int)` variant); if `depth > 1`, call `tlog.Fatal` instead of recursing

#### Scenario: Regression check

- **WHEN** the change has been implemented
- **THEN** the relevant test suites pass
