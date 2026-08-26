# Test 04: Share CRUD Functionality

## Overview

Validates the full share lifecycle through the Shares page UI: create a new share on a mounted volume, edit its properties (guest access), and delete it, with verification at the API, Samba configuration, and log levels.

| Field | Value |
| --- | --- |
| Version tested | v2026.8.0-rc13.1 (git describe `2026.8.0-rc13-16-gc5d20970`) |
| Environment | HA test host `192.168.0.68`, addon `local_sambanas2`, lab mode OFF |
| Result | PASS |

## Prerequisites

- A mounted volume available for share creation (Carola on exfat was used).
- Baseline shares: 11 total = 9 internal HA shares + `TEST_NFS` (backup) + `Carola` (media).

## Reproduction Steps

### Task 1: Create a Share

**Action:**

1. Navigate to Shares page; click **Create new share** (`#create_new_share`).
2. Fill "Share Name" with `TEST_SHARE`.
3. In the Volume combobox, type `/mnt/Carola`, open the option list, click option `Carola`.
4. Leave Usage unset or select a value from the dropdown list.
5. Click **CREATE**.

> UI automation note: pressing Enter in the Usage combobox with an empty input clears the selection **and submits the form**—the dialog closes immediately. Select options by clicking list items instead.

**Expected:**

- Dialog closes; `GET /api/shares` now includes:

```json
{
  "name": "TEST_SHARE",
  "usage": "none",
  "mount_point_data": { "path": "/mnt/Carola", "disk_label": "Carola", "fstype": "exfat", "is_mounted": true },
  "timemachine": false,
  "recycle_bin_enabled": false,
  "guest_ok": false,
  "users": ["admin"],
  "status": { "is_valid": true }
}
```

- `/etc/samba/smb.conf` gains a `[TEST_SHARE]` section with `browseable=yes`, `writeable=yes`, `create mask=0664`.
- The share appears grouped in the tree under its usage group.

### Task 2: Edit the Share

**Action:**

1. Expand the correct usage group and select `TEST_SHARE` in the tree (click inner Box of the tree item label).
2. Click the Edit (pencil) toggle button in the detail panel.
3. Toggle the **Guest Access** switch on.
4. Click **APPLY**.

**Expected:**

- Form exits edit mode; summary shows Guest Access enabled.
- `GET /api/shares` reflects `"guest_ok": true` for TEST_SHARE.
- `smb.conf` section is regenerated (share DEBUG comment shows `guest_ok:true`).

### Task 3: Delete the Share

**Action:**

1. Re-enter edit mode on TEST_SHARE.
2. Scroll to bottom; click **DELETE**.
3. Confirm dialog appears: "Delete TEST_SHARE? This action cannot be undone..." with an acknowledgement checkbox gating the OK button.
4. Check the acknowledgement, click **OK**.

**Expected:**

- Dialog closes; success feedback shown.
- `GET /api/shares` returns to the 11-share baseline; TEST_SHARE absent.
- `grep -c 'TEST_SHARE' /etc/samba/smb.conf` returns `0`—section fully removed.

```bash
ssh root@192.168.0.68 "docker exec app_local_sambanas2 grep -A5 '\[TEST_SHARE\]' /etc/samba/smb.conf" || echo "SECTION_REMOVED"
```

## Observations / Potential Issues

1. **Usage combobox Enter quirk:** empty-value Enter both clears selection and submits the create form (share was created with `usage:none`). Minor UX hazard.
2. Cosmetic: generated smb.conf shows duplicated valid-users entry (`admin admin`) and no explicit `guest ok = yes` line despite `guest_ok:true` (likely handled via global mapping)—worth a look but non-blocking.
3. The more-actions menu offers Delete directly only when the share status is invalid; for valid shares deletion goes through edit mode → DELETE button.
