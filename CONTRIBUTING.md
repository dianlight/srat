<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->

- [Contributing to SRAT](#contributing-to-srat)
  - [1. Code of Conduct](#1-code-of-conduct)
  - [2. Development Environment](#2-development-environment)
  - [3. Branching & Commits](#3-branching--commits)
  - [4. Testing Requirements](#4-testing-requirements)
  - [5. Logging RULE (Context-Aware) ✅](#5-logging-rule-context-aware-)
    - [Allowed When No Context Exists](#allowed-when-no-context-exists)
    - [NEVER Do](#never-do)
    - [Goroutines](#goroutines)
    - [Vendor & Generated Code](#vendor--generated-code)
    - [Suppression / Exceptions](#suppression--exceptions)
    - [Enforcement](#enforcement)
  - [6. Error Handling](#6-error-handling)
  - [7. Database & Migrations](#7-database--migrations)
  - [8. Patches to Dependencies](#8-patches-to-dependencies)
  - [9. Frontend Patterns](#9-frontend-patterns)
  - [10. Documentation](#10-documentation)
  - [11. Security](#11-security)
  - [12. Performance](#12-performance)
  - [13. Pull Request Checklist](#13-pull-request-checklist)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

# Contributing to SRAT

Welcome. This guide documents the project conventions you MUST follow when contributing code.

## 1. Code of Conduct

Be respectful. Provide clear rationale in PR descriptions. Security or stability concerns take precedence over style.

## 2. Development Environment

- Backend: Go 1.25
- Frontend: Bun + React + TypeScript
- Run `make prepare` once to install pre-commit hooks.

## 3. Branching & Commits

- Create feature branches from `main` (or the current integration branch if instructed).
- Keep commits focused; squash noisy WIP commits before merge.
- NEVER push generated artifacts (frontend build output, vendor changes outside patch workflow) unless explicitly required.

## 4. Testing Requirements

- Backend: Add/extend `testify/suite` tests; ensure deterministic output.
- Frontend: Place tests in `__tests__` directories; follow patterns in `/.github/copilot-instructions.md`.
- Minimum coverage thresholds enforced by CI; raise coverage when adding logic.

## 5. Logging RULE (Context-Aware) ✅

You MUST prefer context-aware logging for `slog` and `tlog` when a `context.Context` value is already in scope.

Use:

```go
slog.InfoContext(ctx, "...", "key", val)
tlog.WarnContext(ctx, "...", "key", val)
```

Instead of:

```go
slog.Info("...", "key", val)
tlog.Warn("...", "key", val)
```

### Allowed When No Context Exists

If no legitimate context variable (for example `ctx`, `self.ctx`, `r.Context()`, constructor-local `Ctx`) is naturally available, keep the non-context variant. DO NOT create artificial contexts (no `context.Background()` just to satisfy the rule).

### NEVER Do

```go
slog.ErrorContext(context.Background(), "Doing X") // ❌ artificial context
```

### Goroutines

Only use a context inside a goroutine if one is already captured for other work (timeouts, cancellation). Do not introduce a context purely for logging.

### Vendor & Generated Code

Do not modify vendored dependencies or generated code to retrofit context logging.

### Suppression / Exceptions

Rare false positives can be suppressed with a trailing comment:

```go
slog.Info("legacy", "x", v) // nolint:contextlog
```

Use sparingly and document the reason in the PR.

### Enforcement

A pre-commit hook (`verify-context-logging`) scans for non-context `slog`/`tlog` usage inside functions whose signature includes a `ctx context.Context` parameter. Violations block the commit. Set `SKIP_CONTEXT_LOGGING_LINT=1` (temporary, discouraged) to bypass locally if absolutely necessary.

## 6. Error Handling

Wrap errors with context using the local `errors` helpers. Prefer sentinel errors in `dto` for domain cases.

## 7. Database & Migrations

- Use Goose migration files under `backend/src/dbom/migrations/`.
- New migrations must be idempotent and defensive.

## 8. Patches to Dependencies

Follow patch workflow (`make patch`, `backend/patches/*`). Never add direct `replace` directives in `go.mod`.

## 9. Frontend Patterns

Strictly follow testing & import patterns from `/.github/copilot-instructions.md`. All user interactions must use `@testing-library/user-event`.

## 10. Documentation

Update `CHANGELOG.md` for user-visible changes. Provide rationale for breaking changes.

## 11. Security

Run `make security` locally before opening a PR touching sensitive areas (auth, execution, filesystem). Avoid logging credentials or secrets; masking is handled by `tlog` but still use discretion.

## 12. Performance

Profile hotspots using provided `PPROF.md` guidance for significant performance-related changes.

## 13. Pull Request Checklist

- [ ] Tests added / updated
- [ ] Lint & format pass (`prek run --all-files`)
- [ ] Context logging rule satisfied
- [ ] No stray `replace` directives
- [ ] Documentation updated
- [ ] No secrets or raw tokens in logs

Thank you for contributing. 🚀
