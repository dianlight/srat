<!-- DOCTOC SKIP -->

# SRAT Copilot Instructions

These instructions are the concise, must-follow rules for working in SRAT. Keep changes minimal, readable, and aligned with existing patterns.

## Non‑negotiable rules

- **Read the file header first**: Always read the top comment/header of any file you modify; file‑specific rules override everything else.
- **No git writes**: Never run `git add/commit/push` unless the user explicitly asks.
- **Follow instruction files**: Use the specialized guidance in `.github/instructions/` for Go, Python, React, Markdown, and frontend tests.
- **Ask for clarification**: If a user request is ambiguous or could lead to unintended consequences, ask for clarification before proceeding.
- **Respect existing code**: Follow the established architecture, style, and patterns of the codebase. Avoid introducing new abstractions or styles unless necessary.
- **Prioritize maintainability**: Write clear, readable code that other developers can easily understand and maintain. Avoid clever or complex solutions when a straightforward approach will do.
- **Add tests**: When fixing bugs or adding features, include tests that cover the new behavior and edge cases. Follow the testing guidelines in the instruction files.
- **Test your changes**: Always run the relevant tests after making changes to ensure you haven't introduced regressions. Follow the testing guidelines in the instruction files.
- **Document your changes**: If your change affects the behavior of the system, update the relevant documentation and add comments to your code where necessary to explain non-obvious logic or decisions.

## Repo at a glance

- **Languages**: Go 1.26 backend, TypeScript React frontend (Bun), Python 3.12+ Home Assistant integration.
- **Architecture**: API handlers → services → generated GORM helpers → SQLite (embedded). Frontend uses MUI + RTK Query. Custom component is WebSocket‑only.

## Backend (Go) essentials

- Use **context‑aware logging** (`slog.*Context`, `tlog.*Context`) when a real `context.Context` is already in scope. Never manufacture a context for logging.
- Go 1.26 rules: use `new(expr)` for pointer values, use `any` (not `interface{}`), use `WaitGroup.Go`, prefer `errors.AsType[T]` (standard library).
- Do **not** edit vendored code unless using the patch workflow (`backend/patches/` + `make patch`).

## Frontend essentials

- Use Bun toolchain (`frontend/`). Build outputs go to `backend/src/web/static`.
- **Do not** edit `frontend/src/store/sratApi.ts` or `backend/docs/openapi.*` directly—update Go and run `cd frontend && bun run gen`.
- MUI Grid: use the `size` prop (Grid2 default).

## Custom component essentials (Home Assistant)

- Runtime data lives in `entry.runtime_data` (not `hass.data`).
- WebSocket‑only coordinator: `update_interval=None`.
- Sensors return `None` when unavailable.

## Build, generate, test (short list)

- Backend: `cd backend && make dev|build|test|format|gen`
- Frontend: `cd frontend && bun install && bun run build|dev|lint|test|gen`
- Custom component: `cd custom_components && make check|test|lint|format|typecheck`

## Testing rules (fast summary)

- **Bug fixes require a failing test first**, then the fix, then re‑run tests.
- Frontend tests: use `bun:test`, React Testing Library, and **`user-event` only** (no `fireEvent`).
- Frontend test stability: run `bun test --rerun-each 10` for modified tests.

## Docs & quality gates

- Docs: `make docs-validate` (and `make docs-fix` when needed).
- Security: `make security`.

## Patching external Go libs

- Patches live in `backend/patches/`. Apply with `cd backend && make patch`.
- Update vendor via `cd backend/src && go mod vendor` then re‑apply patches.
