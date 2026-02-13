# Missing Home Assistant Integration Data

<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

This document tracks data that the SRAT custom component currently lacks
or cannot expose as Home Assistant entities because the backend WebSocket
events do not provide it.

All communication between the custom component and the SRAT backend uses
a **single WebSocket connection** (`/ws`). The two events that carry
sensor data are:

| Event       | DTO          | Description                                                                                       |
| ----------- | ------------ | ------------------------------------------------------------------------------------------------- |
| `volumes`   | `[]*Disk`    | Full list of disks and their partitions                                                           |
| `heartbeat` | `HealthPing` | Periodic health snapshot (samba status, process status, disk health, network health, addon stats) |

## Currently Exposed Data

| Sensor                           | Source Event | HealthPing Field               | Status       |
| -------------------------------- | ------------ | ------------------------------ | ------------ |
| Volume Status                    | `volumes`    | —                              | ✅ Available |
| Disk (per device)                | `volumes`    | —                              | ✅ Available |
| Partition (per partition)        | `volumes`    | —                              | ✅ Available |
| Samba Status                     | `heartbeat`  | `samba_status`                 | ✅ Available |
| Samba Process Status             | `heartbeat`  | `samba_process_status`         | ✅ Available |
| Global Disk Health               | `heartbeat`  | `disk_health`                  | ✅ Available |
| Disk IO (per device)             | `heartbeat`  | `disk_health.disk_io`          | ✅ Available |
| Partition Health (per partition) | `heartbeat`  | `disk_health.partition_health` | ✅ Available |

## Missing / Not Yet Exposed Data

The following data is available in the WebSocket events but is **not yet
exposed** as Home Assistant entities or attributes.

### From `heartbeat` → `HealthPing`

| Field              | Type               | Potential Entity       | Notes                                              |
| ------------------ | ------------------ | ---------------------- | -------------------------------------------------- |
| `alive`            | `bool`             | Binary sensor          | Whether the SRAT backend is alive                  |
| `aliveTime`        | `int64`            | Sensor (timestamp)     | Backend start time as Unix epoch                   |
| `uptime`           | `int64`            | Sensor (duration)      | Backend uptime in seconds                          |
| `last_error`       | `string`           | Sensor (diagnostic)    | Last error message from backend                    |
| `update_available` | `bool`             | Update / Binary sensor | Whether an addon update is available               |
| `dirty_tracking`   | `DataDirtyTracker` | Sensor / Attributes    | Tracks unsaved configuration changes               |
| `addon_stats`      | `AddonStatsData`   | Sensor                 | CPU, memory, network stats for the addon container |
| `network_health`   | `NetworkStats`     | Sensor                 | Network interface statistics                       |

### From `hello` → `Welcome`

| Field              | Type       | Potential Entity     | Notes                                                    |
| ------------------ | ---------- | -------------------- | -------------------------------------------------------- |
| `message`          | `string`   | —                    | Welcome message (informational only)                     |
| `active_clients`   | `int32`    | Sensor               | Number of active WebSocket clients                       |
| `supported_events` | `[]string` | Diagnostic attribute | Events the server supports                               |
| `update_channel`   | `string`   | Sensor / Select      | Current update channel (None/Develop/Release/Prerelease) |
| `machine_id`       | `string`   | Device attribute     | Machine identifier                                       |
| `build_version`    | `string`   | Device attribute     | Backend build version                                    |
| `secure_mode`      | `bool`     | Binary sensor        | Whether secure mode is enabled                           |
| `protected_mode`   | `bool`     | Binary sensor        | Whether protected mode is enabled                        |
| `read_only`        | `bool`     | Binary sensor        | Whether the system is in read-only mode                  |
| `startTime`        | `int64`    | Sensor (timestamp)   | Backend start time                                       |

### From `shares` → `[]SharedResource`

| Field      | Type               | Potential Entity | Notes                                                                            |
| ---------- | ------------------ | ---------------- | -------------------------------------------------------------------------------- |
| Share list | `[]SharedResource` | Sensor per share | Individual Samba share entities with attributes (path, permissions, users, etc.) |

### From `dirty_data_tracker` → `DataDirtyTracker`

| Field       | Type               | Potential Entity | Notes                                     |
| ----------- | ------------------ | ---------------- | ----------------------------------------- |
| Dirty flags | `DataDirtyTracker` | Binary sensor    | Whether configuration has unsaved changes |

### From `smart_test_status` → `SmartTestStatus`

| Field              | Type              | Potential Entity | Notes                                    |
| ------------------ | ----------------- | ---------------- | ---------------------------------------- |
| SMART test results | `SmartTestStatus` | Sensor per disk  | S.M.A.R.T. self-test status and progress |

### From `updating` → `UpdateProgress`

| Field           | Type             | Potential Entity | Notes                                       |
| --------------- | ---------------- | ---------------- | ------------------------------------------- |
| Update progress | `UpdateProgress` | Sensor           | Addon update progress percentage and status |

### From `error` → `ErrorDataModel`

| Field         | Type             | Potential Entity                | Notes                               |
| ------------- | ---------------- | ------------------------------- | ----------------------------------- |
| Error details | `ErrorDataModel` | Event / Persistent notification | Backend error with code and message |

## Implementation Priority

1. **High** – `shares` (SharedResource entities), `update_available`, `addon_stats`
2. **Medium** – `alive`/`uptime` binary sensor, `network_health`, `smart_test_status`
3. **Low** – `hello` metadata (version, machine_id), `dirty_data_tracker`, `error` events

## How to Add a New Sensor

1. Register a new listener in `coordinator.py` for the relevant event type.
2. Extract the data in the callback and store it under a new key in `self.data`.
3. Create a new sensor class in `sensor.py` extending `SRATSensorBase`.
4. Add the sensor to the `async_setup_entry` entity list.
5. Run `cd custom_components && make check` to validate.
