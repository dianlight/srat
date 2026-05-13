# [FIX]: Time Machine Compatibility with macOS Tahoe and Later

**Target Repo:** `srat`  **Status:** ✅ Complete  **Issue Link:** https://github.com/dianlight/srat/issues/526 (refs [hassio-addons#536](https://github.com/dianlight/hassio-addons/issues/536))

## 🎯 Objective

Resolve Time Machine backup failures on macOS Tahoe (macOS 15+) connecting to SambaNAS2. Users report that backups start but disconnect mid-transfer with "The network disk disconnected from your Mac while backing up." The root cause is likely a protocol negotiation change introduced in macOS Tahoe that requires updated Samba settings for Apple Extensions (`fruit:*`) compatibility.

> _Context for Copilot: Samba Time Machine support relies on `vfs objects = fruit streams_xattr` and the `fruit:` parameter set in share sections of `smb.conf`. The template is in `backend/src/templates/smb.gtpl`. Time Machine capability is tracked in `dto.MountStruct.TimeMachineSupport`. macOS Tahoe tightened SMB signing and negotiation requirements._

## 🛠️ Technical Specifications

- **Inputs:**
  - Share with `TimeMachineSupport = "supported"`
  - Samba configuration generated from `smb.gtpl`
  - macOS client using SMB3 with signing (Tahoe+ default)

- **Outputs:**
  - Updated `smb.conf` parameters that satisfy macOS Tahoe Time Machine requirements
  - Optional: backend setting to control Samba log level for Time Machine debugging
  - Diagnostic endpoint or log helper that outputs current Samba Time Machine config

- **Dependencies:**
  - `backend/src/templates/smb.gtpl` — Samba share template
  - `backend/src/service/server_process_service.go` — template data preparation
  - `backend/src/dto/` — `MountStruct.TimeMachineSupport`
  - `docs/SMB_OVER_QUIC.md` — existing SMB protocol docs (for context)

## 📝 Task List

- [x] Task 1: Research and document the Samba parameters required for macOS Tahoe Time Machine compatibility — focus on `fruit:model`, `fruit:metadata`, `fruit:posix_rename`, `server signing`, `smb3 unix extensions`
- [x] Task 2: Update `smb.gtpl` Time Machine share block with the verified parameter set (add any missing `fruit:` options, ensure `vfs objects` order is correct)
- [x] Task 3: Add global `smb.conf` options for SMB signing compatibility: `server signing = auto` (or `required` if needed) and `ntlm auth = ntlmv2-only`
- [x] Task 4: Add a documentation page `docs/TIMEMACHINE_COMPATIBILITY.md` covering the required Samba parameters and macOS version compatibility matrix
- [ ] Task 5: Unit tests — template rendering with `TimeMachineSupport = "supported"`: verify `fruit:` section is present and contains required keys
- [ ] Task 6: Regression test — template rendering with `TimeMachineSupport = "unsupported"`: verify `fruit:` section is absent
- [ ] Task 7: Add a note on UI on Time Machine share switch if Compatibility mode is enabled that it may not work with macOS 15+. Also add a note con Compatibility Switch on UI that it can cause issues with macOS 15+ Time Machine backups.

## 🧠 Implementation Notes (Copilot Context)

#### [2026-03-25] Researched Samba parameters for macOS 15 (Tahoe) Time Machine compatibility

**Recommended global (smb.conf) options:**
- vfs objects = fruit streams_xattr
- fruit:aapl = yes
- fruit:model = MacSamba (or another Mac model string for cosmetic icon)
- fruit:metadata = stream
- fruit:veto_appledouble = no
- fruit:nfs_aces = no
- fruit:wipe_intentionally_left_blank_rfork = yes
- fruit:delete_empty_adfiles = yes
- fruit:posix_rename = yes (needed for Time Machine on Samba <4.23, safe to keep for compatibility)
- fruit:zero_file_id = yes
- min protocol = SMB2 (or higher, e.g., SMB3)
- ea support = yes
- server signing = auto (or required if needed for Tahoe)
- ntlm auth = ntlmv2-only

