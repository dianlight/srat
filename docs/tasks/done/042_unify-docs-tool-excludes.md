<!-- DOCTOC SKIP -->

# [REFACTOR]: Unify documentation tool excludes via single .docsignore file

**Target Repo:** `srat`  
**Status:** ✅ Complete  
**Issue Link:** https://github.com/dianlight/srat/issues/653

## 🎯 Objective

Eliminate duplicated exclude/file-set configuration across all documentation validation tools by introducing a single `.docsignore` file as the source of truth. All tools (markdownlint-cli2, Vale, hk, validation scripts, doctoc) must reference this file so that hk, IDE, CI, and local runs produce identical results.

## 🛠️ Technical Specifications

- **Inputs:** Current exclude patterns from `.vale.ini`, `.markdownlint-cli2.jsonc`, `hk.pkl`, `scripts/validate-docs.sh`, `mise run docs-toc`
- **Outputs:** Single `.docsignore` file + updated tool configuration files that reference it
- **Dependencies:** markdownlint-cli2, Vale, hk, mise, shell scripts
- **Scope:** Documentation tools only (markdownlint + Vale + hk orchestration). cspell is out of scope.

## 📝 Task List

- [x] Task 1: Audit and merge all current exclude patterns into a canonical list
- [x] Task 2: Create `.docsignore` as the single source of truth (gitignore-style syntax)
- [x] Task 3: Update `.markdownlint-cli2.jsonc` to derive ignores from `.docsignore`
- [x] Task 4: Update `.vale.ini` to derive ignores from `.docsignore`
- [x] Task 5: Update `hk.pkl` to reference `.docsignore` for file selection
- [x] Task 6: Update `scripts/validate-docs.sh` to use `.docsignore` for both markdownlint and Vale invocations
- [x] Task 7: Update `mise run docs-toc` task to use `.docsignore` instead of inline find excludes
- [x] Task 8: Remove orphaned/disabled legacy link-check configuration files or document decision
- [x] Task 9: Run all validation paths (local `mise run docs-validate`, `hk check`, CI simulation) and verify identical results
- [x] Task 10: Update `scripts/validate-docs.sh` or create a helper that reads `.docsignore` and passes correct flags to each tool
- [x] Task 11: Documentation - update any docs referencing the old scattered tool configuration files
- [x] Task 12: Capture lessons learned and update documentation

## 🧠 Implementation Notes (Copilot Context)

- Work start gate: user chose to continue on `main` (no branch split for this task).

### Canonical exclude list (from audit)

These directories/files must be excluded across all tools:

```text
.ai/
.github/
.opencode/
.vale/
backend/src/vendor/
custom_components/.venv/
docs/refactors/
docs/tasks/
frontend/node_modules/
```

Single-file exclusions (not directories):
- `CHANGELOG.md` (Vale only — auto-generated, not prose-edited)
- `CLAUDE.md` (Vale only — AI system prompt)
- `AGENTS.md` (Vale only — AI system prompt)
- `docs/memory-index.md` (markdownlint only — auto-generated)

### Implemented changes summary

- Added `.docsignore` with shared and tool-scoped patterns (`# tools: ...` metadata comments).
- Added `scripts/docs-ignore.sh` helper to emit:
   - tool-specific pattern lists,
   - markdownlint CLI ignore arguments,
   - `.md` file lists for tool runs.
- Updated `.markdownlint-cli2.jsonc`:
   - removed hardcoded ignore list,
   - set `globs: []` so caller-provided file lists are authoritative.
- Updated `.vale.ini` to remove hardcoded `Ignore` excludes and use `.docsignore`-driven selection from scripts.
- Updated `hk.pkl` to call `./scripts/validate-docs.sh --markdownlint-only`, `--markdownlint-fix-only`, and `--vale-only`.
- Updated `scripts/validate-docs.sh` to use `.docsignore` via `scripts/docs-ignore.sh`.
- Updated `.mise.toml` `docs-toc` task to use `.docsignore` file list from helper.
- Updated docs workflow path filters in `.github/workflows/documentation.yml` to include `.docsignore` and `scripts/docs-ignore.sh`.
- Updated docs:
   - `docs/DOCUMENTATION_GUIDELINES.md`
   - `docs/DOCUMENTATION_VALIDATION_SETUP.md`

### Task 8 decision

- Removed legacy link-check configuration from the repository.
- No legacy link-check cache file was present, so no removal was needed.

### Validation evidence

- `mise run docs-validate` ✅
   - markdownlint: 59 files, 0 errors
   - Vale: warnings only (non-blocking), overall pass
- `hk fix` ✅
- `hk check` ✅
- `mise run docs-toc` ✅ (TOC updated where required)
- `hk check` re-run after TOC updates ✅

### Tool-specific adaptation strategy

1. **`.docsignore`** — Plain text, one pattern per line, gitignore-compatible. Supports `#` comments and `!` negation.

2. **markdownlint-cli2** — Does not natively read ignore files. Options:
   - Use `--ignore` CLI flag populated from `.docsignore` via a shell helper
   - Or keep `ignores` array in `.markdownlint-cli2.jsonc` but generate it from `.docsignore` via a script

3. **Vale** — Supports `--glob` and can read patterns from config. The `Ignore` field in `.vale.ini` accepts glob patterns. Can be driven from `.docsignore` via a pre-processing script or by using Vale's `--glob='!{pattern}'` CLI flags.

4. **hk.pkl** — Pkl config. Can read external files via Pkl's `read()` function or use a shell command to populate the file list.

5. **scripts/validate-docs.sh** — Already uses `find` with `-not -path` patterns. Replace inline excludes with a loop that reads `.docsignore`.

6. **mise run docs-toc** — Currently uses `find ... -not -path`. Replace with `.docsignore`-driven find.

### Key constraint

Do NOT break the existing tool configs — each tool still needs its own config file for tool-specific settings (Vale styles, markdownlint rule overrides, etc.). Only the **exclude/file-set** portion moves to `.docsignore`.

### Verification

After changes, run all three paths and compare output:
```bash
mise run docs-validate    # local full validation
hk check                  # pre-commit check mode
hk fix                    # pre-commit fix mode (no-ops on clean files)
```

All three must scan the same set of files and report identical issues.

## 🔗 Code References & TODOs

- [x] `TODO: .markdownlint-cli2.jsonc` - remove hardcoded `ignores` array
- [x] `TODO: .vale.ini` - remove hardcoded `Ignore` patterns
- [x] `TODO: hk.pkl` lines 52-61 - remove inline exclude patterns
- [x] `TODO: scripts/validate-docs.sh` - replace all `find ... -not -path` with `.docsignore` reader
- [x] `TODO: .mise.toml` line 69 - replace `docs-toc` find excludes
- [x] `FIXME: legacy link-check config` - removed
