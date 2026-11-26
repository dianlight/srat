# Proposal

## Why

The repository currently tracks the Bun `canary` channel because stable Bun 1.4.5 lacks the built-in `bun lint` command. Canary builds are mutable GitHub releases with changing asset digests, requiring fragile mise workarounds (`use_versions_host = false`, `fetch_remote_versions_cache = "5m"`, GitHub tool alias) and causing CI failures when the install method changes. Once a stable Bun release ships `bun lint`, the canary dependency and all its workarounds can be removed.

## What Changes

- Restore the original canary install method (GitHub backend via `[tool_alias]`, `use_versions_host = false`, `fetch_remote_versions_cache = "5m"`) to fix the current CI failure caused by the broken npm-backend approach.
- Add this tracking change to migrate off canary as soon as a stable Bun version includes `bun lint`.
- Define the acceptance criteria for detecting stable `bun lint` availability and the exact files to update when migrating.
- Provide verification steps covering Linux (multi-architecture), macOS, and the Docker/Home Assistant OS runtime environment.

## Capabilities

### New Capabilities

- `bun-toolchain`: management of the Bun runtime version across monorepo toolchains (mise, frontend, oauth_broker, hk hooks), including canary-to-stable migration criteria.

### Modified Capabilities

(none)

## Impact

- Root config: `.mise.toml` (settings, tool alias, bun tool version).
- Frontend: `frontend/package.json` (`packageManager`), `bunfig.toml` (lint config comments).
- OAuth broker: `oauth_broker/package.json` (`packageManager`).
- Hooks: `hk.pkl` (bun-lint step comments).
- CI: `.github/workflows/build.yaml`, `.github/workflows/documentation.yml` (GitHub token comments).
- No backend Go, custom component Python, or runtime API changes.
