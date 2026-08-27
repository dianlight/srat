---
name: test-remote-environment
description: Test on the Homeassitant Dev envoronment (test-case driven)
argument-hint: "Describe what to test and whether custom component interaction is needed (e.g., 'test share creation flow, include custom component: yes'). Selected test cases will be proposed before execution."
---

# Test Remote Environment

Test the SRAT project against the live Home Assistant test environment. Deploys the backend binary via `mise //backend:build:remote`, starts the frontend dev server with `mise run //frontend:dev:remote`, controls the `local_sambanas2` addon (start/stop/restart) via the Home Assistant MCP, reads addon logs, and browses/validates the UI with Playwright at `http://localhost:3080/`.

This skill is **test-case driven**: every remote run is anchored to one or more test cases in `docs/test/{backend,frontend}/<func-id>.<test-id>_<description>.md` (see `test-plan` skill for the 3-phase format). A prepare phase proposes the cases to follow, allows creation of missing cases, tracks deviations during execution, and produces a timed summary with coverage suggestions.

## When to Use

- Validating a backend change against the real HA supervisor and Samba service
- Running end-to-end UI tests against a live backend
- Checking that the addon starts/restarts cleanly after a build change
- Investigating a bug that can only be reproduced on the real device
- Any time the user says "test remote", "deploy to test", "check HA", or "verify on addon"

## Prerequisites

| Requirement | How to verify |
|---|---|
| `HOMEASSISTANT_IP` env var (default 192.168.0.68) | `echo ${HOMEASSISTANT_IP:-192.168.0.68}` — must return an IP address |
| `SUPERVISOR_URL` env var set | `echo ${SUPERVISOR_URL:-http://192.168.0.68/` — must return e.g. `http://192.168.0.68/`; used to derive `API_URL` for the frontend dev server |
| SSH access to HA | `ssh root@${HOMEASSISTANT_IP:-192.168.0.68} echo ok` |
| `sshfs` available for remote mount | `which sshfs` |
| HA MCP server connected | MCP tools `mcp_home-assistan_ha_*` or `home-assistant-dev` must be available |
| Frontend dependencies installed | `cd frontend && bun install` |

## Argument Handling (Custom Component Scope)

Before running the procedure, decide whether custom component deployment is in scope.

1. Parse the user argument for explicit intent:
   - Include custom component flow when argument contains intent like `include custom component: yes`, `with custom component`, or similar.
   - Skip custom component flow when argument explicitly says `include custom component: no`, `backend-only`, or similar.
2. If the argument is ambiguous **and** the requested test could interact with Home Assistant integration behavior, ask:
   - `Should I include custom component remote deployment/reload in this test? (yes/no)`
3. Run the optional custom component steps only when the answer is `yes`.

## Prepare Phase — Test Case Selection (MANDATORY FIRST STEP)

**Do not skip this phase.** Even if the user named a specific feature, you must discover, propose, and confirm the test case manifest before any build/deploy.

### Step 0 — Discover, Propose, and Confirm Test Cases

#### 0.1 Discover existing cases

Scan the repository for test cases:

```bash
ls -1 docs/test/backend/*.md docs/test/frontend/*.md 2>/dev/null | grep -v README
# or via Glob: docs/test/**/*.md
```

For each file, extract:
- File path and filename (`<func-id>.<test-id>_<description>.md`)
- First heading (`# Test Case: ...`) and `## Test Objective` paragraph
- Layer label (`backend` vs `frontend`)

Example inventory (from current repo):

| File | Func | Layer | Objective |
|---|---|---|---|
| `docs/test/backend/001.001_create-a-share.md` | 001 | backend | Verify new SMB share via API + smb.conf |
| `docs/test/frontend/001.001_create-a-share-ui.md` | 001 | frontend | Verify share creation via UI wizard |
| `docs/test/frontend/001.002_edit-a-share-ui.md` | 001 | frontend | Verify share edit via UI |
| `docs/test/backend/002.001_check-disk-health.md` | 002 | backend | Verify disk health + SMART via API |

If `docs/test/README.md` exists, also read it for naming conventions and func-id registry.

#### 0.2 Propose via multi-select

Present the discovered cases to the user with the `question` tool (multi-select). Always include an option to create more cases.

