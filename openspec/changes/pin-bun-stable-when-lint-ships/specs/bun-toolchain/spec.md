# Spec Delta

## Purpose

Defines how the SRAT monorepo pins and upgrades the Bun runtime across root tooling, frontend, oauth broker, and lint hooks, including the conditions under which the repository may leave the canary channel.

## ADDED Requirements

### Requirement: Canary installation must be reproducible in CI

While the repository depends on canary-only `bun lint`, the Bun runtime SHALL be installed through a method that works on a clean CI runner without a pre-installed Bun binary, on Linux (amd64 and arm64) and macOS.

#### Scenario: Clean CI runner installs Bun canary

- **WHEN** a CI job with no pre-installed Bun runs `mise install` on Linux amd64
- **THEN** Bun canary is installed and `bun lint --help` succeeds

#### Scenario: macOS developer installs Bun canary

- **WHEN** a developer on macOS runs `mise install` with the repository config
- **THEN** Bun canary is installed and `bun lint --help` succeeds

### Requirement: Stable migration trigger

The repository SHALL migrate from canary to a pinned stable Bun version only after that stable version provides the built-in `bun lint` command.

#### Scenario: Stable version provides bun lint

- **WHEN** a stable Bun release is available whose `bun lint --help` succeeds on Linux and macOS
- **THEN** the migration to that pinned stable version is eligible to be applied

#### Scenario: Stable version lacks bun lint

- **WHEN** a stable Bun release is available but `bun lint --help` fails with `Script not found "lint"`
- **THEN** the repository SHALL remain on canary

### Requirement: Stable migration removes canary-only workarounds

When migrating to stable Bun, all canary-only configuration SHALL be removed so the toolchain no longer depends on mutable release metadata.

#### Scenario: Canary workarounds removed

- **WHEN** the repository is pinned to a stable Bun version with built-in lint
- **THEN** `use_versions_host = false`, `fetch_remote_versions_cache = "5m"`, and the `[tool_alias] bun = "github:oven-sh/bun"` entry are absent from `.mise.toml`

#### Scenario: Package managers pinned to stable

- **WHEN** the repository is pinned to a stable Bun version with built-in lint
- **THEN** `frontend/package.json` and `oauth_broker/package.json` declare `packageManager` as `bun@<stable-version>`

### Requirement: Post-migration verification

After migration, the lint hook and CI toolchain SHALL work across supported platforms and the Docker/Home Assistant OS runtime environment.

#### Scenario: Lint hook works after migration

- **WHEN** `hk check --all --step bun-lint` is invoked on macOS or Linux
- **THEN** the built-in `bun lint` runs and the check completes without a canary channel

#### Scenario: Build workflow works after migration

- **WHEN** the build workflow runs on a clean Linux runner inside the project's Docker/HA OS build environment
- **THEN** `mise install` and `mise run //frontend:prepare` succeed with the pinned stable Bun
