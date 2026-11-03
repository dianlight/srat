<!-- DOCTOC SKIP -->

# [FEATURE]: Support HA 2026.11 native USB drive mounting

**Target Repo:** `srat`
**Status:** 📅 Planned
**Issue Link:** [srat#1530](https://github.com/dianlight/srat/issues/1530), [hassio-addons#807](https://github.com/dianlight/hassio-addons/issues/807)

## 🎯 Objective

Investigate and decide whether SambaNAS2 should leverage Home Assistant 2026.11's native USB drive mounting (see https://rc.home-assistant.io/blog/2026/10/28/release-202611/#store-backups-and-media-on-local-drives) instead of mounting drives inside the addon. The goal is to tighten addon permissions/security by letting the OS (HA) own drive mounting while SambaNAS2 focuses on share management, if feasible and supported.

> _Context for Copilot: Open question from upstream user @jondycz. Addon currently mounts drives with elevated privileges; HA 2026.11 can mount USB drives natively. Need to assess feasibility, permission model, impact on automount/udev handling, and migration path for existing setups. Decide preferred direction._

## 🛠️ Technical Specifications

- **Inputs:** HA 2026.11 host-mounted drive paths exposed to addon; existing `MountPointEvent` / automount flow
- **Outputs:** Decision + (if adopted) adapter allowing SambaNAS2 to manage shares over HA-mounted paths without performing its own mounts
- **Dependencies:** `backend/src/service/volume/*`, `backend/src/service/volume_service.go`, addon config (`config.yaml`), `custom_components/srat` (if host paths change)

## 📝 Task List

- [ ] Task 1: Research HA 2026.11 native mounting API/paths and addon permission requirements
- [ ] Task 2: Assess impact on existing automount, udev, and permission code
- [ ] Task 3: Decide preferred approach (native vs addon mounting) and document
- [ ] Task 4: If adopted, implement + tests; else document why not supported

## 🧠 Implementation Notes (Copilot Context)

- Upstream: hassio-addons#807 (enhancement, [SambaNas2]).
- Do not break existing setups; consider migration path and backward compatibility.
- Security: leaving mounting to OS may reduce addon privileges; verify share management still works over host-mounted paths.

## 🔗 Code References & TODOs

- [ ] `hassio-addons#807` — Mounting drives in HA 2026.11
- [ ] `srat#1530` — mirror closes once tracked here
