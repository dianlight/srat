<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->

**Table of Contents** _generated with [DocToc](https://github.com/thlorenz/doctoc)_

- [Start Work on a Task](#start-work-on-a-task)
  - [When to Use](#when-to-use)
  - [Outcome](#outcome)
  - [Procedure](#procedure)
    - [0. Optional Strict Task-ID Mode](#0-optional-strict-task-id-mode)
    - [1. Resolve the Target Task](#1-resolve-the-target-task)
    - [2. Ensure GitHub Issue Link Exists (dianlight/srat)](#2-ensure-github-issue-link-exists-dianlightsrat)
    - [3. Branch Safety Gate](#3-branch-safety-gate)
    - [4. Plan Before Coding (No Implementation Yet)](#4-plan-before-coding-no-implementation-yet)
    - [5. Update the Task Document at Every Phase](#5-update-the-task-document-at-every-phase)
    - [6. Handoff to Implementation](#6-handoff-to-implementation)
  - [Completion Checks (Pre-implementation)](#completion-checks-pre-implementation)
  - [Example Prompts](#example-prompts)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

---

name: start-task-work
description: 'Start implementation from a docs/tasks/\*.md file with guardrails: verify/create linked GitHub issue in dianlight/srat, branch from main when needed, inspect code and produce a plan summary before coding, and keep the task doc updated at every phase. Supports strict mode to require a numeric task ID. Triggers on: "start task", "work on task", "begin task 012", "implement docs task", "start implementation from docs/tasks".'
argument-hint: 'Task identifier (preferred: task id like "012") + optional mode "strict-id" (rejects title-only or fuzzy matching)'

---

# Start Work on a Task

Safely starts work from a task document in `docs/tasks/` and prepares implementation without coding yet.

## When to Use

- User wants to begin work from an existing `docs/tasks/*.md` file
- User asks to implement a task by ID/title
- You need a consistent pre-implementation workflow (issue tracking + branch hygiene + planning)

## Outcome

Produces a ready-to-implement state with:

- A concrete task file selected and parsed
- A linked GitHub issue in `dianlight/srat` (created if missing)
- Branch decision made if currently on `main`
- Codebase inspection completed
- A written implementation summary approved by the user before coding
- Task document updated at each phase transition

---

## Procedure

### 0. Optional Strict Task-ID Mode

If the user enables `strict-id` mode (or asks for strict matching):

- Accept only numeric task IDs (for example `012`)
- Reject title-only/fuzzy references
- Require exact `docs/tasks/NNN_*.md` resolution before continuing

Use strict mode when users want deterministic automation and no accidental task selection.

### 1. Resolve the Target Task

1. Find the task file in `docs/tasks/`:

- Prefer exact ID match (`012` → `docs/tasks/012_*.md`)
- Else match by filename/title keywords

2. If ambiguous, ask the user to choose one file.
3. Read the full task document and extract:

- Type (`[FEATURE|FIX|DOCS|REFACTOR]`)
- `**Status:**`
- `**Issue Link:**`
- Task checklist under `## 📝 Task List`
- Constraints from `## 🧠 Implementation Notes`
- Scope from `## 🔗 Code References & TODOs`

### 2. Ensure GitHub Issue Link Exists (dianlight/srat)

1. Check `**Issue Link:**` in the task header.
2. If empty, `_TBD_`, `N/A`, or points to a non-`dianlight/srat` issue:

- Search `dianlight/srat` issues by task title/keywords.
- If a clearly related issue exists, write it into `**Issue Link:**`.
- If no related issue exists, ask user confirmation before creating one in `dianlight/srat`, then write its URL into `**Issue Link:**`.

3. Issue creation defaults:

- Title: task heading text without `[TYPE]:`
- Body: objective + technical specification summary + checklist preview
- Labels:
  - `FEATURE` / `REFACTOR` → `enhancement`
  - `FIX` → `bug`
  - `DOCS` → `documentation`

### 3. Branch Safety Gate

1. Detect current branch.
2. If branch is `main` and the task document does not explicitly require main-branch work:

- Ask user whether to create a feature branch before implementation.
- Suggest branch naming from task title:
  - `feature/<kebab>` for FEATURE
  - `fix/<kebab>` for FIX
  - `docs/<kebab>` for DOCS
  - `refactor/<kebab>` for REFACTOR

3. If user agrees, create/switch to the branch before continuing.

### 4. Plan Before Coding (No Implementation Yet)

1. Inspect referenced code areas from `## 🔗 Code References & TODOs`.
2. Inspect the current codebase to assess what's already implemented and what needs to be done.
3. Identify any additional impacted files/components based on the task description and technical specifications.
4. Check for existing TODOs or FIXMEs in the code that relate to the task.
5. Produce a concise pre-implementation summary that includes:

- Root objective and acceptance criteria
- Impacted files/components
- Proposed step-by-step plan
- Test/validation strategy
- Risks and edge cases

6. Present this summary to the user and wait for approval before writing code.
7. Update `## 🧠 Implementation Notes` with the agreed plan and any constraints or blockers.

### 5. Update the Task Document at Every Phase

After each phase transition, update the task file immediately.

Required updates by phase:

- **Issue linked/created**:
  - Update `**Issue Link:**`
- **Work started**:
  - Set `**Status:**` to `🔄 In Progress` if previously planned
- **Planning completed**:
  - Add a short bullet list under `## 🧠 Implementation Notes` describing the agreed plan
- **Implementation/test/documentation progress**:
  - Check/uncheck items in `## 📝 Task List`
  - Keep `## 🔗 Code References & TODOs` aligned with discovered files/TODOs

Update existing task fields only (`Status`, `Issue Link`, checklist, references, implementation notes). Do not create a separate timestamped progress-log section unless the user explicitly asks.

If a phase cannot proceed, write a short blocker note in `## 🧠 Implementation Notes`.

### 6. Handoff to Implementation

Only after user confirms the summary:

1. Start coding per plan.
2. Continue updating the task document after each meaningful milestone.
3. Keep issue comments/status in sync as progress changes.

---

## Completion Checks (Pre-implementation)

- [ ] Exactly one target task file is selected
- [ ] `**Issue Link:**` is a valid `https://github.com/dianlight/srat/issues/<number>` URL
- [ ] Branch strategy has been confirmed when starting from `main`
- [ ] Planning summary has been presented and acknowledged
- [ ] Task doc has been updated for every completed phase so far

## Example Prompts

- `start task 012`
- `start task 012 strict-id`
- `begin work on docs/tasks/009_smb-ip-address-interface-resolution.md`
- `start implementation for task 010`
- `work on task remove sse code`
