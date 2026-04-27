<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->

- [Contributing to SRAT](#contributing-to-srat)
  - [1. Code of Conduct](#1-code-of-conduct)
  - [2. Development Environment](#2-development-environment)
  - [3. Branching & Commits](#3-branching--commits)
  - [4. Testing Requirements](#4-testing-requirements)
  - [5. Logging Rule (Context Aware) ✅](#5-logging-rule-context-aware-)
    - [Allowed When No Context Exists](#allowed-when-no-context-exists)
    - [NEVER Do](#never-do)
    - [Goroutines](#goroutines)
    - [Vendor & Generated Code](#vendor--generated-code)
    - [Suppression / Exceptions](#suppression--exceptions)
    - [Enforcement](#enforcement)
  - [6. Error Handling](#6-error-handling)
  - [7. Go 1.26 Modern Patterns](#7-go-126-modern-patterns)
  - [8. Database & Migrations](#8-database--migrations)
  - [9. Patches to Dependencies](#9-patches-to-dependencies)
  - [10. Frontend Patterns](#10-frontend-patterns)
    - [TypeScript 6.0/7.0 Compatibility](#typescript-6070-compatibility)
  - [11. Custom Component (Home Assistant)](#11-custom-component-home-assistant)
    - [Tooling](#tooling)
    - [Makefile Targets](#makefile-targets)
    - [Architecture](#architecture)
  - [12. Documentation](#12-documentation)
  - [13. Security](#13-security)
  - [14. Performance](#14-performance)
  - [15. Optimization & Maintenance](#15-optimization--maintenance)
    - [Running an Optimization Round](#running-an-optimization-round)
    - [Reference Documents (Always Maintained)](#reference-documents-always-maintained)
  - [16. Pull Request Checklist](#16-pull-request-checklist)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

# Contributing to SRAT

Welcome. This guide documents the project conventions you MUST follow when contributing code.

## 1. Code of Conduct

Be respectful. Provide clear rationale in PR descriptions. Security or stability concerns take precedence over style.

## 2. Development Environment

- back-end: Go 1.26
- Frontend: Bun + React + TypeScript
- Custom Component: Python 3.12+ with ruff (lint/format) and mypy (type-check)
- Run `mise run prepare` once to install pre-commit hooks.

## 3. Branching & Commits

- Create feature branches from `main` (or the current integration branch if instructed).
- Keep commits focused; squash noisy WIP commits before merge.
- NEVER push generated artifacts (frontend build output, vendor changes outside patch workflow) unless explicitly required.

## 4. Testing Requirements

- back-end: Add/extend `testify/suite` tests; ensure deterministic output.
- Frontend: Place tests in `__tests__` directories; follow patterns in `/.github/copilot-instructions.md`.
- Custom Component: Use `pytest-homeassistant-custom-component` for tests under `custom_components/tests/`. Run with `cd custom_components && mise run test`.
- Minimum coverage thresholds enforced by CI; raise coverage when adding logic.

## 5. Logging Rule (Context Aware) ✅

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

Wrap errors with context using the local `errors` helpers. Prefer sentinel errors in `dto` for domain cases. When using the standard library `errors` package, prefer `errors.AsType[T]()` (Go 1.26) over `errors.As()` for type-safe error checking.

## 7. Go 1.26 Modern Patterns

All new or modified Go code MUST follow these conventions:

- **Pointer creation**: Use `new(expr)` (e.g., `new(false)`, `new("ext4")`). Never use `pointer.Bool()` or similar helper libraries.
- **Type alias**: Use `any` instead of `interface{}`. Generated files are exempt.
- **WaitGroup**: Use `wg.Go(func() { ... })` instead of `wg.Add(1)` + `go func() { defer wg.Done(); ... }()`.
- **Modernizers**: Run `go fix ./...` periodically to apply automated Go modernizations.

See `/.github/copilot-instructions.md` for detailed examples.

## 8. Database & Migrations

- Use Goose migration files under `backend/src/dbom/migrations/`.
- New migrations must be idempotent and defensive.

## 9. Patches to Dependencies

Follow patch workflow (`mise run //backend:patch`, `backend/patches/*`). Never add direct `replace` directives in `go.mod`.

## 10. Frontend Patterns

Strictly follow testing & import patterns from `/.github/copilot-instructions.md`. All user interactions must use `@testing-library/user-event`.

### TypeScript 6.0/7.0 Compatibility

The frontend uses **TypeScript 6.0 Beta / 7.0 Preview (tsgo)** with ES2022 target:

- **Type checking**: Use `bun tsgo --noEmit` (not `tsc`)
- **Compiler**: `@typescript/native-preview` (TypeScript 7.0 Go-based preview)
- **Configuration**: See `frontend/tsconfig.json`
- **Migration Guide**: See `frontend/TYPESCRIPT_MIGRATION.md`

**Key rules when working with TypeScript code**:

1. **Do NOT add deprecated compiler flags** to `tsconfig.json`:
   - ❌ No `experimentalDecorators`
   - ❌ No `useDefineForClassFields: false`
   - ❌ No `target: es5` or ES2015 (minimum ES2022)

2. **Use `override` keyword** for class methods that override parent methods:

   ```typescript
   class MyComponent extends Component {
     public override render() { return <div />; }
   }
   ```

3. **Follow strict type checking**:
   - Avoid `any` type; use `unknown` with type guards
   - Maintain `noImplicitOverride: true` setting
   - See migration guide before enabling `noUncheckedIndexedAccess`

4. **Resources**:
   - Development guidelines: `.github/instructions/typescript-6-es2022.instructions.md`
   - Migration status: `frontend/TYPESCRIPT_MIGRATION.md`
   - Summary: `TYPESCRIPT_6_IMPLEMENTATION_SUMMARY.md`

## 11. Custom Component (Home Assistant)

The HACS-compatible custom component lives in `custom_components/srat/`. It is written in Python 3.12+ and uses the Home Assistant integration framework.

### Tooling

- **Lint & format**: [ruff](https://docs.astral.sh/ruff/) — configured in `custom_components/pyproject.toml`
- **Type checking**: [mypy](https://mypy-lang.org/) — configured in `custom_components/pyproject.toml`
- **Testing**: [pytest-homeassistant-custom-component](https://github.com/MatthewFlamm/pytest-homeassistant-custom-component) — tests in `custom_components/tests/`

### Makefile Targets

Run all targets from the `custom_components/` directory:

```shell
mise run install      # Install dev dependencies (prefers apk on Alpine, falls back to pip)
mise run check        # Run all checks: format + lint + type-check + tests
mise run lint         # ruff lint
mise run format       # ruff format check
mise run typecheck    # mypy type checking
mise run test         # pytest tests
mise run fix          # Auto-fix lint and format issues
mise run clean        # Remove caches and build artifacts
```

### Architecture

The component communicates with the SRAT back-end exclusively via WebSocket (`/ws` endpoint). No REST API polling is used. See [Home Assistant Integration](docs/HOME_ASSISTANT_INTEGRATION.md) for details.

## 12. Documentation

Update `CHANGELOG.md` for user-visible changes. Provide rationale for breaking changes.

## 13. Security

Run `mise run security` locally before opening a PR touching sensitive areas (auth, execution, filesystem). Avoid logging credentials or secrets; masking is handled by `tlog` but still use discretion.

## 14. Performance

Profile hotspots using provided `PPROF.md` guidance for significant performance-related changes.

## 15. Optimization & Maintenance

The SRAT instruction and skills framework is systematically optimized to maintain clarity, reduce duplication, and integrate high-value knowledge from stored memory facts.

### Running an Optimization Round

If tasked with optimizing instructions/skills:

1. **Read the baseline**: See `docs/memory-index.md` (all 19 memory facts), `docs/quick-reference.md` (code patterns)
2. **Use the prompt**: See `.github/prompts/optimize-instructions.prompt.md` for structured guidance (Copilot prompt file format)
3. **Follow the workflow**: Assessment → Planning → Execution → Verification → Completion
4. **Update the index**: Add new facts to `docs/memory-index.md` with integration status
5. **Create a checkpoint**: Document what was done in `session/checkpoints/`

### Reference Documents (Always Maintained)

- `docs/memory-index.md` — Maps all stored memory facts + integration status
- `docs/quick-reference.md` — Copy-paste code patterns with ✅/❌ examples
- `docs/shared-principles.md` — Core principles (Go, TypeScript, Python, Markdown)
- `docs/test-setup-patterns.md` — Unified test patterns across languages
- `.github/prompts/optimize-instructions.prompt.md` — Copilot prompt file for future optimization rounds

**Last optimization round**: 2026-04-25 (Rounds 1–3 complete, ~22% token efficiency gain)

## 16. Pull Request Checklist

- [ ] Tests added / updated
- [ ] Lint & format pass (`prek run --all-files`)
- [ ] Context logging rule satisfied (back-end)
- [ ] Custom component checks pass (`cd custom_components && mise run check`, if applicable)
- [ ] No stray `replace` directives
- [ ] Documentation updated
- [ ] No secrets or raw tokens in logs

Thank you for contributing. 🚀
