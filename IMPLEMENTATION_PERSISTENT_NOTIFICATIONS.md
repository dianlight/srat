# Implementation: Persistent Notifications for Automount Partitions

## Overview

This implementation adds persistent notifications through Home Assistant's `/api/services/` endpoint to notify users about unmounted partitions or errors when mounting partitions on startup that are marked for automount.

## Features Implemented

### 1. **Automount Failure Notifications**

- Creates persistent notifications when a partition marked for automount fails to mount during system startup
- Provides specific error messages based on the failure type:
  - Device not found (device may have been removed)
  - Mount failure (filesystem/permission issues)
  - Generic mount errors

### 2. **Unmounted Partition Notifications**

- Creates notifications for partitions that are configured for automount but are currently unmounted
- Helps identify configuration issues or missing devices
- Automatically dismisses notifications when partitions are successfully mounted

### 3. **Automatic Notification Management**

- Automatically dismisses old notifications when issues are resolved
- Periodic checks during volume data retrieval to keep notifications current
- Smart notification lifecycle management

## Technical Implementation

### Modified Files

#### 1. **Home Assistant Core API** (`backend/src/homeassistant/core_api.yaml`)

- Added `/core/api/services/{domain}/{service}` endpoint for calling Home Assistant services
- Added `ServiceData` and `ServiceResult` schemas for service calls
- Enables calling `persistent_notification.create` and `persistent_notification.dismiss` services

#### 2. **Home Assistant Service** (`backend/src/service/homeassistant_service.go`)

- Added `CreatePersistentNotification(notificationID, title, message string) error`
- Added `DismissPersistentNotification(notificationID string) error`
- Integrated with Home Assistant's persistent notification service

#### 3. **Volume Service** (`backend/src/service/volume_service.go`)

- Added `CreateAutomountFailureNotification(mountPath, device string, err errors.E)`
- Added `CreateUnmountedPartitionNotification(mountPath, device string)`
- Added `DismissAutomountNotification(mountPath string, notificationType string)`
- Added `CheckUnmountedAutomountPartitions() error`
- Integrated notification creation/dismissal into mount/unmount workflows
- Added periodic checks during volume data updates

#### 4. **CLI Application** (`backend/src/cmd/srat-cli/main-cli.go`)

- Enhanced startup automount process with notification support
- Creates failure notifications when automount fails during startup
- Checks for unmounted automount partitions after startup process
- Dismisses notifications when mounts are successful

### Notification Types

#### Automount Failure Notifications

- **ID Format**: `srat_automount_failure_{SHA1(mount_path)}`
- **Title**: "Automount Failed"
- **Messages**:
  - Device not found: "Device '{device}' for mount point '{path}' not found during startup. The device may have been removed or disconnected."
  - Mount failure: "Failed to mount device '{device}' to '{path}' during startup. Check device filesystem and permissions."
  - Generic: "Automount failed for device '{device}' to '{path}': {error_message}"

#### Unmounted Partition Notifications

- **ID Format**: `srat_unmounted_partition_{SHA1(mount_path)}`
- **Title**: "Unmounted Partition with Automount Enabled"
- **Message**: "Partition '{path}' (device: {device}) is configured for automount but is currently unmounted. This may indicate a device issue or the device is not connected."

### Workflow Integration

#### During System Startup

1. System attempts to automount all partitions marked with `is_to_mount_at_startup: true`
2. For successful mounts: Dismiss any existing failure notifications
3. For failed mounts: Create appropriate failure notifications
4. After automount process: Check for any unmounted automount partitions

#### During Regular Operation

1. When manually mounting a partition: Dismiss any existing failure notifications
2. When unmounting a partition marked for automount: Create unmounted partition notification
3. During periodic volume data updates: Check and update notifications for unmounted automount partitions

#### When Devices Are Removed

1. The existing `HandleRemovedDisks` functionality continues to work
2. Creates issues for removed disks (existing functionality)
3. New notification system provides additional user-friendly alerts through Home Assistant

## Usage Examples

### Home Assistant Notifications

When a device marked for automount fails to mount, users will see persistent notifications in their Home Assistant interface with clear, actionable information about what went wrong.

### API Integration

The system uses Home Assistant's standard persistent notification service:

```bash
# Example API call made by the system
POST /api/services/persistent_notification/create
{
  "notification_id": "srat_automount_failure_abc123",
  "title": "Automount Failed",
  "message": "Device '/dev/sda1' for mount point '/mnt/data' not found during startup. The device may have been removed or disconnected."
}
```

## Benefits

1. **Proactive Monitoring**: Users are immediately notified of automount issues
2. **Clear Error Context**: Specific error messages help users understand what went wrong
3. **Automatic Cleanup**: Notifications are automatically dismissed when issues are resolved
4. **Home Assistant Integration**: Uses familiar Home Assistant notification system
5. **Persistent Storage**: Notifications persist across Home Assistant restarts
6. **Non-Intrusive**: Doesn't interfere with existing functionality, only adds notifications

## Configuration

The system works automatically when:

- Home Assistant service is available and configured
- Partitions are marked with `is_to_mount_at_startup: true`
- No additional configuration required

## Error Handling

- Graceful degradation when Home Assistant service is unavailable
- Comprehensive logging for troubleshooting
- Non-blocking operations - notification failures don't affect mount operations
- Duplicate notification prevention through unique IDs

## Future Enhancements

Potential improvements could include:

- Configurable notification severity levels
- Email notifications for critical issues
- Integration with other notification services
- Historical notification tracking
- User-customizable notification messages

This implementation provides a robust, user-friendly way to monitor automount operations and ensure users are aware of any issues that require attention.
