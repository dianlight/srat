---
name: close-task-work
description: 'Close out a completed docs/tasks/*.md task by validating checklist completion, syncing GitHub issue status/comments, and preparing optional PR summary notes. Triggers on: "close task", "finalize task", "wrap up task 012", "complete docs task".'
argument-hint: 'Task identifier (required) + optional publish mode ("issue-only", "issue-and-pr")'
---

# Close a Task Workstream

Finalizes a task from `docs/tasks/` after implementation is complete.

## When to Use

- All implementation checklist items are done
- You want clean closure in task docs and GitHub issue tracking
- You need a concise completion summary for issue/PR communication

## Outcome

- Task status is set to done when appropriate
- Linked issue receives a completion update
- Linked issue is closed (or closure is prepared) when criteria are met
- Optional PR-ready completion summary is produced

---

## Procedure

### 1. Resolve and Validate Task Readiness

1. Resolve exactly one task file by ID/filename.
2. Verify checklist state in `## 📝 Task List`:
  - If any unchecked items remain, do not close task.
  - Report remaining items and keep status in progress.
3. Verify `**Issue Link:**`:
  - Must point to `https://github.com/dianlight/srat/issues/<number>`
  - If missing, ask whether to create/link issue before closing flow

### 2. Final Task Document Updates

If all checklist items are complete:

1. Set `**Status:**` to `✅ Done`
2. Ensure `## 🧠 Implementation Notes` contains a concise completion summary:
  - what changed
  - what was validated
  - notable follow-ups (if any)
3. Ensure `## 🔗 Code References & TODOs` reflects final touched scope

### 3. Sync GitHub Issue

1. Post completion comment on linked issue with:
  - task file reference
  - summary of completed work
  - validation/tests summary
2. If user confirms closure (or mode is explicitly close-enabled), close the issue.
3. If issue cannot be closed (policy/review constraints), leave it open and post status note.

### 4. Optional PR Summary Generation

If requested (`issue-and-pr` mode):

Generate a concise PR-ready markdown summary including:

- Task title and ID
- Problem statement
- Key changes grouped by area
- Validation performed
- Remaining follow-ups

### 5. Final Report Back to User

Provide:

- Task final status
- Issue sync result (commented/closed/open)
- Any unresolved follow-ups

---

## Guardrails

- Never mark task done while checklist still has unchecked items
- Never close GitHub issue without explicit user confirmation unless user requested an auto-close mode
- Keep closure summaries factual and based on executed validation

## Completion Checks

- [ ] Task checklist fully complete
- [ ] Task status updated to `✅ Done`
- [ ] Issue comment posted with completion summary
- [ ] Issue closure handled per user choice/policy
- [ ] Optional PR summary generated when requested

## Example Prompts

- `close task 012`
- `finalize docs/tasks/010_ui-configuration-wizard-guided-tour.md`
- `wrap up task 009 with issue-and-pr summary`
