---
name: sync-tasks
description: 'Sync docs/tasks/*.md planning documents with GitHub issues. TWO modes: (1) IMPORT — pull open GitHub issues from dianlight/srat and dianlight/hassio-addons (SambaNAS2 only) into the task files: map to existing tasks or create new ones, always updating issue links. (2) EXPORT — read completed or updated tasks and push status back to GitHub: comment on linked issues with progress, close issues when all task items are checked, create issues for tasks that have none. Triggers on: "sync tasks", "sync github", "import issues", "export task status", "update tasks from issues", "push task progress", "close issue from task".'
argument-hint: 'Mode and optional scope: "import [from srat|hassio-addons]", "export [all|done]", or "both" (default)'
---

# Sync Tasks ↔ GitHub Issues

Bidirectional synchronisation between task planning documents in `docs/tasks/` and GitHub issues in `dianlight/srat` and `dianlight/hassio-addons`.

## When to Use

- After a session where you created or updated task documents and want to reflect them on GitHub
- When new GitHub issues have been filed and you want to pull them into the task planning system
- When completing a task in `docs/tasks/` and wanting to close or comment on the linked GitHub issue
- Any time the user says "sync tasks", "import issues", "push task status", or "update github from tasks"

---

## Mode A — IMPORT (GitHub Issues → Task Docs)

Pull open issues from GitHub into the local task planning system.

### A-1. Fetch Open Issues

Use the GitHub MCP tools to fetch issues from each repo:

```
dianlight/srat         → state: OPEN, perPage: 100
dianlight/hassio-addons → state: OPEN, perPage: 100
```

For `hassio-addons`, **filter to SambaNAS2-related issues only**: keep issues whose title or labels contain `SambaNas2`, `SambaNAS2`, or `SambaNAS`.

Skip bot/automation issues: filter out issues authored by `renovate` or `github-actions[bot]` and issues titled "Dependency Dashboard".

### A-2. Read Existing Tasks

Scan `docs/tasks/*.md` (excluding `README.md`). For each file, extract:
- **TaskID** (the leading `NNN` from the filename)
- **Title** (first `# [TYPE]:` heading line)
- **Linked issues** (all `[repo#NNN]` patterns in the document)
- **Status** (`📅 Planned` / `🔄 In Progress` / `✅ Done`)

Build an index: `Set<"repo#number">` of already-linked issues.

### A-3. Classify Each Issue

For every fetched issue, determine:

| Condition | Action |
|-----------|--------|
| Issue number already in the index | Skip (already tracked) |
| Issue topic clearly matches an existing task (keyword overlap in title/body) | **UPDATE** the matching task |
| No matching task | **CREATE** a new task |

**Keyword matching heuristic**: compare issue title words against existing task titles + objective paragraphs. A match requires ≥ 2 significant words in common (ignore stop-words: a, an, the, is, to, in, for, of, and, or, not).

For ambiguous cases (score tie or multiple candidates), pick the task whose `Code References` section already contains the most filename overlap with the issue body.

### A-4. Update Existing Task

When an existing task is a match, **open the task file** and:

1. Add the issue link to the header `**Issue Link:**` field if not already present.
2. Append a new task item to the `## 📝 Task List` section describing the issue's request.
3. Append the issue reference to the `## 🔗 Code References & TODOs` section.

Format for the issue reference line:

```markdown
- [ ] [repo#NNN](https://github.com/dianlight/REPO/issues/NNN) — <one-line issue summary>
```

### A-5. Create New Task

When no existing task matches, create a new task file using the same procedure as the `create-task` skill:

1. Determine next TaskID (scan `docs/tasks/*.md`, increment highest).
2. Build filename: `NNN_kebab-title.md`.
3. Populate the task template — set `**Issue Link:**` to the GitHub issue URL.
4. Classify type: use `FIX` for issues with `bug` label, `FEATURE` for `enhancement` / `RoadMap`, `DOCS` otherwise.

---

## Mode B — EXPORT (Task Docs → GitHub Issues)

Push local task progress back to GitHub.

### B-1. Read All Tasks with Issue Links

Scan `docs/tasks/*.md`. For each task, extract:
- All linked issue references (`[repo#NNN]` patterns)
- The **completion ratio**: `checked_items / total_items` from `## 📝 Task List`
- The **status** header field

### B-2. Determine What to Push

| Condition | Action |
|-----------|--------|
| All task items checked  AND  status is `✅ Done` | **Close** the linked GitHub issue(s) with a summary comment |
| ≥ 1 task item checked  AND  status is `🔄 In Progress` | **Post a progress comment** on linked issue(s) |
| Task has no issue link  AND  status is NOT `✅ Done` | **Create a new GitHub issue** for the task |
| Task already closed on GitHub | Skip |

### B-3. Post Progress Comment

For in-progress tasks, post a comment using `mcp_github2_add_issue_comment`:

```markdown
## 🔄 Task Progress Update

Tracked in [`docs/tasks/NNN_title.md`](https://github.com/dianlight/srat/blob/main/docs/tasks/NNN_title.md)

**Progress:** X / Y tasks completed

<details><summary>Task list</summary>

- [x] Task 1: ...
- [ ] Task 2: ...

</details>

_Auto-generated by the `sync-tasks` skill._
```

Only post if the task file was modified since the last sync (avoid duplicate comments). Track last-sync state in a local scratch variable during the session — do not persist a file for this.

### B-4. Close Issue with Summary

For completed tasks, post a closing comment then close the issue:

```markdown
## ✅ Task Completed

This issue has been resolved. Implementation tracked in
[`docs/tasks/NNN_title.md`](https://github.com/dianlight/srat/blob/main/docs/tasks/NNN_title.md).

_Auto-generated by the `sync-tasks` skill._
```

Use `mcp_github2_issue_write` with `state: "CLOSED"` to close the issue.

### B-5. Create Missing Issue

For tasks with no linked issue but implementation planned:

1. Use `mcp_github2_issue_write` to create a new issue in `dianlight/srat`.
2. Title: same as the task doc `# [TYPE]: <Title>` (strip the `[TYPE]:` prefix).
3. Body: paste the `## 🎯 Objective` section.
4. Labels: `enhancement` for FEATURE, `bug` for FIX, `documentation` for DOCS.
5. After creation, update the task file `**Issue Link:**` field with the new issue URL.

---

## Mode C — BOTH (default)

Run Mode A first, then Mode B. This ensures any newly-imported issues are not immediately re-exported.

---

## Repos Reference

| Repo | Issues scope | Label filter |
|------|-------------|-------------|
| `dianlight/srat` | All open issues | _(none — all labels)_ |
| `dianlight/hassio-addons` | SambaNAS2 only | Title/label contains `SambaNas2` or `SambaNAS` |

Skip: `renovate`-authored issues, `Dependency Dashboard` title, closed issues.

---

## Quality Checklist

- [ ] All newly-linked issues appear in the relevant task `🔗 Code References` section
- [ ] No duplicate issue links (check before appending)
- [ ] New task files follow `NNN_kebab-title.md` naming and include `**Issue Link:**` in header
- [ ] Export comments include a link back to the task file in `docs/tasks/`
- [ ] Issues created via B-5 have their URL written back to the task file
- [ ] Renovate / bot issues are never imported or commented on
