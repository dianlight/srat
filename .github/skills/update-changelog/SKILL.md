---
name: update-changelog
description: 'Read completed or updated tasks from docs/tasks/ and append structured entries to CHANGELOG.md under the [ 🚧 Unreleased ] section, following the existing changelog format. Triggers on: "update changelog", "add to changelog", "changelog from tasks", "generate release notes", "write changelog entry".'
argument-hint: 'Optional scope: "all done" (tasks marked ✅ Complete), "since <TaskID>" (tasks from NNN onwards), or a specific task filename/ID to add just one entry'
---

# Update Changelog

Reads completed or in-progress task documents from `docs/tasks/` and appends well-formatted entries to `CHANGELOG.md` under the `[ 🚧 Unreleased ]` section, following the project's existing changelog conventions.

## When to Use

- After completing a task or a batch of tasks and wanting to record the change
- Before cutting a release: ensure the changelog reflects all recent work
- When the user says "update changelog", "add changelog entry", "generate release notes"

---

## Procedure

### 1. Read CHANGELOG.md Format

Read `CHANGELOG.md` and extract:
- The `[ 🚧 Unreleased ]` section header and its subsections (`### ✨ Features`, `### 🐛 Bug Fixes`, `### 🔧 Maintenance`, `### 🧑‍🏫 Documentation`, `### 🔄 Breaking Changes`)
- The line immediately before the first versioned release heading (e.g., `## [ 2026.x.y ]`) — this is the **insertion point**

The format of each entry is a single bullet in the matching subsection:

```markdown
- **Short title**: Description of what was done and why it matters to the user.
```

### 2. Determine Which Tasks to Process

| Argument | Tasks to include |
|----------|-----------------|
| `all done` (default) | All task files where **Status = ✅ Complete** |
| `since NNN` | All task files with TaskID ≥ NNN, regardless of status |
| `NNN` or filename | Only the specified task file |
| _(none)_ | All task files with **Status = ✅ Complete** |

For tasks that are `🔄 In Progress`, only include if the user explicitly passes the task ID or `since NNN` scope.

### 3. Map Task Type → Changelog Section

| Task type | Changelog subsection |
|-----------|---------------------|
| `[FEATURE]` | `### ✨ Features` |
| `[FIX]` | `### 🐛 Bug Fixes` |
| `[DOCS]` | `### 🧑‍🏫 Documentation` |
| `[REFACTOR]` | `### 🔧 Maintenance` |

### 4. Draft the Changelog Entry

For each task, generate a changelog bullet using this structure:

```
- **<Title>**: <One to two sentence user-facing description derived from the task Objective paragraph.> <Optional: issue reference.>
```

Rules for the description:
- Write for an **end user**, not a developer — avoid internal implementation details (adapter names, function signatures, struct fields)
- Focus on **what changed and what benefit it brings**
- If the task has a GitHub issue link, append `([#NNN](url))` at the end
- Maximum 2 sentences; if the objective is long, distill the core user-visible change
- Do **not** include task list items verbatim — summarise

Example derivation:

> Task objective: _"Resolve Time Machine backup failures on macOS Tahoe..."_
> Changelog entry: `- **Time Machine Compatibility (macOS Tahoe)**: Improved Samba parameters for Apple `fruit:` extensions to prevent backup disconnections on macOS 15+. ([hassio-addons#536](https://github.com/dianlight/hassio-addons/issues/536))`

### 5. Check for Duplicates

Before inserting, scan the `[ 🚧 Unreleased ]` section for an existing bullet whose title text matches the task title (case-insensitive). If a match exists, **skip** the task and warn the user rather than creating a duplicate.

### 6. Insert into CHANGELOG.md

For each new entry:

1. Find the correct subsection header (e.g., `### ✨ Features`) inside `[ 🚧 Unreleased ]`.
   - If the subsection does not exist yet, create it immediately before the next `###` or the next `##` heading.
2. Append the bullet as the **last item** in that subsection (immediately before the next `###` or `##` heading).
3. Write the file.

Handle the case where `[ 🚧 Unreleased ]` does not exist: insert the full section at the top of the changelog body (after the `<!-- DOCTOC SKIP -->` comment and the `# Changelog` heading), before the first versioned release.

### 7. Confirm

After all insertions, report:
- How many entries were added
- Which subsections were updated
- Any tasks skipped due to duplicates or unrecognised type

---

## Entry Style Guide

Reference the existing CHANGELOG.md entries as style anchors:

| ✅ Good | ❌ Avoid |
|--------|---------|
| User-facing benefit language | Internal implementation details |
| Present tense ("Adds", "Fixes", "Improves") | Past tense imperative ("Added", "Fixed") |
| Bold short title followed by colon | Unformatted or all-lowercase bullet |
| Issue link in parentheses at end | Issue link mid-sentence |
| ≤ 2 sentences | Paragraph-length bullets |

### Subsection creation template

If a subsection needs to be created from scratch inside `[ 🚧 Unreleased ]`:

```markdown
### ✨ Features

### 🧑‍🏫 Documentation

### 🐛 Bug Fixes

### 🔄 Breaking Changes

### 🔧 Maintenance
```

Always create them in the above order (Features first, Maintenance last).

---

## Quality Checklist

- [ ] Only entries for `✅ Complete` tasks are added (unless the user explicitly requests otherwise)
- [ ] No duplicate entries (check before inserting)
- [ ] Entry text is user-facing, not implementation-facing
- [ ] Each entry is placed in the correct subsection for its task type
- [ ] Issue links are rendered as `([#NNN](url))` — not plain URLs
- [ ] The `[ 🚧 Unreleased ]` section is not created if it already exists
- [ ] File is written using `replace_string_in_file` (targeted insert, not full-file overwrite)
