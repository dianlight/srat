# Global Functional Test Status: Release Candidate v2026.8.0-rc13.1

Test campaign for final release preparation, executed with `experimental_lab_mode` OFF against the remote Home Assistant test environment.

- Version under test: `v2026.8.0-rc13.1` (git describe `2026.8.0-rc13-16-gc5d20970`)
- Environment: HA test host `192.168.0.68`, addon `local_sambanas2` (Samba NAS2)
- Campaign date: 2026-08-24
- Status legend: `no-test` / `pass` / `error`

## Summary Table

| # | Test | Description | Version Tested | Status | Pass Date | Related Issue |
| --- | --- | --- | --- | --- | --- | --- |
| 01 | Install | SRAT custom component install lifecycle + connectivity | v2026.8.0-rc13.1 | pass | 2026-08-24 | [#984](https://github.com/dianlight/srat/issues/984) |
| 02 | Mount | Partition mount via Volumes UI (ext4) | v2026.8.0-rc13.1 | pass | 2026-08-24 | [#990](https://github.com/dianlight/srat/issues/990) |
| 03 | Unmount | Partition unmount with safety acknowledgement | v2026.8.0-rc13.1 | pass | 2026-08-24 | - |
| 04 | Share | Share create / edit / delete lifecycle | v2026.8.0-rc13.1 | pass | 2026-08-24 | [#986](https://github.com/dianlight/srat/issues/986), [#991](https://github.com/dianlight/srat/issues/991) |
| 05 | User | User create / edit / delete lifecycle | v2026.8.0-rc13.1 | pass | 2026-08-24 | [#989](https://github.com/dianlight/srat/issues/989) |

All five functional areas passed. Two findings require follow-up before release (see below); neither blocks core functionality. Cross-cutting frontend findings are tracked under #985, #987, and #988.

## Findings Register

| ID | Area | Severity | Issue | Finding |
| --- | --- | --- | --- | --- |
| F1 | Users | Bug | [#989](https://github.com/dianlight/srat/issues/989) | Editing an existing user requires re-entering the password (PUT `/api/user/<name>` returns 422 "Password is required" on empty password); failure is not surfaced to the user (console-only error). See [05_user.md](./05_user.md). |
| F2 | Install | Observation | [#984](https://github.com/dianlight/srat/issues/984) | Problem `custom_component_restart_required` persists after HA restart and successful component reconnection; not cleared automatically. Also, install UI panel is hidden with lab mode OFF while REST endpoints remain reachable (by design?). See [01_install.md](./01_install.md). |
| F3 | Shares | Minor | [#986](https://github.com/dianlight/srat/issues/986) | Usage combobox: empty-value Enter clears selection AND submits create dialog (created share got `usage:none` unexpectedly). See [04_share.md](./04_share.md). |
| F4 | Frontend | Minor | [#985](https://github.com/dianlight/srat/issues/985) | Problems header badge count drifts upward across session (reached 8) while `GET /api/problems` returns at most one problem—badge/API mismatch. |
| F5 | Volumes | Minor | [#987](https://github.com/dianlight/srat/issues/987) | React "Maximum update depth exceeded" logged 3× on Volumes page render loop. |
| F6 | Frontend | Minor | [#988](https://github.com/dianlight/srat/issues/988) | ErrorBoundary crash (`RovingTabIndexContext missing MenuItem4`) after a back-end addon restart recovers only via full page reload. |
| F7 | Shares | Cosmetic | [#991](https://github.com/dianlight/srat/issues/991) | Generated smb.conf duplicates valid-users entries (`admin admin`); no explicit `guest ok = yes` line despite guest_ok enabled. |
| F8 | Volumes | Cosmetic | [#990](https://github.com/dianlight/srat/issues/990) | WD PC SN740 disk partition count displayed as 3 after session events vs 2 at start—unexplained, needs investigation if reproducible. |

## Environment Notes

- Backend deployed via `mise //backend:build:remote` (static amd64 binaries rsynced to `/addon_configs/local_sambanas2/upgrade/`).
- Container recreation resets `/usr/local/bin` binaries; re-apply copy fix after recreations (plain stop/start is safe).
- Direct API access pattern used throughout: `ssh root@192.168.0.68 "docker exec app_local_sambanas2 curl -sL http://localhost:64289/api/..."`.
- `smbpasswd` broken in addon container; use `pdbedit` for Samba password operations.
