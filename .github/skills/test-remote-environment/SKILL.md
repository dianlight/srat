---
name: test-remote-environment
description: "Test the SRAT project against the live Home Assistant test environment. Deploys the backend binary via 'make build_remote', starts the frontend dev server with 'bun dev:remote', controls the local_sambanas2 addon (start/stop/restart) via the Home Assistant MCP, reads addon logs, and browses/validates the UI with Playwright at http://localhost:3080/. Triggers on: 'test remote', 'test in HA', 'deploy to test', 'check test environment', 'test on addon', 'run integration test'."
argument-hint: "Describe what to test (e.g., 'verify SSE removal', 'test share creation flow')"
---

# Test Remote Environment

Deploys SRAT to the live Home Assistant test environment, controls the addon lifecycle, and validates behaviour via logs, the backend API, and the UI using Playwright.

## When to Use

- Validating a backend change against the real HA supervisor and Samba service
- Running end-to-end UI tests against a live backend
- Checking that the addon starts/restarts cleanly after a build change
- Investigating a bug that can only be reproduced on the real device
- Any time the user says "test remote", "deploy to test", "check HA", or "verify on addon"

## Prerequisites

| Requirement | How to verify |
|---|---|
| `HOMEASSISTANT_IP` env var set | `echo $HOMEASSISTANT_IP` — must return an IP address |
| `SUPERVISOR_URL` env var set | `echo $SUPERVISOR_URL` — must return e.g. `http://192.168.0.68/`; used to derive `API_URL` for the frontend dev server |
| SSH access to HA | `ssh root@$HOMEASSISTANT_IP echo ok` |
| `sshfs` available for remote mount | `which sshfs` |
| HA MCP server connected | MCP tools `mcp_home-assistan_ha_*` must be available |
| Frontend dependencies installed | `cd frontend && bun install` |

## Procedure

### Step 1 — Build and deploy the backend

Run in the `backend/` terminal (background, it may take a minute):

```bash
cd backend && make build_remote
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

### Step 3 — Verify the addon is running via logs

Fetch addon logs:

```
mcp_home-assistan_ha_addon_logs  →  slug: "local_sambanas2"
```

**Look for:**
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

### Step 4 — (Optional) Start the frontend dev server

Only needed when testing UI changes. Run in the `frontend/` terminal (background):

```bash
cd frontend && bun dev:remote
```

- `API_URL` is derived automatically from `SUPERVISOR_URL` — the script computes `${SUPERVISOR_URL%/}:3000/` (strips trailing slash, appends the SRAT backend port). Ensure `SUPERVISOR_URL` is set before running.
- This compiles TypeScript, starts a watch build, and serves the frontend at **`http://localhost:3080/`**.
- Keep the terminal visible — TypeScript type errors and HMR output appear in stdout.
- Wait for the line `Bun.serve listening on :3080` before opening the browser.

### Step 5 — Browse and validate the UI with Playwright

Navigate to the app:

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

**Common validation checks:**
- WebSocket status indicator in the NavBar is green (connected)
- No red error banners on page load
- Browser console has no uncaught errors
- Network requests return 2xx status codes

### Step 6 — Read logs again after UI interaction

After triggering actions in the UI, re-read addon logs to catch backend errors:

```
mcp_home-assistan_ha_addon_logs  →  slug: "local_sambanas2"
```

Look for new `ERROR` or `WARN` lines correlating with the UI actions taken.

### Step 7 — Clean up

- Close the Playwright browser if no longer needed: `mcp_playwright_browser_close`
- The frontend dev server can be left running in the background or stopped with Ctrl+C in the frontend terminal.

## Decision Tree

```
Build successful?
├── No  → Check Go errors, fix, retry Step 1
└── Yes → Addon start OK?
           ├── No  → Read logs (Step 3), stop/start, check again
           └── Yes → UI test needed?
                      ├── No  → Read logs only (Step 6) — done
                      └── Yes → Start frontend (Step 4) → Playwright (Step 5) → Logs (Step 6)
```

## Troubleshooting Reference

| Symptom | Likely cause | Action |
|---|---|---|
| `HOMEASSISTANT_IP is not set` | Env var missing | Set `export HOMEASSISTANT_IP=<ip>` in the shell |
| `SUPERVISOR_URL is not set` / blank API_URL | Env var missing | Set `export SUPERVISOR_URL=http://<ip>/` in the shell |
| `rsync: connection refused` | SSH not running / wrong IP | Verify SSH access manually |
| Addon fails to start, `signal: killed` | OOM or binary mismatch | Check if PPROF port conflicts; rebuild without `PPROF=1` |
| Addon starts but API 404s | Old binary still running | Stop, wait 5 s, start again |
| Frontend build loop / TS errors | Type error in changed file | Fix the error shown in frontend terminal stdout |
| Playwright blank page | Frontend not yet ready | Wait for `Bun.serve listening on :3080` in terminal |
| WebSocket not connecting | Proxy / CORS | Check `bun dev:remote` stdout for proxy errors |
| Browser console CORS errors | API_URL mismatch | Verify `HOMEASSISTANT_IP` matches `API_URL` in `package.json` `dev:remote` |

## Addon Info Reference

Get detailed addon state (version, options, state):

```
mcp_home-assistan_ha_addon_info  →  slug: "local_sambanas2"
```

Update addon options before restart:

```
mcp_home-assistan_ha_set_addon_options  →  slug: "local_sambanas2", options: { ... }
```
