# Test 05: User CRUD Functionality

## Overview

Validates the full Samba user lifecycle through the Users page UI: create a user with password and share assignments, edit share assignments and password, delete the user—verified at API, Samba password database (`pdbedit`), and SMB client authentication levels.

| Field | Value |
| --- | --- |
| Version tested | v2026.8.0-rc13.1 (git describe `2026.8.0-rc13-16-gc5d20970`) |
| Environment | HA test host `192.168.0.68`, addon `local_sambanas2`, lab mode OFF |
| Result | PASS—**with one bug found** (password re-entry required on edit) |

## Prerequisites

- Baseline users: `admin` only (`GET /api/users`); `pdbedit -L` shows `_ha_mount_user_:1000` and `admin:1001`.
- Note: `smbpasswd -L` is broken in the addon container; use `pdbedit` for all password-database checks.

## Reproduction Steps

### Task 1: Create a User

**Action:**

1. Navigate to Users page; click **Create new user** (`#create_new_user`).
2. Username: `TESTUSER`; Password / Repeat Password: `TestPass123!`.
3. In Read/Write Shares combobox type `TEST_NFS`, click the matching option.
4. Click **CREATE USER**.

**Expected:** triple-level verification passes:

```bash
# 1. API
ssh root@192.168.0.68 "docker exec app_local_sambanas2 curl -sL http://localhost:64289/api/users"
# → TESTUSER { is_admin:null, is_valid:true, rw_shares:["TEST_NFS"], ro_shares:[] }
# 2. Samba password DB
ssh root@192.168.0.68 "docker exec app_local_sambanas2 pdbedit -L"
# → TESTUSER:1002 present
# 3. Real authentication
ssh root@192.168.0.68 "docker exec app_local_sambanas2 smbclient -L localhost -U 'TESTUSER%TestPass123!'"
# → share list returned successfully
```

### Task 2: Edit the User (BUG FOUND)

**Action:**

1. Select TESTUSER in the users tree; click **Edit User**.
2. Without entering a password, add `Carola` to Read-Only Shares and click **SAVE CHANGES**.
3. Observe the failure, then fill Password / Repeat Password (`NewPass456!`) and save again.

**Expected vs actual:**

| Step | Expected | Actual |
| --- | --- | --- |
| Save without password | Share change saved; password untouched | ❌ HTTP 422 `{"detail":"Password is required"}`; console shows `[object Object]`; **no user-facing error toast** |
| Save with password | Change saved | ✅ Saved: `ro_shares:["Carola"]`; old password rejected (`NT_STATUS_LOGON_FAILURE`), new password authenticates successfully |

**BUG:** Editing an existing user forces password re-entry—the backend PUT rejects empty passwords even when only changing shares. The frontend surfaces nothing visible (console-only `[object Object]`).

### Task 3: Delete the User

**Action:**

1. With TESTUSER selected, click **Delete User**.
2. Confirm dialog appears with acknowledgement text: "I understand that deleting the user will remove it permanently."
3. Check the acknowledgement checkbox, click **OK**.

**Expected:**

- `GET /api/users` returns to baseline (`admin` only).
- `pdbedit -L` no longer lists TESTUSER.

```bash
ssh root@192.168.0.68 "docker exec app_local_sambanas2 pdbedit -L" | grep TESTUSER || echo "USER_REMOVED"
```

Note: `smbclient` auth with deleted credentials may still succeed because `smb.conf` contains `map to guest = Bad User`—unknown usernames map to guest access. This is expected Samba behavior, not a deletion bug (bogus credentials authenticate identically).

## Observations / Potential Issues

1. **Bug (actionable):** password re-entry required on user edit; silent 422 without toast. Recommend: allow empty password to mean "keep existing", and surface API errors as alerts.
2. Username field correctly disabled in edit mode ("Username cannot be changed").
