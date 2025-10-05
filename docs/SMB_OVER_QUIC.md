# SMB over QUIC

## Overview

SMB over QUIC is a modern transport protocol for SMB (Server Message Block) that provides improved performance, security, and reliability over traditional TCP-based SMB connections. This feature is available in SRAT starting from version 0.0.0-dev.0.

## Features

- **Enhanced Performance**: QUIC provides better performance over high-latency networks and lossy connections
- **Improved Security**: Mandatory SMB3 encryption when QUIC is enabled
- **Better Mobility**: QUIC handles network changes more gracefully than TCP
- **Reduced Latency**: QUIC's 0-RTT connection establishment for faster reconnections

## System Requirements

### Samba Version

SMB over QUIC requires **Samba 4.23.0 or later**. SRAT automatically detects your installed Samba version and will only allow enabling QUIC if the version requirement is met.

### Transport Support

SMB over QUIC requires the QUIC kernel module to be loaded:

- `quic` kernel module
- `net_quic` kernel module (alternative name on some systems)

SRAT will check for both module names and enable QUIC support if either is available along with a sufficient Samba version.

### Checking QUIC Support

You can check if your system supports QUIC by:

1. **Via UI**: Navigate to Settings page. The "SMB over QUIC" switch will be:
   - Enabled (available for toggling) if QUIC is fully supported
   - Disabled with a detailed message explaining what requirements are missing if not supported

2. **Via API**: Call the `/api/capabilities` endpoint:
   ```bash
   curl http://your-srat-instance/api/capabilities
   ```
   
   Response:
   ```json
   {
     "supports_quic": true,
     "has_kernel_module": true,
     "samba_version": "4.23.1",
     "samba_version_sufficient": true,
     "unsupported_reason": ""
   }
   ```
   
   If QUIC is not supported, the response will indicate why:
   ```json
   {
     "supports_quic": false,
     "has_kernel_module": false,
     "samba_version": "4.20.0",
     "samba_version_sufficient": false,
     "unsupported_reason": "Samba version must be >= 4.23.0; QUIC kernel module (quic or net_quic) not loaded"
   }
   ```

3. **Via Command Line**: 
   
   Check Samba version:
   ```bash
   smbd --version
   ```
   
   Check if kernel module is loaded:
   ```bash
   lsmod | grep quic
   # or
   cat /proc/modules | grep quic
   ```

## Configuration

### Enabling SMB over QUIC

1. Navigate to the **Settings** page in SRAT web UI
2. Locate the "SMB over QUIC" switch
3. Toggle the switch to enable (if your system supports QUIC)
4. Click **Apply** to save the settings

The Samba configuration will be automatically updated with QUIC-specific settings.

### Automatic Configuration Changes

When SMB over QUIC is enabled, SRAT automatically configures Samba with:

- **Mandatory Encryption**: `server smb3 encryption = mandatory`
- **UNIX Extensions Disabled**: `smb3 unix extensions = no`
- **QUIC Port**: `smb ports = 443` (QUIC uses port 443 instead of the traditional SMB ports)

### Disabling SMB over QUIC

To disable SMB over QUIC:

1. Navigate to the **Settings** page
2. Toggle the "SMB over QUIC" switch to off
3. Click **Apply** to save

The Samba configuration will revert to standard TCP-based SMB settings.

## Client Configuration

### Windows Clients

Windows 11 and Windows Server 2022 Datacenter: Azure Edition support SMB over QUIC natively.

To connect from a Windows client:

```powershell
# Map a network drive using QUIC
net use Z: \\your-server-name.com\share /TRANSPORT:QUIC
```

### Linux Clients

QUIC support for SMB clients on Linux is currently limited. Check your distribution's documentation for SMB client QUIC support.

## Troubleshooting

### Requirements Not Met

If SRAT reports that QUIC is not supported, check which requirement is missing:

#### Samba Version Too Old

If your Samba version is less than 4.23.0:

1. **Check Current Version**:
   ```bash
   smbd --version
   ```