Use this shape:

```json
{
  "questions": [{
    "header": "Test Cases",
    "question": "Select test cases to execute for this remote run. Choose one or more, or add new cases.",
    "multiple": true,
    "options": [
      {"label": "001.001_create-a-share (backend)", "description": "Verify new SMB share via SRAT API + smb.conf"},
      {"label": "001.001_create-a-share-ui (frontend)", "description": "Verify share creation via UI wizard"},
      {"label": "001.002_edit-a-share-ui (frontend)", "description": "Verify share edit via UI"},
      {"label": "002.001_check-disk-health (backend)", "description": "Verify disk health + SMART"},
      {"label": "Add new test case(s)", "description": "Create missing case(s) via test-plan skill"}
    ]
  }]
}
```

- `multiple: true` is required — the user may select several cases in one run.
- Label format must be `file` + `layer` so the manifest is unambiguous.
- Keep descriptions short (one line from Test Objective).

#### 0.3 Handle "no cases" and "add more"

- **If no test case files are found** (only `README.md` or empty dirs), do not proceed to build. Inform the user and immediately delegate to the `test-plan` skill:

  > No test cases found in `docs/test/`. I will invoke the `test-plan` skill to create the cases to follow.

  Follow `.agents/skills/test-plan/SKILL.md` to create cases following `<func-id>.<test-id>_<description>.md` and the 3-phase format (Preparation, Execution, Validation). After creation, re-run discovery and re-propose.

- **If the user selects "Add new test case(s)"** (alone or with other cases), invoke `test-plan` before execution:

  1. Ask a follow-up via `question` (single select or free-form) to capture what cases to create, e.g.:

     ```
     What additional cases should be created? (e.g., "001.003_delete-a-share backend", "002.002_disk-failure-alert frontend")
     ```

  2. Delegate to `test-plan` skill to scaffold the files in the correct `docs/test/backend/` or `docs/test/frontend/` directory.
  3. Re-discover and confirm the updated manifest (selected old cases + newly created cases) before continuing.

- **If the user selects only existing cases**, that selection becomes the execution manifest.

#### 0.4 Confirm manifest and initialize tracking

- Echo the confirmed manifest in chat:

  ```
  Test manifest confirmed (3 cases):
  - [ ] docs/test/backend/001.001_create-a-share.md
  - [ ] docs/test/frontend/001.001_create-a-share-ui.md
  - [ ] docs/test/backend/002.001_check-disk-health.md
  Start time: 2026-08-27T10:00:00Z
  ```

- Record for each case:
  - `status`: `pending` → `running` → `pass` / `fail` / `skip`
  - `start_time` / `end_time` / `duration`
  - `deviations` (changes vs spec)
  - `evidence` (logs, screenshots, API responses)

- Record global `suite_start_time` (e.g., `date -u +%Y-%m-%dT%H:%M:%SZ` or `date +%s` for elapsed seconds) for the final summary.

This manifest drives Steps 1–8. Each case is executed sequentially following its own 3-phase (Preparation → Execution → Validation) with the shared remote environment prepared once (Steps 1–5) and per-case UI/log checks (Steps 6–7).

---

## Procedure

### Step 1 — Build and deploy the backend

Run in the `backend/` terminal (background, it may take a minute):

```bash
mise //backend:build:remote
```

- This cross-compiles for `amd64`, then `rsync`s the binaries into `/addon_configs/local_sambanas2/upgrade/` on the HA host.
- Wait for the message `Remote build and deployment completed.` before proceeding.
- If `HOMEASSISTANT_IP` is not set, ask the user or check `.env`/shell profile.

### Step 2 — Restart the addon to pick up the new binary

Use the Home Assistant MCP to restart the addon:

```
mcp_home-assistan_ha_restart_addon  →  slug: "local_sambanas2"
```

Wait ~10 seconds for the addon to fully start before proceeding.

### Step 3 — Verify runtime health via logs (core first, then addon)

First fetch Home Assistant core logs (this is where custom component Python logs and tracebacks appear):

```
mcp_home-assistan_ha_core_logs
```

Then fetch addon logs:

```
mcp_home-assistan_ha_addon_logs  →  slug: "local_sambanas2"
```

