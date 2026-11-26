# Design

## Context

Bun 1.4.5 was pinned as stable in commit `66e4bb7c2`, but the built-in `bun lint` command was not included, so commit `8a1eb6273` re-enabled canary. The first re-enable attempt used `npm:bun = "canary"` with `npm.package_manager = "bun"`, which created a circular dependency and broke every npm-tool install in CI. The original canary method (before `66e4bb7c2`) used the GitHub backend with a tool alias and version-host workarounds; that method is restored here as the baseline. See proposal.md for motivation and `specs/bun-toolchain/spec.md` for the behavior contract.

The monorepo pins runtime versions in `.mise.toml`, `frontend/package.json`, and `oauth_broker/package.json`; the lint hook is defined in `hk.pkl` and configured in `bunfig.toml`. CI runs on Linux amd64 runners; developers use macOS; build artifacts target Docker/Home Assistant OS on amd64 and arm64.

## Goals / Non-Goals

**Goals:**

- Restore a CI-safe canary install method immediately.
- Define a repeatable, verified path to pin stable Bun once `bun lint` ships in a stable release.
- Remove mutable-release workarounds when leaving canary.

**Non-Goals:**

- Changing application code (backend Go, frontend React, custom component Python, oauth broker).
- Introducing a different linter or formatter.
- Upgrading unrelated toolchain versions.

## Decisions

- **Use GitHub backend via `[tool_alias]` for canary.** The alias keeps the tool named `bun` so `npm.package_manager = "bun"` can resolve it and order it before `npm:` tools. The npm backend cannot work because installing `npm:bun` requires Bun to already be installed. Alternative: pin stable and drop lint — rejected because the lint hook depends on `bun lint`.
- **Keep `use_versions_host = false` and a short `fetch_remote_versions_cache` while on canary.** The `canary` tag is rebuilt in place; cached metadata and digests go stale. Alternative: rely on default cache — rejected because it causes checksum mismatches in CI.
- **Keep `MISE_GITHUB_TOKEN` in workflows while canary is active.** Direct api.github.com metadata lookups need authentication to avoid the anonymous rate limit. The token can be removed from the canary-specific comments once stable pinning removes the metadata lookup.
- **Gate stable migration on `bun lint --help` succeeding in a stable release.** The failure mode in 1.4.5 was `error: Script not found "lint"`, so the command's help output is the observable capability check.
- **Remove all canary workarounds at migration time.** `use_versions_host`, `fetch_remote_versions_cache`, and the tool alias must be deleted together; leaving any behind keeps CI tied to mutable metadata.

## Risks / Trade-offs

- **[Canary release changes under CI]** → Fetch metadata directly from api.github.com with a short cache and authenticated token.
- **[Stable release claims lint but binary differs by platform]** → Verify `bun lint --help` on Linux amd64/arm64 and macOS before applying the pin.
- **[Migration leaves stale comments/config]** → Check the exact file list in tasks.md and validate with `openspec validate`.
- **[Tool alias confuses other tooling]** → Keep the alias name `bun` (not `github:oven-sh/bun`) so mise's npm dependency ordering still resolves it.

## Migration Plan

1. Restore the GitHub-backend canary setup (already applied in the current working tree) and verify `mise install` and `mise run //frontend:prepare` pass.
2. Monitor Bun stable releases for built-in `bun lint`.
3. When a candidate stable version is found, verify `bun lint --help` on Linux (amd64 and arm64) and macOS.
4. Apply the migration described in tasks.md: pin stable versions, remove canary settings and comments, restore workflow comments.
5. Run full validation (`mise install`, backend/frontend tests, `hk check`) and archive this change.

## Open Questions

None.
