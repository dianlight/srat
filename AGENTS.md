<!-- DOCTOC SKIP -->

# SRAT Copilot Instructions

These instructions are the concise, must-follow rules for working in SRAT. Keep changes minimal, readable, and aligned with existing patterns.

## Non‑negotiable rules

- **Read the file header first**: Always read the top comment/header of any file you modify; file‑specific rules override everything else.
- **No git writes**: Never run `git add/commit/push` unless the user explicitly asks.
- **Follow instruction files**: Use the specialized guidance in `.opencode/instructions/` for Go, Python, React, Markdown, frontend tests, and backend command execution.
- **Mandatory for backend execution**: When implementing or migrating backend command execution, always follow `.opencode/instructions/backend-command-execution.instructions.md`.
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
- Prefer direct persistence in services using `dbom` + GORM (and generated query helpers when available) over introducing new per-entity repository layers, unless a clear documented exception is required.
- Use **generated converters** (`converter.<Type>ToDtoConverterImpl{}` from `converter/`) for all DTO↔DBOM mapping in services. Never write manual `toDTO`/`toDBOM` helper functions — they diverge silently from the generated impl.
- Do **not** edit vendored code unless using the patch workflow (`backend/patches/` + `mise run //backend:patch`).

## Frontend essentials

- Use Bun toolchain (`frontend/`). Build outputs go to `backend/src/web/static`.
- **Do not** edit `frontend/src/store/sratApi.ts` or `backend/docs/openapi.*` directly—update Go and run `cd frontend && bun run gen`.
- **Never** manually add types to `frontend/src/store/wsApi.ts`. All types must come from `sratApi.ts`. WS-only event payload types that have no REST endpoint need a doc-stub handler in `backend/src/api/system.go` (tagged `"system","internal"`).

  **Doc-Stub Pattern:** A handler that always returns an error but declares the DTO(s) in its response type signature. This anchors the types into the OpenAPI schema, which code-generates them into frontend TypeScript. Example:

  ```go
  // HandleCommandEvents is a documentation-only stub that anchors command event schemas
  // into the OpenAPI spec so they code-generate into TypeScript types.
  // Actual command events are delivered over WebSocket.
  func (s *SystemHandler) HandleCommandEvents(ctx context.Context, input *struct{}) (*struct {
    Body struct {
      Started    *dto.CommandStartedNotification    `json:"started,omitempty"`
      Output     *dto.CommandOutputNotification     `json:"output,omitempty"`
      Terminated *dto.CommandTerminatedNotification `json:"terminated,omitempty"`
    }
  }, error) {
    return nil, huma.Error500InternalServerError("Use WebSocket for events", nil)
  }
  ```

  After adding a doc-stub, run `cd frontend && bun run gen` to code-generate the types into `sratApi.ts`, then import and use them in `wsApi.ts` or other files.

- MUI Grid: use the `size` prop (Grid2 default).
- Frontend test isolation: use `msw` for API mocking; shared recurring handlers go in `frontend/src/mocks/customHandlers.ts`.
- **Lab feature gating pattern**: to hide a settings control behind `experimental_lab_mode`, follow `HomeAssistantPanel.tsx` — add `const experimentalLabMode = Boolean(watch("experimental_lab_mode"));`, a `labLabel(text)` helper that appends `<ScienceOutlinedIcon color="warning" fontSize="small" />`, and wrap the feature in `{experimentalLabMode ? (<Feature label={labLabel("Name")} />) : null}`.

## Custom component essentials (Home Assistant)

- Runtime data lives in `entry.runtime_data` (not `hass.data`).
- WebSocket‑only coordinator: `update_interval=None`.
- Sensors return `None` when unavailable.
- **Lifecycle operations (restart, start, stop)**: Use the Supervisor API via `supervisorCoreClient.RestartCoreWithResponse(ctx)` from the `homeassistant/core` package, **not** the HA Core REST proxy (`coreClient.CallServiceWithResponse(...)`). The REST proxy is for entity states and service calls only; using it for lifecycle management causes silent 504 timeouts (HA drops the connection while restarting itself).

## Build, generate, test (short list)

- **Back-end:** `mise run //backend:dev|build|test|format|gen`
- **Frontend:** `mise run //frontend:build|dev|lint|test|gen`
- **Custom component:** `mise run //custom_components:check|test|lint|format|typecheck`
- **Never run `go build`/`go vet`/`go test` directly** inside `backend/src`. The back-end requires `src/config/metadata_constants.go`, which is gitignored and generated by `mise run metadata --version <ver>` as a build prerequisite. Running Go tooling directly (e.g. `GOOS=linux go build ./...`) will fail with `undefined: MetadataJSON`. Always use `mise run //backend:build|test|dev`.

## Testing rules (fast summary)

- **Bug fixes require a failing test first**, then the fix, then re‑run tests.
- For back-end test failures or new back-end functionality, verify in escalation order: use `mise run //backend:test` to run the full test suite. Remember that direct `go test` commands will fail without the generated metadata constants; always use the mise workflow.
- Frontend tests: use `mise run //frontend:test`, React Testing Library, and **`user-event` only** (no `fireEvent`).
- Frontend test stability: run `mise run //frontend:test --rerun-each 10` for modified tests.

## Refactoring

- **Always invoke the `prepare-refactor` skill** when a task type is `[REFACTOR]` or when the work is described as a refactor. Ask the user whether to run a prepare check before starting.
- Refactor tracking documents live in `docs/refactors/<slug>.md`; do not commit them to task docs.

## Shared References (DRY Consolidation)

These documents consolidate principles, patterns, and facts repeated across language-specific instructions, reducing duplication:

