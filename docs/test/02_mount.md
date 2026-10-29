# Test 02: Volume Mount Functionality

## Overview

Validates mounting a partition through the Volumes page UI: partition selection, mount dialog configuration, back-end mount execution, kernel-level verification, and share service notification.

| Field | Value |
| --- | --- |
| Version tested | v2026.8.0-rc13.1 (git describe `2026.8.0-rc13-16-gc5d20970`) |
| Environment | HA test host `192.168.0.68`, addon `local_sambanas2`, lab mode OFF |
| Result | PASS |

## Prerequisites

- A blank partition suitable for testing. In this run: `/dev/sdc1` (USB Flash Disk, 61.9 GB), no filesystem present.
- Backend running; direct API access via `docker exec app_local_sambanas2 curl -sL http://localhost:64289/api/...`.

## Reproduction Steps

### Task 1: Prepare a Fresh Filesystem

**Action:** Format the target partition as ext4 from inside the addon container (the container ships full e2fsprogs):

```bash
ssh root@192.168.0.68 "docker exec app_local_sambanas2 sh -c 'mkfs.ext4 -F -L TESTMOUNT /dev/sdc1'"
```

**Expected:**

- `mkfs.ext4` completes without errors; label `TESTMOUNT` written.

### Task 2: Refresh Backend Hardware Cache

**Action:** Restart the addon to clear the HardwareService cache so it re-reads the filesystem type:

```text
ha_stop_addon slug=local_sambanas2
wait ~5 s
ha_start_addon slug=local_sambanas2
```

> Plain stop/start is safe. Only full container recreation resets `/usr/local/bin`; see deployment notes.

**Expected:**

- After restart, `GET /api/volumes` reports `sdc1` with `fs_type:ext4` and label TESTMOUNT.

### Task 3: Select Partition in the UI

**Action:** Open the frontend at `http://localhost:3080/`, navigate Volumes page, click the disk tree item to expand, then select partition `TESTMOUNT`.

> UI automation note: selection requires clicking the inner content Box (`li[id*="part"] .MuiTreeItem-label > div`); clicking the label itself only toggles expansion.

**Expected:**

- Detail panel shows `TESTMOUNT`, device `sdc1`, filesystem ext4, state "Not Mounted".
- Action buttons visible: ENABLE AUTOMATIC MOUNT, MOUNT PARTITION, CHECK FILESYSTEM, SET LABEL, FORMAT PARTITION.

### Task 4: Configure and Execute Mount

**Action:**

1. Click **MOUNT PARTITION**—the `VolumeMountDialog` opens inline (not a modal `[role=dialog]`).
2. In the "File System Type" autocomplete, type `ext4` and press Enter (selects the option and submits).
3. Leave mount path at the default suggestion.

> UI automation note: MUI Autocomplete resists synthetic clicks; typing + Enter reliably selects and submits.

**Expected:**

- Dialog closes; success toast appears; the volume tree shows a "Mounted" badge on TESTMOUNT.

### Task 5: Verify Mount End-to-End

**Action:** Verify at three levels:

```bash
# 1. API state
ssh root@192.168.0.68 "docker exec app_local_sambanas2 curl -sL http://localhost:64289/api/volumes" | grep -A5 minchia_signor_tenente
# 2. Kernel mount table
ssh root@192.168.0.68 "docker exec app_local_sambanas2 mount" | grep sdc1
# 3. Addon logs
ssh root@192.168.0.68 "docker logs app_local_sambanas2 --since 10m 2>&1 | grep -i 'mount'"
```

**Expected:**

- API: `mount_point_data` contains `{path:/mnt/<label-or-config-path>, fstype:ext4, is_mounted:true}`.
- Kernel: `/dev/sdc1 on /mnt/... type ext4 (rw,relatime)`.
- Logs: `service/volume_mount_manager.go:106 Successfully mounted volume ... mount_fstype=ext4` followed by `share_service.go:138 Received MountPointEvent type=update IsMounted:true`.
- `df` shows the mounted capacity (~56 GB usable).

## Observations / Potential Issues

1. HardwareService cache staleness after out-of-band formatting requires an addon restart to reflect `fs_type`—known behavior, documented in skill troubleshooting.
2. If a stale `mount_point_data` config exists from previous experiments (invalid fstype etc.), the dialog pre-fills it; automatic detection still works when the field is left blank.
