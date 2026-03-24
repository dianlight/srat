---
name: create-pr
category: workflow
scope: workspace
version: 1.0
description: >
  Automate creation or update of a pull request from the current branch to main, with detailed PR body, task/issue linking, and user prompts for draft/auto-merge. Fallback to commit summary if no task is found.
---

# Create or Update Pull Request Skill

## Purpose
Automate the process of creating or updating a pull request from the current branch to `main` in the current repository, following SRAT conventions.

## Workflow Steps

1. **Detect Current Branch**
   - If already on `main`, abort (cannot PR main → main).
2. **Check for Related Task**
   - Search for a related task in `docs/tasks/` matching the branch name or recent commits.
   - If found, extract task details (title, description, checklist, linked issues).
   - If not found, summarize recent commit messages for PR body.
3. **Find Issues to Close**
   - From the related task (if any), collect all open issues that this PR will close.
   - Also scan commit messages for `Closes #<issue>` or similar patterns.
4. **Prompt User**
   - Ask if the PR should be a draft.
   - If not a draft, ask if it should be set to auto-merge.
5. **Create or Update PR**
   - Use `mcp_io_github_git_add_comment_to_pending_review` to add a comment if updating, or create a new PR if none exists.
   - PR body must include:
     - Task details (if any), or commit summary
     - List of issues to close (with GitHub auto-closing syntax)
   - Always use `main` as the base branch.
6. **Post-creation**
   - If auto-merge requested, enable auto-merge for the PR.

## Quality Criteria
- PR body is clear, detailed, and references all relevant tasks/issues.
- No PR is created from `main` to `main`.
- User is prompted for draft/auto-merge status.
- Fallback to commit summary if no task is found.

## Example Prompts
- "Create a PR for my current branch"
- "Update the PR for this feature branch"
- "Open a draft PR for my bugfix branch"

## Related Customizations
- `sync-tasks` skill (for task/issue linking)
- `update-changelog` skill (for changelog automation)

---
