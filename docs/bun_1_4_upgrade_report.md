# Bun 1.4 Frontend Upgrade Report

<!-- DOCTOC SKIP -->

Date: 2026-08-23
Scope: `frontend/` toolchain upgrade from Bun 1.3.14 to Bun 1.4.0 ([release notes](https://bun.com/blog/bun-v1.4)), adoption of features that speed up tests and compiles, plus instruction/doc updates.

## Summary

| Metric (macOS arm64, this machine)                     | Bun 1.3.14 baseline | Bun 1.4.0 after   | Δ                 |
| ------------------------------------------------------ | ------------------- | ----------------- | ----------------- |
| Unit test suite (97 files / 753 tests, Node runtime)   | 95.7 s              | 93.1 s            | ≈ unchanged       |
| Unit test suite (**Bun runtime**, `--bun`)             | n/a                 | **60.9 → 46.3 s** | **−36 % … −52 %** |
| Full test task (`mise run //frontend:test`, tsc+tests) | ~99.6 s             | **49.3 s**        | **≈ −50 %**       |
| Type check `bun tsc --noEmit`                          | 3.93 s              | 3.42 s            | −13 %             |
| Type check `tsconfig.prod.json`                        | 0.90 s              | 0.93 s            | unchanged         |
| Production bundle (`bun.build.ts`)                     | 0.80 s              | 0.76 s            | unchanged         |
| Cold install (`--frozen-lockfile`, warm cache)         | 5.23 s              | 4.49 s            | −14 %             |
| Duplicate packages in `bun.lock`                       | —                   | 3 removed         | leaner graph      |

The headline win is running Vitest on the Bun 1.4 runtime: the module-import phase dropped from ~59.6 s to ~13.3–16.3 s and the happy-dom environment setup from ~17.2 s to ~7.5–9.3 s. All 752 tests pass; verified stable across repeated runs.

## Changes

### Version pins

| File                    | Change                                                               |
| ----------------------- | -------------------------------------------------------------------- |
| `.mise.toml`            | `bun = "1.3.14"` → `bun = "1.4.0"`                                   |
| `frontend/package.json` | `packageManager: bun@1.4.0`; `engines.bun >=1.4.0`                   |
| `frontend/package.json` | `@types/bun` `1.3.14` → `1.4.0`                                      |
| `frontend/bun.lock`     | Migrated to Bun 1.4 format (`configVersion: 1`) + duplicates removed |

CI picks the version up automatically via `oven-sh/setup-bun` with `bun-version-file: frontend/package.json`.

### Test workflow (frontend/.mise.toml)

- `//frontend:test` and `//frontend:test:new` now run `bunx --bun vitest run …` (Vitest on the Bun runtime).
- `//frontend:test:ci` intentionally stays on the Node runtime—see known issue below.
- `bun dedupe` removed 3 duplicate versions from `bun.lock`.

### Docs / instructions updated

- `AGENTS.md`—new "Bun 1.4" bullet under Frontend essentials (pin location, `--bun` rule, coverage caveat).
- `.opencode/instructions/frontend_test.instructions.md`—all examples use `bunx --bun vitest`; added coverage caveat.
- `.opencode/instructions/typescript-7-es2022.instructions.md`—verification command updated.
- `frontend/README.md`—new "Bun 1.4" section (requirement, test runtime, breaking-change review).

## Features evaluated

### Adopted

- **Vitest on the Bun runtime** (`bunx --bun vitest`)—the big test speedup (~2× end-to-end task time).
- **Lock file migration + `bun dedupe`**—Bun 1.4 lock format (`configVersion: 1`), 3 duplicate versions removed.

### Evaluated, not adopted

- **Isolated linker / global virtual store** (`[install] linker = "isolated"`)—up to 7× faster warm installs on Linux CI, but changes `node_modules` layout to symlinks and needs re-validation of both patched dependencies. Decision: **keep hoisted linker** (user choice); macOS local installs are already ~5 s. Revisit if Linux CI install time becomes a problem.
- **`bun test --parallel`**—project standard is Vitest, not `bun:test`; migrating runners would be high-risk/low-reward.
- **`bun run --parallel`**—current scripts have strict sequencing (type check before tests/build), no fan-out opportunity.
- **`bun prune --production`**—no Docker image builds the frontend; nothing to prune.

## Breaking changes reviewed (Bun 1.3 → 1.4)

None affect us:

- Bun does not load `.env` files automatically under `bunx --bun`: we pass environment variables explicitly via mise tasks; no `.env` files exist in `frontend/`.
- `Bun.YAML` YAML 1.2 boolean change: app uses `js-yaml`, not `Bun.YAML`.
- Strict `bunfig.toml` parsing: our bunfig is minimal and valid.
- New monorepo setups default to the isolated linker: we stay hoisted explicitly (single package project).
- `process.versions.modules` 147 / `res.writeHeader()` removal: no native addons or raw `node:http` server usage in the frontend.

## Known issue

**Coverage under the Bun runtime crashes**: with `bunx --bun vitest run --coverage`, all tests pass but report generation fails inside `@bcoe/v8-coverage` merge (`mergeRangeTreeChildren` recursion) when merging Bun-profiler output. Coverage therefore runs on the Node runtime (`//frontend:test:ci`, verified working: 95.1 s + reports). Track upstream; revisit when fixed.

## How to verify

```bash
mise run //frontend:test        # tsc + full suite on Bun runtime (~50 s)
mise run //frontend:test:ci    # coverage on Node runtime
cd frontend && bun tsc --noEmit # type check
```
