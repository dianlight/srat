# Share Volume Verification

This document describes how SRAT verifies shares with associated volumes and handles different volume states.

## Overview

SRAT automatically verifies the state of volumes associated with shares and marks shares as invalid (anomalies) when their volumes are in problematic states. This ensures that only accessible and properly configured shares are active in the Samba configuration.

## Volume States and Share Behavior

### 1. Volume Mounted and Read-Write (RW)

**Behavior:**
- Share can be active or inactive as per the database `disabled` value
- Read-write users are allowed and preserved
- Share operates normally

**Example:**
```json
{
  "name": "data-share",
  "disabled": false,
  "users": [
    {
      "username": "user1",
      "rw_shares": ["data-share"]
    }
  ],
  "mount_point_data": {
    "path": "/mnt/data",
    "is_mounted": true,
    "is_write_supported": true
  },
  "invalid": false
}
```

### 2. Volume Mounted and Read-Only (RO)

**Behavior:**
- Share can be active or inactive as per the database `disabled` value
- Read-write users are **automatically converted to read-only users**
- Any RW permissions for this share are removed from user configurations
- Share operates in read-only mode

**Example:**
```json
{
  "name": "backup-share",
  "disabled": false,
  "users": [],
  "ro_users": [
    {
      "username": "user1"
    }
  ],
  "mount_point_data": {
    "path": "/mnt/backup",
    "is_mounted": true,
    "is_write_supported": false
  },
  "invalid": false
}
```

**Note:** If a user has RW permissions for multiple shares and one is RO, only the RO share is removed from their `rw_shares` list.

### 3. Volume Not Mounted

**Behavior:**
- Share is **automatically disabled** (`disabled` = `true`)
- Share is **marked as invalid** (`invalid` = `true`) - indicates an anomaly
- Warning logged: "Share volume is not mounted"
- Share will not appear in Samba configuration until volume is remounted

**Example:**
```json
{
  "name": "unmounted-share",
  "disabled": true,
  "invalid": true,
  "mount_point_data": {
    "path": "/mnt/external",
    "is_mounted": false
  }
}
```

**Resolution:**
1. Mount the volume
2. Re-enable the share if needed
3. The `invalid` flag will be cleared automatically on next verification

### 4. Volume Does Not Exist

**Behavior:**
- Share is **automatically disabled** (`disabled` = `true`)
- Share is **marked as invalid** (`invalid` = `true`) - indicates an anomaly
- Mount point marked as invalid (`mount_point_data.invalid` = `true`)
- Warning logged: "Share volume does not exist"
- Share will not appear in Samba configuration

**Example:**
```json
{
  "name": "missing-volume-share",
  "disabled": true,
  "invalid": true,
  "mount_point_data": {
    "path": "/mnt/nonexistent",
    "is_mounted": false,
    "invalid": true
  }
}
```

**Resolution:**
1. Create/connect the volume
2. Ensure the volume is properly recognized by the system
3. Re-enable the share
4. The `invalid` flag will be cleared automatically on next verification

## Special Cases

### No Mount Point Data

**Behavior:**
- Share without `mount_point_data` or with empty `path` is automatically disabled and marked invalid
- Warning logged: "Share has no valid MountPointData"

**Example:**
```json
{
  "name": "invalid-share",
  "disabled": true,
  "invalid": true,
  "mount_point_data": null
}
```

### Home Assistant Mount Status

For shares with `usage` other than `internal` or `none`:

**Behavior:**
- If `is_ha_mounted` = `false`, share is disabled and marked invalid
- Internal shares (`usage: "internal"`) ignore Home Assistant mount status
- Warning logged: "Share mount point is not mounted in Home Assistant"

## API Response Fields

| Field | Type | Description |
|-------|------|-------------|
| `disabled` | boolean | Whether the share is active in Samba configuration |
| `invalid` | boolean | Whether the share has anomalies (volume issues) |
| `mount_point_data.is_mounted` | boolean | Whether the volume is currently mounted |
| `mount_point_data.is_write_supported` | boolean | Whether the volume supports write operations |
| `mount_point_data.invalid` | boolean | Whether the mount point itself is invalid |
| `is_ha_mounted` | boolean | Whether Home Assistant recognizes the mount |

## Verification Logic

The `VerifyShare` method in `ShareService` performs the following checks in order:

1. **Nil Check**: Verify share object is not nil
2. **Mount Point Existence**: Check if `mount_point_data` exists and has a valid path
3. **Volume Existence**: Check if mount point is marked as invalid
4. **Mount Status**: Verify the volume is currently mounted
5. **Write Support vs Permissions**: Ensure RO volumes don't have RW users
6. **Home Assistant Status**: For non-internal shares, verify HA mount status

## Testing

Comprehensive tests cover all volume states:

### API Tests (`backend/src/api/shares_test.go`)
- `TestListSharesWithVolumeMountedRW`: Tests RW mounted volumes
- `TestListSharesWithVolumeMountedRO`: Tests RO mounted volumes
- `TestListSharesWithVolumeNotMounted`: Tests unmounted volumes
- `TestListSharesWithVolumeNotExists`: Tests non-existent volumes
- `TestGetShareWithVolumeNotMounted`: Tests individual share retrieval
- `TestCreateShareWithMountedRWVolume`: Tests share creation with RW volume
- `TestCreateShareWithROVolumeHasOnlyROUsers`: Tests RO volume user restrictions

### Service Tests (`backend/src/service/share_service_test.go`)
- `TestVerifyShareWithMountedRWVolume`: Verifies RW volume handling
- `TestVerifyShareWithMountedROVolume`: Verifies RO volume and user conversion
- `TestVerifyShareWithNotMountedVolume`: Verifies unmounted volume anomaly detection
- `TestVerifyShareWithNonExistentVolume`: Verifies non-existent volume handling
- `TestVerifyShareWithNoMountPointData`: Verifies missing mount point handling
- `TestVerifyShareWithNotHAMounted`: Verifies Home Assistant mount status
- `TestVerifyShareInternalUsageIgnoresHAMount`: Verifies internal share exception

## Troubleshooting

### Share Marked as Invalid

1. Check the `mount_point_data.is_mounted` field in the share details
2. Verify the volume exists and is mounted on the system
3. For non-internal shares, check `is_ha_mounted` status
4. Review system logs for specific warning messages

### RW Permissions Removed

1. Check `mount_point_data.is_write_supported` - if `false`, the volume is RO
2. Users will automatically be added to `ro_users` instead of having RW permissions
3. To restore RW access, ensure the volume is mounted with write support

### Share Disabled Unexpectedly

1. Check the `invalid` field - if `true`, there's a volume anomaly
2. Review the verification logic sections above
3. Check mount point status and volume availability
4. Re-enable the share after resolving volume issues

## Related Files

- Implementation: `backend/src/service/share_service.go` (`VerifyShare` method)
- API Handler: `backend/src/api/shares.go`
- DTO Definitions: `backend/src/dto/shared_resource.go`, `backend/src/dto/mount_point_data.go`
- API Tests: `backend/src/api/shares_test.go`
- Service Tests: `backend/src/service/share_service_test.go`

## Future Enhancements

Potential improvements to consider:

1. **Auto-recovery**: Automatically re-enable shares when volumes become available
2. **Notification System**: Alert administrators when shares are disabled due to volume issues
3. **Volume Health Monitoring**: Track volume health metrics and predict failures
4. **Graceful Degradation**: Allow read-only access when write support is temporarily unavailable