- **`docs/shared-principles.md`**: Core principles shared across Go, TypeScript, Python, and Markdown. Covers "Respect existing code", "Error handling", "Testing lifecycle", "Code quality", and "Security". Referenced by all language-specific instruction files.
- **`docs/test-setup-patterns.md`**: Unified test infrastructure patterns for Go (testify/suite + fx), TypeScript (Vitest + RTL), and Python (pytest). Includes critical ordering rules (e.g., cancel context BEFORE waiting on WaitGroup), anti-patterns, and a verification checklist. Referenced by backend, frontend, and custom component test instructions.
- **`docs/memory-index.md`** (NEW): Maps all 19 stored memory facts to their current locations in instruction files. Shows integration status (8 integrated, 6 already covered, 3 pending, 2 archived), cross-references by file, and recommendations for future rounds. Use to understand what patterns are documented and why some facts weren't integrated.
- **`docs/quick-reference.md`** (NEW): Fast lookup for 8 highest-impact patterns with copy-paste code snippets. Covers MSW body clone, DTO type safety, semver comparison, service architecture, HA Supervisor API, test cleanup, RTK lazy hooks, and IssueCard ignored-state. Use when implementing these patterns.
- **`.opencode/commands/optimize-instructions.prompt.md`**: Reusable prompt file for running future optimization rounds. Includes 5 optimization goals (A–E), standard workflow template (5 phases), key files to update, constraints, and success criteria. Start with this when the user requests instruction optimization.

When writing or updating language-specific instructions, link to these shared references instead of duplicating guidance. After optimization rounds, scan `memory-index.md` to identify high-value facts for future integration. For optimization requests, reference `.opencode/commands/optimize-instructions.prompt.md` to ensure systematic, repeatable workflows.

## Core Service Architecture Patterns

The SRAT back-end uses several linked services for lifecycle and state management:

- **ProblemService**: Centralized issue/problem tracking with lifecycle states. New service features (addon config changes, component restart requirements, repairs) call `ProblemService` directly.
- **RepairService**: Legacy compatibility layer that mirrors repair operations into `ProblemService` with best-effort semantics (failures are non-fatal; primary operation success is determined by legacy API only).
- **HomeAssistantComponentService**: Manages SRAT addon lifecycle and custom component tracking. Calls `ProblemService` for `custom_component_restart_required` repairs when actions (install/upgrade/uninstall) complete.
- **AddonConfigWatcherService**: Monitors Home Assistant addon config changes and calls `ProblemService` for `addon_config_changed` problems when mismatches are detected.

**Key principle**: When migrating features from legacy (Repair API) to modern (Problem Service), new service mirrors legacy operations but failures are best-effort. The legacy operation semantics (success/failure) remain unchanged.

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

## Git Commit Message Convention

When asked to generate a Git commit use instructions from `.opencode/commit-rules.md`:

## Context7

Use Context7 MCP to fetch current documentation whenever the user asks about a library, framework, SDK, API, CLI tool, or cloud service -- even well-known ones like React, Next.js, Prisma, Express, Tailwind, Django, or Spring Boot. This includes API syntax, configuration, version migration, library-specific debugging, setup instructions, and CLI tool usage. Use even when you think you know the answer -- your training data may not reflect recent changes. Prefer this over web search for library docs.

Do not use for: refactoring, writing scripts from scratch, debugging business logic, code review, or general programming concepts.

### Steps

1. Always start with `resolve-library-id` using the library name and the user's question, unless the user provides an exact library ID in `/org/project` format
2. Pick the best match (ID format: `/org/project`) by: exact name match, description relevance, code snippet count, source reputation (High/Medium preferred), and benchmark score (higher is better). If results don't look right, try alternate names or queries (e.g., "next.js" not "nextjs", or rephrase the question). Use version-specific IDs when the user mentions a version
3. `query-docs` with the selected library ID and the user's full question (not single words)
4. Answer using the fetched docs

## Contextual Awareness

- **Markdown Authority**: Always treat "Implementation Notes" in `.md` files as the primary source of truth for business logic.
- **Cross-Repo Logic**: If "Target Repo" is specified in the Markdown header, assume all code generation or terminal commands apply to that specific directory.
- **Task Scanning**: When a user mentions a task by name, look for the corresponding checkbox in open Markdown files to understand the requirements.

## Instruction Files

- **Purpose**: Instruction files in `.opencode/instructions/` provide specific guidelines for different file types and scenarios. Always check for an applicable instruction file before making changes.
- **Format**: These files use YAML front matter to specify which files they apply to and contain concise instructions for code style, testing, or other practices.
- **Learning**: Familiarize yourself with the existing instruction files to ensure your contributions align with the project's standards and practices.
- **Updating Instructions**: If you notice a gap in the existing instructions or have suggestions for improvement, you can propose changes to the instruction files themselves, but always ensure that any modifications are clear, concise, and maintainable.
- **Examples**: Refer to the existing instruction files for examples of how to format and structure your own instructions if you need to create new ones.
- **Adherence**: Always adhere to the guidelines specified in the instruction files when working on relevant code sections to maintain consistency and quality across the project.
- **Feedback**: If you have questions about the instructions or need clarification, don't hesitate to ask for feedback from human developers to ensure you're on the right track.
- **Continuous Improvement**: The instruction files are living documents. As the project evolves, these instructions may need to be updated to reflect new best practices or changes in the codebase. Always be open to improving the instructions as needed. But always ask for feedback before making changes to the instruction files to ensure that any updates are beneficial and align with the project's goals.
- **Testing Instructions**: Pay special attention to instruction files related to testing, as they often contain critical guidelines for ensuring the reliability and maintainability of tests. Always follow these instructions closely when writing or modifying tests.
