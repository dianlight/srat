<!-- DOCTOC SKIP -->

# SRAT Copilot Instructions

These instructions are the concise, must-follow rules for working in SRAT. Keep changes minimal, readable, and aligned with existing patterns.

## Non‑negotiable rules

- **Read the file header first**: Always read the top comment/header of any file you modify; file‑specific rules override everything else.
- **No git writes**: Never run `git add/commit/push` unless the user explicitly asks.
- **Follow instruction files**: Use the specialized guidance in `.github/instructions/` for Go, Python, React, Markdown, frontend tests, and backend command execution.
- **Mandatory for backend execution**: When implementing or migrating backend command execution, always follow `.github/instructions/backend-command-execution.instructions.md`.
- **Ask for clarification**: If a user request is ambiguous or could lead to unintended consequences, ask for clarification before proceeding.
- **Respect existing code**: Follow the established architecture, style, and patterns of the codebase. Avoid introducing new abstractions or styles unless necessary.
- **Prioritize maintainability**: Write clear, readable code that other developers can easily understand and maintain. Avoid clever or complex solutions when a straightforward approach will do.
- **Add tests**: When fixing bugs or adding features, include tests that cover the new behavior and edge cases. Follow the testing guidelines in the instruction files.
- **Test your changes**: Always run the relevant tests after making changes to ensure you haven't introduced regressions. Follow the testing guidelines in the instruction files.
- **Document your changes**: If your change affects the behavior of the system, update the relevant documentation and add comments to your code where necessary to explain non-obvious logic or decisions.
- **Verify before finalizing**: Before finalizing any code changes, review your work to ensure it adheres to the above rules and the specific guidelines in the instruction files. If you're unsure about any aspect of your changes, ask for a review or feedback from a human developer. Always aim for high-quality, maintainable code that aligns with the project's standards and goals.
- **Commit precheck**: Ensure that all pre-commit checks (linters, formatters, security scanners) pass before finalizing your changes. If any checks fail, address the issues and re-run the checks until they pass successfully.

## Repo at a glance

- **Languages**: Go 1.26 back-end, TypeScript React frontend (Bun), Python 3.12+ Home Assistant integration.
- **Architecture**: API handlers → services → generated GORM helpers → SQLite (embedded). Frontend uses MUI + RTK Query. Custom component is WebSocket‑only.

## back-end (Go) essentials

- Use **context‑aware logging** (`slog.*Context`, `tlog.*Context`) when a real `context.Context` is already in scope. Never manufacture a context for logging.
- Go 1.26 rules: use `new(expr)` for pointer values, use `any` (not `interface{}`), use `WaitGroup.Go`, prefer `errors.AsType[T]` (standard library).
- Do **not** edit vendored code unless using the patch workflow (`backend/patches/` + `mise run //backend:patch`).

## Frontend essentials

- Use Bun toolchain (`frontend/`). Build outputs go to `backend/src/web/static`.
- **Do not** edit `frontend/src/store/sratApi.ts` or `backend/docs/openapi.*` directly—update Go and run `cd frontend && bun run gen`.
- MUI Grid: use the `size` prop (Grid2 default).

## Custom component essentials (Home Assistant)

- Runtime data lives in `entry.runtime_data` (not `hass.data`).
- WebSocket‑only coordinator: `update_interval=None`.
- Sensors return `None` when unavailable.

## Build, generate, test (short list)

- **Back-end:** `mise run //backend:dev|build|test|format|gen`
- **Frontend:** `mise run //frontend:build|dev|lint|test|gen`
- **Custom component:** `mise run //custom_components:check|test|lint|format|typecheck`

## Testing rules (fast summary)

- **Bug fixes require a failing test first**, then the fix, then re‑run tests.
- Frontend tests: use `mise run //frontend:test`, React Testing Library, and **`user-event` only** (no `fireEvent`).
- Frontend test stability: run `mise run //frontend:test --rerun-each 10` for modified tests.

## Docs & quality gates

- Docs: `mise run docs-validate` (and `mise run docs-fix` when needed).
- Security: `mise run security`.
- Changes Checker: `hk check` (and `hk fix` when needed).

## Patching external Go libs

- Patches live in `backend/patches/`. Apply with `mise run //backend:patch`.
- Update vendor via `cd backend/src && go mod vendor` then re‑apply patches with `mise run //backend:patch`.

## Git Branch Naming Convention

When asked to generate a Git command or branch name from a Markdown task:

1.  Use prefixes: `feature/` for new items, `fix/` for bugs, `docs/` for documentation, and `refactor/` for code improvements.
    
2.  Convert the task title to "kebab-case" (lowercase, replace spaces/underscores with hyphens).
    
3.  Strip emojis, special characters, and common stop-words (a, the, of, for, with).
    
4.  Example: "Task: \[ \] Implement user login validation" -> `feature/implement-user-login-validation`.
    

## Contextual Awareness

*   **Markdown Authority**: Always treat "Implementation Notes" in `.md` files as the primary source of truth for business logic.
    
*   **Cross-Repo Logic**: If "Target Repo" is specified in the Markdown header, assume all code generation or terminal commands apply to that specific directory.
    
*   **Task Scanning**: When a user mentions a task by name, look for the corresponding checkbox in open Markdown files to understand the requirements.

## Instruction Files
*   **Purpose**: Instruction files in `.github/instructions/` provide specific guidelines for different file types and scenarios. Always check for an applicable instruction file before making changes.
*   **Format**: These files use YAML front matter to specify which files they apply to and contain concise instructions for code style, testing, or other practices.
*   **Learning**: Familiarize yourself with the existing instruction files to ensure your contributions align with the project's standards and practices.
*   **Updating Instructions**: If you notice a gap in the existing instructions or have suggestions for improvement, you can propose changes to the instruction files themselves, but always ensure that any modifications are clear, concise, and maintainable.
*  **Examples**: Refer to the existing instruction files for examples of how to format and structure your own instructions if you need to create new ones.
* **Adherence**: Always adhere to the guidelines specified in the instruction files when working on relevant code sections to maintain consistency and quality across the project.
* **Feedback**: If you have questions about the instructions or need clarification, don't hesitate to ask for feedback from human developers to ensure you're on the right track.
* **Continuous Improvement**: The instruction files are living documents. As the project evolves, these instructions may need to be updated to reflect new best practices or changes in the codebase. Always be open to improving the instructions as needed. But always ask for feedback before making changes to the instruction files to ensure that any updates are beneficial and align with the project's goals.
* **Testing Instructions**: Pay special attention to instruction files related to testing, as they often contain critical guidelines for ensuring the reliability and maintainability of tests. Always follow these instructions closely when writing or modifying tests.