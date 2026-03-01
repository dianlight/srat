<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->

- [SRAT Settings Documentation](#srat-settings-documentation)
  - [General Settings](#general-settings)
    - [Hostname](#hostname)
    - [Workgroup](#workgroup)
    - [Local Master](#local-master)
    - [Compatibility Mode](#compatibility-mode)
    - [Allow Guest](#allow-guest)
  - [Network Settings](#network-settings)
    - [Devices](#devices)
      - [Bind All Interfaces](#bind-all-interfaces)
      - [Interfaces](#interfaces)
      - [Multi Channel](#multi-channel)
      - [SMB over QUIC](#smb-over-quic)
    - [Access Control](#access-control)
      - [Allow Hosts](#allow-hosts)
  - [Telemetry Settings](#telemetry-settings)
    - [Telemetry Mode](#telemetry-mode)
  - [Home Assistant Settings](#home-assistant-settings)
    - [Export Stats to Home Assistant](#export-stats-to-home-assistant)
    - [Use Network File System for Home Assistant Integration (Experimental)](#use-network-file-system-for-home-assistant-integration-experimental)
  - [Implementation Details](#implementation-details)
    - [Template Generation](#template-generation)
    - [back-end Storage](#back-end-storage)
    - [API Endpoint](#api-endpoint)
    - [Frontend Integration](#frontend-integration)
  - [Related Documentation](#related-documentation)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

# SRAT Settings Documentation

This document provides detailed information about all SRAT settings available in the web UI and REST API.

## General Settings

### Hostname

- **Type**: String
- **Default**: System hostname
- **Description**: The NetBIOS name for this Samba server (max 15 characters, alphanumeric and hyphens)
- **Example**: `sambanas` or `home-nas`

### Workgroup

- **Type**: String
- **Default**: `NOWORKGROUP`
- **Description**: The Windows workgroup name for this Samba server (max 15 characters, alphanumeric and hyphens)
- **Example**: `WORKGROUP` or `HOMELAB`

### Local Master

- **Type**: Boolean
- **Default**: `true`
- **Description**: When enabled, this server will participate in local master browser elections
- **Impact**: Affects network browsing and server visibility on Windows networks

### Compatibility Mode

- **Type**: Boolean
- **Default**: `false`
- **Description**: When enabled, allows older SMB/CIFS clients (NT1 protocol) to connect. Useful for legacy devices
- **Impact**: Security - disabling this is more secure as NT1 is deprecated

### Allow Guest

- **Type**: Boolean
- **Default**: `false`
- **Description**: When enabled, allows anonymous guest access to Samba shares
- **Configuration**: When enabled, the following settings are added to the Samba global configuration:
  - `guest account = nobody` - Sets the guest account to the 'nobody' user
  - `map to guest = Bad User` - Maps unknown users to the guest account
- **Security Implications**:
  - Enabling this allows unauthenticated access to shares marked as `guest ok`
  - The guest account is mapped to the 'nobody' user for security isolation
  - Only enable if you trust all clients on your network or have proper firewall rules
- **UI Location**: Settings → General → Allow Guest toggle
- **API Field**: `allow_guest` (boolean)

## Network Settings

### Devices

#### Bind All Interfaces

- **Type**: Boolean
- **Default**: `false`
- **Description**: When disabled (default), Samba only binds to specified interfaces. When enabled, binds to all available interfaces
- **Interfaces**: Configure specific network interfaces when this is disabled

#### Interfaces

- **Type**: String Array
- **Default**: Empty (requires binding all interfaces or explicit selection)
- **Description**: List of network interfaces to bind to (for example, `eth0`, `wlan0`)
- **Example**: `["eth0", "docker0"]`

#### Multi Channel

- **Type**: Boolean
- **Default**: `false`
- **Description**: Enables SMB3 multi-channel support for improved performance with multiple network paths

#### SMB over QUIC

- **Type**: Boolean
- **Default**: `true`
- **Description**: Enables QUIC transport protocol for SMB3 (requires Samba 4.23.0+)
- **Performance**: Can provide better performance and lower latency for remote connections
- **Requirements**: Requires Samba 4.23.0 or later

### Access Control

#### Allow Hosts

- **Type**: String Array (IPv4 CIDR, IPv6 CIDR, or single IPs)
- **Default**: Common private subnets
- **Description**: Network addresses/CIDR blocks allowed to connect to Samba
- **Examples**:
  - `192.168.1.0/24` - Allow entire subnet
  - `10.0.0.0/8` - Allow class A private range
  - `fe80::/10` - Allow IPv6 link-local

## Telemetry Settings

### Telemetry Mode

- **Type**: Enum
- **Options**:
  - `Ask` - Prompt user on first run
  - `All` - Send all telemetry data
  - `Errors` - Send only error data
  - `Disabled` - Do not send any telemetry
- **Default**: `Ask`
- **Description**: Controls what usage data and error reports are sent for improvement and debugging

## Home Assistant Settings

### Export Stats to Home Assistant

- **Type**: Boolean
- **Default**: `true`
- **Description**: When enabled, exports share statistics and Samba server statistics to Home Assistant as entities

### Use Network File System for Home Assistant Integration (Experimental)

- **Type**: Boolean
- **Default**: `false`
- **Status**: ⚠️ **Experimental Feature**
- **Description**: When enabled, Home Assistant will mount shares using NFS protocol instead of SMB/CIFS. This can provide better performance and efficiency for Home Assistant integrations.
- **Requirements**:
  - NFS server must be properly configured on the system
  - The `exportfs` command must be available on the system
  - Network File System support must be available in Home Assistant
- **Availability**:
  - If the `exportfs` command is not found on the system, this setting will automatically be disabled (set to `false`) and cannot be enabled
  - The system checks for NFS availability when the setting is updated
- **Benefits**:
  - Potentially better performance compared to SMB/CIFS
  - Lower overhead for Home Assistant operations
  - Native Linux protocol for file sharing
- **Considerations**:
  - This is an experimental feature and may have compatibility issues
  - Ensure your Home Assistant setup supports NFS mounts
  - Test thoroughly before using in production environments
  - The setting will be automatically disabled if NFS tools are not available
- **UI Location**: Settings → HomeAssistant → Use NFS for HA
- **API Field**: `ha_use_nfs` (boolean)

## Implementation Details

### Template Generation

The `AllowGuest` setting is rendered into the Samba configuration file during template generation via the smb.gtpl template:

```gotmpl
{{if .allow_guest -}}
guest account = nobody
map to guest = Bad User
{{- end }}
```

### back-end Storage

Settings are stored in the database and mapped between:

- **DTO Layer**: `dto.Settings.AllowGuest` (pointer to boolean with default false)
- **Config Layer**: `config.Config.AllowGuest` (boolean)
- **Database**: `Property` table with key `"AllowGuest"`

### API Endpoint

- **Path**: `/api/settings`
- **Method**: `GET` (retrieve), `PATCH` (update)
- **Field Name**: `allow_guest`
- **Request Example**:

  ```json
  {
    "allow_guest": true
  }
  ```

### Frontend Integration

- **Location**: Settings page → General section
- **Component**: SwitchElement (toggle switch)
- **Label**: "Allow Guest"
- **ID**: `allow_guest`

## Related Documentation

- [SMB over QUIC Implementation](SMB_OVER_QUIC_IMPLEMENTATION.md)
- [Telemetry Configuration](TELEMETRY_CONFIGURATION.md)
- [Home Assistant Integration](HOME_ASSISTANT_INTEGRATION.md)
