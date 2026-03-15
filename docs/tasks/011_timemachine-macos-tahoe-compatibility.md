# [FIX]: Time Machine Compatibility with macOS Tahoe and Later

**Target Repo:** `srat`  **Status:** 📅 Planned  **Issue Link:** [hassio-addons#536](https://github.com/dianlight/hassio-addons/issues/536)

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

- [ ] Task 1: Research and document the Samba parameters required for macOS Tahoe Time Machine compatibility — focus on `fruit:model`, `fruit:metadata`, `fruit:posix_rename`, `server signing`, `smb3 unix extensions`
- [ ] Task 2: Update `smb.gtpl` Time Machine share block with the verified parameter set (add any missing `fruit:` options, ensure `vfs objects` order is correct)
- [ ] Task 3: Add global `smb.conf` options for SMB signing compatibility: `server signing = auto` (or `required` if needed) and `ntlm auth = ntlmv2-only`
- [ ] Task 4: Expose a `log_level` or `samba_log_level` setting in `AppConfig` that maps to Samba's `log level` directive — enables users to diagnose Time Machine issues without editing raw config
- [ ] Task 5: Add a documentation page `docs/TIMEMACHINE_COMPATIBILITY.md` covering the required Samba parameters and macOS version compatibility matrix
- [ ] Task 6: Unit tests — template rendering with `TimeMachineSupport = "supported"`: verify `fruit:` section is present and contains required keys
- [ ] Task 7: Regression test — template rendering with `TimeMachineSupport = "unsupported"`: verify `fruit:` section is absent

## 🧠 Implementation Notes (Copilot Context)

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

### Diagnosing disconnects

Users can set `log level = 3 vfs:10` in `smb.conf` to capture fruit VFS events. Adding a configurable `samba_log_level` field in `AppConfig` (default `1`) mapped directly to `log level = {{ .SambaLogLevel }}` in `smb.gtpl` would allow runtime debugging without a UI change.

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
