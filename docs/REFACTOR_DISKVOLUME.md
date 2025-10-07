# REFACTOR_VOLUME_CONTROL.md

<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->
**Table of Contents**

- [Overview](#overview)
- [Key Changes](#key-changes)
  - [1. Device Field Standardization](#1-device-field-standardization)
  - [2. Removal of lsblk Package](#2-removal-of-lsblk-package)
  - [3. Volume Service Refactoring](#3-volume-service-refactoring)
  - [4. Hardware Service Enhancements](#4-hardware-service-enhancements)
  - [5. Database Schema Changes](#5-database-schema-changes)
  - [6. Frontend Updates](#6-frontend-updates)
  - [7. Service Layer Changes](#7-service-layer-changes)
- [Migration](#migration)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

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

- [x] Add migration script to migrate DB table to for DeviceId and DevicePath
- [x] Update existing mount point records to use new device fields
- [x] Test mount/unmount operations with new device identification
- [ ] Verify SMART data retrieval works correctly
- [ ] Update any external integrations that depend on old device fields