**Look for:**
- In core logs: `custom_components.srat` import/setup errors, tracebacks, config-entry failures
- `"srat-server started"` or equivalent startup message
- No `FATAL` or `panic` lines
- No `permission denied` errors for the data directory

**If the addon failed to start:**
- Check for `panic` stack traces or `signal: killed` (OOM)
- Stop the addon, wait 5 s, start it again
- Re-read logs after restart

Addon control commands:
```
mcp_home-assistan_ha_stop_addon   →  slug: "local_sambanas2"
mcp_home-assistan_ha_start_addon  →  slug: "local_sambanas2"
```

### Step 4 — (Optional) Deploy custom component when requested

Only run this step when the argument indicates custom component interaction (or user confirms `yes` when asked).

Deploy the custom component to the remote HA environment:

```bash
cd custom_components && mise run install:remote
```

Then reload Home Assistant configuration/core as needed via MCP:

```
mcp_home-assistan_ha_reload_config  →  component: "core"
```

Verify no configuration/runtime errors after deploy (check core logs first, then addon logs):

```
mcp_home-assistan_ha_check_config
mcp_home-assistan_ha_core_logs
mcp_home-assistan_ha_addon_logs  →  slug: "local_sambanas2"
```

**Look for:**
- No Python import errors from `custom_components/srat`
- No `Config entry` setup failures
- No `ERROR`/`Traceback` lines introduced immediately after reload

### Step 5 — (Optional) Start the frontend dev server

Only needed when testing UI changes. Run in the `frontend/` terminal (background):

```bash
export HOMEASSISTANT_IP=192.168.0.68
export SUPERVISOR_URL=http://192.168.0.68/
export API_URL=http://192.168.0.68:3000/  # must be env var — -a flag alone is NOT enough
mise run //frontend:dev:remote
```

- `API_URL` **must be an env var** (`http://192.168.0.68:3000/` for remote). The Bun macro `getApiUrl()` is evaluated at import time (`frontend/src/index.html` imported at `frontend/bun.build.ts:8`) **before** `build()` sets `process.env.API_URL`; passing `-a http://...` alone leaves the macro as `dynamic` → same-origin `http://localhost:3080/` + `ws://localhost:3080/ws` (returns HTML 200, not WS, so `useVolume` shows `Error loading volumes: [object Object]`). See `docs/tasks/045_full-mobile-support.md:65`.
- `SUPERVISOR_URL` is the canonical source; derive `API_URL` as `${SUPERVISOR_URL%/}:3000/` if not set explicitly. Always verify with:
  ```bash
  curl -H "Origin: http://localhost:3080" http://192.168.0.68:3000/api/volumes -v  # expect Access-Control-Allow-Origin: http://localhost:3080
  curl -H "Connection: Upgrade" -H "Upgrade: websocket" http://192.168.0.68:3000/ws -v  # expect SSE frames id:/event:/data:
  ```
  The WS URL is `new URL("ws", apiUrl.replace(/^http/, "ws"))` → `ws://192.168.0.68:3000/ws` via `socat TCP-LISTEN:3000 → 127.0.0.1:64289`.
- **Never bypass `tsc`**: `mise run //frontend:dev:remote` runs `//frontend:generate` (`rtk-query-codegen-openapi` + `bun tsc --noEmit`) first. If it fails with `TS6133: 'container' is declared but its value is never read` / `TS2339`, the dev server will not start. **Do not** fall back to `bun --hot run bun.build.ts -w -s ./out -a ...` — expose the full `tsc` error block, fix unused `container`/`within` destructurings (e.g. `const { container } = await render...()` → `await render...();`) and `import { within }` removal, then retry. `mise run //frontend:dev:remote` will fail fast if `3080` is occupied by a non-stale process (only a stale `python3 -m http.server 3080` is auto-killed by `ensure-dev-port`).
- This compiles TypeScript, starts a watch build, and serves the frontend at **`http://localhost:3080/`**.
- Keep the terminal visible — TypeScript type errors and HMR output appear in stdout.
- Wait for the line `Bun.serve listening on :3080` before opening the browser.

### Step 6 — Browse and validate the UI with Playwright (per test case)

For each test case in the manifest, execute its **Execution** (and **Validation**) steps using Playwright when the case is frontend-layer, or via API/SSH when backend-layer. Navigate to the app:

