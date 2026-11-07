# Design

## Context

Fix transient `Device 'ata-...' not found during startup` on soft add-on Restart. The by-id symlink (or Supervisor hardware snapshot) is not yet visible when the first automount runs, the failure is terminal (no retry), and only a full host reboot recovers.

## Goals / Non-Goals

**Goals:**

- Deliver: Addon Restart Device-Not-Found Race

**Non-Goals:**

- Unrelated refactoring outside the task scope.

## Decisions

- Create task doc with hassio-addons#767 + srat#1266 links
- Failing test — DeviceNotFound schedules HW invalidate + delayed refresh
- Fix HandleMountPointEvent DeviceNotFound path + ScheduleDeviceNotFoundRetry
- Run backend tests + coverage gate, hk check

## Risks / Trade-offs

- Behaviour changes are covered by targeted tests before handoff.
- Legacy task content remains the reference until the change is archived.
