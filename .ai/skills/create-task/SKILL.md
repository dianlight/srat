---
name: create-task
description: 'Create a new task planning document for SRAT. Use when: adding a feature, planning a fix, documenting a refactor, or creating a new TODO item. Produces a <TaskID>_<Title>.md file in docs/tasks/ using the project task template. Triggers on: "create task", "new task", "add task", "plan feature", "create todo", "new issue task", "document work item".'
argument-hint: 'Brief task title or description (e.g., "Add dark mode toggle to settings")'
---

# Create Task

Produces a structured task document in `docs/tasks/` using the [project task template](../../../docs/task-template.md).

## When to Use

- Planning a new feature, bug fix, refactor, or documentation item
- Capturing implementation notes and acceptance criteria before starting work
- If you need to ask questions to clarify the task details before creating the document, do so in the chat first, then use this skill once you have the necessary information.
- Converting a GitHub issue or request into a tracked work item
- Any time the user says "create task", "new task", "plan this", or "add a TODO"

## Procedure

### 1. Gather Information

Collect the following (ask the user for anything not provided):

| Field | Notes |
|-------|-------|
| **Type** | `FEATURE`, `FIX`, `DOCS`, or `REFACTOR` |
| **Title** | Short descriptive name (used in filename and heading) |
| **Objective** | One-paragraph purpose statement |
| **Inputs / Outputs** | What goes in, what comes out |
| **Dependencies** | Modules, APIs, external repos |
| **Implementation Notes** | Pseudo-code, key decisions, constraints |
| **Code References** | TODOs or FIXMEs in the codebase |

### 2. Determine the Next TaskID

Scan `docs/tasks/` for existing files matching the pattern `NNN_*.md`:

```bash
ls docs/tasks/*.md 2>/dev/null | grep -oP '^\d+' | sort -n | tail -1
```

- If no files exist, start at `001`.
- Otherwise increment the highest found number, zero-padded to three digits (e.g., `007` → `008`).

### 3. Build the Filename

Format: `<TaskID>_<kebab-case-title>.md`

Rules (same as git branch naming in copilot-instructions.md):
- Lowercase only
- Spaces and underscores → hyphens
- Strip emojis, special characters, common stop-words (`a`, `the`, `of`, `for`, `with`)

Examples:
- "Add Dark Mode Toggle" → `007_add-dark-mode-toggle.md`
- "Fix user login validation bug" → `008_fix-user-login-validation-bug.md`

### 4. Populate the Template

Use [task-template.md](../../../docs/task-template.md) as the base. Fill in:

```markdown
# [FEATURE/FIX/DOCS/REFACTOR]: <Title>

**Target Repo:** `srat`  
**Status:** 📅 Planned  
**Issue Link:** [Optional]

## 🎯 Objective
...

## 🛠️ Technical Specifications
- **Inputs:** ...
- **Outputs:** ...
- **Dependencies:** ...

## 📝 Task List
- [ ] Task 1: ...
- [ ] Task 2: ...
- [ ] Task 3: Unit testing
- [ ] Task 4: Integration and documentation
- [ ] Task 5: Accessibility (if applicable)
- [ ] Task 6: Code review and cleanup
- [ ] Task 7: Final testing and validation
- [ ] Task 8: Capture the lessons learned and update documentation
- [ ] Task 9: Ask to create a PR with the task implementation and link it here for tracking

## 🧠 Implementation Notes (Copilot Context)
...

## 🔗 Code References & TODOs
...
```

Leave any section as a placeholder (`_TBD_`) if the user hasn't provided enough detail — do not invent specifics.

### 5. Create the File

Write the populated content to `docs/tasks/<TaskID>_<title>.md`.

Create the `docs/tasks/` directory if it does not yet exist.

### 6. Confirm

Report the full relative path of the created file and summarise the task in one sentence.

Suggest to the user:
- A Git branch name following the convention in `copilot-instructions.md` (e.g., `feature/add-dark-mode-toggle`)
- Next steps: assign tasks, link to a GitHub issue, or start implementation

## Quality Checklist

- [ ] Filename matches `NNN_kebab-case-title.md` pattern
- [ ] File is in `docs/tasks/`
- [ ] Heading type matches `FEATURE | FIX | DOCS | REFACTOR`
- [ ] Status is `📅 Planned` for new tasks
- [ ] Task List contains at least one item and always includes testing + documentation tasks
- [ ] Implementation Notes section is present (even if minimal)
