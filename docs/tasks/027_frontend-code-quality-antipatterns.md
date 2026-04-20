<!-- DOCTOC SKIP -->

# [REFACTOR]: Frontend Code Quality — Anti-patterns & Instruction Violations

**Target Repo:** `srat`
**Status:** 📅 Planned
**Issue Link:**

## ⚠️ Compliance & Pre-reqs

- This task is a `[REFACTOR]`. Per `.github/copilot-instructions.md` you MUST ask whether to run the `prepare-refactor` check before editing source files. If the user opts in, create `docs/refactors/<slug>.md` and follow the prepare-refactor phases (identify impacted functions, ensure tests exist, record baseline, run post-refactor verification).
- Read the header of any file you modify before changing it.
- Do NOT run git add/commit/push unless explicitly asked.
- Follow the rules in `.github/instructions/` (notably `reactjs.instructions.md`, `react-hook-form-mui.instructions.md` and `frontend_test.instructions.md`).
- Do NOT edit generated files: `frontend/src/store/sratApi.ts`, `frontend/src/store/wsApi.ts`, or `backend/docs/openapi.*`. If new DTOs/types are required, update Go DTOs and run `cd frontend && bun run gen`.
- Bugfix rule: always add a failing test reproducing the bug first, then implement the fix, then re-run tests.
- Frontend tests: use `@testing-library/user-event` only, await all interactions, prefer semantic queries (`getByRole`, `getByLabelText`) — use `getByTestId` only as a last resort.
- Branch naming: use `refactor/<kebab-case-slug>` (see Copilot instructions for examples).

## 🎯 Objective

Fix recurring frontend code-quality anti-patterns: deprecated testing practices (`fireEvent`, missing `await`), excessive `getByTestId` usage, TS5 HMR casts, raw `console.*` in production code, and raw `fetch()` calls bypassing RTK Query.
Remove also use of inputs without the use of `react-hook-form`'s form state management and validation, such as `useState` for field values, validation errors, submit loading, password show/hide, and manual form submission handling.

## 🛠️ Technical Specifications

- **Inputs:** Files under `frontend/src/`
- **Outputs:** Instruction-compliant source with passing tests and lint
- **Dependencies:** `@testing-library/user-event`, existing RTK Query slices (`githubApi`, `sratApi`), TypeScript 6+, Bun toolchain
- **Acceptance criteria:**
  - All changed tests pass locally with `mise run //frontend:test --rerun-each 10` for touched tests
  - Full frontend test suite (`mise run //frontend:test`) and lint (`mise run //frontend:lint`) pass with zero new errors
  - `docs/refactors/<slug>.md` exists if prepare-check was chosen
  - No edits to generated files (`sratApi.ts`, `wsApi.ts`, `backend/docs/openapi.*`)

## 📝 Task List (detailed)

Preflight:
- [ ] PREP: Ask user whether to run a prepare-refactor check (recommended). If Yes — create `docs/refactors/<slug>.md` and complete Phases 1–4 before changing production code.

Main tasks:
- [ ] Task 1 — Tests: Replace `fireEvent` with `userEvent` (use `const user = userEvent.setup()` and await all interaction calls). File: `frontend/src/pages/volumes/components/__tests__/FilesystemLabelFormatDialog.test.tsx`.
- [ ] Task 2 - Search & fix any use of `useState` for form field values, validation errors, submit loading state, password show/hide toggles, or manual form submission handling. Replace with `react-hook-form` and `react-hook-form-mui` patterns as per `.github/instructions/react-hook-form-mui.instructions.md`.
- [ ] Task 3 — Tests: Fix non-awaited `userEvent` usages in `frontend/src/components/__tests__/NavBar.test.tsx` — consistently use `user = userEvent.setup()` and `await user.click/type/hover`.
- [ ] Task 4 — Tests: Replace `getByTestId` in `frontend/src/components/__tests__/DonationButton.test.tsx` with semantic queries (`getByRole`, `getByLabelText`). If the component lacks an accessible name, prefer adding an `aria-label` or visible accessible text (document the change and update tests) rather than `data-testid`.
- [ ] Task 5 — HMR: Update `frontend/src/pages/dashboard/metrics/DiskHealthMetrics.tsx` to use TS6 native HMR: `if (import.meta.hot) { import.meta.hot.accept(...) }` (remove `as any` casts).
- [ ] Task 6 — Data fetching: Replace raw `fetch()` in `frontend/src/hooks/githubNewsHook.ts` with the `githubApi` RTK Query endpoint (or use `sratApi.endpoints.<endpoint>.useLazyQuery()` for imperative usage). Do not add a new raw fetch wrapper.
- [ ] Task 7 — Production logs: Audit and remove/replace `console.log/error/warn` in production source files. For errors, route to the app's notification/problem-handling system; do not silently swallow errors.
- [ ] Task 8 — Unit verification: For each modified test file, run `mise run //frontend:test --rerun-each 10` and resolve flaky or failing tests.
- [ ] Task 9 — Integration verification: Run `mise run //frontend:test`, `mise run //frontend:lint`, and `mise run docs-validate`. Fix any issues.
- [ ] Task 10 — Docs & lessons: Capture lessons in this task file and (if prepare-check used) finalise `docs/refactors/<slug>.md`. Run `mise run docs-validate`.
- [ ] Task 11 — PR: Prepare a PR on branch `refactor/frontend-code-quality-anti-patterns` (kebab-case if title changes). Include the prepare-check summary, test commands run, and verification steps in the PR body.

## 🧠 Implementation Notes (guidance to implementer)

- Follow `.github/instructions/reactjs.instructions.md` and `.github/instructions/fontend_test.instructions.md`.
- Tests: always initialize `const user = userEvent.setup()` and `await` every interaction.
- Semantic queries: prefer `screen.getByRole('button', { name: /donate/i })` over `getByTestId`.
- RTK Query migration: prefer existing endpoints in `frontend/src/store/githubApi.ts`. For imperative usage, call `sratApi.endpoints.<endpoint>.useLazyQuery()`.
- HMR: remove `as any` cast and use `import.meta.hot`.
- Logging: replace `console.log` debug lines; for errors, use the app's notification or error handling pattern rather than quiet logging.
- Generated files: do not modify `sratApi.ts` or `wsApi.ts`. If new DTOs are required, update backend DTOs and re-run `bun run gen`.

## 🔧 How to run & verify locally

- Install / prepare frontend dev environment (Bun) and deps via the repo's standard commands.

Run changed tests (with retry for flakiness):
```bash
cd frontend
mise run //frontend:test --rerun-each 10 --filter "FilesystemLabelFormatDialog"
```

Run full verification:
```bash
mise run //frontend:test
mise run //frontend:lint
mise run docs-validate
```

## ✅ Acceptance Criteria

- All modified tests pass locally (use `--rerun-each 10` when re-running touched tests)
- Full frontend tests and lint pass with zero new errors
- Prepare-refactor tracking doc present if prepare-check was used
- No direct edits to generated files (`sratApi.ts`, `wsApi.ts`, `backend/docs/openapi.*`)
- PR created with branch name following convention and CI green

## 🔗 References

- `.github/copilot-instructions.md` (must-follow project rules)
- `.github/instructions/reactjs.instructions.md`
- `.github/instructions/fontend_test.instructions.md`
- `.github/instructions/react-hook-form-mui.instructions.md`
- `.github/skills/prepare-refactor/SKILL.md`

---

Make sure to run the prepare-check prompt before starting since this task is a `[REFACTOR]`. If you want, I can proceed and run the prepare-check now.
