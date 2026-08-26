<!-- DOCTOC SKIP -->

# [FEATURE]: Reuse Home Assistant Dropbox Integration OAuth for Rclone Cloud Sync

**Target Repo:** `srat`
**Status:** ✅ Complete (shipped with 049 — HA Dropbox reuse)
**Issue Link:** https://github.com/dianlight/srat/issues/954

## 🎯 Objective

Let the rclone cloud-sync wizard authorize Dropbox by **reusing the OAuth token already held by the user's Home Assistant Dropbox integration** (`homeassistant/components/dropbox` since 2026.4), instead of asking for custom app credentials. The wizard's "Reuse Dropbox integration auth" option is now enabled when `ha_dropbox_available` is true.

> Hosted broker reuse is **not part of this task** — tracked in **issue #1002**.

## 🛠️ Technical Specifications

- **Inputs:** The HA Dropbox integration config entry (access + refresh token + expiry) read from `hass.config_entries` by the SRAT custom component (`custom_components/srat/ha_dropbox.py`), forwarded to the backend over the existing WebSocket-only channel as `ha_dropbox_token` (`dto.HaDropboxTokenMessage`).
- **Outputs:** A populated managed `rclone.conf` remote (`config/create`) with the reused token envelope; link row transitions to `authorized` without a browser redirect (no `state`/`callback` dance).
- **Dependencies:**
  - `custom_components/srat` websocket client (`async_send_ha_dropbox_token`, `ha_dropbox.py`)
  - `backend/src/service/rclone_service.go` (third branch `ha_dropbox` in `StartAuth` — `startHaDropboxAuth` — beside `custom_app`; `broker` lives in `broker.go` but spec is in #1002)
  - `backend/src/dto/rclone.go` (`ha_dropbox_available` in `RcloneProvidersResponse`) and `dto/ha_dropbox_token.go`
  - `backend/src/api/{rclone_handler.go,ws.go}` (providers flag, WS `ha_dropbox_token` inbound, `auth_mode` enum `auto,custom_app,broker,ha_dropbox`)
  - Dropbox token refresh binding: the refresh token stays bound to whichever app created it (HA Cloud Linking vs custom app) — same trade-off as broker mode, documented in `service/rclone/broker.go` header

## 📝 Task List

- [x] Task 1: Custom component — read the Dropbox integration config entry data and expose it via a typed WS message (`ha_dropbox.py`, `websocket_client.py:async_send_ha_dropbox_token`, `__init__.py` push on WS connect)
- [x] Task 2: Backend — accept the reused-token payload via WebSocket `ha_dropbox_token` and cache it single-use in `RcloneService` (`HaDropboxAvailable`/`SetHaDropboxToken`, `dto.HaDropboxTokenMessage.Validate`)
- [x] Task 3: Backend — complete the flow without a browser redirect when `auth_mode=ha_dropbox` (`startHaDropboxAuth` → `config/create`, clear state, set `authorized`)
- [x] Task 4: Frontend — enable the "Reuse Dropbox integration auth" option only when `ha_dropbox_available` (and provider is `dropbox`); wire submit path (`auth_mode: ha_dropbox`, no credentials, browser origin still sent for `redirect_uri` echo)
- [x] Task 5: Unit testing backend (suite+fx+mockio) and frontend (MSW) including failure paths (no entry, revoked token) — `service/rclone/broker_test.go`, `rclone_service_test.go`, `rclone_handler_test.go`, `CloudLinkWizardDialog.test.tsx`
- [x] Task 6: Documentation — task 049 OAuth section points to #1002 for broker and to this doc for HA reuse; `broker.go` header documents HA parallel + PKCE nuance
- [x] Task 7: Code review, coverage gate ≥70% on touched functions, final validation (`mise run //backend:test`, `mise run //frontend:generate`)

## 🧠 Implementation Notes (Copilot Context)

- HA Dropbox uses by default **Home Assistant Cloud Account Linking** (`accounts.home-assistant.io`, Nabu Casa proxy holding `client_id`/`client_secret`) with fan-out via `my.home-assistant.io/redirect/oauth` or cloudhook; `Application Credentials` is the bring-your-own-app alternative. Third-party addons cannot reuse HA's credentials — this task reuses the *token* instead. See `service/rclone/broker.go` header for full HA parallel.
- PKCE (no secret) does not remove the redirect-URI deploy — bottleneck is the pre-registered redirect URI, not the secret (see broker.go header).
- Mode is additive: `auth_mode` is `auto|custom_app|ha_dropbox` for this task (`broker` is defined but spec lives in #1002); extend the enum rather than inventing a parallel mechanism.

## 🔗 Code References & TODOs

- Wizard option: `frontend/src/pages/volumes/components/rclone/CloudLinkWizardDialog.tsx` (`ha_dropbox` value, enabled when `ha_dropbox_available`)
- Mode switch: `startHaDropboxAuth` / `HaDropboxAvailable` / `SetHaDropboxToken` in `backend/src/service/rclone_service.go`
- WS inbound: `backend/src/api/ws.go` (`CLIENTEVENTTYPEHADROPBOXTOKEN`), DTO: `backend/src/dto/ha_dropbox_token.go`
- Custom component: `custom_components/srat/ha_dropbox.py` + `websocket_client.py`
- Related: `docs/tasks/049_rclone-cloud-sync-lab-feature.md` (OAuth modes overview), issue #1002 (broker)
