# SRAT Home Assistant Integration - Implementation Summary

<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->

- [Overview](#overview)
- [Files Created/Modified](#files-createdmodified)
  - [New Files:](#new-files)
  - [Modified Files:](#modified-files)
- [Key Features Implemented](#key-features-implemented)
  - [Entity Types Created:](#entity-types-created)
  - [Integration Points:](#integration-points)
  - [Core Functionality:](#core-functionality)
- [Technical Implementation Details](#technical-implementation-details)
  - [Dependency Injection:](#dependency-injection)
  - [Broadcasting Architecture:](#broadcasting-architecture)
  - [Client Integration:](#client-integration)
  - [Entity Management:](#entity-management)
- [Configuration](#configuration)
- [Testing](#testing)
- [Error Handling](#error-handling)
- [Performance Considerations](#performance-considerations)
- [Future Enhancements](#future-enhancements)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

## Overview

This implementation adds automatic Home Assistant integration to SRAT when a Home Assistant Core API client is configured. The system creates and updates Home Assistant entities via the Core State API for disk management, partition status, and Samba service monitoring.

## Files Created/Modified

### New Files:

1. **`backend/src/service/homeassistant_service.go`** - Core service for managing Home Assistant entities
2. **`backend/src/service/homeassistant_service_test.go`** - Unit tests for the Home Assistant service
3. **`docs/HOME_ASSISTANT_INTEGRATION.md`** - Complete documentation for the integration

### Modified Files:

1. **`backend/src/service/broacaster_service.go`** - Enhanced to send data to Home Assistant
2. **`backend/src/internal/appsetup/appsetup.go`** - Added Home Assistant service and Core API client to DI
3. **`backend/src/api/health.go`** - Added broadcasting of samba status for HA integration

## Key Features Implemented

### Entity Types Created:

- **Volume Status Entity**: Overall storage system status
- **Disk Entities**: Individual disk information and status
- **Partition Entities**: Partition mount/share status
- **Samba Status Entity**: Samba service configuration and sessions
- **Samba Process Status Entity**: Individual Samba process health

### Integration Points:

- **Health Check Broadcasting**: Periodic updates via existing health check system
- **Volume Event Broadcasting**: Real-time updates when volumes change
- **Automatic Entity Management**: Entities created/updated automatically based on system state

### Core Functionality:

- **Client Availability Check**: Only activates when Core API client is configured
- **Entity ID Sanitization**: Safe entity IDs compatible with Home Assistant
- **Error Handling**: Graceful degradation when Home Assistant API is unavailable
- **State Management**: Proper entity states (connected, mounted, shared, running, etc.)

## Technical Implementation Details

### Dependency Injection:

- Home Assistant service depends on Core API client
- Broadcaster service enhanced to use Home Assistant service
- Services ordered correctly to avoid circular dependencies

### Broadcasting Architecture:

- Enhanced `BroadcasterService.BroadcastMessage()` to detect message types
- Automatic routing of disk/samba data to Home Assistant
- Type-safe message handling with appropriate entity updates

### Client Integration:

- Uses generated Core API client from OpenAPI spec
- Proper authentication via supervisor token
- Error handling for HTTP status codes

### Entity Management:

- Dynamic entity creation based on system data
- Consistent entity ID format across all entity types
- Rich attributes providing detailed system information
- State values meaningful for Home Assistant automation

## Configuration

The integration automatically activates when:

- SRAT has a configured Home Assistant Core API client
- `SUPERVISOR_TOKEN` environment variable is available (for addon mode)
- Supervisor URL is accessible

The Core API client is automatically configured when running with the `--addon` flag, but can also be manually configured for other deployment scenarios.

## Testing

Comprehensive test suite covering:

- Service initialization with/without secure mode
- Entity creation with various data types
- Error handling for missing/invalid data
- Entity ID sanitization for special characters

## Error Handling

Robust error handling ensures:

- No crashes if Home Assistant API is unavailable
- Logging of failures for debugging
- Graceful degradation when client is not configured
- Continue normal operation if entity updates fail

## Performance Considerations

- Minimal overhead when Home Assistant client is not configured
- Efficient message type detection using Go type switches
- Non-blocking entity updates (logged warnings on failures)
- Reuses existing health check timing for periodic updates

## Future Enhancements

Potential improvements that could be added:

- Configurable entity update intervals
- Custom entity icons/device classes
- Historical data preservation
- Entity removal on shutdown
- Batch entity updates for better performance
- Additional entity types (network, system stats, etc.)

This implementation provides a solid foundation for Home Assistant integration while maintaining the existing SRAT functionality and performance.
