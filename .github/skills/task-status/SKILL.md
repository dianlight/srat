---
name: task-status
description: 'Generate a Markdown progress report of all tasks in docs/tasks/ grouped by status (📅 Planned / 🔄 In Progress / ✅ Done). Optionally filter by type (FEATURE/FIX/DOCS/REFACTOR) or linked repo. Output is suitable for a weekly standup, GitHub comment, or PR description. Triggers on: "task status", "show task progress", "what tasks are done", "weekly standup", "task report", "progress report", "list tasks".'
argument-hint: 'Optional filter: "all" (default), "done", "in-progress", "planned", "features", "fixes", or a GitHub repo name ("srat" / "hassio-addons")'
---

# Task Status Report

Reads all task documents in `docs/tasks/` and generates a grouped progress summary.

## When to Use

- Weekly standup or progress summary
- Before a release: check which tasks are complete and which are pending
- Quick overview of what is planned vs. in flight vs. done
- When the user says "show task status", "what tasks are done", "progress report", "list all tasks"

---

## Procedure

### 1. Scan Task Files

Read all files in `docs/tasks/` matching `NNN_*.md` (exclude `README.md`).

For each file, extract:

| Field | Where to find it |
|-------|-----------------|
| **TaskID** | Leading digits of the filename (`001`, `002`, …) |
| **Type** | First heading: `# [FEATURE]`, `# [FIX]`, `# [DOCS]`, `# [REFACTOR]` |
| **Title** | Text after the `[TYPE]:` prefix in the first heading |
| **Status** | `**Status:**` field in the header line (must match one of: `📅 Planned`, `🔄 In Progress`, `✅ Done`) |
| **Issue Links** | All `[repo#NNN](url)` patterns in the file |
| **Progress** | Count `- [x]` vs `- [ ]` items under `## 📝 Task List` |

If `**Status:**` is missing or unrecognised, infer from task list:
- 0 checked → `📅 Planned`
- 1+ checked, not all → `🔄 In Progress`
- All checked → `✅ Done`

### 2. Apply Filter

| Argument | Behaviour |
|----------|-----------|
| `all` (default) | Include all tasks regardless of status or type |
| `done` | Only `✅ Done` tasks |
| `in-progress` | Only `🔄 In Progress` tasks |
| `planned` | Only `📅 Planned` tasks |
| `features` | Only `[FEATURE]` type |
| `fixes` | Only `[FIX]` type |
| `srat` | Only tasks with an issue link containing `dianlight/srat` |
| `hassio-addons` | Only tasks with an issue link containing `dianlight/hassio-addons` |

### 3. Build the Report

Output a Markdown document with the following structure:

```markdown
# 📊 SRAT Task Status Report
_Generated: <date>_

## Summary
| Status | Count | Progress |
|--------|-------|----------|
| ✅ Done | N | N/T tasks |
| 🔄 In Progress | N | N/T tasks |
| 📅 Planned | N | N/T tasks |
| **Total** | **T** | **overall %** |

---

## ✅ Done

### [001] Title
- **Type:** FEATURE | **Issues:** [srat#185](url)
- **Progress:** 5 / 5 tasks ✓

---

## 🔄 In Progress

### [003] Title
- **Type:** FIX | **Issues:** [hassio-addons#596](url)
- **Progress:** 2 / 12 tasks (17%)
- **Next:** Task 3: ...  _(first unchecked item)_

---

## 📅 Planned

### [008] Title
- **Type:** FEATURE | **Issues:** [srat#184](url)
- **Progress:** 0 / 10 tasks

...
```

Within each status group, sort tasks by TaskID ascending.

For each in-progress task, include the **first unchecked task item** as "Next:".

### 4. Output

Print the report as a code block in Markdown so the user can copy-paste it directly into:
- A GitHub issue comment
- A PR description
- A team standup note

Also state the total number of tasks processed and the overall completion percentage.

---

## Quality Checklist

- [ ] Every file in `docs/tasks/NNN_*.md` is represented (no silently skipped files)
- [ ] Status inference fallback is used when `**Status:**` field is missing
- [ ] Progress percentages are rounded to nearest integer
- [ ] "Next:" field is only shown for in-progress tasks (has at least one checked AND one unchecked item)
- [ ] Issue links are rendered as clickable Markdown `[label](url)` — never raw URLs
- [ ] Report includes generation date
