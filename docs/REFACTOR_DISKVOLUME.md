# REFACTOR_VOLUME_CONTROL.md

## Overview
This refactor modernizes the disk volume handling in SRAT by replacing the custom `lsblk` package with standard system utilities and improving device identification. The main goals are to:
- Remove dependency on custom lsblk implementation
- Use `psutil` for partition information
- Standardize device fields across DTOs and DBOMs
- Improve hardware service integration with SMART data
- Enhance mount point synchronization

## Key Changes

### 1. Device Field Standardization
- **Old**: `Device` field stored device path (e.g., "/dev/sda1")
- **New**: 
  - `DeviceId`: Unique identifier for the device (e.g., "sda1")
  - `DevicePath`: Full device path (e.g., "/dev/sda1") 
  - `LegacyDeviceName`: Legacy device name for backward compatibility
  - `LegacyDevicePath`: Legacy device path for backward compatibility

### 2. Removal of lsblk Package
- Deleted entire `backend/src/lsblk/` package
- Removed `lsblk.LSBLKInterpreterInterface` dependency from `VolumeService`
- Replaced lsblk-based device info retrieval with psutil

### 3. Volume Service Refactoring
- Added `psutil` dependency for partition information
- Changed `GetVolumesData()` to use psutil for local mount points
- Improved mount point synchronization with database
- Added `AllByDeviceId()` method to `MountPointPathRepository`
- Enhanced error handling and logging

### 4. Hardware Service Enhancements
- Integrated `SmartServiceInterface` for SMART data retrieval
- Added SMART info population for disks and partitions
- Improved device matching between hardware data and partitions

### 5. Database Schema Changes
- Updated `MountPointPath` model to use `DeviceId` instead of `Device`
- Added new fields for device identification
- Updated all related repositories and tests

### 6. Frontend Updates
- Updated TypeScript types to reflect new device fields
- Changed UI components to use `device_path` and `legacy_device_name`
- Updated actionable items list to use `partition.id` instead of `partition.device`

### 7. Service Layer Changes
- Updated `DiskStatsService` to use new device fields
- Modified `HomeAssistantService` to send correct device information
- Enhanced `ShareService` with updated converter calls
- Made `SmartService` an interface for better testability

## Migration
- [ ] Add migration script to migrate DB table to for DeviceId and DevicePath
- [ ] Update existing mount point records to use new device fields
- [ ] Test mount/unmount operations with new device identification
- [ ] Verify SMART data retrieval works correctly
- [ ] Update any external integrations that depend on old device fields

