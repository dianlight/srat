<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->
**Table of Contents** *generated with [DocToc](https://github.com/thlorenz/doctoc)*

- [Release process - using scripts/release.sh via mise](#release-process---using-scriptsreleasesh-via-mise)
  - [Purpose](#purpose)
  - [Location](#location)
  - [Prerequisites](#prerequisites)
  - [Usage (run via mise)](#usage-run-via-mise)
  - [Common flags](#common-flags)
  - [Notes & best practices](#notes--best-practices)
  - [Troubleshooting](#troubleshooting)
  - [Where to look in the repo](#where-to-look-in-the-repo)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

# Release process - using scripts/release.sh via mise

This document explains how to run the repository release helper script (scripts/release.sh) using the project's task runner (`mise`). It summarizes prerequisites, common flags, and an example invocation.

## Purpose

The `scripts/release.sh` helper automates the common release workflow used by this repository: calculating the next tag, updating `CHANGELOG.md`, pushing to `main`, and publishing a GitHub release draft via the `gh` CLI. Running it via `mise` ensures the environment and project tooling are used consistently.

## Location

Script: `scripts/release.sh`

## Prerequisites

- `git` configured with push access to the `dianlight/srat` repository
- `gh` (GitHub CLI) installed and authenticated
- `jq` installed (used by some invocations in CI/automation)
- `mise` available on your PATH (repository task runner)
- A clean working tree (or use `--ignore-uncommitted` to override)

Before making a release it's recommended to run the test and quality gates:

```sh
# run the project's pre-commit / checks
hk check

# run backend and frontend tests (examples)
mise run //backend:test
mise run //frontend:test
```

## Usage (run via mise)

You can invoke the release helper through `mise` (the project task runner). Example:

```sh
# makes an RC release (example)
mise run release --version 2026.04.0-rc4
```

If you omit `--version`, the script calculates the next patch-style version for the current year/month automatically.

## Common flags

- `--version <version>` — specify exact tag/title for the release (e.g. `2026.04.0-rc4`)
- `--ignore-uncommitted` — allow running with local uncommitted changes
- `--no-wait` — error out instead of waiting for draft/release/workflow conditions
- `--interactive` — prompt for confirmation before each push/commit/publish step
- `--help` — show the script help message

## Notes & best practices

- The script expects an `Unreleased` section in `CHANGELOG.md` which it replaces with the released version header. Keep CHANGELOG updates minimal and review the generated commit before publishing.
- Prefer running the script with `--interactive` the first few times to confirm behaviour.
- The script uses `gh` to find/update a draft release and to publish it. Ensure the authenticated `gh` account has permission to publish releases.
- The script will attempt several retries when publishing the release; CI checks (GitHub Actions) must be complete before final publish succeeds.

## Troubleshooting

- If the script fails waiting for a draft release, either create the draft via the GitHub UI or re-run with `--no-wait` to fail fast and diagnose locally.
- If CHANGELOG editing is undesirable for a particular release, the script accepts continuing without editing when it detects the version already present (it will prompt in interactive mode).

## Where to look in the repo

- Release script: `scripts/release.sh`
- CHANGELOG: `CHANGELOG.md`
- Example invocation (used during local/manual releases): `mise run release --version 2026.04.0-rc4`

If you'd like the release workflow added to CI or a helper `mise` alias with additional safety checks, open an issue or a PR referencing this document.
