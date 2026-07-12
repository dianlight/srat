<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->

**Table of Contents** *generated with [DocToc](https://github.com/thlorenz/doctoc)*

- [Release process (GitHub Actions)](#release-process-github-actions)
  - [Purpose](#purpose)
  - [How it works](#how-it-works)
  - [Usage](#usage)
  - [Manual trigger from GitHub UI](#manual-trigger-from-github-ui)
  - [What happens behind the scenes](#what-happens-behind-the-scenes)
  - [Prerequisites](#prerequisites)
  - [Troubleshooting](#troubleshooting)
  - [Where to look in the repo](#where-to-look-in-the-repo)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

# Release process (GitHub Actions)

This document explains how to create a release using the automated GitHub Actions workflows.

## Purpose

The release process automates version tagging, changelog management, building, and publishing. All running server-side in GitHub Actions. No local polling or waiting required.

## How it works

The release is orchestrated by three GitHub Actions workflows:

1. **`release.yaml`** — Triggered manually. Calculates the version (or uses a provided one), updates `CHANGELOG.md`, and pushes a release commit to `main`.
2. **`build.yaml`** — Triggered by the push. Detects the release commit via its message, builds all artifacts, and creates a **draft** GitHub release with the correct version and assets.
3. **`release-publish.yaml`** — Triggered automatically when `build.yaml` completes. Finds the draft release, publishes it, and resets `CHANGELOG.md` for the next development cycle.

```mermaid
flowchart TD
    A[Developer triggers release] --> B[release.yaml]
    B -->|Updates CHANGELOG, pushes| C[build.yaml]
    C -->|Tests + Build + Draft release| D[release-publish.yaml]
    D -->|Publishes release| E[✅ Release live]
    D -->|Resets CHANGELOG| F[Next dev cycle]
```

## Usage

### From the command line (via mise)

```sh
# Auto-calculate version (next patch for current year.month)
mise run release

# Specify exact version
mise run release -- --version 2026.07.1
```

### From GitHub CLI

```sh
# Auto-calculate version
gh workflow run release.yaml --ref main

# Specify version
gh workflow run release.yaml --ref main -f version=2026.07.1
```

## Manual trigger from GitHub UI

1. Go to **Actions → Release → Run workflow**
2. Optionally enter a version (e.g. `2026.07.1`). Leave empty for auto-calculation.
3. Click **Run workflow**

The workflow will handle everything automatically. You can monitor progress in the Actions tab.

## What happens behind the scenes

| Step | Workflow | What it does |
| ------ | ---------- | ------------- |
| 1 | `release.yaml` | Calculates version (or uses input) |
| 2 | `release.yaml` | Replaces `## [ 🚧 Unreleased ]` with `## <version>` in CHANGELOG.md |
| 3 | `release.yaml` | Commits `chore(release): <version>` and pushes to `main` |
| 4 | `build.yaml` | Detects release commit, uses version from commit message |
| 5 | `build.yaml` | Runs backend, frontend, and custom component tests |
| 6 | `build.yaml` | Builds all binaries and HACS component |
| 7 | `build.yaml` | Creates a **draft** GitHub release with all assets |
| 8 | `release-publish.yaml` | Detects release commit from workflow_run event |
| 9 | `release-publish.yaml` | Publishes the draft release (sets `draft=false`) |
| 10 | `release-publish.yaml` | Resets CHANGELOG.md for next development cycle |

## Prerequisites

- GitHub Actions must be enabled for the repository
- The `GITHUB_TOKEN` must have `contents: write` permission (default for Actions)
- A clean `CHANGELOG.md` with an `## [ 🚧 Unreleased ]` section

## Troubleshooting

### Build failed after CHANGELOG was updated

The CHANGELOG commit was pushed but `build.yaml` failed. To recover:
- Check the build failure in Actions → build
- Fix the issue and push to `main`
- The build will re-trigger and `release-publish.yaml` will pick up the release

### No draft release found after build succeeded

If `release-publish.yaml` reports "No draft release found", check:
- The build.yaml `create-release` job logs
- Whether a draft with the expected tag exists in GitHub Releases

### Re-running a failed release

If the release commit is already on `main` but the workflow failed:
```sh
# Re-trigger just the build
gh workflow run build.yaml --ref main

# Or re-trigger the full release process
gh workflow run release.yaml --ref main -f version=2026.07.1
```

## Where to look in the repo

- Release workflow: `.github/workflows/release.yaml`
- Publish workflow: `.github/workflows/release-publish.yaml`
- Build workflow (creates draft): `.github/workflows/build.yaml`
- CHANGELOG: `CHANGELOG.md`
- Legacy script (deprecated): `scripts/release.sh`
