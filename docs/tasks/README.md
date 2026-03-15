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

## Suggested Workflow

1. Create planning docs with `/create-task`.
2. Keep tasks aligned with GitHub using `/sync-tasks`.
3. Generate standup/progress summaries via `/task-status`.
4. Publish user-facing release notes with `/update-changelog`.