```
mcp_playwright_browser_navigate  →  url: "http://localhost:3080/"
```

Take a screenshot to verify the page loaded:

```
mcp_playwright_browser_take_screenshot
```

Then use Playwright tools as needed to interact with the UI:
- `mcp_playwright_browser_click` — click buttons, links, menu items
- `mcp_playwright_browser_fill_form` — fill in forms
- `mcp_playwright_browser_snapshot` — get DOM accessibility snapshot for assertions
- `mcp_playwright_browser_evaluate` — run JavaScript in the page context
- `mcp_playwright_browser_console_messages` — read browser console output (check for JS errors)
- `mcp_playwright_browser_network_requests` — inspect API calls and responses
- `mcp_playwright_browser_wait_for` — wait for an element or condition before continuing

**Per-case execution rules:**
- Before each case, set `case_start_time` and update manifest to `running`.
- Follow the test case's **Preparation** checks first — if any preparation step fails and cannot be completed, mark the case `skip` and continue to the next case (document reason).
- Then follow **Execution** steps exactly, asserting expected outcomes. Capture evidence (API status, `testparm` output, screenshots, console messages).
- Then run **Validation** steps.

**Common validation checks:**
- WebSocket status indicator in the NavBar is green (connected)
- No red error banners on page load
- Browser console has no uncaught errors
- Network requests return 2xx status codes

**MUI TreeView / ToggleButtonGroup interaction note:**
- `browser_snapshot` may return only `RootWebArea` for MUI components that use virtual trees or custom rendering (e.g., `SimpleTreeView`, `ToggleButtonGroup`)
- Use `browser_eval` with JavaScript DOM queries as a fallback: `document.querySelector(...)` or React fiber access via `__reactFiber$<hash>`
- For `ToggleButtonGroup` clicks, React fiber handler invocation is often the only reliable method (MUI resists synthetic `PointerEvent`/`MouseEvent` dispatch)

### Step 7 — Read logs again after UI interaction (core first, then addon) — per test case

After each test case's actions, re-read logs to catch backend and integration errors:

```
mcp_home-assistant_ha_core_logs
```

Then:

```
mcp_home-assistant_ha_addon_logs  →  slug: "local_sambanas2"
```

Look for new `ERROR` or `WARN` lines correlating with the case's actions. Append findings to that case's evidence. If a case fails, keep its logs for the summary.

### Step 7a — Verify cache-sensitive data freshness (recommended, per case when applicable)

When testing features that modify backend state (config saves, DB writes) and then read it back via a different API endpoint, stale caches can produce misleading test results. After any state-mutating action (save, create, delete), verify the data is consistent across all read paths.

**Known pattern — HardwareService cache (30-min TTL):**
- `SaveDeviceConfig` writes to DB but may not invalidate `HardwareService` cache
- `/api/disk/{id}/hdidle/config` reads DB directly (fresh)
- `/api/volumes` reads from `HardwareService` cache (may be stale)
- Result: individual endpoint shows correct data, volumes endpoint shows stale `supported=false`

**Verification approach:**
1. After a state-mutating action, restart the addon to clear all in-memory caches:
   ```
   mcp_home-assistant_ha_stop_addon   →  slug: "local_sambanas2"
   Wait 5 seconds
   mcp_home-assistant_ha_start_addon  →  slug: "local_sambanas2"
   ```
2. Re-read the data from the affected endpoint after restart
3. Compare with the direct-read endpoint to confirm consistency

**When to apply this step:**
- Testing config save flows (HDIdle, shares, users, settings)
- Testing any feature where a POST/PUT is followed by a GET on a different endpoint
- Investigating data inconsistencies between individual and list endpoints

### Handling Changes to Test Cases During Execution

If during Steps 6–7 you discover that the test case spec is incomplete, inaccurate, or needs deviation (e.g., missing preparation step, changed API path, UI flow differs, additional assertion needed):

1. **Pause** execution of that case — do not silently diverge.
2. **Ask the user** via `question` tool:

   ```json
   {
     "questions": [{
       "header": "Test Case Change",
       "question": "Case 001.001_create-a-share differs from spec: API now requires 'browseable' field. Update the test case file?",
       "options": [
         {"label": "Yes, update case", "description": "Apply change to docs/test/...md and continue"},
         {"label": "No, deviate only", "description": "Run deviated step but keep file unchanged"},
         {"label": "Skip case", "description": "Skip this case and log deviation"}
       ]
     }]
   }
   ```