**Time Machine share options:**
- fruit:time machine = yes
- fruit:time machine max size = <SIZE> (optional, to limit backup size)

**Rationale:**
- These options ensure correct Apple SMB extensions, metadata handling, and signing requirements for macOS 15+.
- fruit:posix_rename is still recommended for compatibility, though Samba 4.23+ may not require it.
- server signing and ntlm auth are critical for SMB3.1.1 negotiation in Tahoe.
- The order of vfs objects is important: always use fruit before streams_xattr.


---
#### Pre-Implementation Plan (2026-03-25)

**Objective & Acceptance Criteria**
- Ensure backups complete without disconnects or errors.
- All required Samba parameters for Time Machine/macOS compatibility are present and correct in generated configs.

**Impacted Files/Components**
- `backend/src/templates/smb.gtpl` (Samba config template)
- `backend/src/service/server_process_service.go` (template data prep)
- `backend/src/dto/` (TimeMachineSupport field)
- `docs/TIMEMACHINE_COMPATIBILITY.md` (new doc)
- Unit/regression tests for template rendering

**Step-by-Step Plan**
1. Research & document required Samba parameters for macOS Tahoe (fruit:*, signing, SMB3, etc.).
2. Update `smb.gtpl` Time Machine share block with all required/verified parameters.
3. Add/verify global `server signing` and `ntlm auth` options.
4. Create `docs/TIMEMACHINE_COMPATIBILITY.md` with config matrix and guidance.
5. Add/expand unit and regression tests for template rendering (TimeMachineSupport supported/unsupported).
6. (Optional) Add backend log/endpoint for current Samba Time Machine config.

**Test/Validation Strategy**
- Unit tests: Template renders correct config for both supported/unsupported Time Machine shares.
- Manual/CI: Validate config against macOS Tahoe client (if possible).
- Documentation review: Ensure doc covers all required parameters and compatibility notes.

**Risks & Edge Cases**
- Samba version differences (older versions may not support all fruit: options).
- macOS client-side changes in future releases.
- Unintended config changes affecting non-Time Machine shares.

---

### Known working Samba Time Machine parameters (macOS Ventura/Sonoma/Tahoe)

```ini
[TIMEMACHINE]
  path = /mnt/backup
  vfs objects = catia fruit streams_xattr
  fruit:time machine = yes
  fruit:model = MacPro7,1
  fruit:metadata = stream
  fruit:posix_rename = yes
  fruit:zero_file_id = yes
  fruit:wipe_intentionally_left_blank_rfork = yes
  fruit:delete_empty_adfiles = yes
  read only = no
  inherit acls = yes
```

### Global options for Tahoe SMB signing

```ini
[global]
  server signing = auto
  ntlm auth = ntlmv2-only
  smb3 unix extensions = no
```

### `fruit:model` parameter

Setting `fruit:model = MacPro7,1` causes macOS to display the Time Machine destination as a "Mac Pro" icon. This is cosmetic but helps identify the share in macOS Time Machine preferences. Alternative: `RackMac3,1` or `TimeCapsule8,119`.

### macOS Tahoe changes

macOS Tahoe (15.x) requires SMBv3.1.1 with AES-128-GCM encryption for mDNS-advertised shares. Ensure Samba is compiled with `--with-system-mitkrb5` or equivalent, and that `min protocol = SMB3` is set.

## 🔗 Code References & TODOs

- [ ] [hassio-addons#536](https://github.com/dianlight/hassio-addons/issues/536) — macOS Tahoe Time Machine backup disconnects after upgrade from Sequoia
- [ ] `backend/src/templates/smb.gtpl` — Time Machine share `vfs objects` and `fruit:` parameters
- [ ] `backend/src/service/server_process_service.go` — template data including `TimeMachineSupport`
- [ ] `backend/src/dto/` — `MountStruct.TimeMachineSupport` (existing field)
- [ ] `docs/SMB_OVER_QUIC.md` — related Samba protocol documentation