2. **Upgrade Samba**: Depending on your distribution:
   
   - **Ubuntu/Debian**:
     ```bash
     sudo apt update
     sudo apt install samba
     ```
   
   - **Red Hat/CentOS/Fedora**:
     ```bash
     sudo dnf upgrade samba
     ```

3. **Build from Source**: If your distribution doesn't provide Samba 4.23+, you may need to build from source. See [Samba Build Documentation](https://wiki.samba.org/index.php/Build_Samba_from_Source).

#### QUIC Kernel Module Not Available

If the QUIC kernel module is not available:

**Load QUIC Kernel Module**

1. **Check Kernel Version**: QUIC support requires a relatively recent kernel:
   ```bash
   uname -r
   ```

2. **Load QUIC Module**:
   ```bash
   sudo modprobe quic
   # or
   sudo modprobe net_quic
   ```

3. **Make Persistent**: Add to `/etc/modules` or `/etc/modules-load.d/`:
   ```bash
   echo "quic" | sudo tee /etc/modules-load.d/quic.conf
   ```

4. **Verify Module is Loaded**:
   ```bash
   lsmod | grep quic
   ```

### Connection Issues

If you're having trouble connecting after enabling QUIC:

1. **Firewall**: Ensure port 443 is open for QUIC traffic:
   ```bash
   sudo ufw allow 443/udp
   ```
   
   Note: QUIC uses UDP, not TCP!

2. **Check Samba Status**: Verify Samba is running and using the correct configuration:
   ```bash
   sudo systemctl status smbd
   ```

3. **View Logs**: Check SRAT and Samba logs for any errors:
   - SRAT logs: Available in the Home Assistant supervisor logs
   - Samba logs: Configured location based on your settings

### Performance Issues

If you experience performance degradation after enabling QUIC:

1. **Network Environment**: QUIC provides benefits primarily over high-latency or lossy connections. On a local LAN, traditional TCP may perform better.

2. **MTU Settings**: QUIC can be sensitive to MTU settings. Ensure your network MTU is properly configured.

3. **Disable if Needed**: If QUIC doesn't provide benefits in your environment, you can disable it and revert to TCP-based SMB.

## Security Considerations

### Mandatory Encryption

When QUIC is enabled, SMB3 encryption becomes mandatory. This means:

- All SMB traffic is encrypted
- Clients must support SMB3 encryption
- Older SMB clients (SMB2.1 and earlier) cannot connect

### Port Usage

QUIC uses port 443 (HTTPS port) for SMB traffic. This can:

- Help bypass firewalls that allow HTTPS traffic
- Conflict with other services using port 443
- Require additional firewall configuration

## API Reference

### Get System Capabilities

**Endpoint**: `GET /api/capabilities`

**Response**:
```json
{
  "supports_quic": true,
  "has_kernel_module": true,
  "samba_version": "4.23.1",
  "samba_version_sufficient": true,
  "unsupported_reason": ""
}
```

**Fields**:
- `supports_quic`: Overall QUIC support status (requires Samba 4.23+ AND kernel module)
- `has_kernel_module`: Whether QUIC kernel module (`quic` or `net_quic`) is loaded
- `samba_version`: Installed Samba version string
- `samba_version_sufficient`: Whether Samba version >= 4.23.0
- `unsupported_reason`: Human-readable explanation when `supports_quic` is false (optional)

### Get Settings

**Endpoint**: `GET /api/settings`

**Response** (excerpt):
```json
{
  "smb_over_quic": true,
  ...
}
```

### Update Settings

**Endpoint**: `PATCH /api/settings`

**Request**:
```json
{
  "smb_over_quic": true
}
```

## References

- [Microsoft SMB over QUIC Documentation](https://docs.microsoft.com/en-us/windows-server/storage/file-server/smb-over-quic)
- [QUIC Protocol Specification (RFC 9000)](https://www.rfc-editor.org/rfc/rfc9000.html)
- [Samba SMB3 Documentation](https://wiki.samba.org/index.php/SMB3)

## Related Issues

- [Issue #227: SMB over QUIC Support](https://github.com/dianlight/srat/issues/227)
