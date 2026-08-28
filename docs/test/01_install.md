# Test 01: Custom Component Install Functionality

## Overview

Validates the installation lifecycle of the SRAT Home Assistant custom component from the SRAT back end: baseline detection of the missing component, install via API, problem tracking transitions, and post-restart connectivity verification.

| Field | Value |
| --- | --- |
| Version tested | v2026.8.0-rc13.1 (git describe `2026.8.0-rc13-16-gc5d20970`) |
| Environment | HA test host `192.168.0.68`, addon `local_sambanas2`, lab mode OFF |
| Result | PASS (with findings) |

## Key Finding: Install UI Is Lab-Gated (By Design)

With `experimental_lab_mode` OFF, the **Install custom component panel is hidden** in Settings → Home Assistant. The panel only renders when lab mode is enabled:

```tsx
// frontend/src/pages/settings/panels/HomeAssistantPanel.tsx (~line 101)
{experimentalLabMode ? <HomeAssistantCustomComponentPanel readOnly={readOnly} /> : null}
```

The backend API endpoints are **not** lab-gated, so the flow was verified through the API while confirming the UI gating behavior live.

## Prerequisites

- Backend v2026.8.0-rc13.1 deployed and healthy on addon `local_sambanas2`.
- `experimental_lab_mode` set to `false` (verified via UI toggle and API).
- Direct API access available:

```bash
ssh root@192.168.0.68 "docker exec app_local_sambanas2 curl -sL http://localhost:64289/api/settings/homeassistant/custom-component/status"
```

## Reproduction Steps

### Task 1: Verify Baseline State

**Action:** Query the component status endpoint and the problems list before installing.

```bash
ssh root@192.168.0.68 "docker exec app_local_sambanas2 curl -sL http://localhost:64289/api/settings/homeassistant/custom-component/status"
ssh root@192.168.0.68 "docker exec app_local_sambanas2 curl -sL http://localhost:64289/api/problems"
```

**Expected:**

- Status returns `installed:false`, `can_install:true`, `can_uninstall:false`, `latest_version:2026.8.0-rc13`.
- Problems list contains one entry with `problem_key:"custom_component_missing"` (severity warning).

### Task 2: Confirm Lab-Off UI Gating

**Action:** Navigate to Settings → Home Assistant category in the UI with lab mode OFF.

**Expected:**

- The category shows only "Export Stats to Home Assistant" and "Use mDNS Proxy" switches.
- No install/uninstall/upgrade buttons or component status card are visible.
- With lab mode ON (control check), the `HomeAssistantCustomComponentPanel` appears with Install action and confirm dialog.

### Task 3: Install via API

**Action:** Trigger the install endpoint (POST takes no body).

```bash
ssh root@192.168.0.68 "docker exec app_local_sambanas2 curl -sL -X POST http://localhost:64289/api/settings/homeassistant/custom-component/install"
```

**Expected:**

- HTTP 200 response.
- Response body shows `installed:true`, `installed_version:2026.8.0-rc13`, `can_uninstall:true`, `can_install:false`.
- Component files appear at `/config/custom_components/srat` on the HA host (alongside any existing `hacs` directory).

```bash
ssh root@192.168.0.68 "ls /usr/share/hassio/homeassistant/custom_components/"
```

### Task 4: Verify Problem Lifecycle Transition

**Action:** Re-query the problems list after install.

**Expected:**

- Problem `custom_component_missing` is cleared.
- New problem appears: `custom_component_restart_required` ("Home Assistant should be restarted..."), severity warning, fixable—matching the documented ProblemService pattern for component installs.

### Task 5: Restart Home Assistant Core

**Action:** Restart HA core so it loads the new integration. Use the Supervisor API from inside the addon container:

```bash
ssh root@192.168.0.68 "docker exec app_local_sambanas2 sh -c 'curl -sL -X POST -H \"Authorization: Bearer \$SUPERVISOR_TOKEN\" http://supervisor/core/restart'"
```

Wait ~60 seconds for core to come back up, then verify status:

```bash
ssh root@192.168.0.68 "docker exec app_local_sambanas2 curl -sL http://localhost:64289/api/settings/homeassistant/custom-component/status"
```

**Expected:**

- Status shows `connected:true`, `connected_version:2026.8.0-rc13`, and a populated `connected_at` timestamp.
- Note: the MCP tool `ha_restart` returned a transient 502 during this step; the direct Supervisor call succeeded.

### Task 6: Verify Integration Health in HA Core Logs

**Action:** Inspect Home Assistant core logs after restart.

**Expected:**

- No Python import errors from `custom_components.srat`.
- Log lines confirming:
  - Custom integration `srat` loaded (a benign WARNING about untested integrations may appear).
  - WebSocket client connected: `Connected to SRAT WebSocket at ws://172.30.32.1:64289/ws`.
  - mDNS announcement registered: `homeassistant2._smb._tcp.local port 445`.
  - Coordinator refreshing data every ~5 seconds.

## Observations / Potential Issues

1. **Restart-required warning persists after successful restart+connection.** Problem `custom_component_restart_required` was not cleared automatically once the component reconnected. It remains visible until manually dismissed.
2. **UI gating asymmetry:** the install/uninstall/upgrade REST endpoints are reachable regardless of lab mode; only the UI panel is gated. Acceptable if intentional, worth documenting.