3. Act on the answer:
   - `Yes, update case` → edit the corresponding `docs/test/**/*.md` file to reflect the corrected Preparation/Execution/Validation, note the change in the manifest `deviations`, then continue.
   - `No, deviate only` → execute the deviated step, record the deviation in evidence, but leave the file unchanged.
   - `Skip case` → mark `skip`, record reason, proceed to next case.

Record every deviation in the final summary.

### Step 8 — Clean up

- Close the Playwright browser if no longer needed: `mcp_playwright_browser_close`
- The frontend dev server can be left running in the background or stopped with Ctrl+C in the frontend terminal.

### Step 9 — Summary, Timing, and Coverage Suggestions (MANDATORY LAST STEP)

After all cases in the manifest have been attempted, produce a comprehensive summary **before closing**.

#### 9.1 Per-case results and timing

Record elapsed time using the suite timestamps (`suite_start_time` → `suite_end_time` = `date -u +%Y-%m-%dT%H:%M:%SZ` and `duration = end - start` in seconds/minutes). For each case, report:

| # | Test Case | Layer | Result | Duration | Evidence / Notes |
|---|---|---|---|---|---|
| 1 | `001.001_create-a-share` | backend | ✅ pass | 42s | API 201, smb.conf ok, screenshot `...` |
| 2 | `001.001_create-a-share-ui` | frontend | ❌ fail | 58s | POST /api/shares 500 after save, console error X |
| 3 | `002.001_check-disk-health` | backend | ⏭️ skip | 5s | Preparation failed: no disks detected |

Result legend: `✅ pass` / `❌ fail` / `⏭️ skip`.

Include total suite time:

```
Suite time: 3 cases, 2 pass / 1 fail / 0 skip, total 1m 45s (avg 35s/case)
Started: 2026-08-27T10:00:00Z — Ended: 2026-08-27T10:01:45Z
```

#### 9.2 Coverage gap analysis and suggestions

Check whether all functions are tested:

1. Build the func-id inventory from `docs/test/**` (group by `func-id`):
   - `001` share operations → cases: `001.001`, `001.002` ...
   - `002` disk operations → cases: `002.001` ...
2. Compare against known SRAT functional areas (from `docs/test/README.md`, `docs/` and recent code). At minimum, check these buckets:
   - 001 shares (create/edit/delete/list)
   - 002 disks/health/SMART
   - volumes/mounts, users, settings, hardware/HDIdle, WebSocket, HA integration, Samba service, addon config

3. If any func-id or functional area has zero cases, or a case file exists but was not selected for this run, suggest follow-up cases. Example suggestion block:

   > **Coverage suggestions:**
   > - Func `001` missing `001.003_delete-a-share` (backend) — no deletion test selected.
   > - No test case for `HDIdle config save` (hardware service cache) — consider `003.001_hdidle-save-and-verify`.
   > - Frontend `volumes` list not covered — consider `004.001_volumes-list-ui`.
   >
   > Create these via `test-plan` skill? (yes/no)

4. If the user accepts, delegate to `test-plan` to scaffold the suggested cases. Offer to run them in a follow-up remote session.

#### 9.3 Deliverable

The skill returns a comprehensive test report including:

- **Test manifest**: selected cases and any cases created via `test-plan` during Step 0
- **Build status**: success/failure and any errors encountered
- **Addon health**: start status, logs analysis, and any runtime issues
- **Custom component status** (if deployed): deployment success, runtime errors
- **Per-case results**: pass/fail/skip, duration, evidence, deviations (with user decisions)
- **Timing**: per-case and total suite time
- **Cache consistency** (if state-mutating tests ran): verified data freshness across endpoints after cache-sensitive operations
- **Coverage suggestions**: untested func-ids / functional areas and proposed new cases (with `test-plan` follow-up)
- **Cleanup status**: proper browser and server termination

## Decision Tree

