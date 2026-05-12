---
name: capture-lessons-learned
description: 'Review the current chat session or recently completed task for reusable implementation or testing lessons, then decide whether to update an existing GitHub instruction and skills, create a new instruction/skill, or store the guidance as repo memory. Use when: finishing a fix, stabilizing a test, capturing a lesson learned, or asking "should we document this for future Copilot sessions?" Triggers on: "lesson learned", "capture lessons learned", "review session for reusable pattern", "update github instruction", "document this fix for future".'
argument-hint: 'Optional scope or recent work summary (for example: "review current session" or "frontend MSW test fix")'
---

# Capture Lessons Learned

Turn a one-off implementation or test discovery into reusable project guidance.

> Self-Evolving Skill: This skill improves through use. If instructions are wrong, parameters drifted, or a workaround was needed — fix this file immediately, don't defer. Only update for real, reproducible issues.

## When to Use

Use this skill when:

- a bug fix revealed a repo-specific root cause or pitfall
- a test needed a special setup or helper to stay deterministic
- a command, pattern, or verification step should become standard practice
- the user asks whether the current session produced guidance worth documenting
- you are wrapping up a task and want to prevent the same mistake from recurring

## Outcome

By the end of this workflow, you should produce **one** of these outcomes:

1. **No new guidance needed** — the lesson is too narrow, temporary, or already documented
2. **Propose an instruction update** — preferred when the new lesson fits an existing `.github/instructions/*.md`, `.github/copilot-instructions.md`, or `AGENTS.md`
3. **Propose a new instruction** — only when the lesson is stable, reusable, and has no good existing home
4. **Propose a new skill** — when the lesson is best expressed as a multi-step workflow rather than a single rule
5. **Store as repo memory only** — when the lesson is useful but still too task-specific or newly verified

> Never edit or create an instruction file automatically without asking the user first.

---

## Procedure

### 1. Re-read the Current Session for Evidence

Review the current chat session, changed files, and validation results. Start from the beginning or from the last use of this skill and extract any candidate lesson in one sentence:

- **Implementation pattern** — e.g. a canonical API hook, lifecycle wiring, or config source of truth
- **Testing pattern** — e.g. a shared MSW override helper, stable mock shape, or required test command
- **Pitfall to avoid** — e.g. a runtime error caused by an indirect access pattern
- **Verified command** — e.g. the exact build/lint/test command that proved the fix

> Do not generalize a hunch. Only consider lessons backed by code or fresh verification output.

### 2. Check Whether It Is Already Documented

Before proposing a new rule, inspect:

- `.github/copilot-instructions.md`
- relevant files in `.github/instructions/`
- relevant skills in `.github/skills/`
- `AGENTS.md`
- relevant repo memories under `/memories/repo/` if available

If the lesson already exists, stop and report that no new instruction is needed.

### 3. Decide the Best Home

Use this decision table:

| If the lesson is... | Best home |
|---|---|
| Broad, always applicable across the repo | `.github/copilot-instructions.md` after user approval |
| Language- or folder-specific | matching `.github/instructions/*.instructions.md` after user approval |
| A multi-step repeatable workflow | `.github/skills/<name>/SKILL.md` after user approval |
| Useful but narrow, recent, or still evolving | `/memories/repo/` by default |
| One-off or unlikely to repeat | no documentation update |

### 4. Prefer Repo Memory Until the User Approves Documentation Changes

Default rule:

- **First choice:** store the verified lesson as repo memory or propose wording to the user
- **Second choice:** update an existing instruction/skill file with a short bullet or note, but only after explicit approval
- **Third choice:** create a new instruction/skill file only if no current file clearly owns the lesson and the user explicitly approves it

Avoid instruction sprawl. A new instruction/skill file should only be created when the lesson is both:

- stable and likely to recur
- specific enough to benefit from its own scope or `applyTo` pattern

### 5. Draft the Proposed Guidance

Write the guidance as a short, actionable rule:

- state the preferred pattern
- name the exact helper, command, or file when relevant
- mention the failure mode it prevents
- keep wording concise and imperative

Good example:

- `Use setFilesystemSupportOverride(fsType, response) in frontend tests instead of same-path server.use(...) overrides for /api/filesystem/support to keep Bun/MSW tests deterministic.`

Weak example:

- `Be careful with mocks in tests.`

### 6. Ask the User Before Any Instruction Edit

After drafting, always pause and ask for approval before editing or creating any instruction file.

Use a focused follow-up such as:

- Should this be a **workspace-specific instruction**, **workspace-specific skill**, or just **repo memory**?
- Do you want me to **update the existing instruction/skill now** or only **propose the wording**?
- Should this live under a **language-specific instruction** or a broader repo rule?

> Approval is required before changing `.github/instructions/*.md`, `.github/copilot-instructions.md`, `.github/skills/*.md` or `AGENTS.md`.

### 7. If Confirmed, Apply the Smallest Documentation Change

Only after the user explicitly confirms:

1. update the smallest relevant instruction file, or create a narrowly scoped new one. Avoid creating a new file if an existing one can be updated with a new bullet or note.
2. keep the change minimal and easy to scan. For example, add a single bullet to an existing instruction rather than creating a new section or file.
3. preserve existing style and structure. For example, if the instruction file uses a checklist format, add a new checklist item rather than a new paragraph.
4. record repo memory if the fact is verified and useful beyond the current task. This is the preferred default for new lessons until they are proven stable and reusable enough to justify an instruction update.
5. report back to the user with a link to the updated or new instruction/skill file, or the repo memory entry

---

## Quality Checks

Before finalizing, make sure the lesson:

- [ ] is backed by current-session evidence, code, or test output
- [ ] is likely to help future work in this repository
- [ ] is not already documented elsewhere
- [ ] is written as a concrete, actionable rule
- [ ] uses the smallest appropriate home
- [ ] user approval was obtained before any instruction-file edit or creation
- [ ] avoids creating a brand-new instruction file unless clearly justified
- [ ] avoids proposing a new skill unless the lesson is best expressed as a multi-step workflow
- [ ] is not a vague hunch or general advice without specifics
- [ ] is not too narrow, temporary, or one-off to be worth documenting
- [ ] is not already captured in repo memory if it is still evolving or too specific for an instruction update

## Example Prompts

- `capture lessons learned from this session`
- `review this fix and tell me if we should update github instructions`
- `document the test stabilization pattern we just found`
- `check whether today's implementation produced a reusable repo rule`

# Post-Execution Reflection

After this skill completes, check before closing:

- Did the command succeed? — If not, fix the instruction or error table that caused the failure.
- Did parameters or output change? — If the underlying tool's interface drifted, update Usage examples and Parameters table to match.
- Was a workaround needed? — If you had to improvise (different flags, extra steps), update this SKILL.md so the next invocation doesn't need the same workaround.

Only update if the issue is real and reproducible — not speculative.