<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->

**Table of Contents** _generated with [DocToc](https://github.com/thlorenz/doctoc)_

- [Update Changelog with Commits](#update-changelog-with-commits)
  - [When to Use](#when-to-use)
  - [Procedure](#procedure)
    - [1. Read CHANGELOG.md Format](#1-read-changelogmd-format)
    - [2. Determine Last Release Tag](#2-determine-last-release-tag)
    - [3. Collect Completed Tasks](#3-collect-completed-tasks)
    - [4. Collect Commits Since Last Release](#4-collect-commits-since-last-release)
    - [5. Select Top N Commits (Default: 5)](#5-select-top-n-commits-default-5)
    - [6. Map Sources to Changelog Subsections](#6-map-sources-to-changelog-subsections)
    - [7. Draft Changelog Entries](#7-draft-changelog-entries)
    - [8. Check for Duplicates](#8-check-for-duplicates)
    - [9. Create Missing Subsections (if needed)](#9-create-missing-subsections-if-needed)
    - [10. Insert Entries](#10-insert-entries)
    - [11. Confirm](#11-confirm)
  - [Entry Style Guide](#entry-style-guide)
  - [Quality Checklist](#quality-checklist)
  - [Examples](#examples)
  - [Notes](#notes)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

---

name: update-changelog-with-commits
description: 'Update CHANGELOG.md from completed tasks AND important commits since last release. Combines task-based entries with commit-based entries, prioritizing by lines changed. Triggers on: "update changelog with commits", "changelog from tasks and commits", "generate release notes with git".'
argument-hint: 'Optional: "top N" (default 5) to limit commit entries by lines changed; task selection defaults to all ✅ Complete tasks'

---

# Update Changelog with Commits

Combines completed task documents (`docs/tasks/*.md`) with git commits since the last release tag to produce a comprehensive changelog entry under the `[ 🚧 Unreleased ]` section. Prioritizes commit entries by number of lines changed (top N) and merges them with task-based entries.

## When to Use

- When preparing for a release and need to capture both planned tasks and actual commits
- To ensure important commits that aren't tracked as tasks are included in the changelog
- When you want a data-driven selection of commits (by lines changed) rather than manual curation

---

## Procedure

### 1. Read CHANGELOG.md Format

Read `CHANGELOG.md` and identify:

- The `[ 🚧 Unreleased ]` section and its existing subsections
- The insertion point (before the first `## [ ... ]` versioned release heading)
- Existing bullet entries to avoid duplicates

### 2. Determine Last Release Tag

Run: `git describe --tags --abbrev=0` to get the most recent release tag.

- This defines the commit range: `last_tag..HEAD`
- If no tags exist, use the first commit (`--first-parent` traversal)

### 3. Collect Completed Tasks

Read all task files from `docs/tasks/*.md` and filter:

- **Status = ✅ Complete** (only completed tasks)
- Extract: TaskID, Title, Type ([FEATURE]/[FIX]/[DOCS]/[REFACTOR]), Objective, GitHub issue links

### 4. Collect Commits Since Last Release

Run: `git log --pretty=format:"%H %s" --first-parent last_tag..HEAD`
For each commit:

- Get the commit hash and subject line
- Calculate lines changed: `git show --stat --oneline <hash>` and parse the `insertions(+)/deletions(-)` total
- Keep all commits with their line change count

### 5. Select Top N Commits (Default: 5)

Sort commits by lines changed (descending) and take the top N (configurable via argument `top N`).

- If fewer than N commits exist, take all
- These commits will be converted to changelog entries

### 6. Map Sources to Changelog Subsections

**For tasks:**
| Task type | Subsection |
|-----------|------------|
| `[FEATURE]` | `### ✨ Features` |
| `[FIX]` | `### 🐛 Bug Fixes` |
| `[DOCS]` | `### 🧑‍🏫 Documentation` |
| `[REFACTOR]` | `### 🔧 Maintenance` |

**For commits:** No automatic mapping. Use the commit subject to infer:

- If subject starts with `feat:` or similar feature keywords → `### ✨ Features`
- If subject starts with `fix:` or contains "bug", "error", "issue" → `### 🐛 Bug Fixes`
- If subject contains "refactor", "cleanup", "reorganize" → `### 🔧 Maintenance`
- If subject contains "doc", "documentation", "readme" → `### 🧑‍🏫 Documentation`
- If subject contains "break", "breaking", "incompatible" → `### 🔄 Breaking Changes`
- Otherwise → `### 🏗 Chore` (or ask user to classify)

### 7. Draft Changelog Entries

**Task entry format:**

```
- **<Title>**: <One to two sentence user-facing description derived from Objective.> <Optional: issue link>
```

- Write for end users, not developers
- Distill the core user-visible change
- Append issue link if present: `([#NNN](url))`

**Commit entry format:**

```
- <Commit subject line, cleaned up>
```

- Remove conventional commit prefixes (`feat:`, `fix:`, `chore:`) if present
- Capitalize first letter if needed
- Keep it concise; optionally add brief context if subject is cryptic
- Do NOT include commit hash in the entry
- If the commit references an issue, append `([#NNN](url))` if detectable

### 8. Check for Duplicates

Before inserting, scan existing bullets in `[ 🚧 Unreleased ]`:

- For task entries: compare title (case-insensitive)
- For commit entries: compare cleaned subject (case-insensitive)
- Skip any duplicates and report them

### 9. Create Missing Subsections (if needed)

If a required subsection does not exist, create it in this order inside `[ 🚧 Unreleased ]`:

```markdown
### ✨ Features

### 🐛 Bug Fixes

### 🔄 Breaking Changes

### 🔧 Maintenance

### 🧑‍🏫 Documentation

### 🏗 Chore
```

Place new subsections immediately before the next `###` or `##` heading, maintaining the canonical order.

### 10. Insert Entries

For each selected entry (tasks first, then commits, or interleaved by subsection):

1. Locate the correct subsection header
2. Append the bullet as the last item in that section (before the next heading)
3. Use `replace_string_in_file` for targeted insertion

### 11. Confirm

Report:

- Number of task entries added
- Number of commit entries added
- Subsections updated or created
- Any entries skipped due to duplicates
- The commit range used (last_tag..HEAD)

---

## Entry Style Guide

Follow the existing CHANGELOG.md style:

| ✅ Good                               | ❌ Avoid                             |
| ------------------------------------- | ------------------------------------ |
| User-facing benefit language          | Internal implementation details      |
| Bold title for task entries           | Unformatted bullets for tasks        |
| Cleaned commit subjects (no prefixes) | Raw commit hashes or full commit IDs |
| Issue links in parentheses at end     | Issue links mid-sentence             |
| ≤ 2 sentences for task entries        | Paragraph-length bullets             |

---

## Quality Checklist

- [ ] Only ✅ Complete tasks are included (unless user explicitly overrides)
- [ ] Only top N commits by lines changed are included
- [ ] No duplicate entries (checked against existing Unreleased section)
- [ ] Missing subsections are created in canonical order
- [ ] Task entries are user-facing and summarized from Objective
- [ ] Commit entries are cleaned (no `feat:` prefixes, no hashes)
- [ ] File is written using `replace_string_in_file` (targeted edits)
- [ ] Last release tag is correctly identified via `git describe --tags --abbrev=0`

---

## Examples

**Task entry:**

```
- **HACS Custom Component**: Added a Home Assistant custom component compatible with HACS, supporting UI configuration wizard, Supervisor add-on autodiscovery, and WebSocket-based real-time updates.
```

**Commit entry:**

```
- Improve SMB over QUIC fallback handling for edge cases
```

**Combined output:**

```markdown
### ✨ Features

- **HACS Custom Component**: Added a Home Assistant custom component compatible with HACS...
- Add support for SMB over QUIC with automatic fallback

### 🐛 Bug Fixes

- Fix mount point type defaulting causing database constraint violations
```

---

## Notes

- This skill is **workspace-scoped** for the SRAT repository; it relies on the presence of `docs/tasks/` and the specific CHANGELOG.md format.
- The "top N commits" selection is deterministic based on `git show --stat` line counts.
- If the last release tag cannot be found (no tags), the skill will abort with an error message.