```
Prepare Phase (Step 0) — MANDATORY
├── Discover docs/test/** cases
├── Present multi-select (existing cases + "Add new test case(s)")
├── No cases found OR user selected "Add new"
│   └── Invoke test-plan skill → create cases → re-discover → re-confirm manifest
└── Manifest confirmed → record suite_start_time

Build successful? (Step 1)
├── No  → Check Go errors, fix, retry Step 1
└── Yes → Addon start OK? (Steps 2-3)
                     ├── No  → Read logs (Step 3), stop/start, check again
                     └── Yes → Custom component included?
                                            ├── Yes → Run Step 4 (deploy/reload/check)
                                            └── No  → Continue
                                                                ↓
                                                            UI test needed? (Step 5)
                                                            ├── No  → For each case: run Execution (API/SSH) → Logs (Step 7)
                                                            └── Yes → Start frontend (Step 5) → For each case: Playwright (Step 6) → Logs (Step 7) → Handle deviations (ask user)

Per-case handling:
- Preparation fails → mark skip → next case
- Execution deviates from spec → ask user (update file / deviate only / skip) → record deviation
- After each case → record case_end_time and evidence

All cases done → Step 9 Summary
├── Per-case table + total suite time
├── Coverage gap analysis (func-ids not tested)
└── Suggest new cases via test-plan if gaps remain → ask user
```

## Troubleshooting Reference

| Symptom | Likely cause | Action |
|---|---|---|
| `HOMEASSISTANT_IP is not set` | Env var missing | Set `export HOMEASSISTANT_IP=<ip>` in the shell |
| `SUPERVISOR_URL is not set` / blank API_URL | Env var missing | Set `export SUPERVISOR_URL=http://<ip>/` in the shell |
| `rsync: connection refused` | SSH not running / wrong IP | Verify SSH access manually |
| Custom component errors are missing in addon logs | Looking at wrong log source | Check `mcp_home-assistan_ha_core_logs` first; custom component Python errors are logged in Home Assistant core logs |
| Addon fails to start, `signal: killed` | OOM or binary mismatch | Check if PPROF port conflicts; rebuild without `PPROF=1` |
| Addon fails to start, Mach-O format error | Cross-compilation produced macOS binary | Ensure `GOOS=linux` is set in all build export sections of `backend/.mise.toml` (zig-musl, glibc, static) |
| Addon starts but API 404s | Old binary still running | Stop, wait 5 s, start again |
| Custom component fails after deploy | Reload/setup error in HA | Run `mcp_home-assistan_ha_check_config`, then inspect addon/core logs for traceback and fix component imports/schema |
| Frontend build loop / TS errors | Type error in changed file | Fix the error shown in frontend terminal stdout |
| Playwright blank page | Frontend not yet ready | Wait for `Bun.serve listening on :3080` in terminal |
| WebSocket not connecting | Proxy / CORS | Check `mise run //frontend:dev:remote` stdout for proxy errors |
| Browser console CORS errors | API_URL mismatch | Verify `HOMEASSISTANT_IP` matches `API_URL` in `.mise.toml` `dev:remote` |
| Individual API returns correct data but list API returns stale/defaults | In-memory cache stale (e.g., HardwareService 30-min cache) | Restart addon to clear cache, re-read from list endpoint; file bug if `Save*` methods don't call `Invalidate*` |
| UI panel hidden despite correct DB data | Backend cache stale → `supported=false` → frontend visibility gate blocks rendering | Restart addon, verify panel appears; report as cache invalidation bug |
| Direct API access needed | Cannot reach backend API externally | Use `docker exec addon_local_sambanas2 curl -sL http://localhost:64289/api/...` from the HA host (no auth required — internal-only API) |
| `smbpasswd -L` fails / shows help | `smbpasswd -L` is broken in the addon container | Use `pdbedit -a -u <username>` instead to set Samba passwords; `pdbedit -L` to list existing users |
| No test cases found in docs/test/ | Fresh repo or cases not yet created | Invoke `test-plan` skill to scaffold cases before building |
| User wants more coverage | Func-id gaps detected | Delegate to `test-plan`, then re-run remote test with new manifest |

## Increase Custom Component Verbosity

Use this when custom component behavior is unclear and you need deeper diagnostics.

1. Enable debug logs in Home Assistant `configuration.yaml` (test environment):

