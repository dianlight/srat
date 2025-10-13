# Home Assistant Integration

<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->

- [Entities Created](#entities-created)
  - [Volume Status Entity](#volume-status-entity)
  - [Disk Entities](#disk-entities)
  - [Partition Entities](#partition-entities)
  - [Samba Status Entity](#samba-status-entity)
  - [Samba Process Status Entity](#samba-process-status-entity)
- [How It Works](#how-it-works)
- [Configuration](#configuration)
- [Usage Example](#usage-example)
- [Example Home Assistant Dashboard](#example-home-assistant-dashboard)
- [Troubleshooting](#troubleshooting)
- [Limitations](#limitations)
- [Entity IDs](#entity-ids)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

When SRAT has a configured Home Assistant Core API client, it will automatically create and update Home Assistant entities via the Core State API.

## Entities Created

### Volume Status Entity

- **Entity ID**: `sensor.srat_volume_status`
- **State**: Total number of disks
- **Attributes**:
  - `total_disks`: Total number of disks detected
  - `total_partitions`: Total number of partitions across all disks
  - `mounted_partitions`: Number of partitions currently mounted
  - `shared_partitions`: Number of partitions with active Samba shares

### Disk Entities

For each detected disk, an entity is created:

- **Entity ID**: `sensor.srat_disk_{disk_id}`
- **State**: "connected"
- **Attributes**:
  - `device`: Device path (e.g., `/dev/sda`)
  - `model`: Disk model
  - `vendor`: Disk vendor
  - `serial`: Disk serial number
  - `size_bytes`: Disk size in bytes
  - `size_gb`: Disk size in gigabytes
  - `connection_bus`: Connection type (USB, SATA, etc.)
  - `removable`: Whether the disk is removable
  - `ejectable`: Whether the disk is ejectable
  - `partition_count`: Number of partitions on the disk

### Partition Entities

For each partition on each disk, an entity is created:

- **Entity ID**: `sensor.srat_partition_{partition_id}`
- **State**: "unmounted", "mounted", or "shared"
- **Attributes**:
  - `device`: Partition device path (e.g., `/dev/sda1`)
  - `name`: Partition name/label
  - `size_bytes`: Partition size in bytes
  - `size_gb`: Partition size in gigabytes
  - `system`: Whether it's a system partition
  - `disk_id`: ID of the parent disk
  - `mount_path`: Current mount path (if mounted)
  - `mounted_count`: Number of active mount points
  - `share_count`: Number of active Samba shares

### Samba Status Entity

- **Entity ID**: `sensor.srat_samba_status`
- **State**: "connected", "idle", or "unknown"
- **Attributes**:
  - `version`: Samba version
  - `smb_conf`: Configuration file path
  - `timestamp`: Last status update timestamp
  - `session_count`: Number of active Samba sessions
  - `tcon_count`: Number of active tree connections

### Samba Process Status Entity

- **Entity ID**: `sensor.srat_samba_process_status`
- **State**: "running", "partial", or "stopped"
- **Attributes**:
  - `smbd_running`: Whether smbd process is running
  - `nmbd_running`: Whether nmbd process is running
  - `wsdd2_running`: Whether wsdd2 process is running
  - `avahi_running`: Whether avahi process is running
  - `smbd_pid`: smbd process ID (if running)
  - `smbd_cpu_percent`: smbd CPU usage percentage (if running)
  - `smbd_memory_percent`: smbd memory usage percentage (if running)
  - `nmbd_pid`: nmbd process ID (if running)
  - `nmbd_cpu_percent`: nmbd CPU usage percentage (if running)
  - `nmbd_memory_percent`: nmbd memory usage percentage (if running)

## How It Works

1. When SRAT starts in addon mode (`--addon` flag), it initializes the Home Assistant Core API client using the supervisor URL and token.

2. The `HomeAssistantService` is responsible for creating and updating entities via the Core State API.

3. The `BroadcasterService` is enhanced to detect specific message types (disk data, samba status) and automatically send them to Home Assistant.

4. The health check process periodically broadcasts samba status and process information, which triggers updates to the corresponding Home Assistant entities.

5. Volume data broadcasts (when disks are mounted/unmounted) also trigger updates to disk, partition, and volume status entities.

## Configuration

The integration is automatically enabled when:

- SRAT has a configured Home Assistant Core API client
- The `SUPERVISOR_TOKEN` environment variable is set (when running in addon mode)
- The supervisor URL is accessible

The Core API client is automatically configured when running with the `--addon` flag, but can also be manually configured for standalone installations.

## Usage Example

When running SRAT as a Home Assistant addon:

```bash
srat-server --addon --port 8080 --ha-token "$SUPERVISOR_TOKEN" --ha-url "http://supervisor/"
```

The integration will automatically:

1. Create entities for all detected disks and partitions
2. Update entity states when volumes are mounted/unmounted
3. Monitor Samba service status and process health
4. Send periodic updates every few seconds via the health check system

## Example Home Assistant Dashboard

You can create a dashboard in Home Assistant to monitor your SRAT storage:

```yaml
type: vertical-stack
cards:
  - type: sensor
    entity: sensor.srat_volume_status
    name: "Storage Overview"
  - type: entities
    entities:
      - sensor.srat_samba_status
      - sensor.srat_samba_process_status
    title: "Samba Status"
  - type: auto-entities
    card:
      type: entities
      title: "Disks"
    filter:
      include:
        - entity_id: "sensor.srat_disk_*"
  - type: auto-entities
    card:
      type: entities
      title: "Partitions"
    filter:
      include:
        - entity_id: "sensor.srat_partition_*"
```

## Troubleshooting

If entities are not appearing in Home Assistant:

1. **Check client configuration**: Ensure SRAT has a configured Home Assistant Core API client
2. **Verify token**: Ensure `SUPERVISOR_TOKEN` environment variable is set (for addon mode)
3. **Check logs**: Look for Home Assistant integration messages in SRAT logs
4. **Network connectivity**: Verify SRAT can reach the supervisor URL
5. **Entity cleanup**: If testing, old entities may need to be manually removed from Home Assistant

## Limitations

- Entities are created only when data is broadcasted (on volume changes, health checks)
- Entity state updates depend on the health check interval (configurable)
- Historical data is not preserved if SRAT is restarted
- Entity icons and device classes are fixed and not user-configurable

## Entity IDs

Entity IDs are automatically generated based on the disk/partition IDs, with special characters replaced by underscores and converted to lowercase for Home Assistant compatibility.

For example:

- Disk ID `usb-SanDisk_USB_3.2Gen1-0:0` becomes `sensor.srat_disk_usb_sandisk_usb_3_2gen1_0_0`
- Partition ID `uuid-12345678-1234-1234-1234-123456789abc` becomes `sensor.srat_partition_uuid_12345678_1234_1234_1234_123456789abc`
