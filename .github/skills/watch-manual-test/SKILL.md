---
name: watch-manual-test
description: "Observe a human-driven SRAT remote test session without taking over the UI. Reuses or opens the frontend at `http://localhost:3080/`, waits while the user manually navigates and clicks, records the user's actions, and concurrently monitors browser console/network plus frontend/backend/Home Assistant logs. If an error or strange effect appears, asks whether to ignore it or attempt a fix, then can start a focused autonomous subtask to investigate, implement a fix, verify it, and ask the user to retry. Triggers on: 'watch my testing', 'watch manual test', 'manual remote test', 'observe while I click around', 'monitor browser and backend logs', 'I will test manually, watch for errors'."
argument-hint: "Describe what you will test manually and whether the remote environment is already running (example: 'observe share creation flow; frontend and backend already up')."
---

# Watch Manual Test

Support a **user-led** remote testing session while continuously watching the browser and logs for problems.

## When to Use

Use this skill when:

- the user wants to manually drive the frontend instead of having the agent click through the UI
- the bug is intermittent, visual, or easier for a human to reproduce
- you need to correlate user actions with browser, frontend, backend, addon, or Home Assistant log output
- the user wants to report strange side effects in chat while the session is in progress
- the agent should pause on each issue and ask whether to ignore it or try to fix it

> Use `test-remote-environment` when the agent should deploy and run an end-to-end flow itself. Use this skill when the **human should stay in control of the browser**.

## Outcome

By the end of the workflow, you should produce one or more of these outcomes:

1. a monitored manual test session with the user's actions recorded in context
2. a concise list of observed errors, failed requests, warnings, or unusual effects
3. an explicit decision for each issue: **ignore for now** or **attempt a fix**
4. when approved, a focused autonomous fix cycle followed by a retry of the same manual steps

## Core Behavior Rules

- **Do not take over navigation** unless the user explicitly asks you to click, type, or drive the UI.
- When the browser is open on the frontend, **pause and wait for the user to interact**.
- Keep track of the user's recent actions so logs can be correlated with the exact step that triggered the issue.
- While the user interacts, continuously check:
  - browser state, visible error banners, and WebSocket status
  - browser console errors and failed network requests
  - frontend dev server output
  - backend/addon logs and, when relevant, Home Assistant core logs
- If you detect an issue, immediately ask:
  - `I found an error. Ignore it for now or should I try to correct it?`
- At any time, treat a user message like `I see something strange` or `that looked wrong` as a valid test signal even if logs are not yet clear.
- If the user wants a fix, create a focused subtask/todo for that issue, investigate the root cause, implement the smallest fix, verify it, and then ask the user to retry.

---

## Procedure

### 1. Confirm Session Scope

At the start of the session, confirm:

- what flow the user plans to test manually
- whether the environment is already running
- whether custom component behavior is in scope
- whether the goal is broad exploration or reproduction of a known bug

If the environment is not ready:

- **ask the user first** whether they want you to prepare or restart it
- only after they confirm, use the same setup approach as `test-remote-environment`

### 2. Open or Reuse the Frontend, Then Hand Control to the User

Open or reuse the frontend at:

```text
http://localhost:3080/
```

After the page is ready:

- confirm that the user can interact with it
- state that you are now **watching** rather than driving
- avoid automatic clicks or navigation unless requested

Good handoff wording:

- `The app is open. Please go ahead with your manual steps; I'll watch the browser and logs for issues.`

### 3. Live Observation Loop

While the user performs actions, keep repeating this loop:

1. note the latest user action or reported symptom
2. inspect the browser for visible changes, warnings, banners, or disconnects
3. inspect browser console and network activity for JS errors, failed requests, or slow responses
4. inspect frontend and backend/addon logs for lines that correlate with the last action
5. summarize only meaningful findings back to the user

Prefer short, evidence-based updates such as:

- `I saw a failed `POST /api/shares` request with status 500 right after the save click.`
- `The browser stayed responsive, but the backend logged a validation error for the same action.`

### 4. Error Handling Decision Point

When you find an issue — or the user reports a strange effect — pause the session briefly and ask:

- `I found an error/warning. Should I ignore it for now or try to correct it?`

If the user says **ignore**:

- note the issue briefly
- continue monitoring the rest of the session

If the user says **fix it**:

- start a focused subtask/todo for the issue
- collect the exact evidence first: console error, failed request, stack trace, and the action that triggered it
- investigate the root cause before editing code

### 5. Autonomous Fix Cycle for Approved Issues

When the user approves a fix attempt:

1. create a narrow subtask for the specific issue
2. reproduce the problem from the gathered evidence
3. write or update a focused test when appropriate
4. implement the smallest root-cause fix
5. run the relevant validation commands or tests
6. report the evidence of the result
7. ask the user to retry the same manual steps

If the retry still shows a problem, return to root-cause investigation instead of stacking guesses.

### 6. User-Initiated Reports During the Session

At any time, the user may send messages like:

- `I got an error popup`
- `That felt slow`
- `The UI flickered`
- `I saw a strange effect`

Treat these as first-class inputs. Do not wait for the browser tools or logs to complain first.

When the report is ambiguous, ask for the minimum needed clarification:

- what did they click just before it happened?
- what was visible on screen?
- did it happen once or every time?

Then correlate that report with the surrounding browser and backend evidence.

### 7. Retry and Continue Watching

After a fix or after choosing to ignore an issue:

- ask the user to retry the exact step or continue the next part of the flow
- keep watching the same signals
- confirm whether the symptom is gone, changed, or still present

### 8. Wrap Up the Session

At the end, provide a short summary grouped by:

- **Observed and fixed**
- **Observed and deferred/ignored**
- **Could not reproduce**
- **Suggested next follow-up**

---

## Decision Tree

```text
Manual session started
├── Environment not ready?
│   ├── Yes → ask first whether to prepare/restart it, then proceed if approved
│   └── No  → continue
├── Browser open and user ready?
│   ├── No  → open/reuse frontend and hand off control
│   └── Yes → observe only
├── User performs action or reports strange effect
│   └── Check browser + network + frontend logs + backend/HA logs
├── Problem found?
│   ├── No  → continue watching
│   └── Yes → ask: ignore or try to correct?
│       ├── Ignore → note it and continue the session
│       └── Fix → start focused subtask → investigate → implement → verify → ask user to retry
└── Session complete → summarize findings and remaining follow-ups
```

## Quality Checks

Before ending the session, make sure you:

- [ ] let the user stay in control of the browser unless they asked otherwise
- [ ] correlated at least the important issues with concrete evidence
- [ ] asked whether to ignore or fix each meaningful problem
- [ ] verified any claimed fix with a real command, test, or retry result
- [ ] clearly separated fixed issues from deferred ones

## Example Prompts

- `Watch my manual remote test while I try the share creation flow.`
- `Open the SRAT frontend and observe while I click around; monitor logs too.`
- `I want to test manually in Home Assistant. Please watch for frontend and backend errors.`
- `Stay on the browser page, let me navigate, and tell me if you see anything wrong.`
- `Observe this strange UI behavior while I reproduce it and try to fix it if needed.`