```yaml
logger:
     default: info
     logs:
         custom_components.srat: debug
         custom_components.srat.websocket_client: debug
         custom_components.srat.coordinator: debug
```

2. Reload Home Assistant core to apply logger config changes:

```
mcp_home-assistan_ha_reload_config  →  component: "core"
```

3. Read Home Assistant core logs first:

```
mcp_home-assistan_ha_core_logs
```

4. Correlate with addon logs if needed:

```
mcp_home-assistan_ha_addon_logs  →  slug: "local_sambanas2"
```

5. After debugging, revert logger entries to `info` (or remove them) to avoid noisy logs.

**Notes:**
- Custom component Python logs are emitted in **Home Assistant core logs**, not addon logs.
- Prefer temporary debug enablement only during active investigation.

## Addon Info Reference

Get detailed addon state (version, options, state):

```
mcp_home-assistan_ha_addon_info  →  slug: "local_sambanas2"
```

Update addon options before restart:

```
mcp_home-assistan_ha_set_addon_options  →  slug: "local_sambanas2", options: { ... }
```

## Usage Examples

### Example 1: Test backend cases with multi-select

```
User: "Test share creation on remote"
Agent: [Step 0] Discovers 4 cases, proposes multi-select
User: selects [001.001_create-a-share (backend), 002.001_check-disk-health (backend)]
Agent: manifest confirmed (2 cases), starts build → addon restart → per-case API execution → summary (2 pass, total 1m 10s)
```

### Example 2: No cases yet — create via test-plan

```
User: "Test HDIdle save flow on remote"
Agent: [Step 0] No HDIdle cases found
Agent: "No test cases found for HDIdle. Invoke test-plan to create 003.001_hdidle-save-and-verify?"
User: yes
Agent: delegates to test-plan → creates docs/test/backend/003.001_hdidle-save-and-verify.md → re-proposes manifest → executes
```

### Example 3: Add more cases during prepare

```
User: "Test share flows, include custom component: yes"
Agent: [Step 0] Proposes 4 existing cases + "Add new test case(s)"
User: selects [001.001_create-a-share-ui, 001.002_edit-a-share-ui, Add new test case(s)]
Agent: asks what to create → user: "001.003_delete-a-share-ui"
Agent: test-plan creates 001.003 → manifest now 3 cases → proceeds to build → per-case UI tests → summary suggests additional volumes tests
```

### Example 4: Deviation during execution

```
Agent: [Step 6] Case 001.001 expects POST /api/shares but API now requires `browseable` field
Agent: asks "Update test case file? (Yes update / No deviate only / Skip)"
User: "Yes, update case"
Agent: edits docs/test/backend/001.001_create-a-share.md, records deviation, continues
```

### Example 5: Summary with coverage suggestions

```
Agent: [Step 9] Summary:
Suite time: 2 cases, 2 pass, total 1m 45s
Coverage: func 001 covered, func 002 covered, but no cases for volumes/mounts or users
Suggestion: Create 004.001_volumes-list and 005.001_create-user via test-plan?
User: yes → delegate to test-plan
```

## Return Values

This skill returns a comprehensive test report including:

- **Test manifest**: selected cases and any cases created via `test-plan` during Step 0
- **Build status**: success/failure and any errors encountered
- **Addon health**: start status, logs analysis, and any runtime issues
- **Custom component status** (if deployed): deployment success, runtime errors
- **Per-case results** (if frontend started): pass/fail/skip, duration, evidence, deviations
- **Timing**: per-case and total suite time (started/ended timestamps, avg per case)
- **Cache consistency** (if state-mutating tests ran): verified data freshness across endpoints after cache-sensitive operations
- **Coverage suggestions**: untested func-ids / functional areas and proposed new cases (with `test-plan` follow-up)
- **Cleanup status**: proper browser and server termination

## Error Handling

This skill gracefully handles:

- Missing environment variables
- No test cases found (delegates to `test-plan`)
- SSH connection failures
- Build compilation errors
- Addon startup failures
- Custom component deployment issues
- Frontend server startup problems
- Playwright browser errors
- Network connectivity issues
- Test case preparation failures (marks skip)
- Deviations from spec (asks user, records decision)

All errors are logged with actionable guidance, and the skill provides clear next steps for resolution.
