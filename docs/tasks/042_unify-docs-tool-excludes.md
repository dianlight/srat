<!-- DOCTOC SKIP -->

# [REFACTOR]: Unify documentation tool excludes via single .docsignore file

**Target Repo:** `srat`  
**Status:** 📅 Planned  
**Issue Link:** _TBD_

## 🎯 Objective

Eliminate duplicated exclude/file-set configuration across all documentation validation tools by introducing a single `.docsignore` file as the source of truth. All tools (markdownlint-cli2, Vale, hk, validation scripts, doctoc) must reference this file so that hk, IDE, CI, and local runs produce identical results.

## 🛠️ Technical Specifications

- **Inputs:** Current exclude patterns from `.vale.ini`, `.markdownlint-cli2.jsonc`, `hk.pkl`, `scripts/validate-docs.sh`, `mise run docs-toc`
- **Outputs:** Single `.docsignore` file + updated tool configs that reference it
- **Dependencies:** markdownlint-cli2, Vale, hk, mise, shell scripts
- **Scope:** Documentation tools only (markdownlint + Vale + hk orchestration). Lychee and cspell are out of scope.

## 📝 Task List

- [ ] Task 1: Audit and merge all current exclude patterns into a canonical list
- [ ] Task 2: Create `.docsignore` as the single source of truth (gitignore-style syntax)
- [ ] Task 3: Update `.markdownlint-cli2.jsonc` to derive ignores from `.docsignore`
- [ ] Task 4: Update `.vale.ini` to derive ignores from `.docsignore`
- [ ] Task 5: Update `hk.pkl` to reference `.docsignore` for file selection
- [ ] Task 6: Update `scripts/validate-docs.sh` to use `.docsignore` for both markdownlint and Vale invocations
- [ ] Task 7: Update `mise run docs-toc` task to use `.docsignore` instead of inline find excludes
- [ ] Task 8: Remove orphaned/disabled tool configs (Lychee `.lychee.toml`, `.lycheecache`) or document decision
- [ ] Task 9: Run all validation paths (local `mise run docs-validate`, `hk check`, CI simulation) and verify identical results
- [ ] Task 10: Update `scripts/validate-docs.sh` or create a helper that reads `.docsignore` and passes correct flags to each tool
- [ ] Task 11: Documentation — update any docs referencing the old scattered configs
- [ ] Task 12: Capture lessons learned and update documentation

## 🧠 Implementation Notes (Copilot Context)

### Canonical exclude list (from audit)

These directories/files must be excluded across all tools:

```
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
```
mise run docs-validate    # local full validation
hk check                  # pre-commit check mode
hk fix                    # pre-commit fix mode (no-ops on clean files)
```

All three must scan the same set of files and report identical issues.

## 🔗 Code References & TODOs

- [ ] `TODO: .markdownlint-cli2.jsonc` — remove hardcoded `ignores` array
- [ ] `TODO: .vale.ini` — remove hardcoded `Ignore` patterns
- [ ] `TODO: hk.pkl` lines 52-61 — remove inline exclude patterns
- [ ] `TODO: scripts/validate-docs.sh` — replace all `find ... -not -path` with `.docsignore` reader
- [ ] `TODO: .mise.toml` line 69 — replace `docs-toc` find excludes
- [ ] `FIXME: .lychee.toml` — disabled tool with full config; decide to enable or remove
