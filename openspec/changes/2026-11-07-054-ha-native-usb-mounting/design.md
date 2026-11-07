# Design

## Context

Task 054 requested support for HA 2026.11 native USB drive mounting. The chosen direction was a compatibility layer first: detect HA-managed mounts and expose them as pre-mounted `HOST` volumes while keeping Srat's own mount path for everything else. No addon privilege change in this phase.

## Goals / Non-Goals

- Goals: detect HA-managed mounts, annotate them, guard Srat operations, annotate shares, expose UI badge and hide actions.
- Non-Goals: version gating (tracked in change 055), user-configurable mode (tracked in change 055), reduced-privilege addon profile (tracked in `dianlight/hassio-addons`).

## Decisions

- Detection uses the procfs mount path under `/share/`, `/media/`, `/backup/` combined with a block-device source match, because SRAT always mounts under `/mnt`. This avoids relying on the Supervisor `/mounts` API while the mount is visible inside the addon namespace.
- `IsHAMounted` is read-only and derived at runtime (never persisted); re-derived on DB load and from the path on cache miss.
- `MountVolume` derives the flag from both the path and the cached mount point, so a client that omits the read-only field is still rejected.
- UI hides mount/automount/format/label/unmount for HA-managed partitions but keeps Create Share / Go to Share.

## Risks / Trade-offs

- Detection depends on the HA mount being visible inside the addon namespace (`map: share/media/backup`); verify on a live HA instance.
- `hardware/info` may or may not report the mounted partition; SMART correlation may need `/host/disks` fallback.
- `off`/`auto`/`on` mode control is not implemented yet (change 055).

## Open Questions

- Does nested mount propagation make a disk mounted under the host `/share` visible inside the addon? (Needs live verification.)
- Does `/hardware/info` list a filesystem whose partition is mounted by HA?