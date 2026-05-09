---
name: implement-task-phase
description: 'Execute one checklist phase from a docs/tasks/*.md task with strict sequencing: pick exactly one unchecked item, implement only that scope, run targeted validation, and update task progress immediately. Triggers on: "implement next task item", "do task phase", "work on next checklist item", "execute phase 3 of task 012".'
argument-hint: 'Task identifier + optional phase selector (item number, exact checklist text, or "next")'
---

# Implement a Single Task Phase

Implements one checklist item at a time from `docs/tasks/*.md` with verification gates and immediate task-doc updates.

## When to Use

- User wants controlled, incremental execution instead of full-task implementation
- You need safer progress with frequent tests and clear checkpoints
- Task scope is large and should be split into small, verifiable steps

## Outcome

- Exactly one checklist item is completed (or explicitly blocked)
- Related code/tests/docs are updated only for that item
- Validation is run and summarized
- The task document is updated before moving to the next item

---

## Procedure

### 1. Resolve Task and Select Exactly One Item

1. Resolve task file in `docs/tasks/` by ID or filename.
2. Read `## 📝 Task List` and find unchecked items.
3. Selection rules:
  - If user names an item, use it.
  - If user says `next`, pick the first unchecked item.
  - If ambiguous, ask user to choose one item.
4. Confirm item boundaries (what is included/excluded) before editing code.

### 2. Inspect Scope Before Editing

1. Read referenced files from `## 🔗 Code References & TODOs` related to the selected item.
2. Inspect nearby dependencies likely to be impacted.
3. Define a tiny implementation plan for this one item only.

### 3. Implement Only the Selected Item

1. Apply minimal code changes for the selected checklist item.
2. Avoid unrelated refactors.
3. If additional work is discovered, append a new unchecked task item instead of silently expanding scope.

### 4. Run Targeted Validation Gate

Run the smallest relevant checks first, then broaden only if needed.

Suggested verification order:

- Backend changes: targeted `mise run //backend:test` package(s)
- Frontend changes: targeted `mise run //frontend:test` file(s)
- Custom component changes: targeted `mise run //custom_components:test` test(s)
- Doc-only changes: docs validation checks as needed

If tests fail:

1. Fix within selected-item scope.
2. Re-run targeted checks.
3. If blocked, stop and record blocker in task doc.

### 5. Update Task Document Immediately

After finishing this phase:

- Mark the checklist item `[x]` when done
- Keep `**Status:**` at `🔄 In Progress` until all checklist items are complete
- Add/update brief notes in `## 🧠 Implementation Notes`
- Update `## 🔗 Code References & TODOs` with any newly discovered touched files

If blocked:

- Keep item unchecked
- Add blocker note under `## 🧠 Implementation Notes`
- Add a new checklist item for unblock action if needed

### 6. Report and Wait for Next Phase Instruction

Provide:

- Item completed (or blocked)
- Files changed
- Validation run and result
- Recommended next unchecked item

Do not automatically continue to another checklist item unless user requests it.

---

## Completion Checks

- [ ] Exactly one checklist item was targeted
- [ ] Code changes stayed within selected item scope
- [ ] Relevant validation was run and reported
- [ ] Task document was updated before handoff
- [ ] Any blockers are explicitly recorded

## Example Prompts

- `implement next task item for task 012`
- `do phase 4 from task 009`
- `work on checklist item "remove SSE route" in task 012`
