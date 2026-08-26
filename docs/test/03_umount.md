# Test 03: Volume Unmount Functionality

## Overview

Validates unmounting a mounted partition through the Volumes page UI, including the safety acknowledgement dialog, back-end unmount execution, kernel-level verification, and persistence semantics of the mount configuration.

| Field | Value |
| --- | --- |
| Version tested | v2026.8.0-rc13.1 (git describe `2026.8.0-rc13-16-gc5d20970`) |
| Environment | HA test host `192.168.0.68`, addon `local_sambanas2`, lab mode OFF |
| Result | PASS |

## Prerequisites

- Partition mounted in Test 02 (TESTMOUNT on `/dev/sdc1`, mounted at its configured path).
- Frontend dev server running at `http://localhost:3080/`.

## Reproduction Steps

### Task 1: Select Mounted Partition

**Action:** On the Volumes page, select the mounted `TESTMOUNT` partition from the disk tree (click the inner Box: `li[id*="part"] .MuiTreeItem-label > div`).

**Expected:**

- Detail panel shows the partition with state "Mounted" and the mount path displayed.
- Action buttons include **UNMOUNT PARTITION** and **FORCE UNMOUNT** (warning-styled).

### Task 2: Trigger Unmount with Safety Acknowledgement

**Action:**

1. Click **UNMOUNT PARTITION**.
2. A `material-ui-confirm` dialog appears: "Unmount TESTMOUNT?" with CANCEL / UNMOUNT buttons.
3. Observe that UNMOUNT is disabled until the acknowledgement checkbox is checked.
4. Check the acknowledgement checkbox, then click UNMOUNT.

> UI automation note: checkboxes are invisible to accessibility snapshots; locate via `input[type="checkbox"]` inspection and click the parent `<label>` element.

**Expected:**

- UNMOUNT button stays disabled until acknowledgement is checked (color changes from grey to enabled blue).
- Clicking UNMOUNT closes the dialog; success toast "Volume ... unmounted successfully." appears.
- The "Mounted" badge disappears from the tree item.

### Task 3: Verify Unmount End-to-End

**Action:** Verify at three levels:

```bash
# 1. Kernel: mount must be gone
ssh root@192.168.0.68 "docker exec app_local_sambanas2 mount" | grep sdc1 || echo "KERNEL_NOT_MOUNTED"
# 2. API state
ssh root@192.168.0.68 "docker exec app_local_sambanas2 curl -sL http://localhost:64289/api/volumes" | grep -B2 -A8 '"is_mounted"'
# 3. Logs
ssh root@192.168.0.68 "docker logs app_local_sambanas2 --since 5m 2>&1 | grep -i unmount"
```

**Expected:**

- Kernel: `sdc1` no longer in mount table (`KERNEL_NOT_MOUNTED`).
- API: `mount_point_data.is_mounted:false`; the config entry itself persists with `is_to_mount_at_startup:false`.
- Logs: `service/volume_mount_manager.go:174 Successfully unmounted volume path=/mnt/...` followed by `share_service.go:138 MountPointEvent type=update IsMounted:false IsInvalid:false`.

### Task 4: Verify Config Persistence Semantics

**Action:** Compare `mount_point_data` before and after unmount.

**Expected:**

- Path, fstype, and label are retained in the configuration after unmount (so remount settings survive).
- `is_to_mount_at_startup` is false unless previously enabled.
- This persistence is expected design, not a bug.

## Observations / Potential Issues

1. The acknowledgement checkbox gating works correctly and prevents accidental unmounts.
2. FORCE UNMOUNT variant exists (warning color) for busy volumes; not exercised in this run since the volume was idle.
3. Source reference: `frontend/src/pages/volumes/Volumes.tsx` (~lines 425–473) uses the shared `useConfirm` acknowledgement pattern also used by Users and Shares pages.
