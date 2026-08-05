# [REFACTOR]: Simplify mDNS Settings — Master/Proxy Model

**Target Repo:** `srat`
**Status:** ✅ Complete
**Issue Link:** [Optional]

## 🎯 Objective

Promote the "Addon-side Direct mDNS" feature out of Lab Mode and simplify the
mDNS configuration model:

1. Rename `mdns_registration` → `use_component_mdns_proxy` (implementation
   choice: HA custom component proxy vs. SRAT direct zeroconf).
2. Rename `addon_mdns_registration` → `mdns_registration` — the new **master
   switch** that enables/disables mDNS registration entirely.
3. Remove the mutual-exclusivity validation; the master switch gates
   everything, the proxy switch selects the implementation.
4. UI stays transparent: both switches now live in the General section (master)
   and Home Assistant section (proxy) of the setup flow, no longer gated behind
   `experimental_lab_mode`.

## 🧠 Semantics

| master `mdns_registration` | `use_component_mdns_proxy` (nil → true) | Behavior |
|---|---|---|
| false | any | no mDNS/Samba announce |
| true | true | HA component proxy (old `mdns_registration=true`) |
| true | false | SRAT direct zeroconf (old `addon_mdns_registration=true`) |

## 📝 Task List

- [x] Rename DTO setting `AddonMDNSRegistration` → `UseComponentMDNSProxy *bool` (`default:"true"`); `MDNSRegistration` becomes the master switch (`backend/src/dto/settings.go`)
- [x] Mirror change in addon JSON config (`backend/src/config/addon_json_config.go`)
- [x] Remove mutual-exclusivity block from `ValidateSettings` (`backend/src/service/setting_service.go`)
- [x] Re-gate mDNS broadcast/direct registration on master+proxy (`backend/src/service/mdns_service.go`)
- [x] Add DB migration `00018_migrate_mdns_properties.go` + sqlmock tests in `migrations_guard_test.go`
- [x] Regenerate OpenAPI (`backend/docs/openapi.*`), converter (`config_to_dto_conv_gen.go`), frontend types (`frontend/src/store/sratApi.ts`)
- [x] Frontend: General panel "Samba mDNS Announce" master switch; Home Assistant panel "Use Home Assistant mDNS Proxy" switch (`GeneralPanel.tsx`, `HomeAssistantPanel.tsx`, `settingsConfig.ts`)
- [x] Rewrite frontend panel tests (`GeneralPanel.switches.test.tsx`, `HomeAssistantPanel.mdns.test.tsx`)
- [x] Update `docs/SETTINGS_DOCUMENTATION.md` + TOC
- [x] Update `CHANGELOG.md` Unreleased entry
- [x] Full validation: `mise run //backend:test`, `mise run //frontend:test`, `hk check`, `mise run docs-validate` — all green

## 🧠 Implementation Notes

- Master switch gates every mDNS path; `use_component_mdns_proxy == nil` is
  treated as `true` (default to HA component proxy).
- `shutdownAddonMDNS` renamed to `shutdownDirectMDNS`; `reconfigureAddonMDNS`
  renamed to `reconfigureDirectMDNS`.
- No mutual exclusivity remains — both switches can be on/off independently.

## ⚠️ Out of Scope / Side Effects

- Pre-existing goenums v0.7.0 regeneration WIP (all `backend/src/dto/*_enums.go`,
  `backend/src/events/eventtypes_enums.go`, `backend/src/go.mod`) was folded in;
  `MarshalText` now returns unquoted text per `encoding.TextMarshaler`,
  requiring updates to 4 stale expectations in `backend/src/dto/enums_test.go`.
