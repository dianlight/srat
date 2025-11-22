# TLog Package

<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->

- [Features](#features)
- [Log Levels](#log-levels)
- [Basic Usage](#basic-usage)
  - [Simple Logging](#simple-logging)
  - [Context-Aware Logging](#context-aware-logging)
- [Level Management](#level-management)
  - [Setting Log Levels](#setting-log-levels)
  - [Getting Current Level](#getting-current-level)
  - [Supported Level Strings](#supported-level-strings)
- [Advanced Usage](#advanced-usage)
  - [Event Callbacks](#event-callbacks)
    - [Registering Callbacks](#registering-callbacks)
    - [LogEvent Structure](#logevent-structure)
    - [Managing Callbacks](#managing-callbacks)
    - [Callback Safety Features](#callback-safety-features)
    - [Example: Monitoring Integration](#example-monitoring-integration)
    - [Graceful Shutdown](#graceful-shutdown)
  - [Custom Logger Instances](#custom-logger-instances)
  - [Error Handling](#error-handling)
- [Thread Safety](#thread-safety)
- [Configuration](#configuration)
- [Best Practices](#best-practices)
- [Migration from Previous Version](#migration-from-previous-version)
- [API Reference](#api-reference)
  - [Callback Functions](#callback-functions)
  - [LogCallback Type](#logcallback-type)
- [Enhanced Formatting & Colors](#enhanced-formatting--colors)
  - [Formatter Configuration](#formatter-configuration)
    - [FormatterConfig Structure](#formatterconfig-structure)
    - [Configuration Functions](#configuration-functions)
  - [Advanced Formatters](#advanced-formatters)
    - [Built-in Formatters](#built-in-formatters)
  - [Color-Enhanced Printing](#color-enhanced-printing)
  - [Enhanced Context Support](#enhanced-context-support)
  - [Sensitive Data Protection](#sensitive-data-protection)
  - [Enhanced Error Formatting](#enhanced-error-formatting)
  - [Tozd Error Formatting with Tree Stack Traces](#tozd-error-formatting-with-tree-stack-traces)
    - [Example Usage](#example-usage)
    - [Output Examples](#output-examples)
    - [Features](#features-1)
  - [Time Format Options](#time-format-options)
  - [Color Levels](#color-levels)
- [Enhanced Logger Creation](#enhanced-logger-creation)
  - [Logger with Custom Configuration](#logger-with-custom-configuration)
  - [Logger Methods](#logger-methods)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

The `tlog` package provides an enhanced logging system built on top of Go's `log/slog` package with additional custom log levels and improved functionality.

## Features

- **Custom Log Levels**: Extends slog with `TRACE`, `NOTICE`, and `FATAL` levels
- **Thread-Safe**: All operations are protected by mutexes for concurrent access
- **Context Support**: All logging functions have context-aware variants
- **Case-Insensitive Configuration**: Log level strings are handled case-insensitively
- **Flexible Level Management**: Support for both programmatic and string-based level setting
- **Better Error Messages**: Descriptive error messages with supported level listings
- **Terminal Detection**: Automatically enables/disables colors based on terminal capabilities
- **Event Callbacks**: Asynchronous callback system for log events with panic recovery and queuing

## Log Levels

The package supports the following log levels (from lowest to highest priority):

- **TRACE** (-8): Most verbose logging for detailed execution flow
- **DEBUG** (-4): Debug information for troubleshooting
- **INFO** (0): General information messages
- **NOTICE** (2): Important but normal events
- **WARN** (4): Warning messages for potentially harmful situations
- **ERROR** (8): Error messages that don't halt execution
- **FATAL** (12): Critical errors that cause program termination

## Basic Usage

### Simple Logging

```go
import "github.com/dianlight/srat/tlog"

// Basic logging functions
tlog.Trace("Detailed execution trace", "step", 1)
tlog.Debug("Debug information", "value", someValue)
tlog.Info("Application started", "version", "1.0.0")
tlog.Notice("Configuration loaded", "config", "production")
tlog.Warn("Deprecated function used", "function", "oldFunc")
tlog.Error("Failed to connect", "host", "example.com", "error", err)
tlog.Fatal("Critical system failure") // This will exit the program
```

### Context-Aware Logging

```go
ctx := context.WithValue(context.Background(), "requestId", "req-123")

tlog.TraceContext(ctx, "Processing request")
tlog.DebugContext(ctx, "Database query executed", "query", sql)
tlog.InfoContext(ctx, "Request completed", "duration", duration)
tlog.NoticeContext(ctx, "Cache miss", "key", cacheKey)
tlog.WarnContext(ctx, "Rate limit approaching", "current", current, "limit", limit)
tlog.ErrorContext(ctx, "Validation failed", "field", "email", "value", email)
tlog.FatalContext(ctx, "Database connection lost") // This will exit the program
```

### Logging with Caller Information

The logger automatically includes source file and line information in the output due to the `AddSource: true` configuration. However, you can explicitly add caller information to log entries using the `WithCaller()` helper function:

```go
// Explicit caller information: file:function:line
tlog.Debug("Processing event", append([]any{"event", myEvent}, tlog.WithCaller()...)...)

// Output: ... Processing event event=<event> caller=event_bus.go:emitEvent:165

// Use with any log level
tlog.Info("User logged in", append([]any{"user", username}, tlog.WithCaller()...)...)
tlog.Error("Connection failed", append([]any{"host", hostname, "error", err}, tlog.WithCaller()...)...)

// Combine with context
tlog.DebugContext(ctx, "Query executed", append([]any{"duration", 250}, tlog.WithCaller()...)...)
```

The `WithCaller()` function returns a slice of `[]any` containing `caller` key with value in format `filename:functionName:lineNumber` (e.g., `event_bus.go:emitEvent:165`). This is useful for:

- **Tracing event handlers**: Know exactly where an event was emitted or received
- **Debugging**: Quickly identify the source of log messages
- **Performance monitoring**: Track which functions are logging at specific levels
- **Audit trails**: Include caller context in critical operations

## Level Management

### Setting Log Levels

```go
// Set level programmatically
tlog.SetLevel(tlog.LevelDebug)

// Set level from string (case-insensitive)
err := tlog.SetLevelFromString("debug")
if err != nil {
    log.Fatal(err)
}

// These are all equivalent:
tlog.SetLevelFromString("debug")
tlog.SetLevelFromString("DEBUG")
tlog.SetLevelFromString("Debug")
tlog.SetLevelFromString("  debug  ") // whitespace is trimmed
```

### Getting Current Level

```go
// Get current level as slog.Level
level := tlog.GetLevel()

// Get current level as string
levelStr := tlog.GetLevelString() // Returns "DEBUG", "INFO", etc.

// Check if a specific level is enabled
if tlog.IsLevelEnabled(tlog.LevelDebug) {
    // Expensive debug operation here
    tlog.Debug("Debug info", "data", expensiveOperation())
}
```

### Supported Level Strings

The following strings are supported for `SetLevelFromString()`:

- `trace`, `debug`, `info`, `notice`, `warn`, `warning`, `error`, `fatal`
- All strings are case-insensitive
- `warning` is an alias for `warn`
- Leading and trailing whitespace is automatically trimmed

## Advanced Usage

### Event Callbacks

The tlog package supports registering callback functions that are executed asynchronously whenever a log of a specific level is generated. This is useful for implementing custom log handlers, monitoring systems, alerting, or audit logging.

#### Registering Callbacks

```go
// Register a callback for error-level logs
errorCallbackID := tlog.RegisterCallback(tlog.LevelError, func(event tlog.LogEvent) {
    // Send to monitoring system
    sendToMonitoring(event.Message, event.Level, event.Args)

    // Or send email alert
    if event.Level >= tlog.LevelError {
        sendEmailAlert(event.Message, event.Timestamp)
    }
})

// Register a callback for all info-level logs
infoCallbackID := tlog.RegisterCallback(tlog.LevelInfo, func(event tlog.LogEvent) {
    // Write to audit log
    auditLogger.Log(event.Context, event.Level, event.Message, event.Args...)
})
```

#### LogEvent Structure

The `LogEvent` passed to callbacks contains:

```go
type LogEvent struct {
    Level     slog.Level    // Log level (LevelError, LevelInfo, etc.)
    Message   string        // Log message
    Args      []any         // Key-value pairs passed to the log function
    Timestamp time.Time     // When the log was generated
    Context   context.Context // Context passed to logging function
}
```

#### Managing Callbacks

```go
// Unregister a specific callback
success := tlog.UnregisterCallback(tlog.LevelError, errorCallbackID)

// Clear all callbacks for a specific level
tlog.ClearCallbacks(tlog.LevelError)

// Clear all callbacks for all levels
tlog.ClearAllCallbacks()

// Get callback count for debugging
count := tlog.GetCallbackCount(tlog.LevelError)
fmt.Printf("Number of error callbacks: %d\n", count)
```

#### Callback Safety Features

- **Async Execution**: Callbacks are executed in separate goroutines and won't block logging
- **Panic Recovery**: If a callback panics, it's recovered and logged without affecting the main program
- **Error Isolation**: Failed callbacks don't affect other callbacks or normal logging
- **Buffered Queue**: Events are queued (buffer size: 1000) for processing
- **Non-blocking**: If the queue is full, events are dropped to prevent blocking

#### Example: Monitoring Integration

```go
// Set up error monitoring
tlog.RegisterCallback(tlog.LevelError, func(event tlog.LogEvent) {
    // Extract error information
    var errorMsg string
    var errorCode int

    for i := 0; i < len(event.Args); i += 2 {
        if i+1 < len(event.Args) {
            key := fmt.Sprintf("%v", event.Args[i])
            value := event.Args[i+1]

            switch key {
            case "error":
                errorMsg = fmt.Sprintf("%v", value)
            case "code":
                if code, ok := value.(int); ok {
                    errorCode = code
                }
            }
        }
    }

    // Send to monitoring service
    monitoring.RecordError(monitoring.ErrorEvent{
        Message:   event.Message,
        Error:     errorMsg,
        Code:      errorCode,
        Timestamp: event.Timestamp,
        Level:     event.Level.String(),
    })
})

// Now any error log will trigger monitoring
tlog.Error("Database connection failed", "error", err, "code", 500)
```

#### Graceful Shutdown

When your application shuts down, ensure callbacks are processed:

```go
// Process remaining events and shut down gracefully
defer tlog.Shutdown()
```

### Custom Logger Instances

Create a logger with a specific minimum level without affecting the global logger:

```go
// Create a trace-level logger for detailed debugging
debugLogger := tlog.WithLevel(tlog.LevelTrace)

// Use the custom logger
debugLogger.Log(context.Background(), tlog.LevelTrace, "Detailed trace info")
```

### Error Handling

The package provides descriptive error messages:

```go
err := tlog.SetLevelFromString("invalid")
if err != nil {
    fmt.Println(err)
    // Output: invalid log level 'invalid': supported levels are trace, debug, info, notice, warn, warning, error, fatal
}

err = tlog.SetLevelFromString("")
if err != nil {
    fmt.Println(err)
    // Output: log level cannot be empty
}
```

## Thread Safety

All tlog operations are thread-safe:

```go
// Safe to call concurrently from multiple goroutines
go func() {
    tlog.SetLevel(tlog.LevelDebug)
    tlog.Debug("Debug from goroutine 1")
}()

go func() {
    level := tlog.GetLevel()
    tlog.Info("Current level", "level", level)
}()
```

## Configuration

The package automatically configures itself on initialization:

- **Terminal Detection**: Colors are automatically enabled/disabled based on whether output is a terminal
- **Source Location**: File and line information is included in log output
- **Time Format**: Uses RFC3339 format for timestamps
- **Output**: Logs are written to stderr

## Best Practices

1. **Use appropriate levels**: Use TRACE for very detailed information, DEBUG for troubleshooting, INFO for general information, NOTICE for important events, WARN for potential issues, ERROR for actual problems, and FATAL only for critical failures.

2. **Leverage context**: Use the context variants (`InfoContext`, `ErrorContext`, etc.) when you have relevant context information like request IDs or user information.

3. **Check level enablement**: For expensive operations, check if the level is enabled before performing the work:

   ```go
   if tlog.IsLevelEnabled(tlog.LevelDebug) {
       expensive := doExpensiveCalculation()
       tlog.Debug("Calculation result", "result", expensive)
   }
   ```

4. **Use structured logging**: Prefer key-value pairs over string formatting:

   ```go
   // Good
   tlog.Info("User logged in", "userId", user.ID, "email", user.Email)

   // Less ideal
   tlog.Info(fmt.Sprintf("User %s (%d) logged in", user.Email, user.ID))
   ```

5. **Handle level setting errors**: Always check errors when setting levels from strings:

   ```go
   if err := tlog.SetLevelFromString(configLevel); err != nil {
       tlog.Error("Invalid log level in config", "level", configLevel, "error", err)
       tlog.SetLevel(tlog.LevelInfo) // fallback to reasonable default
   }
   ```

6. **Use callbacks judiciously**: Register callbacks only for levels that need special handling. Keep callback functions lightweight to avoid impacting performance:

   ```go
   // Good - lightweight callback
   tlog.RegisterCallback(tlog.LevelError, func(event tlog.LogEvent) {
       errorCounter.Inc()
       errorQueue <- event
   })

   // Avoid - heavy operations in callbacks
   tlog.RegisterCallback(tlog.LevelInfo, func(event tlog.LogEvent) {
       // Don't do expensive operations here
       sendEmailNotification(event) // This could block
   })
   ```

7. **Clean up callbacks**: Remember to unregister callbacks or clear them when they're no longer needed:

   ```go
   callbackID := tlog.RegisterCallback(tlog.LevelError, errorHandler)
   defer tlog.UnregisterCallback(tlog.LevelError, callbackID)
   ```

## Migration from Previous Version

The improved tlog package is fully backward compatible. Existing code will continue to work without changes. However, you can take advantage of new features:

- Replace `tlog.Info("message")` with `tlog.InfoContext(ctx, "message")` when context is available
- Use `tlog.IsLevelEnabled()` for expensive debug operations
- Take advantage of better error messages in level configuration code
- Register callbacks for critical log levels to implement monitoring and alerting:

  ```go
  // Add monitoring for errors
  tlog.RegisterCallback(tlog.LevelError, func(event tlog.LogEvent) {
      monitoring.RecordError(event.Message, event.Args)
  })
  ```

## API Reference

### Callback Functions

- `RegisterCallback(level slog.Level, callback LogCallback) string` - Register a callback for a log level, returns callback ID
- `UnregisterCallback(level slog.Level, callbackID string) bool` - Remove a specific callback by ID
- `ClearCallbacks(level slog.Level)` - Remove all callbacks for a log level
- `ClearAllCallbacks()` - Remove all callbacks for all levels
- `GetCallbackCount(level slog.Level) int` - Get the number of registered callbacks for a level
- `Shutdown()` - Gracefully shutdown the callback processor
- `RestartProcessor()` - Restart the callback processor (mainly for testing)

### LogCallback Type

```go
type LogCallback func(event LogEvent)

type LogEvent struct {
    Level     slog.Level
    Message   string
    Args      []any
    Timestamp time.Time
    Context   context.Context
}
```

## Enhanced Formatting & Colors

The `tlog` package includes enhanced formatting capabilities powered by `samber/slog-formatter` and color support via `fatih/color`.

### Formatter Configuration

#### FormatterConfig Structure

```go
type FormatterConfig struct {
    EnableColors        bool   // Enable colored output (auto-disabled if terminal doesn't support colors)
    EnableFormatting    bool   // Enable slog-formatter enhancements
    HideSensitiveData   bool   // Hide sensitive data like passwords, tokens, IPs
    TimeFormat          string // Time format for timestamps
}
```

#### Configuration Functions

```go
// Get current formatter configuration
config := tlog.GetFormatterConfig()

// Set custom configuration
customConfig := tlog.FormatterConfig{
    EnableColors:      true,
    EnableFormatting:  true,
    HideSensitiveData: true,
    TimeFormat:        "2006-01-02 15:04:05",
}
tlog.SetFormatterConfig(customConfig)

// Individual configuration options
tlog.EnableColors(true)                    // Enable/disable colors
tlog.EnableSensitiveDataHiding(true)       // Hide sensitive data
tlog.SetTimeFormat("2006-01-02 15:04:05") // Custom time format

// Check current settings
isColorsEnabled := tlog.IsColorsEnabled()
isSensitiveHidden := tlog.IsSensitiveDataHidingEnabled()
timeFormat := tlog.GetTimeFormat()
```

### Advanced Formatters

The package includes specialized formatters for common data types:

#### Built-in Formatters

- **HTTPRequestFormatter**: Formats HTTP request objects with method, URL, protocol, and content length
- **HTTPResponseFormatter**: Formats HTTP response objects with status, status code, protocol, and content length
- **UnixTimestampFormatter**: Converts Unix timestamps to readable RFC3339 format
- **Enhanced PIIFormatter**: Masks sensitive fields including passwords, tokens, API keys, and IP addresses

```go
// These are automatically applied when EnableFormatting is true
tlog.Info("HTTP request processed",
    "timestamp", 1673587200,           // Converted to: 2023-01-13T06:00:00Z
    "client_ip", "192.168.1.100",      // Masked as: *******
    "auth_token", "Bearer jwt-abc",     // Masked as: Bear*******
    "private_key", "-----BEGIN RSA")   // Masked as: ----*******
```

### Color-Enhanced Printing

The package provides color-enhanced printing functions that work alongside terminal detection:

```go
// Basic color printing functions
tlog.ColorTrace("Trace message with color")
tlog.ColorDebug("Debug message with color")
tlog.ColorInfo("Info message with color")
tlog.ColorNotice("Notice message with color")
tlog.ColorWarn("Warning message with color")
tlog.ColorError("Error message with color")
tlog.ColorFatal("Fatal message with color")

// Formatted color printing
tlog.ColorPrint(tlog.LevelInfo, "User %s logged in at %s", username, time.Now())
tlog.ColorPrintln(tlog.LevelWarn, "Connection timeout after %d seconds", 30)

// Print with level prefix - colors only the prefix for levels < WARN
tlog.PrintWithLevel(tlog.LevelInfo, "Information message")   // [INFO] in color, message in normal
tlog.PrintWithLevel(tlog.LevelWarn, "Warning message")       // Full message in warning color

// Demonstrate all log levels with appropriate coloring
tlog.PrintWithLevelAll("Sample message for all levels")
```

### Enhanced Context Support

Context values are automatically extracted and included in log output:

```go
// Create context with tracking information
ctx := context.WithValue(context.Background(), "request_id", "req-12345")
ctx = context.WithValue(ctx, "user_id", "user-456")
ctx = context.WithValue(ctx, "trace_id", "trace-abc-xyz")

// Context values are automatically included in the log output
tlog.InfoContext(ctx, "Processing request", "method", "GET", "path", "/api/users")
// Output: ... Processing request method=GET path=/api/users request_id=req-12345 user_id=user-456 trace_id=trace-abc-xyz

tlog.ErrorContext(ctx, "Request failed", "error", "timeout", "duration", "30s")
// Output: ... Request failed error=timeout duration=30s request_id=req-12345 user_id=user-456 trace_id=trace-abc-xyz
```

Supported context keys that are automatically extracted:

- `request_id`
- `user_id`
- `session_id`
- `trace_id`
- `span_id`

### Sensitive Data Protection

When `HideSensitiveData` is enabled, the following fields are automatically masked:

- **Password fields**: `password`, `pwd`, `pass`, `passwd`
- **Token fields**: `token`, `jwt`, `auth_token`, `access_token`, `refresh_token`
- **API keys**: `key`, `api_key`, `secret`, `client_secret`, `private_key`
- **IP addresses**: `ip`, `addr`, `address`, `remote_addr`, `client_ip`

```go
// Enable sensitive data hiding
tlog.EnableSensitiveDataHiding(true)

// This will mask the sensitive fields
tlog.Info("User login attempt",
    "username", "alice",
    "password", "secret123",          // Masked as "secr*******"
    "auth_token", "Bearer jwt-xyz",   // Masked as "Bear*******"
    "api_key", "sk-test-key-123",     // Masked as "sk-t*******"
    "client_ip", "192.168.1.100",     // Masked as "*******"
    "private_key", "-----BEGIN RSA")  // Masked as "----*******"
```

### Enhanced Error Formatting

The formatter provides enhanced error formatting that includes:

- Error message and type information
- Stack traces (when available)
- Structured error details

```go
import "errors"

err := errors.New("database connection failed")
tlog.Error("Operation failed", "error", err)
// Enhanced output includes error type and structured information
```

### Tozd Error Formatting with Tree Stack Traces

The tlog package provides special formatting for [`gitlab.com/tozd/go/errors`](https://gitlab.com/tozd/go/errors) that includes:

- **Tree-formatted stack traces**: When the terminal supports Unicode and colors are enabled, stack traces are displayed with tree characters (`├─`, `└─`) for better readability
- **Colored output**: Different stack frame depths are colored differently (red for top frame, yellow for recent frames, gray for deeper frames)
- **ASCII fallback**: When Unicode is not supported, ASCII tree characters (`|-`, `` `-`) are used
- **Error details**: Structured key-value details are preserved and displayed
- **Error chains**: Cause relationships are maintained and displayed

#### Example Usage

```go
import "gitlab.com/tozd/go/errors"

// Create error with details
baseErr := errors.WithDetails(
    errors.New("database connection failed"),
    "host", "localhost",
    "port", 5432,
    "database", "myapp",
)

// Add stack trace
stackErr := errors.WithStack(baseErr)

// Log the error - stack trace will be formatted as a tree
tlog.Error("Service initialization failed", "error", stackErr)
```

#### Output Examples

With Unicode support and colors enabled:

```bash
ERROR Service initialization failed
  error:
    message: database connection failed
    details:
      host: localhost
      port: 5432
      database: myapp
    stacktrace:
      frame_0: ├─ /path/to/main.go:42 main.initDatabase
      frame_1: ├─ /path/to/service.go:15 service.Initialize
      frame_2: └─ /path/to/main.go:25 main.main
```

Without Unicode support (ASCII fallback):

```bash
ERROR Service initialization failed
  error:
    message: database connection failed
    stacktrace:
      frame_0: |- /path/to/main.go:42 main.initDatabase
      frame_1: |- /path/to/service.go:15 service.Initialize
      frame_2: `- /path/to/main.go:25 main.main
```

#### Features

- **Automatic terminal detection**: Tree formatting is automatically enabled when the terminal supports Unicode characters
- **Color coding**: Stack frames are colored by depth for easy visual scanning
- **Cause chains**: Wrapped errors show their causes in the output
- **Detail preservation**: Error details are maintained and displayed in a structured format

### Time Format Options

Common time format examples:

```go
// RFC3339 (default)
tlog.SetTimeFormat(time.RFC3339)           // "2006-01-02T15:04:05Z07:00"

// Human-readable formats
tlog.SetTimeFormat("2006-01-02 15:04:05")  // "2006-01-02 15:04:05"
tlog.SetTimeFormat("15:04:05.000")         // "15:04:05.000"
tlog.SetTimeFormat("Jan 02 15:04:05")      // "Jan 02 15:04:05"

// Unix timestamp
tlog.SetTimeFormat("unix")                 // Unix timestamp format
```

### Color Levels

Each log level has an associated color:

- **TRACE**: Bright Black (Gray)
- **DEBUG**: Cyan
- **INFO**: Green
- **NOTICE**: Blue
- **WARN**: Yellow
- **ERROR**: Red
- **FATAL**: Bright Red

Colors are automatically disabled when:

- Terminal doesn't support colors
- Output is redirected to a file
- Colors are explicitly disabled via configuration

## Enhanced Logger Creation

### Logger with Custom Configuration

```go
// Create logger with specific level
debugLogger := tlog.WithLevel(tlog.LevelDebug)
debugLogger.Debug("This will appear")
debugLogger.Trace("This won't appear (below debug level)")

// Create new logger instances
logger1 := tlog.NewLogger()                      // Uses default configuration
logger2 := tlog.NewLoggerWithLevel(tlog.LevelError) // Only logs errors and fatal
```

### Logger Methods

All Logger instances support the same methods as package-level functions:

```go
logger := tlog.NewLogger()

logger.Trace("message", "key", "value")
logger.Debug("message", "key", "value")
logger.Info("message", "key", "value")
logger.Notice("message", "key", "value")
logger.Warn("message", "key", "value")
logger.Error("message", "key", "value")
logger.Fatal("message", "key", "value") // Exits program

// Context variants
ctx := context.Background()
logger.TraceContext(ctx, "message", "key", "value")
logger.DebugContext(ctx, "message", "key", "value")
// ... etc
```
