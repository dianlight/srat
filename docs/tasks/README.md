# Tasks

This directory contains structured task planning documents for the SRAT project.

## Naming Convention

Files follow the pattern `<TaskID>_<kebab-case-title>.md`, where `TaskID` is a zero-padded three-digit integer (e.g., `001`, `002`).

## Creating a New Task

Use the `/create-task` skill in GitHub Copilot Chat, or manually copy [../task-template.md](../task-template.md) and name the file following the convention above.

## Related Skills

Use these companion skills to manage the task lifecycle:

- **`/sync-tasks`**
	- Syncs `docs/tasks/*.md` with GitHub issues.
	- **Import mode**: pulls open issues from `dianlight/srat` and SambaNAS2-related issues from `dianlight/hassio-addons`, then updates matching tasks or creates new task files.
	- **Export mode**: posts progress comments, closes issues for completed tasks, and creates missing issues for planned work.
	- Example prompts: `sync tasks`, `import issues`, `export task status`.

- **`/task-status`**
	- Generates a Markdown progress report grouped by status (`📅 Planned`, `🔄 In Progress`, `✅ Done`).
	- Supports filters by status, type, or repo.
	- Example prompts: `task status`, `weekly standup`, `show task progress`.

- **`/update-changelog`**
	- Reads completed (or scoped) task files and appends structured entries to `CHANGELOG.md` under `## [ 🚧 Unreleased ]`.
	- Maps task type to the correct changelog section (`FEATURE` → Features, `FIX` → Bug Fixes, etc.) and avoids duplicate entries.
	- Example prompts: `update changelog`, `add changelog entry`, `generate release notes`.

- **`/start-task-work`**
	- Starts implementation from an existing task file with guardrails.
	- Ensures `Issue Link` is present (reuses/creates `dianlight/srat` issue), confirms branch strategy when starting on `main`, inspects code, and proposes a plan summary before coding.
	- Supports optional strict mode (`strict-id`) to require exact numeric task ID matching.
	- Keeps the task document updated at every phase transition.
	- Example prompts: `start task 012`, `begin work on docs/tasks/009_*.md`, `start implementation from task`.

- **`/implement-task-phase`**
	- Executes exactly one checklist item from a task at a time.
	- Enforces incremental changes, targeted validation, and immediate task-doc updates before moving on.
	- Example prompts: `implement next task item for task 012`, `do phase 4 from task 009`.

- **`/close-task-work`**
	- Finalizes completed task workstreams.
	- Verifies checklist completion, updates task status to done, syncs completion to linked issue, and optionally prepares PR-ready summary text.
	- Example prompts: `close task 012`, `wrap up task 009 with issue-and-pr summary`.

## Suggested Workflow

1. Create planning docs with `/create-task`.
2. Start safely from a chosen task using `/start-task-work`.
3. Implement incrementally using `/implement-task-phase`.
4. Keep tasks aligned with GitHub using `/sync-tasks`.
5. Generate standup/progress summaries via `/task-status`.
6. Close completed workstreams with `/close-task-work`.
7. Publish user-facing release notes with `/update-changelog`.

## Tiny Cheat Sheet

Copy-paste prompts for the common lifecycle:

- `create task Add SMB share quota warnings`
- `start task 012 strict-id`
- `implement next task item for task 012`
- `sync tasks export all`
- `task status`
- `close task 012`
- `update changelog`
