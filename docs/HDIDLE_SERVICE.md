# HDIdle Service

The HDIdle service provides hard disk idle monitoring and automatic spin-down functionality for SRAT. It uses the [hd-idle](https://github.com/adelolmo/hd-idle) library as a module to replicate the functionality of running `hd-idle -a` (monitor all disks).

## Overview

The service monitors disk activity and automatically spins down idle disks after a configurable timeout period. This helps reduce power consumption and extend disk lifespan in NAS environments.

## Features

- **Automatic disk monitoring**: Continuously monitors all detected disks for I/O activity
- **Configurable idle timeouts**: Set different idle times per disk or use global defaults
- **Multiple command types**: Supports both SCSI and ATA spin-down commands
- **Symlink resolution**: Handles device symlinks (e.g., `/dev/disk/by-id/...`)
- **Debug logging**: Detailed logging for troubleshooting
- **Power condition control**: Fine-grained control of SCSI power conditions
- **Suspend detection**: Detects system sleep/wake events and resets disk states

## CLI Usage

The service can be started via the CLI with the `hdidle` command:

```bash
srat-cli hdidle [options]
```

### Options

- `-i <seconds>`: Default idle time in seconds before spinning down disks (default: 600)
- `-c <type>`: Default command type - "scsi" or "ata" (default: "scsi")
- `-debug`: Enable debug logging
- `-l <path>`: Log file path for disk activity events (optional)
- `-s <policy>`: Symlink resolution policy - 0 (resolve once) or 1 (retry resolution) (default: 0)
- `-I`: Ignore spin down detection and force spin down
- `-loglevel <level>`: Global log level (debug, info, warn, error)

### Examples

**Monitor all disks with default settings (10 minute idle time):**
```bash
srat-cli hdidle
```

**Monitor with 5 minute idle time and debug logging:**
```bash
srat-cli -loglevel debug hdidle -i 300 -debug
```

**Monitor using ATA commands with retry symlink resolution:**
```bash
srat-cli hdidle -c ata -s 1
```

**Monitor with activity logging:**
```bash
srat-cli hdidle -l /var/log/hdidle-activity.log
```

## Service API

The HDIdle service can also be used programmatically:

### Interface

```go
type HDIdleServiceInterface interface {
    // Start begins monitoring disk activity and spinning down idle disks
    Start(config *HDIdleConfig) error
    
    // Stop halts the monitoring process
    Stop() error
    
    // IsRunning returns true if the service is currently monitoring
    IsRunning() bool
    
    // GetStatus returns current monitoring status and disk states
    GetStatus() (*HDIdleStatus, error)
}
```

### Configuration

```go
type HDIdleConfig struct {
    // Devices to monitor with specific configurations
    Devices []HDIdleDeviceConfig
    
    // Default idle time in seconds (default: 600)
    DefaultIdleTime int
    
    // Default command type: "scsi" or "ata" (default: "scsi")
    DefaultCommandType string
    
    // Default power condition (0-15) for SCSI devices (default: 0)
    DefaultPowerCondition uint8
    
    // Enable debug logging
    Debug bool
    
    // Log file path (empty = no file logging)
    LogFile string
    
    // Symlink resolution policy: 0 = resolve once, 1 = retry
    SymlinkPolicy int
    
    // Ignore spin down detection and force spin down
    IgnoreSpinDownDetection bool
}

type HDIdleDeviceConfig struct {
    // Device name (e.g., "sda" or "/dev/disk/by-id/...")
    Name string
    
    // Idle time in seconds (0 = use default)
    IdleTime int
    
    // Command type: "scsi" or "ata" (empty = use default)
    CommandType string
    
    // Power condition for SCSI devices (0-15)
    PowerCondition uint8
}
```

### Example Usage

```go
// Create service
hdidleService := service.NewHDIdleService(service.HDIdleServiceParams{
    ApiContext:       ctx,
    ApiContextCancel: cancel,
    State:            state,
})

// Configure monitoring
config := &service.HDIdleConfig{
    DefaultIdleTime:    300,
    DefaultCommandType: "scsi",
    Debug:              true,
    Devices: []service.HDIdleDeviceConfig{
        {
            Name:     "sda",
            IdleTime: 600, // 10 minutes for this disk
        },
        {
            Name:        "sdb",
            IdleTime:    300, // 5 minutes for this disk
            CommandType: "ata",
        },
    },
}

// Start monitoring
if err := hdidleService.Start(config); err != nil {
    log.Fatal(err)
}

// Get current status
status, _ := hdidleService.GetStatus()
for _, disk := range status.Disks {
    fmt.Printf("Disk %s: SpunDown=%v, LastIO=%v\n", 
        disk.Name, disk.SpunDown, disk.LastIOAt)
}

// Stop monitoring when done
hdidleService.Stop()
```

## How It Works

1. **Initialization**: When started, the service reads disk statistics from `/proc/diskstats`
2. **Monitoring Loop**: Every N seconds (interval = idle_time / 10), the service:
   - Reads current disk statistics
   - Compares with previous readings to detect I/O activity
   - Tracks idle time for disks without activity
   - Spins down disks that exceed their idle threshold
3. **Spin Down**: When a disk is idle for longer than its configured timeout:
   - Issues appropriate SCSI or ATA spin-down command
   - Logs the event
   - Marks disk as spun down
4. **Spin Up Detection**: When a spun-down disk shows new I/O activity:
   - Logs the spin-up event with duration statistics
   - Resets idle timer
5. **Suspend Detection**: If the monitoring interval is much longer than expected (3x normal):
   - Assumes system was suspended
   - Resets all disk states to account for potential spin-ups during suspend

## Implementation Details

- **Thread-safe**: All service operations are protected by mutex locks
- **Context-aware**: Respects context cancellation for graceful shutdown
- **Resource efficient**: Polls at configurable intervals based on shortest idle time
- **Logging**: Uses structured logging (tlog) with sensitive data masking
- **Testing**: Comprehensive test suite with 24 tests covering all service methods and edge cases

## Dependencies

- `github.com/adelolmo/hd-idle` - Core hd-idle library for disk control
- Uses standard SRAT patterns: FX for DI, tlog for logging, errors for error handling

## Notes

- Requires root/sudo permissions to issue disk spin-down commands
- Works with both directly attached and symlinked devices
- Automatically discovers all available disks from `/proc/diskstats`
- Does not interfere with disk spin-up - operating system handles that automatically
- Recommended minimum idle time: 60 seconds (to avoid excessive spin-down cycles)

## Testing

Run the test suite:
```bash
cd backend/src
go test -v ./service -run TestHDIdleServiceSuite
```

All tests validate:
- Configuration validation
- Start/stop lifecycle
- Multiple device configurations
- Command type handling (SCSI/ATA)
- Error handling
- Status reporting
